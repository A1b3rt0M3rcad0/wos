package application_test

import (
	"context"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestIssueAffectedRefsMustResolveInEveryStorageAdapter(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	missing := domain.EntityRef{
		Scope: outcome.Scope(),
		Kind:  domain.EntityKindWorkItem,
		ID:    id("0199e2ff-0000-7000-8000-000000000401"),
	}

	_, err := service.CreateIssue(ctx, commandContext(), application.CreateIssueCommand{
		Scope:        outcome.Scope(),
		Title:        "Invalid affected reference",
		Severity:     domain.IssueSeverityMinor,
		AffectedRefs: []domain.EntityRef{missing},
	})
	if err == nil {
		t.Fatal("CreateIssue accepted an affected_ref that does not exist")
	}

	created, err := service.CreateIssue(ctx, commandContext(), application.CreateIssueCommand{
		Scope:    outcome.Scope(),
		Title:    "Valid issue",
		Severity: domain.IssueSeverityMinor,
	})
	if err != nil {
		t.Fatal(err)
	}
	refs := []domain.EntityRef{missing}
	_, err = service.UpdateIssue(ctx, commandContext(), application.UpdateIssueCommand{
		Scope:           outcome.Scope(),
		IssueID:         created.Value.ID,
		ExpectedVersion: created.Value.Version,
		AffectedRefs:    &refs,
	})
	if err == nil {
		t.Fatal("UpdateIssue accepted an affected_ref that does not exist")
	}

	after, err := service.GetIssue(ctx, outcome.Scope(), created.Value.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Value.Version != created.Value.Version || len(after.Value.AffectedRefs) != 0 {
		t.Fatalf("failed affected_ref update mutated issue: %#v", after.Value)
	}
}

func TestDecisionBlockerCauseWaitsForPersistedDecisionSupport(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	work := createTodoWork(t, service, outcome.Scope(), "decision cause")
	decision := domain.EntityRef{
		Scope: outcome.Scope(),
		Kind:  domain.EntityKindDecision,
		ID:    id("0199e2ff-0000-7000-8000-000000000402"),
	}

	_, err := service.CreateBlocker(ctx, commandContext(), application.CreateBlockerCommand{
		Scope:       outcome.Scope(),
		BlockedRef:  work.Ref(),
		CauseRef:    &decision,
		Description: "Decision support is not implemented yet.",
	})
	if err == nil {
		t.Fatal("CreateBlocker accepted a Decision cause without a persisted Decision aggregate")
	}
}
