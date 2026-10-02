package domain

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"
)

const (
	DomainEventSchemaVersion   uint32 = 1
	MaxDomainEventPayloadBytes        = 256 * 1024
)

type DomainEvent struct {
	EventID                ID                       `json:"event_id"`
	EventType              string                   `json:"event_type"`
	SchemaVersion          uint32                   `json:"schema_version"`
	NamespaceID            ID                       `json:"namespace_id"`
	OutcomeID              ID                       `json:"outcome_id"`
	OutcomeRevision        OutcomeRevision          `json:"outcome_revision"`
	EventIndex             uint32                   `json:"event_index"`
	AggregateRef           EntityRef                `json:"aggregate_ref"`
	AggregateVersionBefore *Version                 `json:"aggregate_version_before,omitempty"`
	AggregateVersionAfter  *Version                 `json:"aggregate_version_after,omitempty"`
	PrincipalID            string                   `json:"principal_id"`
	Actor                  ActorRef                 `json:"actor_ref"`
	RecordedAt             time.Time                `json:"recorded_at"`
	CommandID              ID                       `json:"command_id"`
	CorrelationID          string                   `json:"correlation_id,omitempty"`
	CausationID            *ID                      `json:"causation_id,omitempty"`
	ExecutionContext       ExternalExecutionContext `json:"execution_context,omitempty"`
	Payload                json.RawMessage          `json:"payload"`
}

func (e DomainEvent) Scope() Scope {
	return Scope{NamespaceID: e.NamespaceID, OutcomeID: e.OutcomeID}
}

func (e DomainEvent) Validate() error {
	if err := e.EventID.Validate(); err != nil {
		return WrapError(ErrorCodeInvalidEvent, "event_id is invalid", err)
	}
	if !validEventType(e.EventType) {
		return NewError(ErrorCodeInvalidEvent, "event_type must use <singular_snake_case>.<past_fact>")
	}
	if e.SchemaVersion == 0 {
		return NewError(ErrorCodeInvalidEvent, "schema_version must be positive")
	}
	if err := e.Scope().Validate(); err != nil {
		return WrapError(ErrorCodeInvalidEvent, "event scope is invalid", err)
	}
	if e.OutcomeRevision == 0 {
		return NewError(ErrorCodeInvalidEvent, "outcome_revision must be positive")
	}
	if err := e.AggregateRef.Validate(); err != nil {
		return WrapError(ErrorCodeInvalidEvent, "aggregate_ref is invalid", err)
	}
	if e.AggregateRef.Scope != e.Scope() {
		return NewError(ErrorCodeInvalidEvent, "aggregate_ref scope must match event scope")
	}
	if e.AggregateVersionBefore != nil {
		if err := e.AggregateVersionBefore.Validate(); err != nil {
			return WrapError(ErrorCodeInvalidEvent, "aggregate_version_before is invalid", err)
		}
	}
	if e.AggregateVersionAfter != nil {
		if err := e.AggregateVersionAfter.Validate(); err != nil {
			return WrapError(ErrorCodeInvalidEvent, "aggregate_version_after is invalid", err)
		}
	}
	if e.AggregateVersionBefore != nil && e.AggregateVersionAfter != nil && *e.AggregateVersionAfter <= *e.AggregateVersionBefore {
		return NewError(ErrorCodeInvalidEvent, "aggregate version must advance")
	}
	if strings.TrimSpace(e.PrincipalID) == "" {
		return NewError(ErrorCodeInvalidEvent, "principal_id is required")
	}
	if err := e.Actor.Validate(); err != nil {
		return WrapError(ErrorCodeInvalidEvent, "actor_ref is invalid", err)
	}
	if e.RecordedAt.IsZero() {
		return NewError(ErrorCodeInvalidEvent, "recorded_at is required")
	}
	if err := e.CommandID.Validate(); err != nil {
		return WrapError(ErrorCodeInvalidEvent, "command_id is invalid", err)
	}
	if e.CausationID != nil {
		if err := e.CausationID.Validate(); err != nil {
			return WrapError(ErrorCodeInvalidEvent, "causation_id is invalid", err)
		}
	}
	if len(e.Payload) == 0 || !json.Valid(e.Payload) {
		return NewError(ErrorCodeInvalidEvent, "payload must be valid JSON")
	}
	if len(e.Payload) > MaxDomainEventPayloadBytes {
		return NewError(ErrorCodeInvalidEvent, "payload exceeds 256 KiB")
	}
	if bytes.Equal(bytes.TrimSpace(e.Payload), []byte("null")) {
		return NewError(ErrorCodeInvalidEvent, "payload cannot be null")
	}
	return nil
}

func validEventType(value string) bool {
	parts := strings.Split(value, ".")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return false
	}
	for _, part := range parts {
		for i, r := range part {
			if r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '_' && i > 0 {
				continue
			}
			return false
		}
	}
	return true
}
