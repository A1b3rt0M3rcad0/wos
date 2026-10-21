package memory

import (
	"context"
	"sort"
	"strings"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

var _ ports.CredentialManagementReader = (*Store)(nil)
var _ ports.CredentialUseObserver = (*Store)(nil)

func (s *Store) ListManagedCredentials(ctx context.Context, q ports.CredentialManagementQuery) (ports.CredentialManagementPage, error) {
	var page ports.CredentialManagementPage
	if err := validateCredentialManagementQuery(q); err != nil {
		return page, err
	}
	if q.NamespaceID.Validate() != nil || q.Now.IsZero() || (!q.Cursor.IsZero() && q.Cursor.Validate() != nil) || (q.Status != "" && q.Status != "active" && q.Status != "expired" && q.Status != "revoked") {
		return page, d.NewError(d.ErrorCodeInvalidArgument, "invalid bounded credential query")
	}
	if err := ctx.Err(); err != nil {
		return page, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return page, err
	}
	return s.listManagedCredentialsLocked(q)
}

func (s *Store) ListManagedCredentialsAuthorized(ctx context.Context, q ports.CredentialManagementQuery, access ports.SecurityAdminReadAccess) (ports.CredentialManagementPage, error) {
	var page ports.CredentialManagementPage
	if err := validateCredentialManagementQuery(q); err != nil {
		return page, err
	}
	if q.NamespaceID.Validate() != nil || q.Now.IsZero() || (!q.Cursor.IsZero() && q.Cursor.Validate() != nil) || (q.Status != "" && q.Status != "active" && q.Status != "expired" && q.Status != "revoked") {
		return page, d.NewError(d.ErrorCodeInvalidArgument, "invalid bounded credential query")
	}
	if err := ctx.Err(); err != nil {
		return page, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if access.NamespaceID != q.NamespaceID || access.Now != q.Now {
		return page, d.NewError(d.ErrorCodeInvalidArgument, "credential query authorization scope mismatch")
	}
	if _, err := authorizeMemorySecurityAdminRead(s.security, access); err != nil {
		return page, err
	}
	return s.listManagedCredentialsLocked(q)
}

func (s *Store) listManagedCredentialsLocked(q ports.CredentialManagementQuery) (ports.CredentialManagementPage, error) {
	var page ports.CredentialManagementPage
	if _, ok := s.security.Namespaces[q.NamespaceID]; !ok {
		return page, missingSigning()
	}
	credentials := make([]ports.Credential, 0)
	for _, credential := range s.security.Credentials {
		if credential.NamespaceID != q.NamespaceID || credential.ParentDigest != "" || credential.ID <= q.Cursor {
			continue
		}
		if q.PrincipalID != "" && credential.PrincipalID != q.PrincipalID {
			continue
		}
		if q.ActorKind != "" && credential.Actor.Kind != q.ActorKind {
			continue
		}
		if q.ActorID != "" && credential.Actor.ID != q.ActorID {
			continue
		}
		status := credentialStatus(credential, q.Now)
		if q.Status != "" && status != q.Status {
			continue
		}
		credentials = append(credentials, credential)
	}
	sort.Slice(credentials, func(i, j int) bool { return credentials[i].ID < credentials[j].ID })
	page.Items = make([]ports.CredentialAdministrativeRecord, 0, min(q.Limit+1, len(credentials)))
	for _, credential := range credentials {
		page.Items = append(page.Items, s.credentialRecordLocked(credential, q.Now))
		if len(page.Items) > q.Limit {
			break
		}
	}
	if len(page.Items) > q.Limit {
		page.NextCursor = page.Items[q.Limit-1].Credential.ID
		page.Items = page.Items[:q.Limit]
	}
	return page, nil
}

func (s *Store) ReadManagedCredential(ctx context.Context, ns, id d.ID, now time.Time) (ports.CredentialAdministrativeRecord, error) {
	var empty ports.CredentialAdministrativeRecord
	if ns.Validate() != nil || id.Validate() != nil || now.IsZero() {
		return empty, d.NewError(d.ErrorCodeInvalidArgument, "invalid credential detail query")
	}
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	return s.readManagedCredentialLocked(ns, id, now)
}

func (s *Store) ReadManagedCredentialAuthorized(ctx context.Context, ns, id d.ID, now time.Time, access ports.SecurityAdminReadAccess) (ports.CredentialAdministrativeRecord, error) {
	var empty ports.CredentialAdministrativeRecord
	if ns.Validate() != nil || id.Validate() != nil || now.IsZero() || access.NamespaceID != ns || access.Now != now {
		return empty, d.NewError(d.ErrorCodeInvalidArgument, "invalid credential detail query")
	}
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := authorizeMemorySecurityAdminRead(s.security, access); err != nil {
		return empty, err
	}
	return s.readManagedCredentialLocked(ns, id, now)
}

func (s *Store) readManagedCredentialLocked(ns, id d.ID, now time.Time) (ports.CredentialAdministrativeRecord, error) {
	var empty ports.CredentialAdministrativeRecord
	if _, ok := s.security.Namespaces[ns]; !ok {
		return empty, missingSigning()
	}
	for _, credential := range s.security.Credentials {
		if credential.NamespaceID == ns && credential.ID == id && credential.ParentDigest == "" {
			return s.credentialRecordLocked(credential, now), nil
		}
	}
	return empty, d.NewError(d.ErrorCodeNotFound, "credential not found")
}

func (s *Store) PreviewCredentialRevocation(ctx context.Context, ns, id d.ID, now time.Time, access ports.SecurityAdminReadAccess) (ports.CredentialRevocationImpact, error) {
	var result ports.CredentialRevocationImpact
	if ns.Validate() != nil || id.Validate() != nil || now.IsZero() || access.NamespaceID != ns || access.Now != now {
		return result, d.NewError(d.ErrorCodeInvalidArgument, "invalid credential revocation preview")
	}
	if err := ctx.Err(); err != nil {
		return result, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, err := authorizeMemorySecurityAdminRead(s.security, access); err != nil {
		return result, err
	}
	var target ports.Credential
	found := false
	for _, c := range s.security.Credentials {
		if c.NamespaceID == ns && c.ID == id && c.ParentDigest == "" {
			target, found = c, true
			break
		}
	}
	if !found {
		return result, d.NewError(d.ErrorCodeNotFound, "credential not found")
	}
	result = ports.CredentialRevocationImpact{NamespaceID: ns, CredentialID: id, PrincipalID: target.PrincipalID, LegacyLeasesMayBeUnattributed: true, LegacyContractsMayBeUnattributed: true, AuthorityRemovedOnCommit: true}
	for _, child := range s.security.Credentials {
		if child.ParentDigest == target.Digest && !child.Revoked && now.Before(child.ExpiresAt) {
			result.ActiveSessions++
		}
	}
	for _, contract := range s.signedContracts {
		if contract.Scope.NamespaceID == ns && contract.SignedBinding != nil && contract.SignedBinding.CredentialID == id && contract.Status == d.ContractActive && now.Before(contract.ExpiresAt) {
			result.ActiveSignedWorkContracts++
		}
	}
	for _, contract := range s.signedReview.Contracts {
		if contract.Scope.NamespaceID == ns && contract.Binding.CredentialID == id && contract.Status == d.ContractActive && now.Before(contract.ExpiresAt) {
			result.ActiveSignedReviewContracts++
		}
	}
	result.ExternalProcessesStopped = false
	return result, nil
}

func (s *Store) credentialRecordLocked(credential ports.Credential, now time.Time) ports.CredentialAdministrativeRecord {
	credential.Digest, credential.ParentDigest = "", ""
	key := securityKey(credential.NamespaceID, credential.ID.String())
	metadata, found := s.security.Metadata[key]
	if !found {
		metadata = ports.CredentialMetadata{NamespaceID: credential.NamespaceID, CredentialID: credential.ID, Name: "Legacy credential"}
	}
	grant := s.security.Grants[securityKey(credential.NamespaceID, credential.PrincipalID)]
	record := ports.CredentialAdministrativeRecord{Credential: credential, Metadata: deepCopy(metadata), Status: credentialStatus(credential, now), GrantPermissionCompatibilityVersion: grant.PermissionCompatibilityVersion}
	policy, explicit := s.security.Policies[key]
	record.PolicyExplicit = explicit
	if explicit {
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
		sort.Strings(record.EffectivePolicy.PermittedOperations)
	}
	return record
}

func credentialStatus(credential ports.Credential, now time.Time) string {
	if credential.Revoked {
		return "revoked"
	}
	if !now.Before(credential.ExpiresAt) {
		return "expired"
	}
	return "active"
}

func validateCredentialManagementQuery(q ports.CredentialManagementQuery) error {
	if q.Limit < 1 || q.Limit > 100 || q.PrincipalID != strings.TrimSpace(q.PrincipalID) || q.ActorID != strings.TrimSpace(q.ActorID) {
		return d.NewError(d.ErrorCodeInvalidArgument, "invalid credential query limit or filter")
	}
	return nil
}

func (s *Store) ObserveCredentialUse(ctx context.Context, ns, id d.ID, now time.Time, interval time.Duration) error {
	if ns.Validate() != nil || id.Validate() != nil || now.IsZero() || interval <= 0 {
		return d.NewError(d.ErrorCodeInvalidArgument, "invalid credential use observation")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	key := securityKey(ns, id.String())
	metadata, ok := s.security.Metadata[key]
	if !ok {
		return nil
	}
	if metadata.LastUsedAt != nil && metadata.LastUsedAt.After(now.Add(-interval)) {
		return nil
	}
	at := now.UTC()
	metadata.LastUsedAt = &at
	s.security.Metadata[key] = metadata
	return nil
}
