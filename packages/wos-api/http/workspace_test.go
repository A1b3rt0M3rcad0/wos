package httptransport

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	mcptransport "github.com/A1b3rt0M3rcad0/wos/packages/wos-api/mcp"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestWorkspaceHTTPMCPAndSDKShareContextNotesAndTypedErrors(t *testing.T) {
	handler := newTestHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()
	client, err := sdk.New(server.URL, "", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	identity := application.Identity{PrincipalID: handler.auth.Principal(), Actor: handler.auth.Actor()}
	mcpServer, err := mcptransport.New(handler.service, handler.ids, mcptransport.Options{LocalIdentity: &identity})
	if err != nil {
		t.Fatal(err)
	}
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := mcpServer.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "workspace-parity", Version: "1"}, nil)
	session, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	namespace := domain.MustParseID("0199ec00-0000-7000-8000-000000000001")
	created, err := client.CreateOutcome(ctx, "workspace-parity-outcome-0001", application.CreateOutcomeCommand{NamespaceID: namespace, Title: "Readable shared work", DesiredState: "A reviewer can inspect current proof", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	scope := created.Value.Scope()
	objective, err := client.CreateObjective(ctx, "workspace-parity-objective-000001", application.CreateObjectiveCommand{Scope: scope, Title: "Task owner", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	work, err := client.CreateWorkItem(ctx, "workspace-parity-work-000001", application.CreateWorkItemCommand{Scope: scope, ObjectiveID: &objective.Value.ID, Title: "Document one observation", Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleBacklog})
	if err != nil {
		t.Fatal(err)
	}
	noteCommand := application.AppendEntityNoteCommand{Scope: scope, SubjectRef: work.Value.Ref(), Type: domain.EntityNoteUpdate, Body: "I finished writing a draft; acceptance remains pending."}
	note, err := client.AppendEntityNote(ctx, "workspace-parity-note-000001", noteCommand)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := commands.Encode(noteCommand)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "wos_append_entity_note", Arguments: map[string]any{"idempotency_key": "workspace-parity-note-000001", "command": json.RawMessage(payload)}})
	if err != nil || replay.IsError {
		t.Fatalf("MCP note replay: %v %+v", err, replay)
	}
	var replayed application.MutationResult[domain.EntityNote]
	raw, _ := json.Marshal(replay.StructuredContent)
	if err = json.Unmarshal(raw, &replayed); err != nil || !replayed.IdempotentReplay || replayed.Value.ID != note.Value.ID || replayed.Value.PrincipalID != identity.PrincipalID {
		t.Fatalf("replay/authorship changed: %s %v", raw, err)
	}
	fullNote, err := client.GetEntityNote(ctx, scope, note.Value.ID)
	if err != nil || fullNote.Body != noteCommand.Body {
		t.Fatalf("full note: %+v %v", fullNote, err)
	}
	viewQuery := application.WorkspaceViewQuery{Subject: application.WorkspaceSubject{Kind: "entity", Ref: work.Value.Ref()}}
	view, err := client.GetEntityWorkspaceView(ctx, scope, viewQuery)
	if err != nil || view.Header.Lifecycle != "backlog" || view.EntityVersion != work.Value.Version {
		t.Fatalf("coherent SDK context: %+v %v", view, err)
	}
	mcpView, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "wos_get_workspace_view", Arguments: map[string]any{"scope": scope, "query": viewQuery}})
	if err != nil || mcpView.IsError {
		t.Fatalf("MCP workspace context: %v %+v", err, mcpView)
	}
	raw, _ = json.Marshal(mcpView.StructuredContent)
	var exact map[string]json.RawMessage
	if err = json.Unmarshal(raw, &exact); err != nil || len(exact["outcome_revision"]) < 3 || exact["outcome_revision"][0] != '"' {
		t.Fatalf("projection revision is not exact quoted decimal: %s", raw)
	}
	page, err := client.GetWorkspaceCards(ctx, scope, application.WorkspaceCardsQuery{Refs: []domain.EntityRef{work.Value.Ref()}})
	if err != nil || len(page.Items) != 1 || page.Items[0].Lifecycle != "backlog" {
		t.Fatalf("documentary note mutated work: %+v %v", page, err)
	}
	timeline, err := client.GetWorkspaceTimeline(ctx, scope, application.WorkspaceTimelineQuery{SubjectRef: ptrWorkspaceRef(work.Value.Ref()), Limit: 20})
	if err != nil {
		t.Fatal(err)
	}
	found := 0
	for _, item := range timeline.Items {
		if item.Origin == "entity_note" && item.Source.ID == note.Value.ID {
			found++
		}
	}
	if found != 1 {
		t.Fatalf("note/event duplicated or missing in timeline: %+v", timeline)
	}
	_, err = client.ActivateOutcome(ctx, "workspace-parity-activate-001", application.ActivateOutcomeCommand{Scope: scope, ExpectedVersion: created.Value.Version})
	var remote *sdk.Error
	if !errors.As(err, &remote) || remote.Code != "precondition_failed" || remote.Details == nil || remote.Details.ReasonCode != "missing_required_criterion" {
		t.Fatalf("SDK lost typed domain condition: %+v", err)
	}
	failed, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "wos_activate_outcome", Arguments: map[string]any{"idempotency_key": "workspace-parity-activate-002", "command": map[string]any{"scope": scope, "expected_version": created.Value.Version}}})
	if err != nil || !failed.IsError {
		t.Fatalf("MCP accepted missing condition: %v %+v", err, failed)
	}
	raw, _ = json.Marshal(failed.StructuredContent)
	var failure struct {
		Error struct {
			Code    string               `json:"code"`
			Details *domain.ErrorDetails `json:"details"`
		} `json:"error"`
	}
	if err = json.Unmarshal(raw, &failure); err != nil || failure.Error.Code != remote.Code || failure.Error.Details == nil || failure.Error.Details.ReasonCode != remote.Details.ReasonCode {
		t.Fatalf("HTTP/MCP condition differs: %s %v", raw, err)
	}
}

