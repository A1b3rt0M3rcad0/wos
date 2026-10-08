package application

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"strings"
	"time"
)

type SigningSecurityIntent struct {
	NamespaceID              d.ID                    `json:"namespace_id"`
	ExpectedNamespaceVersion d.Version               `json:"expected_namespace_version,string"`
	Operation                string                  `json:"operation"`
	PrincipalID              string                  `json:"principal_id,omitempty"`
	CredentialID             d.ID                    `json:"credential_id,omitempty"`
	KeyID                    d.ID                    `json:"key_id,omitempty"`
	ExpectedVersion          d.Version               `json:"expected_version,string"`
	EnrollmentID             d.ID                    `json:"enrollment_id,omitempty"`
	TTLSeconds               int                     `json:"ttl_seconds,omitempty"`
	ExpectedFingerprint      string                  `json:"expected_fingerprint,omitempty"`
	CredentialPolicy         *d.CredentialPolicy     `json:"credential_policy,omitempty"`
	AcceptancePolicy         *d.WorkAcceptancePolicy `json:"acceptance_policy,omitempty"`
	SeparationGroup          string                  `json:"separation_group,omitempty"`
	Reason                   string                  `json:"reason,omitempty"`
	Proof                    *signing.Envelope       `json:"proof,omitempty"`
}
type SigningSecurityResult struct {
	NamespaceVersion d.Version               `json:"namespace_version,string"`
	Enrollment       *d.SigningEnrollment    `json:"enrollment,omitempty"`
	Key              *d.SigningKey           `json:"key,omitempty"`
	CredentialPolicy *d.CredentialPolicy     `json:"credential_policy,omitempty"`
	AcceptancePolicy *d.WorkAcceptancePolicy `json:"acceptance_policy,omitempty"`
	IdempotentReplay bool                    `json:"idempotent_replay"`
}

