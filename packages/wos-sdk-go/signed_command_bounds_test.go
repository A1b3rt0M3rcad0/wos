package wossdk

import (
	"context"
	"encoding/json"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSignedCommandsReadBoundedOriginalResponse(t *testing.T) {
	scope := d.Scope{NamespaceID: d.ID("0199a555-0000-7000-8000-000000000001"), OutcomeID: d.ID("0199a555-0000-7000-8000-000000000002")}
	payloadSize := 300 << 10
	endpoint := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"value": map[string]any{"acquired": false, "search_complete": true, "reasons": []string{strings.Repeat("x", payloadSize)}}, "command_id": "0199a555-0000-7000-8000-000000000003", "outcome_revision": "1"})
	}))
	defer endpoint.Close()
	client, e := New(endpoint.URL, "", endpoint.Client())
	if e != nil {
		t.Fatal(e)
	}
	cmd := a.AcquireNextSignedWorkContractCommand{Scope: scope, SignerKeyID: scope.NamespaceID, Limit: 1}
	result, e := client.AcquireNextSignedWorkContract(context.Background(), "original-signed-search", cmd)
	if e != nil || len(result.Value.Reasons) != 1 || len(result.Value.Reasons[0]) != payloadSize {
		t.Fatalf("bounded original response truncated: %v", e)
	}
	if _, e = command[a.WorkContractAcquisition](context.Background(), client, "acquire_next_work_contract", "original-legacy-search", a.AcquireNextWorkContractCommand{Scope: scope}); e == nil {
		t.Fatal("legacy command response bound widened")
	}
	payloadSize = d.MaxSignedOperationResultBytes + (64 << 10)
	if _, e = client.AcquireNextSignedWorkContract(context.Background(), "oversized-signed-search", cmd); e == nil || !strings.Contains(e.Error(), "exceeds") {
		t.Fatalf("oversized signed response accepted: %v", e)
	}
}
