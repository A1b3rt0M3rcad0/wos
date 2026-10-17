package wossdk_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

func TestCommandReceiptObservationPreservesCompactCommitWithoutClaimingFullResult(t *testing.T) {
	ns := d.MustParseID("0199ff42-0000-7000-8000-000000000001")
	id := d.MustParseID("0199ff42-0000-7000-8000-000000000002")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/tenants/"+ns.String()+"/commands/"+id.String() {
			t.Error("receipt requested from wrong endpoint")
		}
		json.NewEncoder(w).Encode(map[string]any{"command_id": id, "outcome_revision": 3, "result_omitted": true})
	}))
	defer server.Close()
	client, err := sdk.New(server.URL, "", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	observed, err := client.GetCommandReceiptObservation(context.Background(), ns, id)
	if err != nil || !observed.ResultOmitted || observed.CommandID != id || observed.OutcomeRevision != 3 || len(observed.ResponseJSON) != 0 {
		t.Fatal("compact commit was lost or presented as a full result", observed, err)
	}
	if _, err = client.GetCommandReceipt(context.Background(), ns, id); err == nil || !strings.Contains(err.Error(), "GetCommandReceiptObservation") {
		t.Fatal("legacy full-result method silently accepted an omitted response", err)
	}
}

func TestCommandReceiptObservationRejectsSwappedAndIncompleteReceipts(t *testing.T) {
	ns := d.MustParseID("0199ff42-0000-7000-8000-000000000001")
	id := d.MustParseID("0199ff42-0000-7000-8000-000000000002")
	other := d.MustParseID("0199ff42-0000-7000-8000-000000000003")
	for _, body := range []map[string]any{
		{"command_id": other, "outcome_revision": 3, "response_json": map[string]any{"value": "other command"}},
		{"command_id": id, "outcome_revision": 0, "response_json": map[string]any{"value": "no commit revision"}},
		{"command_id": id, "outcome_revision": 3},
	} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { json.NewEncoder(w).Encode(body) }))
		client, err := sdk.New(server.URL, "", server.Client())
		if err != nil {
			t.Fatal(err)
		}
		_, err = client.GetCommandReceiptObservation(context.Background(), ns, id)
		server.Close()
		if err == nil {
			t.Fatal("incomplete or foreign receipt accepted", body)
		}
	}
}
