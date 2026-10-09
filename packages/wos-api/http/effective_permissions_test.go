package httptransport

import (
	"context"
	"encoding/json"
	mcptransport "github.com/A1b3rt0M3rcad0/wos/packages/wos-api/mcp"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/http"
	"net/http/httptest"
	"testing"
)

// Effective permission hints are additive and non-cacheable. The public
// transport must not claim a capability query authorizes execution.
func TestEffectivePermissionHTTPQuery(t *testing.T) {
	h := newTestHandler(t)
	server := httptest.NewServer(h)
	defer server.Close()
	path := server.URL + "/api/v1/namespaces/0199ec00-0000-7000-8000-000000000001/effective-permissions"
	v := doJSON[struct {
		Permissions []string `json:"permissions"`
		Authority   bool     `json:"execution_authority"`
	}](t, server.Client(), http.MethodGet, path, nil, nil, http.StatusOK)
	if len(v.Value.Permissions) == 0 || v.Value.Authority || v.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("invalid presentation hints: %+v", v)
	}
	r := doRequest(t, server.Client(), http.MethodGet, path+"?outcome_id=invalid", nil, nil)
	defer r.Body.Close()
	if r.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid scope: %d", r.StatusCode)
	}
}

func TestEffectivePermissionMCPParity(t *testing.T) {
	h := newTestHandler(t)
	server := httptest.NewServer(h)
	defer server.Close()
	ns := "0199ec00-0000-7000-8000-000000000001"
	httpResult := doJSON[map[string]any](t, server.Client(), http.MethodGet, server.URL+"/api/v1/namespaces/"+ns+"/effective-permissions", nil, nil, http.StatusOK).Value
	ctx := context.Background()
	identity := application.Identity{PrincipalID: h.auth.Principal(), Actor: h.auth.Actor()}
	protocol, err := mcptransport.New(h.service, h.ids, mcptransport.Options{LocalIdentity: &identity})
	if err != nil {
		t.Fatal(err)
	}
	ct, st := mcp.NewInMemoryTransports()
	ss, err := protocol.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer ss.Close()
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "permission-parity", Version: "1"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer cs.Close()
	result, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "wos_effective_permissions", Arguments: map[string]any{"namespace_id": ns}})
	if err != nil || result.IsError {
		t.Fatalf("query failed: %+v %v", result, err)
	}
	got, _ := json.Marshal(result.StructuredContent)
	want, _ := json.Marshal(httpResult)
	if string(got) != string(want) {
		t.Fatalf("permission transport mismatch: %s %s", got, want)
	}
}
