package domain

import (
	"reflect"
	"testing"
	"time"
)

func assignmentIssue(t *testing.T, now time.Time) Issue {
	t.Helper()
	scope := Scope{
		NamespaceID: MustParseID("0199f150-0000-7000-8000-000000000001"),
		OutcomeID:   MustParseID("0199f150-0000-7000-8000-000000000002"),
	}
	objectiveID := MustParseID("0199f150-0000-7000-8000-000000000003")
	issue, err := NewIssueForObjective(
		MustParseID("0199f150-0000-7000-8000-000000000004"),
		scope,
		&objectiveID,
		"Coordinate the issue",
		"Keep its assignment independent of execution authority.",
		IssueSeverityMajor,
		nil,
		ActorRef{Kind: ActorKindHuman, Provider: "local", ID: "reporter"},
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	return issue
}

func assignmentActor(id string) ActorRef {
	return ActorRef{Kind: ActorKindAgent, Provider: "test", ID: id}
}

func TestIssueAssignAndReassignPreserveLifecycleAndReporter(t *testing.T) {
	created := time.Date(2026, 10, 20, 12, 0, 0, 0, time.FixedZone("test", -4*60*60))
	issue := assignmentIssue(t, created)
	reporter := issue.ReportedBy
	assignedAt := created.Add(time.Minute)
	first := assignmentActor("agent-a")
	if err := issue.Assign(first, assignedAt); err != nil {
		t.Fatal(err)
	}
	if issue.AssigneeRef == nil || *issue.AssigneeRef != first {
		t.Fatalf("assignee after assign = %#v, want %#v", issue.AssigneeRef, first)
	}
	if issue.Lifecycle != IssueLifecycleOpen || issue.ReportedBy != reporter {
		t.Fatalf("assignment changed lifecycle/reporter: %#v", issue)
	}
	if issue.Version != 2 || !issue.UpdatedAt.Equal(assignedAt.UTC()) || issue.UpdatedAt.Location() != time.UTC {
		t.Fatalf("assignment version/time = %d/%v, want 2/%v UTC", issue.Version, issue.UpdatedAt, assignedAt.UTC())
	}

	reassignedAt := created.Add(2 * time.Minute)
	second := assignmentActor("agent-b")
	if err := issue.Assign(second, reassignedAt); err != nil {
		t.Fatal(err)
	}
	if issue.AssigneeRef == nil || *issue.AssigneeRef != second {
		t.Fatalf("assignee after reassignment = %#v, want %#v", issue.AssigneeRef, second)
	}
	if issue.Lifecycle != IssueLifecycleOpen || issue.ReportedBy != reporter {
		t.Fatalf("reassignment changed lifecycle/reporter: %#v", issue)
	}
	if issue.Version != 3 || !issue.UpdatedAt.Equal(reassignedAt.UTC()) {
		t.Fatalf("reassignment version/time = %d/%v, want 3/%v", issue.Version, issue.UpdatedAt, reassignedAt.UTC())
	}

	unchanged := issue
	if err := issue.Assign(second, created.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(issue, unchanged) {
		t.Fatalf("assigning the current assignee should be a no-op: before=%#v after=%#v", unchanged, issue)
	}
}

func TestIssueUnassignOpenAllowsBlankRationale(t *testing.T) {
	now := time.Date(2026, 10, 20, 13, 0, 0, 0, time.UTC)
	issue := assignmentIssue(t, now)
	if err := issue.Assign(assignmentActor("agent-a"), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	version := issue.Version
	updatedAt := now.Add(2 * time.Minute)
	if err := issue.Unassign("   ", updatedAt); err != nil {
		t.Fatal(err)
	}
	if issue.AssigneeRef != nil || issue.Lifecycle != IssueLifecycleOpen {
		t.Fatalf("open Issue after unassign = %#v", issue)
	}
	if issue.Version != version+1 || !issue.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("open unassign version/time = %d/%v, want %d/%v", issue.Version, issue.UpdatedAt, version+1, updatedAt)
	}
}

func TestIssueUnassignInvestigatingRequiresRationale(t *testing.T) {
	now := time.Date(2026, 10, 20, 14, 0, 0, 0, time.UTC)
	issue := assignmentIssue(t, now)
	if err := issue.Assign(assignmentActor("agent-a"), now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := issue.Investigate(now.Add(2 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	before := issue
	if err := issue.Unassign(" \t ", now.Add(3*time.Minute)); err == nil {
		t.Fatal("investigating Issue accepted a blank unassignment rationale")
	}
	if !reflect.DeepEqual(issue, before) {
		t.Fatalf("failed unassignment mutated Issue: before=%#v after=%#v", before, issue)
	}

	reporter := issue.ReportedBy
	updatedAt := now.Add(4 * time.Minute)
	if err := issue.Unassign("reassign investigation to the service desk", updatedAt); err != nil {
		t.Fatal(err)
	}
	if issue.AssigneeRef != nil || issue.Lifecycle != IssueLifecycleInvestigating || issue.ReportedBy != reporter {
		t.Fatalf("investigating unassign changed unrelated state: %#v", issue)
	}
	if issue.Version != before.Version+1 || !issue.UpdatedAt.Equal(updatedAt) {
		t.Fatalf("investigating unassign version/time = %d/%v, want %d/%v", issue.Version, issue.UpdatedAt, before.Version+1, updatedAt)
	}
	unchanged := issue
	if err := issue.Unassign(" ", updatedAt.Add(time.Minute)); err == nil {
		t.Fatal("investigating Issue accepted a blank rationale when already unassigned")
	}
	if !reflect.DeepEqual(issue, unchanged) {
		t.Fatalf("rejected no-op unassignment mutated Issue: before=%#v after=%#v", unchanged, issue)
	}
}

func TestIssueAssignmentOperationsRejectTerminalIssues(t *testing.T) {
	now := time.Date(2026, 10, 20, 15, 0, 0, 0, time.UTC)
	terminal := []struct {
		name   string
		change func(*Issue) error
	}{
		{name: "resolved", change: func(issue *Issue) error { return issue.Resolve("resolved", now.Add(time.Minute)) }},
		{name: "wont_fix", change: func(issue *Issue) error { return issue.MarkWontFix("wont fix", now.Add(time.Minute)) }},
		{name: "duplicate", change: func(issue *Issue) error {
			return issue.MarkDuplicate(MustParseID("0199f150-0000-7000-8000-000000000005"), now.Add(time.Minute))
		}},
	}
	for _, test := range terminal {
		t.Run(test.name, func(t *testing.T) {
			issue := assignmentIssue(t, now)
			if err := issue.Assign(assignmentActor("agent-a"), now); err != nil {
				t.Fatal(err)
			}
			if err := test.change(&issue); err != nil {
				t.Fatal(err)
			}
			before := issue
			if err := issue.Assign(assignmentActor("agent-b"), now.Add(2*time.Minute)); err == nil {
				t.Fatal("terminal Issue accepted assignment")
			}
			if !reflect.DeepEqual(issue, before) {
				t.Fatalf("failed terminal assignment mutated Issue: before=%#v after=%#v", before, issue)
			}
			if err := issue.Unassign("reassignment requested", now.Add(2*time.Minute)); err == nil {
				t.Fatal("terminal Issue accepted unassignment")
			}
			if !reflect.DeepEqual(issue, before) {
				t.Fatalf("failed terminal unassignment mutated Issue: before=%#v after=%#v", before, issue)
			}
		})
	}
}

func TestIssueAssignRejectsInvalidActor(t *testing.T) {
	now := time.Date(2026, 10, 20, 16, 0, 0, 0, time.UTC)
	issue := assignmentIssue(t, now)
	before := issue
	if err := issue.Assign(ActorRef{}, now.Add(time.Minute)); err == nil {
		t.Fatal("invalid Issue assignee was accepted")
	}
	if !reflect.DeepEqual(issue, before) {
		t.Fatalf("invalid assignment mutated Issue: before=%#v after=%#v", before, issue)
	}
}
