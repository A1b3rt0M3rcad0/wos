package application

import (
	"context"
	"errors"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func operationalReopenCommand(n int) domain.CommandContext {
	return domain.CommandContext{
		PrincipalID: "operator",
		Actor:       domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "test", ID: "operator"},
		CommandID:   operationalGateID(5000 + n),
	}
}

func newOperationalReopenService(t *testing.T, fixture operationalGateFixture) *Service {
	t.Helper()
	service, err := NewService(fixture.store, acquisitionGateClock{now: operationalGateNow}, &acquisitionGateIDs{})
	if err != nil {
		t.Fatal(err)
	}
	return service
}

type overflowStructureTransactionManager struct {
	ports.TransactionManager
}

func (m overflowStructureTransactionManager) Begin(ctx context.Context) (ports.UnitOfWork, error) {
	uow, err := m.TransactionManager.Begin(ctx)
	if err != nil {
		return nil, err
	}
	reads, ok := uow.(ports.ContinuityReadUnitOfWork)
	if !ok {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "fixture unit of work lacks continuity reads")
	}
	workspace, ok := reads.ContinuityReads().(ports.WorkspaceReadRepository)
	if !ok {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "fixture unit of work lacks workspace reads")
	}
	return overflowStructureUnitOfWork{
		UnitOfWork: uow,
		reads: overflowStructureReadRepository{
			ContinuityReadRepository: reads.ContinuityReads(),
			WorkspaceReadRepository:  workspace,
		},
	}, nil
}

type overflowStructureUnitOfWork struct {
	ports.UnitOfWork
	reads overflowStructureReadRepository
}

func (u overflowStructureUnitOfWork) ContinuityReads() ports.ContinuityReadRepository {
	return u.reads
}

type overflowStructureReadRepository struct {
	ports.ContinuityReadRepository
	ports.WorkspaceReadRepository
}

func (r overflowStructureReadRepository) WorkspaceStructure(_ context.Context, _ domain.Scope, limit int) ([]domain.Objective, []domain.WorkItem, error) {
	return make([]domain.Objective, limit), nil, nil
}

func cancelFixtureTask(t *testing.T, fixture operationalGateFixture) domain.WorkItem {
	t.Helper()
	ctx := context.Background()
	uow, err := fixture.store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	item, err := uow.WorkItems().Get(ctx, fixture.scope, fixture.task.ID)
	if err != nil {
		t.Fatal(err)
	}
	expected := item.Version
	conclusion := domain.Conclusion{PrincipalID: "operator", Actor: operationalReopenCommand(1).Actor, Reason: "fixture task cancellation", ConcludedAt: operationalGateNow}
	if err := item.Cancel(conclusion, operationalGateNow); err != nil {
		t.Fatal(err)
	}
	if err := uow.WorkItems().Save(ctx, item, expected); err != nil {
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}
	return item
}

func achieveFixtureObjective(t *testing.T, fixture operationalGateFixture, objectiveID domain.ID) domain.Objective {
	t.Helper()
	ctx := context.Background()
	uow, err := fixture.store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	objective, err := uow.Objectives().Get(ctx, fixture.scope, objectiveID)
	if err != nil {
		t.Fatal(err)
	}
	expected := objective.Version
	if len(objective.Criteria.Items) == 0 {
		t.Fatal("fixture Objective has no criterion")
	}
	criterion := objective.Criteria.Items[0]
	assessment := domain.CriterionAssessment{
		ID:                operationalGateID(900),
		CriterionID:       criterion.ID,
		CriterionRevision: criterion.Revision,
		Result:            domain.AssessmentResultMet,
		Rationale:         "fixture criterion verified",
		PrincipalID:       "operator",
		Actor:             operationalReopenCommand(2).Actor,
		AssessedAt:        operationalGateNow,
	}
	if err := objective.AssessCriterionAttestation(assessment, operationalGateNow); err != nil {
		t.Fatal(err)
	}
	if err := uow.Objectives().Save(ctx, objective, expected); err != nil {
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}

	uow, err = fixture.store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	objective, err = uow.Objectives().Get(ctx, fixture.scope, objectiveID)
	if err != nil {
		t.Fatal(err)
	}
	expected = objective.Version
	conclusion := domain.Conclusion{PrincipalID: "operator", Actor: operationalReopenCommand(2).Actor, Reason: "fixture Objective achieved", ConcludedAt: operationalGateNow}
	if err := objective.Achieve(conclusion, operationalGateNow); err != nil {
		t.Fatal(err)
	}
	if err := uow.Objectives().Save(ctx, objective, expected); err != nil {
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}
	return objective
}

