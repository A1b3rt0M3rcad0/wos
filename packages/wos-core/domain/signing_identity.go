package domain

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"strings"
	"time"
)

type AcceptanceMode string

const (
	AcceptanceDirect            AcceptanceMode = "direct"
	AcceptanceIndependentReview AcceptanceMode = "independent_review"
)

func (a AcceptanceMode) Valid() bool {
	return a == AcceptanceDirect || a == AcceptanceIndependentReview
}
func StrongerAcceptance(modes ...AcceptanceMode) AcceptanceMode {
	for _, m := range modes {
		if m == AcceptanceIndependentReview {
			return m
		}
	}
	return AcceptanceDirect
}

type SigningKey struct {
	ID          ID        `json:"id"`
	NamespaceID ID        `json:"namespace_id"`
	PrincipalID string    `json:"principal_id"`
	Purpose     string    `json:"purpose"`
	Algorithm   string    `json:"algorithm"`
	PublicKey   string    `json:"public_key"`
	Fingerprint string    `json:"fingerprint"`
	Version     Version   `json:"version,string"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Reason      string    `json:"reason,omitempty"`
}

func (k SigningKey) Validate() error {
	if k.ID.Validate() != nil || k.NamespaceID.Validate() != nil || k.Version.Validate() != nil || strings.TrimSpace(k.PrincipalID) == "" || len(k.PrincipalID) > 256 {
		return NewError(ErrorCodeInvalidArgument, "invalid signing key identity")
	}
	if k.CreatedAt.IsZero() || k.UpdatedAt.Before(k.CreatedAt) {
		return NewError(ErrorCodeInvalidArgument, "invalid signing key chronology")
	}
	if k.Algorithm != "Ed25519" || (k.Purpose != "agent" && k.Purpose != "issuer") || (k.Status != "active" && k.Status != "retired" && k.Status != "revoked") {
		return NewError(ErrorCodeInvalidArgument, "invalid signing key purpose/state/algorithm")
	}
	b, err := base64.StdEncoding.Strict().DecodeString(k.PublicKey)
	if err != nil || len(b) != 32 || base64.StdEncoding.EncodeToString(b) != k.PublicKey {
		return NewError(ErrorCodeInvalidArgument, "invalid Ed25519 public key")
	}
	hash := sha256.Sum256(b)
	if k.Fingerprint != "sha256:"+hex.EncodeToString(hash[:]) {
		return NewError(ErrorCodeInvalidArgument, "public key fingerprint differs")
	}
	return nil
}
func (k SigningKey) PublicBytes() ([]byte, error) {
	if err := k.Validate(); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(k.PublicKey)
}

type SigningEnrollment struct {
	ID                  ID        `json:"id"`
	NamespaceID         ID        `json:"namespace_id"`
	PrincipalID         string    `json:"principal_id"`
	CredentialID        ID        `json:"credential_id"`
	KeyID               ID        `json:"key_id"`
	PreviousKeyID       *ID       `json:"previous_key_id,omitempty"`
	Purpose             string    `json:"purpose"`
	Nonce               string    `json:"nonce"`
	ExpiresAt           time.Time `json:"expires_at"`
	AuthorizedBy        string    `json:"authorized_by"`
	ExpectedFingerprint string    `json:"expected_fingerprint,omitempty"`
	Version             Version   `json:"version,string"`
	Consumed            bool      `json:"consumed"`
	AcceptedDigest      string    `json:"accepted_digest,omitempty"`
}

func (e SigningEnrollment) Validate() error {
	if e.ID.Validate() != nil || e.NamespaceID.Validate() != nil || e.CredentialID.Validate() != nil || e.KeyID.Validate() != nil || e.Version.Validate() != nil || e.PrincipalID == "" || e.AuthorizedBy == "" || e.Purpose != "agent" || len(e.Nonce) < 32 || len(e.Nonce) > 128 || e.ExpiresAt.IsZero() {
		return NewError(ErrorCodeInvalidArgument, "invalid signing enrollment")
	}
	if e.PreviousKeyID != nil && e.PreviousKeyID.Validate() != nil {
		return NewError(ErrorCodeInvalidArgument, "invalid previous signing identity")
	}
	return nil
}

type CredentialPolicy struct {
	NamespaceID              ID             `json:"namespace_id"`
	CredentialID             ID             `json:"credential_id"`
	PrincipalID              string         `json:"principal_id"`
	Version                  Version        `json:"revision,string"`
	PermittedOperations      []string       `json:"permitted_operations"`
	AcceptanceFloor          AcceptanceMode `json:"acceptance_floor"`
	AllowedOutcomeIDs        []ID           `json:"allowed_outcome_ids"`
	AllowedSigningKeyIDs     []ID           `json:"allowed_signing_key_ids"`
	MaxActiveWorkContracts   int            `json:"max_active_work_contracts"`
	MaxActiveReviewContracts int            `json:"max_active_review_contracts"`
	UpdatedAt                time.Time      `json:"updated_at"`
}

func (p CredentialPolicy) Validate() error {
	if p.NamespaceID.Validate() != nil || p.CredentialID.Validate() != nil || p.Version.Validate() != nil || p.PrincipalID == "" || !p.AcceptanceFloor.Valid() || len(p.PermittedOperations) > 32 || len(p.AllowedOutcomeIDs) > 100 || len(p.AllowedSigningKeyIDs) > 100 || p.MaxActiveWorkContracts < 1 || p.MaxActiveWorkContracts > 100 || p.MaxActiveReviewContracts < 1 || p.MaxActiveReviewContracts > 100 {
		return NewError(ErrorCodeInvalidArgument, "invalid credential policy")
	}
	seen := map[string]bool{}
	for _, op := range p.PermittedOperations {
		if op == "" || seen[op] {
			return NewError(ErrorCodeInvalidArgument, "duplicate/empty permitted operation")
		}
		seen[op] = true
	}
	for _, ids := range [][]ID{p.AllowedOutcomeIDs, p.AllowedSigningKeyIDs} {
		seen := map[ID]bool{}
		for _, id := range ids {
			if id.Validate() != nil || seen[id] {
				return NewError(ErrorCodeInvalidArgument, "invalid/duplicate policy binding")
			}
			seen[id] = true
		}
	}
	return nil
}
func (p CredentialPolicy) Permits(operation string, outcome ID) bool {
	allowed := false
	for _, op := range p.PermittedOperations {
		allowed = allowed || op == operation
	}
	if !allowed {
		return false
	}
	if outcome.IsZero() || len(p.AllowedOutcomeIDs) == 0 {
		return true
	}
	for _, id := range p.AllowedOutcomeIDs {
		if id == outcome {
			return true
		}
	}
	return false
}
func (p CredentialPolicy) PermitsKey(key ID) bool {
	for _, id := range p.AllowedSigningKeyIDs {
		if id == key {
			return true
		}
	}
	return false
}

type WorkAcceptancePolicy struct {
	NamespaceID              ID             `json:"namespace_id"`
	OutcomeID                *ID            `json:"outcome_id,omitempty"`
	WorkItemID               *ID            `json:"work_item_id,omitempty"`
	Version                  Version        `json:"revision,string"`
	AcceptanceFloor          AcceptanceMode `json:"acceptance_floor"`
	MaxActiveWorkContracts   int            `json:"max_active_work_contracts"`
	MaxActiveReviewContracts int            `json:"max_active_review_contracts"`
	UpdatedAt                time.Time      `json:"updated_at"`
}

func (p WorkAcceptancePolicy) Validate() error {
	if p.NamespaceID.Validate() != nil || p.Version.Validate() != nil || !p.AcceptanceFloor.Valid() || p.MaxActiveWorkContracts < 1 || p.MaxActiveWorkContracts > 100 || p.MaxActiveReviewContracts < 1 || p.MaxActiveReviewContracts > 100 {
		return NewError(ErrorCodeInvalidArgument, "invalid acceptance policy")
	}
	if p.OutcomeID != nil && p.OutcomeID.Validate() != nil {
		return NewError(ErrorCodeInvalidArgument, "invalid acceptance Outcome")
	}
	if p.WorkItemID != nil && (p.OutcomeID == nil || p.WorkItemID.Validate() != nil) {
		return NewError(ErrorCodeInvalidArgument, "invalid acceptance WorkItem")
	}
	return nil
}
