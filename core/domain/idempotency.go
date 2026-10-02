package domain

import (
	"encoding/json"
	"strings"
)

const (
	MinIdempotencyKeyLength = 16
	MaxIdempotencyKeyLength = 128
)

type IdempotencyIdentity struct {
	NamespaceID   ID     `json:"namespace_id"`
	PrincipalID   string `json:"principal_id"`
	CommandName   string `json:"command_name"`
	IdempotencyKey string `json:"idempotency_key"`
}

func (i IdempotencyIdentity) Validate() error {
	if err := i.NamespaceID.Validate(); err != nil {
		return WrapError(ErrorCodeInvalidIdempotencyKey, "namespace_id is invalid", err)
	}
	if strings.TrimSpace(i.PrincipalID) == "" {
		return NewError(ErrorCodeInvalidIdempotencyKey, "principal_id is required")
	}
	if strings.TrimSpace(i.CommandName) == "" {
		return NewError(ErrorCodeInvalidIdempotencyKey, "command_name is required")
	}
	if err := ValidateIdempotencyKey(i.IdempotencyKey); err != nil {
		return err
	}
	return nil
}

func ValidateIdempotencyKey(value string) error {
	if len(value) < MinIdempotencyKeyLength || len(value) > MaxIdempotencyKeyLength {
		return NewError(ErrorCodeInvalidIdempotencyKey, "idempotency key must contain 16 to 128 characters")
	}
	for _, r := range value {
		if r > 127 || !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' || r == '_' || r == '.' || r == ':') {
			return NewError(ErrorCodeInvalidIdempotencyKey, "idempotency key contains unsafe ASCII characters")
		}
	}
	return nil
}

type StoredCommandResult struct {
	CommandID       ID              `json:"command_id"`
	OutcomeRevision OutcomeRevision `json:"outcome_revision"`
	ResponseJSON    json.RawMessage `json:"response_json"`
}

func (r StoredCommandResult) Validate() error {
	if err := r.CommandID.Validate(); err != nil {
		return WrapError(ErrorCodeIdempotencyState, "stored command_id is invalid", err)
	}
	if r.OutcomeRevision == 0 {
		return NewError(ErrorCodeIdempotencyState, "stored outcome_revision must be positive")
	}
	if len(r.ResponseJSON) == 0 || !json.Valid(r.ResponseJSON) {
		return NewError(ErrorCodeIdempotencyState, "stored response must be valid JSON")
	}
	return nil
}

type IdempotencyReservation struct {
	Identity    IdempotencyIdentity
	Fingerprint string
	Replay      *StoredCommandResult
}

func (r IdempotencyReservation) IsReplay() bool {
	return r.Replay != nil
}
