package application_test

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

type operationalPolicyTestAuthorizer struct {
	requests  []ports.AuthorizationRequest
	denyAdmin bool
}

func (a *operationalPolicyTestAuthorizer) Authorize(_ context.Context, request ports.AuthorizationRequest) error {
	a.requests = append(a.requests, request)
	if err := request.Validate(); err != nil {
		return err
	}
	if a.denyAdmin && request.Permission == ports.PermissionNamespaceAdmin {
		return domain.NewError(domain.ErrorCodeForbidden, "namespace admin permission denied")
	}
	return nil
}

type operationalPolicyFixture struct {
	service    *application.Service
	store      *memory.Store
	authorizer *operationalPolicyTestAuthorizer
	scope      domain.Scope
	identity   application.Identity
	ctx        context.Context
	created    application.MutationResult[domain.Outcome]
	command    domain.CommandContext
}

func newOperationalPolicyFixture(t *testing.T) operationalPolicyFixture {
	t.Helper()
	store := memory.New()
	authorizer := &operationalPolicyTestAuthorizer{}
	service, err := application.NewAuthorizedService(store, fixedClock{now: time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)}, &sequenceIDs{}, authorizer)
	if err != nil {
		t.Fatal(err)
	}
	namespaceID := id("0199f100-0000-7000-8000-000000000001")
	command := commandContext()
	identity := application.Identity{NamespaceID: namespaceID, PrincipalID: command.PrincipalID, Actor: command.Actor}
	ctx := application.WithIdentity(context.Background(), identity)
	created, err := service.CreateOutcome(ctx, command, application.CreateOutcomeCommand{
		NamespaceID: namespaceID, Title: "Profile migration", DesiredState: "Migration only changes the operational profile", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	return operationalPolicyFixture{service: service, store: store, authorizer: authorizer, scope: created.Value.Scope(), identity: identity, ctx: ctx, created: created, command: command}
}

func migrationRevision(value domain.OutcomeRevision) *domain.OutcomeRevision { return &value }

func (f operationalPolicyFixture) migrationCommand(reason, key string, version domain.Version, revision domain.OutcomeRevision) (domain.CommandContext, application.ConfigureOutcomeOperationalProfileCommand) {
	command := f.command
	command.IdempotencyKey = key
	return command, application.ConfigureOutcomeOperationalProfileCommand{
		Scope: f.scope, ExpectedVersion: version, ExpectedOutcomeRevision: migrationRevision(revision), Reason: reason,
	}
}

func TestOperationalProfilePreviewIsAuthorizedAndReadOnly(t *testing.T) {
	f := newOperationalPolicyFixture(t)
	before, err := f.service.GetOutcomeState(f.ctx, f.scope)
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := f.store.SnapshotDomainEvents(f.scope)
	receiptsBefore := f.store.IdempotencyRecordCount()
	preview, err := f.service.PreviewOutcomeOperationalProfileMigration(f.ctx, f.scope)
	if err != nil {
		t.Fatalf("authorized preview failed: %v", err)
	}
	if !preview.Value.CanMigrate || preview.Value.Outcome.OperationalProfile != domain.OperationalProfileCompatibilityV1 || preview.Value.TargetOperationalProfile != domain.OperationalProfileOperationalV2 || preview.Value.TargetCompletionPolicy != domain.CompletionPolicyStrictWork {
		t.Fatalf("unexpected preview: %#v", preview.Value)
	}
	if preview.OutcomeRevision != before.OutcomeRevision || preview.Value.OutcomeRevision != before.OutcomeRevision {
		t.Fatalf("preview revision=%d/%d, want %d", preview.OutcomeRevision, preview.Value.OutcomeRevision, before.OutcomeRevision)
	}
	after, err := f.service.GetOutcomeState(f.ctx, f.scope)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(eventsBefore, f.store.SnapshotDomainEvents(f.scope)) || receiptsBefore != f.store.IdempotencyRecordCount() {
		t.Fatal("read-only preview changed persisted state, revision, event log, or idempotency receipts")
	}
	foundAdmin := false
	for _, request := range f.authorizer.requests {
		if request.Permission == ports.PermissionNamespaceAdmin && request.NamespaceID == f.scope.NamespaceID && request.OutcomeID == f.scope.OutcomeID && request.PrincipalID == f.identity.PrincipalID {
			foundAdmin = true
		}
	}
	if !foundAdmin {
		t.Fatal("preview did not authorize the authenticated principal as namespace administrator")
	}
}

func TestOperationalProfilePreviewRequiresNamespaceAdminIdentity(t *testing.T) {
	f := newOperationalPolicyFixture(t)
	f.authorizer.denyAdmin = true
	if _, err := f.service.PreviewOutcomeOperationalProfileMigration(f.ctx, f.scope); err == nil {
		t.Fatal("preview succeeded without NamespaceAdmin permission")
	}
	if _, err := f.service.PreviewOutcomeOperationalProfileMigration(context.Background(), f.scope); err == nil {
		t.Fatal("preview succeeded without authenticated identity")
	}
}

func TestConfigureOperationalProfileRejectsStaleVersionOrRevisionWithoutWrites(t *testing.T) {
	for _, test := range []struct {
		name     string
		version  domain.Version
		revision domain.OutcomeRevision
	}{
		{name: "stale outcome version", version: domain.Version(99), revision: 1},
		{name: "stale outcome revision", version: domain.InitialVersion, revision: 99},
	} {
		t.Run(test.name, func(t *testing.T) {
			f := newOperationalPolicyFixture(t)
			command, migration := f.migrationCommand("Move this Outcome to operational v2", map[string]string{"stale outcome version": "migration_stale_version", "stale outcome revision": "migration_stale_revision"}[test.name], test.version, test.revision)
			stateBefore, err := f.service.GetOutcomeState(f.ctx, f.scope)
			if err != nil {
				t.Fatal(err)
			}
			eventsBefore := f.store.SnapshotDomainEvents(f.scope)
			receiptsBefore := f.store.IdempotencyRecordCount()
			if _, err := f.service.ConfigureOutcomeOperationalProfile(f.ctx, command, migration); err == nil {
				t.Fatal("stale migration was accepted")
			} else if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeVersionConflict {
				t.Fatalf("error code=%s, want version conflict: %v", code, err)
			}
			stateAfter, err := f.service.GetOutcomeState(f.ctx, f.scope)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(stateBefore, stateAfter) || !reflect.DeepEqual(eventsBefore, f.store.SnapshotDomainEvents(f.scope)) || receiptsBefore != f.store.IdempotencyRecordCount() {
				t.Fatal("stale migration changed state, events, or durable receipts")
			}
		})
	}
}

func TestConfigureOperationalProfileAcceptsExplicitZeroRevision(t *testing.T) {
	store := memory.New()
	authorizer := &operationalPolicyTestAuthorizer{}
	service, err := application.NewAuthorizedService(store, fixedClock{now: time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)}, &sequenceIDs{}, authorizer)
	if err != nil {
		t.Fatal(err)
	}
	ns := id("0199f101-0000-7000-8000-000000000001")
	createdAt := time.Date(2026, 10, 21, 11, 0, 0, 0, time.UTC)
	outcome, err := domain.NewOutcome(id("0199f101-0000-7000-8000-000000000002"), ns, "Legacy", "", "State", domain.PriorityNormal, createdAt)
	if err != nil {
		t.Fatal(err)
	}
	uow, err := store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := uow.Outcomes().Insert(context.Background(), outcome); err != nil {
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}
	cc := commandContext()
	ctx := application.WithIdentity(context.Background(), application.Identity{NamespaceID: ns, PrincipalID: cc.PrincipalID, Actor: cc.Actor})
	zero := domain.OutcomeRevision(0)
	cc.IdempotencyKey = "initial-revision-migration"
	configured, err := service.ConfigureOutcomeOperationalProfile(ctx, cc, application.ConfigureOutcomeOperationalProfileCommand{
		Scope: outcome.Scope(), ExpectedVersion: outcome.Version, ExpectedOutcomeRevision: &zero, Reason: "Explicitly migrate initial state",
	})
	if err != nil {
		t.Fatalf("explicit zero revision was rejected: %v", err)
	}
	if configured.Value.OperationalProfile != domain.OperationalProfileOperationalV2 || configured.OutcomeRevision != 1 {
		t.Fatalf("configured result=%#v", configured)
	}
	var eventCount int
	for _, event := range store.SnapshotDomainEvents(outcome.Scope()) {
		if event.EventType == "outcome.operational_profile_configured" {
			eventCount++
		}
	}
	if eventCount != 1 {
		t.Fatalf("profile migration event count=%d, want 1", eventCount)
	}
}

