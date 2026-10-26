package application

import (
	"context"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type duplicateChainIssues map[domain.ID]domain.Issue

func (r duplicateChainIssues) Get(_ context.Context, scope domain.Scope, id domain.ID) (domain.Issue, error) {
	issue, ok := r[id]
	if !ok || issue.Scope != scope {
		return domain.Issue{}, domain.NewError(domain.ErrorCodeNotFound, "issue not found")
	}
	return issue, nil
}

func (r duplicateChainIssues) ListByOutcome(context.Context, domain.Scope) ([]domain.Issue, error) {
	return nil, nil
}

func (r duplicateChainIssues) ListPendingIssuesByObjective(context.Context, domain.Scope, domain.ID, string, int) ([]domain.Issue, int, error) {
	return nil, 0, nil
}

func (r duplicateChainIssues) ListDuplicateIssuesByObjective(context.Context, domain.Scope, domain.ID, string, int) ([]domain.Issue, int, error) {
	return nil, 0, nil
}

func (r duplicateChainIssues) Insert(context.Context, domain.Issue) error { return nil }

func (r duplicateChainIssues) Save(context.Context, domain.Issue, domain.Version) error { return nil }

func duplicateChainIssue(t *testing.T, scope domain.Scope, id, objective domain.ID) domain.Issue {
	t.Helper()
	issue, err := domain.NewIssueForObjective(
		id, scope, &objective, "Issue "+id.String(), "", domain.IssueSeverityMinor, nil,
		domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "test", ID: "reporter"},
		time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	return issue
}

func TestIssueDuplicateChainGateRequiresClosedCanonicalIssue(t *testing.T) {
	scope := domain.Scope{
		NamespaceID: domain.MustParseID("0199f210-0000-7000-8000-000000000001"),
		OutcomeID:   domain.MustParseID("0199f210-0000-7000-8000-000000000002"),
	}
	objectiveID := domain.MustParseID("0199f210-0000-7000-8000-000000000003")
	first := duplicateChainIssue(t, scope, domain.MustParseID("0199f210-0000-7000-8000-000000000004"), objectiveID)
	second := duplicateChainIssue(t, scope, domain.MustParseID("0199f210-0000-7000-8000-000000000005"), objectiveID)
	canonical := duplicateChainIssue(t, scope, domain.MustParseID("0199f210-0000-7000-8000-000000000006"), objectiveID)
	now := time.Date(2026, 10, 20, 12, 1, 0, 0, time.UTC)
	if err := first.MarkDuplicate(second.ID, now); err != nil {
		t.Fatal(err)
	}
	if err := second.MarkDuplicate(canonical.ID, now); err != nil {
		t.Fatal(err)
	}
	if err := canonical.Resolve("canonical issue was fixed", now); err != nil {
		t.Fatal(err)
	}
	repo := duplicateChainIssues{first.ID: first, second.ID: second, canonical.ID: canonical}

	closed, terminal, err := issueDuplicateChainClosed(context.Background(), repo, scope, first)
	if err != nil || !closed || terminal.ID != canonical.ID || terminal.Lifecycle != domain.IssueLifecycleResolved {
		t.Fatalf("resolved canonical chain = closed:%v terminal:%#v err:%v", closed, terminal, err)
	}

	canonical.Lifecycle = domain.IssueLifecycleOpen
	canonical.ResolutionSummary = ""
	canonical.Version++
	repo[canonical.ID] = canonical
	closed, terminal, err = issueDuplicateChainClosed(context.Background(), repo, scope, first)
	if err != nil || closed || terminal.ID != canonical.ID {
		t.Fatalf("open canonical chain = closed:%v terminal:%#v err:%v", closed, terminal, err)
	}
}

func TestIssueDuplicateChainGateFailsClosedForMissingTargetAndCycle(t *testing.T) {
	scope := domain.Scope{
		NamespaceID: domain.MustParseID("0199f211-0000-7000-8000-000000000001"),
		OutcomeID:   domain.MustParseID("0199f211-0000-7000-8000-000000000002"),
	}
	objectiveID := domain.MustParseID("0199f211-0000-7000-8000-000000000003")
	first := duplicateChainIssue(t, scope, domain.MustParseID("0199f211-0000-7000-8000-000000000004"), objectiveID)
	second := duplicateChainIssue(t, scope, domain.MustParseID("0199f211-0000-7000-8000-000000000005"), objectiveID)
	now := time.Date(2026, 10, 20, 12, 1, 0, 0, time.UTC)
	missing := domain.MustParseID("0199f211-0000-7000-8000-000000000006")
	if err := first.MarkDuplicate(missing, now); err != nil {
		t.Fatal(err)
	}
	repo := duplicateChainIssues{first.ID: first}
	if closed, _, err := issueDuplicateChainClosed(context.Background(), repo, scope, first); err == nil || closed {
		t.Fatalf("missing duplicate target did not fail closed: closed=%v err=%v", closed, err)
	}

	first = duplicateChainIssue(t, scope, first.ID, objectiveID)
	second = duplicateChainIssue(t, scope, second.ID, objectiveID)
	if err := first.MarkDuplicate(second.ID, now); err != nil {
		t.Fatal(err)
	}
	if err := second.MarkDuplicate(first.ID, now); err != nil {
		t.Fatal(err)
	}
	repo = duplicateChainIssues{first.ID: first, second.ID: second}
	if closed, _, err := issueDuplicateChainClosed(context.Background(), repo, scope, first); err == nil || closed {
		t.Fatalf("duplicate cycle did not fail closed: closed=%v err=%v", closed, err)
	}
}
