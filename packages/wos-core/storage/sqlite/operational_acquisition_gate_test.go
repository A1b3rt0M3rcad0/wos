package sqlite

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

func TestSignedAcquisitionAndNextRespectPlannedObjective(t *testing.T) {
	f := newResolutionFixture(t, "memory")
	store := f.store.(*memory.Store)
	scope := d.Scope{NamespaceID: d.ID(f.request.NamespaceID), OutcomeID: d.ID(f.request.OutcomeID)}
	task := preparePlannedObjectiveSignedCandidate(t, f, scope)
	beforeTask, beforeOutcome, beforeRevision := readSignedAcquisitionGateState(t, f.store, scope, task.ID)
	beforeEvents := store.SnapshotDomainEvents(scope)
	beforeReceipts := store.IdempotencyRecordCount()

	commandContext := f.cc
	commandContext.CommandID = testID("0199a555-0000-7000-8000-000000000088")
	commandContext.IdempotencyKey = "planned-objective-signed-id"
	_, err := f.service.AcquireSignedWorkContract(f.ctx, commandContext, a.AcquireSignedWorkContractCommand{
		Scope: scope, WorkItemID: task.ID, ExpectedWorkItemVersion: task.Version,
		SignerKeyID: d.ID(f.request.SignerKeyID), TTLSeconds: 30,
	})
	expectSignedPlannedObjectiveError(t, err)
	assertSignedAcquisitionGateState(t, f.store, store, scope, beforeTask, beforeOutcome, beforeRevision, beforeEvents)
	if got := store.IdempotencyRecordCount(); got != beforeReceipts {
		t.Fatalf("denied signed ID acquisition wrote a receipt: got %d, want %d", got, beforeReceipts)
	}

	commandContext.CommandID = testID("0199a555-0000-7000-8000-000000000089")
	commandContext.IdempotencyKey = "planned-objective-signed-next-empty"
	search := a.AcquireNextSignedWorkContractCommand{
		Scope: scope, SignerKeyID: d.ID(f.request.SignerKeyID), WorkItemIDs: []d.ID{task.ID}, TTLSeconds: 30,
	}
	result, err := f.service.AcquireNextSignedWorkContract(f.ctx, commandContext, search)
	if err != nil {
		t.Fatalf("signed bounded search should return an empty result: %v", err)
	}
	if result.Value.Acquired || result.Value.Result != nil || !result.Value.SearchComplete || !reflect.DeepEqual(result.Value.Reasons, []string{"no_eligible_work"}) {
		t.Fatalf("signed search=%#v, want a complete empty result", result.Value)
	}
	if result.OutcomeRevision != beforeRevision {
		t.Fatalf("signed empty search revision=%d, want %d", result.OutcomeRevision, beforeRevision)
	}
	assertSignedAcquisitionGateState(t, f.store, store, scope, beforeTask, beforeOutcome, beforeRevision, beforeEvents)
	if got := store.IdempotencyRecordCount(); got != beforeReceipts+1 {
		t.Fatalf("empty signed search receipt count=%d, want %d", got, beforeReceipts+1)
	}

	commandContext.CommandID = testID("0199a555-0000-7000-8000-000000000090")
	replay, err := f.service.AcquireNextSignedWorkContract(f.ctx, commandContext, search)
	if err != nil {
		t.Fatalf("signed empty search replay: %v", err)
	}
	if !replay.IdempotentReplay || replay.CommandID != result.CommandID || replay.OutcomeRevision != result.OutcomeRevision || !reflect.DeepEqual(replay.Value, result.Value) {
		t.Fatalf("signed replay=%#v, original=%#v", replay, result)
	}
	assertSignedAcquisitionGateState(t, f.store, store, scope, beforeTask, beforeOutcome, beforeRevision, beforeEvents)
	if got := store.IdempotencyRecordCount(); got != beforeReceipts+1 {
		t.Fatalf("signed replay wrote another receipt: got %d, want %d", got, beforeReceipts+1)
	}
}

