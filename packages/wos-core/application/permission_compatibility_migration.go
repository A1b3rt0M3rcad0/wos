package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type PermissionMigrationPreviewRequest struct {
	NamespaceID  domain.ID
	PrincipalID  string
	CredentialID domain.ID
}

type PermissionMigrationPreview struct {
	Snapshot                    ports.PermissionMigrationSnapshot
	MapVersion                  string
	GrantCapabilitiesAfter      []ports.Permission
	PolicyCapabilitiesAfter     []ports.Permission
	EffectiveCapabilitiesBefore []ports.Permission
	EffectiveCapabilitiesAfter  []ports.Permission
}

type PermissionMigrationApplyRequest struct {
	NamespaceID              domain.ID
	PrincipalID              string
	CredentialID             domain.ID
	ExpectedNamespaceVersion domain.Version
	ExpectedPolicyVersion    domain.Version
	PolicyVersionPresent     bool
	Reason                   string
}

func (r PermissionMigrationApplyRequest) validate() error {
	if r.NamespaceID.Validate() != nil || strings.TrimSpace(r.PrincipalID) == "" || len(r.PrincipalID) > 256 {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "permission migration target is invalid")
	}
	if r.ExpectedNamespaceVersion == 0 {
		return domain.NewError(domain.ErrorCodeInvalidVersion, "expected_namespace_version is required")
	}
	if r.CredentialID.IsZero() == r.PolicyVersionPresent {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "credential_id and expected policy version presence must agree")
	}
	if !r.PolicyVersionPresent && r.ExpectedPolicyVersion != 0 {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "policy version is only valid with a credential target")
	}
	if r.PolicyVersionPresent && r.ExpectedPolicyVersion == 0 {
		// Zero deliberately means that the credential has no policy yet.
	}
	reason := strings.TrimSpace(r.Reason)
	if len(reason) < 3 || len(reason) > 1000 {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "migration reason must contain 3 to 1000 bytes")
	}
	return nil
}

// PreviewPermissionMigration returns a read-only, per-principal compatibility
// view. It never changes grants, policies, revisions, or audit receipts.
func (s SecurityService) PreviewPermissionMigration(ctx context.Context, request PermissionMigrationPreviewRequest) (PermissionMigrationPreview, error) {
	if request.NamespaceID.Validate() != nil || strings.TrimSpace(request.PrincipalID) == "" || len(request.PrincipalID) > 256 {
		return PermissionMigrationPreview{}, domain.NewError(domain.ErrorCodeInvalidArgument, "permission migration target is invalid")
	}
	if err := s.Require(ctx, request.NamespaceID, ports.PermissionPermissionsManage); err != nil {
		return PermissionMigrationPreview{}, err
	}
	reader, ok := s.Store.(ports.PermissionCompatibilityStore)
	if !ok {
		return PermissionMigrationPreview{}, domain.NewError(domain.ErrorCodeInvalidConfig, "permission compatibility migration is unavailable")
	}
	snapshot, err := reader.ReadPermissionMigration(ctx, ports.PermissionMigrationRead{NamespaceID: request.NamespaceID, PrincipalID: request.PrincipalID, CredentialID: request.CredentialID})
	if err != nil {
		return PermissionMigrationPreview{}, err
	}
	preview := PermissionMigrationPreview{Snapshot: snapshot, MapVersion: PermissionCompatibilityMapVersion}
	preview.GrantCapabilitiesAfter = legacyCompatibilityCapabilities(snapshot.Grant.Permissions)
	var policyOperations []ports.Permission
	policyPresent := snapshot.Policy != nil
	if policyPresent {
		policyOperations = make([]ports.Permission, len(snapshot.Policy.PermittedOperations))
		for i, operation := range snapshot.Policy.PermittedOperations {
			policyOperations[i] = ports.Permission(operation)
		}
		preview.PolicyCapabilitiesAfter = legacyCompatibilityCapabilities(policyOperations)
	}
	policyCompatibility := uint32(0)
	if policyPresent {
		policyCompatibility = snapshot.Policy.PermissionCompatibilityVersion
	}
	preview.EffectiveCapabilitiesBefore = effectivePermissionSet(snapshot.Grant.Permissions, snapshot.Grant.PermissionCompatibilityVersion, policyOperations, policyCompatibility, policyPresent)
	preview.EffectiveCapabilitiesAfter = effectivePermissionIntersection(snapshot.Grant.Permissions, policyOperations, policyPresent)
	return preview, nil
}

// ApplyPermissionMigration marks the grant and optional CredentialPolicy for
// the frozen v1 command crosswalk. It does not rewrite permission lists or
// create absent policies; storage applies both markers under CAS atomically.
func (s SecurityService) ApplyPermissionMigration(ctx context.Context, key string, request PermissionMigrationApplyRequest) (ports.PermissionMigrationResult, error) {
	var empty ports.PermissionMigrationResult
	if err := request.validate(); err != nil {
		return empty, err
	}
	if err := domain.ValidateIdempotencyKey(key); err != nil {
		return empty, err
	}
	if err := s.Require(ctx, request.NamespaceID, ports.PermissionPermissionsManage); err != nil {
		return empty, err
	}
	identity, ok := IdentityFromContext(ctx)
	if !ok || identity.PrincipalID == "" || identity.NamespaceID != request.NamespaceID {
		return empty, domain.NewError(domain.ErrorCodeForbidden, "authenticated namespace identity is required")
	}
	store, ok := s.Store.(ports.PermissionCompatibilityStore)
	if !ok {
		return empty, domain.NewError(domain.ErrorCodeInvalidConfig, "permission compatibility migration is unavailable")
	}
	intent := struct {
		NamespaceID              domain.ID      `json:"namespace_id"`
		PrincipalID              string         `json:"principal_id"`
		CredentialID             domain.ID      `json:"credential_id,omitempty"`
		ExpectedNamespaceVersion domain.Version `json:"expected_namespace_version"`
		ExpectedPolicyVersion    domain.Version `json:"expected_policy_version"`
		PolicyVersionPresent     bool           `json:"policy_version_present"`
		Reason                   string         `json:"reason"`
		MapVersion               string         `json:"map_version"`
	}{request.NamespaceID, request.PrincipalID, request.CredentialID, request.ExpectedNamespaceVersion, request.ExpectedPolicyVersion, request.PolicyVersionPresent, strings.TrimSpace(request.Reason), PermissionCompatibilityMapVersion}
	raw, err := json.Marshal(intent)
	if err != nil {
		return empty, err
	}
	hash := sha256.Sum256(raw)
	now := time.Now().UTC()
	if s.Clock != nil {
		now = s.Clock.Now().UTC()
	}
	result, err := store.ApplyPermissionMigration(ctx, ports.PermissionMigrationRequest{Command: ports.PermissionMigrationApply{
		NamespaceID: request.NamespaceID, PrincipalID: request.PrincipalID, CredentialID: request.CredentialID,
		ExpectedNamespaceVersion: request.ExpectedNamespaceVersion, ExpectedPolicyVersion: request.ExpectedPolicyVersion,
		PolicyVersionPresent: request.PolicyVersionPresent, CallerPrincipalID: identity.PrincipalID,
		CallerActor: identity.Actor, CallerCredentialDigest: identity.CredentialDigest,
		IdempotencyKey: key, Fingerprint: hex.EncodeToString(hash[:]), Reason: strings.TrimSpace(request.Reason), Now: now,
	}})
	return result, err
}
