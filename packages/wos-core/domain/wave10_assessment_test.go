package domain

import (
	"testing"
	"time"
)

func TestCriterionDefinitionRevisionSnapshotsCurrentDefinition(t *testing.T) {
	scope := Scope{
		NamespaceID: MustParseID("0199ef00-0000-7000-8000-000000000001"),
		OutcomeID:   MustParseID("0199ef00-0000-7000-8000-000000000010"),
	}
	owner := EntityRef{Scope: scope, Kind: EntityKindOutcome, ID: scope.OutcomeID}
	criterion, err := NewSuccessCriterion(
		MustParseID("0199ef00-0000-7000-8000-000000000020"),
		owner,
		"Latency below target",
		"p95 under 200ms",
		true,
		VerificationModeEvidenceReview,
	)
	if err != nil {
		t.Fatal(err)
	}
	revision := criterion.DefinitionRevision()
	if err := revision.Validate(); err != nil {
		t.Fatal(err)
	}
	if revision.CriterionID != criterion.ID ||
		revision.Revision != criterion.Revision ||
		revision.Title != criterion.Title ||
		revision.VerificationMode != criterion.VerificationMode {
		t.Fatalf("revision snapshot = %#v", revision)
	}
}

func TestRecordAssessmentEnforcesVerificationModeContract(t *testing.T) {
	scope := Scope{
		NamespaceID: MustParseID("0199ef00-0000-7000-8000-000000000001"),
		OutcomeID:   MustParseID("0199ef00-0000-7000-8000-000000000010"),
	}
	owner := EntityRef{Scope: scope, Kind: EntityKindOutcome, ID: scope.OutcomeID}
	criterion, err := NewSuccessCriterion(
		MustParseID("0199ef00-0000-7000-8000-000000000021"),
		owner,
		"Benchmark passes",
		"",
		true,
		VerificationModeEvidenceReview,
	)
	if err != nil {
		t.Fatal(err)
	}
	set := NewCriterionSet()
	if err := set.Add(criterion); err != nil {
		t.Fatal(err)
	}
	assessment := CriterionAssessment{
		ID:                MustParseID("0199ef00-0000-7000-8000-000000000030"),
		CriterionID:       criterion.ID,
		CriterionRevision: criterion.Revision,
		Result:            AssessmentResultMet,
		Rationale:         "benchmark passed",
		PrincipalID:       "tester",
		Actor:             ActorRef{Kind: ActorKindService, Provider: "test", ID: "validator"},
		AssessedAt:        time.Date(2026, 10, 2, 20, 0, 0, 0, time.UTC),
	}
	if err := set.RecordAssessment(assessment, false); err == nil {
		t.Fatal("evidence_review assessment without Evidence must fail")
	}
	assessment.EvidenceIDs = []ID{MustParseID("0199ef00-0000-7000-8000-000000000040")}
	if err := set.RecordAssessment(assessment, false); err != nil {
		t.Fatal(err)
	}
}

func TestRecordAssessmentRequiresWaiverAuthorization(t *testing.T) {
	scope := Scope{
		NamespaceID: MustParseID("0199ef00-0000-7000-8000-000000000001"),
		OutcomeID:   MustParseID("0199ef00-0000-7000-8000-000000000010"),
	}
	owner := EntityRef{Scope: scope, Kind: EntityKindOutcome, ID: scope.OutcomeID}
	criterion, err := NewSuccessCriterion(
		MustParseID("0199ef00-0000-7000-8000-000000000022"),
		owner,
		"Manual approval",
		"",
		true,
		VerificationModeAttestation,
	)
	if err != nil {
		t.Fatal(err)
	}
	set := NewCriterionSet()
	if err := set.Add(criterion); err != nil {
		t.Fatal(err)
	}
	assessment := CriterionAssessment{
		ID:                MustParseID("0199ef00-0000-7000-8000-000000000031"),
		CriterionID:       criterion.ID,
		CriterionRevision: criterion.Revision,
		Result:            AssessmentResultWaived,
		Rationale:         "explicit exception",
		PrincipalID:       "approver",
		Actor:             ActorRef{Kind: ActorKindHuman, Provider: "test", ID: "approver"},
		AssessedAt:        time.Date(2026, 10, 2, 20, 0, 0, 0, time.UTC),
	}
	if err := set.RecordAssessment(assessment, false); err == nil {
		t.Fatal("waived assessment without authorization must fail")
	}
	if err := set.RecordAssessment(assessment, true); err != nil {
		t.Fatal(err)
	}
}

func TestExternalEvaluationRequiresEvaluatorIdentity(t *testing.T) {
	scope := Scope{
		NamespaceID: MustParseID("0199ef00-0000-7000-8000-000000000001"),
		OutcomeID:   MustParseID("0199ef00-0000-7000-8000-000000000010"),
	}
	owner := EntityRef{Scope: scope, Kind: EntityKindObjective, ID: MustParseID("0199ef00-0000-7000-8000-000000000050")}
	criterion, err := NewSuccessCriterion(
		MustParseID("0199ef00-0000-7000-8000-000000000023"),
		owner,
		"External certification",
		"",
		true,
		VerificationModeExternalEvaluation,
	)
	if err != nil {
		t.Fatal(err)
	}
	set := NewCriterionSet()
	if err := set.Add(criterion); err != nil {
		t.Fatal(err)
	}
	assessment := CriterionAssessment{
		ID:                MustParseID("0199ef00-0000-7000-8000-000000000032"),
		CriterionID:       criterion.ID,
		CriterionRevision: criterion.Revision,
		Result:            AssessmentResultMet,
		Rationale:         "external evaluator accepted",
		PrincipalID:       "validator",
		Actor:             ActorRef{Kind: ActorKindService, Provider: "test", ID: "validator"},
		AssessedAt:        time.Date(2026, 10, 2, 20, 0, 0, 0, time.UTC),
	}
	if err := set.RecordAssessment(assessment, false); err == nil {
		t.Fatal("external evaluation without evaluator_ref must fail")
	}
	assessment.EvaluatorRef = &EvaluatorRef{Provider: "certifier", ID: "engine", Version: "2026.10"}
	if err := set.RecordAssessment(assessment, false); err != nil {
		t.Fatal(err)
	}
}
