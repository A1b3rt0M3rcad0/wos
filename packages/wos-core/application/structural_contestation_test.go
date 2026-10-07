package application_test

import (
	"context"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"testing"
)

func TestTerminalOutcomeRequiresExplicitReopenBeforeChangingObligations(t *testing.T) {
	ctx := context.Background()
	s, _ := newWave09Service(t)
	cc := commandContext()
	created, err := s.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{NamespaceID: domain.MustParseID("0199d060-0000-7000-8000-000000000001"), Title: "Structural proof", DesiredState: "Evidence remains reproducible", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	o := created.Value
	criterion, err := s.AddCriterion(ctx, cc, application.AddCriterionCommand{Owner: o.Ref(), ExpectedVersion: o.Version, Title: "Initial obligation", Required: true, VerificationMode: domain.VerificationModeAttestation})
	if err != nil {
		t.Fatal(err)
	}
	activated, err := s.ActivateOutcome(ctx, cc, application.ActivateOutcomeCommand{Scope: o.Scope(), ExpectedVersion: o.Version + 1})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.AttestCriterion(ctx, cc, application.AttestCriterionCommand{Owner: o.Ref(), CriterionID: criterion.Value.ID, CriterionRevision: criterion.Value.Revision, ExpectedVersion: activated.Value.Version, Result: domain.AssessmentResultMet, Rationale: "Verified"})
	if err != nil {
		t.Fatal(err)
	}
	objective, err := s.CreateObjective(ctx, cc, application.CreateObjectiveCommand{Scope: o.Scope(), Title: "Required objective", Priority: domain.PriorityNormal, RequiredForOutcome: true})
	if err != nil {
		t.Fatal(err)
	}
	childCriterion, err := s.AddCriterion(ctx, cc, application.AddCriterionCommand{Owner: objective.Value.Ref(), ExpectedVersion: objective.Value.Version, Title: "Child proof", Required: true, VerificationMode: domain.VerificationModeAttestation})
	if err != nil {
		t.Fatal(err)
	}
	childStarted, err := s.StartObjective(ctx, cc, application.StartObjectiveCommand{Scope: o.Scope(), ObjectiveID: objective.Value.ID, ExpectedVersion: objective.Value.Version + 1})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.AttestCriterion(ctx, cc, application.AttestCriterionCommand{Owner: objective.Value.Ref(), CriterionID: childCriterion.Value.ID, CriterionRevision: childCriterion.Value.Revision, ExpectedVersion: childStarted.Value.Version, Result: domain.AssessmentResultMet, Rationale: "Child verified"})
	if err != nil {
		t.Fatal(err)
	}
	childAchieved, err := s.AchieveObjective(ctx, cc, application.AchieveObjectiveCommand{Scope: o.Scope(), ObjectiveID: objective.Value.ID, ExpectedVersion: childStarted.Value.Version + 1, Reason: "Child certified"})
	if err != nil {
		t.Fatal(err)
	}
	achieved, err := s.AchieveOutcome(ctx, cc, application.AchieveOutcomeCommand{Scope: o.Scope(), ExpectedVersion: activated.Value.Version + 1, Reason: "Explicitly certified"})
	if err != nil {
		t.Fatal(err)
	}
	before, _ := json.Marshal(achieved.Value.CurrentConclusion)

	_, err = s.ReopenObjective(ctx, cc, application.ReopenObjectiveCommand{Scope: o.Scope(), ObjectiveID: childAchieved.Value.ID, ExpectedVersion: childAchieved.Value.Version, Reason: "Later obligation"})
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodePreconditionFailed {
		t.Fatalf("expected explicit parent reopen requirement: %v", err)
	}
	_, err = s.ReviseCriterion(ctx, cc, application.ReviseCriterionCommand{Owner: o.Ref(), CriterionID: criterion.Value.ID, ExpectedVersion: achieved.Value.Version, Title: "Changed obligation", Required: true, VerificationMode: domain.VerificationModeAttestation})
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeInvalidTransition {
		t.Fatalf("allowed terminal criterion change: %v", err)
	}
	_, err = s.CreateObjective(ctx, cc, application.CreateObjectiveCommand{Scope: o.Scope(), Title: "New required objective", Priority: domain.PriorityNormal, RequiredForOutcome: true})
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeInvalidTransition {
		t.Fatalf("allowed terminal structural change: %v", err)
	}
	current, err := s.GetOutcomeState(ctx, o.Scope())
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(current.Outcome.CurrentConclusion)
	if string(before) != string(after) || current.Outcome.Lifecycle != domain.OutcomeLifecycleAchieved || current.ConclusionContested {
		t.Fatal("rejected structural changes altered the conclusion")
	}
}
