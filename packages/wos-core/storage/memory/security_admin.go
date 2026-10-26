package memory

import (
	"context"
	"encoding/json"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

var _ ports.SecurityAdministrationStore = (*Store)(nil)

func (s *Store) ReadSecurityAdminReceipt(ctx context.Context, r ports.SecurityAdminRequest) (ports.SecurityAdminResult, bool, error) {
	var empty ports.SecurityAdminResult
	if err := ctx.Err(); err != nil {
		return empty, false, err
	}
	if err := d.ValidateIdempotencyKey(r.IdempotencyKey); err != nil {
		return empty, false, err
	}
	if (r.Command.Operation != "issue_credential" && r.Command.Operation != "rotate_credential") || r.Command.NamespaceID.Validate() != nil || r.Command.ExpectedVersion == 0 || strings.TrimSpace(r.PrincipalID) == "" || r.Fingerprint == "" || r.Now.IsZero() {
		return empty, false, d.NewError(d.ErrorCodeInvalidArgument, "invalid credential receipt request")
	}
	required, ok := ports.SecurityAdminPermission(r.Command.Operation)
	if !ok {
		return empty, false, d.NewError(d.ErrorCodeInvalidArgument, "unknown security operation")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return empty, false, err
	}
	state := s.security
	if _, active := state.Namespaces[r.Command.NamespaceID]; !active {
		return empty, false, d.NewError(d.ErrorCodeNotFound, "namespace not found")
	}
	grant, exists := state.Grants[securityKey(r.Command.NamespaceID, r.PrincipalID)]
	if !exists {
		return empty, false, d.NewError(d.ErrorCodeForbidden, "namespace administration denied")
	}
	authorized, delegate := false, false
	for _, permission := range grant.Permissions {
		authorized = authorized || permission == required
		delegate = delegate || permission == ports.PermissionActorDelegate
	}
	if !authorized {
		return empty, false, d.NewError(d.ErrorCodeForbidden, "security operation permission denied")
	}
	if r.Command.Credential != nil && r.Command.Credential.Actor != r.Actor && !delegate {
		return empty, false, d.NewError(d.ErrorCodeForbidden, "actor delegation denied")
	}
	if _, err := authorizeMemorySecurityAdminRead(state, ports.SecurityAdminReadAccess{NamespaceID: r.Command.NamespaceID, PrincipalID: r.PrincipalID, CredentialID: r.CallerCredentialID, CredentialDigest: r.CredentialDigest, Actor: r.Actor, Permissions: []ports.Permission{required}, Now: r.Now}); err != nil {
		return empty, false, err
	}
	receiptKey := securityKey(r.Command.NamespaceID, r.PrincipalID) + "/" + r.IdempotencyKey
	previous, found := state.Receipts[receiptKey]
	if !found {
		return empty, false, nil
	}
	if previous.Fingerprint != r.Fingerprint {
		return empty, false, d.NewError(d.ErrorCodeIdempotencyConflict, "key already represents a different intent")
	}
	if err := json.Unmarshal(previous.Payload, &empty); err != nil {
		return ports.SecurityAdminResult{}, false, err
	}
	empty.IdempotentReplay = true
	empty.TokenOmitted = empty.Credential != nil
	return empty, true, nil
}

func (s *Store) ApplySecurityAdmin(ctx context.Context, r ports.SecurityAdminRequest) (ports.SecurityAdminResult, error) {
	var empty ports.SecurityAdminResult
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	if err := d.ValidateIdempotencyKey(r.IdempotencyKey); err != nil {
		return empty, err
	}
	if r.Command.NamespaceID.Validate() != nil || r.Command.ExpectedVersion == 0 || strings.TrimSpace(r.PrincipalID) == "" || r.Fingerprint == "" || r.Now.IsZero() {
		return empty, d.NewError(d.ErrorCodeInvalidArgument, "invalid security administration request")
	}
	required, ok := ports.SecurityAdminPermission(r.Command.Operation)
	if !ok {
		return empty, d.NewError(d.ErrorCodeInvalidArgument, "unknown security operation")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return empty, err
	}

	state := cloneMemorySecurityState(s.security)
	version, exists := state.Versions[r.Command.NamespaceID]
	if _, active := state.Namespaces[r.Command.NamespaceID]; !exists || !active {
		return empty, d.NewError(d.ErrorCodeNotFound, "namespace not found")
	}
	grant, exists := state.Grants[securityKey(r.Command.NamespaceID, r.PrincipalID)]
	if !exists {
		return empty, d.NewError(d.ErrorCodeForbidden, "namespace administration denied")
	}
	authorized, delegate := false, false
	for _, permission := range grant.Permissions {
		authorized = authorized || permission == required
		delegate = delegate || permission == ports.PermissionActorDelegate
	}
	if !authorized {
		return empty, d.NewError(d.ErrorCodeForbidden, "security operation permission denied")
	}
	if r.Command.Credential != nil && r.Command.Credential.Actor != r.Actor && !delegate {
		return empty, d.NewError(d.ErrorCodeForbidden, "actor delegation denied")
	}
	rootCredentialID, err := authorizeMemorySecurityAdminRead(state, ports.SecurityAdminReadAccess{NamespaceID: r.Command.NamespaceID, PrincipalID: r.PrincipalID, CredentialID: r.CallerCredentialID, CredentialDigest: r.CredentialDigest, Actor: r.Actor, Permissions: []ports.Permission{required}, Now: r.Now})
	if err != nil {
		return empty, err
	}

	receiptKey := securityKey(r.Command.NamespaceID, r.PrincipalID) + "/" + r.IdempotencyKey
	if previous, found := state.Receipts[receiptKey]; found {
		if previous.Fingerprint != r.Fingerprint {
			return empty, d.NewError(d.ErrorCodeIdempotencyConflict, "key already represents a different intent")
		}
		var replay ports.SecurityAdminResult
		if err := json.Unmarshal(previous.Payload, &replay); err != nil {
			return empty, err
		}
		replay.IdempotentReplay = true
		replay.TokenOmitted = replay.Credential != nil
		return replay, nil
	}
	if version != r.Command.ExpectedVersion {
		return empty, d.NewError(d.ErrorCodeVersionConflict, "namespace version changed")
	}
	beforeState, err := memorySecurityAdminBefore(state, r.Command, r.Now)
	if err != nil {
		return empty, err
	}
	result := ports.SecurityAdminResult{}
	target := ""
	switch r.Command.Operation {
	case "create_namespace":
		n, credential := r.Command.NewNamespace, r.Command.Credential
		if n == nil || credential == nil || n.ID.Validate() != nil || credential.NamespaceID != n.ID || credential.PrincipalID != r.PrincipalID {
			return empty, d.NewError(d.ErrorCodeInvalidArgument, "namespace and initial credential required")
		}
		for _, current := range state.Namespaces {
			if current.Name == n.Name {
				return empty, d.NewError(d.ErrorCodeAlreadyExists, "namespace name exists")
			}
		}
		initialGrant := ports.NamespaceGrant{NamespaceID: n.ID, PrincipalID: credential.PrincipalID, Permissions: append([]ports.Permission(nil), grant.Permissions...)}
		if err := validateMemoryCredentialBundle(state, s.outcomes, *credential, r.Command.CredentialPolicy, r.Command.CredentialMetadata, initialGrant, r.Now, r.PrincipalID, r.Actor); err != nil {
			return empty, err
		}
		if err := insertMemoryCredential(state, *credential, *r.Command.CredentialPolicy, *r.Command.CredentialMetadata); err != nil {
			return empty, err
		}
		state.Namespaces[n.ID] = *n
		state.Versions[n.ID] = 1
		state.Grants[securityKey(n.ID, initialGrant.PrincipalID)] = initialGrant
		result.CreatedNamespace = n
		result.Credential = credential
		result.CredentialMetadata = r.Command.CredentialMetadata
		target = n.ID.String()
	case "set_grant":
		g := r.Command.Grant
		if g == nil || g.NamespaceID != r.Command.NamespaceID || strings.TrimSpace(g.PrincipalID) == "" {
			return empty, d.NewError(d.ErrorCodeInvalidArgument, "grant required")
		}
		if err := ports.ValidatePermissionValues(g.Permissions); err != nil {
			return empty, err
		}
		grant := deepCopy(*g)
		grant.PermissionCompatibilityVersion = 0
		state.Grants[securityKey(g.NamespaceID, g.PrincipalID)] = grant
		target = g.PrincipalID
	case "issue_credential":
		credential := r.Command.Credential
		if credential == nil || credential.NamespaceID != r.Command.NamespaceID {
			return empty, d.NewError(d.ErrorCodeInvalidArgument, "credential required")
		}
		targetGrant, found := state.Grants[securityKey(credential.NamespaceID, credential.PrincipalID)]
		if !found {
			return empty, d.NewError(d.ErrorCodeForbidden, "target principal has no namespace grant")
		}
		if err := validateMemoryCredentialBundle(state, s.outcomes, *credential, r.Command.CredentialPolicy, r.Command.CredentialMetadata, targetGrant, r.Now, r.PrincipalID, r.Actor); err != nil {
			return empty, err
		}
		if err := insertMemoryCredential(state, *credential, *r.Command.CredentialPolicy, *r.Command.CredentialMetadata); err != nil {
			return empty, err
		}
		result.Credential = credential
		result.CredentialMetadata = r.Command.CredentialMetadata
		target = credential.ID.String()
	case "update_credential":
		credential, metadata, err := updateMemoryCredential(state, s.outcomes, r)
		if err != nil {
			return empty, err
		}
		result.Credential, result.CredentialMetadata = &credential, &metadata
		target = credential.ID.String()
	case "rotate_credential":
		credential, metadata, err := rotateMemoryCredential(state, s.outcomes, r)
		if err != nil {
			return empty, err
		}
		result.Credential, result.CredentialMetadata = &credential, &metadata
		target = r.Command.CredentialID.String() + "->" + credential.ID.String()
	case "revoke_credential":
		if r.Command.CredentialID.Validate() != nil {
			return empty, d.NewError(d.ErrorCodeInvalidArgument, "credential ID required")
		}
		impact, impactErr := memoryCredentialRevocationImpact(state, s.signedContracts, s.signedReview.Contracts, r.Command.NamespaceID, r.Command.CredentialID, r.Now)
		if impactErr != nil {
			return empty, impactErr
		}
		result.RevocationImpact = &impact
		found := false
		for digest, credential := range state.Credentials {
			if credential.NamespaceID == r.Command.NamespaceID && credential.ID == r.Command.CredentialID {
				credential.Revoked = true
				state.Credentials[digest] = credential
				found = true
				break
			}
		}
		if !found {
			return empty, d.NewError(d.ErrorCodeNotFound, "credential not found")
		}
		target = r.Command.CredentialID.String()
	default:
		return empty, d.NewError(d.ErrorCodeInvalidArgument, "unknown security operation")
	}
	adminNamespace := r.Command.NamespaceID
	if r.Command.Operation == "create_namespace" {
		if result.CreatedNamespace == nil {
			return empty, d.NewError(d.ErrorCodeInvalidTransition, "created Namespace missing from result")
		}
		adminNamespace = result.CreatedNamespace.ID
	}
	if err = ensureRecoverableAdminMemory(state, adminNamespace, r.Now); err != nil {
		return empty, err
	}

	result.NamespaceVersion = version + 1
	state.Versions[r.Command.NamespaceID] = result.NamespaceVersion
	afterState, err := memorySecurityAdminAfter(state, r.Command, result, r.Now)
	if err != nil {
		return empty, err
	}
	beforeJSON, err := json.Marshal(beforeState)
	if err != nil {
		return empty, err
	}
	afterJSON, err := json.Marshal(afterState)
	if err != nil {
		return empty, err
	}
	state.Audit[r.Command.NamespaceID] = append(state.Audit[r.Command.NamespaceID], ports.SecurityAudit{NamespaceVersion: result.NamespaceVersion, PrincipalID: r.PrincipalID, Actor: r.Actor, AuthorCredentialID: r.CallerCredentialID, RootCredentialID: rootCredentialID, Operation: r.Command.Operation, Target: target, Reason: r.Command.Reason, Before: beforeJSON, After: afterJSON, RecordedAt: r.Now.UTC()})
	payload, err := json.Marshal(result)
	if err != nil {
		return empty, err
	}
	state.Receipts[receiptKey] = memoryReceipt{Fingerprint: r.Fingerprint, Payload: append(json.RawMessage(nil), payload...)}
	if err = ctx.Err(); err != nil {
		return empty, err
	}
	s.security = state
	return result, nil
}

func validateMemoryCredentialBundle(state memorySecurityState, outcomes map[string]d.Outcome, credential ports.Credential, policy *d.CredentialPolicy, metadata *ports.CredentialMetadata, grant ports.NamespaceGrant, now time.Time, issuerPrincipal string, issuerActor d.ActorRef) error {
	if policy == nil || metadata == nil {
		return d.NewError(d.ErrorCodeInvalidArgument, "credential, explicit policy, and metadata are required")
	}
	if err := policy.Validate(); err != nil {
		return err
	}
	if credential.ID.Validate() != nil || credential.NamespaceID != grant.NamespaceID || credential.PrincipalID != grant.PrincipalID || credential.Actor.Validate() != nil || credential.Digest == "" || credential.Revoked || credential.ParentDigest != "" || !credential.ExpiresAt.After(now) || credential.ExpiresAt.After(now.Add(366*24*time.Hour)) {
		return d.NewError(d.ErrorCodeInvalidArgument, "credential must be an active root credential with a bounded expiration")
	}
	if policy.NamespaceID != credential.NamespaceID || policy.CredentialID != credential.ID || policy.PrincipalID != credential.PrincipalID || policy.PermissionCompatibilityVersion != 0 {
		return d.NewError(d.ErrorCodeInvalidArgument, "credential policy scope does not match its grant")
	}
	if metadata.NamespaceID != credential.NamespaceID || metadata.CredentialID != credential.ID || metadata.PolicyRevision != policy.Version || metadata.MetadataRevision != 1 || metadata.CreatedByPrincipalID != issuerPrincipal || metadata.CreatedByActor != issuerActor || !metadata.CreatedAt.Equal(now) || !metadata.UpdatedAt.Equal(now) || metadata.CreatedByActor.Validate() != nil {
		return d.NewError(d.ErrorCodeInvalidArgument, "credential metadata does not match issuance")
	}
	if utf8.RuneCountInString(metadata.Name) < 1 || utf8.RuneCountInString(metadata.Name) > 100 || utf8.RuneCountInString(metadata.Description) > 1000 {
		return d.NewError(d.ErrorCodeInvalidArgument, "credential metadata is outside the supported length")
	}
	granted := make(map[ports.Permission]struct{}, len(grant.Permissions))
	for _, permission := range grant.Permissions {
		granted[permission] = struct{}{}
	}
	for _, operation := range policy.PermittedOperations {
		if _, ok := granted[ports.Permission(operation)]; !ok {
			return d.NewError(d.ErrorCodeForbidden, "credential policy exceeds the target Principal grant")
		}
	}
	for _, outcome := range policy.AllowedOutcomeIDs {
		value, ok := outcomes[outcomeKey(credential.NamespaceID, outcome)]
		if !ok || value.NamespaceID != credential.NamespaceID {
			return d.NewError(d.ErrorCodeInvalidArgument, "credential Outcome scope must reference Outcomes in the same Namespace")
		}
	}
	for _, keyID := range policy.AllowedSigningKeyIDs {
		key, ok := state.Keys[securityKey(credential.NamespaceID, keyID.String())]
		if !ok || key.PrincipalID != credential.PrincipalID || key.Status != "active" {
			return d.NewError(d.ErrorCodeInvalidArgument, "credential signing-key scope must reference active keys of the target Principal")
		}
	}
	if _, exists := state.Credentials[credential.Digest]; exists {
		return d.NewError(d.ErrorCodeAlreadyExists, "credential digest exists")
	}
	for _, current := range state.Credentials {
		if current.ID == credential.ID {
			return d.NewError(d.ErrorCodeAlreadyExists, "credential ID exists")
		}
	}
	return nil
}

func insertMemoryCredential(state memorySecurityState, credential ports.Credential, policy d.CredentialPolicy, metadata ports.CredentialMetadata) error {
	state.Credentials[credential.Digest] = credential
	state.Policies[securityKey(credential.NamespaceID, credential.ID.String())] = deepCopy(policy)
	state.Metadata[securityKey(credential.NamespaceID, credential.ID.String())] = deepCopy(metadata)
	return nil
}

func updateMemoryCredential(state memorySecurityState, outcomes map[string]d.Outcome, r ports.SecurityAdminRequest) (ports.Credential, ports.CredentialMetadata, error) {
	var emptyCredential ports.Credential
	var emptyMetadata ports.CredentialMetadata
	cmd, now := r.Command, r.Now.UTC()
	var credential ports.Credential
	var digest string
	for value, current := range state.Credentials {
		if current.NamespaceID == cmd.NamespaceID && current.ID == cmd.CredentialID && current.ParentDigest == "" {
			credential, digest = current, value
			break
		}
	}
	if digest == "" {
		return emptyCredential, emptyMetadata, d.NewError(d.ErrorCodeNotFound, "credential not found")
	}
	key := securityKey(cmd.NamespaceID, cmd.CredentialID.String())
	metadata, found := state.Metadata[key]
	if !found || metadata.MetadataRevision == 0 {
		return emptyCredential, emptyMetadata, d.NewError(d.ErrorCodeInvalidTransition, "legacy credential metadata is not editable")
	}
	policy, hasPolicy := state.Policies[key]
	policyVersion := d.Version(0)
	if hasPolicy {
		policyVersion = policy.Version
	}
	if metadata.MetadataRevision != cmd.ExpectedMetadataRevision || policyVersion != cmd.ExpectedPolicyRevision {
		return emptyCredential, emptyMetadata, d.NewError(d.ErrorCodeVersionConflict, "credential revision changed")
	}
	if cmd.CredentialExpiresAt == nil || !cmd.CredentialExpiresAt.After(now) || cmd.CredentialExpiresAt.After(now.Add(366*24*time.Hour)) {
		return emptyCredential, emptyMetadata, d.NewError(d.ErrorCodeInvalidArgument, "credential expiration must be within one year")
	}
	grant, ok := state.Grants[securityKey(cmd.NamespaceID, credential.PrincipalID)]
	if !ok {
		return emptyCredential, emptyMetadata, d.NewError(d.ErrorCodeForbidden, "credential Principal has no active Namespace grant")
	}
	newPolicyVersion := policyVersion
	if cmd.CredentialPolicy != nil {
		candidate := *cmd.CredentialPolicy
		compatibility := uint32(0)
		if hasPolicy {
			compatibility = policy.PermissionCompatibilityVersion
		}
		if candidate.Version != policyVersion+1 || candidate.NamespaceID != cmd.NamespaceID || candidate.CredentialID != cmd.CredentialID || candidate.PrincipalID != credential.PrincipalID || candidate.PermissionCompatibilityVersion != compatibility {
			return emptyCredential, emptyMetadata, d.NewError(d.ErrorCodeVersionConflict, "credential policy revision or scope changed")
		}
		if err := validateMemoryPolicyWithinGrant(state, outcomes, candidate, grant); err != nil {
			return emptyCredential, emptyMetadata, err
		}
		state.Policies[key] = deepCopy(candidate)
		newPolicyVersion = candidate.Version
	}
	metadata.Name, metadata.Description = cmd.CredentialName, cmd.CredentialDescription
	metadata.UpdatedAt, metadata.MetadataRevision, metadata.PolicyRevision = now, metadata.MetadataRevision+1, newPolicyVersion
	if cmd.CredentialPolicy != nil {
		metadata.RoleTemplateID, metadata.RoleTemplateVersion = "", 0
	}
	state.Metadata[key] = deepCopy(metadata)
	credential.ExpiresAt = *cmd.CredentialExpiresAt
	state.Credentials[digest] = credential
	credential.Digest, credential.ParentDigest = "", ""
	return credential, metadata, nil
}

func rotateMemoryCredential(state memorySecurityState, outcomes map[string]d.Outcome, r ports.SecurityAdminRequest) (ports.Credential, ports.CredentialMetadata, error) {
	var emptyCredential ports.Credential
	var emptyMetadata ports.CredentialMetadata
	cmd, now := r.Command, r.Now.UTC()
	if cmd.Credential == nil || cmd.CredentialPolicy == nil || cmd.CredentialMetadata == nil || cmd.OverlapUntil == nil || cmd.CredentialExpiresAt == nil {
		return emptyCredential, emptyMetadata, d.NewError(d.ErrorCodeInvalidArgument, "rotation requires explicit overlap")
	}
	var previous ports.Credential
	var previousDigest string
	for digest, current := range state.Credentials {
		if current.NamespaceID == cmd.NamespaceID && current.ID == cmd.CredentialID && current.ParentDigest == "" {
			previous, previousDigest = current, digest
			break
		}
	}
	if previousDigest == "" {
		return emptyCredential, emptyMetadata, d.NewError(d.ErrorCodeNotFound, "credential not found")
	}
	oldKey := securityKey(cmd.NamespaceID, cmd.CredentialID.String())
	oldMetadata, metadataExists := state.Metadata[oldKey]
	oldPolicy, policyExists := state.Policies[oldKey]
	if !metadataExists || !policyExists || oldMetadata.MetadataRevision == 0 {
		return emptyCredential, emptyMetadata, d.NewError(d.ErrorCodeInvalidTransition, "legacy credential must be managed before rotation")
	}
	if credentialStatus(previous, now) != "active" || oldMetadata.MetadataRevision != cmd.ExpectedMetadataRevision || oldPolicy.Version != cmd.ExpectedPolicyRevision {
		return emptyCredential, emptyMetadata, d.NewError(d.ErrorCodeVersionConflict, "credential changed or is not active")
	}
	overlap := cmd.OverlapUntil.UTC()
	if !overlap.After(now) || overlap.After(*cmd.CredentialExpiresAt) || overlap.After(previous.ExpiresAt) {
		return emptyCredential, emptyMetadata, d.NewError(d.ErrorCodeInvalidArgument, "rotation overlap exceeds credential expiration")
	}
	replacement := *cmd.Credential
	if replacement.ID == previous.ID || replacement.NamespaceID != previous.NamespaceID || replacement.PrincipalID != previous.PrincipalID || replacement.Actor != previous.Actor || replacement.ParentDigest != "" || replacement.Revoked || replacement.Digest == "" || !replacement.ExpiresAt.After(overlap) {
		return emptyCredential, emptyMetadata, d.NewError(d.ErrorCodeInvalidArgument, "replacement must preserve the old holder and remain active after overlap")
	}
	grant, ok := state.Grants[securityKey(cmd.NamespaceID, replacement.PrincipalID)]
	if !ok {
		return emptyCredential, emptyMetadata, d.NewError(d.ErrorCodeForbidden, "credential Principal has no active Namespace grant")
	}
	policy := *cmd.CredentialPolicy
	if policy.Version != 1 || policy.NamespaceID != replacement.NamespaceID || policy.CredentialID != replacement.ID || policy.PrincipalID != replacement.PrincipalID || policy.PermissionCompatibilityVersion != 0 {
		return emptyCredential, emptyMetadata, d.NewError(d.ErrorCodeInvalidArgument, "replacement policy scope is invalid")
	}
	if err := validateMemoryRotationPolicy(policy, oldPolicy); err != nil {
		return emptyCredential, emptyMetadata, err
	}
	if err := validateMemoryPolicyWithinGrant(state, outcomes, policy, grant); err != nil {
		return emptyCredential, emptyMetadata, err
	}
	metadata := *cmd.CredentialMetadata
	if metadata.ReplacesCredentialID != previous.ID || metadata.RotationOverlapUntil == nil || !metadata.RotationOverlapUntil.Equal(overlap) || metadata.CreatedByPrincipalID != r.PrincipalID || metadata.CreatedByActor != r.Actor {
		return emptyCredential, emptyMetadata, d.NewError(d.ErrorCodeInvalidArgument, "replacement provenance is invalid")
	}
	oldMetadata.ReplacedByCredentialID, oldMetadata.RotationOverlapUntil = replacement.ID, &overlap
	oldMetadata.UpdatedAt, oldMetadata.MetadataRevision = now, oldMetadata.MetadataRevision+1
	state.Metadata[oldKey] = oldMetadata
	previous.ExpiresAt = overlap
	state.Credentials[previousDigest] = previous
	if err := validateMemoryCredentialBundle(state, outcomes, replacement, &policy, &metadata, grant, now, r.PrincipalID, r.Actor); err != nil {
		return emptyCredential, emptyMetadata, err
	}
	if err := insertMemoryCredential(state, replacement, policy, metadata); err != nil {
		return emptyCredential, emptyMetadata, err
	}
	replacement.Digest, replacement.ParentDigest = "", ""
	return replacement, metadata, nil
}

func validateMemoryPolicyWithinGrant(state memorySecurityState, outcomes map[string]d.Outcome, policy d.CredentialPolicy, grant ports.NamespaceGrant) error {
	if err := policy.Validate(); err != nil {
		return err
	}
	if policy.NamespaceID != grant.NamespaceID || policy.PrincipalID != grant.PrincipalID {
		return d.NewError(d.ErrorCodeInvalidScope, "credential policy does not match target grant")
	}
	allowed := make(map[ports.Permission]struct{}, len(grant.Permissions))
	for _, permission := range grant.Permissions {
		allowed[permission] = struct{}{}
	}
	for _, operation := range policy.PermittedOperations {
		if _, ok := allowed[ports.Permission(operation)]; !ok {
			return d.NewError(d.ErrorCodeForbidden, "credential policy exceeds the target Principal grant")
		}
	}
	for _, id := range policy.AllowedOutcomeIDs {
		if outcome, ok := outcomes[outcomeKey(policy.NamespaceID, id)]; !ok || outcome.NamespaceID != policy.NamespaceID {
			return d.NewError(d.ErrorCodeInvalidArgument, "credential Outcome scope must reference Outcomes in the same Namespace")
		}
	}
	for _, id := range policy.AllowedSigningKeyIDs {
		key, ok := state.Keys[securityKey(policy.NamespaceID, id.String())]
		if !ok || key.PrincipalID != policy.PrincipalID || key.Status != "active" {
			return d.NewError(d.ErrorCodeInvalidArgument, "credential signing-key scope must reference active keys of the target Principal")
		}
	}
	return nil
}

func validateMemoryRotationPolicy(next, previous d.CredentialPolicy) error {
	old := map[string]struct{}{}
	for _, operation := range previous.PermittedOperations {
		old[operation] = struct{}{}
	}
	for _, operation := range next.PermittedOperations {
		if _, ok := old[operation]; !ok {
			return d.NewError(d.ErrorCodeForbidden, "rotation cannot broaden credential permissions")
		}
	}
	if next.AcceptanceFloor != previous.AcceptanceFloor || next.MaxActiveWorkContracts > previous.MaxActiveWorkContracts || next.MaxActiveReviewContracts > previous.MaxActiveReviewContracts {
		return d.NewError(d.ErrorCodeForbidden, "rotation cannot broaden credential policy")
	}
	if len(previous.AllowedOutcomeIDs) > 0 {
		oldIDs := map[d.ID]struct{}{}
		for _, id := range previous.AllowedOutcomeIDs {
			oldIDs[id] = struct{}{}
		}
		if len(next.AllowedOutcomeIDs) == 0 {
			return d.NewError(d.ErrorCodeForbidden, "rotation cannot remove an Outcome restriction")
		}
		for _, id := range next.AllowedOutcomeIDs {
			if _, ok := oldIDs[id]; !ok {
				return d.NewError(d.ErrorCodeForbidden, "rotation cannot broaden Outcome scope")
			}
		}
	}
	oldKeys := map[d.ID]struct{}{}
	for _, id := range previous.AllowedSigningKeyIDs {
		oldKeys[id] = struct{}{}
	}
	for _, id := range next.AllowedSigningKeyIDs {
		if _, ok := oldKeys[id]; !ok {
			return d.NewError(d.ErrorCodeForbidden, "rotation cannot broaden signing-key scope")
		}
	}
	return nil
}

func (s *Store) ReadSecurityAdmin(ctx context.Context, namespace d.ID) (ports.SecurityAdminSnapshot, error) {
	var result ports.SecurityAdminSnapshot
	if err := ctx.Err(); err != nil {
		return result, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return result, err
	}
	return s.readSecurityAdminLocked(namespace)
}

func (s *Store) ReadSecurityAdminAuthorized(ctx context.Context, access ports.SecurityAdminReadAccess) (ports.SecurityAdminSnapshot, error) {
	var empty ports.SecurityAdminSnapshot
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := authorizeMemorySecurityAdminRead(s.security, access); err != nil {
		return empty, err
	}
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	return s.readSecurityAdminLocked(access.NamespaceID)
}

func (s *Store) readSecurityAdminLocked(namespace d.ID) (ports.SecurityAdminSnapshot, error) {
	var result ports.SecurityAdminSnapshot
	n, ok := s.security.Namespaces[namespace]
	if !ok {
		return result, missingSigning()
	}
	result.Namespace, result.NamespaceVersion = n, s.security.Versions[namespace]
	result.Grants, result.Credentials, result.CredentialMetadata, result.Audit = []ports.NamespaceGrant{}, []ports.Credential{}, []ports.CredentialMetadata{}, []ports.SecurityAudit{}
	for _, grant := range s.security.Grants {
		if grant.NamespaceID == namespace {
			result.Grants = append(result.Grants, deepCopy(grant))
		}
	}
	sort.Slice(result.Grants, func(i, j int) bool { return result.Grants[i].PrincipalID < result.Grants[j].PrincipalID })
	for _, credential := range s.security.Credentials {
		if credential.NamespaceID != namespace || credential.ParentDigest != "" {
			continue
		}
		credential.Digest, credential.ParentDigest = "", ""
		result.Credentials = append(result.Credentials, credential)
		if metadata, exists := s.security.Metadata[securityKey(namespace, credential.ID.String())]; exists {
			result.CredentialMetadata = append(result.CredentialMetadata, deepCopy(metadata))
		}
	}
	sort.Slice(result.Credentials, func(i, j int) bool { return result.Credentials[i].ID < result.Credentials[j].ID })
	sort.Slice(result.CredentialMetadata, func(i, j int) bool {
		return result.CredentialMetadata[i].CredentialID < result.CredentialMetadata[j].CredentialID
	})
	result.Audit = append(result.Audit, s.security.Audit[namespace]...)
	sort.Slice(result.Audit, func(i, j int) bool { return result.Audit[i].NamespaceVersion > result.Audit[j].NamespaceVersion })
	if len(result.Grants) > 100 {
		result.Grants = result.Grants[:100]
		result.Truncated = true
	}
	if len(result.Credentials) > 100 {
		result.Credentials = result.Credentials[:100]
		result.Truncated = true
	}
	if len(result.CredentialMetadata) > 100 {
		result.CredentialMetadata = result.CredentialMetadata[:100]
		result.Truncated = true
	}
	if len(result.Audit) > 100 {
		result.Audit = result.Audit[:100]
		result.Truncated = true
	}
	return result, nil
}

func authorizeMemorySecurityAdminRead(state memorySecurityState, access ports.SecurityAdminReadAccess) (d.ID, error) {
	denied := d.NewError(d.ErrorCodeForbidden, "credential or namespace permission inactive")
	if access.NamespaceID.Validate() != nil || access.CredentialID.Validate() != nil || access.PrincipalID == "" || access.CredentialDigest == "" || access.Actor.Validate() != nil || access.Now.IsZero() || len(access.Permissions) == 0 {
		return "", denied
	}
	if _, ok := state.Namespaces[access.NamespaceID]; !ok {
		return "", denied
	}
	grant, ok := state.Grants[securityKey(access.NamespaceID, access.PrincipalID)]
	if !ok {
		return "", denied
	}
	for _, required := range access.Permissions {
		allowed := false
		for _, granted := range grant.Permissions {
			allowed = allowed || required == granted
		}
		if !allowed {
			return "", denied
		}
	}
	caller, ok := state.Credentials[access.CredentialDigest]
	if !ok || caller.ID != access.CredentialID || caller.PrincipalID != access.PrincipalID || caller.NamespaceID != access.NamespaceID || caller.Actor != access.Actor || caller.Revoked || !access.Now.Before(caller.ExpiresAt) {
		return "", denied
	}
	root := caller
	if caller.ParentDigest != "" {
		root, ok = state.Credentials[caller.ParentDigest]
		if !ok || root.ParentDigest != "" || root.PrincipalID != caller.PrincipalID || root.NamespaceID != caller.NamespaceID || root.Actor != caller.Actor || root.Revoked || !access.Now.Before(root.ExpiresAt) {
			return "", denied
		}
	}
	if policy, exists := state.Policies[securityKey(root.NamespaceID, root.ID.String())]; exists {
		for _, required := range access.Permissions {
			if !policy.PermitsWithCompatibility(string(required), "", "") {
				return "", denied
			}
		}
	}
	return root.ID, nil
}

func memorySecurityAdminBefore(state memorySecurityState, command ports.SecurityAdminCommand, now time.Time) (map[string]any, error) {
	before := map[string]any{"exists": false}
	switch command.Operation {
	case "set_grant":
		if command.Grant == nil {
			return before, nil
		}
		grant, found := state.Grants[securityKey(command.NamespaceID, command.Grant.PrincipalID)]
		if !found {
			return ports.SecurityAuditGrantState(false, nil, 0), nil
		}
		return ports.SecurityAuditGrantState(true, grant.Permissions, grant.PermissionCompatibilityVersion), nil
	case "issue_credential", "create_namespace":
		return before, nil
	case "update_credential", "rotate_credential", "revoke_credential":
		for _, credential := range state.Credentials {
			if credential.NamespaceID == command.NamespaceID && credential.ID == command.CredentialID && credential.ParentDigest == "" {
				return ports.SecurityAuditCredentialState(memoryCredentialAuditRecord(state, credential, now)), nil
			}
		}
		return nil, d.NewError(d.ErrorCodeNotFound, "credential not found")
	default:
		return before, nil
	}
}

func memoryCredentialAuditRecord(state memorySecurityState, credential ports.Credential, now time.Time) ports.CredentialAdministrativeRecord {
	credential.Digest, credential.ParentDigest = "", ""
	key := securityKey(credential.NamespaceID, credential.ID.String())
	metadata := state.Metadata[key]
	if metadata.CredentialID.IsZero() {
		metadata = ports.CredentialMetadata{NamespaceID: credential.NamespaceID, CredentialID: credential.ID, Name: "Legacy credential"}
	}
	policy, hasPolicy := state.Policies[key]
	grant := state.Grants[securityKey(credential.NamespaceID, credential.PrincipalID)]
	record := ports.CredentialAdministrativeRecord{Credential: credential, Metadata: deepCopy(metadata), Status: credentialStatus(credential, now), GrantPermissionCompatibilityVersion: grant.PermissionCompatibilityVersion, PolicyExplicit: hasPolicy}
	if hasPolicy {
		copyPolicy := deepCopy(policy)
		record.Policy = &copyPolicy
		record.EffectivePolicy = copyPolicy
		allowed := make(map[ports.Permission]struct{}, len(grant.Permissions))
		for _, permission := range grant.Permissions {
			allowed[permission] = struct{}{}
		}
		record.EffectivePolicy.PermittedOperations = make([]string, 0, len(policy.PermittedOperations))
		for _, operation := range policy.PermittedOperations {
			if _, ok := allowed[ports.Permission(operation)]; ok {
				record.EffectivePolicy.PermittedOperations = append(record.EffectivePolicy.PermittedOperations, operation)
			}
		}
		sort.Strings(record.EffectivePolicy.PermittedOperations)
	} else {
		record.EffectivePolicy = d.CredentialPolicy{NamespaceID: credential.NamespaceID, CredentialID: credential.ID, PrincipalID: credential.PrincipalID, AcceptanceFloor: d.AcceptanceDirect, MaxActiveWorkContracts: 3, MaxActiveReviewContracts: 1, PermissionCompatibilityVersion: grant.PermissionCompatibilityVersion}
		for _, permission := range grant.Permissions {
			record.EffectivePolicy.PermittedOperations = append(record.EffectivePolicy.PermittedOperations, string(permission))
		}
	}
	sort.Strings(record.EffectivePolicy.PermittedOperations)
	return record
}

func memorySecurityAdminAfter(state memorySecurityState, command ports.SecurityAdminCommand, result ports.SecurityAdminResult, now time.Time) (map[string]any, error) {
	switch command.Operation {
	case "set_grant":
		if command.Grant == nil {
			return nil, d.NewError(d.ErrorCodeInvalidArgument, "grant required for audit")
		}
		grant, ok := state.Grants[securityKey(command.Grant.NamespaceID, command.Grant.PrincipalID)]
		if !ok {
			return nil, d.NewError(d.ErrorCodeInvalidTransition, "updated grant missing from audit state")
		}
		return ports.SecurityAuditGrantState(true, grant.Permissions, grant.PermissionCompatibilityVersion), nil
	case "issue_credential", "rotate_credential", "update_credential", "revoke_credential":
		credentialID := command.CredentialID
		if command.Operation != "revoke_credential" {
			if result.Credential == nil {
				return nil, d.NewError(d.ErrorCodeInvalidTransition, "credential result missing for audit")
			}
			credentialID = result.Credential.ID
		}
		for _, credential := range state.Credentials {
			if credential.NamespaceID == command.NamespaceID && credential.ID == credentialID && credential.ParentDigest == "" {
				projected := ports.SecurityAuditCredentialState(memoryCredentialAuditRecord(state, credential, now))
				if result.RevocationImpact != nil {
					projected["revocation_impact"] = result.RevocationImpact
				}
				return projected, nil
			}
		}
		return nil, d.NewError(d.ErrorCodeNotFound, "credential not found for audit")
	case "create_namespace":
		if result.CreatedNamespace == nil || result.Credential == nil {
			return nil, d.NewError(d.ErrorCodeInvalidTransition, "namespace result missing for audit")
		}
		grant, ok := state.Grants[securityKey(result.CreatedNamespace.ID, result.Credential.PrincipalID)]
		if !ok {
			return nil, d.NewError(d.ErrorCodeInvalidTransition, "initial Namespace grant missing from audit state")
		}
		var credential *ports.Credential
		for _, candidate := range state.Credentials {
			if candidate.NamespaceID == result.CreatedNamespace.ID && candidate.ID == result.Credential.ID && candidate.ParentDigest == "" {
				copy := candidate
				credential = &copy
				break
			}
		}
		if credential == nil {
			return nil, d.NewError(d.ErrorCodeInvalidTransition, "initial Namespace credential missing from audit state")
		}
		return map[string]any{"exists": true, "namespace_id": result.CreatedNamespace.ID, "name": result.CreatedNamespace.Name, "initial_grant": ports.SecurityAuditGrantState(true, grant.Permissions, grant.PermissionCompatibilityVersion), "initial_credential": ports.SecurityAuditCredentialState(memoryCredentialAuditRecord(state, *credential, now))}, nil
	default:
		return map[string]any{"exists": true}, nil
	}
}

func ensureRecoverableAdminMemory(state memorySecurityState, namespace d.ID, now time.Time) error {
	candidates := []ports.Permission{ports.PermissionPermissionsManage, ports.PermissionCredentialsIssue, ports.PermissionCredentialsUpdate, ports.PermissionCredentialsRotate}
	for _, credential := range state.Credentials {
		if credential.NamespaceID != namespace || credential.ParentDigest != "" || credential.Revoked || !now.Before(credential.ExpiresAt) {
			continue
		}
		grant, found := state.Grants[securityKey(namespace, credential.PrincipalID)]
		if !found {
			continue
		}
		policy, explicit := state.Policies[securityKey(namespace, credential.ID.String())]
		for _, candidate := range candidates {
			grantHas := false
			for _, permission := range grant.Permissions {
				grantHas = grantHas || permission == candidate
			}
			policyHas := !explicit || policy.PermitsWithCompatibility(string(candidate), "", "")
			if grantHas && policyHas {
				return nil
			}
		}
	}
	return d.NewError(d.ErrorCodeInvalidTransition, "operation would remove the last recoverable administrator credential")
}

func memoryCredentialRevocationImpact(state memorySecurityState, work map[string]d.WorkContract, reviews map[string]d.ReviewContract, namespace, id d.ID, now time.Time) (ports.CredentialRevocationImpact, error) {
	var impact ports.CredentialRevocationImpact
	var target ports.Credential
	found := false
	for _, credential := range state.Credentials {
		if credential.NamespaceID == namespace && credential.ID == id && credential.ParentDigest == "" {
			target, found = credential, true
			break
		}
	}
	if !found {
		return impact, d.NewError(d.ErrorCodeNotFound, "credential not found")
	}
	impact = ports.CredentialRevocationImpact{NamespaceID: namespace, CredentialID: id, PrincipalID: target.PrincipalID, LegacyLeasesMayBeUnattributed: true, LegacyContractsMayBeUnattributed: true, AuthorityRemovedOnCommit: true}
	for _, session := range state.Credentials {
		if session.ParentDigest == target.Digest && !session.Revoked && now.Before(session.ExpiresAt) {
			impact.ActiveSessions++
		}
	}
	for _, contract := range work {
		if contract.Scope.NamespaceID == namespace && contract.SignedBinding != nil && contract.SignedBinding.CredentialID == id && contract.Status == d.ContractActive && now.Before(contract.ExpiresAt) {
			impact.ActiveSignedWorkContracts++
		}
	}
	for _, contract := range reviews {
		if contract.Scope.NamespaceID == namespace && contract.Binding.CredentialID == id && contract.Status == d.ContractActive && now.Before(contract.ExpiresAt) {
			impact.ActiveSignedReviewContracts++
		}
	}
	impact.ExternalProcessesStopped = false
	return impact, nil
}

func (s *Store) ReadSecurityAdminVersion(ctx context.Context, namespace d.ID) (d.Version, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	version, ok := s.security.Versions[namespace]
	if _, active := s.security.Namespaces[namespace]; !ok || !active {
		return 0, missingSigning()
	}
	return version, ctx.Err()
}
