package wossdk

import (
	"context"
	"encoding/base64"
	"encoding/json"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOperationExpansionReadsFullBoundedRegistryResult(t *testing.T) {
	scope := d.Scope{NamespaceID: d.ID("0199a555-0000-7000-8000-000000000001"), OutcomeID: d.ID("0199a555-0000-7000-8000-000000000002")}
	payload := []byte(strings.Repeat("x", d.MaxSignedOperationResultBytes))
	response := a.SignedStateResult{NamespaceID: scope.NamespaceID, Resource: "operation", OperationPayload: base64.StdEncoding.EncodeToString(payload)}
	oversized := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if oversized {
			_, _ = w.Write([]byte(strings.Repeat("x", base64.StdEncoding.EncodedLen(d.MaxSignedOperationResultBytes)+(64<<10)+1)))
			return
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	client, e := New(server.URL, "", server.Client())
	if e != nil {
		t.Fatal(e)
	}
	query := a.SignedStateQuery{Scope: scope, Resource: "operation", IdempotencyKey: "frozen-intention"}
	result, e := client.ReadSignedState(context.Background(), query)
	if e != nil {
		t.Fatal(e)
	}
	decoded, e := base64.StdEncoding.DecodeString(result.OperationPayload)
	if e != nil || string(decoded) != string(payload) {
		t.Fatal("full original result changed")
	}
	if e = client.do(context.Background(), http.MethodGet, "/ordinary-query", "", nil, &result); e == nil {
		t.Fatal("legacy response limit widened")
	}
	oversized = true
	if _, e = client.ReadSignedState(context.Background(), query); e == nil || !strings.Contains(e.Error(), "exceeds") {
		t.Fatalf("oversized expansion accepted: %v", e)
	}
}
