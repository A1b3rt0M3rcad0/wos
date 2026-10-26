package application_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

type hierarchyDeadlockIDs struct {
	mu   sync.Mutex
	next uint64
}

func (g *hierarchyDeadlockIDs) NewID() (d.ID, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.next++
	return d.ParseID(fmt.Sprintf("0199e310-0000-7000-8000-%012x", g.next))
}

func hierarchyDeadlockService(t *testing.T) (*a.Service, *memory.Store) {
	t.Helper()
	store := memory.New()
	service, err := a.NewService(store, fixedClock{now: time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)}, &hierarchyDeadlockIDs{})
	if err != nil {
		t.Fatal(err)
	}
	return service, store
}

func hierarchyDeadlockTask(t *testing.T, service *a.Service, scope d.Scope, owner d.ID, title string) d.WorkItem {
	t.Helper()
	created, err := service.CreateWorkItem(context.Background(), commandContext(), a.CreateWorkItemCommand{
		Scope: scope, ObjectiveID: &owner, Title: title, Priority: d.PriorityNormal, Lifecycle: d.WorkItemLifecycleBacklog,
	})
	if err != nil {
		t.Fatal(err)
	}
	return created.Value
}

func setHierarchyDeadlockProfile(t *testing.T, store *memory.Store, scope d.Scope, profile d.OperationalProfile) {
	t.Helper()
	ctx := context.Background()
	uow, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		_ = uow.Rollback()
		t.Fatal(err)
	}
	outcome.OperationalProfile = profile
	expectedVersion := outcome.Version
	outcome.Version++
	if err := uow.Outcomes().Save(ctx, outcome, expectedVersion); err != nil {
		_ = uow.Rollback()
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}
}

func insertLegacyOrphanTask(t *testing.T, store *memory.Store, scope d.Scope, suffix string, done bool) d.WorkItem {
	t.Helper()
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	item, err := d.NewWorkItem(d.MustParseID("0199e3ff-0000-7000-8000-"+suffix), scope, "Legacy orphan", "", d.PriorityNormal, d.WorkItemLifecycleBacklog, now)
	if err != nil {
		t.Fatal(err)
	}
	if done {
		item.Lifecycle = d.WorkItemLifecycleDone
		item.ResultSummary = "Completed before Objective ownership was introduced."
		conclusion := d.Conclusion{
			PrincipalID: "legacy-operator",
			Actor:       d.ActorRef{Kind: d.ActorKindHuman, Provider: "local", ID: "legacy-operator"},
			Reason:      "Preserve a historical completed Task",
			ConcludedAt: now,
		}
		if err := conclusion.Bind(item.Ref(), item.Version, string(item.Lifecycle), d.ConclusionObligations{ResultSummaryRequired: true}); err != nil {
			t.Fatal(err)
		}
		item.CurrentConclusion = &conclusion
	}
	uow, err := store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if err := uow.WorkItems().Insert(context.Background(), item); err != nil {
		_ = uow.Rollback()
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}
	return item
}

