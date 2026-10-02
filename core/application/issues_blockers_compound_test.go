package application_test

import (
	"context"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func TestReportIssueWithBlockerIsAtomicAndEmitsPerAggregateEvents(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	outcome := setupActiveOutcome(t, service)
	work := createTodoWork(t, service, outcome.Scope(), "blocked atomically")

	result, err := service.ReportIssueWithBlocker(ctx, commandContext(), application.ReportIssueWithBlockerCommand{
		Scope:              outcome.Scope(),
		Title:              "Dependency unavailable",
		Description:        "The external dependency is not usable.",
		Severity:           domain.IssueSeverityMajor,
		AffectedRefs:       []domain.EntityRef{work.Ref()},
		BlockedRef:         work.Ref(),
		BlockerDescription: "Execution cannot continue.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.Value.Blocker.CauseRef == nil || *result.Value.Blocker.CauseRef != result.Value.Issue.Ref() {
		t.Fatalf("blocker cause = %#v, want issue ref", result.Value.Blocker.CauseRef)
	}

	events := store.SnapshotDomainEvents(outcome.Scope())
	var compound []domain.DomainEvent
	for _, event := range events {
		if event.OutcomeRevision == result.OutcomeRevision {
			compound = append(compound, event)
		}
	}
	if len(compound) != 2 {
		t.Fatalf("compound events = %d, want 2", len(compound))
	}
	if compound[0].EventType != "issue.created" || compound[0].AggregateRef != result.Value.Issue.Ref() {
		t.Fatalf("event 0 = %#v", compound[0])
	}
	if compound[1].EventType != "blocker.created" || compound[1].AggregateRef != result.Value.Blocker.Ref() {
		t.Fatalf("event 1 = %#v", compound[1])
	}
	if compound[0].CommandID != compound[1].CommandID || compound[0].OutcomeRevision != compound[1].OutcomeRevision {
		t.Fatal("compound events do not share command identity and outcome revision")
	}

	snapshot, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Issues) != 1 || len(snapshot.Blockers) != 1 {
		t.Fatalf("snapshot issue/blocker counts = %d/%d", len(snapshot.Issues), len(snapshot.Blockers))
	}
	state, ok := blockingState(snapshot.BlockingStates, work.Ref())
	if !ok || !state.IsBlocked || len(state.ActiveBlockers) != 1 {
		t.Fatalf("snapshot blocking state = %#v, found=%v", state, ok)
	}
}

func TestResolveIssueAndBlockersRollsBackEveryAggregateOnConflict(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	work := createTodoWork(t, service, outcome.Scope(), "compound rollback")

	reported, err := service.ReportIssueWithBlocker(ctx, commandContext(), application.ReportIssueWithBlockerCommand{
		Scope:              outcome.Scope(),
		Title:              "Root cause",
		Severity:           domain.IssueSeverityCritical,
		AffectedRefs:       []domain.EntityRef{work.Ref()},
		BlockedRef:         work.Ref(),
		BlockerDescription: "first blocker",
	})
	if err != nil {
		t.Fatal(err)
	}
	issue := reported.Value.Issue
	first := reported.Value.Blocker
	cause := issue.Ref()
	secondResult, err := service.CreateBlocker(ctx, commandContext(), application.CreateBlockerCommand{
		Scope:       outcome.Scope(),
		BlockedRef:  work.Ref(),
		CauseRef:    &cause,
		Description: "second blocker",
	})
	if err != nil {
		t.Fatal(err)
	}
	second := secondResult.Value

	_, err = service.ResolveIssueAndBlockers(ctx, commandContext(), application.ResolveIssueAndBlockersCommand{
		Scope:                  outcome.Scope(),
		IssueID:                issue.ID,
		IssueExpectedVersion:   issue.Version,
		IssueResolutionSummary: "root cause fixed",
		Blockers: []application.BlockerResolution{
			{BlockerID: first.ID, ExpectedVersion: first.Version, ResolutionSummary: "first released", ReleaseConfirmed: true},
			{BlockerID: second.ID, ExpectedVersion: second.Version + 1, ResolutionSummary: "second released", ReleaseConfirmed: true},
		},
	})
	if err == nil {
		t.Fatal("compound command unexpectedly succeeded with stale blocker version")
	}

	issueAfter, err := service.GetIssue(ctx, outcome.Scope(), issue.ID)
	if err != nil {
		t.Fatal(err)
	}
	firstAfter, err := service.GetBlocker(ctx, outcome.Scope(), first.ID)
	if err != nil {
		t.Fatal(err)
	}
	secondAfter, err := service.GetBlocker(ctx, outcome.Scope(), second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if issueAfter.Value.Lifecycle != domain.IssueLifecycleOpen {
		t.Fatalf("issue lifecycle after rollback = %q", issueAfter.Value.Lifecycle)
	}
	if firstAfter.Value.Lifecycle != domain.BlockerLifecycleActive || secondAfter.Value.Lifecycle != domain.BlockerLifecycleActive {
		t.Fatalf("blockers after rollback = %q/%q", firstAfter.Value.Lifecycle, secondAfter.Value.Lifecycle)
	}
}

func TestResolveIssueAndBlockersRequiresExplicitReleaseAndOnlyResolvesListedBlockers(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	work := createTodoWork(t, service, outcome.Scope(), "explicit release")
	reported, err := service.ReportIssueWithBlocker(ctx, commandContext(), application.ReportIssueWithBlockerCommand{
		Scope:              outcome.Scope(),
		Title:              "Explicit root cause",
		Severity:           domain.IssueSeverityMajor,
		BlockedRef:         work.Ref(),
		BlockerDescription: "first",
	})
	if err != nil {
		t.Fatal(err)
	}
	issue := reported.Value.Issue
	first := reported.Value.Blocker
	cause := issue.Ref()
	secondResult, err := service.CreateBlocker(ctx, commandContext(), application.CreateBlockerCommand{
		Scope: outcome.Scope(), BlockedRef: work.Ref(), CauseRef: &cause, Description: "second",
	})
	if err != nil {
		t.Fatal(err)
	}
	second := secondResult.Value

	_, err = service.ResolveIssueAndBlockers(ctx, commandContext(), application.ResolveIssueAndBlockersCommand{
		Scope:                  outcome.Scope(),
		IssueID:                issue.ID,
		IssueExpectedVersion:   issue.Version,
		IssueResolutionSummary: "fixed",
		Blockers: []application.BlockerResolution{{
			BlockerID: first.ID, ExpectedVersion: first.Version, ResolutionSummary: "release", ReleaseConfirmed: false,
		}},
	})
	if err == nil {
		t.Fatal("compound resolution accepted missing release confirmation")
	}

	resolved, err := service.ResolveIssueAndBlockers(ctx, commandContext(), application.ResolveIssueAndBlockersCommand{
		Scope:                  outcome.Scope(),
		IssueID:                issue.ID,
		IssueExpectedVersion:   issue.Version,
		IssueResolutionSummary: "fixed",
		Blockers: []application.BlockerResolution{{
			BlockerID: first.ID, ExpectedVersion: first.Version, ResolutionSummary: "first released", ReleaseConfirmed: true,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Value.Issue.Lifecycle != domain.IssueLifecycleResolved || len(resolved.Value.Blockers) != 1 {
		t.Fatalf("resolved result = %#v", resolved.Value)
	}
	remaining, err := service.GetBlocker(ctx, outcome.Scope(), second.ID)
	if err != nil {
		t.Fatal(err)
	}
	if remaining.Value.Lifecycle != domain.BlockerLifecycleActive {
		t.Fatalf("unlisted blocker lifecycle = %q, want active", remaining.Value.Lifecycle)
	}
	state, _, err := service.GetBlockingState(ctx, work.Ref())
	if err != nil {
		t.Fatal(err)
	}
	if !state.IsBlocked || len(state.ActiveBlockers) != 1 || state.ActiveBlockers[0].Blocker.ID != second.ID {
		t.Fatalf("remaining blocking state = %#v", state)
	}
}

func blockingState(states []application.BlockingState, ref domain.EntityRef) (application.BlockingState, bool) {
	for _, state := range states {
		if state.Ref == ref {
			return state, true
		}
	}
	return application.BlockingState{}, false
}
