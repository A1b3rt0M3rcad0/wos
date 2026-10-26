package application

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

const credentialPageDefault = 25

type CredentialUpdateIntent struct {
	NamespaceID              domain.ID
	CredentialID             domain.ID
	ExpectedNamespaceVersion domain.Version
	ExpectedMetadataRevision domain.Version
	ExpectedPolicyRevision   domain.Version
	Name                     string
	Description              string
	ExpiresAt                *time.Time
	Policy                   *domain.CredentialPolicy
	Reason                   string
}

type CredentialRotationIntent struct {
	NamespaceID              domain.ID
	CredentialID             domain.ID
	ExpectedNamespaceVersion domain.Version
	ExpectedMetadataRevision domain.Version
	ExpectedPolicyRevision   domain.Version
	ExpiresAt                time.Time
	OverlapUntil             time.Time
	Name                     string
	Description              string
	Reason                   string
}

func (s SecurityService) ListCredentials(ctx context.Context, ns domain.ID, query ports.CredentialManagementQuery) (ports.CredentialManagementPage, error) {
	reader, ok := s.Store.(ports.AuthorizedCredentialManagementReader)
	if !ok {
		return ports.CredentialManagementPage{}, domain.NewError(domain.ErrorCodeInvalidConfig, "credential management unavailable")
	}
	if query.Limit == 0 {
		query.Limit = credentialPageDefault
	}
	if query.Limit < 1 || query.Limit > 100 {
		return ports.CredentialManagementPage{}, domain.NewError(domain.ErrorCodeInvalidArgument, "credential page limit must be between 1 and 100")
	}
	access, err := s.securityAdminReadAccess(ctx, ns, []ports.Permission{ports.PermissionCredentialsRead})
	if err != nil {
		return ports.CredentialManagementPage{}, err
	}
	query.NamespaceID, query.Now = ns, access.Now
	return reader.ListManagedCredentialsAuthorized(ctx, query, access)
}

func (s SecurityService) GetCredential(ctx context.Context, ns, id domain.ID) (ports.CredentialAdministrativeRecord, error) {
	reader, ok := s.Store.(ports.AuthorizedCredentialManagementReader)
	if !ok {
		return ports.CredentialAdministrativeRecord{}, domain.NewError(domain.ErrorCodeInvalidConfig, "credential management unavailable")
	}
	access, err := s.securityAdminReadAccess(ctx, ns, []ports.Permission{ports.PermissionCredentialsRead})
	if err != nil {
		return ports.CredentialAdministrativeRecord{}, err
	}
	return reader.ReadManagedCredentialAuthorized(ctx, ns, id, access.Now, access)
}

func (s SecurityService) PreviewCredentialRevocation(ctx context.Context, ns, id domain.ID) (ports.CredentialRevocationImpact, error) {
	reader, ok := s.Store.(ports.AuthorizedCredentialManagementReader)
	if !ok {
		return ports.CredentialRevocationImpact{}, domain.NewError(domain.ErrorCodeInvalidConfig, "credential management unavailable")
	}
	access, err := s.securityAdminReadAccess(ctx, ns, []ports.Permission{ports.PermissionCredentialsRevoke})
	if err != nil {
		return ports.CredentialRevocationImpact{}, err
	}
	return reader.PreviewCredentialRevocation(ctx, ns, id, access.Now, access)
}

