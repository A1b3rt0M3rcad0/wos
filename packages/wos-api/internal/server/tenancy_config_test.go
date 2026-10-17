package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTenantHostRoutingUsesExactConfiguredHostWithoutFallback(t *testing.T) {
	hits := map[string]int{}
	child := func(name string) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hits[name]++
			if r.URL.Path != "/session" || r.Header.Get("Authorization") != "Bearer scoped-credential" {
				t.Fatal("router changed authentication or URL")
			}
			w.Header().Set("X-Tenant", name)
			w.WriteHeader(http.StatusUnauthorized)
		})
	}
	router := tenantHostRouter(map[string]http.Handler{"a.example.test": child("a"), "b.example.test": child("b")})
	for _, test := range []struct{ host, want string }{
		{"a.example.test", "a"}, {"A.Example.Test:8443", "a"}, {"b.example.test", "b"},
		{"unknown.example.test", ""}, {"a.example.test.evil", ""}, {"a.example.test@evil", ""}, {"", ""},
	} {
		r := httptest.NewRequest(http.MethodPost, "http://localhost/session", nil)
		r.Host = test.host
		r.Header.Set("Authorization", "Bearer scoped-credential")
		r.Header.Set("X-Forwarded-Host", "a.example.test")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		if test.want == "" {
			if w.Code != http.StatusMisdirectedRequest || w.Header().Get("X-Tenant") != "" {
				t.Fatalf("unknown host %q reached a child: %d", test.host, w.Code)
			}
		} else if w.Code != http.StatusUnauthorized || w.Header().Get("X-Tenant") != test.want {
			t.Fatalf("host %q bypassed child authentication", test.host)
		}
	}
	if hits["a"] != 2 || hits["b"] != 1 {
		t.Fatal("unknown hosts fell back to a tenant", hits)
	}
}

func TestHostedTenantConfigurationRejectsAmbiguousOrUnboundMappings(t *testing.T) {
	t.Setenv("WOS_TEST_TENANT_A_TOKEN", strings.Repeat("a", 40))
	cfg := DefaultConfig()
	cfg.Storage.Driver = StorageDriverPostgres
	cfg.Storage.PostgresDSN = "postgres://localhost/test"
	cfg.Auth.Mode = AuthModeAPIToken
	cfg.TenantHosts = []TenantHostConfig{{Host: "a.example.test", TenantID: cfg.Auth.LocalTenantID, BootstrapTokenEnv: "WOS_TEST_TENANT_A_TOKEN"}}
	if err := cfg.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, change := range []func(*Config){
		func(c *Config) { c.TenantHosts[0].Host = "*.example.test" },
		func(c *Config) { c.TenantHosts[0].Host = "https://a.example.test" },
		func(c *Config) { c.TenantHosts[0].TenantID = "caller-supplied-schema" },
		func(c *Config) { c.TenantHosts[0].BootstrapTokenEnv = "missing-secret" },
		func(c *Config) { c.TenantHosts = append(c.TenantHosts, c.TenantHosts[0]) },
		func(c *Config) {
			c.TenantHosts = append(c.TenantHosts, TenantHostConfig{Host: "second.example.test", TenantID: "0199d000-0000-7000-8000-000000000002", BootstrapTokenEnv: "WOS_TEST_TENANT_A_TOKEN"})
		},
		func(c *Config) { c.Storage.TenantID = c.Auth.LocalTenantID },
		func(c *Config) { c.Storage.PostgresSchema = "public" },
		func(c *Config) { c.Storage.PostgresCatalogSchema = "public" },
		func(c *Config) { c.Storage.PostgresCatalogSchema = "pg_catalog" },
		func(c *Config) { c.Storage.PostgresCatalogSchema = "wos_t_data" },
	} {
		invalid := cfg
		invalid.TenantHosts = append([]TenantHostConfig(nil), cfg.TenantHosts...)
		change(&invalid)
		if invalid.Validate() == nil {
			t.Fatal("accepted unsafe hosted Tenant configuration")
		}
	}
	child := cfg.hostedTenantConfig(cfg.TenantHosts[0])
	if child.Storage.TenantID != cfg.Auth.LocalTenantID || child.Auth.BootstrapToken != strings.Repeat("a", 40) || len(child.TenantHosts) != 0 {
		t.Fatal("child lost its fixed binding or credential reference")
	}
}

