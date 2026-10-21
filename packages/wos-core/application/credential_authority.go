package application

import (
	"context"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// leaseCredentialAuthority distinguishes the credential presented with the
// request from the root credential that owns a legacy lease. A session may be
// replaced or renewed while preserving its root's lease ownership.
type leaseCredentialAuthority struct {
	presentedID domain.ID
	holderID    domain.ID
	sessionID   *domain.ID
}

// resolveLeaseCredentialAuthority validates the presented credential and its
// optional parent against the transaction snapshot. The resolved root is used
// only as an ownership key; command authorization remains the responsibility
// of the normal authorization pipeline, which evaluates the request credential.
//
// A nil authority is returned only for the explicitly trusted embedded
// compatibility path when the request has no credential identity and the
// existing lease is not already credential-bound.
func (s *Service) resolveLeaseCredentialAuthority(
	ctx context.Context,
	uow ports.UnitOfWork,
	commandContext domain.CommandContext,
	scope domain.Scope,
	lease *domain.WorkLease,
	commandName string,
	now time.Time,
) (*leaseCredentialAuthority, error) {
	identity, ok := IdentityFromContext(ctx)
	credentialPresented := ok && identity.CredentialDigest != ""
	boundLease := lease != nil && lease.CredentialBindingVersion != 0
	if !s.requireIdentity && !credentialPresented && !boundLease {
		return nil, nil
	}
	if !ok || identity.CredentialID.Validate() != nil || identity.CredentialDigest == "" || identity.NamespaceID != scope.NamespaceID || identity.PrincipalID == "" || identity.PrincipalID != commandContext.PrincipalID || identity.Actor != commandContext.Actor {
		return nil, leaseCredentialAuthorityDenied("authenticated credential does not match the command scope")
	}
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if now.IsZero() {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "authoritative lease evaluation time is required")
	}
	repository, err := signingRepository(uow)
	if err != nil {
		return nil, err
	}
	leaf, err := repository.CredentialByDigest(ctx, identity.CredentialDigest)
	if err != nil {
		if code, _ := domain.ErrorCodeOf(err); code == domain.ErrorCodeNotFound {
			return nil, leaseCredentialAuthorityDenied("authenticated credential is unavailable")
		}
		return nil, err
	}
	if leaf.ID != identity.CredentialID || leaf.Digest != identity.CredentialDigest || leaf.NamespaceID != scope.NamespaceID || leaf.NamespaceID != identity.NamespaceID || leaf.PrincipalID != commandContext.PrincipalID || leaf.PrincipalID != identity.PrincipalID || leaf.Actor != commandContext.Actor || leaf.Actor != identity.Actor || leaf.Revoked || !now.UTC().Before(leaf.ExpiresAt) {
		return nil, leaseCredentialAuthorityDenied("authenticated credential is inactive or outside the command scope")
	}

	authority := &leaseCredentialAuthority{presentedID: leaf.ID, holderID: leaf.ID}
	if leaf.ParentDigest != "" {
		root, err := repository.CredentialByDigest(ctx, leaf.ParentDigest)
		if err != nil {
			if code, _ := domain.ErrorCodeOf(err); code == domain.ErrorCodeNotFound {
				return nil, leaseCredentialAuthorityDenied("parent credential is unavailable")
			}
			return nil, err
		}
		if root.ID.Validate() != nil || root.ID == leaf.ID || root.Digest != leaf.ParentDigest || root.ParentDigest != "" || root.NamespaceID != leaf.NamespaceID || root.PrincipalID != leaf.PrincipalID || root.Actor != leaf.Actor || root.Revoked || !now.UTC().Before(root.ExpiresAt) {
			return nil, leaseCredentialAuthorityDenied("parent credential is inactive or inconsistent")
		}
		authority.holderID = root.ID
		authority.sessionID = credentialIDPointer(leaf.ID)
	}
	if commandName != "" && s.requireIdentity {
		permission := commandPermission(commandName)
		snapshot, ok := uow.(ports.AccessSnapshotUnitOfWork)
		if !ok {
			return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "transactional lease credential authorization unavailable")
		}
		if err := snapshot.AuthorizeAccessSnapshot(ctx, ports.AccessSnapshotRequest{
			Authorization: ports.AuthorizationRequest{
				NamespaceID: scope.NamespaceID, OutcomeID: scope.OutcomeID,
				PrincipalID: commandContext.PrincipalID, Permission: permission,
				LegacyPermission: legacyPermissionForCommand(commandName, permission),
			},
			CredentialDigest: identity.CredentialDigest, Actor: identity.Actor, Now: now.UTC(),
		}); err != nil {
			return nil, err
		}
	}
	return authority, nil
}

func credentialIDPointer(id domain.ID) *domain.ID {
	value := id
	return &value
}

func leaseCredentialAuthorityDenied(message string) error {
	return domain.NewDetailedError(domain.ErrorCodeForbidden, message, domain.ErrorDetails{ReasonCode: "lease_credential_authority_unavailable"})
}

func leaseOwnerMismatch() error {
	return domain.NewDetailedError(domain.ErrorCodeLease, "this lease belongs to another credential", domain.ErrorDetails{ReasonCode: "lease_owned_by_other_credential"})
}

func leaseCredentialBindingUnavailable() error {
	return domain.NewDetailedError(domain.ErrorCodeLease, "this legacy lease has no credential ownership record", domain.ErrorDetails{ReasonCode: "lease_credential_binding_unavailable"})
}
