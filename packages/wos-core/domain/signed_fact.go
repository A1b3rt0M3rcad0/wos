package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"
)

// SignedFact persists the exact authenticated payload and detached proof. The
// Application verifies them; storage validates size, digest and scope only.
// Consumers must verify payload/proof with their pinned purpose and public key.
type SignedFact struct {
	ID             ID              `json:"id"`
	Scope          Scope           `json:"scope"`
	ContractID     ID              `json:"contract_id"`
	ContractKind   string          `json:"contract_kind"`
	Kind           string          `json:"kind"`
	PrincipalID    string          `json:"principal_id"`
	KeyID          ID              `json:"key_id"`
	RequestID      *ID             `json:"request_id,omitempty"`
	IdempotencyKey string          `json:"idempotency_key,omitempty"`
	RequestDigest  string          `json:"request_digest,omitempty"`
	PayloadDigest  string          `json:"payload_digest"`
	Payload        []byte          `json:"payload"`
	Proof          json.RawMessage `json:"proof"`
	RecordedAt     time.Time       `json:"recorded_at"`
}

func (f SignedFact) Validate() error {
	if f.ID.Validate() != nil || f.Scope.Validate() != nil || f.ContractID.Validate() != nil || f.KeyID.Validate() != nil || strings.TrimSpace(f.PrincipalID) == "" || len(f.PrincipalID) > 256 || f.RecordedAt.IsZero() || (f.ContractKind != "execution" && f.ContractKind != "review") {
		return NewError(ErrorCodeInvalidArgument, "invalid signed fact identity")
	}
	if f.Kind != "specification" && f.Kind != "authority" && f.Kind != "return" && f.Kind != "acceptance" {
		return NewError(ErrorCodeInvalidArgument, "unknown signed fact purpose")
	}
	limit := 180 * 1024
	if f.Kind == "specification" {
		limit = 128 * 1024
	}
	if len(f.Payload) == 0 || len(f.Payload) > limit || !json.Valid(f.Payload) || len(f.Proof) == 0 || len(f.Proof) > 4096 || !json.Valid(f.Proof) {
		return NewError(ErrorCodeInvalidArgument, "signed fact payload/proof invalid or too large")
	}
	sum := sha256.Sum256(f.Payload)
	if f.PayloadDigest != "sha256:"+hex.EncodeToString(sum[:]) {
		return NewError(ErrorCodeContractSpecMismatch, "signed fact exact payload digest differs")
	}
	if f.Kind == "return" || f.Kind == "acceptance" {
		if f.RequestID == nil || f.RequestID.Validate() != nil || ValidateIdempotencyKey(f.IdempotencyKey) != nil || !ValidSignedDigest(f.RequestDigest) {
			return NewError(ErrorCodeInvalidArgument, "signed operation fact request binding required")
		}
	} else if f.RequestID != nil || f.IdempotencyKey != "" || f.RequestDigest != "" {
		return NewError(ErrorCodeInvalidArgument, "issuance cannot have request receipt fields")
	}
	return nil
}
