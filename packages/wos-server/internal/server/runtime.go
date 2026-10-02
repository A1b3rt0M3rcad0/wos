package server

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/authentication/local"
	httptransport "github.com/A1b3rt0M3rcad0/wos/packages/wos-api/http"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
)

type Runtime struct {
	config Config
	store  *sqlite.Store
	http   *http.Server
	root   http.Handler
}

func OpenRuntime(config Config) (*Runtime, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	if config.Storage.Driver != StorageDriverSQLite {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "Wave 05 standalone runtime currently requires SQLite")
	}
	if config.Auth.Mode != AuthModeLocal {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "Wave 05 standalone runtime currently requires local auth")
	}
	if !config.HTTP.Enabled {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "Wave 05 standalone runtime requires HTTP to be enabled")
	}

	store, err := sqlite.Open(config.Storage.SQLitePath, sqlite.Options{
		MigrateOnOpen: config.Storage.MigrateOnStart,
	})
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
	service, err := application.NewServiceWithAuthorizer(store, systemClock{}, ids, authorizer)
	if err != nil {
		_ = store.Close()
		return nil, err
	}
	api, err := httptransport.New(httptransport.Options{
		Prefix:         config.HTTP.Prefix,
		RequestTimeout: config.HTTP.RequestTimeout,
		Service:        service,
		IDs:            ids,
		LocalAuth:      auth,
	})
	if err != nil {
		_ = store.Close()
		return nil, err
	}

	runtime := &Runtime{config: config, store: store}
	root := http.NewServeMux()
	root.HandleFunc("GET /livez", runtime.handleLive)
	root.HandleFunc("GET /readyz", runtime.handleReady)
	root.Handle("/", api)
	runtime.root = root
	runtime.http = &http.Server{
		Addr:              config.Server.Listen,
		Handler:           root,
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

func (r *Runtime) handleLive(w http.ResponseWriter, _ *http.Request) {
	writeHealth(w, http.StatusOK, "live")
}

func (r *Runtime) handleReady(w http.ResponseWriter, request *http.Request) {
	ctx, cancel := context.WithTimeout(request.Context(), 2*time.Second)
	defer cancel()
	version, err := r.store.SchemaVersion(ctx)
	if err != nil || version < 1 {
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