type operationalReopenSnapshot struct {
	outcome  domain.Outcome
	root     domain.Objective
	child    domain.Objective
	task     domain.WorkItem
	revision domain.OutcomeRevision
	events   []domain.DomainEvent
}

func readOperationalReopenSnapshot(t *testing.T, fixture operationalGateFixture) operationalReopenSnapshot {
	t.Helper()
	ctx := context.Background()
	var snapshot operationalReopenSnapshot
	func() {
		uow, err := fixture.store.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		defer uow.Rollback()
		coord, err := uow.Coordination().LockOutcome(ctx, fixture.scope)
		if err != nil {
			t.Fatal(err)
		}
		outcome, err := uow.Outcomes().Get(ctx, fixture.scope.NamespaceID, fixture.scope.OutcomeID)
		if err != nil {
			t.Fatal(err)
		}
		root, err := uow.Objectives().Get(ctx, fixture.scope, fixture.root.ID)
		if err != nil {
			t.Fatal(err)
		}
		child, err := uow.Objectives().Get(ctx, fixture.scope, fixture.child.ID)
		if err != nil {
			t.Fatal(err)
		}
		task, err := uow.WorkItems().Get(ctx, fixture.scope, fixture.task.ID)
		if err != nil {
			t.Fatal(err)
		}
		snapshot = operationalReopenSnapshot{outcome: outcome, root: root, child: child, task: task, revision: coord.Revision}
	}()
	snapshot.events = fixture.store.SnapshotDomainEvents(fixture.scope)
	return snapshot
}

func TestOperationalV2ReopenTaskRequiresActiveObjectiveAncestry(t *testing.T) {
	ctx := context.Background()
	tests := []struct {
		name      string
		rootState domain.ObjectiveLifecycle
		child     domain.ObjectiveLifecycle
		achieve   domain.ID
		want      domain.ReadinessReason
	}{
		{name: "planned owner", rootState: domain.ObjectiveLifecycleInProgress, child: domain.ObjectiveLifecyclePlanned, want: domain.ReadinessReasonObjectiveNotStarted},
		{name: "achieved owner", rootState: domain.ObjectiveLifecycleInProgress, child: domain.ObjectiveLifecycleInProgress, achieve: operationalGateID(21), want: domain.ReadinessReasonObjectiveTerminal},
		{name: "planned ancestor", rootState: domain.ObjectiveLifecyclePlanned, child: domain.ObjectiveLifecycleInProgress, want: domain.ReadinessReasonAncestorObjectiveNotStarted},
		{name: "achieved ancestor", rootState: domain.ObjectiveLifecycleInProgress, child: domain.ObjectiveLifecycleInProgress, achieve: operationalGateID(20), want: domain.ReadinessReasonAncestorObjectiveTerminal},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newOperationalGateFixture(t, domain.OperationalProfileOperationalV2, test.rootState, test.child)
			if !test.achieve.IsZero() {
				achieveFixtureObjective(t, fixture, test.achieve)
			}
			terminal := cancelFixtureTask(t, fixture)
			service := newOperationalReopenService(t, fixture)
			before := readOperationalReopenSnapshot(t, fixture)
			_, err := service.ReopenWorkItem(ctx, operationalReopenCommand(10), ReopenWorkItemCommand{Scope: fixture.scope, WorkItemID: terminal.ID, ExpectedVersion: terminal.Version, Reason: "resume required work"})
			var typed *domain.Error
			if !errors.As(err, &typed) || typed.Details == nil || typed.Details.ReasonCode != string(test.want) {
				t.Fatalf("reopen error=%v, want reason %s", err, test.want)
			}
			after := readOperationalReopenSnapshot(t, fixture)
			if !reflect.DeepEqual(before, after) {
				t.Fatal("rejected v2 reopen changed task, hierarchy, Outcome revision, or events")
			}
		})
	}
}

func TestCompatibilityReopenKeepsHistoricalAncestryBehavior(t *testing.T) {
	ctx := context.Background()
	fixture := newOperationalGateFixture(t, domain.OperationalProfileCompatibilityV1, domain.ObjectiveLifecyclePlanned, domain.ObjectiveLifecyclePlanned)
	terminal := cancelFixtureTask(t, fixture)
	service := newOperationalReopenService(t, fixture)
	result, err := service.ReopenWorkItem(ctx, operationalReopenCommand(20), ReopenWorkItemCommand{Scope: fixture.scope, WorkItemID: terminal.ID, ExpectedVersion: terminal.Version, Reason: "compatibility retry"})
	if err != nil {
		t.Fatalf("compatibility reopen was tightened by operational_v2: %v", err)
	}
	if result.Value.Lifecycle != domain.WorkItemLifecycleTodo || result.Value.CurrentConclusion != nil || len(result.Value.ConclusionHistory) != 1 {
		t.Fatalf("reopen state did not preserve legacy lifecycle/history semantics: %+v", result.Value)
	}
}

