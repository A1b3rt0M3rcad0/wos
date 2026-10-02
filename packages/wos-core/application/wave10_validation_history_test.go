package application_test

import (
	"context"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestWave10ValidationHistoryReadSurface(t *testing.T) {
	ctx := context.Background()
	service, _ := newWave09Service(t)
	cc := commandContext()
	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  domain.MustParseID("0199ef90-0000-7000-8000-000000000001"),
		Title:        "Validation history",
		DesiredState: "history can be read explicitly",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value
	added, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner:            outcome.Ref(),
		ExpectedVersion:  outcome.Version,
		Title:            "Initial",
		Description:      "revision one",
		Required:         true,
		VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	criterion := added.Value
	current, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	revised, err := service.ReviseCriterion(ctx, cc, application.ReviseCriterionCommand{
		Owner:            outcome.Ref(),
		CriterionID:      criterion.ID,
		ExpectedVersion:  current.Value.Version,
		Title:            "Revised",
		Description:      "revision two",
		Required:         true,
		VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	current, err = service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ActivateOutcome(ctx, cc, application.ActivateOutcomeCommand{
		Scope: outcome.Scope(), ExpectedVersion: current.Value.Version,
	}); err != nil {
		t.Fatal(err)
	}
	current, err = service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	assessed, err := service.AttestCriterion(ctx, cc, application.AttestCriterionCommand{
		Owner:             outcome.Ref(),
		CriterionID:       criterion.ID,
		CriterionRevision: revised.Value.Revision,
		ExpectedVersion:   current.Value.Version,
		Result:            domain.AssessmentResultMet,
		Rationale:         "verified",
	})
	if err != nil {
		t.Fatal(err)
	}

	history, err := service.GetCriterionHistory(ctx, outcome.Ref(), criterion.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(history.Value.DefinitionRevisions) != 2 {
		t.Fatalf("definition revisions = %#v", history.Value.DefinitionRevisions)
	}
	if len(history.Value.Assessments) != 1 ||
		history.Value.Assessments[0].ID != assessed.Value.ID {
		t.Fatalf("assessment history = %#v", history.Value.Assessments)
	}
	if history.Value.CurrentAssessment == nil ||
		history.Value.CurrentAssessment.ID != assessed.Value.ID {
		t.Fatalf("current assessment = %#v", history.Value.CurrentAssessment)
	}

	current, err = service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	achieveContext := commandContext()
	achieved, err := service.AchieveOutcome(ctx, achieveContext, application.AchieveOutcomeCommand{
		Scope: outcome.Scope(), ExpectedVersion: current.Value.Version, Reason: "verified",
	})
	if err != nil {
		t.Fatal(err)
	}
	conclusions, err := service.ListConclusions(ctx, outcome.Ref())
	if err != nil {
		t.Fatal(err)
	}
	if conclusions.Value.Current == nil ||
		conclusions.Value.Current.ID != achieved.Value.CurrentConclusion.ID {
		t.Fatalf("current conclusion = %#v", conclusions.Value.Current)
	}
	fetched, err := service.GetConclusion(ctx, outcome.Ref(), achieved.Value.CurrentConclusion.ID)
	if err != nil {
		t.Fatal(err)
	}
	if fetched.Value.ID != achieved.Value.CurrentConclusion.ID {
		t.Fatalf("fetched conclusion id = %s, want %s", fetched.Value.ID, achieved.Value.CurrentConclusion.ID)
	}
}