func signingRepository(uow ports.UnitOfWork) (ports.SigningIdentityRepository, error) {
	r, ok := uow.(ports.SigningIdentityUnitOfWork)
	if !ok {
		return nil, d.NewError(d.ErrorCodeInvalidConfig, "signing identity repository unavailable")
	}
	return r.SigningIdentity(), nil
}
func signingError(err error) error {
	if e, ok := err.(*signing.Error); ok {
		return d.NewError(d.ErrorCode(e.Code), e.Message)
	}
	return err
}
func securityTransactionTime(ctx context.Context, uow ports.UnitOfWork, clock ports.Clock) (time.Time, error) {
	if c, ok := uow.(ports.TransactionClock); ok {
		return c.EvaluationTime(ctx)
	}
	return clock.Now().UTC(), nil
}
func (s SecurityService) SigningMutation(ctx context.Context, key string, cmd SigningSecurityIntent) (SigningSecurityResult, error) {
	var result SigningSecurityResult
	if cmd.NamespaceID.Validate() != nil || d.ValidateIdempotencyKey(key) != nil {
		return result, d.NewError(d.ErrorCodeInvalidArgument, "namespace and idempotency key required")
	}
	id, ok := IdentityFromContext(ctx)
	if !ok || id.NamespaceID != cmd.NamespaceID || id.CredentialDigest == "" {
		return result, d.NewError(d.ErrorCodeForbidden, "authenticated scoped credential required")
	}
	permission := ports.PermissionNamespaceAdmin
	switch cmd.Operation {
	case "register_signing_key":
		permission = ports.PermissionSigningKeyEnroll
	case "create_rotation_enrollment":
		permission = ports.PermissionSigningKeyRotate
	case "create_enrollment", "set_credential_policy", "set_acceptance_policy", "set_review_group", "revoke_signing_key":
	default:
		return result, d.NewError(d.ErrorCodeInvalidArgument, "unknown signing security operation")
	}
	if err := s.Require(ctx, cmd.NamespaceID, permission); err != nil {
		return result, err
	}
	manager, ok := s.Store.(ports.TransactionManager)
	if !ok {
		return result, d.NewError(d.ErrorCodeInvalidConfig, "transactional signing administration unavailable")
	}
	uow, err := manager.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer uow.Rollback()
	repo, err := signingRepository(uow)
	if err != nil {
		return result, err
	}
	version, err := repo.LockNamespace(ctx, cmd.NamespaceID)
	if err != nil {
		return result, err
	}
	if err = s.authorizeSigningInUnitOfWork(ctx, uow, ports.AuthorizationRequest{NamespaceID: cmd.NamespaceID, PrincipalID: id.PrincipalID, Permission: permission}); err != nil {
		return result, err
	}
	fingerprint, err := d.SemanticDigest(cmd)
	if err != nil {
		return result, err
	}
	previous, raw, err := repo.Receipt(ctx, cmd.NamespaceID, id.PrincipalID, key)
	if err == nil {
		if previous != fingerprint {
			return result, d.NewError(d.ErrorCodeIdempotencyConflict, "security key represents another intent")
		}
		if err = json.Unmarshal(raw, &result); err != nil {
			return result, err
		}
		result.IdempotentReplay = true
		return result, nil
	}
	if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodeNotFound {
		return result, err
	}
	if permission == ports.PermissionNamespaceAdmin && (cmd.ExpectedNamespaceVersion != version || cmd.ExpectedNamespaceVersion < 1) {
		return result, d.NewError(d.ErrorCodeVersionConflict, "namespace security version changed")
	}
	now, err := securityTransactionTime(ctx, uow, s.Clock)
	if err != nil {
		return result, err
	}
	target := ""
	switch cmd.Operation {
	case "set_credential_policy":
		if cmd.CredentialPolicy == nil {
			return result, d.NewError(d.ErrorCodeInvalidArgument, "credential policy required")
		}
		policy := *cmd.CredentialPolicy
		if policy.NamespaceID != cmd.NamespaceID {
			return result, d.NewError(d.ErrorCodeInvalidScope, "policy namespace differs")
		}
		policy.Version = cmd.ExpectedVersion + 1
		policy.UpdatedAt = now
		for _, op := range policy.PermittedOperations {
			if !ports.Permission(op).Valid() {
				return result, d.NewError(d.ErrorCodeInvalidArgument, "unknown policy permission")
			}
		}
		if err = repo.SaveCredentialPolicy(ctx, policy, cmd.ExpectedVersion); err != nil {
			return result, err
		}
		result.CredentialPolicy = &policy
		target = policy.CredentialID.String()
	case "set_acceptance_policy":
		if cmd.AcceptancePolicy == nil {
			return result, d.NewError(d.ErrorCodeInvalidArgument, "acceptance policy required")
		}
		policy := *cmd.AcceptancePolicy
		if policy.NamespaceID != cmd.NamespaceID {
			return result, d.NewError(d.ErrorCodeInvalidScope, "policy namespace differs")
		}
		policy.Version = cmd.ExpectedVersion + 1
		policy.UpdatedAt = now
		if err = repo.SaveAcceptancePolicy(ctx, policy, cmd.ExpectedVersion); err != nil {
			return result, err
		}
		result.AcceptancePolicy = &policy
		target = "acceptance_policy"
	case "set_review_group":
		if err = repo.SetPrincipalGroup(ctx, cmd.NamespaceID, cmd.PrincipalID, cmd.SeparationGroup); err != nil {
			return result, err
		}
		target = cmd.PrincipalID
	case "create_enrollment":
		if s.ServerID == "" {
			return result, d.NewError(d.ErrorCodeInvalidConfig, "persistent server identity required")
		}
		credential, e := repo.Credential(ctx, cmd.NamespaceID, cmd.CredentialID)
		if e != nil {
			return result, e
		}
		if credential.PrincipalID != cmd.PrincipalID || credential.Revoked || !now.Before(credential.ExpiresAt) || credential.ParentDigest != "" {
			return result, d.NewError(d.ErrorCodeForbidden, "enrollment target credential inactive/different")
		}
		ttl := cmd.TTLSeconds
		if ttl == 0 {
			ttl = 300
		}
		if ttl < 30 || ttl > 600 {
			return result, d.NewError(d.ErrorCodeInvalidArgument, "enrollment TTL outside 30..600")
		}
		enrollmentID, e := s.IDs.NewID()
		if e != nil {
			return result, e
		}
		keyID, e := s.IDs.NewID()
		if e != nil {
			return result, e
		}
		nonce := make([]byte, 32)
		if _, e = rand.Read(nonce); e != nil {
			return result, e
		}
		enrollment := d.SigningEnrollment{ID: enrollmentID, NamespaceID: cmd.NamespaceID, PrincipalID: cmd.PrincipalID, CredentialID: credential.ID, KeyID: keyID, Purpose: "agent", Nonce: base64.RawURLEncoding.EncodeToString(nonce), ExpiresAt: now.Add(time.Duration(ttl) * time.Second), AuthorizedBy: id.PrincipalID, ExpectedFingerprint: cmd.ExpectedFingerprint, Version: 1}
		if err = repo.SaveEnrollment(ctx, enrollment, 0); err != nil {
			return result, err
		}
		result.Enrollment = &enrollment
		target = enrollment.ID.String()
	case "create_rotation_enrollment":
		if cmd.Proof == nil || s.ServerID == "" {
			return result, d.NewError(d.ErrorCodeInvalidArgument, "signed rotation authorization required")
		}
		keyID, e := d.ParseID(cmd.Proof.SignaturesKeyID())
		if e != nil {
			return result, e
		}
		old, e := repo.Key(ctx, cmd.NamespaceID, keyID)
		if e != nil {
			return result, e
		}
		if old.PrincipalID != id.PrincipalID || old.Status != "active" || old.Purpose != "agent" {
			return result, d.NewError(d.ErrorCodeForbidden, "previous signing key inactive/different")
		}
		public, e := old.PublicBytes()
		if e != nil {
			return result, e
		}
		payload, _, e := signing.Decode[signing.EnrollmentPayload](*cmd.Proof, signing.KeyEnrollment, keyID.String(), public)
		if e != nil {
			return result, signingError(e)
		}
		expires, e := time.Parse(time.RFC3339Nano, payload.ExpiresAt)
		if e != nil || !expires.After(now) || expires.After(now.Add(10*time.Minute)) {
			return result, d.NewError(d.ErrorCodeInvalidArgument, "rotation authorization expired/unbounded")
		}
		if payload.ServerID != s.ServerID || payload.NamespaceID != cmd.NamespaceID.String() || payload.PrincipalID != id.PrincipalID || payload.Purpose != "rotation_authorization" || payload.PreviousKeyID != keyID.String() || payload.OutcomeID != "" {
			return result, d.NewError(d.ErrorCodeForbidden, "rotation proof binding differs")
		}
		enrollmentID, e := d.ParseID(payload.EnrollmentID)
		if e != nil {
			return result, e
		}
		newID, e := d.ParseID(payload.NewKeyID)
		if e != nil {
			return result, e
		}
		pub, e := base64.StdEncoding.Strict().DecodeString(payload.PublicKey)
		if e != nil || len(pub) != 32 {
			return result, d.NewError(d.ErrorCodeInvalidArgument, "new public key invalid")
		}
		fp, e := signing.Fingerprint(pub)
		if e != nil {
			return result, signingError(e)
		}
		credential, e := repo.CredentialByDigest(ctx, id.CredentialDigest)
		if e != nil {
			return result, e
		}
		if credential.ParentDigest != "" {
			return result, d.NewError(d.ErrorCodeForbidden, "session cannot enroll signing identity")
		}
		policy, e := repo.CredentialPolicy(ctx, cmd.NamespaceID, credential.ID)
		if e != nil || !policy.PermitsKey(keyID) {
			return result, d.NewError(d.ErrorCodeForbidden, "credential does not authorize prior key")
		}
		nonce := make([]byte, 32)
		if _, err = rand.Read(nonce); err != nil {
			return result, err
		}
		enrollment := d.SigningEnrollment{ID: enrollmentID, NamespaceID: cmd.NamespaceID, PrincipalID: id.PrincipalID, CredentialID: credential.ID, KeyID: newID, PreviousKeyID: &keyID, Purpose: "agent", Nonce: base64.RawURLEncoding.EncodeToString(nonce), ExpiresAt: expires, AuthorizedBy: id.PrincipalID, ExpectedFingerprint: fp, Version: 1}
		if err = repo.SaveEnrollment(ctx, enrollment, 0); err != nil {
			return result, err
		}
		result.Enrollment = &enrollment
		target = enrollment.ID.String()
	case "register_signing_key":
		if cmd.Proof == nil || s.ServerID == "" {
			return result, d.NewError(d.ErrorCodeInvalidArgument, "signed possession proof required")
		}
		enrollment, e := repo.Enrollment(ctx, cmd.NamespaceID, cmd.EnrollmentID)
		if e != nil {
			return result, e
		}
		credential, e := repo.CredentialByDigest(ctx, id.CredentialDigest)
		if e != nil {
			return result, e
		}
		if enrollment.Consumed || !now.Before(enrollment.ExpiresAt) || enrollment.PrincipalID != id.PrincipalID || enrollment.CredentialID != credential.ID || credential.ParentDigest != "" {
			return result, d.NewError(d.ErrorCodeForbidden, "enrollment expired/consumed/different")
		}
		document, e := signing.ToDocument(*cmd.Proof)
		if e != nil {
			return result, signingError(e)
		}
		var candidate signing.EnrollmentPayload
		if e = signing.DecodeStrict(document.Payload, &candidate, signing.MaxPayloadBytes); e != nil {
			return result, signingError(e)
		}
		public, e := base64.StdEncoding.Strict().DecodeString(candidate.PublicKey)
		if e != nil || len(public) != 32 {
			return result, d.NewError(d.ErrorCodeInvalidArgument, "invalid proposed public key")
		}
		payload, verified, e := signing.Decode[signing.EnrollmentPayload](*cmd.Proof, signing.KeyEnrollment, enrollment.KeyID.String(), public)
		if e != nil {
			return result, signingError(e)
		}
		previousID := ""
		if enrollment.PreviousKeyID != nil {
			previousID = enrollment.PreviousKeyID.String()
		}
		if payload.ServerID != s.ServerID || payload.NamespaceID != cmd.NamespaceID.String() || payload.PrincipalID != id.PrincipalID || payload.EnrollmentID != enrollment.ID.String() || payload.Nonce != enrollment.Nonce || payload.Purpose != enrollment.Purpose || payload.PreviousKeyID != previousID || payload.ExpiresAt != enrollment.ExpiresAt.UTC().Format(time.RFC3339Nano) || payload.OutcomeID != "" || payload.NewKeyID != "" {
			return result, d.NewError(d.ErrorCodeForbidden, "possession proof challenge/binding differs")
		}
		fp, e := signing.Fingerprint(public)
		if e != nil {
			return result, signingError(e)
		}
		if enrollment.ExpectedFingerprint != "" && enrollment.ExpectedFingerprint != fp {
			return result, d.NewError(d.ErrorCodeForbidden, "enrollment fingerprint differs")
		}
		policy, e := repo.CredentialPolicy(ctx, cmd.NamespaceID, credential.ID)
		if e != nil || !policy.Permits(string(ports.PermissionSigningKeyEnroll), "") {
			return result, d.NewError(d.ErrorCodeForbidden, "explicit enrollment credential policy required")
		}
		registered := d.SigningKey{ID: enrollment.KeyID, NamespaceID: cmd.NamespaceID, PrincipalID: id.PrincipalID, Purpose: "agent", Algorithm: "Ed25519", PublicKey: base64.StdEncoding.EncodeToString(public), Fingerprint: fp, Status: "active", Version: 1, CreatedAt: now, UpdatedAt: now}
		if err = repo.SaveKey(ctx, registered, 0); err != nil {
			return result, err
		}
		if enrollment.PreviousKeyID != nil {
			old, e := repo.Key(ctx, cmd.NamespaceID, *enrollment.PreviousKeyID)
			if e != nil {
				return result, e
			}
			if old.PrincipalID != id.PrincipalID || old.Status != "active" || !policy.PermitsKey(old.ID) {
				return result, d.NewError(d.ErrorCodeForbidden, "previous rotation key no longer authorized")
			}
			oldVersion := old.Version
			old.Version++
			old.Status = "retired"
			old.UpdatedAt = now
			old.Reason = "authorized rotation"
			if e = repo.SaveKey(ctx, old, oldVersion); e != nil {
				return result, e
			}
		}
		oldPolicyVersion := policy.Version
		policy.Version++
		policy.UpdatedAt = now
		if !policy.PermitsKey(registered.ID) {
			policy.AllowedSigningKeyIDs = append(policy.AllowedSigningKeyIDs, registered.ID)
		}
		if err = repo.SaveCredentialPolicy(ctx, policy, oldPolicyVersion); err != nil {
			return result, err
		}
		oldVersion := enrollment.Version
		enrollment.Version++
		enrollment.Consumed = true
		enrollment.AcceptedDigest = signing.Digest(verified)
		if err = repo.SaveEnrollment(ctx, enrollment, oldVersion); err != nil {
			return result, err
		}
		result.Enrollment = &enrollment
		result.Key = &registered
		result.CredentialPolicy = &policy
		target = registered.ID.String()
	case "revoke_signing_key":
		if strings.TrimSpace(cmd.Reason) == "" || len(cmd.Reason) > 4096 {
			return result, d.NewError(d.ErrorCodeInvalidArgument, "key revocation reason required")
		}
		registered, e := repo.Key(ctx, cmd.NamespaceID, cmd.KeyID)
		if e != nil {
			return result, e
		}
		if registered.Version != cmd.ExpectedVersion {
			return result, d.NewError(d.ErrorCodeVersionConflict, "key version changed")
		}
		registered.Version++
		registered.Status = "revoked"
		registered.Reason = cmd.Reason
		registered.UpdatedAt = now
		if err = repo.SaveKey(ctx, registered, cmd.ExpectedVersion); err != nil {
			return result, err
		}
		result.Key = &registered
		target = registered.ID.String()
	}
	result.NamespaceVersion, err = repo.Audit(ctx, cmd.NamespaceID, id.PrincipalID, id.Actor, cmd.Operation, target, now)
	if err != nil {
		return result, err
	}
	raw, err = json.Marshal(result)
	if err != nil {
		return result, err
	}
	if err = repo.SaveReceipt(ctx, cmd.NamespaceID, id.PrincipalID, key, fingerprint, raw); err != nil {
		return result, err
	}
	if err = uow.Commit(); err != nil {
		return result, err
	}
	return result, nil
}

