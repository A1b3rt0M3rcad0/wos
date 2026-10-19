package woscli

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

func TestWorkspacePlanningRecoveryFreezesNoteAndProtectsProfile(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	w, config, material := journalFixture(t)
	scope := material.Contract.Scope
	credential := domain.MustParseID("0199ffaa-0000-7000-8000-000000000020")
	config.Profile = "planner"
	command := application.AppendEntityNoteCommand{Scope: scope, SubjectRef: material.WorkItem.Ref(), Type: domain.EntityNoteUpdate, Body: "Original observation, without acceptance."}
	input, err := commands.Encode(command)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(t.TempDir(), "note.json")
	if err = os.WriteFile(file, input, 0600); err != nil {
		t.Fatal(err)
	}
	calls := 0
	keys := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		calls++
		keys[request.Header.Get("Idempotency-Key")] = true
		if request.URL.Path != "/api/v1/commands/append_entity_note" {
			t.Error("planning used a privileged endpoint")
		}
		var body struct {
			Command json.RawMessage `json:"command"`
		}
		if err := json.NewDecoder(request.Body).Decode(&body); err != nil || string(body.Command) != string(input) {
			t.Errorf("frozen command changed: %s %v", body.Command, err)
		}
		if calls == 1 {
			connection, _, _ := writer.(http.Hijacker).Hijack()
			connection.Close()
			return
		}
		json.NewEncoder(writer).Encode(map[string]any{"command_id": material.Contract.ID, "outcome_revision": 2, "idempotent_replay": true, "value": map[string]any{"id": material.WorkItem.ID}})
	}))
	defer server.Close()
	config.Connection.ServerURL = server.URL
	client, _ := sdk.New(server.URL, "", server.Client())
	o := options{args: []string{"planning", "apply"}, values: map[string]string{"command": "append_entity_note", "file": file}}
	if _, err = workspacePlanningCommand(context.Background(), w, client, config, scope, credential, o); err == nil {
		t.Fatal("lost response was treated as confirmation")
	}
	if calls != 1 {
		t.Fatalf("intention failed before the simulated response loss: %v", err)
	}
	if err = w.ensureNoWorkspacePlanningPending(credential); err == nil {
		t.Fatal("profile removal could strand a pending intention")
	}
	command.Body = "Changed draft must never replace the frozen observation."
	changed, _ := commands.Encode(command)
	if err = os.WriteFile(file, changed, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = workspacePlanningCommand(context.Background(), w, client, config, scope, credential, o); err == nil || calls != 1 {
		t.Fatal("new draft replaced an uncertain intention", err)
	}
	recovery := options{args: []string{"planning", "recover"}, values: map[string]string{}}
	result, err := workspacePlanningCommand(context.Background(), w, client, config, scope, credential, recovery)
	if err != nil || !result.Committed || calls != 2 || len(keys) != 1 {
		t.Fatalf("exact recovery failed: %+v %v calls=%d keys=%d", result, err, calls, len(keys))
	}
	if err = w.ensureNoWorkspacePlanningPending(credential); err != nil {
		t.Fatal("confirmed documentary intention still blocks profile removal", err)
	}
	if _, err = workspacePlanningCommand(context.Background(), w, client, config, scope, credential, recovery); err != nil || calls != 2 {
		t.Fatal("confirmed local receipt caused another network mutation", err)
	}
}

func TestWorkspacePlanningRefusesExecutionAndForeignScopeBeforeSending(t *testing.T) {
	w, config, material := journalFixture(t)
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { calls++ }))
	defer server.Close()
	config.Connection.ServerURL = server.URL
	client, _ := sdk.New(server.URL, "", server.Client())
	for _, command := range []string{"claim_work_item", "return_signed_work", "administrative_complete_work_item"} {
		o := options{args: []string{"planning", "apply"}, values: map[string]string{"command": command}}
		if _, err := workspacePlanningCommand(context.Background(), w, client, config, material.Contract.Scope, "", o); err == nil {
			t.Fatalf("unsigned execution escape exposed: %s", command)
		}
	}
	foreign := material.Contract.Scope
	foreign.OutcomeID = domain.MustParseID("0199ffaa-0000-7000-8000-000000000099")
	command := application.AppendEntityNoteCommand{Scope: foreign, SubjectRef: domain.EntityRef{Scope: foreign, Kind: domain.EntityKindWorkItem, ID: material.WorkItem.ID}, Type: domain.EntityNoteUpdate, Body: "Do not redirect the selected scope."}
	raw, _ := commands.Encode(command)
	file := filepath.Join(t.TempDir(), "foreign.json")
	if err := os.WriteFile(file, raw, 0600); err != nil {
		t.Fatal(err)
	}
	o := options{args: []string{"planning", "apply"}, values: map[string]string{"command": "append_entity_note", "file": file}}
	if _, err := workspacePlanningCommand(context.Background(), w, client, config, material.Contract.Scope, "", o); err == nil || !strings.Contains(err.Error(), "Outcome differs") {
		t.Fatal("foreign scope was not rejected", err)
	}
	if calls != 0 {
		t.Fatalf("rejected planning intent reached transport: %d", calls)
	}
}
