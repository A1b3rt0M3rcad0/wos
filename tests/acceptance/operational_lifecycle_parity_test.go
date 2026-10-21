package acceptance_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/authentication/local"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	ht "github.com/A1b3rt0M3rcad0/wos/packages/wos-api/http"
	mt "github.com/A1b3rt0M3rcad0/wos/packages/wos-api/mcp"
	cli "github.com/A1b3rt0M3rcad0/wos/packages/wos-cli"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type operationalParityClock struct{ now time.Time }

func (c operationalParityClock) Now() time.Time { return c.now }

type operationalParityFixture struct {
	store              *memory.Store
	service            *a.Service
	scope              d.Scope
	plannedObjectiveID d.ID
	claimableTask      d.WorkItem
	backlogTask        d.WorkItem
}

type operationalParityFailure struct {
	Code        string   `json:"code"`
	ReasonCode  string   `json:"reason_code"`
	ReasonCodes []string `json:"reason_codes"`
}

type operationalParitySnapshot struct {
	state           a.OutcomeState
	events          []d.DomainEvent
	receipts        int
	work            map[d.ID]a.WorkItemOperationalReadResult
	contracts       []d.WorkContract
	signedContracts []d.WorkContract
}

func newOperationalParityFixture(t *testing.T) operationalParityFixture {
	t.Helper()
	ctx := context.Background()
	store := memory.New()
	clock := operationalParityClock{now: time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)}
	generator := &ids{}
	service, err := a.NewService(store, clock, generator)
	if err != nil {
		t.Fatal(err)
	}
	scope := d.Scope{
		NamespaceID: d.MustParseID("0199ff21-0000-7000-8000-000000000001"),
		OutcomeID:   d.MustParseID("0199ff21-0000-7000-8000-000000000002"),
	}
	now := clock.Now()
	outcome, err := d.NewOutcome(scope.OutcomeID, scope.NamespaceID, "Operational lifecycle parity", "", "All transports must enforce the same gates", d.PriorityNormal, now)
	if err != nil {
		t.Fatal(err)
	}
	outcome.Lifecycle = d.OutcomeLifecycleActive
	outcome.OperationalProfile = d.OperationalProfileOperationalV2
	outcome.CompletionPolicy = d.CompletionPolicyStrictWork

	root, err := d.NewObjective(d.MustParseID("0199ff21-0000-7000-8000-000000000010"), scope, "Active root", "", d.PriorityNormal, false, now)
	if err != nil {
		t.Fatal(err)
	}
	root.Lifecycle = d.ObjectiveLifecycleInProgress
	addParityCriterion(t, &root, "Root criterion", "0199ff21-0000-7000-8000-000000000110")
	planned, err := d.NewObjective(d.MustParseID("0199ff21-0000-7000-8000-000000000011"), scope, "Planned objective", "", d.PriorityNormal, false, now)
	if err != nil {
		t.Fatal(err)
	}
	planned.ParentObjectiveID = &root.ID
	planned.Lifecycle = d.ObjectiveLifecyclePlanned
	addParityCriterion(t, &planned, "Objective criterion", "0199ff21-0000-7000-8000-000000000111")

	claimable, err := d.NewWorkItemForObjective(d.MustParseID("0199ff21-0000-7000-8000-000000000020"), scope, &planned.ID, "Claim gate task", "Must remain unclaimed until its Objective starts", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
	if err != nil {
		t.Fatal(err)
	}
	claimable.ContractsEnabled = true
	addParityCriterion(t, &claimable, "Claim criterion", "0199ff21-0000-7000-8000-000000000120")
	backlog, err := d.NewWorkItemForObjective(d.MustParseID("0199ff21-0000-7000-8000-000000000021"), scope, &planned.ID, "Activation gate task", "Must remain in Backlog until its Objective starts", d.PriorityNormal, d.WorkItemLifecycleBacklog, now)
	if err != nil {
		t.Fatal(err)
	}
	addParityCriterion(t, &backlog, "Activation criterion", "0199ff21-0000-7000-8000-000000000121")

	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Coordination().LockOutcome(ctx, scope); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	for _, insert := range []func() error{
		func() error { return tx.Outcomes().Insert(ctx, outcome) },
		func() error { return tx.Objectives().Insert(ctx, root) },
		func() error { return tx.Objectives().Insert(ctx, planned) },
		func() error { return tx.WorkItems().Insert(ctx, claimable) },
		func() error { return tx.WorkItems().Insert(ctx, backlog) },
	} {
		if err := insert(); err != nil {
			_ = tx.Rollback()
			t.Fatal(err)
		}
	}
	if _, err := tx.Coordination().AdvanceOutcome(ctx, scope); err != nil {
		_ = tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return operationalParityFixture{store: store, service: service, scope: scope, plannedObjectiveID: planned.ID, claimableTask: claimable, backlogTask: backlog}
}

func addParityCriterion(t *testing.T, owner interface{ Ref() d.EntityRef }, title, id string) {
	t.Helper()
	criterion, err := d.NewSuccessCriterion(d.MustParseID(id), owner.Ref(), title, "", true, d.VerificationModeAttestation)
	if err != nil {
		t.Fatal(err)
	}
	switch value := owner.(type) {
	case *d.Objective:
		if err := value.Criteria.Add(criterion); err != nil {
			t.Fatal(err)
		}
	case *d.WorkItem:
		if err := value.Criteria.Add(criterion); err != nil {
			t.Fatal(err)
		}
	default:
		t.Fatalf("unsupported criterion owner %T", owner)
	}
}

func (f operationalParityFixture) snapshot(t *testing.T) operationalParitySnapshot {
	t.Helper()
	ctx := context.Background()
	state, err := f.service.GetOutcomeState(ctx, f.scope)
	if err != nil {
		t.Fatal(err)
	}
	work := make(map[d.ID]a.WorkItemOperationalReadResult, len(state.WorkItems))
	for _, item := range state.WorkItems {
		operational, err := f.service.GetWorkItemOperationalState(ctx, f.scope, item.ID)
		if err != nil {
			t.Fatal(err)
		}
		work[item.ID] = operational
	}
	events := f.store.SnapshotDomainEvents(f.scope)
	receipts := f.store.IdempotencyRecordCount()
	tx, err := f.store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	filter := ports.ContractFilter{Limit: 101}
	contractUOW, ok := tx.(ports.WorkContractUnitOfWork)
	if !ok {
		t.Fatal("memory unit of work does not expose unsigned work contracts")
	}
	signedContractUOW, ok := tx.(ports.SignedContractUnitOfWork)
	if !ok {
		t.Fatal("memory unit of work does not expose signed work contracts")
	}
	contracts, err := contractUOW.WorkContracts().List(ctx, f.scope, filter)
	if err != nil {
		t.Fatal(err)
	}
	signedContracts, err := signedContractUOW.SignedWorkContracts().List(ctx, f.scope, filter)
	if err != nil {
		t.Fatal(err)
	}
	return operationalParitySnapshot{
		state: state, events: events, receipts: receipts, work: work,
		contracts: contracts, signedContracts: signedContracts,
	}
}

func assertOperationalParityUnchanged(t *testing.T, f operationalParityFixture, before operationalParitySnapshot) {
	t.Helper()
	after := f.snapshot(t)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("rejected operational command changed Outcome revision, entity versions, events, receipts or work authority\nbefore=%#v\nafter=%#v", before, after)
	}
	for id, state := range after.work {
		if state.State.LeaseStatus != d.LeaseStatusNone {
			t.Fatalf("rejected command created a lease for Task %s: %#v", id, state.State)
		}
	}
}

