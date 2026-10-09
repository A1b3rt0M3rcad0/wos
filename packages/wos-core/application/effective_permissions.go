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
	identity, ok := IdentityFromContext(ctx)
	result := []ports.Permission{}
	for _, permission := range ports.AllPermissions() {
		if !s.requireIdentity && !privilegedPresentationPermission(permission) {
			result = append(result, permission)
			continue
		}
		if !ok {
			continue
		}
		err = s.authorizer.Authorize(ctx, ports.AuthorizationRequest{NamespaceID: scope.NamespaceID, OutcomeID: scope.OutcomeID, PrincipalID: identity.PrincipalID, Permission: permission})
		if err == nil {
			result = append(result, permission)
			continue
		}
		if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeForbidden {
			return nil, err
		}
	}
	// A revoked session cannot succeed just because each candidate permission was denied.
	if scope.OutcomeID.IsZero() {
		err = s.authorizeNamespaceMetadataRead(ctx, scope.NamespaceID)
	} else {
		err = s.authorizeScopedRead(ctx, scope)
	}
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
