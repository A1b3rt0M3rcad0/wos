package application_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func editingObjective(t *testing.T, service *a.Service, scope d.Scope, title string, parent *d.ID) d.Objective {
	t.Helper()
	result, err := service.CreateObjective(context.Background(), commandContext(), a.CreateObjectiveCommand{Scope: scope, Title: title, Priority: d.PriorityNormal, ParentObjectiveID: parent})
	if err != nil {
		t.Fatal(err)
	}
	return result.Value
}

func requireEditingReason(t *testing.T, err error, code d.ErrorCode, reason string) {
	t.Helper()
	var actual *d.Error
	if !errors.As(err, &actual) || actual.Code != code || actual.Details == nil || actual.Details.ReasonCode != reason {
		t.Fatalf("expected %s/%s, got %#v", code, reason, err)
	}
}

func TestProgressiveWorkDetailsPersistReplaceClearAndReplay(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	outcome := setupActiveOutcome(t, service)
	objective := editingObjective(t, service, outcome.Scope(), "Condition", nil)
	created, err := service.CreateWorkItem(ctx, commandContext(), a.CreateWorkItemCommand{Scope: outcome.Scope(), Title: "Implement result", Priority: d.PriorityNormal, ObjectiveID: &objective.ID})
	if err != nil {
		t.Fatal(err)
	}
	work := created.Value
	cc := commandContext()
	cc.IdempotencyKey = "progressive-edit-spec"
	cmd := a.SetWorkItemExecutionSpecCommand{Scope: work.Scope, WorkItemID: work.ID, ExpectedVersion: work.Version, ExecutionSpec: &d.ExecutionSpec{Instructions: []string{"Implement and report findings"}, Constraints: []string{"Preserve authorization"}, Deliverables: []string{"Reviewable patch"}, ContextRefs: []d.EntityRef{objective.Ref()}}}
	updated, err := service.SetWorkItemExecutionSpec(ctx, cc, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Value.ID != work.ID || updated.Value.Version != work.Version+1 || updated.Value.Lifecycle != d.WorkItemLifecycleBacklog || updated.Value.CurrentLease != nil {
		t.Fatal("editing changed Task identity or granted execution")
	}
	work = updated.Value
	beforeEvents := len(store.SnapshotDomainEvents(work.Scope))
	replayed, err := service.SetWorkItemExecutionSpec(ctx, cc, cmd)
	if err != nil || !replayed.IdempotentReplay || replayed.Value.Version != work.Version || len(store.SnapshotDomainEvents(work.Scope)) != beforeEvents {
		t.Fatal("spec edit replay duplicated state or events", err)
	}
	cc.IdempotencyKey = "progressive-edit-noop"
	cmd.ExpectedVersion = work.Version
	noop, err := service.SetWorkItemExecutionSpec(ctx, cc, cmd)
	if err != nil || noop.Value.Version != work.Version || noop.OutcomeRevision != updated.OutcomeRevision || len(store.SnapshotDomainEvents(work.Scope)) != beforeEvents {
		t.Fatal("equal replacement advanced revision or emitted an event", err)
	}
	when := time.Date(2026, 11, 1, 9, 0, 0, 0, time.FixedZone("client", -10800))
	scheduled, err := service.SetWorkItemNotBefore(ctx, commandContext(), a.SetWorkItemNotBeforeCommand{Scope: work.Scope, WorkItemID: work.ID, ExpectedVersion: work.Version, NotBefore: &when})
	if err != nil || scheduled.Value.NotBefore == nil || !scheduled.Value.NotBefore.Equal(when) || scheduled.Value.CurrentLease != nil {
		t.Fatal("schedule did not persist without execution", err)
	}
	cleared, err := service.SetWorkItemNotBefore(ctx, commandContext(), a.SetWorkItemNotBeforeCommand{Scope: work.Scope, WorkItemID: work.ID, ExpectedVersion: scheduled.Value.Version, NotBefore: nil})
	if err != nil || cleared.Value.NotBefore != nil {
		t.Fatal("null scheduling did not clear", err)
	}
	clearedSpec, err := service.SetWorkItemExecutionSpec(ctx, commandContext(), a.SetWorkItemExecutionSpecCommand{Scope: work.Scope, WorkItemID: work.ID, ExpectedVersion: cleared.Value.Version, ExecutionSpec: nil})
	if err != nil || clearedSpec.Value.ExecutionSpec != nil {
		t.Fatal("null specification did not clear", err)
	}
	loaded, err := service.GetWorkItem(ctx, work.Scope, work.ID)
	if err != nil || loaded.Value.Version != clearedSpec.Value.Version || loaded.Value.ExecutionSpec != nil || loaded.Value.NotBefore != nil || loaded.Value.ObjectiveID == nil || *loaded.Value.ObjectiveID != objective.ID {
		t.Fatal("new client did not see persisted edits", err)
	}
	foreign := objective.Ref()
	foreign.Scope.OutcomeID = id("0199e300-0000-7000-8000-000000000099")
	_, err = service.SetWorkItemExecutionSpec(ctx, commandContext(), a.SetWorkItemExecutionSpecCommand{Scope: work.Scope, WorkItemID: work.ID, ExpectedVersion: loaded.Value.Version, ExecutionSpec: &d.ExecutionSpec{ContextRefs: []d.EntityRef{foreign}}})
	if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodeInvalidScope {
		t.Fatal("foreign context reference accepted", err)
	}
}

func TestWorkObjectiveMovePreservesRecordsAndRejectsOwnHardDependency(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	outcome := setupActiveOutcome(t, service)
	first := editingObjective(t, service, outcome.Scope(), "First", nil)
	second := editingObjective(t, service, outcome.Scope(), "Second", nil)
	created, err := service.CreateWorkItem(ctx, commandContext(), a.CreateWorkItemCommand{Scope: outcome.Scope(), Title: "Work", Priority: d.PriorityNormal, ObjectiveID: &first.ID})
	if err != nil {
		t.Fatal(err)
	}
	work := created.Value
	criterion, err := service.AddCriterion(ctx, commandContext(), a.AddCriterionCommand{Owner: work.Ref(), ExpectedVersion: work.Version, Title: "Preserved criterion", Required: true, VerificationMode: d.VerificationModeAttestation})
	if err != nil {
		t.Fatal(err)
	}
	work.Version++
	_, err = service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{Scope: work.Scope, SourceRef: work.Ref(), TargetRef: first.Ref(), Strength: d.DependencyStrengthHard})
	requireEditingReason(t, err, d.ErrorCodePreconditionFailed, "own_objective_hard_dependency")
	advisory, err := service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{Scope: work.Scope, SourceRef: work.Ref(), TargetRef: first.Ref(), Strength: d.DependencyStrengthAdvisory})
	if err != nil {
		t.Fatal("advisory association was incorrectly rejected", err)
	}
	dependency, err := service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{Scope: work.Scope, SourceRef: work.Ref(), TargetRef: second.Ref(), Strength: d.DependencyStrengthHard})
	if err != nil {
		t.Fatal(err)
	}
	before := len(store.SnapshotDomainEvents(work.Scope))
	_, err = service.SetWorkItemObjective(ctx, commandContext(), a.SetWorkItemObjectiveCommand{Scope: work.Scope, WorkItemID: work.ID, ExpectedVersion: work.Version, ObjectiveID: &second.ID})
	requireEditingReason(t, err, d.ErrorCodePreconditionFailed, "own_objective_hard_dependency")
	if len(store.SnapshotDomainEvents(work.Scope)) != before {
		t.Fatal("rejected move emitted events")
	}
	if _, err := service.RemoveDependency(ctx, commandContext(), a.RemoveDependencyCommand{Scope: work.Scope, RelationID: dependency.Value.ID, ExpectedVersion: dependency.Value.Version, Reason: "Use Objective belonging instead"}); err != nil {
		t.Fatal(err)
	}
	moved, err := service.SetWorkItemObjective(ctx, commandContext(), a.SetWorkItemObjectiveCommand{Scope: work.Scope, WorkItemID: work.ID, ExpectedVersion: work.Version, ObjectiveID: &second.ID})
	if err != nil || moved.Value.ID != work.ID || moved.Value.ObjectiveID == nil || *moved.Value.ObjectiveID != second.ID || len(moved.Value.Criteria.Items) != 1 || moved.Value.Criteria.Items[0].ID != criterion.Value.ID {
		t.Fatal("move lost identity or criteria", err)
	}
	eventsBeforeUnparent := len(store.SnapshotDomainEvents(work.Scope))
	_, err = service.SetWorkItemObjective(ctx, commandContext(), a.SetWorkItemObjectiveCommand{Scope: work.Scope, WorkItemID: work.ID, ExpectedVersion: moved.Value.Version, ObjectiveID: nil})
	requireEditingReason(t, err, d.ErrorCodePreconditionFailed, "task_objective_required")
	unchanged, err := service.GetWorkItem(ctx, work.Scope, work.ID)
	if err != nil || unchanged.Value.Version != moved.Value.Version || unchanged.Value.ObjectiveID == nil || *unchanged.Value.ObjectiveID != second.ID || len(unchanged.Value.Criteria.Items) != 1 || unchanged.Value.Criteria.Items[0].ID != criterion.Value.ID || len(store.SnapshotDomainEvents(work.Scope)) != eventsBeforeUnparent {
		t.Fatal("rejected unparenting changed Task, criteria, version or history", err)
	}
	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if relation, err := tx.Relations().Get(ctx, work.Scope, advisory.Value.ID); err != nil || relation.Lifecycle != d.RelationLifecycleActive {
		t.Fatal("moving Task rewrote an existing dependency", err)
	}
}