func ptrWorkspaceRef(ref domain.EntityRef) *domain.EntityRef { return &ref }

func TestWorkspaceOversizedBatchRetainsEveryCreatedIDOnReplayAndReceiptLookup(t *testing.T) {
	handler := newTestHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()
	client, err := sdk.New(server.URL, "", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	namespace := domain.MustParseID("0199ec00-0000-7000-8000-000000000001")
	created, err := client.CreateOutcome(ctx, "workspace-large-plan-root-001", application.CreateOutcomeCommand{NamespaceID: namespace, Title: "Large but bounded batch", DesiredState: "Every returned identity survives compact receipts", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	scope := created.Value.Scope()
	command := application.CreatePlanItemsCommand{Scope: scope}
	for i := 0; i < 50; i++ {
		command.Items = append(command.Items, application.PlanItemInput{LocalKey: fmt.Sprintf("objective-%02d", i), Kind: domain.EntityKindObjective, Title: fmt.Sprintf("Objective %02d", i), InitialCriteria: []application.InitialCriterion{
			{Title: "First verifiable condition", Description: strings.Repeat("a", 2200), Required: true, VerificationMode: domain.VerificationModeEvidenceReview},
			{Title: "Second verifiable condition", Description: strings.Repeat("b", 2200), Required: true, VerificationMode: domain.VerificationModeEvidenceReview},
		}})
	}
	payload, err := commands.Encode(command)
	if err != nil || len(payload) > commands.MaxPayloadBytes {
		t.Fatalf("fixture is not a valid bounded command: bytes=%d %v", len(payload), err)
	}
	result, err := client.CreatePlanItems(ctx, "workspace-large-plan-batch-001", command)
	if err != nil || !result.ResultOmitted || len(result.CreatedRefs) != 50 || len(result.CreatedItems) != 50 {
		t.Fatalf("compact result lost batch identities: omitted=%v refs=%d items=%d %v", result.ResultOmitted, len(result.CreatedRefs), len(result.CreatedItems), err)
	}
	criteria := 0
	for _, item := range result.CreatedItems {
		criteria += len(item.CriterionIDs)
		if item.Version == 0 || result.CreatedRefs[item.LocalKey] != item.Ref {
			t.Fatal("compact creation metadata does not resolve", item)
		}
	}
	if criteria != 100 {
		t.Fatalf("compact receipt lost criterion IDs: %d", criteria)
	}
	replay, err := client.CreatePlanItems(ctx, "workspace-large-plan-batch-001", command)
	if err != nil || !replay.IdempotentReplay || replay.CommandID != result.CommandID || !reflect.DeepEqual(result.CreatedRefs, replay.CreatedRefs) || !reflect.DeepEqual(result.CreatedItems, replay.CreatedItems) {
		t.Fatalf("replay changed created identities: %+v %v", replay, err)
	}
	raw, err := client.QueryJSON(ctx, "/namespaces/"+scope.NamespaceID.String()+"/commands/"+result.CommandID.String())
	if err != nil {
		t.Fatal(err)
	}
	var retained sdk.CommandResult[application.CreatePlanItemsResult]
	if err = json.Unmarshal(raw, &retained); err != nil || !retained.ResultOmitted || !reflect.DeepEqual(result.CreatedRefs, retained.CreatedRefs) || !reflect.DeepEqual(result.CreatedItems, retained.CreatedItems) {
		t.Fatalf("bounded receipt lookup lost created identities: %s %v", raw, err)
	}
	identity := application.Identity{PrincipalID: handler.auth.Principal(), Actor: handler.auth.Actor()}
	mcpServer, err := mcptransport.New(handler.service, handler.ids, mcptransport.Options{LocalIdentity: &identity})
	if err != nil {
		t.Fatal(err)
	}
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := mcpServer.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer serverSession.Close()
	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "workspace-compact-parity", Version: "1"}, nil)
	session, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()
	mcpReplay, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "wos_create_plan_items", Arguments: map[string]any{"idempotency_key": "workspace-large-plan-batch-001", "command": json.RawMessage(payload)}})
	if err != nil || mcpReplay.IsError {
		t.Fatalf("MCP compact replay failed: %v %+v", err, mcpReplay)
	}
	raw, _ = json.Marshal(mcpReplay.StructuredContent)
	var mcpReceipt sdk.CommandResult[application.CreatePlanItemsResult]
	if err = json.Unmarshal(raw, &mcpReceipt); err != nil || !mcpReceipt.ResultOmitted || !mcpReceipt.IdempotentReplay || !reflect.DeepEqual(result.CreatedRefs, mcpReceipt.CreatedRefs) || !reflect.DeepEqual(result.CreatedItems, mcpReceipt.CreatedItems) {
		t.Fatalf("MCP compact receipt lost original creation identities: %s %v", raw, err)
	}
}
