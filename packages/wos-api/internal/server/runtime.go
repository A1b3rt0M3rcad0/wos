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
	catalog             *postgres.TenantCatalog
	lease               *postgres.TenantLease
	ownCatalog          bool
	children            []*Runtime
	observer            *observability.Observer
	config              Config
	store               runtimeStore
	http                *http.Server
	root                http.Handler
	mcpServer           *mcp.Server
	deliveryWorker      *integration.Worker
	deliveryDiagnostics *integration.Diagnostics
	issuerIdentity      *domain.ServerIdentity
}

type runtimeStore interface {
	ports.TransactionManager
	ports.SecurityStore
	Close() error
	SchemaVersion(context.Context) (int64, error)
	SignedIssuerReady(context.Context, *domain.ServerIdentity) (bool, error)
	Migrate(context.Context) error
}

func OpenRuntime(config Config) (*Runtime, error) {
	config = config.withTenantBindingDefaults()
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if len(config.TenantHosts) > 0 {
		return openTenantHostRuntime(config)
	}
	return openSingleRuntime(config, nil)
}

func openSingleRuntime(config Config, catalog *postgres.TenantCatalog) (opened *Runtime, openingErr error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	var store runtimeStore
	var err error
	// Only an API/complete process may initialize an installation. A worker must
	// connect to an already migrated schema, even when defaults enable migration.
	migrate := config.Storage.MigrateOnStart && config.ServesAPI()
	var lease *postgres.TenantLease
	ownCatalog := false
	defer func() {
		if opened != nil {
			return
		}
		if lease != nil {
			_ = lease.Close()
		}
		if ownCatalog {
			_ = catalog.Close()
		}
	}()
	if config.Storage.Driver == StorageDriverPostgres {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		binding, openErr := openTenantStorage(ctx, config, migrate, catalog)
		if openErr != nil {
			return nil, openErr
		}
		store, catalog, lease, ownCatalog = binding.Store, binding.Catalog, binding.Lease, binding.OwnCatalog
		config.Auth.BootstrapTenantName = lease.Placement().Name
	} else {
		tenantID, _ := domain.ParseID(config.Storage.TenantID)
		tenantName := config.Auth.LocalTenantName
		if config.Auth.Mode == AuthModeAPIToken {
			tenantName = config.Auth.BootstrapTenantName
		}
		store, err = sqlite.Open(config.Storage.SQLitePath, sqlite.Options{MigrateOnOpen: migrate, TenantID: tenantID, TenantName: tenantName})
	}
	if err != nil {
		return nil, err
	}

	// Migration-disabled startup must also refuse unsupported schemas before
	// bootstrap credentials, tenants or any serving component can write.
	version, err := store.SchemaVersion(context.Background())
	supported, schemaErr := sqlite.LatestSchemaVersion()
	if config.Storage.Driver == StorageDriverPostgres {
		supported, schemaErr = postgres.LatestSchemaVersion()
	}
	if err != nil || schemaErr != nil || version != supported {
		_ = store.Close()
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "database schema is incompatible with this binary; run a compatible migration before serving")
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
	var localTenants []ports.Tenant
	var resolve func(*http.Request) (application.Identity, error)
	var appAuthorizer ports.Authorizer = authorizer
	if config.Auth.Mode == AuthModeLocal {
		ns, _ := domain.ParseID(config.Auth.LocalTenantID)
		tenant := ports.Tenant{ID: ns, Name: config.Auth.LocalTenantName}
		if err = store.PutTenant(context.Background(), tenant); err != nil {
			if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeAlreadyExists {
				store.Close()
				return nil, err
			}
		}
		localTenants = []ports.Tenant{tenant}
	}
	if config.Auth.Mode == AuthModeAPIToken {
		security = &application.SecurityService{Store: store, Clock: systemClock{}, IDs: ids}
		if config.ServesAPI() {
			ns, _ := domain.ParseID(config.Auth.BootstrapTenantID)
			if err := security.Bootstrap(context.Background(), ports.Tenant{ID: ns, Name: config.Auth.BootstrapTenantName}, config.Auth.LocalPrincipalID, config.Auth.BootstrapToken); err != nil {
				store.Close()
				return nil, err
			}
		}
		appAuthorizer = security
		resolve = func(r *http.Request) (application.Identity, error) {
			if identity, ok := application.IdentityFromContext(r.Context()); ok {
				if config.Storage.TenantID != "" && identity.TenantID.String() != config.Storage.TenantID {
					return application.Identity{}, domain.NewError(domain.ErrorCodeForbidden, "identity does not belong to the bound Tenant")
				}
				return identity, nil
			}
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
	if stats, ok := store.(ports.StoragePoolObserver); ok {
		observer.Storage = stats
	}
	service.SetObserver(observer)
	service.SetIndependentReviewer(config.Auth.IndependentReviewer)
	issuerIdentity, err := configureSignedRuntime(context.Background(), config, store, service, security, ids)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	if config.ServesAPI() && len(config.Integration.Endpoints) > 0 {
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
	var api http.Handler
	if config.ServesAPI() {
		api, err = httptransport.New(httptransport.Options{
			Prefix:          config.HTTP.Prefix,
			RequestTimeout:  config.HTTP.RequestTimeout,
			Service:         service,
			IDs:             ids,
			LocalAuth:       auth,
			ResolveIdentity: resolve,
			Security:        security,
			LocalTenants:    localTenants,
		})
		if err != nil {
			_ = store.Close()
			return nil, err
		}
	}

	deliveryDiagnostics := integration.NewDiagnostics(config.RunsDeliveryWorker())
	observer.DeliveryDiagnostics = deliveryDiagnostics
	if err = service.ConfigureDeliveryDiagnostics(deliveryDiagnostics, config.Integration.MaxBacklogAge); err != nil {
		store.Close()
		return nil, err
	}
	runtime := &Runtime{config: config, store: store, issuerIdentity: issuerIdentity, observer: observer, deliveryDiagnostics: deliveryDiagnostics, catalog: catalog, lease: lease, ownCatalog: ownCatalog}
	if config.RunsDeliveryWorker() {
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
		runtime.deliveryWorker.Concurrency = config.Integration.WorkerConcurrency
		runtime.deliveryWorker.ClaimPolicy = ports.DeliveryClaimPolicy{MaxInflightPerTenant: config.Integration.MaxInflightPerTenant, MaxInflightPerEndpoint: config.Integration.MaxInflightPerEndpoint}
		if config.Storage.Driver == StorageDriverSQLite && config.Integration.WorkerConcurrency > 1 {
			slog.Warn("sqlite_delivery_parallelism", "advisory", "SQLite serializes writes; measure before increasing delivery concurrency")
		}
		runtime.deliveryWorker.Observer = observer
		runtime.deliveryWorker.Diagnostics = deliveryDiagnostics
	}
	root := http.NewServeMux()
	if config.ServesAPI() {
		root.HandleFunc("GET /app/config", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			json.NewEncoder(w).Encode(map[string]any{"api_prefix": config.HTTP.Prefix, "auth_mode": config.Auth.Mode, "tenant_bound": config.Storage.TenantID != "", "tenant_id": config.Storage.TenantID})
		})
		root.Handle("GET /app/", web.Handler())
		root.HandleFunc("GET /app", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/app/", http.StatusSeeOther) })
		if security != nil {
			root.Handle("/session", sessionHandler(security))
		}
	}
	root.Handle("GET /metrics", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if security != nil {
			id, err := resolve(r)
			if err != nil {
				http.Error(w, "unauthorized", 401)
				return
			}
			ctx := application.WithIdentity(r.Context(), id)
			if err = security.RequireInstallation(ctx, ports.PermissionInstallationObserve); err != nil {
				http.Error(w, "forbidden", 403)
				return
			}
		}
		observer.ServeHTTP(w, r)
	}))
	root.HandleFunc("GET /livez", runtime.handleLive)
	root.HandleFunc("GET /readyz", runtime.handleReady)
	if config.ServesAPI() && config.MCP.Enabled {
		options := mcptransport.Options{ProductVersion: Version, Security: security, RequestTimeout: config.HTTP.RequestTimeout}
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
	if config.ServesAPI() {
		root.Handle("/", api)
	}
	admittedProduct := tenantAdmissionHTTP(config.HTTP.MaxInflightPerTenant, resolve, root)
	business := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if config.Auth.Mode == AuthModeLocal && !loopbackHost(r.Host) {
			http.Error(w, "loopback host required", http.StatusForbidden)
			return
		}
		if r.Method != "GET" && r.Method != "HEAD" && r.Method != "OPTIONS" && !sameOrigin(r) {
			http.Error(w, "origin rejected", http.StatusForbidden)
			return
		}
		if config.ServesAPI() && (r.URL.Path == config.HTTP.Prefix || strings.HasPrefix(r.URL.Path, strings.TrimSuffix(config.HTTP.Prefix, "/")+"/") || config.MCP.Enabled && r.URL.Path == config.MCP.Path) {
			admittedProduct.ServeHTTP(w, r)
			return
		}
		root.ServeHTTP(w, r)
	})
	runtime.root = observer.HTTP(admissionHTTP(config, observer, business))
	runtime.http = &http.Server{
		Addr:              config.Server.Listen,
		Handler:           runtime.root,
		ReadHeaderTimeout: config.HTTP.ReadHeaderTimeout,
		ReadTimeout:       config.HTTP.ReadTimeout,
		WriteTimeout:      config.HTTP.WriteTimeout,
		IdleTimeout:       config.HTTP.IdleTimeout,
		MaxHeaderBytes:    config.HTTP.MaxHeaderBytes,
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
	return r.serve(ctx, listener)
}

func (r *Runtime) serve(ctx context.Context, listener net.Listener) (serveErr error) {
	defer listener.Close()
	listener = &admissionListener{Listener: listener, slots: make(chan struct{}, r.config.Server.MaxConnections), observer: r.observer}
	backgroundCtx, stopBackground := context.WithCancel(ctx)
	backgroundDone := make(chan error, 1)
	backgroundConsumed := false
	go func() { backgroundDone <- r.runBackground(backgroundCtx) }()
	defer func() {
		stopBackground()
		if backgroundConsumed {
			return
		}
		timer := time.NewTimer(r.config.Server.ShutdownTimeout)
		defer timer.Stop()
		select {
		case err := <-backgroundDone:
			serveErr = errors.Join(serveErr, err)
		case <-timer.C:
			serveErr = errors.Join(serveErr, domain.NewError(domain.ErrorCode("deadline_exceeded"), "background shutdown grace expired"))
		}
	}()

	errCh := make(chan error, 1)
	go func() {
		errCh <- r.http.Serve(listener)
	}()

	select {
	case backgroundErr := <-backgroundDone:
		backgroundConsumed = true
		shutdownCtx, cancel := context.WithTimeout(context.Background(), r.config.Server.ShutdownTimeout)
		defer cancel()
		if err := r.http.Shutdown(shutdownCtx); err != nil {
			_ = r.http.Close()
		}
		<-errCh
		if ctx.Err() != nil {
			return backgroundErr
		}
		if backgroundErr == nil {
			backgroundErr = domain.NewError(domain.ErrorCodeInvalidConfig, "runtime background supervision stopped")
		}
		return backgroundErr
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), r.config.Server.ShutdownTimeout)
		defer cancel()
		if err := r.http.Shutdown(shutdownCtx); err != nil {
			_ = r.http.Close()
			<-errCh
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
	if r == nil {
		return nil
	}
	var errs []error
	if r.http != nil {
		errs = append(errs, r.http.Close())
	}
	for _, child := range r.children {
		errs = append(errs, child.Close())
	}
	if r.store != nil {
		errs = append(errs, r.store.Close())
	}
	if r.lease != nil {
		errs = append(errs, r.lease.Close())
	}
	if r.ownCatalog && r.catalog != nil {
		errs = append(errs, r.catalog.Close())
	}
	return errors.Join(errs...)
}

func (r *Runtime) Migrate(ctx context.Context) error {
	if r.lease != nil {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "PostgreSQL Tenant migrations require drained runtimes and the suspended Tenant migration operation")
	}
	if len(r.children) > 0 {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "migrate each drained Tenant explicitly before starting a multi-Tenant host")
	}
	return r.store.Migrate(ctx)
}
func (r *Runtime) Backup(ctx context.Context, path string) error {
	if store, ok := r.store.(interface {
		Backup(context.Context, string) error
	}); ok {
		return store.Backup(ctx, path)
	}
	return domain.NewError(domain.ErrorCodeInvalidConfig, "PostgreSQL backups use pg_dump --format=custom; see docs/operations.md")
}

