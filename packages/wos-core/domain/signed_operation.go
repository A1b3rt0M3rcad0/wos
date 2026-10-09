package domain

import (
	"encoding/hex"
	"strings"
	"time"
)

const MaxSignedOperationResultBytes = 1 << 20

// LegacySignedOperationResult is trusted persisted cache metadata eligible for
// explicit authenticated promotion. CID and scope must be resolved from the
// original contract before it becomes a durable result, never from a new caller.
type LegacySignedOperationResult struct {
	CommandName string
	Fingerprint string
	Result      StoredCommandResult
	RecordedAt  time.Time
}

// SignedOperationResult records a committed command response independently of
// the short idempotency cache. It grants no new authority: replay returns the
// original issued documents, versions, fencing and expiry, never a fresh lease.
type SignedOperationResult struct {
	Scope          Scope               `json:"scope"`
	PrincipalID    string              `json:"principal_id"`
	CredentialID   ID                  `json:"credential_id"`
	CommandName    string              `json:"command_name"`
	IdempotencyKey string              `json:"idempotency_key"`
	Fingerprint    string              `json:"fingerprint"`
	Result         StoredCommandResult `json:"result"`
	RecordedAt     time.Time           `json:"recorded_at"`
}

func (r SignedOperationResult) Validate() error {
	if r.Scope.Validate() != nil || r.CredentialID.Validate() != nil || strings.TrimSpace(r.PrincipalID) == "" || len(r.PrincipalID) > 256 || r.RecordedAt.IsZero() || ValidateIdempotencyKey(r.IdempotencyKey) != nil {
		return NewError(ErrorCodeInvalidArgument, "invalid durable signed operation binding")
	}
	switch r.CommandName {
	case "AcquireSignedWorkContract", "AcquireNextSignedWorkContract", "RenewSignedWorkContract", "ResumeSignedWorkContract", "ReturnSignedWork", "AcquireSignedReviewContract", "RenewSignedReviewContract", "ResumeSignedReviewContract", "ReturnSignedReview", "InterveneSignedReviewCase", "RevokeSignedContract", "ReconcileSignedContracts":
	default:
		return NewError(ErrorCodeInvalidArgument, "unknown durable signed operation")
	}
	raw, e := hex.DecodeString(r.Fingerprint)
	if e != nil || len(raw) != 32 || hex.EncodeToString(raw) != r.Fingerprint {
		return NewError(ErrorCodeInvalidArgument, "invalid durable operation fingerprint")
	}
	if len(r.Result.ResponseJSON) > MaxSignedOperationResultBytes {
		return NewError(ErrorCodeInvalidArgument, "durable signed operation result exceeds 1 MiB")
	}
	return r.Result.Validate()
}
