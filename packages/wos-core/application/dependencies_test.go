package application_test

import (
	"context"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
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
		Scope:     scope,
		Title:     title,
		Priority:  domain.PriorityNormal,
		Lifecycle: domain.WorkItemLifecycleTodo,
	})
	if err != nil {
		t.Fatal(err)
	}
	return result.Value
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
	source := createTodoWork(t, service, outcome.Scope(), "Source")

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
	source := createTodoWork(t, service, outcome.Scope(), "Advisory source")

	if _, err := service.AddDependency(ctx, commandContext(), application.AddDependencyCommand{
		Scope:     outcome.Scope(),
		SourceRef: source.Ref(),
		TargetRef: target.Ref(),
		Strength:  domain.DependencyStrengthAdvisory,
		Reason:    "useful ordering hint",
	}); err != nil {
		t.Fatal(err)
	}
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
	a := createTodoWork(t, service, outcome.Scope(), "A")
	b := createTodoWork(t, service, outcome.Scope(), "B")
	c := createTodoWork(t, service, outcome.Scope(), "C")

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
	source := createTodoWork(t, service, outcome.Scope(), "Blocked source")

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
	ready, err := service.ListReadyWork(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if containsReady(ready.Items, source.ID) {
		t.Fatal("cancelled dependency target must not satisfy target_completed")
	}
}

func TestAddingDependencyToInProgressSourceRequiresReasonAndCompletionGate(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	target := createTodoWork(t, service, outcome.Scope(), "Late dependency target")
	source := createTodoWork(t, service, outcome.Scope(), "Already running source")

	claimed, err := service.ClaimWorkItem(ctx, commandContext(), application.ClaimWorkItemCommand{
		Scope:           outcome.Scope(),
		WorkItemID:      source.ID,
		ExpectedVersion: source.Version,
		TTL:             domain.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	source = claimed.Value

	_, err = service.AddDependency(ctx, commandContext(), application.AddDependencyCommand{
		Scope:     outcome.Scope(),
		SourceRef: source.Ref(),
		TargetRef: target.Ref(),
		Strength:  domain.DependencyStrengthHard,
	})
	if err == nil {
		t.Fatal("in-progress dependency mutation without reason unexpectedly succeeded")
	}
	code, ok := domain.ErrorCodeOf(err)
	if !ok || code != domain.ErrorCodeInvalidArgument {
		t.Fatalf("error code = %q, want invalid_argument: %v", code, err)
	}

	if _, err := service.AddDependency(ctx, commandContext(), application.AddDependencyCommand{
		Scope:     outcome.Scope(),
		SourceRef: source.Ref(),
		TargetRef: target.Ref(),
		Strength:  domain.DependencyStrengthHard,
		Reason:    "new prerequisite discovered during execution",
	}); err != nil {
		t.Fatal(err)
	}

	_, err = service.CompleteWorkItem(ctx, commandContext(), application.CompleteWorkItemCommand{
		Scope:           outcome.Scope(),
		WorkItemID:      source.ID,
		ExpectedVersion: source.Version,
		ClaimID:         source.CurrentLease.ClaimID,
		FencingToken:    source.CurrentLease.FencingToken,
		ResultSummary:   "source work finished",
		Reason:          "attempt completion while dependency is unmet",
	})
	if err == nil {
		t.Fatal("completion unexpectedly ignored an unsatisfied hard dependency")
	}
	code, ok = domain.ErrorCodeOf(err)
	if !ok || code != domain.ErrorCodePreconditionFailed {
		t.Fatalf("error code = %q, want precondition_failed: %v", code, err)
	}
}
