package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestIssueAffectedRefsMustResolveInEveryStorageAdapter(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	owner := issueObjectiveForTest(t, service, outcome.Scope())
	missing := domain.EntityRef{
		Scope: outcome.Scope(),
		Kind:  domain.EntityKindWorkItem,
		ID:    id("0199e2ff-0000-7000-8000-000000000401"),
	}

	_, err := service.CreateIssue(ctx, commandContext(), application.CreateIssueCommand{
		Scope:        outcome.Scope(),
		ObjectiveID:  owner,
		Title:        "Invalid affected reference",
		Severity:     domain.IssueSeverityMinor,
		AffectedRefs: []domain.EntityRef{missing},
	})
	if err == nil {
		t.Fatal("CreateIssue accepted an affected_ref that does not exist")
	}

	created, err := service.CreateIssue(ctx, commandContext(), application.CreateIssueCommand{
		Scope:       outcome.Scope(),
		ObjectiveID: owner,
		Title:       "Valid issue",
		Severity:    domain.IssueSeverityMinor,
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

func TestCreateIssueRequiresExistingNonterminalObjectiveOwner(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	valid := issueObjectiveForTest(t, service, outcome.Scope())
	if _, err := service.CreateIssue(ctx, commandContext(), application.CreateIssueCommand{
		Scope: outcome.Scope(), Title: "Missing owner", Severity: domain.IssueSeverityMinor,
	}); err == nil {
		t.Fatal("CreateIssue accepted a missing Objective owner")
	}
	if _, err := service.ReportIssueWithBlocker(ctx, commandContext(), application.ReportIssueWithBlockerCommand{
		Scope: outcome.Scope(), Title: "Missing compound owner", Severity: domain.IssueSeverityMinor,
		BlockedRef: domain.EntityRef{Scope: outcome.Scope(), Kind: domain.EntityKindObjective, ID: valid},
	}); err == nil {
		t.Fatal("ReportIssueWithBlocker accepted a missing Objective owner")
	}
	if _, err := service.CreateIssue(ctx, commandContext(), application.CreateIssueCommand{
		Scope: outcome.Scope(), ObjectiveID: id("0199e2ff-0000-7000-8000-000000000499"),
		Title: "Unknown owner", Severity: domain.IssueSeverityMinor,
	}); err == nil {
		t.Fatal("CreateIssue accepted an Objective that does not exist in the Outcome")
	}
	created, err := service.CreateIssue(ctx, commandContext(), application.CreateIssueCommand{
		Scope: outcome.Scope(), ObjectiveID: valid, Title: "Valid owner", Severity: domain.IssueSeverityMinor,
	})
	if err != nil {
		t.Fatalf("CreateIssue rejected a nonterminal same-Outcome Objective: %v", err)
	}
	if created.Value.ObjectiveID == nil || *created.Value.ObjectiveID != valid {
		t.Fatalf("created Issue owner = %v, want %s", created.Value.ObjectiveID, valid)
	}

	terminal, err := service.CreateObjective(ctx, commandContext(), application.CreateObjectiveCommand{
		Scope: outcome.Scope(), Title: "Closed Objective", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	terminal, err = service.CancelObjective(ctx, commandContext(), application.CancelObjectiveCommand{
		Scope: outcome.Scope(), ObjectiveID: terminal.Value.ID, ExpectedVersion: terminal.Value.Version, Reason: "No longer in scope",
	})
	if err != nil {
		t.Fatal(err)
	}
	before, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.CreateIssue(ctx, commandContext(), application.CreateIssueCommand{
		Scope: outcome.Scope(), ObjectiveID: terminal.Value.ID, Title: "Issue on a closed Objective", Severity: domain.IssueSeverityMinor,
	})
	var typed *domain.Error
	if !errors.As(err, &typed) || typed.Code != domain.ErrorCodePreconditionFailed || typed.Details == nil || typed.Details.ReasonCode != "objective_terminal" {
		t.Fatalf("terminal Objective owner error = %#v, want objective_terminal precondition", err)
	}
	after, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if after.OutcomeRevision != before.OutcomeRevision || len(after.Issues) != len(before.Issues) {
		t.Fatalf("rejected Issue creation mutated state: before revision/issues=%d/%d, after=%d/%d", before.OutcomeRevision, len(before.Issues), after.OutcomeRevision, len(after.Issues))
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
