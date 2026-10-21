package application

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// EffectivePermissions is presentation metadata, never execution authority.
// Revocation and scoped credential restrictions are checked on each query.
func (s *Service) EffectivePermissions(ctx context.Context, scope domain.Scope) ([]ports.Permission, error) {
	if err := scope.NamespaceID.Validate(); err != nil {
		return nil, err
	}
	var err error
	if scope.OutcomeID.IsZero() {
		err = s.authorizeNamespaceMetadataRead(ctx, scope.NamespaceID)
	} else {
		if err = scope.Validate(); err == nil {
			err = s.authorizeScopedRead(ctx, scope)
		}
	}
	if err != nil {
		return nil, err
	}
	result := []ports.Permission{}
	if !s.requireIdentity {
		for _, permission := range ports.AllPermissions() {
			if !privilegedPresentationPermission(permission) {
				result = append(result, permission)
			}
		}
		return result, nil
	}
	identity, ok := IdentityFromContext(ctx)
	if !ok {
		return nil, domain.NewError(domain.ErrorCodeForbidden, "authenticated identity required")
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer uow.Rollback()
	now, err := securityTransactionTime(ctx, uow, s.clock)
	if err != nil {
		return nil, err
	}
	authorize := func(permission ports.Permission) error {
		request := ports.AuthorizationRequest{NamespaceID: scope.NamespaceID, OutcomeID: scope.OutcomeID, PrincipalID: identity.PrincipalID, Permission: permission}
		if identity.CredentialDigest != "" {
			snapshot, ok := uow.(ports.AccessSnapshotUnitOfWork)
			if !ok {
				return domain.NewError(domain.ErrorCodeInvalidConfig, "transactional permission query unavailable")
			}
			return snapshot.AuthorizeAccessSnapshot(ctx, ports.AccessSnapshotRequest{Authorization: request, CredentialDigest: identity.CredentialDigest, Actor: identity.Actor, Now: now})
		}
		if transactional, ok := s.authorizer.(ports.TransactionalAuthorizer); ok {
			return transactional.AuthorizeInUnitOfWork(ctx, uow, request)
		}
		return s.authorizer.Authorize(ctx, request)
	}
	readPermission := ports.PermissionStateRead
	if err = authorize(readPermission); err != nil {
		return nil, err
	}
	for _, permission := range ports.AllPermissions() {
		err = authorize(permission)
		if err == nil {
			result = append(result, permission)
			continue
		}
		if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeForbidden {
			return nil, err
		}
	}
	// Recheck read permission in the same transaction so a revoked session
	// cannot succeed merely because every candidate permission was denied.
	err = authorize(readPermission)
	if err != nil {
		return nil, err
	}
	return result, nil
}
func privilegedPresentationPermission(p ports.Permission) bool {
	switch p {
	case ports.PermissionNamespaceAdmin, ports.PermissionActorDelegate, ports.PermissionWorkAdminCancel, ports.PermissionWorkAdminComplete, ports.PermissionAssessmentWaive, ports.PermissionWorkContractAcquire, ports.PermissionWorkContractRevoke, ports.PermissionWorkContractReturn, ports.PermissionWorkCompleteDirect, ports.PermissionWorkReviewAcquire, ports.PermissionWorkReviewDecide, ports.PermissionSigningKeyEnroll, ports.PermissionSigningKeyRotate, ports.PermissionSigningKeyRevoke:
		return true
	}
	return false
}
