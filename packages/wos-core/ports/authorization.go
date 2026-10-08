package ports

import (
	"context"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type Permission string

const (
	PermissionWorkContractReturn  Permission = "work.contract.return"
	PermissionWorkCompleteDirect  Permission = "work.contract.complete_direct"
	PermissionWorkReviewAcquire   Permission = "work.review.acquire"
	PermissionWorkReviewDecide    Permission = "work.review.decide"
	PermissionSigningKeyEnroll    Permission = "identity.signing_key.enroll"
	PermissionSigningKeyRotate    Permission = "identity.signing_key.rotate"
	PermissionSigningKeyRevoke    Permission = "identity.signing_key.revoke"
	PermissionWorkContractAcquire Permission = "work.contract.acquire"
	PermissionWorkContractRevoke  Permission = "work.contract.revoke"
	PermissionStateRead           Permission = "state:read"
	PermissionOutcomeWrite        Permission = "outcome:write"
	PermissionPlanningWrite       Permission = "planning:write"
	PermissionWorkWrite           Permission = "work:write"
	PermissionRecordsWrite        Permission = "records:write"
	PermissionAssessmentWrite     Permission = "assessment:write"
	PermissionConclusionWrite     Permission = "conclusion:write"
	PermissionNamespaceAdmin      Permission = "namespace:admin"
	PermissionIntegrationWrite    Permission = "integration:write"
	PermissionActorDelegate       Permission = "actor:delegate"
	PermissionWorkAdminCancel     Permission = "work:admin_cancel"
	PermissionWorkAdminComplete   Permission = "work:admin_complete"
	PermissionAssessmentWaive     Permission = "assessment:waive"
)

func (p Permission) Valid() bool {
	switch p {
	case PermissionWorkContractReturn, PermissionWorkCompleteDirect, PermissionWorkReviewAcquire, PermissionWorkReviewDecide, PermissionSigningKeyEnroll, PermissionSigningKeyRotate, PermissionSigningKeyRevoke, PermissionWorkContractAcquire, PermissionWorkContractRevoke, PermissionStateRead, PermissionOutcomeWrite, PermissionPlanningWrite, PermissionWorkWrite, PermissionRecordsWrite, PermissionAssessmentWrite, PermissionConclusionWrite, PermissionNamespaceAdmin, PermissionIntegrationWrite, PermissionActorDelegate, PermissionWorkAdminCancel, PermissionWorkAdminComplete, PermissionAssessmentWaive:
		return true
	default:
		return false
	}
}

type AuthorizationRequest struct {
	NamespaceID domain.ID
	OutcomeID   domain.ID
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

// TransactionalAuthorizer revalidates dynamic grants before idempotent disclosure
// and mutation, using the UnitOfWork's coherent security snapshot.
type TransactionalAuthorizer interface {
	AuthorizeInUnitOfWork(context.Context, UnitOfWork, AuthorizationRequest) error
}
type AccessSnapshotRequest struct {
	Authorization    AuthorizationRequest
	CredentialDigest string
	Actor            domain.ActorRef
	Now              time.Time
}
type AccessSnapshotUnitOfWork interface {
	AuthorizeAccessSnapshot(context.Context, AccessSnapshotRequest) error
}