func TestReopenObjectivePreservesConclusionAndStartsNewPlannedCycle(t *testing.T) {
	ctx := context.Background()
	fixture := newOperationalGateFixture(t, domain.OperationalProfileOperationalV2, domain.ObjectiveLifecycleInProgress, domain.ObjectiveLifecycleInProgress)
	service := newOperationalReopenService(t, fixture)
	command := operationalReopenCommand(30)
	cancelled, err := service.CancelObjective(ctx, command, CancelObjectiveCommand{Scope: fixture.scope, ObjectiveID: fixture.child.ID, ExpectedVersion: fixture.child.Version, Reason: "pause this branch"})
	if err != nil {
		t.Fatal(err)
	}
	command.CommandID = operationalGateID(5031)
	reopened, err := service.ReopenObjective(ctx, command, ReopenObjectiveCommand{Scope: fixture.scope, ObjectiveID: fixture.child.ID, ExpectedVersion: cancelled.Value.Version, Reason: "resume planning"})
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Value.Lifecycle != domain.ObjectiveLifecyclePlanned || reopened.Value.CompletionCycle != 2 || reopened.Value.CurrentConclusion != nil || len(reopened.Value.ConclusionHistory) != 1 {
		t.Fatalf("Objective reopen did not preserve cycle/conclusion semantics: %+v", reopened.Value)
	}
	if reopened.Value.ConclusionHistory[0].ID != cancelled.Value.CurrentConclusion.ID {
		t.Fatal("reopen did not preserve the cancelled Conclusion in history")
	}
	if reopened.Value.ParentObjectiveID == nil || *reopened.Value.ParentObjectiveID != fixture.root.ID {
		t.Fatal("reopening an Objective changed its hierarchy")
	}
}

func setFixtureTaskState(t *testing.T, fixture operationalGateFixture, edit func(*domain.WorkItem)) domain.WorkItem {
	t.Helper()
	ctx := context.Background()
	uow, err := fixture.store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	item, err := uow.WorkItems().Get(ctx, fixture.scope, fixture.task.ID)
	if err != nil {
		t.Fatal(err)
	}
	expected := item.Version
	edit(&item)
	next, err := expected.Next()
	if err != nil {
		t.Fatal(err)
	}
	item.Version = next
	item.UpdatedAt = operationalGateNow
	if err := uow.WorkItems().Save(ctx, item, expected); err != nil {
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}
	return item
}

func startFixtureObjective(t *testing.T, fixture operationalGateFixture, objectiveID domain.ID) domain.Objective {
	t.Helper()
	ctx := context.Background()
	uow, err := fixture.store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	objective, err := uow.Objectives().Get(ctx, fixture.scope, objectiveID)
	if err != nil {
		t.Fatal(err)
	}
	expected := objective.Version
	if err := objective.Start(operationalGateNow); err != nil {
		t.Fatal(err)
	}
	if err := uow.Objectives().Save(ctx, objective, expected); err != nil {
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}
	return objective
}