func TestTenantHostEnvironmentParsingIsStrict(t *testing.T) {
	for _, raw := range []string{"", "null", "[]", `{}`, `[{"host":"a.test","unknown":"value"}]`, `[] {}`, strings.Repeat(" ", 65<<10)} {
		if _, invalid := parseTenantHosts(raw); !invalid {
			t.Fatalf("accepted invalid host configuration %q", raw[:min(len(raw), 80)])
		}
	}
	valid := `[{"host":"a.test","tenant_id":"0199d000-0000-7000-8000-000000000001"}]`
	if hosts, invalid := parseTenantHosts(valid); invalid || len(hosts) != 1 {
		t.Fatal("rejected valid host configuration")
	}
}

func TestPostgresRuntimeCannotSilentlyUseSharedPublicTables(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Storage.Driver = StorageDriverPostgres
	cfg.Storage.PostgresDSN = "postgres://localhost/test"
	cfg.Auth.Mode = AuthModeAPIToken
	cfg.Auth.BootstrapToken = strings.Repeat("a", 40)
	if cfg.Validate() == nil {
		t.Fatal("unbound PostgreSQL runtime was accepted")
	}
	cfg.Storage.TenantID = cfg.Auth.LocalTenantID
	if err := cfg.Validate(); err != nil {
		t.Fatal("explicit Tenant did not provide bootstrap binding", err)
	}
	cfg.Auth.BootstrapTenantID = "0199d000-0000-7000-8000-000000000002"
	if cfg.Validate() == nil {
		t.Fatal("conflicting storage and credential Tenant bindings accepted")
	}
}

func TestHostedTenantConnectionBudgetIncludesEveryPoolAndLease(t *testing.T) {
	t.Setenv("WOS_TEST_BUDGET_A_TOKEN", strings.Repeat("a", 40))
	t.Setenv("WOS_TEST_BUDGET_B_TOKEN", strings.Repeat("b", 40))
	cfg := DefaultConfig()
	cfg.Storage.Driver = StorageDriverPostgres
	cfg.Storage.PostgresDSN = "postgres://unused.invalid/test"
	cfg.Storage.PostgresMaxOpenConnections = 8
	cfg.Auth.Mode = AuthModeAPIToken
	cfg.TenantHosts = []TenantHostConfig{
		{Host: "a.example.test", TenantID: "0199d000-0000-7000-8000-000000000001", BootstrapTokenEnv: "WOS_TEST_BUDGET_A_TOKEN"},
		{Host: "b.example.test", TenantID: "0199d000-0000-7000-8000-000000000002", BootstrapTokenEnv: "WOS_TEST_BUDGET_B_TOKEN"},
	}
	report, err := cfg.DeploymentDiagnostics("shared", DeploymentDeclarations{ExpectedProcesses: 3, DatabaseConnectionBudget: 80, ReservedConnections: 20})
	if err != nil {
		t.Fatal(err)
	}
	if report.HostedTenants != 2 || report.CatalogPoolLimit != 4 || report.LeaseConnections != 2 || report.ProcessConnectionLimit != 22 || report.ExpectedConnectionDemand != 86 {
		t.Fatal("hosted runtime omitted data pools, catalog pool or lifecycle leases", report)
	}
	for _, advisory := range report.Advisories {
		if advisory.Code == "database_connection_budget" {
			return
		}
	}
	t.Fatal("hosted runtime did not report the exhausted shared database budget")
}
