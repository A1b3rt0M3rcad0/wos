package server

import (
	"context"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/http/httptest"
	"path/filepath"
	"testing"
	"time"
)

func TestRealMCPClientSharesDurableStateWithHTTP(t *testing.T) {
	cfg := DefaultConfig()
	cfg.MCP.Enabled = true
	cfg.Storage.SQLitePath = filepath.Join(t.TempDir(), "mcp.db")
	runtime, err := OpenRuntime(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer runtime.Close()
	httpServer := httptest.NewServer(runtime.Handler())
	defer httpServer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client := mcp.NewClient(&mcp.Implementation{Name: "independent-test-client", Version: "1"}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: httpServer.URL + cfg.MCP.Path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	tools, err := session.ListTools(ctx, &mcp.ListToolsParams{})
	if err != nil {
		t.Fatal(err)
	}
	if len(tools.Tools) < 76 {
		t.Fatalf("only %d tools advertised", len(tools.Tools))
	}
	ns := "0199d020-0000-7000-8000-000000000001"
	arguments := map[string]any{"idempotency_key": "mcp-create-outcome-0001", "command": map[string]any{"namespace_id": ns, "title": "MCP shared outcome", "description": "independent client", "desired_state": "human and agent can continue", "priority": "normal"}}
	created, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "wos_create_outcome", Arguments: arguments})
	if err != nil {
		t.Fatal(err)
	}
	if created.IsError {
		t.Fatalf("create failed: %+v", created.Content)
	}
	var result application.MutationResult[domain.Outcome]
	raw, err := json.Marshal(created.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	if result.Value.ID.IsZero() {
		t.Fatalf("missing structured outcome: %s", raw)
	}
	replay, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "wos_create_outcome", Arguments: arguments})
	if err != nil || replay.IsError {
		t.Fatalf("replay error %v", err)
	}
	raw, _ = json.Marshal(replay.StructuredContent)
	var replayed application.MutationResult[domain.Outcome]
	json.Unmarshal(raw, &replayed)
	if !replayed.IdempotentReplay || replayed.Value.ID != result.Value.ID {
		t.Fatal("MCP replay changed identity")
	}
	discovered, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "wos_search_outcomes", Arguments: map[string]any{"namespace_id": ns, "text": "shared"}})
	if err != nil || discovered.IsError {
		t.Fatalf("discover error %v", err)
	}
	raw, _ = json.Marshal(discovered.StructuredContent)
	if string(raw) == "null" {
		t.Fatal("query lacks structured data")
	}
	resp, err := httpServer.Client().Get(httpServer.URL + cfg.HTTP.Prefix + "/namespaces/" + ns + "/outcomes/" + result.Value.ID.String())
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("HTTP could not read MCP outcome: %d", resp.StatusCode)
	}
	var read application.ReadResult[domain.Outcome]
	if err = json.NewDecoder(resp.Body).Decode(&read); err != nil {
		t.Fatal(err)
	}
	if read.Value.ID != result.Value.ID || read.Value.Title != result.Value.Title {
		t.Fatal("HTTP/MCP state diverged")
	}
	invalid, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "wos_activate_outcome", Arguments: map[string]any{"idempotency_key": "mcp-stale-outcome-0001", "command": map[string]any{"scope": map[string]any{"namespace_id": ns, "outcome_id": result.Value.ID}, "expected_version": 99}}})
	if err != nil {
		t.Fatal(err)
	}
	if !invalid.IsError {
		t.Fatal("MCP accepted stale version")
	}
}