// SigningOperation allows currently authorized receipt lookup independently of a
// retired signing key or permissions to perform another enrollment.
func (s SecurityService) SigningOperation(ctx context.Context, key string) (SigningSecurityResult, error) {
	var result SigningSecurityResult
	id, ok := IdentityFromContext(ctx)
	if !ok {
		return result, d.NewError(d.ErrorCodeForbidden, "identity required")
	}
	if err := s.Require(ctx, id.NamespaceID, ports.PermissionStateRead); err != nil {
		return result, err
	}
	manager, ok := s.Store.(ports.TransactionManager)
	if !ok {
		return result, d.NewError(d.ErrorCodeInvalidConfig, "signing transaction unavailable")
	}
	uow, err := manager.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer uow.Rollback()
	if err = s.authorizeSigningInUnitOfWork(ctx, uow, ports.AuthorizationRequest{NamespaceID: id.NamespaceID, PrincipalID: id.PrincipalID, Permission: ports.PermissionStateRead}); err != nil {
		return result, err
	}
	repo, err := signingRepository(uow)
	if err != nil {
		return result, err
	}
	_, raw, err := repo.Receipt(ctx, id.NamespaceID, id.PrincipalID, key)
	if err != nil {
		return result, err
	}
	err = json.Unmarshal(raw, &result)
	return result, err
}

