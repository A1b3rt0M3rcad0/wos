package wossdk_test

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

func TestCommandReceiptObservationPreservesCompactCommitWithoutClaimingFullResult(t *testing.T) {
	ns := d.MustParseID("0199ff42-0000-7000-8000-000000000001")
	id := d.MustParseID("0199ff42-0000-7000-8000-000000000002")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/namespaces/"+ns.String()+"/commands/"+id.String() {
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

func TestCommandReceiptObservationPreservesOperationalRecoveryTargets(t *testing.T) {
	ns := d.MustParseID("0199ff42-0000-7000-8000-000000000001")
	id := d.MustParseID("0199ff42-0000-7000-8000-000000000002")
	delivery := d.MustParseID("0199ff42-0000-7000-8000-000000000003")
	withdrawal := d.MustParseID("0199ff42-0000-7000-8000-000000000004")
	authority := d.MustParseID("0199ff42-0000-7000-8000-000000000005")
	digest := "sha256:" + strings.Repeat("a", 64)
	for _, kind := range []string{"native_delivery", "completion_authority"} {
		t.Run(kind, func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				if r.Method != http.MethodGet || r.URL.Path != "/api/v1/namespaces/"+ns.String()+"/commands/"+id.String() {
					t.Error("receipt recovery changed the operation or destination")
				}
				body := map[string]any{"command_id": id, "outcome_revision": json.Number("9007199254740993"), "result_omitted": true}
				if kind == "native_delivery" {
					body["delivery_id"], body["delivery_digest"], body["delivery_round"], body["withdrawal_id"] = delivery, digest, "18446744073709551615", withdrawal
				} else {
					body["completion_authority_id"] = authority
				}
				_ = json.NewEncoder(w).Encode(body)
			}))
			defer server.Close()
			client, err := sdk.New(server.URL, "", server.Client())
			if err != nil {
				t.Fatal(err)
			}
			observed, err := client.GetCommandReceiptObservation(context.Background(), ns, id)
			if err != nil || !observed.ResultOmitted || observed.CommandID != id || observed.OutcomeRevision != 9007199254740993 || len(observed.ResponseJSON) != 0 || requests.Load() != 1 {
				t.Fatalf("lookup lost the durable compact result or retried a mutation: %+v %v", observed, err)
			}
			if kind == "native_delivery" {
				if observed.DeliveryID == nil || *observed.DeliveryID != delivery || observed.DeliveryDigest != digest || observed.DeliveryRound != math.MaxUint64 || observed.WithdrawalID == nil || *observed.WithdrawalID != withdrawal {
					t.Fatalf("lookup lost the native round or withdrawal target: %+v", observed)
				}
			} else if observed.CompletionAuthorityID == nil || *observed.CompletionAuthorityID != authority {
				t.Fatalf("lookup lost the completion delegation target: %+v", observed)
			}
		})
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
