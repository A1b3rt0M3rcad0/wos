package domain

import (
	"encoding/base64"
	"time"
)

// ServerIdentity is persistent public trust metadata. Its private key is held
// only by the deployment's local signer and never stored in this record.
type ServerIdentity struct {
	ID          ID        `json:"server_id"`
	IssuerKeyID ID        `json:"issuer_key_id"`
	PublicKey   string    `json:"public_key"`
	Fingerprint string    `json:"fingerprint"`
	CreatedAt   time.Time `json:"created_at"`
}

func (s ServerIdentity) Validate() error {
	if s.ID.Validate() != nil || s.IssuerKeyID.Validate() != nil || s.CreatedAt.IsZero() {
		return NewError(ErrorCodeInvalidConfig, "invalid persistent server identity")
	}
	k := SigningKey{ID: s.IssuerKeyID, NamespaceID: s.ID, PrincipalID: s.PrincipalID(), Purpose: "issuer", Algorithm: "Ed25519", PublicKey: s.PublicKey, Fingerprint: s.Fingerprint, Version: 1, Status: "active", CreatedAt: s.CreatedAt, UpdatedAt: s.CreatedAt}
	return k.Validate()
}
func (s ServerIdentity) PrincipalID() string { return "wos-server:" + s.ID.String() }
func (s ServerIdentity) PublicBytes() ([]byte, error) {
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return base64.StdEncoding.DecodeString(s.PublicKey)
}
