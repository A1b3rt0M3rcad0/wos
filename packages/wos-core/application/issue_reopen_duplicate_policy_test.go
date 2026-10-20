package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func createIssueForPolicyTest(t *testing.T, service *application.Service, scope domain.Scope, objectiveID domain.ID, title string) domain.Issue {
	t.Helper()
	created, err := service.CreateIssue(context.Background(), commandContext(), application.CreateIssueCommand{
		Scope: scope, ObjectiveID: objectiveID, Title: title, Severity: domain.IssueSeverityMinor,
	})
	if err != nil {
		t.Fatal(err)
	}
	return created.Value
}

func achieveIssueOwnerForTest(t *testing.T, service *application.Service, scope domain.Scope, objectiveID domain.ID) domain.Objective {
	t.Helper()
	loaded, err := service.GetObjective(context.Background(), scope, objectiveID)
	if err != nil {
		t.Fatal(err)
	}
	objective := loaded.Value
	criterion, err := service.AddCriterion(context.Background(), commandContext(), application.AddCriterionCommand{
		Owner: objective.Ref(), ExpectedVersion: objective.Version, Title: "Issue disposition recorded",
		Required: true, VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	objective.Version++
	started, err := service.StartObjective(context.Background(), commandContext(), application.StartObjectiveCommand{
		Scope: scope, ObjectiveID: objective.ID, ExpectedVersion: objective.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	objective = started.Value
	_, err = service.AttestCriterion(context.Background(), commandContext(), application.AttestCriterionCommand{
		Owner: objective.Ref(), CriterionID: criterion.Value.ID,
		CriterionRevision: domain.InitialCriterionRevision, ExpectedVersion: objective.Version,
		Result: domain.AssessmentResultMet, Rationale: "triage was reviewed",
	})
	if err != nil {
		t.Fatal(err)
	}
	objective.Version++
	achieved, err := service.AchieveObjective(context.Background(), commandContext(), application.AchieveObjectiveCommand{
		Scope: scope, ObjectiveID: objective.ID, ExpectedVersion: objective.Version, Reason: "criteria verified",
	})
	if err != nil {
		t.Fatal(err)
	}
	return achieved.Value
}

func TestReopenIssueRequiresReopeningAchievedObjectiveAndIsAtomic(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	outcome := setupActiveOutcome(t, service)
	ownerID := issueObjectiveForTest(t, service, outcome.Scope())
	issue := createIssueForPolicyTest(t, service, outcome.Scope(), ownerID, "Resolved issue under Objective")
	resolved, err := service.ResolveIssue(ctx, commandContext(), application.ResolveIssueCommand{
		Scope: outcome.Scope(), IssueID: issue.ID, ExpectedVersion: issue.Version, ResolutionSummary: "fixed and verified",
	})
	if err != nil {
		t.Fatal(err)
	}
	owner := achieveIssueOwnerForTest(t, service, outcome.Scope(), ownerID)
	if owner.Lifecycle != domain.ObjectiveLifecycleAchieved {
		t.Fatalf("test owner lifecycle = %q", owner.Lifecycle)
	}

	before, err := service.GetIssue(ctx, outcome.Scope(), issue.ID)
	if err != nil {
		t.Fatal(err)
	}
	ownerBefore, err := service.GetObjective(ctx, outcome.Scope(), owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := len(store.SnapshotDomainEvents(outcome.Scope()))
	_, err = service.ReopenIssue(ctx, commandContext(), application.ReopenIssueCommand{
		Scope: outcome.Scope(), IssueID: issue.ID, ExpectedVersion: resolved.Value.Version,
	})
	if err == nil {
		t.Fatal("Issue under an achieved Objective was reopened")
	}
	var coded *domain.Error
	if !errors.As(err, &coded) || coded.Code != domain.ErrorCodePreconditionFailed || coded.Details == nil {
		t.Fatalf("reopen error = %#v, want structured precondition failure", err)
	}
	if coded.Details.ReasonCode != "issue_objective_achieved" || coded.Details.SubjectRef == nil || *coded.Details.SubjectRef != before.Value.Ref() {
		t.Fatalf("reopen error details = %#v", coded.Details)
	}
	if len(coded.Details.Remedies) != 1 || coded.Details.Remedies[0].ActionID != "reopen_objective" || coded.Details.Remedies[0].SubjectRef == nil || *coded.Details.Remedies[0].SubjectRef != owner.Ref() {
		t.Fatalf("reopen remedies = %#v", coded.Details.Remedies)
	}

	after, err := service.GetIssue(ctx, outcome.Scope(), issue.ID)
	if err != nil {
		t.Fatal(err)
	}
	ownerAfter, err := service.GetObjective(ctx, outcome.Scope(), owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Value.Version != before.Value.Version || after.Value.Lifecycle != before.Value.Lifecycle || after.Value.ResolutionSummary != before.Value.ResolutionSummary {
		t.Fatalf("rejected reopen changed Issue: before=%#v after=%#v", before.Value, after.Value)
	}
	if ownerAfter.Value.Version != ownerBefore.Value.Version || ownerAfter.Value.Lifecycle != domain.ObjectiveLifecycleAchieved {
		t.Fatalf("rejected reopen changed Objective: before=%#v after=%#v", ownerBefore.Value, ownerAfter.Value)
	}
	if after.OutcomeRevision != before.OutcomeRevision || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsBefore {
		t.Fatalf("rejected reopen changed revision or events: revision %d -> %d, events %d -> %d", before.OutcomeRevision, after.OutcomeRevision, eventsBefore, len(store.SnapshotDomainEvents(outcome.Scope())))
	}
}

func TestReopenIssueRemainsAvailableWhileOwnerObjectiveIsOpen(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	ownerID := issueObjectiveForTest(t, service, outcome.Scope())
	issue := createIssueForPolicyTest(t, service, outcome.Scope(), ownerID, "Reopen under active Objective")
	resolved, err := service.ResolveIssue(ctx, commandContext(), application.ResolveIssueCommand{
		Scope: outcome.Scope(), IssueID: issue.ID, ExpectedVersion: issue.Version, ResolutionSummary: "temporary fix",
	})
	if err != nil {
		t.Fatal(err)
	}
	reopened, err := service.ReopenIssue(ctx, commandContext(), application.ReopenIssueCommand{
		Scope: outcome.Scope(), IssueID: issue.ID, ExpectedVersion: resolved.Value.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Value.Lifecycle != domain.IssueLifecycleOpen || reopened.Value.ResolutionSummary != "" || reopened.Value.Version != resolved.Value.Version+1 {
		t.Fatalf("reopened Issue = %#v", reopened.Value)
	}
}

func TestReopenIssueUsesExpectedVersionBeforeObjectivePolicy(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	outcome := setupActiveOutcome(t, service)
	ownerID := issueObjectiveForTest(t, service, outcome.Scope())
	issue := createIssueForPolicyTest(t, service, outcome.Scope(), ownerID, "Stale reopen")
	resolved, err := service.ResolveIssue(ctx, commandContext(), application.ResolveIssueCommand{
		Scope: outcome.Scope(), IssueID: issue.ID, ExpectedVersion: issue.Version, ResolutionSummary: "fixed",
	})
	if err != nil {
		t.Fatal(err)
	}
	achieveIssueOwnerForTest(t, service, outcome.Scope(), ownerID)
	before, err := service.GetIssue(ctx, outcome.Scope(), issue.ID)
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := len(store.SnapshotDomainEvents(outcome.Scope()))
	_, err = service.ReopenIssue(ctx, commandContext(), application.ReopenIssueCommand{
		Scope: outcome.Scope(), IssueID: issue.ID, ExpectedVersion: resolved.Value.Version - 1,
	})
	var coded *domain.Error
	if !errors.As(err, &coded) || coded.Code != domain.ErrorCodeVersionConflict {
		t.Fatalf("stale reopen error = %v, want version conflict", err)
	}
	after, err := service.GetIssue(ctx, outcome.Scope(), issue.ID)
	if err != nil {
		t.Fatal(err)
	}
	if after.Value.Version != before.Value.Version || after.Value.Lifecycle != before.Value.Lifecycle || after.OutcomeRevision != before.OutcomeRevision || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsBefore {
		t.Fatal("stale reopen changed Issue, revision, or events")
	}
}

func TestMarkIssueDuplicateRejectsSelfAndMissingTargetWithoutMutation(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	outcome := setupActiveOutcome(t, service)
	ownerID := issueObjectiveForTest(t, service, outcome.Scope())
	issue := createIssueForPolicyTest(t, service, outcome.Scope(), ownerID, "Source issue")
	missing := id("0199e2ff-0000-7000-8000-000000000088")
	for _, test := range []struct {
		name   string
		target domain.ID
	}{
		{name: "self", target: issue.ID},
		{name: "missing target", target: missing},
	} {
		t.Run(test.name, func(t *testing.T) {
			before, err := service.GetIssue(ctx, outcome.Scope(), issue.ID)
			if err != nil {
				t.Fatal(err)
			}
			eventsBefore := len(store.SnapshotDomainEvents(outcome.Scope()))
			_, err = service.MarkIssueDuplicate(ctx, commandContext(), application.MarkIssueDuplicateCommand{
				Scope: outcome.Scope(), IssueID: issue.ID, ExpectedVersion: before.Value.Version, DuplicateOfID: test.target,
			})
			if err == nil {
				t.Fatal("invalid duplicate target was accepted")
			}
			after, err := service.GetIssue(ctx, outcome.Scope(), issue.ID)
			if err != nil {
				t.Fatal(err)
			}
			if after.Value.Version != before.Value.Version || after.Value.Lifecycle != before.Value.Lifecycle || after.OutcomeRevision != before.OutcomeRevision || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsBefore {
				t.Fatalf("rejected duplicate changed state: before=%#v after=%#v", before, after)
			}
		})
	}
}
