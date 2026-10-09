package domain

import (
	"strings"
	"time"
)

// IssuerRecovery is immutable deployment audit, not a tenant domain event or
// permission. Both identities contain public trust material only.
type IssuerRecovery struct {
	Previous     ServerIdentity `json:"previous"`
	Replacement  ServerIdentity `json:"replacement"`
	IntentDigest string         `json:"intent_digest"`
	Reason       string         `json:"reason"`
	RecordedAt   time.Time      `json:"recorded_at"`
}

func (r IssuerRecovery) Validate() error {
	if r.Previous.Validate() != nil || r.Replacement.Validate() != nil || r.Previous.ID != r.Replacement.ID || r.Previous.CreatedAt != r.Replacement.CreatedAt || r.Previous.IssuerKeyID == r.Replacement.IssuerKeyID || r.Previous.Fingerprint == r.Replacement.Fingerprint || !ValidSignedDigest(r.IntentDigest) || r.RecordedAt.IsZero() || r.RecordedAt.Before(r.Previous.CreatedAt) || strings.TrimSpace(r.Reason) == "" || len(r.Reason) > 4096 {
		return NewError(ErrorCodeInvalidArgument, "invalid explicit issuer recovery")
	}
	return nil
}