func requireHierarchyDeadlock(t *testing.T, err error, field string, path ...d.EntityRef) *d.Error {
	t.Helper()
	var typed *d.Error
	if !errors.As(err, &typed) || typed.Code != d.ErrorCodePreconditionFailed || typed.Details == nil || typed.Details.ReasonCode != "dependency_hierarchy_deadlock" {
		t.Fatalf("expected dependency hierarchy deadlock, got %v", err)
	}
	if len(typed.Details.FieldPaths) != 1 || typed.Details.FieldPaths[0] != field {
		t.Fatalf("structured field path = %v, want %q", typed.Details.FieldPaths, field)
	}
	for _, ref := range path {
		found := false
		for _, actual := range typed.Details.RelatedRefs {
			if actual == ref {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("structured path %v omits %v", typed.Details.RelatedRefs, ref)
		}
	}
	if len(typed.Details.Remedies) != 0 {
		t.Fatalf("deadlock error must not attach remedies to an unpersisted dependency: %+v", typed.Details.Remedies)
	}
	return typed
}

func TestHierarchyDependencyDeadlocksAndAdvisoryRelationships(t *testing.T) {
	ctx := context.Background()
	service, store := hierarchyDeadlockService(t)
	outcome := setupActiveOutcome(t, service)
	setHierarchyDeadlockProfile(t, store, outcome.Scope(), d.OperationalProfileOperationalV2)
	root := editingObjective(t, service, outcome.Scope(), "Root", nil)
	child := editingObjective(t, service, outcome.Scope(), "Child", &root.ID)
	task := hierarchyDeadlockTask(t, service, outcome.Scope(), child.ID, "Child work")

	_, err := service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: task.Ref(), TargetRef: child.Ref(), Strength: d.DependencyStrengthHard,
	})
	requireEditingReason(t, err, d.ErrorCodePreconditionFailed, "own_objective_hard_dependency")

	_, err = service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: task.Ref(), TargetRef: root.Ref(), Strength: d.DependencyStrengthHard,
	})
	requireHierarchyDeadlock(t, err, "target_ref", task.Ref(), child.Ref(), root.Ref())

	if _, err = service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: task.Ref(), TargetRef: root.Ref(), Strength: d.DependencyStrengthAdvisory,
	}); err != nil {
		t.Fatalf("advisory ancestor relationship should remain informational: %v", err)
	}
	_, err = service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: child.Ref(), TargetRef: root.Ref(), Strength: d.DependencyStrengthHard,
	})
	requireHierarchyDeadlock(t, err, "target_ref", child.Ref(), root.Ref())
	if _, err = service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: child.Ref(), TargetRef: root.Ref(), Strength: d.DependencyStrengthAdvisory,
	}); err != nil {
		t.Fatalf("advisory Objective ancestry relationship should remain informational: %v", err)
	}

	secondTask := hierarchyDeadlockTask(t, service, outcome.Scope(), child.ID, "Second child work")
	_, err = service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: root.Ref(), TargetRef: secondTask.Ref(), Strength: d.DependencyStrengthHard,
	})
	requireHierarchyDeadlock(t, err, "target_ref", root.Ref(), child.Ref(), secondTask.Ref())
	if _, err = service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: root.Ref(), TargetRef: secondTask.Ref(), Strength: d.DependencyStrengthAdvisory,
	}); err != nil {
		t.Fatalf("advisory descendant relationship should remain informational: %v", err)
	}
}

func TestHierarchyDeadlockValidationUsesFinalDependencyBatch(t *testing.T) {
	ctx := context.Background()
	service, store := hierarchyDeadlockService(t)
	outcome := setupActiveOutcome(t, service)
	setHierarchyDeadlockProfile(t, store, outcome.Scope(), d.OperationalProfileOperationalV2)
	root := editingObjective(t, service, outcome.Scope(), "Root", nil)
	child := editingObjective(t, service, outcome.Scope(), "Child", &root.ID)
	task := hierarchyDeadlockTask(t, service, outcome.Scope(), child.ID, "Child work")
	old, err := service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: root.Ref(), TargetRef: task.Ref(), Strength: d.DependencyStrengthAdvisory,
	})
	if err != nil {
		t.Fatal(err)
	}
	current, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	cmd := a.ApplyDependencyChangesCommand{
		Scope: outcome.Scope(), ExpectedOutcomeRevision: current.OutcomeRevision,
		ExpectedSources: []a.DependencySourceVersion{{SourceRef: root.Ref(), ExpectedVersion: root.Version}},
		Changes: []a.DependencyChange{
			{Action: a.DependencyChangeAdd, SourceRef: dependencyRefPointer(root.Ref()), TargetRef: dependencyRefPointer(task.Ref()), Strength: d.DependencyStrengthHard},
			{Action: a.DependencyChangeRemove, RelationID: idPointer(old.Value.ID), ExpectedVersion: versionPointer(old.Value.Version), Reason: "replace informational link with a gate"},
		},
	}
	eventsBefore := len(store.SnapshotDomainEvents(outcome.Scope()))
	preview, err := service.PreviewDependencyChanges(ctx, outcome.Scope(), a.DependencyChangesPreviewQuery{
		ExpectedOutcomeRevision: cmd.ExpectedOutcomeRevision, ExpectedSources: cmd.ExpectedSources, Changes: cmd.Changes,
	})
	if err != nil || preview.Availability != "blocked" || preview.Error == nil || preview.Error.Details == nil || preview.Error.Details.ReasonCode != "dependency_hierarchy_deadlock" {
		t.Fatalf("hierarchy batch preview = %+v, err=%v", preview, err)
	}
	_, err = service.ApplyDependencyChanges(ctx, commandContext(), cmd)
	requireHierarchyDeadlock(t, err, "target_ref", root.Ref(), child.Ref(), task.Ref())
	after, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil || after.OutcomeRevision != current.OutcomeRevision || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsBefore {
		t.Fatalf("rejected final batch changed outcome: revision=%d err=%v", after.OutcomeRevision, err)
	}
	relations := dependencyRelations(t, store, outcome.Scope())
	if len(relations) != 1 || relations[0].ID != old.Value.ID || relations[0].Lifecycle != d.RelationLifecycleActive || relations[0].Strength != d.DependencyStrengthAdvisory {
		t.Fatalf("rejected final batch partially changed dependencies: %+v", relations)
	}
}

