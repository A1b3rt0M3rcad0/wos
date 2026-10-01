package domain

import "strings"

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
