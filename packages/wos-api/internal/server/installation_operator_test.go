package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestInstallationMetricsAndProvisioningRequireExplicitOperator(t *testing.T) {
	forEachRuntimeStorage(t, func(t *testing.T, cfg Config) {
		cfg.MCP.Enabled = true
		cfg.Auth.Mode = AuthModeAPIToken
		cfg.Auth.BootstrapToken = strings.Repeat("b", 40)
		cfg.Auth.BootstrapNamespaceID = "0199d098-0000-7000-8000-000000000011"
		cfg.Auth.BootstrapNamespaceName = "Shared installation"
		runtime, err := OpenRuntime(cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer runtime.Close()
		security := application.SecurityService{Store: runtime.store, Clock: systemClock{}, IDs: uuidV7Generator{}}
		ctx := context.Background()
		operator, err := security.Authenticate(ctx, cfg.Auth.BootstrapToken)
		if err != nil {
			t.Fatal(err)
		}
		operatorCtx := application.WithIdentity(ctx, operator)
		ns := operator.NamespaceID
		if err = security.SetGrant(operatorCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "tenant", Permissions: []ports.Permission{ports.PermissionStateRead, ports.PermissionNamespaceAdmin}}); err != nil {
			t.Fatal(err)
		}
		_, tenantToken, err := security.IssueCredential(operatorCtx, ns, "tenant", domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "test", ID: "tenant"}, time.Now().Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		for _, tc := range []struct {
			token  string
			status int
		}{{"", http.StatusUnauthorized}, {tenantToken, http.StatusForbidden}, {cfg.Auth.BootstrapToken, http.StatusOK}} {
			r := httptest.NewRequest(http.MethodGet, "/metrics", nil)
			if tc.token != "" {
				r.Header.Set("Authorization", "Bearer "+tc.token)
			}
			w := httptest.NewRecorder()
			runtime.Handler().ServeHTTP(w, r)
			if w.Code != tc.status {
				t.Fatalf("metrics: got %d want %d: %s", w.Code, tc.status, w.Body.String())
			}
		}
		payload, _ := json.Marshal(application.AdministrativeIntent{NamespaceID: ns, ExpectedVersion: 1, Operation: "create_namespace", NamespaceName: "Created through HTTP"})
		r := httptest.NewRequest(http.MethodPost, "/api/v1/security/commands", bytes.NewReader(payload))
		r.Header.Set("Authorization", "Bearer "+tenantToken)
		r.Header.Set("Idempotency-Key", "tenant-http-create-0001")
		r.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		runtime.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusForbidden {
			t.Fatalf("tenant HTTP provision returned %d: %s", w.Code, w.Body.String())
		}
		// The authenticated MCP adapter must enforce exactly the same boundary.
		host := httptest.NewServer(runtime.Handler())
		defer host.Close()
		client := mcp.NewClient(&mcp.Implementation{Name: "operator-boundary", Version: "1"}, nil)
		httpClient := &http.Client{Transport: bearerTransport{token: tenantToken, next: host.Client().Transport}, Timeout: 10 * time.Second}
		session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: host.URL + cfg.MCP.Path, HTTPClient: httpClient, DisableStandaloneSSE: true}, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer session.Close()
		denied, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "wos_administer_namespace", Arguments: map[string]any{"idempotency_key": "tenant-mcp-create-0001", "command": map[string]any{"namespace_id": ns, "expected_namespace_version": 1, "operation": "create_namespace", "namespace_name": "Forbidden"}}})
		if err != nil || !denied.IsError {
			t.Fatalf("tenant MCP provision was not denied: %+v %v", denied, err)
		}
		r = httptest.NewRequest(http.MethodPost, "/api/v1/security/commands", bytes.NewReader(payload))
		r.Header.Set("Authorization", "Bearer "+cfg.Auth.BootstrapToken)
		r.Header.Set("Idempotency-Key", "operator-http-create-0001")
		r.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()
		runtime.Handler().ServeHTTP(w, r)
		if w.Code != http.StatusOK {
			t.Fatalf("operator HTTP provision returned %d: %s", w.Code, w.Body.String())
		}
	})
}

