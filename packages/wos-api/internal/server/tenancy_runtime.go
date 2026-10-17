package server

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/observability"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
)

// openTenantHostRuntime composes complete independent Tenant services behind one
// listener. Host selection never authenticates a caller: the selected child still
// resolves credentials and applies its own Tenant-local permissions.
func openTenantHostRuntime(config Config) (*Runtime, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	catalog, err := postgres.OpenTenantCatalog(ctx, config.Storage.PostgresDSN, postgres.CatalogOptions{
		Schema: config.Storage.PostgresCatalogSchema, MaxOpenConnections: 4,
		MaxLeases: len(config.TenantHosts), MigrateOnOpen: config.Storage.MigrateOnStart && config.ServesAPI(),
	})
	if err != nil {
		return nil, err
	}
	runtime := &Runtime{config: config, catalog: catalog, ownCatalog: true, observer: observability.New(slog.Default())}
	hosts := make(map[string]http.Handler, len(config.TenantHosts))
	for _, entry := range config.TenantHosts {
		child, err := openSingleRuntime(config.hostedTenantConfig(entry), catalog)
		if err != nil {
			_ = runtime.Close()
			return nil, err
		}
		runtime.children = append(runtime.children, child)
		host, _ := canonicalTenantHost(entry.Host)
		hosts[host] = child.Handler()
	}
	runtime.root = admissionHTTP(config, runtime.observer, tenantHostRouter(hosts))
	runtime.http = &http.Server{
		Addr: config.Server.Listen, Handler: runtime.root,
		ReadHeaderTimeout: config.HTTP.ReadHeaderTimeout, ReadTimeout: config.HTTP.ReadTimeout,
		WriteTimeout: config.HTTP.WriteTimeout, IdleTimeout: config.HTTP.IdleTimeout, MaxHeaderBytes: config.HTTP.MaxHeaderBytes,
	}
	return runtime, nil
}

func tenantHostRouter(hosts map[string]http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		host, valid := canonicalTenantHost(request.Host)
		handler, found := hosts[host]
		if !valid || !found {
			w.Header().Set("Cache-Control", "no-store")
			writeTransportError(w, http.StatusMisdirectedRequest, "unknown_tenant_host", "this host is not assigned to a Tenant")
			return
		}
		handler.ServeHTTP(w, request)
	})
}

// runBackground owns workers, maintenance and lease supervision independently
// of HTTP. A host composition runs one instance for each fixed-schema child.
func (r *Runtime) runBackground(parent context.Context) (result error) {
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	var work sync.WaitGroup
	failures := make(chan error, len(r.children)+2)
	start := func(run func() error) {
		work.Add(1)
		go func() {
			defer work.Done()
			if err := run(); err != nil && ctx.Err() == nil {
				select {
				case failures <- err:
				case <-ctx.Done():
				}
			}
		}()
	}
	for _, child := range r.children {
		start(func() error { return child.runBackground(ctx) })
	}
	if r.store != nil && r.config.ProcessRole() != ProcessRoleAPI {
		start(func() error { r.runMaintenance(ctx, time.Minute); return nil })
	}
	if r.deliveryWorker != nil {
		start(func() error {
			err := r.deliveryWorker.Run(ctx)
			if ctx.Err() != nil {
				return nil
			}
			slog.Error("delivery_worker_stopped", "failure_class", r.deliveryDiagnostics.DeliveryWorkerStatus().LastFailureClass)
			if r.config.ProcessRole() != ProcessRoleWorker {
				return nil
			}
			if err == nil {
				err = errors.New("delivery worker exited unexpectedly")
			}
			return domain.NewError(domain.ErrorCodeInvalidConfig, "delivery worker stopped; inspect classified process diagnostics")
		})
	}
	if r.lease != nil {
		start(func() error {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return nil
				case <-ticker.C:
					check, stop := context.WithTimeout(ctx, 2*time.Second)
					err := r.lease.CheckActive(check)
					stop()
					if err != nil && ctx.Err() == nil {
						return domain.NewError(domain.ErrorCodeInvalidConfig, "Tenant lifecycle lease is unavailable; restart after verifying placement")
					}
				}
			}
		})
	}
	select {
	case <-ctx.Done():
	case result = <-failures:
	}
	cancel()
	done := make(chan struct{})
	go func() { work.Wait(); close(done) }()
	timer := time.NewTimer(r.config.Server.ShutdownTimeout)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		result = errors.Join(result, domain.NewError(domain.ErrorCode("deadline_exceeded"), "Tenant background shutdown grace expired"))
	}
	return result
}