func TestHardDependenciesFailClosedForUnownedLegacyTasks(t *testing.T) {
	ctx := context.Background()
	service, store := hierarchyDeadlockService(t)
	outcome := setupActiveOutcome(t, service)
	setHierarchyDeadlockProfile(t, store, outcome.Scope(), d.OperationalProfileOperationalV2)
	objective := editingObjective(t, service, outcome.Scope(), "Objective", nil)
	pendingOrphan := insertLegacyOrphanTask(t, store, outcome.Scope(), "000000000001", false)
	_, err := service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: objective.Ref(), TargetRef: pendingOrphan.Ref(), Strength: d.DependencyStrengthHard,
	})
	requireEditingReason(t, err, d.ErrorCodePreconditionFailed, "dependency_hierarchy_incomplete")
	_, err = service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: pendingOrphan.Ref(), TargetRef: objective.Ref(), Strength: d.DependencyStrengthHard,
	})
	requireEditingReason(t, err, d.ErrorCodePreconditionFailed, "dependency_hierarchy_incomplete")

	completedOrphan := insertLegacyOrphanTask(t, store, outcome.Scope(), "000000000002", true)
	if _, err := service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: objective.Ref(), TargetRef: completedOrphan.Ref(), Strength: d.DependencyStrengthHard,
	}); err != nil {
		t.Fatalf("already-satisfied dependency to a historical unowned Task is unambiguous: %v", err)
	}
}

func TestTaskMoveCannotTurnDependencyTargetIntoAncestor(t *testing.T) {
	ctx := context.Background()
	service, store := hierarchyDeadlockService(t)
	outcome := setupActiveOutcome(t, service)
	setHierarchyDeadlockProfile(t, store, outcome.Scope(), d.OperationalProfileOperationalV2)
	root := editingObjective(t, service, outcome.Scope(), "Root", nil)
	child := editingObjective(t, service, outcome.Scope(), "Child", &root.ID)
	other := editingObjective(t, service, outcome.Scope(), "Other", nil)
	task := hierarchyDeadlockTask(t, service, outcome.Scope(), other.ID, "Move into branch")
	if _, err := service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: task.Ref(), TargetRef: root.Ref(), Strength: d.DependencyStrengthHard,
	}); err != nil {
		t.Fatalf("dependency should be valid before the move: %v", err)
	}
	previewContext := a.WithIdentity(ctx, a.Identity{NamespaceID: outcome.Scope().NamespaceID, PrincipalID: "human-1", Actor: commandContext().Actor})
	preview, err := service.GetEntityWorkspaceView(previewContext, outcome.Scope(), a.WorkspaceViewQuery{
		Subject: a.WorkspaceSubject{Kind: "entity", Ref: task.Ref()}, PlacementPreview: &a.PlacementPreviewQuery{ParentID: &child.ID},
	})
	if err != nil || preview.PlacementPreview == nil || preview.PlacementPreview.Availability != "blocked" || preview.PlacementPreview.Error == nil || preview.PlacementPreview.Error.Details == nil || preview.PlacementPreview.Error.Details.ReasonCode != "dependency_hierarchy_deadlock" {
		t.Fatalf("Task placement preview did not explain the hierarchy deadlock: preview=%+v err=%v", preview.PlacementPreview, err)
	}
	_, err = service.SetWorkItemObjective(ctx, commandContext(), a.SetWorkItemObjectiveCommand{
		Scope: outcome.Scope(), WorkItemID: task.ID, ExpectedVersion: task.Version, ObjectiveID: &child.ID,
	})
	requireHierarchyDeadlock(t, err, "objective_id", task.Ref(), child.Ref(), root.Ref())
	loaded, err := service.GetWorkItem(ctx, task.Scope, task.ID)
	if err != nil || loaded.Value.ObjectiveID == nil || *loaded.Value.ObjectiveID != other.ID || loaded.Value.Version != task.Version {
		t.Fatalf("rejected move changed Task ownership: task=%+v err=%v", loaded.Value, err)
	}
}