func TestOfflineOperatorUpgradeIsExplicitAuditedAndDoesNotBootstrap(t *testing.T) {
	forEachRuntimeStorage(t, func(t *testing.T, cfg Config) {
		cfg.Auth.Mode = AuthModeAPIToken
		cfg.Auth.BootstrapToken = "legacy-bootstrap-credential-at-least-32"
		cfg.Auth.BootstrapNamespaceID = "0199d098-0000-7000-8000-000000000012"
		cfg.Auth.BootstrapNamespaceName = "Legacy"
		var store runtimeStore
		var err error
		if cfg.Storage.Driver == StorageDriverPostgres {
			store, err = postgres.Open(cfg.Storage.PostgresDSN, postgres.Options{MigrateOnOpen: true})
		} else {
			store, err = sqlite.Open(cfg.Storage.SQLitePath, sqlite.Options{MigrateOnOpen: true})
		}
		if err != nil {
			t.Fatal(err)
		}
		security := application.SecurityService{Store: store, Clock: systemClock{}, IDs: uuidV7Generator{}}
		ns, _ := domain.ParseID(cfg.Auth.BootstrapNamespaceID)
		if err = security.Bootstrap(context.Background(), ports.Namespace{ID: ns, Name: "Legacy"}, "legacy", cfg.Auth.BootstrapToken); err != nil {
			t.Fatal(err)
		}
		// This direct composition setup models pre-upgrade persisted grants.
		if err = store.PutGrant(context.Background(), ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "legacy", Permissions: []ports.Permission{ports.PermissionStateRead, ports.PermissionNamespaceAdmin}}); err != nil {
			t.Fatal(err)
		}
		if err = store.Close(); err != nil {
			t.Fatal(err)
		}
		intent := InstallationOperatorIntent{NamespaceID: ns, PrincipalID: "legacy", ExpectedVersion: 1, IdempotencyKey: "legacy-operator-upgrade-0001", Reason: "Explicitly authorize the existing installation owner"}
		if _, err = AuthorizeInstallationOperator(context.Background(), cfg, intent); err == nil {
			t.Fatal("upgrade accepted missing drained-writer assertion")
		}
		intent.DrainedWriters = true
		result, err := AuthorizeInstallationOperator(context.Background(), cfg, intent)
		if err != nil || result.NamespaceVersion != 2 {
			t.Fatalf("promotion: %+v %v", result, err)
		}
		replay, err := AuthorizeInstallationOperator(context.Background(), cfg, intent)
		if err != nil || !replay.IdempotentReplay || replay.NamespaceVersion != 2 {
			t.Fatalf("replay: %+v %v", replay, err)
		}
		intent.Reason = "Different reason is a different intent"
		if _, err = AuthorizeInstallationOperator(context.Background(), cfg, intent); err == nil {
			t.Fatal("changed intent reused operator receipt")
		}
		if cfg.Storage.Driver == StorageDriverPostgres {
			store, err = postgres.Open(cfg.Storage.PostgresDSN, postgres.Options{})
		} else {
			store, err = sqlite.Open(cfg.Storage.SQLitePath, sqlite.Options{})
		}
		if err != nil {
			t.Fatal(err)
		}
		defer store.Close()
		security.Store = store
		id, err := security.Authenticate(context.Background(), cfg.Auth.BootstrapToken)
		if err != nil {
			t.Fatal(err)
		}
		ctx := application.WithIdentity(context.Background(), id)
		if err = security.RequireInstallation(ctx, ports.PermissionInstallationObserve); err != nil {
			t.Fatal(err)
		}
		snapshot, err := security.AdministrativeSnapshot(ctx, ns)
		if err != nil || len(snapshot.Credentials) != 1 || len(snapshot.Audit) != 1 || snapshot.Audit[0].Operation != "authorize_installation_operator" || !strings.Contains(snapshot.Audit[0].Target, "Explicitly authorize") {
			t.Fatalf("promotion audit or credential changed: %+v %v", snapshot, err)
		}
		// Normal bootstrap still cannot restore an explicitly revoked operator grant.
		if err = store.PutGrant(context.Background(), ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "legacy", Permissions: []ports.Permission{ports.PermissionStateRead, ports.PermissionNamespaceAdmin}}); err != nil {
			t.Fatal(err)
		}
		if err = security.Bootstrap(context.Background(), ports.Namespace{ID: ns, Name: "Legacy"}, "legacy", cfg.Auth.BootstrapToken); err != nil {
			t.Fatal(err)
		}
		if err = security.RequireInstallation(ctx, ports.PermissionInstallationObserve); err == nil {
			t.Fatal("restart restored operator authority")
		}
	})
}

func TestInstallationOperatorIntentRejectsUnknownOrOversizedFields(t *testing.T) {
	for _, raw := range []string{`{"drained_writers":true,"unknown":true}`, strings.Repeat(" ", 16<<10) + "{}"} {
		if _, err := DecodeInstallationOperatorIntent(strings.NewReader(raw)); err == nil {
			t.Fatal("invalid operator intent decoded")
		}
	}
}
