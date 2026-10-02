package domain_test

import (
	"encoding/json"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func TestIDJSONRoundTripNormalizesUUIDv7(t *testing.T) {
	var id domain.ID
	if err := json.Unmarshal([]byte(`"0199E100-0000-7000-8000-000000000001"`), &id); err != nil {
		t.Fatalf("json.Unmarshal(ID) error = %v", err)
	}
	if got := id.String(); got != "0199e100-0000-7000-8000-000000000001" {
		t.Fatalf("ID = %q, want normalized lowercase", got)
	}

	payload, err := json.Marshal(id)
	if err != nil {
		t.Fatalf("json.Marshal(ID) error = %v", err)
	}
	if string(payload) != `"0199e100-0000-7000-8000-000000000001"` {
		t.Fatalf("json.Marshal(ID) = %s", payload)
	}
}

func TestEntityRefUsesCanonicalFlatAddressingJSON(t *testing.T) {
	ref := domain.EntityRef{
		Scope: domain.Scope{
			NamespaceID: domain.MustParseID("0199e100-0000-7000-8000-000000000001"),
			OutcomeID:   domain.MustParseID("0199e100-0000-7000-8000-000000000010"),
		},
		Kind: domain.EntityKindObjective,
		ID:   domain.MustParseID("0199e100-0000-7000-8000-000000000020"),
	}

	payload, err := json.Marshal(ref)
	if err != nil {
		t.Fatalf("json.Marshal(EntityRef) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("decode EntityRef JSON: %v", err)
	}
	for _, key := range []string{"namespace_id", "outcome_id", "kind", "id"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("EntityRef JSON missing %q: %s", key, payload)
		}
	}
	if _, nested := got["scope"]; nested {
		t.Fatalf("EntityRef JSON must be flat, got nested scope: %s", payload)
	}
}

func TestEnumJSONRejectsUnknownValues(t *testing.T) {
	var entityKind domain.EntityKind
	if err := json.Unmarshal([]byte(`"unknown"`), &entityKind); err == nil {
		t.Fatal("unknown EntityKind unexpectedly decoded")
	}

	var actorKind domain.ActorKind
	if err := json.Unmarshal([]byte(`"unknown"`), &actorKind); err == nil {
		t.Fatal("unknown ActorKind unexpectedly decoded")
	}
}

func TestActorKindJSONRoundTrip(t *testing.T) {
	payload, err := json.Marshal(domain.ActorKindAgent)
	if err != nil {
		t.Fatalf("json.Marshal(ActorKind) error = %v", err)
	}
	if string(payload) != `"agent"` {
		t.Fatalf("json.Marshal(ActorKind) = %s, want %q", payload, `"agent"`)
	}

	var kind domain.ActorKind
	if err := json.Unmarshal(payload, &kind); err != nil {
		t.Fatalf("json.Unmarshal(ActorKind) error = %v", err)
	}
	if kind != domain.ActorKindAgent {
		t.Fatalf("ActorKind = %q, want %q", kind, domain.ActorKindAgent)
	}
}
