package domain

import "testing"

func TestRetireCriterionCreatesImmutableSemanticRevision(t *testing.T) {
	scope := Scope{
		NamespaceID: MustParseID("0199f000-0000-7000-8000-000000000001"),
		OutcomeID:   MustParseID("0199f000-0000-7000-8000-000000000010"),
	}
	owner := EntityRef{Scope: scope, Kind: EntityKindOutcome, ID: scope.OutcomeID}
	criterion, err := NewSuccessCriterion(
		MustParseID("0199f000-0000-7000-8000-000000000020"),
		owner,
		"Retirable criterion",
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
	if err := set.Retire(criterion.ID); err != nil {
		t.Fatal(err)
	}

	current, err := set.Find(criterion.ID)
	if err != nil {
		t.Fatal(err)
	}
	if current.Revision != 2 || current.Status != CriterionStatusRetired {
		t.Fatalf("retired criterion = %#v", current)
	}
	if len(set.DefinitionRevisions) != 2 {
		t.Fatalf("revision history length = %d, want 2", len(set.DefinitionRevisions))
	}
	if set.DefinitionRevisions[0].Status != CriterionStatusActive {
		t.Fatalf("revision one status = %q, want active", set.DefinitionRevisions[0].Status)
	}
	if set.DefinitionRevisions[1].Status != CriterionStatusRetired {
		t.Fatalf("revision two status = %q, want retired", set.DefinitionRevisions[1].Status)
	}
	if err := set.ValidateForOwner(owner); err != nil {
		t.Fatal(err)
	}
	if err := set.Retire(criterion.ID); err == nil {
		t.Fatal("second retirement unexpectedly succeeded")
	}
}
