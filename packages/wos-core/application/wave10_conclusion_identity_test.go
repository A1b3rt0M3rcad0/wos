package application_test

import (
	"context"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestWave10ServiceConclusionUsesCommandIdentity(t *testing.T) {
	ctx := context.Background()
	service, _ := newWave09Service(t)
	cc := commandContext()
	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  domain.MustParseID("0199ef80-0000-7000-8000-000000000001"),
		Title:        "Conclusion identity",
		DesiredState: "public conclusion identity",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value
	added, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner:            outcome.Ref(),
		ExpectedVersion:  outcome.Version,
		Title:            "Verified",
		Required:         true,
		VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	current, err := service.GetOutcome(ctx, outcome.Scope())
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
	if _, err := service.AttestCriterion(ctx, cc, application.AttestCriterionCommand{
		Owner:             outcome.Ref(),
		CriterionID:       added.Value.ID,
		CriterionRevision: added.Value.Revision,
		ExpectedVersion:   current.Value.Version,
		Result:            domain.AssessmentResultMet,
		Rationale:         "verified",
	}); err != nil {
		t.Fatal(err)
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
	if achieved.Value.CurrentConclusion == nil {
		t.Fatal("current conclusion missing")
	}
	if achieved.Value.CurrentConclusion.ID != achieveContext.CommandID {
		t.Fatalf("conclusion id = %s, command id = %s", achieved.Value.CurrentConclusion.ID, achieveContext.CommandID)
	}
}
