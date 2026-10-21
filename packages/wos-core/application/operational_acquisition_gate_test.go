package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type acquisitionGateClock struct{ now time.Time }

func (c acquisitionGateClock) Now() time.Time { return c.now }

type acquisitionGateIDs struct{ next int }

func (g *acquisitionGateIDs) NewID() (domain.ID, error) {
	g.next++
	return domain.ParseID(fmt.Sprintf("0199f101-0000-7000-8000-%012x", g.next))
}

func newAcquisitionGateService(t *testing.T, fixture operationalGateFixture, now time.Time) (*Service, domain.CommandContext) {
	t.Helper()
	service, err := NewService(fixture.store, acquisitionGateClock{now: now}, &acquisitionGateIDs{})
	if err != nil {
		t.Fatal(err)
	}
	actor := domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "executor"}
	return service, domain.CommandContext{PrincipalID: "executor", Actor: actor, CommandID: operationalGateID(700)}
}

func readOperationalGateTask(t *testing.T, fixture operationalGateFixture, id domain.ID) domain.WorkItem {
	t.Helper()
	uow, err := fixture.store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	value, err := uow.WorkItems().Get(context.Background(), fixture.scope, id)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func readOperationalGateOutcome(t *testing.T, fixture operationalGateFixture) domain.Outcome {
	t.Helper()
	uow, err := fixture.store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	value, err := uow.Outcomes().Get(context.Background(), fixture.scope.NamespaceID, fixture.scope.OutcomeID)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func enableOperationalGateContracts(t *testing.T, fixture operationalGateFixture, taskID domain.ID) {
	t.Helper()
	uow, err := fixture.store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	item, err := uow.WorkItems().Get(context.Background(), fixture.scope, taskID)
	if err != nil {
		t.Fatal(err)
	}
	expected := item.Version
	item.ContractsEnabled = true
	item.Version, err = item.Version.Next()
	if err != nil {
		t.Fatal(err)
	}
	if err := uow.WorkItems().Save(context.Background(), item, expected); err != nil {
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}
}

func expectPlannedObjectiveGateError(t *testing.T, err error) {
	t.Helper()
	var coded *domain.Error
	if err == nil {
		t.Fatal("operation unexpectedly passed a Planned Objective gate")
	}
	if !errors.As(err, &coded) || coded.Details == nil {
		t.Fatalf("expected structured operational precondition, got %T: %v", err, err)
	}
	if coded.Code != domain.ErrorCodePreconditionFailed || coded.Details.ReasonCode != string(domain.ReadinessReasonObjectiveNotStarted) {
		t.Fatalf("precondition=%#v, want objective_not_started", coded.Details)
	}
}

func TestClaimAndReclaimRespectPlannedObjectiveWithoutMutation(t *testing.T) {
	ctx := context.Background()
	for _, test := range []string{"claim", "reclaim", "contract_acquire"} {
		t.Run(test, func(t *testing.T) {
			fixture := newOperationalGateFixture(t, domain.OperationalProfileOperationalV2, domain.ObjectiveLifecycleInProgress, domain.ObjectiveLifecyclePlanned)
			if test == "contract_acquire" {
				enableOperationalGateContracts(t, fixture, fixture.task.ID)
			}
			if test == "reclaim" {
				uow, err := fixture.store.Begin(ctx)
				if err != nil {
					t.Fatal(err)
				}
				item, err := uow.WorkItems().Get(ctx, fixture.scope, fixture.task.ID)
				if err != nil {
					t.Fatal(err)
				}
				expected := item.Version
				if err := item.Claim(operationalGateID(710), "prior-worker", domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "prior-worker"}, domain.MinLeaseTTL, operationalGateNow.Add(-domain.MinLeaseTTL)); err != nil {
					t.Fatal(err)
				}
				if err := uow.WorkItems().Save(ctx, item, expected); err != nil {
					t.Fatal(err)
				}
				if err := uow.Commit(); err != nil {
					t.Fatal(err)
				}
			}

			service, cc := newAcquisitionGateService(t, fixture, operationalGateNow)
			beforeTask := readOperationalGateTask(t, fixture, fixture.task.ID)
			beforeOutcome := readOperationalGateOutcome(t, fixture)
			beforeEvents := fixture.store.SnapshotDomainEvents(fixture.scope)
			beforeReceipts := fixture.store.IdempotencyRecordCount()
			if test == "claim" {
				_, err := service.ClaimWorkItem(ctx, cc, ClaimWorkItemCommand{Scope: fixture.scope, WorkItemID: beforeTask.ID, ExpectedVersion: beforeTask.Version, TTL: domain.DefaultLeaseTTL})
				expectPlannedObjectiveGateError(t, err)
			} else if test == "reclaim" {
				_, err := service.ReclaimWorkItem(ctx, cc, ReclaimWorkItemCommand{Scope: fixture.scope, WorkItemID: beforeTask.ID, ExpectedVersion: beforeTask.Version, TTL: domain.DefaultLeaseTTL})
				expectPlannedObjectiveGateError(t, err)
			} else {
				_, err := service.AcquireWorkContract(ctx, cc, AcquireWorkContractCommand{Scope: fixture.scope, WorkItemID: beforeTask.ID, ExpectedWorkItemVersion: beforeTask.Version})
				expectPlannedObjectiveGateError(t, err)
			}
			afterTask := readOperationalGateTask(t, fixture, fixture.task.ID)
			afterOutcome := readOperationalGateOutcome(t, fixture)
			if !reflect.DeepEqual(beforeTask, afterTask) || !reflect.DeepEqual(beforeOutcome, afterOutcome) {
				t.Fatalf("denied %s changed Task or Outcome state", test)
			}
			if beforeReceipts != fixture.store.IdempotencyRecordCount() || !reflect.DeepEqual(beforeEvents, fixture.store.SnapshotDomainEvents(fixture.scope)) {
				t.Fatalf("denied %s wrote a receipt or event", test)
			}
		})
	}
}

func TestReadyWorkAndContractDetailSharePlannedObjectiveReason(t *testing.T) {
	ctx := context.Background()
	fixture := newOperationalGateFixture(t, domain.OperationalProfileCompatibilityV1, domain.ObjectiveLifecycleInProgress, domain.ObjectiveLifecyclePlanned)
	service, cc := newAcquisitionGateService(t, fixture, operationalGateNow)

	enableOperationalGateContracts(t, fixture, fixture.task.ID)
	item := readOperationalGateTask(t, fixture, fixture.task.ID)
	secondTask, err := domain.NewWorkItemForObjective(operationalGateID(32), fixture.scope, &fixture.child.ID, "Available candidate", "", domain.PriorityNormal, domain.WorkItemLifecycleTodo, operationalGateNow)
	if err != nil {
		t.Fatal(err)
	}
	secondTask.ContractsEnabled = true
	addOperationalGateCriterion(t, &secondTask, operationalGateID(33))
	uow, err := fixture.store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := uow.WorkItems().Insert(ctx, secondTask); err != nil {
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}

	cc.IdempotencyKey = "planned-parent-contract-acquire"
	acquired, err := service.AcquireWorkContract(ctx, cc, AcquireWorkContractCommand{Scope: fixture.scope, WorkItemID: item.ID, ExpectedWorkItemVersion: item.Version})
	if err != nil {
		t.Fatalf("compatibility acquisition setup failed: %v", err)
	}

	uow, err = fixture.store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := uow.Outcomes().Get(ctx, fixture.scope.NamespaceID, fixture.scope.OutcomeID)
	if err != nil {
		t.Fatal(err)
	}
	expectedOutcomeVersion := outcome.Version
	outcome.OperationalProfile = domain.OperationalProfileOperationalV2
	nextOutcomeVersion, err := outcome.Version.Next()
	if err != nil {
		t.Fatal(err)
	}
	outcome.Version = nextOutcomeVersion
	if err := uow.Outcomes().Save(ctx, outcome, expectedOutcomeVersion); err != nil {
		t.Fatal(err)
	}
	if _, err := uow.Coordination().AdvanceOutcome(ctx, fixture.scope); err != nil {
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}

	ready, err := service.ListReadyWork(ctx, fixture.scope)
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range ready.Items {
		if value.WorkItem.ID == secondTask.ID {
			t.Fatal("Task under a Planned Objective appeared in ready-work")
		}
	}
	available, err := service.ListAvailableWork(ctx, fixture.scope, 50, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, value := range available.Items {
		if value.Candidate.ID == secondTask.ID {
			t.Fatal("Task under a Planned Objective appeared in available-work")
		}
	}
	func() {
		uow, err := fixture.store.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer uow.Rollback()
		candidate, err := uow.WorkItems().Get(ctx, fixture.scope, secondTask.ID)
		if err != nil {
			t.Fatal(err)
		}
		readiness, err := evaluateTaskAcquisition(ctx, uow, candidate, operationalGateNow)
		if err != nil {
			t.Fatal(err)
		}
		if !hasGateReason(readiness, domain.ReadinessReasonObjectiveNotStarted) {
			t.Fatalf("ready-query shared gate reasons=%v, want objective_not_started", readiness.Reasons)
		}
	}()
	view, err := service.GetWorkContract(ctx, fixture.scope, acquired.Value.Contract.ID)
	if err != nil {
		t.Fatal(err)
	}
	if view.Value.ExecutionAllowed || !containsOperationalReason(view.Value.Reasons, domain.ReadinessReasonObjectiveNotStarted) {
		t.Fatalf("detailed contract readiness=%#v, want objective_not_started", view.Value)
	}
}

func TestUnsignedNextSkipsTaskUnderPlannedObjectiveAndReplaysEmptySearch(t *testing.T) {
	ctx := context.Background()
	fixture := newOperationalGateFixture(t, domain.OperationalProfileOperationalV2, domain.ObjectiveLifecycleInProgress, domain.ObjectiveLifecyclePlanned)
	enableOperationalGateContracts(t, fixture, fixture.task.ID)
	setup, err := fixture.store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := setup.Coordination().AdvanceOutcome(ctx, fixture.scope); err != nil {
		t.Fatal(err)
	}
	if err := setup.Commit(); err != nil {
		t.Fatal(err)
	}
	service, cc := newAcquisitionGateService(t, fixture, operationalGateNow)
	cc.IdempotencyKey = "planned-objective-unsigned-next"
	command := AcquireNextWorkContractCommand{Scope: fixture.scope, WorkItemIDs: []domain.ID{fixture.task.ID}}
	beforeTask := readOperationalGateTask(t, fixture, fixture.task.ID)
	beforeOutcome := readOperationalGateOutcome(t, fixture)
	beforeEvents := fixture.store.SnapshotDomainEvents(fixture.scope)
	beforeReceipts := fixture.store.IdempotencyRecordCount()
	beforeRevision := readOperationalGateRevision(t, fixture)

	result, err := service.AcquireNextWorkContract(ctx, cc, command)
	if err != nil {
		t.Fatalf("bounded search should return an empty result, got %v", err)
	}
	if result.Value.Acquired || result.Value.Result != nil || !result.Value.SearchComplete || !reflect.DeepEqual(result.Value.Reasons, []string{"no_eligible_work"}) {
		t.Fatalf("search=%#v, want a complete empty result", result.Value)
	}
	if result.OutcomeRevision != beforeRevision {
		t.Fatalf("empty search revision=%d, want %d", result.OutcomeRevision, beforeRevision)
	}
	assertAcquisitionGateUnchanged(t, fixture, beforeTask, beforeOutcome, beforeEvents, beforeRevision)
	if got := fixture.store.IdempotencyRecordCount(); got != beforeReceipts+1 {
		t.Fatalf("empty search receipt count=%d, want %d", got, beforeReceipts+1)
	}

	cc.CommandID = operationalGateID(702)
	replay, err := service.AcquireNextWorkContract(ctx, cc, command)
	if err != nil {
		t.Fatalf("empty search replay: %v", err)
	}
	if !replay.IdempotentReplay || replay.CommandID != result.CommandID || replay.OutcomeRevision != result.OutcomeRevision || !reflect.DeepEqual(replay.Value, result.Value) {
		t.Fatalf("replay=%#v, original=%#v", replay, result)
	}
	assertAcquisitionGateUnchanged(t, fixture, beforeTask, beforeOutcome, beforeEvents, beforeRevision)
	if got := fixture.store.IdempotencyRecordCount(); got != beforeReceipts+1 {
		t.Fatalf("replay wrote another receipt: count=%d, want %d", got, beforeReceipts+1)
	}
}

func readOperationalGateRevision(t *testing.T, fixture operationalGateFixture) domain.OutcomeRevision {
	t.Helper()
	uow, err := fixture.store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	coordination, err := uow.Coordination().LockOutcome(context.Background(), fixture.scope)
	if err != nil {
		t.Fatal(err)
	}
	return coordination.Revision
}

func assertAcquisitionGateUnchanged(
	t *testing.T,
	fixture operationalGateFixture,
	task domain.WorkItem,
	outcome domain.Outcome,
	events []domain.DomainEvent,
	revision domain.OutcomeRevision,
) {
	t.Helper()
	if after := readOperationalGateTask(t, fixture, task.ID); !reflect.DeepEqual(task, after) {
		t.Fatal("empty acquisition search changed Task authority, lifecycle or version")
	}
	if after := readOperationalGateOutcome(t, fixture); !reflect.DeepEqual(outcome, after) {
		t.Fatal("empty acquisition search changed Outcome")
	}
	if after := readOperationalGateRevision(t, fixture); after != revision {
		t.Fatalf("empty acquisition search changed Outcome revision: got %d, want %d", after, revision)
	}
	if after := fixture.store.SnapshotDomainEvents(fixture.scope); !reflect.DeepEqual(events, after) {
		t.Fatal("empty acquisition search appended domain events")
	}
}

func containsOperationalReason(values []domain.ReadinessReason, target domain.ReadinessReason) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

var _ ports.IDGenerator = (*acquisitionGateIDs)(nil)
