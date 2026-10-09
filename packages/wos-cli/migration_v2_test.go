package woscli

import (
	"bytes"
	"context"
	"encoding/json"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestMigrationDryRunPreservesDraftsUnknownIntentAndMakesNoRequest(t *testing.T) {
	w, config, contract := journalFixture(t)
	calls := 0
	remote := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer remote.Close()
	config.Connection.ServerURL = remote.URL
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	must(w.WriteDocument(".wos/config.yaml", config))
	must(w.saveJSON(".wos/trust.json", config.Destination()))
	_, e := w.materialize(config, contract, contract.Contract.ID)
	must(e)
	draft := filepath.Join(stateDir(contract.Contract.WorkItemID, contract.Contract.ID), "result.yaml")
	original, e := w.ReadV2(draft)
	must(e)
	path, intent, e := w.Prepare(config, contract.Contract.Scope, ".wos/checkout/unknown", "acquire_work_contract", a.AcquireWorkContractCommand{Scope: contract.Contract.Scope, WorkItemID: contract.Contract.WorkItemID, ExpectedWorkItemVersion: contract.WorkItem.Version, TTLSeconds: 300})
	must(e)
	intent.State = "sent_unknown"
	must(w.saveJSON(path, intent))
	before, _, e := migrationInventoryV2(w)
	must(e)
	var output, diagnostic bytes.Buffer
	code := Run(context.Background(), []string{"workspace", "migrate", "--to", "2", "--profile", "executor", "--dry-run", "--workspace", w.canonicalPath, "--output", "json"}, &output, &diagnostic)
	if code != 0 {
		t.Fatalf("%d: %s %s", code, output.String(), diagnostic.String())
	}
	var result Output
	must(json.Unmarshal(output.Bytes(), &result))
	after, _, e := migrationInventoryV2(w)
	must(e)
	if calls != 0 || before.InventoryDigest != after.InventoryDigest || len(after.UnresolvedIntentions) != 1 || result.RequiresAction == "" {
		t.Fatal("dry run mutated legacy state or reconciled unknown remote result")
	}
	actual, e := w.ReadV2(draft)
	must(e)
	if !bytes.Equal(original, actual) {
		t.Fatal("draft changed during inventory")
	}
	intentRaw, e := w.ReadV2(path)
	must(e)
	var unchanged Intent
	must(json.Unmarshal(intentRaw, &unchanged))
	if unchanged.State != "sent_unknown" {
		t.Fatal("dry run altered original intention")
	}
	must(os.WriteFile(filepath.Join(w.canonicalPath, draft), append(original, []byte("\n# operator edit\n")...), 0600))
	changed, _, e := migrationInventoryV2(w)
	must(e)
	if changed.InventoryDigest == after.InventoryDigest {
		t.Fatal("inventory ignored exact draft edit")
	}
}

func TestMigrationDryRunRejectsChangedTrustAndForgedConfirmedReceipt(t *testing.T) {
	w, config, contract := journalFixture(t)
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	must(w.WriteDocument(".wos/config.yaml", config))
	must(w.saveJSON(".wos/trust.json", config.Destination()))
	path, intent, e := w.Prepare(config, contract.Contract.Scope, ".wos/checkout/confirmed", "acquire_work_contract", a.AcquireWorkContractCommand{Scope: contract.Contract.Scope, WorkItemID: contract.Contract.WorkItemID, ExpectedWorkItemVersion: contract.WorkItem.Version})
	must(e)
	intent.State = "confirmed"
	must(w.saveJSON(path, intent))
	receiptPath := filepath.Join(filepath.Dir(filepath.Dir(path)), "receipts", intent.IdempotencyKey+".json")
	receipt := Receipt{SchemaVersion: 1, Destination: intent.Destination, IdempotencyKey: intent.IdempotencyKey, PayloadDigest: "another-payload", Response: json.RawMessage(`{"accepted":true}`)}
	must(w.saveJSON(receiptPath, receipt))
	if _, _, e = migrationInventoryV2(w); e == nil {
		t.Fatal("forged confirmed receipt trusted for migration")
	}
	receipt.PayloadDigest = intent.PayloadDigest
	must(w.saveJSON(receiptPath, receipt))
	if _, _, e = migrationInventoryV2(w); e != nil {
		t.Fatal(e)
	}
	changed := config.Destination()
	changed.ServerURL = "https://replacement.example"
	must(w.saveJSON(".wos/trust.json", changed))
	if _, _, e = migrationInventoryV2(w); e == nil {
		t.Fatal("changed destination silently accepted for migration")
	}
}

func TestMigrationReportsConfirmedAcquisitionWithoutMaterialization(t *testing.T) {
	w, config, contract := journalFixture(t)
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	must(w.WriteDocument(".wos/config.yaml", config))
	must(w.saveJSON(".wos/trust.json", config.Destination()))
	path, intent, e := w.Prepare(config, contract.Contract.Scope, ".wos/checkout/confirmed", "acquire_work_contract", a.AcquireWorkContractCommand{Scope: contract.Contract.Scope, WorkItemID: contract.Contract.WorkItemID, ExpectedWorkItemVersion: contract.WorkItem.Version})
	must(e)
	intent.State = "confirmed"
	must(w.saveJSON(path, intent))
	response, e := json.Marshal(sdk.CommandResult[a.WorkContractResult]{Value: contract, CommandID: d.MustParseID("0199ffaa-0000-7000-8000-000000000010"), OutcomeRevision: 2})
	must(e)
	receipt := Receipt{SchemaVersion: 1, Destination: intent.Destination, IdempotencyKey: intent.IdempotencyKey, PayloadDigest: intent.PayloadDigest, Response: response}
	must(w.saveJSON(filepath.Join(filepath.Dir(filepath.Dir(path)), "receipts", intent.IdempotencyKey+".json"), receipt))
	inventory, _, e := migrationInventoryV2(w)
	must(e)
	if len(inventory.UnmaterializedAcquisitions) != 1 || len(inventory.UnresolvedIntentions) != 0 {
		t.Fatal("accepted unmaterialized acquisition was hidden")
	}
	_, e = convertWorkspaceV2(context.Background(), w, config, inventory, options{})
	if e == nil {
		t.Fatal("migration discarded original accepted acquisition")
	}
	_, e = w.materialize(config, contract, contract.Contract.ID)
	must(e)
	inventory, _, e = migrationInventoryV2(w)
	must(e)
	if len(inventory.UnmaterializedAcquisitions) != 0 {
		t.Fatal("original accepted materialization not recognized")
	}
}
