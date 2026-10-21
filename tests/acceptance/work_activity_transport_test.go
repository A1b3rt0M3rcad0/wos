package acceptance_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Reuse the public enrollment/acquisition journey so all adapters exercise real
// signed authority rather than granting activity writes through a test bypass.
func workActivityTransportJourney(t *testing.T, ctx context.Context, worker *sdk.Client, session *mcp.ClientSession, acquired a.WorkContractResult, signer d.ID) a.AppendWorkActivityCommand {
	t.Helper()
	contract := acquired.Contract
	scope := contract.Scope
	command := a.AppendWorkActivityCommand{
		Scope: scope, ContractID: contract.ID, SignerKeyID: signer,
		Authority:       a.ContractAuthority{ExecutionID: contract.ExecutionID, FencingToken: contract.FencingToken, SpecDigest: contract.SignedBinding.SpecificationDigest},
		AuthorityDigest: signing.Digest(acquired.IssuedAuthority.Payload), ExpectedContractVersion: contract.Version,
		Type: d.WorkActivityProgress, Description: "Examined the contracted task and recorded a bounded observation.",
	}
	first, err := worker.AppendWorkActivity(ctx, "transport-activity-progress", command)
	if err != nil {
		t.Fatal(err)
	}
	originalCommand := command
	encoded, err := commands.Encode(command)
	if err != nil {
		t.Fatal(err)
	}
	call := func(name string, args, target any) {
		t.Helper()
		result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		if err != nil || result.IsError {
			t.Fatalf("%s: %v %+v", name, err, result)
		}
		raw, err := json.Marshal(result.StructuredContent)
		if err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(raw, target); err != nil {
			t.Fatal(err)
		}
	}
	var replay a.MutationResult[a.WorkActivityResult]
	call("wos_append_work_activity", map[string]any{"idempotency_key": "transport-activity-progress", "command": json.RawMessage(encoded)}, &replay)
	if !replay.IdempotentReplay || !reflect.DeepEqual(replay.Value.Activity, first.Value.Activity) || replay.OutcomeRevision != first.OutcomeRevision {
		t.Fatal("cross-transport activity replay changed the immutable record")
	}
	command.Type, command.Description = d.WorkActivityDiscovery, "Found the existing public contract; no implementation result is claimed."
	second, err := worker.AppendWorkActivity(ctx, "transport-activity-discovery", command)
	if err != nil {
		t.Fatal(err)
	}
	filter := ports.WorkActivityFilter{WorkItemID: contract.WorkItemID, ContractID: contract.ID}
	page, err := worker.ListWorkActivity(ctx, scope, filter, 1, "")
	if err != nil || len(page.Items) != 1 || page.Items[0].ID != first.Value.Activity.ID || page.NextCursor == "" {
		t.Fatalf("first activity page: %+v %v", page, err)
	}
	var mcpPage a.WorkActivityPage
	call("wos_list_work_activity", map[string]any{"namespace_id": scope.NamespaceID, "outcome_id": scope.OutcomeID, "work_item_id": contract.WorkItemID, "contract_id": contract.ID, "limit": 1}, &mcpPage)
	if !reflect.DeepEqual(page, mcpPage) {
		t.Fatalf("HTTP/MCP activity mismatch: %+v %+v", page, mcpPage)
	}
	// A later append cannot shift or extend an in-flight immutable traversal.
	command.Type, command.Description = d.WorkActivityValidation, "Recorded a local validation attempt; criterion assessment is still separate."
	third, err := worker.AppendWorkActivity(ctx, "transport-activity-validation", command)
	if err != nil {
		t.Fatal(err)
	}
	next, err := worker.ListWorkActivity(ctx, scope, filter, 1, page.NextCursor)
	if err != nil || len(next.Items) != 1 || next.Items[0].ID != second.Value.Activity.ID || next.NextCursor != "" || next.HistoryThroughRevision != page.HistoryThroughRevision {
		t.Fatalf("history traversal changed after append: %+v %v", next, err)
	}
	all, err := worker.ListWorkActivity(ctx, scope, filter, 0, "")
	if err != nil || len(all.Items) != 3 || all.Items[2].ID != third.Value.Activity.ID {
		t.Fatalf("fresh activity history: %+v %v", all, err)
	}
	newestFilter := filter
	newestFilter.Descending = true
	newest, err := worker.ListWorkActivity(ctx, scope, newestFilter, 1, "")
	if err != nil || newest.Order != "desc" || len(newest.Items) != 1 || newest.Items[0].ID != third.Value.Activity.ID || newest.NextCursor == "" {
		t.Fatalf("latest activity page: %+v %v", newest, err)
	}
	var newestMCP a.WorkActivityPage
	call("wos_list_work_activity", map[string]any{"namespace_id": scope.NamespaceID, "outcome_id": scope.OutcomeID, "work_item_id": contract.WorkItemID, "contract_id": contract.ID, "limit": 1, "descending": true}, &newestMCP)
	if !reflect.DeepEqual(newest, newestMCP) {
		t.Fatal("descending HTTP/MCP activity differs")
	}
	older, err := worker.ListWorkActivity(ctx, scope, newestFilter, 1, newest.NextCursor)
	if err != nil || len(older.Items) != 1 || older.Items[0].ID != second.Value.Activity.ID {
		t.Fatalf("older descending activity page: %+v %v", older, err)
	}
	if _, err = worker.ListWorkActivity(ctx, scope, filter, 1, newest.NextCursor); err == nil {
		t.Fatal("activity cursor allowed changing order")
	}
	_, err = worker.ListWorkActivity(ctx, scope, ports.WorkActivityFilter{}, 1, page.NextCursor)
	if err == nil {
		t.Fatal("filter-changing activity cursor accepted")
	}
	_, err = worker.ListWorkActivity(ctx, scope, filter, 101, "")
	var remote *sdk.Error
	if !errors.As(err, &remote) || remote.Status != http.StatusUnprocessableEntity {
		t.Fatalf("invalid limit not rejected: %v", err)
	}
	attention, err := worker.GetWorkAttention(ctx, scope, contract.WorkItemID)
	if err != nil {
		t.Fatal(err)
	}
	var mcpAttention a.WorkAttention
	call("wos_get_work_attention", map[string]any{"namespace_id": scope.NamespaceID, "outcome_id": scope.OutcomeID, "work_item_id": contract.WorkItemID}, &mcpAttention)
	// Time of observation may differ; the persisted work, signals and revision must agree.
	mcpAttention.EvaluatedAt = attention.EvaluatedAt
	mcpAttention.OperationalState.EvaluatedAt = attention.OperationalState.EvaluatedAt
	if attention.Verification != nil && mcpAttention.Verification != nil {
		// Verification is evaluated inside each independent read frame. Normalize
		// only its observation timestamp; source, status, authority, and revision
		// remain covered by the exact parity assertion below.
		mcpAttention.Verification.EvaluatedAt = attention.Verification.EvaluatedAt
	}
	if !reflect.DeepEqual(attention, mcpAttention) {
		t.Fatalf("HTTP/MCP attention mismatch: %+v %+v", attention, mcpAttention)
	}
	current, err := worker.ReadSignedState(ctx, a.SignedStateQuery{Scope: scope, Resource: "execution", ID: contract.ID})
	if err != nil || current.Contract == nil || current.Contract.Version != signing.Decimal(contract.Version) || current.Contract.LeaseVersion != signing.Decimal(contract.LeaseVersion) || current.Contract.Status != d.ContractActive {
		t.Fatalf("activity changed execution authority: %+v %v", current, err)
	}
	workExecutionDiscoveryParity(t, ctx, worker, session, originalCommand, d.ContractActive)
	return originalCommand
}