func TestObjectiveReparentRejectsCyclesAndFrozenDescendants(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	root := editingObjective(t, service, outcome.Scope(), "Root", nil)
	child := editingObjective(t, service, outcome.Scope(), "Child", &root.ID)
	destination := editingObjective(t, service, outcome.Scope(), "Destination", nil)
	_, err := service.SetObjectiveParent(ctx, commandContext(), a.SetObjectiveParentCommand{Scope: root.Scope, ObjectiveID: root.ID, ExpectedVersion: root.Version, ParentObjectiveID: &child.ID})
	requireEditingReason(t, err, d.ErrorCodeDependencyCycle, "objective_hierarchy_cycle")
	work, err := service.CreateWorkItem(ctx, commandContext(), a.CreateWorkItemCommand{Scope: root.Scope, Title: "Descendant work", Priority: d.PriorityNormal, Lifecycle: d.WorkItemLifecycleTodo, ObjectiveID: &child.ID})
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := service.ClaimWorkItem(ctx, commandContext(), a.ClaimWorkItemCommand{Scope: root.Scope, WorkItemID: work.Value.ID, ExpectedVersion: work.Value.Version, TTL: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.SetObjectiveParent(ctx, commandContext(), a.SetObjectiveParentCommand{Scope: root.Scope, ObjectiveID: root.ID, ExpectedVersion: root.Version, ParentObjectiveID: &destination.ID})
	requireEditingReason(t, err, d.ErrorCodeWorkAlreadyClaimed, "active_lease")
	lease := claimed.Value.CurrentLease
	released, err := service.ReleaseWorkItem(ctx, commandContext(), a.ReleaseWorkItemCommand{Scope: root.Scope, WorkItemID: work.Value.ID, ExpectedVersion: claimed.Value.Version, ClaimID: lease.ClaimID, FencingToken: lease.FencingToken})
	if err != nil {
		t.Fatal(err)
	}
	moved, err := service.SetObjectiveParent(ctx, commandContext(), a.SetObjectiveParentCommand{Scope: root.Scope, ObjectiveID: root.ID, ExpectedVersion: root.Version, ParentObjectiveID: &destination.ID})
	if err != nil || moved.Value.ParentObjectiveID == nil || *moved.Value.ParentObjectiveID != destination.ID {
		t.Fatal("planned branch did not move after explicit authority release", err)
	}
	loaded, err := service.GetWorkItem(ctx, root.Scope, work.Value.ID)
	if err != nil || loaded.Value.Version != released.Value.Version || *loaded.Value.ObjectiveID != child.ID {
		t.Fatal("Objective move rewrote descendant work", err)
	}
}

func TestDefinitionSettersDoNotRevokeActiveOrExpiredContracts(t *testing.T) {
	ctx := context.Background()
	service, _, clock, work := contractServiceFixture(t)
	acquired, err := service.AcquireWorkContract(ctx, commandContext(), a.AcquireWorkContractCommand{Scope: work.Scope, WorkItemID: work.ID, ExpectedWorkItemVersion: work.Version, TTLSeconds: 30})
	if err != nil {
		t.Fatal(err)
	}
	cmd := a.SetWorkItemExecutionSpecCommand{Scope: work.Scope, WorkItemID: work.ID, ExpectedVersion: acquired.Value.WorkItem.Version, ExecutionSpec: &d.ExecutionSpec{Instructions: []string{"Changed work"}}}
	_, err = service.SetWorkItemExecutionSpec(ctx, commandContext(), cmd)
	requireEditingReason(t, err, d.ErrorCodePreconditionFailed, "work_item_planning_frozen")
	clock.set(acquired.Value.Contract.ExpiresAt.Add(time.Second))
	_, err = service.SetWorkItemExecutionSpec(ctx, commandContext(), cmd)
	requireEditingReason(t, err, d.ErrorCodePreconditionFailed, "work_item_planning_frozen")
	loaded, err := service.GetWorkItem(ctx, work.Scope, work.ID)
	if err != nil || loaded.Value.CurrentContractID == nil || *loaded.Value.CurrentContractID != acquired.Value.Contract.ID || loaded.Value.Version != cmd.ExpectedVersion {
		t.Fatal("definition edit implicitly revoked or reconciled a contract", err)
	}
}

func TestConcurrentOppositeObjectiveMovesCannotCreateCycle(t *testing.T) {
	ctx := context.Background()
	service, _, _, work := contractServiceFixture(t)
	first := editingObjective(t, service, work.Scope, "First", nil)
	second := editingObjective(t, service, work.Scope, "Second", nil)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	for i, pair := range [][2]d.Objective{{first, second}, {second, first}} {
		wg.Add(1)
		go func(i int, pair [2]d.Objective) {
			defer wg.Done()
			cc := commandContext()
			if i == 0 {
				cc.IdempotencyKey = "move-first-under-second"
			} else {
				cc.IdempotencyKey = "move-second-under-first"
			}
			_, err := service.SetObjectiveParent(ctx, cc, a.SetObjectiveParentCommand{Scope: work.Scope, ObjectiveID: pair[0].ID, ExpectedVersion: pair[0].Version, ParentObjectiveID: &pair[1].ID})
			results <- err
		}(i, pair)
	}
	wg.Wait()
	close(results)
	winners := 0
	for err := range results {
		if err == nil {
			winners++
		}
	}
	if winners != 1 {
		t.Fatalf("expected one structural move, got %d", winners)
	}
}

func TestParticipationNoOpStillChecksObservedVersion(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	objective := editingObjective(t, service, outcome.Scope(), "Condition", nil)
	work := createTodoWork(t, service, outcome.Scope(), "Task")
	for _, err := range []error{
		func() error {
			_, err := service.SetOutcomeOwners(ctx, commandContext(), a.SetOutcomeOwnersCommand{Scope: outcome.Scope(), ExpectedVersion: outcome.Version - 1, OwnerRefs: []d.ActorRef{}})
			return err
		}(),
		func() error {
			_, err := service.SetObjectiveOwners(ctx, commandContext(), a.SetObjectiveOwnersCommand{Scope: objective.Scope, ObjectiveID: objective.ID, ExpectedVersion: objective.Version + 1, OwnerRefs: []d.ActorRef{}})
			return err
		}(),
		func() error {
			_, err := service.SetWorkItemAssignees(ctx, commandContext(), a.SetWorkItemAssigneesCommand{Scope: work.Scope, WorkItemID: work.ID, ExpectedVersion: work.Version + 1, AssigneeRefs: []d.ActorRef{}})
			return err
		}(),
	} {
		if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodeVersionConflict {
			t.Fatal("unchanged participation bypassed expected_version", err)
		}
	}
}

func TestStructureMovementRejectsStaleImpactPreviewWithoutChangingSource(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	first := editingObjective(t, service, outcome.Scope(), "First", nil)
	second := editingObjective(t, service, outcome.Scope(), "Second", nil)
	work := deferPlanningWork(t, service, createTodoWork(t, service, outcome.Scope(), "Task"))
	observed, err := service.GetWorkItem(ctx, work.Scope, work.ID)
	if err != nil {
		t.Fatal(err)
	}
	// An unrelated domain write advances the coherent impact snapshot while
	// leaving the Task/Objective versions unchanged.
	_ = editingObjective(t, service, work.Scope, "Concurrent branch", nil)
	_, err = service.SetWorkItemObjective(ctx, commandContext(), a.SetWorkItemObjectiveCommand{Scope: work.Scope, WorkItemID: work.ID, ExpectedVersion: work.Version, ExpectedOutcomeRevision: &observed.OutcomeRevision, ObjectiveID: &first.ID})
	requireEditingReason(t, err, d.ErrorCodePreconditionFailed, "structure_preview_changed")
	_, err = service.SetObjectiveParent(ctx, commandContext(), a.SetObjectiveParentCommand{Scope: first.Scope, ObjectiveID: first.ID, ExpectedVersion: first.Version, ExpectedOutcomeRevision: &observed.OutcomeRevision, ParentObjectiveID: &second.ID})
	requireEditingReason(t, err, d.ErrorCodePreconditionFailed, "structure_preview_changed")
	loaded, err := service.GetWorkItem(ctx, work.Scope, work.ID)
	if err != nil || loaded.Value.Version != work.Version || loaded.Value.ObjectiveID == nil || *loaded.Value.ObjectiveID != *work.ObjectiveID {
		t.Fatal("stale preview changed Task placement", err)
	}
}
