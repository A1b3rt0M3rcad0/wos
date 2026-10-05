package server

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/observability"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/authentication/local"
	httptransport "github.com/A1b3rt0M3rcad0/wos/packages/wos-api/http"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/integration"
	mcptransport "github.com/A1b3rt0M3rcad0/wos/packages/wos-api/mcp"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/web"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/url"
)

type Runtime struct {
	config         Config
	store          runtimeStore
	http           *http.Server
	root           http.Handler
	mcpServer      *mcp.Server
	deliveryWorker *integration.Worker
}

type runtimeStore interface {
	ports.TransactionManager
	ports.SecurityStore
	Close() error
	SchemaVersion(context.Context) (int64, error)
	Migrate(context.Context) error
}

func OpenRuntime(config Config) (*Runtime, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if !config.HTTP.Enabled {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "Wave 05 standalone runtime requires HTTP to be enabled")
	}

	var store runtimeStore
	var err error
	if config.Storage.Driver == StorageDriverPostgres {
		store, err = postgres.Open(config.Storage.PostgresDSN, postgres.Options{MigrateOnOpen: config.Storage.MigrateOnStart})
	} else {
		store, err = sqlite.Open(config.Storage.SQLitePath, sqlite.Options{MigrateOnOpen: config.Storage.MigrateOnStart})
	}
	if err != nil {
		return nil, err
	}

	ids := uuidV7Generator{}
	auth, err := local.New(config.Auth.LocalPrincipalID)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	authorizer, err := local.NewAuthorizer(config.Auth.LocalPrincipalID, config.Auth.LocalAllowAdministrativeOverrides)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	var security *application.SecurityService
	var localNamespaces []ports.Namespace
	var resolve func(*http.Request) (application.Identity, error)
	var appAuthorizer ports.Authorizer = authorizer
	if config.Auth.Mode == AuthModeLocal {
		ns, _ := domain.ParseID(config.Auth.LocalNamespaceID)
		namespace := ports.Namespace{ID: ns, Name: config.Auth.LocalNamespaceName}
		if err = store.PutNamespace(context.Background(), namespace); err != nil {
			if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeAlreadyExists {
				store.Close()
				return nil, err
			}
		}
		localNamespaces = []ports.Namespace{namespace}
	}
	if config.Auth.Mode == AuthModeAPIToken {
		security = &application.SecurityService{Store: store, Clock: systemClock{}, IDs: ids}
		ns, _ := domain.ParseID(config.Auth.BootstrapNamespaceID)
		if err := security.Bootstrap(context.Background(), ports.Namespace{ID: ns, Name: config.Auth.BootstrapNamespaceName}, config.Auth.LocalPrincipalID, config.Auth.BootstrapToken); err != nil {
			store.Close()
			return nil, err
		}
		appAuthorizer = security
		resolve = func(r *http.Request) (application.Identity, error) {
			raw := r.Header.Get("Authorization")
			if raw == "" {
				if cookie, err := r.Cookie(sessionCookie); err == nil {
					if r.Method != "GET" && r.Method != "HEAD" && !sameOrigin(r) {
						return application.Identity{}, domain.NewError(domain.ErrorCodeForbidden, "origin rejected")
					}
					return security.Authenticate(r.Context(), cookie.Value)
				}
			}
			if !strings.HasPrefix(raw, "Bearer ") {
				return application.Identity{}, domain.NewError(domain.ErrorCodeForbidden, "bearer credential required")
			}
			return security.Authenticate(r.Context(), strings.TrimPrefix(raw, "Bearer "))
		}
	}
	service, err := application.NewAuthorizedService(store, systemClock{}, ids, appAuthorizer)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	observer := observability.New(slog.New(slog.NewJSONHandler(os.Stderr, nil)))
	service.SetObserver(observer)
	service.SetIndependentReviewer(config.Auth.IndependentReviewer)
	if len(config.Integration.Endpoints) > 0 {
		installer, ok := store.(interface {
			InstallEndpoints(context.Context, []ports.WebhookEndpoint) error
		})
		if !ok {
			store.Close()
			return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "endpoint installation unavailable")
		}
		if err = installer.InstallEndpoints(context.Background(), config.Integration.Endpoints); err != nil {
			store.Close()
			return nil, err
		}
	}
	api, err := httptransport.New(httptransport.Options{
		Prefix:          config.HTTP.Prefix,
		RequestTimeout:  config.HTTP.RequestTimeout,
		Service:         service,
		IDs:             ids,
		LocalAuth:       auth,
		ResolveIdentity: resolve,
		Security:        security,
		LocalNamespaces: localNamespaces,
	})
	if err != nil {
		_ = store.Close()
		return nil, err
	}

	runtime := &Runtime{config: config, store: store}
	if config.Integration.WorkerEnabled {
		outbox, ok := store.(ports.DeliveryStore)
		if !ok {
			store.Close()
			return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "delivery store unavailable")
		}
		hosts := []string{}
		for _, endpoint := range config.Integration.Endpoints {
			u, _ := url.Parse(endpoint.URL)
			hosts = append(hosts, u.Hostname())
		}
		runtime.deliveryWorker, err = integration.New(outbox, systemClock{}, ids, config.Integration.Secrets, integration.Policy{AllowedHosts: hosts, AllowLoopback: config.Integration.AllowLoopback})
		if err != nil {
			store.Close()
			return nil, err
		}
	}
	root := http.NewServeMux()
	root.HandleFunc("GET /app/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", "no-store")
		json.NewEncoder(w).Encode(map[string]any{"api_prefix": config.HTTP.Prefix, "auth_mode": config.Auth.Mode})
	})
	root.Handle("GET /app/", web.Handler())
	root.HandleFunc("GET /app", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/app/", http.StatusSeeOther) })
	if security != nil {
		root.Handle("/session", sessionHandler(security))
	}
	root.Handle("GET /metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if security != nil {
			id, err := resolve(r)
			if err != nil {
				http.Error(w, "unauthorized", 401)
				return
			}
			ctx := application.WithIdentity(r.Context(), id)
			if err = security.Require(ctx, id.NamespaceID, ports.PermissionNamespaceAdmin); err != nil {
				http.Error(w, "forbidden", 403)
				return
			}
		}
		observer.ServeHTTP(w, r)
	}))
	root.HandleFunc("GET /livez", runtime.handleLive)
	root.HandleFunc("GET /readyz", runtime.handleReady)
	if config.MCP.Enabled {
		options := mcptransport.Options{Security: security, RequestTimeout: config.HTTP.RequestTimeout}
		if resolve != nil {
			options.ResolveIdentity = func(ctx context.Context, headers http.Header) (application.Identity, error) {
				request, _ := http.NewRequestWithContext(ctx, "POST", "http://wos.local/mcp", nil)
				request.Header = headers
				if id, ok := application.IdentityFromContext(ctx); ok {
					return id, nil
				}
				return resolve(request)
			}
		} else {
			identity := application.Identity{PrincipalID: auth.Principal(), Actor: auth.Actor()}
			options.LocalIdentity = &identity
		}
		protocol, err := mcptransport.New(service, ids, options)
		if err != nil {
			store.Close()
			return nil, err
		}
		runtime.mcpServer = protocol
		stream := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server { return protocol }, &mcp.StreamableHTTPOptions{Stateless: config.MCP.Stateless, JSONResponse: true})
		root.Handle(config.MCP.Path, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if resolve != nil {
				if id, err := resolve(r); err != nil {
					w.Header().Set("WWW-Authenticate", "Bearer")
					http.Error(w, "unauthorized", http.StatusUnauthorized)
					return
				} else {
					r = r.WithContext(application.WithIdentity(r.Context(), id))
				}
			}
			r.Body = http.MaxBytesReader(w, r.Body, mcptransport.MaxPayloadBytes)
			stream.ServeHTTP(w, r)
		}))
	}
	root.Handle("/", api)
	runtime.root = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if config.Auth.Mode == AuthModeLocal && !loopbackHost(r.Host) {
			http.Error(w, "loopback host required", http.StatusForbidden)
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" && !sameOrigin(r) {
			http.Error(w, "origin rejected", http.StatusForbidden)
			return
		}
		observer.HTTP(root).ServeHTTP(w, r)
	})
	runtime.http = &http.Server{
		Addr:              config.Server.Listen,
		Handler:           runtime.root,
		ReadHeaderTimeout: 5 * time.Second,
	}
	return runtime, nil
}

