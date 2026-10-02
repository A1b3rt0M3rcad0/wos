package memory

import (
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestWave10ValidationHistoryAppendOnlyGuard(t *testing.T) {
	now := time.Date(2026, 10, 2, 20, 45, 0, 0, time.UTC)
	scope := domain.Scope{
		NamespaceID: domain.MustParseID("0199efe0-0000-7000-8000-000000000001"),
		OutcomeID:   domain.MustParseID("0199efe0-0000-7000-8000-000000000010"),
	}
	owner := domain.EntityRef{Scope: scope, Kind: domain.EntityKindOutcome, ID: scope.OutcomeID}
	criterion, err := domain.NewSuccessCriterion(
		domain.MustParseID("0199efe0-0000-7000-8000-000000000020"),
		owner,
		"Initial",
		"revision one",
		true,
		domain.VerificationModeAttestation,
	)
	if err != nil {
		t.Fatal(err)
	}
	current := domain.NewCriterionSet()
	if err := current.Add(criterion); err != nil {
		t.Fatal(err)
	}

	next := cloneCriteria(current)
	if err := next.Revise(
		criterion.ID,
		"Revised",
		"revision two",
		true,
		domain.VerificationModeAttestation,
	); err != nil {
		t.Fatal(err)
	}
	next.DefinitionRevisions[0].Title = "rewritten history"
	if err := validateOwnedValidationHistoryAppendOnly(
		current, next, nil, nil, nil, nil,
	); err == nil {
		t.Fatal("rewritten criterion history unexpectedly accepted")
	}

	next = cloneCriteria(current)
	assessment := domain.CriterionAssessment{
		ID:                domain.MustParseID("0199efe0-0000-7000-8000-000000000030"),
		CriterionID:       criterion.ID,
		CriterionRevision: criterion.Revision,
		Result:            domain.AssessmentResultMet,
		Rationale:         "verified",
		PrincipalID:       "tester",
		Actor:             domain.ActorRef{Kind: domain.ActorKindService, Provider: "test", ID: "tester"},
		AssessedAt:        now,
	}
	if err := next.RecordAssessment(assessment, false); err != nil {
		t.Fatal(err)
	}
	committed := cloneCriteria(next)
	next = cloneCriteria(committed)
	next.Assessments[0].Rationale = "rewritten assessment"
	if err := validateOwnedValidationHistoryAppendOnly(
		committed, next, nil, nil, nil, nil,
	); err == nil {
		t.Fatal("rewritten assessment unexpectedly accepted")
	}
}

func TestWave10ConclusionAppendOnlyGuard(t *testing.T) {
	now := time.Date(2026, 10, 2, 20, 50, 0, 0, time.UTC)
	conclusion := domain.Conclusion{
		ID:          domain.MustParseID("0199efe1-0000-7000-8000-000000000001"),
		PrincipalID: "tester",
		Actor:       domain.ActorRef{Kind: domain.ActorKindService, Provider: "test", ID: "tester"},
		Reason:      "verified",
		ConcludedAt: now,
	}
	mutated := conclusion
	mutated.Reason = "rewritten"
	if err := validateOwnedValidationHistoryAppendOnly(
		domain.NewCriterionSet(),
		domain.NewCriterionSet(),
		&conclusion,
		nil,
		&mutated,
		nil,
	); err == nil {
		t.Fatal("current conclusion replacement unexpectedly accepted")
	}

	if err := validateOwnedValidationHistoryAppendOnly(
		domain.NewCriterionSet(),
		domain.NewCriterionSet(),
		&conclusion,
		nil,
		nil,
		[]domain.Conclusion{conclusion},
	); err != nil {
		t.Fatalf("valid reopen append rejected: %v", err)
	}
}
