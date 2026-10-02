package domain

import (
	"testing"
	"time"
)

func TestIssueLifecycleAndDuplicateRules(t *testing.T) {
	now := time.Date(2026, 10, 2, 16, 0, 0, 0, time.UTC)
	scope := Scope{
		NamespaceID: MustParseID("0199f000-0000-7000-8000-000000000001"),
		OutcomeID:   MustParseID("0199f000-0000-7000-8000-000000000002"),
	}
	reporter := ActorRef{Kind: ActorKindHuman, Provider: "local", ID: "tester"}
	affected := EntityRef{Scope: scope, Kind: EntityKindWorkItem, ID: MustParseID("0199f000-0000-7000-8000-000000000003")}
	issue, err := NewIssue(
		MustParseID("0199f000-0000-7000-8000-000000000004"),
		scope,
		"Unexpected result",
		"Observed mismatch",
		IssueSeverityMajor,
		[]EntityRef{affected},
		reporter,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := issue.Investigate(now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if issue.Lifecycle != IssueLifecycleInvestigating || issue.Version != 2 {
		t.Fatalf("issue after investigate = %#v", issue)
	}
	if err := issue.Resolve("Root cause corrected", now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if issue.Lifecycle != IssueLifecycleResolved || issue.ResolutionSummary == "" {
		t.Fatalf("issue after resolve = %#v", issue)
	}
	if err := issue.Reopen(now.Add(3 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	if issue.Lifecycle != IssueLifecycleOpen || issue.ResolutionSummary != "" {
		t.Fatalf("issue after reopen = %#v", issue)
	}
	target := MustParseID("0199f000-0000-7000-8000-000000000005")
	if err := issue.MarkDuplicate(target, now.Add(4*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if issue.Lifecycle != IssueLifecycleDuplicate || issue.DuplicateOfIssueID == nil || *issue.DuplicateOfIssueID != target {
		t.Fatalf("issue duplicate state = %#v", issue)
	}
}

func TestIssueRejectsCrossScopeAffectedRef(t *testing.T) {
	now := time.Date(2026, 10, 2, 16, 0, 0, 0, time.UTC)
	scope := Scope{
		NamespaceID: MustParseID("0199f001-0000-7000-8000-000000000001"),
		OutcomeID:   MustParseID("0199f001-0000-7000-8000-000000000002"),
	}
	other := Scope{NamespaceID: scope.NamespaceID, OutcomeID: MustParseID("0199f001-0000-7000-8000-000000000003")}
	_, err := NewIssue(
		MustParseID("0199f001-0000-7000-8000-000000000004"),
		scope,
		"Cross scope",
		"",
		IssueSeverityMinor,
		[]EntityRef{{Scope: other, Kind: EntityKindWorkItem, ID: MustParseID("0199f001-0000-7000-8000-000000000005")}},
		ActorRef{Kind: ActorKindHuman, Provider: "local", ID: "tester"},
		now,
	)
	if err == nil {
		t.Fatal("cross-scope issue affected_ref unexpectedly accepted")
	}
}

func TestBlockerDefaultsAndClosureAreExplicit(t *testing.T) {
	now := time.Date(2026, 10, 2, 16, 0, 0, 0, time.UTC)
	scope := Scope{
		NamespaceID: MustParseID("0199f002-0000-7000-8000-000000000001"),
		OutcomeID:   MustParseID("0199f002-0000-7000-8000-000000000002"),
	}
	objective := EntityRef{Scope: scope, Kind: EntityKindObjective, ID: MustParseID("0199f002-0000-7000-8000-000000000003")}
	issue := EntityRef{Scope: scope, Kind: EntityKindIssue, ID: MustParseID("0199f002-0000-7000-8000-000000000004")}
	blocker, err := NewBlocker(
		MustParseID("0199f002-0000-7000-8000-000000000005"),
		objective,
		&issue,
		nil,
		"",
		"",
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if blocker.Propagation != BlockerPropagationSubtree {
		t.Fatalf("objective default propagation = %q, want subtree", blocker.Propagation)
	}
	if err := blocker.Resolve("Verified that execution can continue", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if blocker.Lifecycle != BlockerLifecycleResolved || blocker.ResolvedAt == nil {
		t.Fatalf("blocker after resolve = %#v", blocker)
	}
}

func TestWorkItemBlockerRejectsSubtreePropagation(t *testing.T) {
	now := time.Date(2026, 10, 2, 16, 0, 0, 0, time.UTC)
	scope := Scope{
		NamespaceID: MustParseID("0199f003-0000-7000-8000-000000000001"),
		OutcomeID:   MustParseID("0199f003-0000-7000-8000-000000000002"),
	}
	work := EntityRef{Scope: scope, Kind: EntityKindWorkItem, ID: MustParseID("0199f003-0000-7000-8000-000000000003")}
	_, err := NewBlocker(
		MustParseID("0199f003-0000-7000-8000-000000000004"),
		work,
		nil,
		nil,
		"External dependency unavailable",
		BlockerPropagationSubtree,
		now,
	)
	if err == nil {
		t.Fatal("work_item subtree blocker unexpectedly accepted")
	}
}

func TestBlockerRequiresCauseOrDescription(t *testing.T) {
	now := time.Date(2026, 10, 2, 16, 0, 0, 0, time.UTC)
	scope := Scope{
		NamespaceID: MustParseID("0199f004-0000-7000-8000-000000000001"),
		OutcomeID:   MustParseID("0199f004-0000-7000-8000-000000000002"),
	}
	_, err := NewBlocker(
		MustParseID("0199f004-0000-7000-8000-000000000003"),
		EntityRef{Scope: scope, Kind: EntityKindOutcome, ID: scope.OutcomeID},
		nil,
		nil,
		"",
		"",
		now,
	)
	if err == nil {
		t.Fatal("blocker without cause or description unexpectedly accepted")
	}
}
