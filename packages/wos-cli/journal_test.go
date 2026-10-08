package woscli

import (
	"context"
	"encoding/json"
	"errors"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync"
	"testing"
	"time"
)

func journalFixture(t *testing.T) (*Workspace, Config, a.WorkContractResult) {
	t.Helper()
	w, err := OpenWorkspace(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { w.Close() })
	scope := d.Scope{NamespaceID: d.MustParseID("0199ffaa-0000-7000-8000-000000000001"), OutcomeID: d.MustParseID("0199ffaa-0000-7000-8000-000000000002")}
	now := time.Now().UTC()
	work, err := d.NewWorkItem(d.MustParseID("0199ffaa-0000-7000-8000-000000000003"), scope, "Recover only original", "", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
	if err != nil {
		t.Fatal(err)
	}
	work.ContractsEnabled = true
	c, err := d.NewWorkContract(d.MustParseID("0199ffaa-0000-7000-8000-000000000004"), d.MustParseID("0199ffaa-0000-7000-8000-000000000005"), work, "holder", d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "worker"}, d.DefaultContractLeasePolicy(), 300, d.WorkContractSpec{Title: work.Title}, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	if err = work.BindContract(c, now); err != nil {
		t.Fatal(err)
	}
	config := Config{SchemaVersion: 1, Kind: "WOSWorkspace", Connection: ConnectionConfig{ServerURL: "http://placeholder", CredentialRef: "env:WOS_TOKEN"}, Scope: WorkspaceScope{scope.NamespaceID, scope.OutcomeID}, Workspace: WorkspaceConfig{".wos/work", "yaml"}, Lease: LeaseConfig{300}, Output: OutputConfig{"json"}}
	return w, config, a.WorkContractResult{Contract: c, WorkItem: work, EvaluatedAt: now}
}
func TestJournalReplaysExactIntentAfterCommittedResponseLoss(t *testing.T) {
	w, config, result := journalFixture(t)
	var mu sync.Mutex
	calls := 0
	seen := map[string]bool{}
	drop := true
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		calls++
		key := r.Header.Get("Idempotency-Key")
		seen[key] = true
		entries, err := w.intentPaths(".wos/checkout/original")
		if err != nil || len(entries) != 1 {
			t.Error("request arrived before durable intent")
		}
		raw, _ := w.Read(entries[0])
		var intent Intent
		json.Unmarshal(raw, &intent)
		if intent.State != "sent_unknown" {
			t.Error("send not journaled")
		}
		if drop {
			drop = false
			connection, _, _ := rw.(http.Hijacker).Hijack()
			connection.Close()
			return
		}
		json.NewEncoder(rw).Encode(sdk.CommandResult[a.WorkContractResult]{Value: result, CommandID: d.MustParseID("0199ffaa-0000-7000-8000-000000000010"), OutcomeRevision: 2})
	}))
	defer server.Close()
	config.Connection.ServerURL = server.URL
	client, _ := sdk.New(server.URL, "", server.Client())
	cmd := a.AcquireWorkContractCommand{Scope: result.Contract.Scope, WorkItemID: result.WorkItem.ID, ExpectedWorkItemVersion: 1}
	path, intent, err := w.Prepare(config, cmd.Scope, ".wos/checkout/original", "acquire_work_contract", cmd)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = w.Replay(context.Background(), client, config, path); err == nil {
		t.Fatal("response loss unnoticed")
	}
	// A local draft/config change cannot alter the original request or redirect credentials.
	changed := config
	changed.Connection.ServerURL = "https://other.invalid"
	if _, _, err = w.Replay(context.Background(), client, changed, path); err == nil {
		t.Fatal("destination redirect accepted")
	}
	raw, receipt, err := w.Replay(context.Background(), client, config, path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = w.Read(receipt); err != nil {
		t.Fatal(err)
	}
	saved, _ := w.Read(path)
	var confirmed Intent
	json.Unmarshal(saved, &confirmed)
	if confirmed.State != "confirmed" || confirmed.PayloadDigest != intent.PayloadDigest || string(confirmed.Payload) != string(intent.Payload) {
		t.Fatal("intent changed during retry")
	}
	if _, _, err = w.Replay(context.Background(), client, config, path); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(seen) != 1 || calls != 2 {
		t.Fatalf("duplicate intent/replayed network: %d %d", len(seen), calls)
	}
	var response sdk.CommandResult[a.WorkContractResult]
	json.Unmarshal(raw, &response)
	if _, err = w.materialize(config, response.Value, response.CommandID); err != nil {
		t.Fatal(err)
	}
}
func TestJournalDiskFailureAfterCommitDoesNotCreateNewAcquisition(t *testing.T) {
	w, config, result := journalFixture(t)
	calls := 0
	keys := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		calls++
		keys[r.Header.Get("Idempotency-Key")] = true
		json.NewEncoder(rw).Encode(sdk.CommandResult[a.WorkContractResult]{Value: result, CommandID: d.MustParseID("0199ffaa-0000-7000-8000-000000000011"), OutcomeRevision: 2})
	}))
	defer server.Close()
	config.Connection.ServerURL = server.URL
	client, _ := sdk.New(server.URL, "", server.Client())
	dir := ".wos/checkout/disk-failure"
	path, _, err := w.Prepare(config, result.Contract.Scope, dir, "acquire_work_contract", a.AcquireWorkContractCommand{Scope: result.Contract.Scope, WorkItemID: result.WorkItem.ID, ExpectedWorkItemVersion: 1})
	if err != nil {
		t.Fatal(err)
	}
	// A regular file occupying receipts/ deterministically simulates disk materialization failure on all platforms.
	if err = w.AtomicWrite(filepath.Join(dir, "receipts"), []byte("blocked")); err != nil {
		t.Fatal(err)
	}
	_, _, err = w.Replay(context.Background(), client, config, path)
	var local *LocalError
	if !errors.As(err, &local) || !local.Committed {
		t.Fatalf("post-commit failure: %v", err)
	}
	if err = w.ensureNoPending(dir); exitCode(err) != 6 {
		t.Fatal("new intent was not blocked")
	}
	if err = w.root.Remove(filepath.Join(dir, "receipts")); err != nil {
		t.Fatal(err)
	}
	if _, _, err = w.Replay(context.Background(), client, config, path); err != nil {
		t.Fatal(err)
	}
	if len(keys) != 1 || calls != 2 {
		t.Fatal("recovery created another acquisition")
	}
	if _, err = w.materialize(config, result, d.MustParseID("0199ffaa-0000-7000-8000-000000000011")); err != nil {
		t.Fatal(err)
	}
	workspace := stateDir(result.WorkItem.ID, result.Contract.ID)
	custom := []byte("# local content\nsummary: retained\n")
	if err = w.AtomicWrite(filepath.Join(workspace, "result.yaml"), custom); err != nil {
		t.Fatal(err)
	}
	if _, err = w.materialize(config, result, d.MustParseID("0199ffaa-0000-7000-8000-000000000011")); err != nil {
		t.Fatal(err)
	}
	saved, _ := w.Read(filepath.Join(workspace, "result.yaml"))
	if string(saved) != string(custom) {
		t.Fatal("materialization overwrote local edits")
	}
}

func TestSDKRefusesRedirectedMutationAndTrustRebinding(t *testing.T) {
	w, config, result := journalFixture(t)
	leaked := false
	destination := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) { leaked = true; rw.WriteHeader(200) }))
	defer destination.Close()
	redirect := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		http.Redirect(rw, r, destination.URL, http.StatusTemporaryRedirect)
	}))
	defer redirect.Close()
	config.Connection.ServerURL = redirect.URL
	client, _ := sdk.New(redirect.URL, "credential-must-stay-bound", redirect.Client())
	path, _, err := w.Prepare(config, result.Contract.Scope, ".wos/checkout/redirect", "acquire_work_contract", a.AcquireWorkContractCommand{Scope: result.Contract.Scope, WorkItemID: result.WorkItem.ID, ExpectedWorkItemVersion: 1})
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = w.Replay(context.Background(), client, config, path); exitCode(err) != 6 {
		t.Fatalf("redirect category %v", err)
	}
	if leaked {
		t.Fatal("redirected mutation reached another destination")
	}
}
