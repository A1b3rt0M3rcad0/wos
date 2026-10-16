package domain

import (
	"encoding/json"
	"strings"
	"time"
)

// SignedReturnIntentResolution is an immutable negative decision, separate
// from successful SignedOperationResult. Its receipt never undoes external work.
type SignedReturnIntentResolution struct {
	Scope                    Scope           `json:"scope"`
	PrincipalID              string          `json:"principal_id"`
	CredentialID             ID              `json:"credential_id"`
	OriginalCommandName      string          `json:"original_command_name"`
	OriginalIdempotencyKey   string          `json:"original_idempotency_key"`
	OriginalRequestID        ID              `json:"original_request_id"`
	OriginalRequestDigest    string          `json:"original_request_digest"`
	OriginalEnvelopeDigest   string          `json:"original_envelope_digest"`
	ContractID               ID              `json:"contract_id"`
	ResolutionRequestID      ID              `json:"resolution_request_id"`
	ResolutionIdempotencyKey string          `json:"resolution_idempotency_key"`
	ResolutionDigest         string          `json:"resolution_digest"`
	IssuerKeyID              ID              `json:"issuer_key_id"`
	Decision                 string          `json:"decision"`
	Reason                   string          `json:"reason"`
	OutcomeRevision          OutcomeRevision `json:"outcome_revision"`
	Receipt                  json.RawMessage `json:"receipt"`
	RecordedAt               time.Time       `json:"recorded_at"`
}

func (r SignedReturnIntentResolution) Validate() error {
	if r.Scope.Validate() != nil || strings.TrimSpace(r.PrincipalID) == "" || len(r.PrincipalID) > 256 || r.CredentialID.Validate() != nil || r.ContractID.Validate() != nil || r.OriginalRequestID.Validate() != nil || r.ResolutionRequestID.Validate() != nil || r.IssuerKeyID.Validate() != nil || r.RecordedAt.IsZero() || r.OutcomeRevision == 0 {
		return NewError(ErrorCodeInvalidArgument, "invalid signed return resolution binding")
	}
	if (r.OriginalCommandName != "ReturnSignedWork" && r.OriginalCommandName != "ReturnSignedReview") || ValidateIdempotencyKey(r.OriginalIdempotencyKey) != nil || ValidateIdempotencyKey(r.ResolutionIdempotencyKey) != nil || r.OriginalIdempotencyKey == r.ResolutionIdempotencyKey || r.OriginalRequestID == r.ResolutionRequestID || !ValidSignedDigest(r.OriginalRequestDigest) || !ValidSignedDigest(r.OriginalEnvelopeDigest) || !ValidSignedDigest(r.ResolutionDigest) {
		return NewError(ErrorCodeInvalidArgument, "invalid original/resolution intent identity")
	}
	if r.Decision != "closed_without_acceptance" || strings.TrimSpace(r.Reason) == "" || len(r.Reason) > 4096 || len(r.Receipt) == 0 || len(r.Receipt) > 32<<10 || !json.Valid(r.Receipt) {
		return NewError(ErrorCodeInvalidArgument, "invalid terminal resolution receipt")
	}
	return nil
}