func TestCancelObjectiveBlocksActiveDescendantAndAllowsExplicitLeaseRelease(t *testing.T) {
	ctx := context.Background()
	fixture := newOperationalGateFixture(t, domain.OperationalProfileOperationalV2, domain.ObjectiveLifecycleInProgress, domain.ObjectiveLifecyclePlanned)
	actor := operationalReopenCommand(40).Actor
	claimed := setFixtureTaskState(t, fixture, func(item *domain.WorkItem) {
		if err := item.Claim(operationalGateID(700), "operator", actor, domain.DefaultLeaseTTL, operationalGateNow); err != nil {
			t.Fatal(err)
		}
	})
	service := newOperationalReopenService(t, fixture)
	command := operationalReopenCommand(41)
	before := readOperationalReopenSnapshot(t, fixture)
	_, err := service.CancelObjective(ctx, command, CancelObjectiveCommand{Scope: fixture.scope, ObjectiveID: fixture.root.ID, ExpectedVersion: fixture.root.Version, Reason: "retire parent"})
	var typed *domain.Error
	if !errors.As(err, &typed) || typed.Details == nil || typed.Details.ReasonCode != "objective_descendant_active" {
		t.Fatalf("cancellation error=%v, want objective_descendant_active", err)
	}
	if len(typed.Details.RelatedRefs) != 1 || typed.Details.RelatedRefs[0] != claimed.Ref() || len(typed.Details.Remedies) == 0 || typed.Details.Remedies[0].ActionID != "ReleaseWorkItem" {
		t.Fatalf("cancellation did not point to active descendant and release remedy: %+v", typed.Details)
	}
	if after := readOperationalReopenSnapshot(t, fixture); !reflect.DeepEqual(before, after) {
		t.Fatal("blocked cancellation changed Task, Objective, revision, or events")
	}
	command.CommandID = operationalGateID(5042)
	released, err := service.ReleaseWorkItem(ctx, command, ReleaseWorkItemCommand{Scope: fixture.scope, WorkItemID: claimed.ID, ExpectedVersion: claimed.Version, ClaimID: claimed.CurrentLease.ClaimID, FencingToken: claimed.CurrentLease.FencingToken})
	if err != nil || released.Value.CurrentLease != nil || released.Value.Lifecycle != domain.WorkItemLifecycleTodo {
		t.Fatalf("explicit lease release failed while parent work was blocked: item=%+v err=%v", released.Value, err)
	}
	command.CommandID = operationalGateID(5043)
	cancelled, err := service.CancelObjective(ctx, command, CancelObjectiveCommand{Scope: fixture.scope, ObjectiveID: fixture.root.ID, ExpectedVersion: fixture.root.Version, Reason: "retire parent after release"})
	if err != nil || cancelled.Value.Lifecycle != domain.ObjectiveLifecycleCancelled {
		t.Fatalf("cancellation did not proceed after explicit release: objective=%+v err=%v", cancelled.Value, err)
	}
}

func TestCancelObjectiveExpiredLeaseIsNotAuthorityButInProgressTaskStillBlocks(t *testing.T) {
	ctx := context.Background()
	fixture := newOperationalGateFixture(t, domain.OperationalProfileOperationalV2, domain.ObjectiveLifecycleInProgress, domain.ObjectiveLifecyclePlanned)
	actor := operationalReopenCommand(45).Actor
	task := setFixtureTaskState(t, fixture, func(item *domain.WorkItem) {
		acquiredAt := operationalGateNow.Add(-time.Minute)
		if err := item.Claim(operationalGateID(701), "operator", actor, domain.MinLeaseTTL, acquiredAt); err != nil {
			t.Fatal(err)
		}
		if item.CurrentLease.ValidAt(operationalGateNow) {
			t.Fatal("test lease must be expired at cancellation time")
		}
	})
	service := newOperationalReopenService(t, fixture)
	before := readOperationalReopenSnapshot(t, fixture)
	_, err := service.CancelObjective(ctx, operationalReopenCommand(46), CancelObjectiveCommand{Scope: fixture.scope, ObjectiveID: fixture.root.ID, ExpectedVersion: fixture.root.Version, Reason: "retire parent"})
	var typed *domain.Error
	if !errors.As(err, &typed) || typed.Details == nil || typed.Details.ReasonCode != "objective_descendant_active" {
		t.Fatalf("cancellation error=%v, want objective_descendant_active", err)
	}
	if len(typed.Details.RelatedRefs) != 1 || typed.Details.RelatedRefs[0] != task.Ref() {
		t.Fatalf("cancellation did not identify the still-in-progress Task: %+v", typed.Details.RelatedRefs)
	}
	if len(typed.Details.Remedies) != 1 || typed.Details.Remedies[0].ActionID != "AdministrativeCancelWorkItem" {
		t.Fatalf("expired lease was incorrectly treated as active authority or releasable lease: %+v", typed.Details.Remedies)
	}
	if after := readOperationalReopenSnapshot(t, fixture); !reflect.DeepEqual(before, after) {
		t.Fatal("blocked cancellation changed the still-active Task, Objective, revision, or events")
	}
}