func assertParityDenial(t *testing.T, failure operationalParityFailure, want string) {
	t.Helper()
	if failure.Code != string(d.ErrorCodePreconditionFailed) || failure.ReasonCode != want || !reflect.DeepEqual(failure.ReasonCodes, []string{want}) {
		t.Fatalf("denial=%#v, want precondition_failed with reason_code(s) %q", failure, want)
	}
}

func TestOperationalLifecycleDenialParityAcrossTransports(t *testing.T) {
	ctx := context.Background()
	fixture := newOperationalParityFixture(t)
	generator := &ids{}
	auth, err := local.New("e13-operational-parity")
	if err != nil {
		t.Fatal(err)
	}
	identity := a.Identity{PrincipalID: auth.Principal(), Actor: auth.Actor(), NamespaceID: fixture.scope.NamespaceID}
	handler, err := ht.New(ht.Options{Prefix: "/api/v1", RequestTimeout: 10 * time.Second, Service: fixture.service, IDs: generator, LocalAuth: auth})
	if err != nil {
		t.Fatal(err)
	}
	protocol, err := mt.New(fixture.service, generator, mt.Options{LocalIdentity: &identity})
	if err != nil {
		t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/api/", handler)
	mux.Handle("/mcp", mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return protocol }, &mcp.StreamableHTTPOptions{JSONResponse: true}))
	server := httptest.NewServer(mux)
	defer server.Close()
	client, err := sdk.New(server.URL, "", server.Client())
	if err != nil {
		t.Fatal(err)
	}
	session, err := mcp.NewClient(&mcp.Implementation{Name: "operational-parity-test", Version: "1"}, nil).Connect(ctx, &mcp.StreamableClientTransport{Endpoint: server.URL + "/mcp"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer session.Close()

	// E13 keeps the generated catalog, SDK, MCP and historical resource routes
	// on one fixture so the same gate and untouched authority can be compared.
	var catalog struct {
		Commands []struct {
			Name string `json:"name"`
		} `json:"commands"`
	}
	rawCatalog, err := client.QueryJSON(ctx, "/commands")
	if err != nil || json.Unmarshal(rawCatalog, &catalog) != nil {
		t.Fatalf("read HTTP command catalog: %v", err)
	}
	for _, name := range []string{"activate_work_item", "claim_work_item", "create_work_item"} {
		found := false
		for _, command := range catalog.Commands {
			if command.Name == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("HTTP command catalog omitted %q", name)
		}
	}

	base := fmt.Sprintf("%s/api/v1/namespaces/%s/outcomes/%s", server.URL, fixture.scope.NamespaceID, fixture.scope.OutcomeID)
	activate := a.ActivateWorkItemCommand{Scope: fixture.scope, WorkItemID: fixture.backlogTask.ID, ExpectedVersion: fixture.backlogTask.Version}
	claim := a.ClaimWorkItemCommand{Scope: fixture.scope, WorkItemID: fixture.claimableTask.ID, ExpectedVersion: fixture.claimableTask.Version, TTL: d.DefaultLeaseTTL}
	createTodo := a.CreateWorkItemCommand{
		Scope: fixture.scope, ObjectiveID: &fixture.plannedObjectiveID, Title: "Direct To Do creation gate",
		Description: "This Task must not bypass activation readiness", Priority: d.PriorityNormal,
		Lifecycle:       d.WorkItemLifecycleTodo,
		InitialCriteria: []a.InitialCriterion{{Title: "Definition is complete", Required: true, VerificationMode: d.VerificationModeAttestation}},
	}

	tests := []struct {
		name       string
		wantReason string
		command    any
		catalog    string
		resource   string
		restBody   any
		callSDK    func() error
		callMCP    func(string) operationalParityFailure
	}{
		{
			name:       "E09 activation",
			wantReason: string(d.ReadinessReasonObjectiveNotStarted),
			command:    activate, catalog: "activate_work_item", resource: "work-items/" + fixture.backlogTask.ID.String() + "/actions/activate", restBody: map[string]any{},
			callSDK: func() error { _, err := client.ActivateWorkItem(ctx, "e13-sdk-activate-0001", activate); return err },
			callMCP: func(key string) operationalParityFailure {
				return callParityMCP(t, ctx, session, "wos_activate_work_item", key, activate)
			},
		},
		{
			name:       "E08 legacy claim",
			wantReason: string(d.ReadinessReasonObjectiveNotStarted),
			command:    claim, catalog: "claim_work_item", resource: "work-items/" + fixture.claimableTask.ID.String() + "/actions/claim", restBody: map[string]any{"lease_ttl_seconds": int64(d.DefaultLeaseTTL / time.Second)},
			callSDK: func() error { _, err := client.ClaimWorkItem(ctx, "e13-sdk-claim-0001", claim); return err },
			callMCP: func(key string) operationalParityFailure {
				return callParityMCP(t, ctx, session, "wos_claim_work_item", key, claim)
			},
		},
		{
			name:       "E09 direct To Do creation",
			wantReason: string(d.ReadinessReasonObjectiveNotStarted),
			command:    createTodo, catalog: "create_work_item", resource: "", restBody: nil,
			callSDK: func() error { _, err := client.CreateWorkItem(ctx, "e13-sdk-create-todo-0001", createTodo); return err },
			callMCP: func(key string) operationalParityFailure {
				return callParityMCP(t, ctx, session, "wos_create_work_item", key, createTodo)
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			baseline := fixture.snapshot(t)
			failures := make([]operationalParityFailure, 0, 4)
			if test.resource != "" {
				failures = append(failures, callParityHistoricalREST(t, server.Client(), base+"/"+test.resource, test.restBody, test.command, fixture))
				assertOperationalParityUnchanged(t, fixture, baseline)
			}

			failures = append(failures, callParityCommandCatalog(t, server.Client(), base, test.catalog, "e13-catalog-"+strings.ReplaceAll(test.catalog, "_", "-")+"-0001", test.command, fixture))
			assertOperationalParityUnchanged(t, fixture, baseline)

			sdkErr := test.callSDK()
			failures = append(failures, parityFailureFromSDK(t, sdkErr))
			assertOperationalParityUnchanged(t, fixture, baseline)

			mcpFailure := test.callMCP("e13-mcp-" + strings.ReplaceAll(test.catalog, "_", "-") + "-0001")
			failures = append(failures, mcpFailure)
			assertOperationalParityUnchanged(t, fixture, baseline)

			for i, failure := range failures {
				assertParityDenial(t, failure, test.wantReason)
				if i > 0 && !reflect.DeepEqual(failures[0], failure) {
					t.Fatalf("transport error details diverged: first=%#v transport[%d]=%#v", failures[0], i, failure)
				}
			}
		})
	}

	// The legacy CLI shares the same server fixture. Checkout exercises the E08
	// acquisition gate; planning creation exercises E09 without adding an
	// alternate lifecycle or execution capability to the CLI.
	workspace := t.TempDir()
	var stdout, stderr bytes.Buffer
	if code := cli.Run(ctx, []string{"init", "--server", server.URL, "--namespace", fixture.scope.NamespaceID.String(), "--outcome", fixture.scope.OutcomeID.String(), "--workspace", workspace, "--output", "json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("CLI init: code=%d stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	cliBaseline := fixture.snapshot(t)
	stdout.Reset()
	stderr.Reset()
	code := cli.Run(ctx, []string{"work", "checkout", fixture.claimableTask.ID.String(), "--version", fmt.Sprint(fixture.claimableTask.Version), "--workspace", workspace, "--output", "json"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("CLI checkout unexpectedly passed planned Objective gate: %s", stdout.String())
	}
	var checkoutOutput cli.Output
	if err := json.Unmarshal(stdout.Bytes(), &checkoutOutput); err != nil || checkoutOutput.ErrorCode != string(d.ErrorCodePreconditionFailed) || checkoutOutput.ErrorDetails == nil {
		t.Fatalf("CLI checkout did not return structured operational denial: %#v err=%v stderr=%s", checkoutOutput, err, stderr.String())
	}
	assertParityDenial(t, operationalParityFailure{Code: checkoutOutput.ErrorCode, ReasonCode: checkoutOutput.ErrorDetails.ReasonCode, ReasonCodes: checkoutOutput.ErrorDetails.ReasonCodes}, string(d.ReadinessReasonObjectiveNotStarted))
	assertOperationalParityUnchanged(t, fixture, cliBaseline)

	createFile := "create-todo.json"
	createBytes, err := commands.Encode(createTodo)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(workspace+"/"+createFile, createBytes, 0600); err != nil {
		t.Fatal(err)
	}
	stdout.Reset()
	stderr.Reset()
	code = cli.Run(ctx, []string{"work", "create", "--file", createFile, "--workspace", workspace, "--output", "json"}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("CLI planning creation unexpectedly bypassed To Do gate: %s", stdout.String())
	}
	var planningOutput cli.Output
	if err := json.Unmarshal(stdout.Bytes(), &planningOutput); err != nil || planningOutput.ErrorCode != string(d.ErrorCodePreconditionFailed) || planningOutput.ErrorDetails == nil {
		t.Fatalf("CLI planning did not return structured operational denial: %#v err=%v stderr=%s", planningOutput, err, stderr.String())
	}
	assertParityDenial(t, operationalParityFailure{Code: planningOutput.ErrorCode, ReasonCode: planningOutput.ErrorDetails.ReasonCode, ReasonCodes: planningOutput.ErrorDetails.ReasonCodes}, string(d.ReadinessReasonObjectiveNotStarted))
	assertOperationalParityUnchanged(t, fixture, cliBaseline)

	// E11 reopen/cancel/recovery cases and E12 capability freshness/remedy
	// contracts intentionally remain extension points. Add vertical assertions
	// only after those application and transport capabilities are finalized.
}

func callParityHistoricalREST(t *testing.T, client *http.Client, url string, body any, command any, fixture operationalParityFixture) operationalParityFailure {
	t.Helper()
	encodedCommand, err := commands.Encode(command)
	if err != nil {
		t.Fatal(err)
	}
	var requestBody []byte
	if body == nil {
		requestBody = encodedCommand
	} else {
		requestBody, err = json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
	}
	var item d.WorkItem
	switch cmd := command.(type) {
	case a.ActivateWorkItemCommand:
		item = fixture.backlogTask
		_ = cmd
	case a.ClaimWorkItemCommand:
		item = fixture.claimableTask
	case a.CreateWorkItemCommand:
		t.Fatalf("historical REST route does not accept initial criteria for direct To Do creation: %#v", cmd)
	default:
		t.Fatalf("unsupported historical REST command %T", command)
	}
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(requestBody))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", "e13-rest-"+strings.ReplaceAll(item.ID.String()[len(item.ID.String())-6:], "-", "")+"-0001")
	request.Header.Set("If-Match", fmt.Sprintf("\"work_item:%s:v%d\"", item.ID, item.Version))
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var envelope struct {
		Error struct {
			Code    string         `json:"code"`
			Details d.ErrorDetails `json:"details"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode historical REST denial (status %d): %v", response.StatusCode, err)
	}
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("historical REST status=%d, want 409; error=%#v", response.StatusCode, envelope.Error)
	}
	return operationalParityFailure{Code: envelope.Error.Code, ReasonCode: envelope.Error.Details.ReasonCode, ReasonCodes: envelope.Error.Details.ReasonCodes}
}

func callParityCommandCatalog(t *testing.T, client *http.Client, base, name, key string, command any, _ operationalParityFixture) operationalParityFailure {
	t.Helper()
	input, err := commands.Encode(command)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(map[string]json.RawMessage{"command": input})
	if err != nil {
		t.Fatal(err)
	}
	url := strings.TrimRight(strings.Split(base, "/api/v1/")[0], "/") + "/api/v1/commands/" + name
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", key)
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	var envelope struct {
		Error struct {
			Code    string         `json:"code"`
			Details d.ErrorDetails `json:"details"`
		} `json:"error"`
	}
	if err := json.NewDecoder(response.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode command catalog denial (status %d): %v", response.StatusCode, err)
	}
	if response.StatusCode != http.StatusConflict {
		t.Fatalf("command catalog %s status=%d, want 409; error=%#v", name, response.StatusCode, envelope.Error)
	}
	return operationalParityFailure{Code: envelope.Error.Code, ReasonCode: envelope.Error.Details.ReasonCode, ReasonCodes: envelope.Error.Details.ReasonCodes}
}

func parityFailureFromSDK(t *testing.T, err error) operationalParityFailure {
	t.Helper()
	var remote *sdk.Error
	if !errors.As(err, &remote) || remote == nil || remote.Details == nil {
		t.Fatalf("SDK denial did not preserve structured details: %T %v", err, err)
	}
	return operationalParityFailure{Code: remote.Code, ReasonCode: remote.Details.ReasonCode, ReasonCodes: remote.Details.ReasonCodes}
}

func callParityMCP(t *testing.T, ctx context.Context, session *mcp.ClientSession, tool, key string, command any) operationalParityFailure {
	t.Helper()
	input, err := commands.Encode(command)
	if err != nil {
		t.Fatal(err)
	}
	result, err := session.CallTool(ctx, &mcp.CallToolParams{Name: tool, Arguments: map[string]any{"idempotency_key": key, "command": json.RawMessage(input)}})
	if err != nil {
		t.Fatalf("MCP tool transport failed before returning domain denial: %v", err)
	}
	if !result.IsError {
		t.Fatalf("MCP %s unexpectedly succeeded: %#v", tool, result)
	}
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatal(err)
	}
	var envelope struct {
		Error struct {
			Code    string         `json:"code"`
			Details d.ErrorDetails `json:"details"`
		} `json:"error"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("decode MCP %s denial %s: %v", tool, raw, err)
	}
	return operationalParityFailure{Code: envelope.Error.Code, ReasonCode: envelope.Error.Details.ReasonCode, ReasonCodes: envelope.Error.Details.ReasonCodes}
}
