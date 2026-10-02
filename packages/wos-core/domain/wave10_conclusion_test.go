package domain

import (
	"testing"
	"time"
)

func TestWaivedRequiredCriterionCanSatisfyGateWhenAlreadyAuthorized(t *testing.T) {
	now := time.Date(2026, 10, 2, 23, 30, 0, 0, time.UTC)
	namespaceID := MustParseID("0199ef40-0000-7000-8000-000000000001")
	outcomeID := MustParseID("0199ef40-0000-7000-8000-000000000010")
	outcome, err := NewOutcome(outcomeID, namespaceID, "Waived gate", "", "authorized exception is explicit", PriorityNormal, now)
	if err != nil {
		t.Fatal(err)
	}
	criterion, err := NewSuccessCriterion(
		MustParseID("0199ef40-0000-7000-8000-000000000020"),
		outcome.Ref(),
		"Required check",
		"",
		true,
		VerificationModeAttestation,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := outcome.AddCriterion(criterion, now); err != nil {
		t.Fatal(err)
	}
	assessment := CriterionAssessment{
		ID:                MustParseID("0199ef40-0000-7000-8000-000000000030"),
		CriterionID:       criterion.ID,
		CriterionRevision: criterion.Revision,
		Result:            AssessmentResultWaived,
		Rationale:         "authorized exception",
		PrincipalID:       "approver",
		Actor:             ActorRef{Kind: ActorKindHuman, Provider: "test", ID: "approver"},
		AssessedAt:        now,
	}
	if err := outcome.RecordCriterionAssessment(assessment, true, now); err != nil {
		t.Fatal(err)
	}
	if err := outcome.Activate(now); err != nil {
		t.Fatal(err)
	}
	conclusion := Conclusion{
		PrincipalID: "approver",
		Actor:       assessment.Actor,
		Reason:      "accepted with explicit waiver",
		ConcludedAt: now,
	}
	if err := outcome.Achieve(conclusion, now); err != nil {
		t.Fatal(err)
	}
	if outcome.CurrentConclusion == nil {
		t.Fatal("current conclusion missing")
	}
	if got := outcome.CurrentConclusion.Assessments[0].Result; got != AssessmentResultWaived {
		t.Fatalf("conclusion assessment result = %q, want waived", got)
	}
	if len(outcome.CurrentConclusion.Obligations.RequiredCriteria) != 1 {
		t.Fatalf("criterion obligations = %#v", outcome.CurrentConclusion.Obligations.RequiredCriteria)
	}
	if outcome.CurrentConclusion.OwnerVersion == nil || *outcome.CurrentConclusion.OwnerVersion != outcome.Version {
		t.Fatalf("conclusion owner version = %#v, outcome version = %d", outcome.CurrentConclusion.OwnerVersion, outcome.Version)
	}
}

func TestOutcomeConclusionSnapshotsRequiredObjectives(t *testing.T) {
	now := time.Date(2026, 10, 2, 23, 40, 0, 0, time.UTC)
	namespaceID := MustParseID("0199ef41-0000-7000-8000-000000000001")
	outcomeID := MustParseID("0199ef41-0000-7000-8000-000000000010")
	outcome, err := NewOutcome(outcomeID, namespaceID, "Structural obligations", "", "snapshot objectives", PriorityNormal, now)
	if err != nil {
		t.Fatal(err)
	}
	criterion, err := NewSuccessCriterion(
		MustParseID("0199ef41-0000-7000-8000-000000000020"),
		outcome.Ref(),
		"Verified",
		"",
		true,
		VerificationModeAttestation,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := outcome.AddCriterion(criterion, now); err != nil {
		t.Fatal(err)
	}
	assessment := CriterionAssessment{
		ID:                MustParseID("0199ef41-0000-7000-8000-000000000030"),
		CriterionID:       criterion.ID,
		CriterionRevision: criterion.Revision,
		Result:            AssessmentResultMet,
		Rationale:         "verified",
		PrincipalID:       "tester",
		Actor:             ActorRef{Kind: ActorKindService, Provider: "test", ID: "tester"},
		AssessedAt:        now,
	}
	if err := outcome.AssessCriterionAttestation(assessment, now); err != nil {
		t.Fatal(err)
	}
	if err := outcome.Activate(now); err != nil {
		t.Fatal(err)
	}
	requiredObjectiveID := MustParseID("0199ef41-0000-7000-8000-000000000050")
	conclusion := Conclusion{
		PrincipalID: "tester",
		Actor:       assessment.Actor,
		Reason:      "all obligations satisfied",
		ConcludedAt: now,
	}
	if err := outcome.AchieveWithObligations(conclusion, []ID{requiredObjectiveID}, now); err != nil {
		t.Fatal(err)
	}
	if len(outcome.CurrentConclusion.Obligations.RequiredObjectiveIDs) != 1 ||
		outcome.CurrentConclusion.Obligations.RequiredObjectiveIDs[0] != requiredObjectiveID {
		t.Fatalf("required objective snapshot = %#v", outcome.CurrentConclusion.Obligations.RequiredObjectiveIDs)
	}
}
