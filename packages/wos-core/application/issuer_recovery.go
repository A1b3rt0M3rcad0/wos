package application

import (
	"context"
	"encoding/json"
	"strings"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

type IssuerRecoveryIntent struct {
	ServerID            d.ID   `json:"server_id"`
	PreviousIssuerID    d.ID   `json:"previous_issuer_id"`
	PreviousFingerprint string `json:"previous_fingerprint"`
	NewIssuerID         d.ID   `json:"new_issuer_id"`
	NewFingerprint      string `json:"new_fingerprint"`
	Reason              string `json:"reason"`
}

// RecoverServerIssuer is trusted deployment composition with direct storage
// access. It deliberately has no Service command, HTTP/MCP route or tenant grant.
// The local signer is tested before the transaction, and is never installed into
// an already serving Service. Restart replicas with the approved new signer.
func RecoverServerIssuer(ctx context.Context, manager ports.TransactionManager, clock ports.Clock, intent IssuerRecoveryIntent, issuer ports.LocalContractSigner) (d.IssuerRecovery, bool, error) {
	var zero d.IssuerRecovery
	if manager == nil || clock == nil || issuer == nil || intent.ServerID.Validate() != nil || intent.PreviousIssuerID.Validate() != nil || intent.NewIssuerID.Validate() != nil || intent.PreviousIssuerID == intent.NewIssuerID || strings.TrimSpace(intent.Reason) == "" || len(intent.Reason) > 4096 {
		return zero, false, d.NewError(d.ErrorCodeInvalidArgument, "complete explicit host recovery intention required")
	}
	if _, ok := IdentityFromContext(ctx); ok {
		return zero, false, d.NewError(d.ErrorCodeForbidden, "issuer recovery is a host operation, not tenant administration")
	}
	replacement := issuer.Identity()
	if replacement.Validate() != nil || replacement.ID != intent.ServerID || replacement.IssuerKeyID != intent.NewIssuerID || replacement.Fingerprint != intent.NewFingerprint {
		return zero, false, d.NewError(d.ErrorCodeInvalidConfig, "replacement signer differs from approved public identity")
	}
	raw, err := signing.Canonical(intent)
	if err != nil {
		return zero, false, err
	}
	// Prove the replacement signer matches the declared public key; do not store
	// this challenge proof as a work authority or historical signature.
	challenge, err := signing.Canonical(struct {
		ProtocolVersion int                  `json:"protocol_version"`
		SignerKeyID     string               `json:"signer_key_id"`
		Operation       string               `json:"operation"`
		Intent          IssuerRecoveryIntent `json:"intent"`
	}{2, intent.NewIssuerID.String(), "host_issuer_recovery", intent})
	if err != nil {
		return zero, false, err
	}
	proofRaw, err := issuer.SignCanonical(string(signing.KeyEnrollment), challenge)
	if err != nil {
		return zero, false, err
	}
	var proof signing.Proof
	if err = signing.DecodeStrict(proofRaw, &proof, 4096); err != nil {
		return zero, false, err
	}
	document := signing.Document{Payload: json.RawMessage(challenge), Proof: proof}
	envelope, err := document.Envelope()
	if err != nil {
		return zero, false, err
	}
	public, err := replacement.PublicBytes()
	if err != nil {
		return zero, false, err
	}
	if _, err = signing.Verify(envelope, signing.KeyEnrollment, replacement.IssuerKeyID.String(), public); err != nil {
		return zero, false, signingError(err)
	}
	digest := signing.Digest(raw)
	u, err := manager.Begin(ctx)
	if err != nil {
		return zero, false, err
	}
	defer u.Rollback()
	identityRepo, err := serverIdentityRepository(u)
	if err != nil {
		return zero, false, err
	}
	repo, ok := identityRepo.(ports.IssuerRecoveryRepository)
	if !ok {
		return zero, false, d.NewError(d.ErrorCodeInvalidConfig, "issuer recovery storage unavailable")
	}
	previous, err := repo.LockServerIssuer(ctx)
	if err != nil {
		return zero, false, err
	}
	// A frozen receipt remains replayable after a subsequent rotation. Returning
	// it must never reinstall its superseded key.
	existing, err := repo.IssuerRecovery(ctx, intent.NewIssuerID)
	if err == nil {
		if existing.IntentDigest != digest || existing.Replacement != replacement {
			return zero, false, d.NewError(d.ErrorCodeIdempotencyConflict, "issuer recovery represents another intention")
		}
		return existing, true, nil
	}
	if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodeNotFound {
		return zero, false, err
	}
	if previous.ID != intent.ServerID || previous.IssuerKeyID != intent.PreviousIssuerID || previous.Fingerprint != intent.PreviousFingerprint || previous.CreatedAt != replacement.CreatedAt {
		return zero, false, d.NewError(d.ErrorCodeVersionConflict, "persistent issuer differs from expected predecessor")
	}
	now, err := securityTransactionTime(ctx, u, clock)
	if err != nil {
		return zero, false, err
	}
	record := d.IssuerRecovery{Previous: previous, Replacement: replacement, IntentDigest: digest, Reason: intent.Reason, RecordedAt: now}
	if err = record.Validate(); err != nil {
		return zero, false, err
	}
	if err = repo.ReplaceServerIssuer(ctx, record); err != nil {
		return zero, false, err
	}
	if err = u.Commit(); err != nil {
		return zero, false, err
	}
	return record, false, nil
}
