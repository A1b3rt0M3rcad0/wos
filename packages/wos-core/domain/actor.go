package domain

import (
	"fmt"
	"strings"
)

// ActorKind describes the declared author of a domain action. ActorRef records
// attribution; it is not an authentication or authorization credential.
type ActorKind string

const (
	ActorKindHuman          ActorKind = "human"
	ActorKindAgent          ActorKind = "agent"
	ActorKindService        ActorKind = "service"
	ActorKindAutomation     ActorKind = "automation"
	ActorKindExternalSystem ActorKind = "external_system"
)

func (kind ActorKind) String() string { return string(kind) }

func (kind ActorKind) Valid() bool {
	switch kind {
	case ActorKindHuman,
		ActorKindAgent,
		ActorKindService,
		ActorKindAutomation,
		ActorKindExternalSystem:
		return true
	default:
		return false
	}
}

func ParseActorKind(value string) (ActorKind, error) {
	kind := ActorKind(strings.TrimSpace(value))
	if !kind.Valid() {
		return "", NewError(ErrorCodeInvalidActorKind, fmt.Sprintf("unsupported actor kind %q", value))
	}
	return kind, nil
}

func (kind ActorKind) MarshalText() ([]byte, error) {
	if !kind.Valid() {
		return nil, NewError(ErrorCodeInvalidActorKind, fmt.Sprintf("unsupported actor kind %q", kind))
	}
	return []byte(kind), nil
}

func (kind *ActorKind) UnmarshalText(text []byte) error {
	if kind == nil {
		return NewError(ErrorCodeInvalidActorKind, "cannot unmarshal actor kind into nil receiver")
	}
	parsed, err := ParseActorKind(string(text))
	if err != nil {
		return err
	}
	*kind = parsed
	return nil
}

// ActorRef is the declared author/actor associated with a domain action. The
// authenticated Principal is represented by application/authentication layers
// and intentionally remains separate from this value object.
type ActorRef struct {
	Kind        ActorKind `json:"kind"`
	Provider    string    `json:"provider"`
	ID          string    `json:"id"`
	DisplayName string    `json:"display_name,omitempty"`
}

func (ref ActorRef) Validate() error {
	if !ref.Kind.Valid() {
		return NewError(ErrorCodeInvalidActorKind, "actor kind is invalid")
	}
	if strings.TrimSpace(ref.Provider) == "" {
		return NewError(ErrorCodeInvalidActorRef, "actor provider is required")
	}
	if strings.TrimSpace(ref.ID) == "" {
		return NewError(ErrorCodeInvalidActorRef, "actor id is required")
	}
	return nil
}