func (r *Runtime) handleLive(w http.ResponseWriter, _ *http.Request) {
	r.writeHealth(w, http.StatusOK, "live")
}

func (r *Runtime) handleReady(w http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
	defer cancel()
	if r.lease != nil && r.lease.CheckActive(ctx) != nil {
		r.writeHealth(w, http.StatusServiceUnavailable, "not_ready")
		return
	}
	version, err := r.store.SchemaVersion(ctx)
	supported, _ := sqlite.LatestSchemaVersion()
	if r.config.Storage.Driver == StorageDriverPostgres {
		supported, _ = postgres.LatestSchemaVersion()
	}
	if err != nil || version != supported {
		r.writeHealth(w, http.StatusServiceUnavailable, "not_ready")
		return
	}
	if r.config.ProcessRole() == ProcessRoleWorker {
		worker := r.deliveryDiagnostics.DeliveryWorkerStatus()
		if !worker.Running {
			r.writeHealth(w, http.StatusServiceUnavailable, "not_ready")
			return
		}
		r.writeHealth(w, http.StatusOK, "ready")
		return
	}
	issuerReady, err := r.store.SignedIssuerReady(ctx, r.issuerIdentity)
	if err != nil || !issuerReady {
		r.writeHealth(w, http.StatusServiceUnavailable, "not_ready")
		return
	}
	r.writeHealth(w, http.StatusOK, "ready")
}

