package server

import (
	"context"
	"crypto/ed25519"
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestProcessRolesPreserveDefaultAndRejectUnsafeCompositions(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.ProcessRole() != ProcessRoleComplete || !cfg.ServesAPI() || cfg.RunsDeliveryWorker() {
		t.Fatal("default role changed historical runtime behavior")
	}
	cfg.Server.Role = ""
	if cfg.ProcessRole() != ProcessRoleComplete || cfg.Validate() != nil {
		t.Fatal("omitted embedded role is not backward compatible")
	}
	for _, role := range []string{"all", "API", "unknown"} {
		cfg.Server.Role = role
		if cfg.Validate() == nil {
			t.Fatal("unknown role accepted", role)
		}
	}
	cfg.Server.Role = ProcessRoleAPI
	cfg.Integration.WorkerEnabled = true
	if cfg.Validate() == nil {
		t.Fatal("API process silently started a worker")
	}
	cfg = workerTestConfig(DefaultConfig())
	if err := cfg.Validate(); err != nil {
		t.Fatal("worker requires bootstrap or public API", err)
	}
	if !cfg.RunsDeliveryWorker() || cfg.ServesAPI() {
		t.Fatal("worker role does not select worker-only composition")
	}
	cfg.Storage.Driver = StorageDriverSQLite
	if cfg.Validate() == nil {
		t.Fatal("separate worker accepted SQLite multi-process topology")
	}
	cfg = workerTestConfig(DefaultConfig())
	cfg.MCP.Enabled = true
	if cfg.Validate() == nil {
		t.Fatal("worker exposed MCP")
	}
	t.Setenv("WOS_PROCESS_ROLE", "")
	if ConfigFromEnv().Validate() == nil {
		t.Fatal("explicitly empty environment role ignored")
	}
}

func workerTestConfig(cfg Config) Config {
	if cfg.Storage.TenantID == "" {
		cfg.Storage.TenantID = cfg.Auth.BootstrapTenantID
		if cfg.Storage.TenantID == "" {
			cfg.Storage.TenantID = cfg.Auth.LocalTenantID
		}
	}
	cfg.Server.Role = ProcessRoleWorker
	cfg.Storage.Driver = StorageDriverPostgres
	if cfg.Storage.PostgresDSN == "" {
		cfg.Storage.PostgresDSN = "postgres://unused.invalid/wos"
	}
	cfg.Auth.Mode = AuthModeAPIToken
	cfg.Auth.BootstrapToken = ""
	cfg.Auth.BootstrapTenantID = ""
	cfg.Auth.BootstrapTenantName = ""
	cfg.HTTP.Enabled = false // The operational listener is independent of product HTTP.
	return cfg
}

func TestScaleAdmissionEnvironmentBounds(t *testing.T) {
	for _, name := range []string{"WOS_DELIVERY_MAX_INFLIGHT_PER_TENANT", "WOS_DELIVERY_MAX_INFLIGHT_PER_ENDPOINT"} {
		for _, value := range []string{"-1", "65", "invalid"} {
			t.Run(name+value, func(t *testing.T) {
				t.Setenv(name, value)
				if ConfigFromEnv().Validate() == nil {
					t.Fatal("invalid persisted delivery cap accepted", name, value)
				}
			})
		}
		for _, value := range []string{"0", "1", "64"} {
			t.Run(name+value, func(t *testing.T) {
				t.Setenv(name, value)
				if err := ConfigFromEnv().Validate(); err != nil {
					t.Fatal("valid persisted delivery cap refused", err)
				}
			})
		}
	}
	for _, value := range []string{"-1", "129", "invalid"} {
		t.Run("http"+value, func(t *testing.T) {
			t.Setenv("WOS_HTTP_MAX_INFLIGHT_PER_TENANT", value)
			if ConfigFromEnv().Validate() == nil {
				t.Fatal("invalid process admission cap accepted", value)
			}
		})
	}
}

func TestWorkerRefusesToInitializeDatabase(t *testing.T) {
	forEachRuntimeStorage(t, func(t *testing.T, cfg Config) {
		if cfg.Storage.Driver != StorageDriverPostgres {
			t.Skip("independent worker requires PostgreSQL")
		}
		cfg = workerTestConfig(cfg)
		if runtime, err := openRuntimeFixture(t, &cfg); err == nil {
			runtime.Close()
			t.Fatal("worker initialized a missing installation")
		}
		db, err := sql.Open("pgx", cfg.Storage.PostgresDSN)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		var count int
		if err = db.QueryRow(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema=current_schema()`).Scan(&count); err != nil || count != 0 {
			t.Fatal("worker wrote schema before initialization", count, err)
		}
	})
}

func TestWorkerIssuerIsOptionalAndCannotInitializeOrReplaceIdentity(t *testing.T) {
	forEachRuntimeStorage(t, func(t *testing.T, cfg Config) {
		if cfg.Storage.Driver != StorageDriverPostgres {
			t.Skip("independent worker requires PostgreSQL")
		}
		cfg.Auth.Mode = AuthModeAPIToken
		cfg.Auth.BootstrapToken = strings.Repeat("i", 40)
		cfg.Auth.BootstrapTenantID = cfg.Auth.LocalTenantID
		cfg.Auth.BootstrapTenantName = "Issuer installation"
		api, err := openRuntimeFixture(t, &cfg)
		if err != nil {
			t.Fatal(err)
		}
		api.Close()
		seed := make([]byte, ed25519.SeedSize)
		seed[0] = 41
		t.Setenv("WOS_ROLE_TEST_SEED", base64.StdEncoding.EncodeToString(seed))
		workerCfg := workerTestConfig(cfg)
		workerCfg.Signing.SeedEnv = "WOS_ROLE_TEST_SEED"
		if worker, err := openRuntimeFixture(t, &workerCfg); err == nil {
			worker.Close()
			t.Fatal("worker initialized first issuer identity")
		}
		cfg.Signing.SeedEnv = "WOS_ROLE_TEST_SEED"
		api, err = openRuntimeFixture(t, &cfg)
		if err != nil {
			t.Fatal(err)
		}
		identity := *api.issuerIdentity
		api.Close()
		worker, err := openRuntimeFixture(t, &workerCfg)
		if err != nil {
			t.Fatal("worker rejected matching optional issuer seed", err)
		}
		if worker.issuerIdentity == nil || *worker.issuerIdentity != identity {
			t.Fatal("worker changed persisted issuer identity")
		}
		worker.Close()
		seed[0]++
		t.Setenv("WOS_ROLE_TEST_SEED", base64.StdEncoding.EncodeToString(seed))
		if worker, err = openRuntimeFixture(t, &workerCfg); err == nil {
			worker.Close()
			t.Fatal("worker accepted a replacement issuer seed")
		}
		workerCfg.Signing.SeedEnv = ""
		worker, err = openRuntimeFixture(t, &workerCfg)
		if err != nil {
			t.Fatal("worker requires possession of issuer private key", err)
		}
		defer worker.Close()
		if worker.issuerIdentity != nil {
			t.Fatal("worker without seed silently loaded an issuer private key")
		}
	})
}

func TestIndependentWorkerOperationalListenerAndSupervision(t *testing.T) {
	forEachRuntimeStorage(t, func(t *testing.T, cfg Config) {
		if cfg.Storage.Driver != StorageDriverPostgres {
			t.Skip("independent worker requires PostgreSQL")
		}
		cfg.Auth.Mode = AuthModeAPIToken
		cfg.Auth.BootstrapToken = strings.Repeat("p", 40)
		cfg.Auth.BootstrapTenantID = cfg.Auth.LocalTenantID
		cfg.Auth.BootstrapTenantName = "Existing installation"
		api, err := openRuntimeFixture(t, &cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer api.Close()
		cfg = workerTestConfig(cfg)
		worker, err := openRuntimeFixture(t, &cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer worker.Close()
		for _, path := range []string{"/app/", "/app/config", "/session", "/api/v1/capabilities", "/mcp"} {
			response := httptest.NewRecorder()
			worker.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://localhost"+path, nil))
			if response.Code != http.StatusNotFound {
				t.Fatal("worker exposed business route", path, response.Code)
			}
		}
		response := httptest.NewRecorder()
		worker.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://localhost/readyz", nil))
		if response.Code != http.StatusServiceUnavailable {
			t.Fatal("worker claimed readiness before its pool started")
		}
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan error, 1)
		go func() { done <- worker.serve(ctx, listener) }()
		defer cancel()
		deadline := time.Now().Add(3 * time.Second)
		for !worker.deliveryDiagnostics.DeliveryWorkerStatus().Running {
			if time.Now().After(deadline) {
				t.Fatal("worker pool did not start")
			}
			time.Sleep(time.Millisecond)
		}
		result, err := http.Get("http://" + listener.Addr().String() + "/readyz")
		if err != nil {
			t.Fatal(err)
		}
		var health struct {
			Role  string `json:"process_role"`
			Scope string `json:"worker_scope"`
		}
		err = json.NewDecoder(result.Body).Decode(&health)
		result.Body.Close()
		if result.StatusCode != http.StatusOK || err != nil || health.Role != ProcessRoleWorker || health.Scope != "this_process" {
			t.Fatal("worker health is not role-aware", result.StatusCode, health, err)
		}
		cancel()
		select {
		case err = <-done:
			if err != nil {
				t.Fatal("worker failed graceful shutdown", err)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("worker did not stop")
		}
		if worker.deliveryDiagnostics.DeliveryWorkerStatus().Running {
			t.Fatal("worker still running after shutdown")
		}
		// A worker-only process must exit on a fatal lane failure so the
		// deployment supervisor can restore delivery capacity.
		failed, err := openRuntimeFixture(t, &cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer failed.Close()
		failed.deliveryWorker.Store = fatalDeliveryStore{failed.deliveryWorker.Store}
		listener, err = net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		failureCtx, stopFailure := context.WithCancel(context.Background())
		defer stopFailure()
		go func() { done <- failed.serve(failureCtx, listener) }()
		select {
		case err = <-done:
			if err == nil || strings.Contains(err.Error(), "private detail") {
				t.Fatal("worker failure was hidden or leaked storage details", err)
			}
		case <-time.After(4 * time.Second):
			t.Fatal("failed worker-only process remained live")
		}
	})
}
