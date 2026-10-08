package application

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func (s *Service) authorizeScopedRead(ctx context.Context, scope d.Scope) error {
	if !s.requireIdentity {
		return nil
	}
	id, ok := IdentityFromContext(ctx)
	if !ok || id.PrincipalID == "" {
		return d.NewError(d.ErrorCodeForbidden, "authenticated identity required")
	}
	return s.authorizer.Authorize(ctx, ports.AuthorizationRequest{NamespaceID: scope.NamespaceID, OutcomeID: scope.OutcomeID, PrincipalID: id.PrincipalID, Permission: ports.PermissionStateRead})
}
func (s *Service) authorizeScopedReadInUnitOfWork(ctx context.Context, u ports.UnitOfWork, scope d.Scope) error {
	if !s.requireIdentity {
		return nil
	}
	id, ok := IdentityFromContext(ctx)
	if !ok || id.PrincipalID == "" {
		return d.NewError(d.ErrorCodeForbidden, "authenticated identity required")
	}
	request := ports.AuthorizationRequest{NamespaceID: scope.NamespaceID, OutcomeID: scope.OutcomeID, PrincipalID: id.PrincipalID, Permission: ports.PermissionStateRead}
	if id.CredentialDigest != "" {
		snapshot, ok := u.(ports.AccessSnapshotUnitOfWork)
		if !ok {
			return d.NewError(d.ErrorCodeInvalidConfig, "transactional read authorization unavailable")
		}
		now, err := securityTransactionTime(ctx, u, s.clock)
		if err != nil {
			return err
		}
		return snapshot.AuthorizeAccessSnapshot(ctx, ports.AccessSnapshotRequest{Authorization: request, CredentialDigest: id.CredentialDigest, Actor: id.Actor, Now: now})
	}
	if dynamic, ok := s.authorizer.(ports.TransactionalAuthorizer); ok {
		return dynamic.AuthorizeInUnitOfWork(ctx, u, request)
	}
	return nil // Static local authorization was already checked before opening.
}

// Namespace metadata (instance trust/protocol and one's own identity) carries
// no other Outcome's state. Collections must instead apply these restrictions.
func (s *Service) authorizeNamespaceMetadataRead(ctx context.Context, ns d.ID) error {
	if !s.requireIdentity {
		return nil
	}
	return s.Authorize(ctx, ns, ports.PermissionStateRead)
}
func (s *Service) readOutcomeRestrictions(ctx context.Context, u ports.UnitOfWork, ns d.ID) ([]d.ID, error) {
	if err := s.authorizeScopedReadInUnitOfWork(ctx, u, d.Scope{NamespaceID: ns}); err != nil {
		return nil, err
	}
	id, ok := IdentityFromContext(ctx)
	if !s.requireIdentity || !ok || id.CredentialDigest == "" {
		return nil, nil
	}
	repo, err := signingRepository(u)
	if err != nil {
		return nil, err
	}
	credential, err := repo.CredentialByDigest(ctx, id.CredentialDigest)
	if err != nil {
		return nil, err
	}
	if credential.ParentDigest != "" {
		credential, err = repo.CredentialByDigest(ctx, credential.ParentDigest)
		if err != nil {
			return nil, err
		}
	}
	policy, err := repo.CredentialPolicy(ctx, ns, credential.ID)
	if code, _ := d.ErrorCodeOf(err); code == d.ErrorCodeNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return append([]d.ID{}, policy.AllowedOutcomeIDs...), nil
}
