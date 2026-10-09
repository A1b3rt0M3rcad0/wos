package httptransport

import (
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
