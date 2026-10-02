package ports

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type Permission string

const (
	PermissionWorkAdminCancel   Permission = "work:admin_cancel"
	PermissionWorkAdminComplete Permission = "work:admin_complete"
	PermissionAssessmentWaive   Permission = "assessment:waive"
)

func (p Permission) Valid() bool {
	switch p {
	case PermissionWorkAdminCancel, PermissionWorkAdminComplete, PermissionAssessmentWaive:
		return true
	default:
		return false
	}
}

type AuthorizationRequest struct {
	NamespaceID domain.ID
	PrincipalID string
	Permission  Permission
}

func (r AuthorizationRequest) Validate() error {
	if err := r.NamespaceID.Validate(); err != nil {
		return domain.WrapError(domain.ErrorCodeInvalidArgument, "authorization namespace is invalid", err)
	}
	if r.PrincipalID == "" {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "authorization principal is required")
	}
	if !r.Permission.Valid() {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "authorization permission is invalid")
	}
	return nil
}

type Authorizer interface {
	Authorize(ctx context.Context, request AuthorizationRequest) error
}

// DenyPrivilegedAuthorizer is the safe default until a deployment composes an
// explicit authorization adapter. Normal commands do not consult this port.
type DenyPrivilegedAuthorizer struct{}

func (DenyPrivilegedAuthorizer) Authorize(_ context.Context, request AuthorizationRequest) error {
	if err := request.Validate(); err != nil {
		return err
	}
	return domain.NewError(domain.ErrorCodeForbidden, "principal is not authorized for "+string(request.Permission))
}
