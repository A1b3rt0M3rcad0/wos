package domain

import (
	"testing"
	"time"
)

func TestCriterionDefinitionRevisionHistoryIsCompleteAndImmutable(t *testing.T) {
	scope := Scope{
		NamespaceID: MustParseID("0199ef70-0000-7000-8000-000000000001"),
		OutcomeID:   MustParseID("0199ef70-0000-7000-8000-000000000010"),
	}
	owner := EntityRef{Scope: scope, Kind: EntityKindOutcome, ID: scope.OutcomeID}
	criterion, err := NewSuccessCriterion(
		MustParseID("0199ef70-0000-7000-8000-000000000020"),
		owner,
		"Initial definition",
		"revision one",
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
	if err := set.Revise(
		criterion.ID,
		"Evidence definition",
		"revision two",
		true,
		VerificationModeEvidenceReview,
	); err != nil {
		t.Fatal(err)
	}

	current, err := set.Find(criterion.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Revision != 2 {
		t.Fatalf("current revision = %d, want 2", current.Revision)
	}
	if len(set.DefinitionRevisions) != 2 {
		t.Fatalf("revision history length = %d, want 2", len(set.DefinitionRevisions))
	}
	if set.DefinitionRevisions[0].Title != "Initial definition" ||
		set.DefinitionRevisions[0].VerificationMode != VerificationModeAttestation {
		t.Fatalf("revision one changed: %#v", set.DefinitionRevisions[0])
	}
	if set.DefinitionRevisions[1] != current.DefinitionRevision() {
		t.Fatalf("latest history = %#v, current = %#v", set.DefinitionRevisions[1], current.DefinitionRevision())
	}
	if err := set.ValidateForOwner(owner); err != nil {
		t.Fatal(err)
	}

	broken := set
	broken.DefinitionRevisions = append([]CriterionDefinitionRevision(nil), set.DefinitionRevisions[1:]...)
	if err := broken.ValidateForOwner(owner); err == nil {
		t.Fatal("incomplete revision history unexpectedly validated")
	}

	_ = time.Time{}
}
