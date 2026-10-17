package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
)

func TestHostedPostgresTenantsIsolateCredentialsSessionsAndPersistentState(t *testing.T) {
	dsn := os.Getenv("WOS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("real PostgreSQL required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	ids := uuidV7Generator{}
	firstID, err := ids.NewID()
	if err != nil {
		t.Fatal(err)
	}
	secondID, err := ids.NewID()
	if err != nil {
		t.Fatal(err)
	}
	control := "wos_host_" + strings.ReplaceAll(firstID.String(), "-", "")
	catalog, err := postgres.OpenTenantCatalog(ctx, dsn, postgres.CatalogOptions{Schema: control, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	var placements []postgres.TenantPlacement
	t.Cleanup(func() {
		_ = catalog.Close()
		db, err := sql.Open("pgx", dsn)
		if err != nil {
			t.Error(err)
			return
		}
		defer db.Close()
		cleanup, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, placement := range placements {
			if _, err := db.ExecContext(cleanup, "DROP SCHEMA IF EXISTS "+placement.Schema+" CASCADE"); err != nil {
				t.Error(err)
			}
		}
		if _, err := db.ExecContext(cleanup, "DROP SCHEMA IF EXISTS "+control+" CASCADE"); err != nil {
			t.Error(err)
		}
	})
	for _, id := range []domain.ID{firstID, secondID} {
		placement, err := catalog.ProvisionTenant(ctx, postgres.TenantProvision{TenantID: id, Name: "Hosted " + id.String(), Reason: "host routing isolation acceptance"}, func(ctx context.Context, p postgres.TenantPlacement) error {
			store, err := postgres.Open(dsn, p.InitializerOptions())
			if err != nil {
				return err
			}
			return store.Close()
		})
		if err != nil {
			t.Fatal(err)
		}
		placements = append(placements, placement)
	}
	t.Setenv("WOS_HOST_FIRST_TOKEN", strings.Repeat("a", 40))
	t.Setenv("WOS_HOST_SECOND_TOKEN", strings.Repeat("b", 40))
	cfg := DefaultConfig()
	cfg.Auth.Mode = AuthModeAPIToken
	cfg.Storage.Driver, cfg.Storage.PostgresDSN, cfg.Storage.PostgresCatalogSchema = StorageDriverPostgres, dsn, control
	cfg.Storage.PostgresMaxOpenConnections = 2
	cfg.HTTP.MaxInflightRequests = 1
	cfg.MCP.Enabled = true
	cfg.TenantHosts = []TenantHostConfig{
		{Host: "first.example.test", TenantID: firstID.String(), BootstrapTokenEnv: "WOS_HOST_FIRST_TOKEN"},
		{Host: "second.example.test", TenantID: secondID.String(), BootstrapTokenEnv: "WOS_HOST_SECOND_TOKEN"},
	}
	runtime, err := OpenRuntime(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	request := func(host, method, path, token, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "https://"+host+path, bytes.NewBufferString(body))
		if token != "" {
			r.Header.Set("Authorization", "Bearer "+token)
		}
		if cookie != nil {
			r.AddCookie(cookie)
		}
		if method == http.MethodPost {
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("Idempotency-Key", "host-routing-create-outcome-0001")
		}
		w := httptest.NewRecorder()
		runtime.Handler().ServeHTTP(w, r)
		return w
	}
	for _, path := range []string{"/app/", "/app/config", "/app/app.js"} {
		w := request("first.example.test", http.MethodGet, path, "", "", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("hosted web route %s: %d %s", path, w.Code, w.Body.String())
		}
	}
	var config struct {
		TenantBound bool   `json:"tenant_bound"`
		TenantID    string `json:"tenant_id"`
	}
	if err := json.Unmarshal(request("first.example.test", http.MethodGet, "/app/config", "", "", nil).Body.Bytes(), &config); err != nil || !config.TenantBound || config.TenantID != firstID.String() {
		t.Fatal("browser did not receive fixed tenant binding", config, err)
	}
	for _, test := range []struct {
		host, token string
		status      int
	}{
		{"first.example.test", strings.Repeat("a", 40), http.StatusOK},
		{"second.example.test", strings.Repeat("b", 40), http.StatusOK},
		{"second.example.test", strings.Repeat("a", 40), http.StatusUnauthorized},
		{"unknown.example.test", strings.Repeat("a", 40), http.StatusMisdirectedRequest},
	} {
		w := request(test.host, http.MethodGet, "/api/v1/tenants", test.token, "", nil)
		if w.Code != test.status {
			t.Fatalf("credential routing: %s got%d want%d %s", test.host, w.Code, test.status, w.Body.String())
		}
	}
	w := request("first.example.test", http.MethodPost, "/session", "", `{"token":"`+strings.Repeat("a", 40)+`"}`, nil)
	if w.Code != http.StatusOK || len(w.Result().Cookies()) != 1 {
		t.Fatal("session failed", w.Code, w.Body.String())
	}
	cookie := w.Result().Cookies()[0]
	if cookie.Domain != "" || cookie.Path != "/" || !cookie.HttpOnly {
		t.Fatal("session cookie lost host-only isolation", cookie)
	}
	if w = request("second.example.test", http.MethodGet, "/api/v1/tenants", "", "", cookie); w.Code != http.StatusUnauthorized {
		t.Fatal("session crossed Tenant schemas", w.Code, w.Body.String())
	}
	createPath := "/api/v1/tenants/" + firstID.String() + "/outcomes"
	started, released := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	release := func() { releaseOnce.Do(func() { close(released) }) }
	defer release()
	held := httptest.NewRequest(http.MethodPost, "https://first.example.test"+createPath, nil)
	held.Header.Set("Authorization", "Bearer "+strings.Repeat("a", 40))
	held.Header.Set("Content-Type", "application/json")
	held.Header.Set("Idempotency-Key", "host-admission-held-command-0001")
	held.Body = &tenantHeldBody{reader: strings.NewReader(`{"title":"Concurrent plan","desired_state":"one process admission budget","priority":"normal"}`), started: started, released: released}
	heldResponse := make(chan *httptest.ResponseRecorder, 1)
	go func() {
		response := httptest.NewRecorder()
		runtime.Handler().ServeHTTP(response, held)
		heldResponse <- response
	}()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("first hosted request did not enter body read")
	}
	w = request("second.example.test", http.MethodGet, "/api/v1/tenants", strings.Repeat("b", 40), "", nil)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatal("hosted children bypassed process-wide admission", w.Code, w.Body.String())
	}
	release()
	select {
	case response := <-heldResponse:
		if response.Code != http.StatusCreated {
			t.Fatal("held hosted request failed", response.Code, response.Body.String())
		}
	case <-time.After(5 * time.Second):
		t.Fatal("held hosted request did not finish")
	}
	w = request("first.example.test", http.MethodPost, createPath, strings.Repeat("a", 40), `{"title":"First private plan","desired_state":"isolated result","priority":"normal"}`, nil)
	if w.Code != http.StatusCreated {
		t.Fatal("create bound outcome", w.Code, w.Body.String())
	}
	w = request("second.example.test", http.MethodGet, "/api/v1/tenants/"+secondID.String()+"/outcomes", strings.Repeat("b", 40), "", nil)
	if w.Code != http.StatusOK || strings.Contains(w.Body.String(), "First private plan") {
		t.Fatal("persistent state crossed schema", w.Code, w.Body.String())
	}
	w = request("first.example.test", http.MethodGet, "/api/v1/tenants/"+secondID.String()+"/outcomes", strings.Repeat("a", 40), "", nil)
	if w.Code != http.StatusForbidden {
		t.Fatal("payload Tenant bypassed bound authorization", w.Code, w.Body.String())
	}
	w = request("second.example.test", http.MethodPost, "/mcp", strings.Repeat("a", 40), `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"cross-tenant","version":"1"}}}`, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatal("MCP used another Tenant credential", w.Code, w.Body.String())
	}
	if _, err = catalog.ChangeTenantState(ctx, firstID, placements[0].Version, postgres.TenantSuspended, "must refuse live host"); err == nil {
		t.Fatal("suspended Tenant while pool and lease were live")
	}
	if err = runtime.Close(); err != nil {
		t.Fatal(err)
	}
	suspended, err := catalog.ChangeTenantState(ctx, firstID, placements[0].Version, postgres.TenantSuspended, "drained host accepted suspension")
	if err != nil {
		t.Fatal(err)
	}
	if forbidden, err := OpenRuntime(cfg); err == nil {
		forbidden.Close()
		t.Fatal("restarted suspended Tenant")
	}
	if _, err = catalog.ChangeTenantState(ctx, firstID, suspended.Version, postgres.TenantActive, "resume same persistent state"); err != nil {
		t.Fatal(err)
	}
	restarted, err := OpenRuntime(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer restarted.Close()
	runtime = restarted
	w = request("first.example.test", http.MethodGet, createPath, strings.Repeat("a", 40), "", cookie)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "First private plan") {
		t.Fatal("resume lost persistent state or original credential", w.Code, w.Body.String())
	}
	// A failed held lifecycle lease must stop the composed runtime, even though
	// its remaining data connections continue protecting the schema from DROP.
	if err := restarted.children[0].lease.Close(); err != nil {
		t.Fatal(err)
	}
	supervise, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if err := restarted.runBackground(supervise); err == nil || supervise.Err() != nil {
		t.Fatal("host supervision did not report the lost Tenant lifecycle lease", err)
	}
}

type tenantHeldBody struct {
	reader   io.Reader
	started  chan struct{}
	released <-chan struct{}
	once     sync.Once
}

func (body *tenantHeldBody) Read(data []byte) (int, error) {
	body.once.Do(func() { close(body.started) })
	<-body.released
	return body.reader.Read(data)
}
func (body *tenantHeldBody) Close() error { return nil }