func TestObjectiveReparentCannotCreateAncestorTaskWait(t *testing.T) {
	ctx := context.Background()
	service, store := hierarchyDeadlockService(t)
	outcome := setupActiveOutcome(t, service)
	setHierarchyDeadlockProfile(t, store, outcome.Scope(), d.OperationalProfileOperationalV2)
	root := editingObjective(t, service, outcome.Scope(), "Root", nil)
	branch := editingObjective(t, service, outcome.Scope(), "Branch", nil)
	task := hierarchyDeadlockTask(t, service, outcome.Scope(), branch.ID, "Branch work")
	if _, err := service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: task.Ref(), TargetRef: root.Ref(), Strength: d.DependencyStrengthHard,
	}); err != nil {
		t.Fatalf("dependency should be valid before reparenting: %v", err)
	}
	previewContext := a.WithIdentity(ctx, a.Identity{NamespaceID: outcome.Scope().NamespaceID, PrincipalID: "human-1", Actor: commandContext().Actor})
	preview, err := service.GetEntityWorkspaceView(previewContext, outcome.Scope(), a.WorkspaceViewQuery{
		Subject: a.WorkspaceSubject{Kind: "entity", Ref: branch.Ref()}, PlacementPreview: &a.PlacementPreviewQuery{ParentID: &root.ID},
	})
	if err != nil || preview.PlacementPreview == nil || preview.PlacementPreview.Availability != "blocked" || preview.PlacementPreview.Error == nil || preview.PlacementPreview.Error.Details == nil || preview.PlacementPreview.Error.Details.ReasonCode != "dependency_hierarchy_deadlock" {
		t.Fatalf("Objective placement preview did not explain the hierarchy deadlock: preview=%+v err=%v", preview.PlacementPreview, err)
	}
	_, err = service.SetObjectiveParent(ctx, commandContext(), a.SetObjectiveParentCommand{
		Scope: outcome.Scope(), ObjectiveID: branch.ID, ExpectedVersion: branch.Version, ParentObjectiveID: &root.ID,
	})
	requireHierarchyDeadlock(t, err, "parent_objective_id", task.Ref(), branch.Ref(), root.Ref())
	loaded, err := service.GetObjective(ctx, branch.Scope, branch.ID)
	if err != nil || loaded.Value.ParentObjectiveID != nil || loaded.Value.Version != branch.Version {
		t.Fatalf("rejected reparent changed Objective hierarchy: objective=%+v err=%v", loaded.Value, err)
	}
}

func TestConcurrentOppositeDependenciesInSameOutcomeCannotCreateCycle(t *testing.T) {
	ctx := context.Background()
	service, store := hierarchyDeadlockService(t)
	outcome := setupActiveOutcome(t, service)
	setHierarchyDeadlockProfile(t, store, outcome.Scope(), d.OperationalProfileOperationalV2)
	objective := editingObjective(t, service, outcome.Scope(), "Shared owner", nil)
	first := hierarchyDeadlockTask(t, service, outcome.Scope(), objective.ID, "First")
	second := hierarchyDeadlockTask(t, service, outcome.Scope(), objective.ID, "Second")

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for _, edge := range []struct{ source, target d.EntityRef }{{first.Ref(), second.Ref()}, {second.Ref(), first.Ref()}} {
		wg.Add(1)
		go func(source, target d.EntityRef) {
			defer wg.Done()
			<-start
			_, err := service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{
				Scope: outcome.Scope(), SourceRef: source, TargetRef: target, Strength: d.DependencyStrengthHard,
			})
			errs <- err
		}(edge.source, edge.target)
	}
	close(start)
	wg.Wait()
	close(errs)
	succeeded := 0
	for err := range errs {
		if err == nil {
			succeeded++
			continue
		}
		code, ok := d.ErrorCodeOf(err)
		if !ok || (code != d.ErrorCodeDependencyCycle && code != d.ErrorCodeTransactionConflict) {
			t.Fatalf("opposing dependency failed for an unexpected reason: %v", err)
		}
	}
	if succeeded != 1 {
		t.Fatalf("opposite dependency insertions succeeded %d times, want exactly one", succeeded)
	}
	active := 0
	for _, relation := range dependencyRelations(t, store, outcome.Scope()) {
		if relation.Lifecycle == d.RelationLifecycleActive && relation.RelationType == d.RelationTypeDependsOn {
			active++
		}
	}
	if active != 1 {
		t.Fatalf("final dependency graph has %d active edges, want one", active)
	}
}

