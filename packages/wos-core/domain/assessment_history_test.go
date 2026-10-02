package domain_test

import (
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestAssessmentHistoryIsImmutableAcrossSupersessionAndRevision(t *testing.T) {
	now := time.Date(2026, 10, 1, 22, 30, 0, 0, time.UTC)
	outcome, err := domain.NewOutcome(
		domain.MustParseID("0199e400-0000-7000-8000-000000000010"),
		domain.MustParseID("0199e400-0000-7000-8000-000000000001"),
		"Assessment history",
		"",
		"State",
		domain.PriorityNormal,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}

	criterionID := domain.MustParseID("0199e400-0000-7000-8000-000000000020")
	criterion, err := domain.NewSuccessCriterion(
		criterionID,
		outcome.Ref(),
		"Verified",
		"",
		true,
		domain.VerificationModeAttestation,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := outcome.AddCriterion(criterion, now); err != nil {
		t.Fatal(err)
	}

	first := domain.CriterionAssessment{
		ID:                domain.MustParseID("0199e400-0000-7000-8000-000000000030"),
		CriterionID:       criterionID,
		CriterionRevision: 1,
		Result:            domain.AssessmentResultNotMet,
		Rationale:         "first inspection failed",
		PrincipalID:       "human-1",
		Actor:             actor(),
		AssessedAt:        now,
	}
	if err := outcome.AssessCriterionAttestation(first, now); err != nil {
		t.Fatal(err)
	}

	second := domain.CriterionAssessment{
		ID:                domain.MustParseID("0199e400-0000-7000-8000-000000000031"),
		CriterionID:       criterionID,
		CriterionRevision: 1,
		Result:            domain.AssessmentResultMet,
		Rationale:         "second inspection passed",
		PrincipalID:       "human-1",
		Actor:             actor(),
		AssessedAt:        now.Add(time.Minute),
	}
	if err := outcome.AssessCriterionAttestation(second, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	if len(outcome.Criteria.Assessments) != 2 {
		t.Fatalf("assessment history length = %d, want 2", len(outcome.Criteria.Assessments))
	}
	current := outcome.Criteria.CurrentAssessments[criterionID]
	if current.ID != second.ID || current.SupersedesAssessmentID == nil || *current.SupersedesAssessmentID != first.ID {
		t.Fatalf("unexpected current assessment: %#v", current)
	}

	if err := outcome.ReviseCriterion(
		criterionID,
		"Verified more strictly",
		"",
		true,
		domain.VerificationModeAttestation,
		now.Add(2*time.Minute),
	); err != nil {
		t.Fatal(err)
	}

	if _, exists := outcome.Criteria.CurrentAssessments[criterionID]; exists {
		t.Fatal("criterion revision must invalidate current assessment projection")
	}
	if len(outcome.Criteria.Assessments) != 2 {
		t.Fatalf("criterion revision erased immutable history: %d", len(outcome.Criteria.Assessments))
	}
	if err := outcome.Criteria.ValidateForOwner(outcome.Ref()); err != nil {
		t.Fatalf("ValidateForOwner() error = %v", err)
	}
}
