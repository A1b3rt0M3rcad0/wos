package httptransport

import (
	"context"
	"encoding/json"
	"fmt"
	mcptransport "github.com/A1b3rt0M3rcad0/wos/packages/wos-api/mcp"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestReferenceHTTPQueryContract(t *testing.T) {
	handler := newTestHandler(t)
	s := httptest.NewServer(handler)
	defer s.Close()
	client := s.Client()
	base := s.URL + "/api/v1/namespaces/0199ec00-0000-7000-8000-000000000001/outcomes"
	out := doJSON[application.MutationResult[domain.Outcome]](t, client, http.MethodPost, base, []byte(`{"title":"Reference HTTP","desired_state":"Bounded search","priority":"normal"}`), map[string]string{"Idempotency-Key": "refs-outcome-00001"}, http.StatusCreated).Value.Value
	path := base + "/" + out.ID.String() + "/references"
	for _, bad := range []string{"?kind=outcome&limit=0", "?kind=invalid", "?kind=outcome&limit=101", "?kind=outcome&cursor=malformed"} {
		r := doRequest(t, client, http.MethodGet, path+bad, nil, nil)
		if r.StatusCode != http.StatusUnprocessableEntity {
			t.Fatalf("query %s: status %d", bad, r.StatusCode)
		}
		r.Body.Close()
	}
	page := doJSON[application.ReferencePage](t, client, http.MethodGet, path+"?kind=outcome&query=reference&limit=1", nil, nil, http.StatusOK)
	if len(page.Value.Items) != 1 || page.Value.Items[0].Ref.ID != out.ID || page.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("unexpected page: %+v", page)
	}
	resolve := doJSON[application.ReferencePage](t, client, http.MethodGet, path+"?kind=outcome&id="+out.ID.String(), nil, nil, http.StatusOK)
	raw, _ := json.Marshal(resolve.Value)
	if len(resolve.Value.Items) != 1 || len(raw) > 2048 {
		t.Fatalf("unbounded resolution: %s", raw)
	}
	// The HTTP adapter preserves the existing domain-error status convention.
	for i := 0; i < 3; i++ {
		body, _ := json.Marshal(map[string]any{"command": map[string]any{"scope": out.Scope(), "title": "HTTP duplicate task", "priority": "normal", "lifecycle": "todo"}})
		doJSON[application.MutationResult[domain.WorkItem]](t, client, http.MethodPost, s.URL+"/api/v1/commands/create_work_item", body, map[string]string{"Idempotency-Key": fmt.Sprintf("ref-http-work-%04d", i)}, http.StatusOK)
	}
	old := doJSON[application.ReferencePage](t, client, http.MethodGet, path+"?kind=work_item&limit=1", nil, nil, http.StatusOK).Value
	if old.NextCursor == "" {
		t.Fatal("missing task cursor")
	}
	body, _ := json.Marshal(map[string]any{"command": map[string]any{"scope": out.Scope(), "title": "Change revision", "priority": "normal", "lifecycle": "todo"}})
	doJSON[application.MutationResult[domain.WorkItem]](t, client, http.MethodPost, s.URL+"/api/v1/commands/create_work_item", body, map[string]string{"Idempotency-Key": "ref-http-work-change"}, http.StatusOK)
	stale := doRequest(t, client, http.MethodGet, path+"?kind=work_item&cursor="+old.NextCursor, nil, nil)
	if stale.StatusCode != http.StatusConflict {
		t.Fatalf("stale cursor status %d", stale.StatusCode)
	}
	stale.Body.Close()
	// Refresh the expected live revision before comparing public transports.
	page = doJSON[application.ReferencePage](t, client, http.MethodGet, path+"?kind=outcome&query=reference&limit=1", nil, nil, http.StatusOK)
	// Both public adapters expose the same typed query, not a second search model.
	ctx := context.Background()
	id := application.Identity{PrincipalID: handler.auth.Principal(), Actor: handler.auth.Actor()}
	protocol, err := mcptransport.New(handler.service, handler.ids, mcptransport.Options{LocalIdentity: &id})
	if err != nil {
		t.Fatal(err)
	}
	ct, st := mcp.NewInMemoryTransports()
	serverSession, err := protocol.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	clientSession, err := mcp.NewClient(&mcp.Implementation{Name: "reference-parity", Version: "1"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer clientSession.Close()
	result, err := clientSession.CallTool(ctx, &mcp.CallToolParams{Name: "wos_search_references", Arguments: map[string]any{"namespace_id": out.NamespaceID.String(), "outcome_id": out.ID.String(), "kinds": []string{"outcome"}, "text": "reference", "limit": 1}})
	if err != nil || result.IsError {
		t.Fatalf("MCP query failed: %+v %v", result, err)
	}
	raw, err = json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var mcpPage application.ReferencePage
	if err = json.Unmarshal(raw, &mcpPage); err != nil {
		t.Fatal(err)
	}
	if len(mcpPage.Items) != 1 || mcpPage.Items[0] != page.Value.Items[0] || mcpPage.OutcomeRevision != page.Value.OutcomeRevision {
		t.Fatalf("HTTP/MCP query mismatch: %+v %+v", page.Value, mcpPage)
	}
}
