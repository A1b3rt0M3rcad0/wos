package domain

import (
	"fmt"
	"strings"
)

// ID is the canonical WOS identifier. Local domain entities use UUIDv7 values
// normalized to lowercase textual representation.
type ID string

// ParseID validates and normalizes a UUIDv7 identifier.
func ParseID(value string) (ID, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if len(value) != 36 {
		return "", NewError(ErrorCodeInvalidID, "UUIDv7 must contain 36 characters")
	}

	for i := 0; i < len(value); i++ {
		switch i {
		case 8, 13, 18, 23:
			if value[i] != '-' {
				return "", NewError(ErrorCodeInvalidID, "UUIDv7 has invalid separator positions")
			}
		default:
			if !isLowerHex(value[i]) {
				return "", NewError(ErrorCodeInvalidID, "UUIDv7 contains a non-hexadecimal character")
			}
		}
	}

	if value[14] != '7' {
		return "", NewError(ErrorCodeInvalidID, "identifier must be UUID version 7")
	}
	if !isRFC4122Variant(value[19]) {
		return "", NewError(ErrorCodeInvalidID, "identifier must use the RFC 4122 variant")
	}

	return ID(value), nil
}

func MustParseID(value string) ID {
	id, err := ParseID(value)
	if err != nil {
		panic(err)
	}
	return id
}

func (id ID) String() string { return string(id) }

func (id ID) IsZero() bool { return id == "" }

func (id ID) Validate() error {
	if id.IsZero() {
		return NewError(ErrorCodeInvalidID, "identifier is required")
	}
	parsed, err := ParseID(string(id))
	if err != nil {
		return err
	}
	if parsed != id {
		return NewError(ErrorCodeInvalidID, "identifier must be normalized to lowercase")
	}
	return nil
}

func (id ID) MarshalText() ([]byte, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	return []byte(id), nil
}

func (id *ID) UnmarshalText(text []byte) error {
	if id == nil {
		return NewError(ErrorCodeInvalidID, "cannot unmarshal identifier into nil receiver")
	}
	parsed, err := ParseID(string(text))
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

func isLowerHex(ch byte) bool {
	return ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f'
}

func isRFC4122Variant(ch byte) bool {
	return ch == '8' || ch == '9' || ch == 'a' || ch == 'b'
}

// Scope identifies the namespace and Outcome boundary in which a local entity
// is authorized and addressed.
type Scope struct {
	NamespaceID ID `json:"namespace_id"`
	OutcomeID   ID `json:"outcome_id"`
}

func (s Scope) Validate() error {
	if err := s.NamespaceID.Validate(); err != nil {
		return WrapError(ErrorCodeInvalidScope, "namespace_id is invalid", err)
	}
	if err := s.OutcomeID.Validate(); err != nil {
		return WrapError(ErrorCodeInvalidScope, "outcome_id is invalid", err)
	}
	return nil
}

// EntityKind is the closed registry of addressable local entity kinds in the
// initial WOS contract.
type EntityKind string

const (
	EntityKindOutcome       EntityKind = "outcome"
	EntityKindObjective     EntityKind = "objective"
	EntityKindWorkItem      EntityKind = "work_item"
	EntityKindIssue         EntityKind = "issue"
	EntityKindBlocker       EntityKind = "blocker"
	EntityKindArtifact      EntityKind = "artifact"
	EntityKindEvidence      EntityKind = "evidence"
	EntityKindDecision      EntityKind = "decision"
	EntityKindRelation      EntityKind = "relation"
	EntityKindEvidenceLink  EntityKind = "evidence_link"
	EntityKindRoadmap       EntityKind = "roadmap"
	EntityKindTrigger       EntityKind = "trigger"
)

func (kind EntityKind) String() string { return string(kind) }

func (kind EntityKind) Valid() bool {
	switch kind {
	case EntityKindOutcome,
		EntityKindObjective,
		EntityKindWorkItem,
		EntityKindIssue,
		EntityKindBlocker,
		EntityKindArtifact,
		EntityKindEvidence,
		EntityKindDecision,
		EntityKindRelation,
		EntityKindEvidenceLink,
		EntityKindRoadmap,
		EntityKindTrigger:
		return true
	default:
		return false
	}
}

func ParseEntityKind(value string) (EntityKind, error) {
	kind := EntityKind(strings.TrimSpace(value))
	if !kind.Valid() {
		return "", NewError(ErrorCodeInvalidEntityKind, fmt.Sprintf("unsupported entity kind %q", value))
	}
	return kind, nil
}

func (kind EntityKind) MarshalText() ([]byte, error) {
	if !kind.Valid() {
		return nil, NewError(ErrorCodeInvalidEntityKind, fmt.Sprintf("unsupported entity kind %q", kind))
	}
	return []byte(kind), nil
}

func (kind *EntityKind) UnmarshalText(text []byte) error {
	if kind == nil {
		return NewError(ErrorCodeInvalidEntityKind, "cannot unmarshal entity kind into nil receiver")
	}
	parsed, err := ParseEntityKind(string(text))
	if err != nil {
		return err
	}
	*kind = parsed
	return nil
}

// EntityRef is an explicit local reference. Scope is embedded so its JSON
// representation follows the canonical flat addressing contract:
// namespace_id, outcome_id, kind, id.
type EntityRef struct {
	Scope
	Kind EntityKind `json:"kind"`
	ID   ID         `json:"id"`
}

func (ref EntityRef) Validate() error {
	if err := ref.Scope.Validate(); err != nil {
		return WrapError(ErrorCodeInvalidEntityRef, "entity scope is invalid", err)
	}
	if !ref.Kind.Valid() {
		return NewError(ErrorCodeInvalidEntityRef, "entity kind is invalid")
	}
	if err := ref.ID.Validate(); err != nil {
		return WrapError(ErrorCodeInvalidEntityRef, "entity id is invalid", err)
	}
	return nil
}