func workActivityClosedAuthorityJourney(t *testing.T, ctx context.Context, worker *sdk.Client, session *mcp.ClientSession, command a.AppendWorkActivityCommand) {
	t.Helper()
	replay, err := worker.AppendWorkActivity(ctx, "transport-activity-progress", command)
	if err != nil || !replay.IdempotentReplay {
		t.Fatalf("closed contract lost durable activity replay: %+v %v", replay, err)
	}
	_, err = worker.AppendWorkActivity(ctx, "transport-activity-after-close", command)
	var remote *sdk.Error
	if !errors.As(err, &remote) || remote.Status != http.StatusConflict {
		t.Fatalf("closed contract accepted new activity: %v", err)
	}
	workExecutionDiscoveryParity(t, ctx, worker, session, command, d.ContractDelivered)
}

func workExecutionDiscoveryParity(t *testing.T, ctx context.Context, worker *sdk.Client, session *mcp.ClientSession, command a.AppendWorkActivityCommand, status d.ContractStatus) {
	t.Helper()
	contract, err := worker.ReadSignedState(ctx, a.SignedStateQuery{Scope: command.Scope, Resource: "execution", ID: command.ContractID})
	if err != nil {
		t.Fatal(err)
	}
	if contract.Contract == nil {
		t.Fatal("signed execution metadata missing")
	}
	query := a.SignedStateQuery{Scope: command.Scope, Resource: "executions", HolderPrincipalID: contract.Contract.HolderPrincipalID, WorkItemID: contract.Contract.WorkItemID, Status: status, Limit: 1}
	listed, err := worker.ReadSignedState(ctx, query)
	if err != nil || len(listed.Executions) != 1 || listed.Executions[0].ID != command.ContractID || listed.Executions[0].Status != status || listed.Executions[0].EffectiveStatus != status {
		t.Fatalf("signed execution discovery: %+v %v", listed, err)
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: "wos_read_signed_state", Arguments: query})
	if err != nil || result.IsError {
		t.Fatalf("MCP execution discovery: %+v %v", result, err)
	}
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var mcpListed a.SignedStateResult
	if err = json.Unmarshal(raw, &mcpListed); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(listed.Executions, mcpListed.Executions) || listed.SearchComplete != mcpListed.SearchComplete || listed.OutcomeRevision != mcpListed.OutcomeRevision {
		t.Fatal("HTTP/MCP signed discovery differs")
	}
}