func TestCancelObjectiveBlocksActiveNestedObjectiveContractAndReview(t *testing.T) {
	tests := []struct {
		name string
		edit func(*domain.WorkItem)
		want domain.EntityKind
	}{
		{name: "in-progress nested Objective", want: domain.EntityKindObjective},
		{name: "nested active contract", want: domain.EntityKindWorkItem, edit: func(item *domain.WorkItem) {
			item.ContractsEnabled = true
			id := operationalGateID(710)
			item.CurrentContractID = &id
			item.Lifecycle = domain.WorkItemLifecycleInProgress
		}},
		{name: "nested pending review", want: domain.EntityKindWorkItem, edit: func(item *domain.WorkItem) {
			item.ContractsEnabled = true
			id := operationalGateID(711)
			item.PendingReviewCaseID = &id
			item.Lifecycle = domain.WorkItemLifecycleInProgress
		}},
	}
	for i, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newOperationalGateFixture(t, domain.OperationalProfileOperationalV2, domain.ObjectiveLifecycleInProgress, domain.ObjectiveLifecyclePlanned)
			var activeRef domain.EntityRef
			if test.edit == nil {
				activeRef = startFixtureObjective(t, fixture, fixture.child.ID).Ref()
			} else {
				activeRef = setFixtureTaskState(t, fixture, test.edit).Ref()
			}
			service := newOperationalReopenService(t, fixture)
			before := readOperationalReopenSnapshot(t, fixture)
			_, err := service.CancelObjective(context.Background(), operationalReopenCommand(50+i), CancelObjectiveCommand{Scope: fixture.scope, ObjectiveID: fixture.root.ID, ExpectedVersion: fixture.root.Version, Reason: "retire parent"})
			var typed *domain.Error
			if !errors.As(err, &typed) || typed.Details == nil || typed.Details.ReasonCode != "objective_descendant_active" {
				t.Fatalf("cancellation error=%v, want objective_descendant_active", err)
			}
			found := false
			for _, ref := range typed.Details.RelatedRefs {
				if ref == activeRef && ref.Kind == test.want {
					found = true
				}
			}
			if !found {
				t.Fatalf("cancellation omitted nested active ref %s: %+v", activeRef, typed.Details.RelatedRefs)
			}
			if after := readOperationalReopenSnapshot(t, fixture); !reflect.DeepEqual(before, after) {
				t.Fatal("blocked cancellation changed hierarchy, work, revision, or events")
			}
		})
	}
}

func TestCancelObjectiveFailsClosedWhenStructuralSnapshotIsTooLarge(t *testing.T) {
	ctx := context.Background()
	fixture := newOperationalGateFixture(t, domain.OperationalProfileOperationalV2, domain.ObjectiveLifecycleInProgress, domain.ObjectiveLifecyclePlanned)
	before := readOperationalReopenSnapshot(t, fixture)
	service := newOperationalReopenService(t, fixture)
	service.tx = overflowStructureTransactionManager{TransactionManager: fixture.store}
	_, err := service.CancelObjective(ctx, operationalReopenCommand(60), CancelObjectiveCommand{Scope: fixture.scope, ObjectiveID: fixture.root.ID, ExpectedVersion: fixture.root.Version, Reason: "retire parent"})
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeGraphLimitExceeded {
		t.Fatalf("large structural read error=%v, want graph limit", err)
	}
	if after := readOperationalReopenSnapshot(t, fixture); !reflect.DeepEqual(before, after) {
		t.Fatal("failed bounded cancellation changed state, revision, or events")
	}
}

func TestConcurrentObjectiveCancellationAndTaskAcquisitionCannotOrphanLease(t *testing.T) {
	fixture := newOperationalGateFixture(t, domain.OperationalProfileOperationalV2, domain.ObjectiveLifecycleInProgress, domain.ObjectiveLifecycleInProgress)
	service := newOperationalReopenService(t, fixture)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	wg.Add(2)
	go func() {
		defer wg.Done()
		_, err := service.CancelObjective(context.Background(), operationalReopenCommand(70), CancelObjectiveCommand{Scope: fixture.scope, ObjectiveID: fixture.root.ID, ExpectedVersion: fixture.root.Version, Reason: "retire branch"})
		results <- err
	}()
	go func() {
		defer wg.Done()
		_, err := service.ClaimWorkItem(context.Background(), operationalReopenCommand(71), ClaimWorkItemCommand{Scope: fixture.scope, WorkItemID: fixture.task.ID, ExpectedVersion: fixture.task.Version, TTL: domain.DefaultLeaseTTL})
		results <- err
	}()
	wg.Wait()
	close(results)
	succeeded := 0
	for err := range results {
		if err == nil {
			succeeded++
		}
	}
	if succeeded != 1 {
		t.Fatalf("concurrent cancellation/acquisition successes=%d, want exactly one", succeeded)
	}
	after := readOperationalReopenSnapshot(t, fixture)
	if after.root.Lifecycle == domain.ObjectiveLifecycleCancelled && after.task.CurrentLease != nil {
		t.Fatal("Objective cancellation committed while a descendant lease was acquired")
	}
	if after.task.CurrentLease != nil && after.root.Lifecycle != domain.ObjectiveLifecycleInProgress {
		t.Fatal("Task acquired a lease after its ancestor Objective was cancelled")
	}
}
