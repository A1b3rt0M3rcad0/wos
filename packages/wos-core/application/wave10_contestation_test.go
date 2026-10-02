package application_test

import (
	"context"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestWave10LaterContradictionContestsConclusionWithoutReopening(t *testing.T) {
	ctx := context.Background()
	service, _ := newWave09Service(t)
	outcome := setupActiveOutcome(t, service)
	scope := outcome.Scope()
	cc := commandContext()

	current, err := service.GetOutcome(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	added, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner:            outcome.Ref(),
		ExpectedVersion:  current.Value.Version,
		Title:            "Verified condition",
		Required:         true,
		VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	criterion := added.Value

	current, err = service.GetOutcome(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	met, err := service.AttestCriterion(ctx, cc, application.AttestCriterionCommand{
		Owner:             outcome.Ref(),
		CriterionID:       criterion.ID,
		CriterionRevision: criterion.Revision,
		ExpectedVersion:   current.Value.Version,
		Result:            domain.AssessmentResultMet,
		Rationale:         "initial verification",
	})
	if err != nil {
		t.Fatal(err)
	}
	current, err = service.GetOutcome(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	achieved, err := service.AchieveOutcome(ctx, cc, application.AchieveOutcomeCommand{
		Scope:           scope,
		ExpectedVersion: current.Value.Version,
		Reason:          "verified",
	})
	if err != nil {
		t.Fatal(err)
	}
	if achieved.Value.Lifecycle != domain.OutcomeLifecycleAchieved {
		t.Fatalf("lifecycle = %q", achieved.Value.Lifecycle)
	}

	contradicted, err := service.RecordCriterionAssessment(ctx, cc, application.RecordCriterionAssessmentCommand{
		Owner:             outcome.Ref(),
		CriterionID:       criterion.ID,
		CriterionRevision: criterion.Revision,
		ExpectedVersion:   achieved.Value.Version,
		Result:            domain.AssessmentResultNotMet,
		Rationale:         "later verification failed",
	})
	if err != nil {
		t.Fatal(err)
	}
	if contradicted.Value.ID == met.Value.ID {
		t.Fatal("contradictory assessment reused immutable assessment identity")
	}

	state, err := service.GetOutcomeState(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if state.Outcome.Lifecycle != domain.OutcomeLifecycleAchieved {
		t.Fatalf("contestation changed lifecycle to %q", state.Outcome.Lifecycle)
	}
	if !state.ConclusionContested || len(state.ConclusionContestations) != 1 {
		t.Fatalf("contestation projection = %#v", state.ConclusionContestations)
	}
	cause := state.ConclusionContestations[0]
	if cause.Kind != domain.ConclusionContestationAssessmentContradiction ||
		cause.CriterionID != criterion.ID ||
		cause.AssessmentID != contradicted.Value.ID {
		t.Fatalf("contestation cause = %#v", cause)
	}
}
