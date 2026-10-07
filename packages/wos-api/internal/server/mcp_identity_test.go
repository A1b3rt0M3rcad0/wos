package server

import (
	"context"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	wossdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

type bearerTransport struct {
	token string
	next  http.RoundTripper
}

func (t bearerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	copy := r.Clone(r.Context())
	copy.Header = r.Header.Clone()
	copy.Header.Set("Authorization", "Bearer "+t.token)
	return t.next.RoundTrip(copy)
}
func TestRemoteMCPRevalidatesGrantsAndAuthenticatedAuthorship(t *testing.T) {
	forEachRuntimeStorage(t, func(t *testing.T, cfg Config) {
		cfg.MCP.Enabled = true
		cfg.Auth.Mode = AuthModeAPIToken
		cfg.Auth.BootstrapToken = strings.Repeat("a", 40)
		cfg.Auth.BootstrapNamespaceID = "0199d120-0000-7000-8000-000000000001"
		cfg.Auth.BootstrapNamespaceName = "Remote"
		server, err := OpenRuntime(cfg)
		if err != nil {
			t.Fatal(err)
		}
		defer server.Close()
		httpServer := httptest.NewServer(server.Handler())
		defer httpServer.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		security := application.SecurityService{Store: server.store, Clock: systemClock{}, IDs: uuidV7Generator{}}
		human, err := security.Authenticate(ctx, cfg.Auth.BootstrapToken)
		if err != nil {
			t.Fatal(err)
		}
		humanCtx := application.WithIdentity(ctx, human)
		ns, _ := domain.ParseID(cfg.Auth.BootstrapNamespaceID)
		if err = security.SetGrant(humanCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "agent", Permissions: []ports.Permission{ports.PermissionStateRead, ports.PermissionWorkWrite}}); err != nil {
			t.Fatal(err)
		}
		credential, token, err := security.IssueCredential(humanCtx, ns, "agent", domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "independent", ID: "agent-1"}, time.Now().Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		httpClient := &http.Client{Transport: bearerTransport{token: token, next: httpServer.Client().Transport}, Timeout: 10 * time.Second}
		client := mcp.NewClient(&mcp.Implementation{Name: "remote-agent", Version: "1"}, nil)
		session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: httpServer.URL + cfg.MCP.Path, HTTPClient: httpClient, DisableStandaloneSSE: true}, nil)
		if err != nil {
			t.Fatal(err)
		}
		defer session.Close()
		humanSDK, _ := wossdk.New(httpServer.URL, cfg.Auth.BootstrapToken, httpServer.Client())
		key, _ := wossdk.NewIdempotencyKey()
		outcome, err := humanSDK.CreateOutcome(ctx, key, application.CreateOutcomeCommand{NamespaceID: ns, Title: "Mixed human agent state", DesiredState: "Authenticated shared context", Priority: domain.PriorityNormal})
		if err != nil {
			t.Fatal(err)
		}
		args := map[string]any{"idempotency_key": "remote-work-create-00001", "command": map[string]any{"scope": outcome.Value.Scope(), "title": "Agent writes shared work", "priority": "normal", "lifecycle": "todo"}}
		created, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "wos_create_work_item", Arguments: args})
		if err != nil || created.IsError {
			t.Fatalf("agent create: %v %+v", err, created)
		}
		raw, _ := json.Marshal(created.StructuredContent)
		var work application.MutationResult[domain.WorkItem]
		if err = json.Unmarshal(raw, &work); err != nil {
			t.Fatal(err)
		}
		facts, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "wos_get_timeline", Arguments: map[string]any{"namespace_id": ns, "outcome_id": outcome.Value.ID, "principal_id": "agent"}})
		if err != nil || facts.IsError {
			t.Fatal("timeline", err)
		}
		raw, _ = json.Marshal(facts.StructuredContent)
		if !strings.Contains(string(raw), "agent-1") {
			t.Fatalf("authenticated actor missing: %s", raw)
		}
		resource, err := session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "wos://namespaces/" + ns.String() + "/outcomes/" + outcome.Value.ID.String() + "/continuity"})
		if err != nil || len(resource.Contents) != 1 {
			t.Fatal("authorized continuity resource", err)
		}
		cross, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "wos_search_outcomes", Arguments: map[string]any{"namespace_id": "0199d120-0000-7000-8000-000000000002"}})
		if err != nil || !cross.IsError {
			t.Fatal("cross namespace tool was not denied", err)
		}
		if err = security.SetGrant(humanCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "agent", Permissions: []ports.Permission{ports.PermissionStateRead}}); err != nil {
			t.Fatal(err)
		}
		replay, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "wos_create_work_item", Arguments: args})
		if err != nil || !replay.IsError {
			t.Fatal("long-lived MCP session bypassed revoked write grant", err)
		}
		if err = security.RevokeCredential(humanCtx, ns, credential.ID); err != nil {
			t.Fatal(err)
		}
		if _, err = session.ReadResource(ctx, &mcp.ReadResourceParams{URI: "wos://namespaces/" + ns.String() + "/outcomes/" + outcome.Value.ID.String() + "/continuity"}); err == nil {
			t.Fatal("long-lived session resource survived revocation")
		}

	})
}
func TestRealStdioMCPSubprocess(t *testing.T) {
	if testing.Short() {
		t.Skip("stdio subprocess disabled in short mode")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cwd, _ := os.Getwd()
	root := filepath.Clean(filepath.Join(cwd, "../../../.."))
	command := exec.Command(filepath.Join(runtime.GOROOT(), "bin/go"), "run", "./packages/wos-api/cmd/wos", "mcp", "stdio")
	command.Dir = root
	command.Env = append(os.Environ(), "WOS_AUTH_MODE=local", "WOS_LISTEN=127.0.0.1:8080", "WOS_SQLITE_PATH="+filepath.Join(t.TempDir(), "stdio.db"), "WOS_STORAGE_DRIVER=sqlite")
	client := mcp.NewClient(&mcp.Implementation{Name: "independent-stdio-client", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.CommandTransport{Command: command}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil || len(tools.Tools) < 80 {
		t.Fatal("stdio discovery", err)
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "wos_create_outcome", Arguments: map[string]any{"idempotency_key": "stdio-create-outcome-001", "command": map[string]any{"namespace_id": "0199d130-0000-7000-8000-000000000001", "title": "Stdio outcome", "desired_state": "Independent consumer uses real protocol", "priority": "normal"}}})
	if err != nil || result.IsError {
		t.Fatal("stdio mutation", err)
	}
}
