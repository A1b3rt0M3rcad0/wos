package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func (s *Store) ReadPermissionMigration(ctx context.Context, r ports.PermissionMigrationRead) (ports.PermissionMigrationSnapshot, error) {
	var out ports.PermissionMigrationSnapshot
	if r.NamespaceID.Validate() != nil || strings.TrimSpace(r.PrincipalID) == "" {
		return out, d.NewError(d.ErrorCodeInvalidArgument, "permission migration target is invalid")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return out, err
	}
	version, ok := s.security.Versions[r.NamespaceID]
	if !ok {
		return out, missingSigning()
	}
	grant, ok := s.security.Grants[securityKey(r.NamespaceID, r.PrincipalID)]
	if !ok {
		return out, missingSigning()
	}
	out.NamespaceVersion, out.Grant = version, deepCopy(grant)
	if r.CredentialID.IsZero() {
		return out, nil
	}
	credential, ok := findMemoryCredential(s.security.Credentials, r.NamespaceID, r.CredentialID)
	if !ok {
		return out, missingSigning()
	}
	if credential.PrincipalID != r.PrincipalID {
		return out, d.NewError(d.ErrorCodeInvalidScope, "credential does not belong to target principal")
	}
	out.Credential = &credential
	if policy, found := s.security.Policies[securityKey(r.NamespaceID, r.CredentialID.String())]; found {
		copy := deepCopy(policy)
		out.Policy = &copy
	}
	return out, nil
}

func (s *Store) ApplyPermissionMigration(ctx context.Context, r ports.PermissionMigrationRequest) (ports.PermissionMigrationResult, error) {
	var out ports.PermissionMigrationResult
	c := r.Command
	if c.NamespaceID.Validate() != nil || strings.TrimSpace(c.PrincipalID) == "" || strings.TrimSpace(c.CallerPrincipalID) == "" || c.ExpectedNamespaceVersion == 0 || strings.TrimSpace(c.IdempotencyKey) == "" || strings.TrimSpace(c.Fingerprint) == "" || len(strings.TrimSpace(c.Reason)) < 3 || len(strings.TrimSpace(c.Reason)) > 1000 {
		return out, d.NewError(d.ErrorCodeInvalidArgument, "invalid permission migration request")
	}
	if !c.CredentialID.IsZero() && !c.PolicyVersionPresent {
		return out, d.NewError(d.ErrorCodeInvalidArgument, "expected policy version presence is required")
	}
	if c.CredentialID.IsZero() && (c.PolicyVersionPresent || c.ExpectedPolicyVersion != 0) {
		return out, d.NewError(d.ErrorCodeInvalidArgument, "policy expectation requires a credential target")
	}
	if err := c.CallerActor.Validate(); err != nil {
		return out, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return out, err
	}
	version, ok := s.security.Versions[c.NamespaceID]
	if !ok {
		return out, missingSigning()
	}
	if err := authorizeMemoryPermissionMigration(s.security, c); err != nil {
		return out, err
	}
	receiptKey := securityKey(c.NamespaceID, c.CallerPrincipalID) + "/" + c.IdempotencyKey
	if previous, found := s.security.CompatibilityReceipts[receiptKey]; found {
		if previous.Fingerprint != c.Fingerprint {
			return out, d.NewError(d.ErrorCodeIdempotencyConflict, "idempotency key represents another permission migration")
		}
		if err := json.Unmarshal(previous.Payload, &out); err != nil {
			return out, err
		}
		out.IdempotentReplay = true
		return out, nil
	}
	if version != c.ExpectedNamespaceVersion {
		return out, d.NewError(d.ErrorCodeVersionConflict, "namespace version changed")
	}
	grantKey := securityKey(c.NamespaceID, c.PrincipalID)
	grant, ok := s.security.Grants[grantKey]
	if !ok {
		return out, missingSigning()
	}
	var policy *d.CredentialPolicy
	if !c.CredentialID.IsZero() {
		credential, exists := findMemoryCredential(s.security.Credentials, c.NamespaceID, c.CredentialID)
		if !exists {
			return out, missingSigning()
		}
		if credential.PrincipalID != c.PrincipalID {
			return out, d.NewError(d.ErrorCodeInvalidScope, "credential does not belong to target principal")
		}
		value, exists := s.security.Policies[securityKey(c.NamespaceID, c.CredentialID.String())]
		actual := d.Version(0)
		if exists {
			actual = value.Version
			policy = &value
		}
		if !c.PolicyVersionPresent || actual != c.ExpectedPolicyVersion {
			return out, d.NewError(d.ErrorCodeVersionConflict, "credential policy version changed")
		}
	}
	if grant.PermissionCompatibilityVersion == ports.PermissionCompatibilityLegacyV1 && (policy == nil || policy.PermissionCompatibilityVersion == d.PermissionCompatibilityLegacyV1) {
		out = ports.PermissionMigrationResult{NamespaceID: c.NamespaceID, PrincipalID: c.PrincipalID, CredentialID: c.CredentialID, NamespaceVersion: version, MapVersion: "legacy-v1", AlreadyMigrated: true}
		if policy != nil {
			out.PolicyVersion = policy.Version
		}
		return s.storeMemoryMigrationReceipt(receiptKey, c.Fingerprint, out)
	}
	next := version + 1
	if next == 0 {
		return out, d.NewError(d.ErrorCodeVersionConflict, "namespace version exhausted")
	}
	if policy != nil && policy.PermissionCompatibilityVersion != d.PermissionCompatibilityLegacyV1 && policy.Version == ^d.Version(0) {
		return out, d.NewError(d.ErrorCodeVersionConflict, "credential policy version exhausted")
	}
	grant.PermissionCompatibilityVersion = ports.PermissionCompatibilityLegacyV1
	s.security.Grants[grantKey] = deepCopy(grant)
	s.security.Versions[c.NamespaceID] = next
	policyMigrated := false
	policyVersion := d.Version(0)
	if policy != nil {
		policyVersion = policy.Version
		if policy.PermissionCompatibilityVersion != d.PermissionCompatibilityLegacyV1 {
			policy.Version++
			policy.PermissionCompatibilityVersion = d.PermissionCompatibilityLegacyV1
			s.security.Policies[securityKey(c.NamespaceID, c.CredentialID.String())] = deepCopy(*policy)
			policyMigrated = true
			policyVersion = policy.Version
		}
	}
	target := fmt.Sprintf("principal=%s credential=%s reason=%s", c.PrincipalID, c.CredentialID.String(), strings.TrimSpace(c.Reason))
	s.security.Audit[c.NamespaceID] = append(s.security.Audit[c.NamespaceID], ports.SecurityAudit{NamespaceVersion: next, PrincipalID: c.CallerPrincipalID, Actor: c.CallerActor, Operation: "permission_compatibility_migrate", Target: target, RecordedAt: c.Now.UTC()})
	out = ports.PermissionMigrationResult{NamespaceID: c.NamespaceID, PrincipalID: c.PrincipalID, CredentialID: c.CredentialID, NamespaceVersion: next, PolicyVersion: policyVersion, PolicyMigrated: policyMigrated, MapVersion: "legacy-v1"}
	return s.storeMemoryMigrationReceipt(receiptKey, c.Fingerprint, out)
}

