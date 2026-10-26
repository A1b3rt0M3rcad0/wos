package application_test

import (
	"context"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func setupActiveOutcome(t *testing.T, service *application.Service) domain.Outcome {
	t.Helper()
	ctx := context.Background()
	cc := commandContext()
	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  id("0199e260-0000-7000-8000-000000000001"),
		Title:        "Dependency outcome",
		DesiredState: "Dependency semantics are deterministic",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value
	if _, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner:            outcome.Ref(),
		ExpectedVersion:  outcome.Version,
		Title:            "Outcome criterion",
		Required:         true,
		VerificationMode: domain.VerificationModeAttestation,
	}); err != nil {
		t.Fatal(err)
	}
	outcome.Version++
	activated, err := service.ActivateOutcome(ctx, cc, application.ActivateOutcomeCommand{
		Scope:           outcome.Scope(),
		ExpectedVersion: outcome.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	return activated.Value
}

func createTodoWork(t *testing.T, service *application.Service, scope domain.Scope, title string) domain.WorkItem {
	t.Helper()
	result, err := service.CreateWorkItem(context.Background(), commandContext(), application.CreateWorkItemCommand{
		Scope: scope, ObjectiveID: objectiveForWorkFixture(t, service, scope),
		Title:     title,
		Priority:  domain.PriorityNormal,
		Lifecycle: domain.WorkItemLifecycleTodo,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.Value
}

func issueObjectiveForTest(t *testing.T, service *application.Service, scope domain.Scope) domain.ID {
	t.Helper()
	return *objectiveForWorkFixture(t, service, scope)
}

func containsReady(items []application.ReadyWorkItem, id domain.ID) bool {
	for _, item := range items {
		if item.WorkItem.ID == id {
			return true
		}
	}
	return false
}

func TestHardDependencyControlsReadyWorkAndRemovalRestoresReadiness(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	outcome := setupActiveOutcome(t, service)
	target := createTodoWork(t, service, outcome.Scope(), "Target")
	source := deferPlanningWork(t, service, createTodoWork(t, service, outcome.Scope(), "Source"))

	added, err := service.AddDependency(ctx, commandContext(), application.AddDependencyCommand{
		Scope:     outcome.Scope(),
		SourceRef: source.Ref(),
		TargetRef: target.Ref(),
		Strength:  domain.DependencyStrengthHard,
		Reason:    "source requires target",
	})
	if err != nil {
		t.Fatal(err)
	}

	source = activatePlanningWork(t, service, source)
	ready, err := service.ListReadyWork(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if !containsReady(ready.Items, target.ID) {
		t.Fatal("target should be ready")
	}
	if containsReady(ready.Items, source.ID) {
		t.Fatal("source should not be ready while hard dependency is unsatisfied")
	}

	source = deferPlanningWork(t, service, source)
	removed, err := service.RemoveDependency(ctx, commandContext(), application.RemoveDependencyCommand{
		Scope:           outcome.Scope(),
		RelationID:      added.Value.ID,
		ExpectedVersion: added.Value.Version,
		Reason:          "dependency no longer applies",
	})
	if err != nil {
		t.Fatal(err)
	}
	if removed.Value.Lifecycle != domain.RelationLifecycleRemoved {
		t.Fatalf("relation lifecycle = %q, want removed", removed.Value.Lifecycle)
	}

	source = activatePlanningWork(t, service, source)
	ready, err = service.ListReadyWork(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if !containsReady(ready.Items, source.ID) {
		t.Fatal("source should become ready after hard dependency removal")
	}

	events := store.SnapshotDomainEvents(outcome.Scope())
	var created, removedEvent bool
	for _, event := range events {
		switch event.EventType {
		case "relation.created":
			created = true
		case "relation.removed":
			removedEvent = true
		}
	}
	if !created || !removedEvent {
		t.Fatalf("dependency audit events missing: created=%v removed=%v", created, removedEvent)
	}
}

func TestAdvisoryDependencyDoesNotBlockReadyWork(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	target := createTodoWork(t, service, outcome.Scope(), "Advisory target")
	source := deferPlanningWork(t, service, createTodoWork(t, service, outcome.Scope(), "Advisory source"))

	if _, err := service.AddDependency(ctx, commandContext(), application.AddDependencyCommand{
		Scope:     outcome.Scope(),
		SourceRef: source.Ref(),
		TargetRef: target.Ref(),
		Strength:  domain.DependencyStrengthAdvisory,
		Reason:    "useful ordering hint",
	}); err != nil {
		t.Fatal(err)
	}
	source = activatePlanningWork(t, service, source)
	ready, err := service.ListReadyWork(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if !containsReady(ready.Items, source.ID) {
		t.Fatal("advisory dependency must not block readiness")
	}
}

func TestDependencyCycleRejectsIndirectCycle(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	a := deferPlanningWork(t, service, createTodoWork(t, service, outcome.Scope(), "A"))
	b := deferPlanningWork(t, service, createTodoWork(t, service, outcome.Scope(), "B"))
	c := deferPlanningWork(t, service, createTodoWork(t, service, outcome.Scope(), "C"))

	for _, cmd := range []application.AddDependencyCommand{
		{Scope: outcome.Scope(), SourceRef: a.Ref(), TargetRef: b.Ref(), Strength: domain.DependencyStrengthHard},
		{Scope: outcome.Scope(), SourceRef: b.Ref(), TargetRef: c.Ref(), Strength: domain.DependencyStrengthAdvisory},
	} {
		if _, err := service.AddDependency(ctx, commandContext(), cmd); err != nil {
			t.Fatal(err)
		}
	}
	_, err := service.AddDependency(ctx, commandContext(), application.AddDependencyCommand{
		Scope:     outcome.Scope(),
		SourceRef: c.Ref(),
		TargetRef: a.Ref(),
		Strength:  domain.DependencyStrengthHard,
	})
	if err == nil {
		t.Fatal("indirect dependency cycle unexpectedly succeeded")
	}
	code, ok := domain.ErrorCodeOf(err)
	if !ok || code != domain.ErrorCodeDependencyCycle {
		t.Fatalf("error code = %q, want dependency_cycle: %v", code, err)
	}
}

func TestCancelledTargetDoesNotSatisfyHardDependency(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	target := createTodoWork(t, service, outcome.Scope(), "Cancelled target")
	source := deferPlanningWork(t, service, createTodoWork(t, service, outcome.Scope(), "Blocked source"))

	if _, err := service.AddDependency(ctx, commandContext(), application.AddDependencyCommand{
		Scope:     outcome.Scope(),
		SourceRef: source.Ref(),
		TargetRef: target.Ref(),
		Strength:  domain.DependencyStrengthHard,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CancelWorkItem(ctx, commandContext(), application.CancelWorkItemCommand{
		Scope:           outcome.Scope(),
		WorkItemID:      target.ID,
		ExpectedVersion: target.Version,
		Reason:          "target invalidated",
	}); err != nil {
		t.Fatal(err)
	}
	source = activatePlanningWork(t, service, source)
	ready, err := service.ListReadyWork(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if containsReady(ready.Items, source.ID) {
		t.Fatal("cancelled dependency target must not satisfy target_completed")
	}
}

func TestAddingDependencyToInProgressSourceIsRejectedWithoutChangingWork(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	target := createTodoWork(t, service, outcome.Scope(), "Late dependency target")
	source := createTodoWork(t, service, outcome.Scope(), "Already running source")
	claimed, err := service.ClaimWorkItem(ctx, commandContext(), application.ClaimWorkItemCommand{Scope: source.Scope, WorkItemID: source.ID, ExpectedVersion: source.Version, TTL: domain.DefaultLeaseTTL})
	if err != nil {
		t.Fatal(err)
	}
	source = claimed.Value
	for _, reason := range []string{"", "new prerequisite discovered during execution"} {
		_, err = service.AddDependency(ctx, commandContext(), application.AddDependencyCommand{Scope: source.Scope, SourceRef: source.Ref(), TargetRef: target.Ref(), Strength: domain.DependencyStrengthHard, Reason: reason})
		code, _ := domain.ErrorCodeOf(err)
		if code != domain.ErrorCodePreconditionFailed {
			t.Fatalf("expected frozen planning, got %v", err)
		}
	}
	loaded, err := service.GetWorkItem(ctx, source.Scope, source.ID)
	if err != nil || loaded.Value.Version != source.Version {
		t.Fatal("rejected edits changed work", err)
	}
	completed, err := service.CompleteWorkItem(ctx, commandContext(), application.CompleteWorkItemCommand{Scope: source.Scope, WorkItemID: source.ID, ExpectedVersion: source.Version, ClaimID: source.CurrentLease.ClaimID, FencingToken: source.CurrentLease.FencingToken, ResultSummary: "Finished the agreed work", Reason: "No late dependency was persisted"})
	if err != nil || completed.Value.Lifecycle != domain.WorkItemLifecycleDone {
		t.Fatal("rejected dependency changed completion gates", err)
	}
}

func deferPlanningWork(t *testing.T, service *application.Service, value domain.WorkItem) domain.WorkItem {
	t.Helper()
	result, err := service.DeferWorkItem(context.Background(), commandContext(), application.DeferWorkItemCommand{Scope: value.Scope, WorkItemID: value.ID, ExpectedVersion: value.Version})
	if err != nil {
		t.Fatal(err)
	}
	return result.Value
}
func activatePlanningWork(t *testing.T, service *application.Service, value domain.WorkItem) domain.WorkItem {
	t.Helper()
	loaded, err := service.GetWorkItem(context.Background(), value.Scope, value.ID)
	if err != nil {
		t.Fatal(err)
	}
	result, err := service.ActivateWorkItem(context.Background(), commandContext(), application.ActivateWorkItemCommand{Scope: value.Scope, WorkItemID: value.ID, ExpectedVersion: loaded.Value.Version})
	if err != nil {
		t.Fatal(err)
	}
	return result.Value
}

func TestDependencyChangesPreviewIsReadOnlyAndApplyCommitsFinalGraphOnce(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	outcome := setupActiveOutcome(t, service)
	source := deferPlanningWork(t, service, createTodoWork(t, service, outcome.Scope(), "Source"))
	target := deferPlanningWork(t, service, createTodoWork(t, service, outcome.Scope(), "Target"))
	old, err := service.AddDependency(ctx, commandContext(), application.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: source.Ref(), TargetRef: target.Ref(), Strength: domain.DependencyStrengthHard,
	})
	if err != nil {
		t.Fatal(err)
	}
	current, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	cc := commandContext()
	cc.IdempotencyKey = "dependency-batch-replacement"
	cmd := application.ApplyDependencyChangesCommand{
		Scope: outcome.Scope(), ExpectedOutcomeRevision: current.OutcomeRevision,
		ExpectedSources: []application.DependencySourceVersion{{SourceRef: source.Ref(), ExpectedVersion: source.Version}, {SourceRef: target.Ref(), ExpectedVersion: target.Version}},
		Changes: []application.DependencyChange{
			{Action: application.DependencyChangeAdd, SourceRef: dependencyRefPointer(target.Ref()), TargetRef: dependencyRefPointer(source.Ref()), Strength: domain.DependencyStrengthAdvisory, Reason: "replace the edge direction atomically"},
			{Action: application.DependencyChangeRemove, RelationID: idPointer(old.Value.ID), ExpectedVersion: versionPointer(old.Value.Version), Reason: "replace the dependency contract"},
		},
	}
	eventsBefore := len(store.SnapshotDomainEvents(outcome.Scope()))
	preview, err := service.PreviewDependencyChanges(ctx, outcome.Scope(), application.DependencyChangesPreviewQuery{
		ExpectedOutcomeRevision: cmd.ExpectedOutcomeRevision, ExpectedSources: cmd.ExpectedSources, Changes: cmd.Changes,
	})
	if err != nil || preview.Availability != "available" || preview.FinalActiveDependencyCount != 1 || len(preview.Changes) != 2 {
		t.Fatalf("dependency batch preview = %+v, err=%v", preview, err)
	}
	afterPreview, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil || afterPreview.OutcomeRevision != current.OutcomeRevision || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsBefore {
		t.Fatalf("preview mutated durable state: revision=%d err=%v events=%d", afterPreview.OutcomeRevision, err, len(store.SnapshotDomainEvents(outcome.Scope())))
	}
	beforeRelations := dependencyRelations(t, store, outcome.Scope())
	if len(beforeRelations) != 1 || beforeRelations[0].ID != old.Value.ID || beforeRelations[0].Lifecycle != domain.RelationLifecycleActive {
		t.Fatalf("preview changed relations: %+v", beforeRelations)
	}

	result, err := service.ApplyDependencyChanges(ctx, cc, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if result.OutcomeRevision != current.OutcomeRevision+1 || len(result.Value.Changes) != 2 {
		t.Fatalf("batch result = %+v", result)
	}
	finalRelations := dependencyRelations(t, store, outcome.Scope())
	if len(finalRelations) != 2 {
		t.Fatalf("expected historical removed relation plus replacement, got %+v", finalRelations)
	}
	var added, removed domain.Relation
	for _, relation := range finalRelations {
		if relation.ID == old.Value.ID {
			removed = relation
		} else {
			added = relation
		}
	}
	if removed.Lifecycle != domain.RelationLifecycleRemoved || added.Lifecycle != domain.RelationLifecycleActive || added.Strength != domain.DependencyStrengthAdvisory || added.SourceRef != target.Ref() || added.TargetRef != source.Ref() {
		t.Fatalf("replacement did not produce the requested final graph: removed=%+v added=%+v", removed, added)
	}
	events := store.SnapshotDomainEvents(outcome.Scope())
	batchEvents := events[len(events)-2:]
	if batchEvents[0].EventType != "relation.created" || batchEvents[1].EventType != "relation.removed" || batchEvents[0].OutcomeRevision != result.OutcomeRevision || batchEvents[1].OutcomeRevision != result.OutcomeRevision || batchEvents[0].CommandID != cc.CommandID || batchEvents[1].CommandID != cc.CommandID {
		t.Fatalf("batch events are not correlated at one revision: %+v", batchEvents)
	}
	replay, err := service.ApplyDependencyChanges(ctx, cc, cmd)
	if err != nil || !replay.IdempotentReplay || replay.Value.Changes[0].Relation.ID != result.Value.Changes[0].Relation.ID || replay.OutcomeRevision != result.OutcomeRevision {
		t.Fatalf("batch replay = %+v, err=%v", replay, err)
	}
}

func TestDependencyChangesRejectFinalGraphCycleWithoutPartialWrites(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	outcome := setupActiveOutcome(t, service)
	a := deferPlanningWork(t, service, createTodoWork(t, service, outcome.Scope(), "A"))
	b := deferPlanningWork(t, service, createTodoWork(t, service, outcome.Scope(), "B"))
	current, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	cmd := application.ApplyDependencyChangesCommand{
		Scope: outcome.Scope(), ExpectedOutcomeRevision: current.OutcomeRevision,
		ExpectedSources: []application.DependencySourceVersion{{SourceRef: a.Ref(), ExpectedVersion: a.Version}, {SourceRef: b.Ref(), ExpectedVersion: b.Version}},
		Changes: []application.DependencyChange{
			{Action: application.DependencyChangeAdd, SourceRef: dependencyRefPointer(a.Ref()), TargetRef: dependencyRefPointer(b.Ref()), Strength: domain.DependencyStrengthHard},
			{Action: application.DependencyChangeAdd, SourceRef: dependencyRefPointer(b.Ref()), TargetRef: dependencyRefPointer(a.Ref()), Strength: domain.DependencyStrengthAdvisory},
		},
	}
	eventsBefore := len(store.SnapshotDomainEvents(outcome.Scope()))
	preview, err := service.PreviewDependencyChanges(ctx, outcome.Scope(), application.DependencyChangesPreviewQuery{ExpectedOutcomeRevision: cmd.ExpectedOutcomeRevision, ExpectedSources: cmd.ExpectedSources, Changes: cmd.Changes})
	if err != nil || preview.Availability != "blocked" || preview.Error == nil || preview.Error.Code != domain.ErrorCodeDependencyCycle {
		t.Fatalf("cycle preview = %+v, err=%v", preview, err)
	}
	_, err = service.ApplyDependencyChanges(ctx, commandContext(), cmd)
	if code, ok := domain.ErrorCodeOf(err); !ok || code != domain.ErrorCodeDependencyCycle {
		t.Fatalf("cycle apply error = %v", err)
	}
	after, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil || after.OutcomeRevision != current.OutcomeRevision || len(dependencyRelations(t, store, outcome.Scope())) != 0 || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsBefore {
		t.Fatalf("rejected cyclic batch changed persisted state: revision=%d err=%v", after.OutcomeRevision, err)
	}
}

func TestDependencyChangesGuardEverySourceAndRemovalVersion(t *testing.T) {
	ctx := context.Background()
	t.Run("second source version conflict rejects all additions", func(t *testing.T) {
		service, store := newService(t)
		outcome := setupActiveOutcome(t, service)
		a := deferPlanningWork(t, service, createTodoWork(t, service, outcome.Scope(), "A"))
		b := deferPlanningWork(t, service, createTodoWork(t, service, outcome.Scope(), "B"))
		c := createTodoWork(t, service, outcome.Scope(), "C")
		current, _ := service.GetOutcome(ctx, outcome.Scope())
		cmd := application.ApplyDependencyChangesCommand{
			Scope: outcome.Scope(), ExpectedOutcomeRevision: current.OutcomeRevision,
			ExpectedSources: []application.DependencySourceVersion{{SourceRef: a.Ref(), ExpectedVersion: a.Version}, {SourceRef: b.Ref(), ExpectedVersion: b.Version + 1}},
			Changes: []application.DependencyChange{
				{Action: application.DependencyChangeAdd, SourceRef: dependencyRefPointer(a.Ref()), TargetRef: dependencyRefPointer(c.Ref()), Strength: domain.DependencyStrengthHard},
				{Action: application.DependencyChangeAdd, SourceRef: dependencyRefPointer(b.Ref()), TargetRef: dependencyRefPointer(c.Ref()), Strength: domain.DependencyStrengthHard},
			},
		}
		eventsBefore := len(store.SnapshotDomainEvents(outcome.Scope()))
		_, err := service.ApplyDependencyChanges(ctx, commandContext(), cmd)
		if code, ok := domain.ErrorCodeOf(err); !ok || code != domain.ErrorCodePreconditionFailed {
			t.Fatalf("stale second source error = %v", err)
		}
		after, _ := service.GetOutcome(ctx, outcome.Scope())
		if after.OutcomeRevision != current.OutcomeRevision || len(dependencyRelations(t, store, outcome.Scope())) != 0 || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsBefore {
			t.Fatal("stale source conflict partially wrote batch")
		}
	})
	t.Run("stale removal version rejects accompanying add", func(t *testing.T) {
		service, store := newService(t)
		outcome := setupActiveOutcome(t, service)
		a := deferPlanningWork(t, service, createTodoWork(t, service, outcome.Scope(), "A"))
		b := createTodoWork(t, service, outcome.Scope(), "B")
		c := deferPlanningWork(t, service, createTodoWork(t, service, outcome.Scope(), "C"))
		relation, err := service.AddDependency(ctx, commandContext(), application.AddDependencyCommand{Scope: outcome.Scope(), SourceRef: a.Ref(), TargetRef: b.Ref(), Strength: domain.DependencyStrengthHard})
		if err != nil {
			t.Fatal(err)
		}
		current, _ := service.GetOutcome(ctx, outcome.Scope())
		cmd := application.ApplyDependencyChangesCommand{
			Scope: outcome.Scope(), ExpectedOutcomeRevision: current.OutcomeRevision,
			ExpectedSources: []application.DependencySourceVersion{{SourceRef: a.Ref(), ExpectedVersion: a.Version}, {SourceRef: c.Ref(), ExpectedVersion: c.Version}},
			Changes: []application.DependencyChange{
				{Action: application.DependencyChangeAdd, SourceRef: dependencyRefPointer(c.Ref()), TargetRef: dependencyRefPointer(b.Ref()), Strength: domain.DependencyStrengthAdvisory},
				{Action: application.DependencyChangeRemove, RelationID: idPointer(relation.Value.ID), ExpectedVersion: versionPointer(relation.Value.Version + 1), Reason: "attempt stale removal"},
			},
		}
		eventsBefore := len(store.SnapshotDomainEvents(outcome.Scope()))
		_, err = service.ApplyDependencyChanges(ctx, commandContext(), cmd)
		if code, ok := domain.ErrorCodeOf(err); !ok || code != domain.ErrorCodePreconditionFailed {
			t.Fatalf("stale removal version error = %v", err)
		}
		after, _ := service.GetOutcome(ctx, outcome.Scope())
		relations := dependencyRelations(t, store, outcome.Scope())
		if after.OutcomeRevision != current.OutcomeRevision || len(relations) != 1 || relations[0].Lifecycle != domain.RelationLifecycleActive || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsBefore {
			t.Fatal("stale removal conflict partially wrote batch")
		}
	})
}

func TestDependencyChangesRejectsFrozenSourcesAsOneBatch(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	outcome := setupActiveOutcome(t, service)
	frozen := createTodoWork(t, service, outcome.Scope(), "Frozen")
	claim, err := service.ClaimWorkItem(ctx, commandContext(), application.ClaimWorkItemCommand{Scope: frozen.Scope, WorkItemID: frozen.ID, ExpectedVersion: frozen.Version})
	if err != nil {
		t.Fatal(err)
	}
	backlog := deferPlanningWork(t, service, createTodoWork(t, service, outcome.Scope(), "Other source"))
	target := createTodoWork(t, service, outcome.Scope(), "Target")
	current, _ := service.GetOutcome(ctx, outcome.Scope())
	cmd := application.ApplyDependencyChangesCommand{
		Scope: outcome.Scope(), ExpectedOutcomeRevision: current.OutcomeRevision,
		ExpectedSources: []application.DependencySourceVersion{{SourceRef: claim.Value.Ref(), ExpectedVersion: claim.Value.Version}, {SourceRef: backlog.Ref(), ExpectedVersion: backlog.Version}},
		Changes: []application.DependencyChange{
			{Action: application.DependencyChangeAdd, SourceRef: dependencyRefPointer(backlog.Ref()), TargetRef: dependencyRefPointer(target.Ref()), Strength: domain.DependencyStrengthHard},
			{Action: application.DependencyChangeAdd, SourceRef: dependencyRefPointer(claim.Value.Ref()), TargetRef: dependencyRefPointer(target.Ref()), Strength: domain.DependencyStrengthHard},
		},
	}
	eventsBefore := len(store.SnapshotDomainEvents(outcome.Scope()))
	_, err = service.ApplyDependencyChanges(ctx, commandContext(), cmd)
	if err == nil {
		t.Fatal("batch with frozen source unexpectedly succeeded")
	}
	after, _ := service.GetOutcome(ctx, outcome.Scope())
	if after.OutcomeRevision != current.OutcomeRevision || len(dependencyRelations(t, store, outcome.Scope())) != 0 || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsBefore {
		t.Fatal("frozen source failure partially wrote dependency batch")
	}
}

func TestDependencyChangesRejectsInvalidEndpointAndStaleOutcomeSnapshotAtomically(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	outcome := setupActiveOutcome(t, service)
	a := deferPlanningWork(t, service, createTodoWork(t, service, outcome.Scope(), "A"))
	b := deferPlanningWork(t, service, createTodoWork(t, service, outcome.Scope(), "B"))
	target := createTodoWork(t, service, outcome.Scope(), "Valid target")
	current, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	missing := domain.EntityRef{Scope: outcome.Scope(), Kind: domain.EntityKindWorkItem, ID: id("0199e260-0000-7000-8000-000000009999")}
	cmd := application.ApplyDependencyChangesCommand{
		Scope: outcome.Scope(), ExpectedOutcomeRevision: current.OutcomeRevision,
		ExpectedSources: []application.DependencySourceVersion{{SourceRef: a.Ref(), ExpectedVersion: a.Version}, {SourceRef: b.Ref(), ExpectedVersion: b.Version}},
		Changes: []application.DependencyChange{
			{Action: application.DependencyChangeAdd, SourceRef: dependencyRefPointer(a.Ref()), TargetRef: dependencyRefPointer(target.Ref()), Strength: domain.DependencyStrengthHard},
			{Action: application.DependencyChangeAdd, SourceRef: dependencyRefPointer(b.Ref()), TargetRef: dependencyRefPointer(missing), Strength: domain.DependencyStrengthAdvisory},
		},
	}
	eventsBefore := len(store.SnapshotDomainEvents(outcome.Scope()))
	_, err = service.ApplyDependencyChanges(ctx, commandContext(), cmd)
	if code, ok := domain.ErrorCodeOf(err); !ok || code != domain.ErrorCodeNotFound {
		t.Fatalf("missing endpoint error = %v", err)
	}
	if len(dependencyRelations(t, store, outcome.Scope())) != 0 || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsBefore {
		t.Fatal("invalid endpoint failure partially wrote batch")
	}

	cmd.ExpectedOutcomeRevision--
	cmd.Changes = cmd.Changes[:1]
	cmd.ExpectedSources = cmd.ExpectedSources[:1]
	_, err = service.ApplyDependencyChanges(ctx, commandContext(), cmd)
	if code, ok := domain.ErrorCodeOf(err); !ok || code != domain.ErrorCodePreconditionFailed {
		t.Fatalf("stale Outcome snapshot error = %v", err)
	}
	after, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil || after.OutcomeRevision != current.OutcomeRevision || len(dependencyRelations(t, store, outcome.Scope())) != 0 || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsBefore {
		t.Fatal("stale snapshot failure partially wrote batch")
	}
}

func dependencyRelations(t *testing.T, store ports.TransactionManager, scope domain.Scope) []domain.Relation {
	t.Helper()
	uow, err := store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	values, err := uow.Relations().ListByOutcome(context.Background(), scope)
	if err != nil {
		t.Fatal(err)
	}
	return values
}

func dependencyRefPointer(ref domain.EntityRef) *domain.EntityRef { return &ref }
func idPointer(value domain.ID) *domain.ID                        { return &value }
func versionPointer(value domain.Version) *domain.Version         { return &value }