func preparePlannedObjectiveSignedCandidate(t *testing.T, f resolutionFixture, scope d.Scope) d.WorkItem {
	t.Helper()
	uow, err := f.store.Begin(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	outcome, err := uow.Outcomes().Get(f.ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		t.Fatal(err)
	}
	objectives, err := uow.Objectives().ListByOutcome(f.ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(objectives) == 0 || objectives[0].Lifecycle != d.ObjectiveLifecyclePlanned {
		t.Fatal("signed fixture has no Planned Objective for the candidate")
	}
	now := time.Now().UTC()
	candidate, err := d.NewWorkItemForObjective(
		testID("0199a555-0000-7000-8000-000000000070"), scope, &objectives[0].ID,
		"Planned Objective gate candidate", "", d.PriorityNormal, d.WorkItemLifecycleBacklog, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	candidate.ContractsEnabled = true
	criterion, err := d.NewSuccessCriterion(
		testID("0199a555-0000-7000-8000-000000000071"), candidate.Ref(),
		"A required, verifiable result", "", true, d.VerificationModeEvidenceReview,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := candidate.AddCriterion(criterion, now); err != nil {
		t.Fatal(err)
	}
	if err := candidate.Activate(now); err != nil {
		t.Fatal(err)
	}
	if err := uow.WorkItems().Insert(f.ctx, candidate); err != nil {
		t.Fatal(err)
	}
	expectedOutcomeVersion := outcome.Version
	outcome.OperationalProfile = d.OperationalProfileOperationalV2
	outcome.Version, err = outcome.Version.Next()
	if err != nil {
		t.Fatal(err)
	}
	if err := uow.Outcomes().Save(f.ctx, outcome, expectedOutcomeVersion); err != nil {
		t.Fatal(err)
	}
	if _, err := uow.Coordination().AdvanceOutcome(f.ctx, scope); err != nil {
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}
	return candidate
}

func readSignedAcquisitionGateState(
	t *testing.T,
	store ports.TransactionManager,
	scope d.Scope,
	taskID d.ID,
) (d.WorkItem, d.Outcome, d.OutcomeRevision) {
	t.Helper()
	uow, err := store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	task, err := uow.WorkItems().Get(context.Background(), scope, taskID)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := uow.Outcomes().Get(context.Background(), scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		t.Fatal(err)
	}
	coordination, err := uow.Coordination().LockOutcome(context.Background(), scope)
	if err != nil {
		t.Fatal(err)
	}
	return task, outcome, coordination.Revision
}

func assertSignedAcquisitionGateState(
	t *testing.T,
	transaction ports.TransactionManager,
	store *memory.Store,
	scope d.Scope,
	task d.WorkItem,
	outcome d.Outcome,
	revision d.OutcomeRevision,
	events []d.DomainEvent,
) {
	t.Helper()
	gotTask, gotOutcome, gotRevision := readSignedAcquisitionGateState(t, transaction, scope, task.ID)
	if !reflect.DeepEqual(gotTask, task) {
		t.Fatal("denied/empty signed acquisition changed Task authority, lifecycle or version")
	}
	if !reflect.DeepEqual(gotOutcome, outcome) {
		t.Fatal("denied/empty signed acquisition changed Outcome")
	}
	if gotRevision != revision {
		t.Fatalf("denied/empty signed acquisition changed Outcome revision: got %d, want %d", gotRevision, revision)
	}
	if got := store.SnapshotDomainEvents(scope); !reflect.DeepEqual(got, events) {
		t.Fatal("denied/empty signed acquisition appended domain events")
	}
}

func expectSignedPlannedObjectiveError(t *testing.T, err error) {
	t.Helper()
	var coded *d.Error
	if err == nil || !errors.As(err, &coded) || coded.Details == nil {
		t.Fatalf("expected structured Planned Objective denial, got %T: %v", err, err)
	}
	if coded.Code != d.ErrorCodePreconditionFailed || coded.Details.ReasonCode != string(d.ReadinessReasonObjectiveNotStarted) {
		t.Fatalf("precondition=%#v, want objective_not_started", coded.Details)
	}
}