func (s *Store) storeMemoryMigrationReceipt(key, fingerprint string, result ports.PermissionMigrationResult) (ports.PermissionMigrationResult, error) {
	payload, err := json.Marshal(result)
	if err != nil {
		return ports.PermissionMigrationResult{}, err
	}
	s.security.CompatibilityReceipts[key] = memoryReceipt{Fingerprint: fingerprint, Payload: payload}
	return result, nil
}

func authorizeMemoryPermissionMigration(state memorySecurityState, c ports.PermissionMigrationApply) error {
	grant, ok := state.Grants[securityKey(c.NamespaceID, c.CallerPrincipalID)]
	if !ok {
		return d.NewError(d.ErrorCodeForbidden, "permission migration authority denied")
	}
	allowed := false
	for _, permission := range grant.Permissions {
		allowed = allowed || permission == ports.PermissionPermissionsManage
	}
	if !allowed {
		return d.NewError(d.ErrorCodeForbidden, "permission migration requires permissions:manage")
	}
	if c.CallerCredentialDigest == "" {
		return nil
	}
	credential, ok := state.Credentials[c.CallerCredentialDigest]
	if !ok || credential.NamespaceID != c.NamespaceID || credential.PrincipalID != c.CallerPrincipalID || credential.Actor != c.CallerActor || credential.Revoked || !c.Now.Before(credential.ExpiresAt) {
		return d.NewError(d.ErrorCodeForbidden, "permission migration credential inactive")
	}
	if credential.ParentDigest != "" {
		parent, found := state.Credentials[credential.ParentDigest]
		if !found || parent.ParentDigest != "" || parent.NamespaceID != c.NamespaceID || parent.PrincipalID != c.CallerPrincipalID || parent.Actor != c.CallerActor || parent.Revoked || !c.Now.Before(parent.ExpiresAt) {
			return d.NewError(d.ErrorCodeForbidden, "permission migration session parent inactive")
		}
		credential = parent
	}
	if policy, found := state.Policies[securityKey(c.NamespaceID, credential.ID.String())]; found && !policy.Permits(string(ports.PermissionPermissionsManage), d.ID("")) {
		return d.NewError(d.ErrorCodeForbidden, "permission migration credential policy denies authority")
	}
	return nil
}

func findMemoryCredential(credentials map[string]ports.Credential, ns, id d.ID) (ports.Credential, bool) {
	for _, credential := range credentials {
		if credential.NamespaceID == ns && credential.ID == id {
			return credential, true
		}
	}
	return ports.Credential{}, false
}

var _ ports.PermissionCompatibilityStore = (*Store)(nil)