func (r *Runtime) Handler() http.Handler {
	if r == nil {
		return http.NotFoundHandler()
	}
	return r.root
}

func (r *Runtime) Serve(ctx context.Context) error {
	if r == nil || r.http == nil {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "server runtime is not initialized")
	}
	listener, err := net.Listen("tcp", r.config.Server.Listen)
	if err != nil {
		return err
	}
	if r.deliveryWorker != nil {
		workerCtx, stop := context.WithCancel(ctx)
		done := make(chan struct{})
		go func() { defer close(done); _ = r.deliveryWorker.Run(workerCtx) }()
		defer func() { stop(); <-done }()
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- r.http.Serve(listener)
	}()

	select {
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), r.config.Server.ShutdownTimeout)
		defer cancel()
		if err := r.http.Shutdown(shutdownCtx); err != nil {
			return err
		}
		err := <-errCh
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	}
}

func (r *Runtime) Close() error {
	if r == nil || r.store == nil {
		return nil
	}
	return r.store.Close()
}

func (r *Runtime) Migrate(ctx context.Context) error { return r.store.Migrate(ctx) }
func (r *Runtime) Backup(ctx context.Context, path string) error {
	if store, ok := r.store.(interface {
		Backup(context.Context, string) error
	}); ok {
		return store.Backup(ctx, path)
	}
	return domain.NewError(domain.ErrorCodeInvalidConfig, "PostgreSQL backups use pg_dump --format=custom; see docs/operations.md")
}

func (r *Runtime) handleLive(w http.ResponseWriter, _ *http.Request) {
	writeHealth(w, http.StatusOK, "live")
}

func (r *Runtime) handleReady(w http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
	defer cancel()
	version, err := r.store.SchemaVersion(ctx)
	supported, _ := sqlite.LatestSchemaVersion()
	if err != nil || version != supported {
		writeHealth(w, http.StatusServiceUnavailable, "not_ready")
		return
	}
	writeHealth(w, http.StatusOK, "ready")
}

func writeHealth(w http.ResponseWriter, status int, state string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": state})
}

func (r *Runtime) RunStdio(ctx context.Context) error {
	if r.mcpServer == nil {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "MCP is disabled")
	}
	return r.mcpServer.Run(ctx, &mcp.StdioTransport{MaxLineLength: mcptransport.MaxPayloadBytes})
}
