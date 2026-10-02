package application_test

import (
	"context"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func TestInheritedBlockersControlReadinessAndResolveIndependently(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	outcome := setupActiveOutcome(t, service)
	parentResult, err := service.CreateObjective(ctx, commandContext(), application.CreateObjectiveCommand{Scope: outcome.Scope(), Title: "Parent", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	parent := parentResult.Value
	childResult, err := service.CreateObjective(ctx, commandContext(), application.CreateObjectiveCommand{Scope: outcome.Scope(), Title: "Child", Priority: domain.PriorityNormal, ParentObjectiveID: &parent.ID})
	if err != nil {
		t.Fatal(err)
	}
	child := childResult.Value
	workResult, err := service.CreateWorkItem(ctx, commandContext(), application.CreateWorkItemCommand{Scope: outcome.Scope(), Title: "Nested work", Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleTodo, ObjectiveID: &child.ID})
	if err != nil {
		t.Fatal(err)
	}
	work := workResult.Value
	issueResult, err := service.CreateIssue(ctx, commandContext(), application.CreateIssueCommand{Scope: outcome.Scope(), Title: "Shared cause", Severity: domain.IssueSeverityMajor, AffectedRefs: []domain.EntityRef{work.Ref()}})
	if err != nil {
		t.Fatal(err)
	}
	issue := issueResult.Value
	cause := issue.Ref()
	b1r, err := service.CreateBlocker(ctx, commandContext(), application.CreateBlockerCommand{Scope: outcome.Scope(), BlockedRef: parent.Ref(), CauseRef: &cause, Description: "First impediment", Propagation: domain.BlockerPropagationSubtree})
	if err != nil {
		t.Fatal(err)
	}
	b1 := b1r.Value
	b2r, err := service.CreateBlocker(ctx, commandContext(), application.CreateBlockerCommand{Scope: outcome.Scope(), BlockedRef: parent.Ref(), Description: "Second impediment", Propagation: domain.BlockerPropagationSubtree})
	if err != nil {
		t.Fatal(err)
	}
	b2 := b2r.Value
	ready, err := service.ListReadyWork(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if containsReady(ready.Items, work.ID) {
		t.Fatal("blocked work unexpectedly ready")
	}
	state, _, err := service.GetBlockingState(ctx, work.Ref())
	if err != nil {
		t.Fatal(err)
	}
	if !state.IsBlocked || len(state.ActiveBlockers) != 2 {
		t.Fatalf("blocking=%#v", state)
	}
	for _, applied := range state.ActiveBlockers {
		if !applied.Inherited || applied.InheritedFrom == nil || *applied.InheritedFrom != parent.Ref() {
			t.Fatalf("inheritance=%#v", applied)
		}
	}
	if _, err := service.ResolveIssue(ctx, commandContext(), application.ResolveIssueCommand{Scope: outcome.Scope(), IssueID: issue.ID, ExpectedVersion: issue.Version, ResolutionSummary: "root cause corrected"}); err != nil {
		t.Fatal(err)
	}
	state, _, err = service.GetBlockingState(ctx, work.Ref())
	if err != nil {
		t.Fatal(err)
	}
	if len(state.ActiveBlockers) != 2 {
		t.Fatal("issue resolution silently resolved blocker")
	}
	if _, err := service.ResolveBlocker(ctx, commandContext(), application.ResolveBlockerCommand{Scope: outcome.Scope(), BlockerID: b1.ID, ExpectedVersion: b1.Version, ResolutionSummary: "first clear"}); err != nil {
		t.Fatal(err)
	}
	state, _, err = service.GetBlockingState(ctx, work.Ref())
	if err != nil {
		t.Fatal(err)
	}
	if !state.IsBlocked || len(state.ActiveBlockers) != 1 || state.ActiveBlockers[0].Blocker.ID != b2.ID {
		t.Fatalf("remaining=%#v", state)
	}
	if _, err := service.ResolveBlocker(ctx, commandContext(), application.ResolveBlockerCommand{Scope: outcome.Scope(), BlockerID: b2.ID, ExpectedVersion: b2.Version, ResolutionSummary: "second clear"}); err != nil {
		t.Fatal(err)
	}
	ready, err = service.ListReadyWork(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if !containsReady(ready.Items, work.ID) {
		t.Fatal("work did not return to ready")
	}
	events := store.SnapshotDomainEvents(outcome.Scope())
	counts := map[string]int{}
	for _, e := range events {
		counts[e.EventType]++
	}
	if counts["issue.created"] != 1 || counts["issue.resolved"] != 1 || counts["blocker.created"] != 2 || counts["blocker.resolved"] != 2 {
		t.Fatalf("events=%v", counts)
	}
}

func TestTerminalTargetRejectsNewBlocker(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	work := createTodoWork(t, service, outcome.Scope(), "Terminal")
	cancelled, err := service.CancelWorkItem(ctx, commandContext(), application.CancelWorkItemCommand{Scope: outcome.Scope(), WorkItemID: work.ID, ExpectedVersion: work.Version, Reason: "invalidated"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.CreateBlocker(ctx, commandContext(), application.CreateBlockerCommand{Scope: outcome.Scope(), BlockedRef: cancelled.Value.Ref(), Description: "too late"})
	if err == nil {
		t.Fatal("terminal target accepted blocker")
	}
}
func TestIssueDuplicateChainRejectsCycle(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	ar, err := service.CreateIssue(ctx, commandContext(), application.CreateIssueCommand{Scope: outcome.Scope(), Title: "A", Severity: domain.IssueSeverityMinor})
	if err != nil {
		t.Fatal(err)
	}
	br, err := service.CreateIssue(ctx, commandContext(), application.CreateIssueCommand{Scope: outcome.Scope(), Title: "B", Severity: domain.IssueSeverityMinor})
	if err != nil {
		t.Fatal(err)
	}
	a, b := ar.Value, br.Value
	if _, err := service.MarkIssueDuplicate(ctx, commandContext(), application.MarkIssueDuplicateCommand{Scope: outcome.Scope(), IssueID: a.ID, ExpectedVersion: a.Version, DuplicateOfID: b.ID}); err != nil {
		t.Fatal(err)
	}
	_, err = service.MarkIssueDuplicate(ctx, commandContext(), application.MarkIssueDuplicateCommand{Scope: outcome.Scope(), IssueID: b.ID, ExpectedVersion: b.Version, DuplicateOfID: a.ID})
	if err == nil {
		t.Fatal("duplicate cycle succeeded")
	}
}