func TestConfigureOperationalProfileRequiresRevisionAndReason(t *testing.T) {
	f := newOperationalPolicyFixture(t)
	for _, test := range []struct {
		name     string
		key      string
		revision *domain.OutcomeRevision
		reason   string
	}{
		{name: "missing revision", key: "invalid_revision", revision: nil, reason: "required reason"},
		{name: "missing reason", key: "invalid_reason_key", revision: migrationRevision(f.created.OutcomeRevision), reason: "  "},
	} {
		t.Run(test.name, func(t *testing.T) {
			cc := f.command
			cc.IdempotencyKey = test.key
			_, err := f.service.ConfigureOutcomeOperationalProfile(f.ctx, cc, application.ConfigureOutcomeOperationalProfileCommand{Scope: f.scope, ExpectedVersion: f.created.Value.Version, ExpectedOutcomeRevision: test.revision, Reason: test.reason})
			if err == nil {
				t.Fatal("invalid migration command succeeded")
			}
		})
	}
}

func TestConfigureOperationalProfileBlocksLegacyLeaseAndRollsBack(t *testing.T) {
	f := newOperationalPolicyFixture(t)
	createdObjective, err := f.service.CreateObjective(f.ctx, f.command, application.CreateObjectiveCommand{Scope: f.scope, Title: "Legacy objective", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	objectiveID := createdObjective.Value.ID
	createdTask, err := f.service.CreateWorkItem(f.ctx, f.command, application.CreateWorkItemCommand{Scope: f.scope, ObjectiveID: &objectiveID, Title: "Legacy work", Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleBacklog})
	if err != nil {
		t.Fatal(err)
	}
	claimTime := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	uow, err := f.store.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	item, err := uow.WorkItems().Get(f.ctx, f.scope, createdTask.Value.ID)
	if err != nil {
		t.Fatal(err)
	}
	expected := item.Version
	if err := item.Activate(claimTime); err != nil {
		t.Fatal(err)
	}
	if err := uow.WorkItems().Save(f.ctx, item, expected); err != nil {
		t.Fatal(err)
	}
	expected = item.Version
	if err := item.Claim(id("0199f102-0000-7000-8000-000000000009"), f.identity.PrincipalID, f.identity.Actor, 0, claimTime); err != nil {
		t.Fatal(err)
	}
	if err := uow.WorkItems().Save(f.ctx, item, expected); err != nil {
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}

	stateBefore, err := f.service.GetOutcomeState(f.ctx, f.scope)
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := f.store.SnapshotDomainEvents(f.scope)
	receiptsBefore := f.store.IdempotencyRecordCount()
	command, migration := f.migrationCommand("Blocked until legacy authority is reconciled", "blocked-legacy-lease", f.created.Value.Version, stateBefore.OutcomeRevision)
	if _, err := f.service.ConfigureOutcomeOperationalProfile(f.ctx, command, migration); err == nil {
		t.Fatal("migration accepted an unbound legacy CurrentLease")
	} else if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodePreconditionFailed {
		t.Fatalf("error code=%s, want precondition failed: %v", code, err)
	}
	stateAfter, err := f.service.GetOutcomeState(f.ctx, f.scope)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stateBefore, stateAfter) || !reflect.DeepEqual(eventsBefore, f.store.SnapshotDomainEvents(f.scope)) || receiptsBefore != f.store.IdempotencyRecordCount() {
		t.Fatal("blocked migration changed Outcome, Objective, Task, events, or receipt state")
	}
	preview, err := f.service.PreviewOutcomeOperationalProfileMigration(f.ctx, f.scope)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, finding := range preview.Value.Findings {
		if finding.Code == "legacy_lease_unreconciled" && finding.Ref != nil && finding.Ref.ID == createdTask.Value.ID && finding.Remedy != "" {
			found = true
		}
	}
	if !found {
		t.Fatalf("preview did not provide a concrete lease reference and remedy: %#v", preview.Value.Findings)
	}
}

func TestConfigureOperationalProfileBlocksActiveUnsignedContract(t *testing.T) {
	f := newOperationalPolicyFixture(t)
	const now = "2026-10-21T12:00:00Z"
	at, err := time.Parse(time.RFC3339, now)
	if err != nil {
		t.Fatal(err)
	}
	workID := id("0199f103-0000-7000-8000-000000000001")
	contractID := id("0199f103-0000-7000-8000-000000000002")
	executionID := id("0199f103-0000-7000-8000-000000000003")
	work, err := domain.NewWorkItem(workID, f.scope, "Contract work", "", domain.PriorityNormal, domain.WorkItemLifecycleTodo, at)
	if err != nil {
		t.Fatal(err)
	}
	work.ContractsEnabled = true
	contract, err := domain.NewWorkContract(contractID, executionID, work, f.identity.PrincipalID, f.identity.Actor, domain.DefaultContractLeasePolicy(), 300, domain.WorkContractSpec{Title: work.Title}, f.created.OutcomeRevision, at)
	if err != nil {
		t.Fatal(err)
	}
	work.Lifecycle = domain.WorkItemLifecycleInProgress
	work.CurrentContractID = &contractID
	work.Version++
	work.UpdatedAt = at
	if err := work.Validate(); err != nil {
		t.Fatal(err)
	}
	uow, err := f.store.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := uow.WorkItems().Insert(f.ctx, work); err != nil {
		t.Fatal(err)
	}
	contractUOW, ok := uow.(ports.WorkContractUnitOfWork)
	if !ok {
		t.Fatal("memory UOW lacks work contract test fixture port")
	}
	if err := contractUOW.WorkContracts().Insert(f.ctx, contract); err != nil {
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}
	stateBefore, err := f.service.GetOutcomeState(f.ctx, f.scope)
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := f.store.SnapshotDomainEvents(f.scope)
	command, migration := f.migrationCommand("Wait until unsigned work authority is reconciled", "active_unsigned_contract", f.created.Value.Version, stateBefore.OutcomeRevision)
	if _, err := f.service.ConfigureOutcomeOperationalProfile(f.ctx, command, migration); err == nil {
		t.Fatal("migration accepted an active unsigned work contract")
	} else if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodePreconditionFailed {
		t.Fatalf("error code=%s, want precondition failed: %v", code, err)
	}
	stateAfter, err := f.service.GetOutcomeState(f.ctx, f.scope)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(stateBefore, stateAfter) || !reflect.DeepEqual(eventsBefore, f.store.SnapshotDomainEvents(f.scope)) {
		t.Fatal("blocked contract migration changed persisted state or event history")
	}
	preview, err := f.service.PreviewOutcomeOperationalProfileMigration(f.ctx, f.scope)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, finding := range preview.Value.Findings {
		if finding.Code == "unsigned_work_authority_unbound" && finding.Ref != nil && finding.Ref.ID == workID && finding.Remedy != "" {
			found = true
		}
	}
	if !found {
		t.Fatalf("preview did not identify active unsigned contract and remedy: %#v", preview.Value.Findings)
	}
}

func TestConfigureOperationalProfileChangesOnlyOutcomePolicyAndReplaysDurably(t *testing.T) {
	f := newOperationalPolicyFixture(t)
	objective, err := f.service.CreateObjective(f.ctx, f.command, application.CreateObjectiveCommand{
		Scope: f.scope, Title: "Existing Objective", Priority: domain.PriorityNormal,
		InitialCriteria: []application.InitialCriterion{{Title: "Objective criterion", Required: true, VerificationMode: domain.VerificationModeAttestation}},
	})
	if err != nil {
		t.Fatal(err)
	}
	objectiveID := objective.Value.ID
	task, err := f.service.CreateWorkItem(f.ctx, f.command, application.CreateWorkItemCommand{
		Scope: f.scope, ObjectiveID: &objectiveID, Title: "Existing Task", Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleTodo,
		InitialCriteria: []application.InitialCriterion{{Title: "Task criterion", Required: true, VerificationMode: domain.VerificationModeAttestation}},
	})
	if err != nil {
		t.Fatal(err)
	}
	stateBefore, err := f.service.GetOutcomeState(f.ctx, f.scope)
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := f.store.SnapshotDomainEvents(f.scope)
	command, migration := f.migrationCommand("Enable the explicit operational profile", "replay-profile-migration", stateBefore.Outcome.Version, stateBefore.OutcomeRevision)
	configured, err := f.service.ConfigureOutcomeOperationalProfile(f.ctx, command, migration)
	if err != nil {
		t.Fatalf("configure: %v", err)
	}
	if configured.Value.OperationalProfile != domain.OperationalProfileOperationalV2 || configured.Value.CompletionPolicy != domain.CompletionPolicyStrictWork || configured.Value.Version != stateBefore.Outcome.Version+1 || configured.OutcomeRevision != stateBefore.OutcomeRevision+1 {
		t.Fatalf("unexpected configured Outcome: %#v", configured)
	}
	stateAfter, err := f.service.GetOutcomeState(f.ctx, f.scope)
	if err != nil {
		t.Fatal(err)
	}
	expectedOutcome := stateBefore.Outcome
	expectedOutcome.OperationalProfile = domain.OperationalProfileOperationalV2
	expectedOutcome.CompletionPolicy = domain.CompletionPolicyStrictWork
	expectedOutcome.Version++
	if !reflect.DeepEqual(expectedOutcome, stateAfter.Outcome) {
		t.Fatal("migration changed Outcome fields outside profile, completion policy, and version")
	}
	if !reflect.DeepEqual(stateBefore.Objectives, stateAfter.Objectives) || !reflect.DeepEqual(stateBefore.WorkItems, stateAfter.WorkItems) {
		t.Fatal("migration changed Objective or Task lifecycle, ownership, criteria, or other fields")
	}
	eventsAfter := f.store.SnapshotDomainEvents(f.scope)
	if len(eventsAfter) != len(eventsBefore)+1 {
		t.Fatalf("event count=%d, want %d", len(eventsAfter), len(eventsBefore)+1)
	}
	event := eventsAfter[len(eventsAfter)-1]
	if event.EventType != "outcome.operational_profile_configured" || event.OutcomeRevision != configured.OutcomeRevision || event.AggregateVersionBefore == nil || *event.AggregateVersionBefore != stateBefore.Outcome.Version || event.AggregateVersionAfter == nil || *event.AggregateVersionAfter != configured.Value.Version {
		t.Fatalf("migration event does not capture durable transition: %#v", event)
	}
	var payload struct {
		Reason                  string                 `json:"reason"`
		ExpectedOutcomeRevision domain.OutcomeRevision `json:"expected_outcome_revision,string"`
	}
	if err := json.Unmarshal(event.Payload, &payload); err != nil || payload.Reason != migration.Reason || payload.ExpectedOutcomeRevision != stateBefore.OutcomeRevision {
		t.Fatalf("migration event omitted audit reason/revision: payload=%s err=%v", event.Payload, err)
	}
	countAfterApply := len(eventsAfter)
	replay, err := f.service.ConfigureOutcomeOperationalProfile(f.ctx, command, migration)
	if err != nil {
		t.Fatalf("idempotent replay: %v", err)
	}
	if !replay.IdempotentReplay || replay.OutcomeRevision != configured.OutcomeRevision || !sameWorkspaceJSON(replay.Value, configured.Value) || len(f.store.SnapshotDomainEvents(f.scope)) != countAfterApply {
		t.Fatalf("idempotent replay changed migration: replay=%#v original=%#v event-count=%d/%d", replay, configured, len(f.store.SnapshotDomainEvents(f.scope)), countAfterApply)
	}
	if task.Value.Lifecycle != domain.WorkItemLifecycleTodo || objective.Value.Lifecycle != domain.ObjectiveLifecyclePlanned {
		t.Fatal("fixture lifecycles unexpectedly changed")
	}
}