func (r *Runtime) writeHealth(w http.ResponseWriter, status int, state string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]any{"status": state, "process_role": r.config.ProcessRole(), "worker_scope": "this_process", "delivery_worker": r.deliveryDiagnostics.DeliveryWorkerStatus()})
}

func (r *Runtime) RunStdio(ctx context.Context) error {
	if len(r.children) > 0 {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "stdio requires one explicit Tenant; hostname routing is available over HTTP")
	}
	if r.mcpServer == nil {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "MCP is disabled")
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	backgroundDone := make(chan error, 1)
	protocolDone := make(chan error, 1)
	go func() { backgroundDone <- r.runBackground(runCtx) }()
	go func() {
		protocolDone <- r.mcpServer.Run(runCtx, &mcp.StdioTransport{MaxLineLength: mcptransport.MaxPayloadBytes})
	}()
	var result error
	var remaining <-chan error
	select {
	case result = <-protocolDone:
		remaining = backgroundDone
	case result = <-backgroundDone:
		remaining = protocolDone
	}
	cancel()
	timer := time.NewTimer(r.config.Server.ShutdownTimeout)
	defer timer.Stop()
	select {
	case err := <-remaining:
		if err != nil && !errors.Is(err, context.Canceled) {
			result = errors.Join(result, err)
		}
	case <-timer.C:
		result = errors.Join(result, domain.NewError(domain.ErrorCode("deadline_exceeded"), "stdio shutdown grace expired"))
	}
	return result
}