func TestCompatibilityV1KeepsHistoricalDependencyAndPlacementBehavior(t *testing.T) {
	ctx := context.Background()
	service, store := hierarchyDeadlockService(t)
	outcome := setupActiveOutcome(t, service)
	setHierarchyDeadlockProfile(t, store, outcome.Scope(), d.OperationalProfileCompatibilityV1)
	root := editingObjective(t, service, outcome.Scope(), "Root", nil)
	child := editingObjective(t, service, outcome.Scope(), "Child", &root.ID)
	other := editingObjective(t, service, outcome.Scope(), "Other", nil)
	descendantTask := hierarchyDeadlockTask(t, service, outcome.Scope(), child.ID, "Descendant work")
	if _, err := service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: descendantTask.Ref(), TargetRef: root.Ref(), Strength: d.DependencyStrengthHard,
	}); err != nil {
		t.Fatalf("compatibility_v1 must retain the historical Task-to-ancestor behavior: %v", err)
	}
	if _, err := service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: root.Ref(), TargetRef: child.Ref(), Strength: d.DependencyStrengthHard,
	}); err != nil {
		t.Fatalf("compatibility_v1 must retain historical Objective dependency behavior: %v", err)
	}
	if _, err := service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: descendantTask.Ref(), TargetRef: child.Ref(), Strength: d.DependencyStrengthHard,
	}); err == nil {
		t.Fatal("compatibility_v1 must retain the direct own-Objective hard-dependency veto")
	} else {
		requireEditingReason(t, err, d.ErrorCodePreconditionFailed, "own_objective_hard_dependency")
	}

	batchTask := hierarchyDeadlockTask(t, service, outcome.Scope(), child.ID, "Batch dependency work")
	current, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	batch := a.ApplyDependencyChangesCommand{
		Scope: outcome.Scope(), ExpectedOutcomeRevision: current.OutcomeRevision,
		ExpectedSources: []a.DependencySourceVersion{{SourceRef: root.Ref(), ExpectedVersion: root.Version}},
		Changes:         []a.DependencyChange{{Action: a.DependencyChangeAdd, SourceRef: dependencyRefPointer(root.Ref()), TargetRef: dependencyRefPointer(batchTask.Ref()), Strength: d.DependencyStrengthHard}},
	}
	if _, err := service.ApplyDependencyChanges(ctx, commandContext(), batch); err != nil {
		t.Fatalf("compatibility_v1 batch must not impose the operational_v2 hierarchy gate: %v", err)
	}

	moveTask := hierarchyDeadlockTask(t, service, outcome.Scope(), other.ID, "Move under dependency ancestor")
	if _, err := service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: moveTask.Ref(), TargetRef: root.Ref(), Strength: d.DependencyStrengthHard,
	}); err != nil {
		t.Fatalf("compatibility_v1 dependency setup failed: %v", err)
	}
	if _, err := service.SetWorkItemObjective(ctx, commandContext(), a.SetWorkItemObjectiveCommand{
		Scope: outcome.Scope(), WorkItemID: moveTask.ID, ExpectedVersion: moveTask.Version, ObjectiveID: &child.ID,
	}); err != nil {
		t.Fatalf("compatibility_v1 Task move must not impose the operational_v2 hierarchy gate: %v", err)
	}
	ownMoveTask := hierarchyDeadlockTask(t, service, outcome.Scope(), other.ID, "Move into own dependency")
	if _, err := service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: ownMoveTask.Ref(), TargetRef: child.Ref(), Strength: d.DependencyStrengthHard,
	}); err != nil {
		t.Fatalf("compatibility_v1 own-Objective move setup failed: %v", err)
	}
	_, err = service.SetWorkItemObjective(ctx, commandContext(), a.SetWorkItemObjectiveCommand{
		Scope: outcome.Scope(), WorkItemID: ownMoveTask.ID, ExpectedVersion: ownMoveTask.Version, ObjectiveID: &child.ID,
	})
	requireEditingReason(t, err, d.ErrorCodePreconditionFailed, "own_objective_hard_dependency")

	branch := editingObjective(t, service, outcome.Scope(), "Branch", nil)
	branchTask := hierarchyDeadlockTask(t, service, outcome.Scope(), branch.ID, "Reparented work")
	if _, err := service.AddDependency(ctx, commandContext(), a.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: branchTask.Ref(), TargetRef: root.Ref(), Strength: d.DependencyStrengthHard,
	}); err != nil {
		t.Fatalf("compatibility_v1 reparent dependency setup failed: %v", err)
	}
	if _, err := service.SetObjectiveParent(ctx, commandContext(), a.SetObjectiveParentCommand{
		Scope: outcome.Scope(), ObjectiveID: branch.ID, ExpectedVersion: branch.Version, ParentObjectiveID: &root.ID,
	}); err != nil {
		t.Fatalf("compatibility_v1 Objective reparent must not impose the operational_v2 hierarchy gate: %v", err)
	}
}
