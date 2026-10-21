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
	if r.Command.Operation != "issue_credential" || r.Command.NamespaceID.Validate() != nil || r.Command.ExpectedVersion == 0 || strings.TrimSpace(r.PrincipalID) == "" || r.Fingerprint == "" || r.Now.IsZero() {
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
	if r.CredentialDigest != "" {
		caller, found := state.Credentials[r.CredentialDigest]
		if !found || caller.PrincipalID != r.PrincipalID || caller.NamespaceID != r.Command.NamespaceID || caller.Actor != r.Actor || caller.Revoked || !r.Now.Before(caller.ExpiresAt) {
			return empty, false, d.NewError(d.ErrorCodeForbidden, "credential inactive")
		}
		if caller.ParentDigest != "" {
			parent, found := state.Credentials[caller.ParentDigest]
			if !found || parent.ParentDigest != "" || parent.PrincipalID != caller.PrincipalID || parent.NamespaceID != caller.NamespaceID || parent.Actor != caller.Actor || parent.Revoked || !r.Now.Before(parent.ExpiresAt) {
				return empty, false, d.NewError(d.ErrorCodeForbidden, "session parent inactive")
			}
			caller = parent
		}
		if policy, found := state.Policies[securityKey(caller.NamespaceID, caller.ID.String())]; found && !policy.PermitsWithCompatibility(string(required), "", d.ID("")) {
			return empty, false, d.NewError(d.ErrorCodeForbidden, "credential policy restricts security operation")
		}
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
	if r.CredentialDigest != "" {
		caller, found := state.Credentials[r.CredentialDigest]
		if !found || caller.PrincipalID != r.PrincipalID || caller.NamespaceID != r.Command.NamespaceID || caller.Actor != r.Actor || caller.Revoked || !r.Now.Before(caller.ExpiresAt) {
			return empty, d.NewError(d.ErrorCodeForbidden, "credential inactive")
		}
		if caller.ParentDigest != "" {
			parent, found := state.Credentials[caller.ParentDigest]
			if !found || parent.ParentDigest != "" || parent.PrincipalID != caller.PrincipalID || parent.NamespaceID != caller.NamespaceID || parent.Actor != caller.Actor || parent.Revoked || !r.Now.Before(parent.ExpiresAt) {
				return empty, d.NewError(d.ErrorCodeForbidden, "session parent inactive")
			}
			caller = parent
		}
		if policy, found := state.Policies[securityKey(caller.NamespaceID, caller.ID.String())]; found && !policy.PermitsWithCompatibility(string(required), "", d.ID("")) {
			return empty, d.NewError(d.ErrorCodeForbidden, "credential policy restricts security operation")
		}
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
		state.Grants[securityKey(g.NamespaceID, g.PrincipalID)] = deepCopy(*g)
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
	case "revoke_credential":
		if r.Command.CredentialID.Validate() != nil {
			return empty, d.NewError(d.ErrorCodeInvalidArgument, "credential ID required")
		}
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

	result.NamespaceVersion = version + 1
	state.Versions[r.Command.NamespaceID] = result.NamespaceVersion
	state.Audit[r.Command.NamespaceID] = append(state.Audit[r.Command.NamespaceID], ports.SecurityAudit{NamespaceVersion: result.NamespaceVersion, PrincipalID: r.PrincipalID, Actor: r.Actor, Operation: r.Command.Operation, Target: target, RecordedAt: r.Now.UTC()})
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
	if metadata.NamespaceID != credential.NamespaceID || metadata.CredentialID != credential.ID || metadata.PolicyRevision != policy.Version || metadata.CreatedByPrincipalID != issuerPrincipal || metadata.CreatedByActor != issuerActor || !metadata.CreatedAt.Equal(now) || metadata.CreatedByActor.Validate() != nil {
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