// SigningIdentityView reveals only the current credential's public identity.
// Private signing material never enters WOS or this response.
type SigningIdentityView struct {
	ServerID     string               `json:"server_id"`
	NamespaceID  d.ID                 `json:"namespace_id"`
	PrincipalID  string               `json:"principal_id"`
	CredentialID d.ID                 `json:"credential_id"`
	Policy       *d.CredentialPolicy  `json:"credential_policy,omitempty"`
	Keys         []d.SigningKey       `json:"keys"`
	Enrollment   *d.SigningEnrollment `json:"enrollment,omitempty"`
}

func (s SecurityService) SigningIdentity(ctx context.Context, enrollmentID *d.ID) (SigningIdentityView, error) {
	var result SigningIdentityView
	id, ok := IdentityFromContext(ctx)
	if !ok {
		return result, d.NewError(d.ErrorCodeForbidden, "authenticated identity required")
	}
	if enrollmentID != nil && enrollmentID.Validate() != nil {
		return result, d.NewError(d.ErrorCodeInvalidArgument, "invalid enrollment identifier")
	}
	if err := s.Require(ctx, id.NamespaceID, ports.PermissionStateRead); err != nil {
		return result, err
	}
	manager, ok := s.Store.(ports.TransactionManager)
	if !ok {
		return result, d.NewError(d.ErrorCodeInvalidConfig, "transactional signing identity unavailable")
	}
	uow, err := manager.Begin(ctx)
	if err != nil {
		return result, err
	}
	defer uow.Rollback()
	if err = s.authorizeSigningInUnitOfWork(ctx, uow, ports.AuthorizationRequest{NamespaceID: id.NamespaceID, PrincipalID: id.PrincipalID, Permission: ports.PermissionStateRead}); err != nil {
		return result, err
	}
	repo, err := signingRepository(uow)
	if err != nil {
		return result, err
	}
	credential, err := repo.CredentialByDigest(ctx, id.CredentialDigest)
	if err != nil {
		return result, err
	}
	if credential.ParentDigest != "" {
		credential, err = repo.CredentialByDigest(ctx, credential.ParentDigest)
		if err != nil {
			return result, err
		}
	}
	result = SigningIdentityView{ServerID: s.ServerID, NamespaceID: id.NamespaceID, PrincipalID: id.PrincipalID, CredentialID: credential.ID}
	policy, err := repo.CredentialPolicy(ctx, id.NamespaceID, credential.ID)
	if err == nil {
		result.Policy = &policy
	} else if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodeNotFound {
		return result, err
	}
	result.Keys, err = repo.Keys(ctx, id.NamespaceID, id.PrincipalID, 101)
	if err != nil {
		return result, err
	}
	if enrollmentID != nil {
		enrollment, e := repo.Enrollment(ctx, id.NamespaceID, *enrollmentID)
		if e != nil {
			return result, e
		}
		if enrollment.PrincipalID != id.PrincipalID || enrollment.CredentialID != credential.ID {
			return result, d.NewError(d.ErrorCodeNotFound, "enrollment unavailable")
		}
		result.Enrollment = &enrollment
	}
	return result, nil
}

// Signed operations evaluate access at the same authoritative transaction time
// as challenge/lease checks; a stale caller clock cannot extend a credential.
func (s SecurityService) authorizeSigningInUnitOfWork(ctx context.Context, uow ports.UnitOfWork, request ports.AuthorizationRequest) error {
	if err := request.Validate(); err != nil {
		return err
	}
	id, ok := IdentityFromContext(ctx)
	if !ok || id.NamespaceID != request.NamespaceID || id.PrincipalID != request.PrincipalID {
		return d.NewError(d.ErrorCodeForbidden, "credential scope differs")
	}
	access, ok := uow.(ports.AccessSnapshotUnitOfWork)
	if !ok {
		return d.NewError(d.ErrorCodeInvalidConfig, "transactional access unavailable")
	}
	now, err := securityTransactionTime(ctx, uow, s.Clock)
	if err != nil {
		return err
	}
	return access.AuthorizeAccessSnapshot(ctx, ports.AccessSnapshotRequest{Authorization: request, CredentialDigest: id.CredentialDigest, Actor: id.Actor, Now: now})
}
