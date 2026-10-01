package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

const (
	namespaceID = "0199e100-0000-7000-8000-000000000001"
	outcomeID   = "0199e100-0000-7000-8000-000000000010"
	objectiveID = "0199e100-0000-7000-8000-000000000020"
)

func TestParseIDAcceptsAndNormalizesUUIDv7(t *testing.T) {
	got, err := domain.ParseID("0199E100-0000-7000-8000-000000000001")
	if err != nil {
		t.Fatalf("ParseID() error = %v", err)
	}
	if got.String() != namespaceID {
		t.Fatalf("ParseID() = %q, want %q", got, namespaceID)
	}
}

func TestParseIDRejectsNonV7AndWrongVariant(t *testing.T) {
	tests := []string{
		"0199e100-0000-6000-8000-000000000001",
		"0199e100-0000-7000-7000-000000000001",
		"not-a-uuid",
	}
	for _, value := range tests {
		t.Run(value, func(t *testing.T) {
			if _, err := domain.ParseID(value); err == nil {
				t.Fatalf("ParseID(%q) unexpectedly succeeded", value)
			}
		})
	}
}

func TestScopeAndEntityRefValidation(t *testing.T) {
	namespace := domain.MustParseID(namespaceID)
	outcome := domain.MustParseID(outcomeID)
	objective := domain.MustParseID(objectiveID)

	ref := domain.EntityRef{
		Scope: domain.Scope{NamespaceID: namespace, OutcomeID: outcome},
		Kind:  domain.EntityKindObjective,
		ID:    objective,
	}
	if err := ref.Validate(); err != nil {
		t.Fatalf("EntityRef.Validate() error = %v", err)
	}

	ref.Kind = domain.EntityKind("unknown")
	if err := ref.Validate(); err == nil {
		t.Fatal("EntityRef.Validate() unexpectedly accepted unknown kind")
	}
}

func TestEntityKindJSONIsStableString(t *testing.T) {
	payload, err := json.Marshal(domain.EntityKindWorkItem)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	if string(payload) != `"work_item"` {
		t.Fatalf("json.Marshal() = %s, want %q", payload, `"work_item"`)
	}
}