// UpdateCredential changes descriptive metadata, expiration, and optionally
// the frozen credential policy under both the Namespace and record revisions.
func (s SecurityService) UpdateCredential(ctx context.Context, key string, in CredentialUpdateIntent) (ports.SecurityAdminResult, error) {
	if err := s.Require(ctx, in.NamespaceID, ports.PermissionCredentialsUpdate); err != nil {
		return ports.SecurityAdminResult{}, err
	}
	if err := domain.ValidateIdempotencyKey(key); err != nil {
		return ports.SecurityAdminResult{}, err
	}
	if in.ExpectedNamespaceVersion < 1 || in.ExpectedMetadataRevision < 1 || in.ExpectedPolicyRevision < 0 || in.CredentialID.Validate() != nil {
		return ports.SecurityAdminResult{}, domain.NewError(domain.ErrorCodeInvalidArgument, "credential and expected revisions are required")
	}
	name, description := strings.TrimSpace(in.Name), strings.TrimSpace(in.Description)
	if name == "" || utf8.RuneCountInString(name) > 100 || utf8.RuneCountInString(description) > 1000 {
		return ports.SecurityAdminResult{}, domain.NewError(domain.ErrorCodeInvalidArgument, "credential name is required and description is limited to 1000 characters")
	}
	now := s.Clock.Now().UTC()
	reader, ok := s.Store.(ports.CredentialManagementReader)
	if !ok {
		return ports.SecurityAdminResult{}, domain.NewError(domain.ErrorCodeInvalidConfig, "credential management unavailable")
	}
	current, err := reader.ReadManagedCredential(ctx, in.NamespaceID, in.CredentialID, now)
	if err != nil {
		return ports.SecurityAdminResult{}, err
	}
	if current.Metadata.MetadataRevision != in.ExpectedMetadataRevision || current.EffectivePolicy.Version != in.ExpectedPolicyRevision {
		return ports.SecurityAdminResult{}, domain.NewError(domain.ErrorCodeVersionConflict, "credential revision changed")
	}
	if current.Metadata.MetadataRevision == 0 {
		return ports.SecurityAdminResult{}, domain.NewError(domain.ErrorCodeInvalidTransition, "legacy credentials cannot be edited until replaced")
	}
	expires := current.Credential.ExpiresAt
	if in.ExpiresAt != nil {
		expires = in.ExpiresAt.UTC()
		if !expires.After(now) || expires.After(now.Add(366*24*time.Hour)) {
			return ports.SecurityAdminResult{}, domain.NewError(domain.ErrorCodeInvalidArgument, "credential expiration must be within one year")
		}
	}
	var policy *domain.CredentialPolicy
	if in.Policy != nil {
		grant, grantErr := s.Store.GetGrant(ctx, in.NamespaceID, current.Credential.PrincipalID)
		if grantErr != nil {
			return ports.SecurityAdminResult{}, domain.NewError(domain.ErrorCodeForbidden, "credential Principal has no active Namespace grant")
		}
		allowed := make(map[ports.Permission]struct{}, len(grant.Permissions))
		for _, permission := range grant.Permissions {
			allowed[permission] = struct{}{}
		}
		candidate := *in.Policy
		candidate.NamespaceID, candidate.CredentialID, candidate.PrincipalID = in.NamespaceID, in.CredentialID, current.Credential.PrincipalID
		candidate.Version, candidate.UpdatedAt = in.ExpectedPolicyRevision+1, now
		if current.Policy != nil {
			candidate.PermissionCompatibilityVersion = current.Policy.PermissionCompatibilityVersion
		} else {
			candidate.PermissionCompatibilityVersion = 0
		}
		for _, operation := range candidate.PermittedOperations {
			if _, ok := allowed[ports.Permission(operation)]; !ok {
				return ports.SecurityAdminResult{}, domain.NewError(domain.ErrorCodeForbidden, "credential policy exceeds the target Principal grant")
			}
		}
		if err = candidate.Validate(); err != nil {
			return ports.SecurityAdminResult{}, err
		}
		policy = &candidate
	}
	cmd := ports.SecurityAdminCommand{NamespaceID: in.NamespaceID, ExpectedVersion: in.ExpectedNamespaceVersion, Operation: "update_credential", CredentialID: in.CredentialID, ExpectedMetadataRevision: in.ExpectedMetadataRevision, ExpectedPolicyRevision: in.ExpectedPolicyRevision, CredentialName: name, CredentialDescription: description, CredentialPolicy: policy, CredentialExpiresAt: &expires, Reason: in.Reason}
	result, _, err := s.AdministrativeMutation(ctx, key, cmd)
	return result, err
}

func (s SecurityService) RotateCredential(ctx context.Context, key string, in CredentialRotationIntent) (ports.SecurityAdminResult, string, error) {
	if err := s.Require(ctx, in.NamespaceID, ports.PermissionCredentialsRotate); err != nil {
		return ports.SecurityAdminResult{}, "", err
	}
	if err := domain.ValidateIdempotencyKey(key); err != nil {
		return ports.SecurityAdminResult{}, "", err
	}
	if in.ExpectedNamespaceVersion < 1 || in.ExpectedMetadataRevision < 1 || in.ExpectedPolicyRevision < 0 || in.CredentialID.Validate() != nil {
		return ports.SecurityAdminResult{}, "", domain.NewError(domain.ErrorCodeInvalidArgument, "credential and expected revisions are required")
	}
	name, description := strings.TrimSpace(in.Name), strings.TrimSpace(in.Description)
	if utf8.RuneCountInString(name) > 100 || utf8.RuneCountInString(description) > 1000 {
		return ports.SecurityAdminResult{}, "", domain.NewError(domain.ErrorCodeInvalidArgument, "credential name or description exceeds its limit")
	}
	expires, overlap := in.ExpiresAt.UTC(), in.OverlapUntil.UTC()
	now := s.Clock.Now().UTC()
	if !expires.After(now) || expires.After(now.Add(366*24*time.Hour)) || !overlap.After(now) || overlap.After(expires) {
		return ports.SecurityAdminResult{}, "", domain.NewError(domain.ErrorCodeInvalidArgument, "rotation requires bounded expiration and explicit overlap")
	}
	cmd := ports.SecurityAdminCommand{NamespaceID: in.NamespaceID, ExpectedVersion: in.ExpectedNamespaceVersion, Operation: "rotate_credential", CredentialID: in.CredentialID, ExpectedMetadataRevision: in.ExpectedMetadataRevision, ExpectedPolicyRevision: in.ExpectedPolicyRevision, CredentialName: name, CredentialDescription: description, CredentialExpiresAt: &expires, OverlapUntil: &overlap, Reason: in.Reason}
	return s.AdministrativeMutation(ctx, key, cmd)
}
