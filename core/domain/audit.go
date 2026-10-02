package domain

import (
	"encoding/json"
	"strings"
	"time"
)

// AdministrativeAuditRecord is deliberately separate from DomainEvent because
// namespace/credential/grant/endpoint administration has no Outcome revision.
type AdministrativeAuditRecord struct {
	RecordID      ID              `json:"record_id"`
	NamespaceID   ID              `json:"namespace_id"`
	Action        string          `json:"action"`
	PrincipalID   string          `json:"principal_id"`
	Actor         ActorRef        `json:"actor_ref"`
	RecordedAt    time.Time       `json:"recorded_at"`
	CommandID     ID              `json:"command_id"`
	CorrelationID string          `json:"correlation_id,omitempty"`
	Payload       json.RawMessage `json:"payload"`
}

func (r AdministrativeAuditRecord) Validate() error {
	if err := r.RecordID.Validate(); err != nil {
		return err
	}
	if err := r.NamespaceID.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(r.Action) == "" || strings.TrimSpace(r.PrincipalID) == "" {
		return NewError(ErrorCodeInvalidArgument, "administrative audit action and principal are required")
	}
	if err := r.Actor.Validate(); err != nil {
		return err
	}
	if r.RecordedAt.IsZero() {
		return NewError(ErrorCodeInvalidArgument, "administrative audit recorded_at is required")
	}
	if err := r.CommandID.Validate(); err != nil {
		return err
	}
	if len(r.Payload) == 0 || !json.Valid(r.Payload) {
		return NewError(ErrorCodeInvalidArgument, "administrative audit payload must be valid JSON")
	}
	return nil
}
