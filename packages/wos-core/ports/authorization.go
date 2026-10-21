package ports

import (
	"context"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type Permission string

const MaxPermissionValues = domain.MaxPermissionValues

const (
	PermissionWorkContractReturn           Permission = "work.contract.return"
	PermissionWorkCompleteDirect           Permission = "work.contract.complete_direct"
	PermissionWorkReviewAcquire            Permission = "work.review.acquire"
	PermissionWorkReviewDecide             Permission = "work.review.decide"
	PermissionSigningKeyEnroll             Permission = "identity.signing_key.enroll"
	PermissionSigningKeyRotate             Permission = "identity.signing_key.rotate"
	PermissionSigningKeyRevoke             Permission = "identity.signing_key.revoke"
	PermissionWorkContractAcquire          Permission = "work.contract.acquire"
	PermissionWorkContractRevoke           Permission = "work.contract.revoke"
	PermissionStateRead                    Permission = "state:read"
	PermissionOutcomeWrite                 Permission = "outcome:write"
	PermissionPlanningWrite                Permission = "planning:write"
	PermissionWorkWrite                    Permission = "work:write"
	PermissionRecordsWrite                 Permission = "records:write"
	PermissionAssessmentWrite              Permission = "assessment:write"
	PermissionConclusionWrite              Permission = "conclusion:write"
	PermissionNamespaceAdmin               Permission = "namespace:admin"
	PermissionIntegrationWrite             Permission = "integration:write"
	PermissionActorDelegate                Permission = "actor:delegate"
	PermissionWorkAdminCancel              Permission = "work:admin_cancel"
	PermissionWorkAdminComplete            Permission = "work:admin_complete"
	PermissionAssessmentWaive              Permission = "assessment:waive"
	PermissionCredentialsRead              Permission = "credentials:read"
	PermissionAuditRead                    Permission = "audit:read"
	PermissionObjectiveCreate              Permission = "objective:create"
	PermissionObjectiveUpdate              Permission = "objective:update"
	PermissionObjectiveStart               Permission = "objective:start"
	PermissionObjectiveConclude            Permission = "objective:conclude"
	PermissionObjectiveReopen              Permission = "objective:reopen"
	PermissionObjectiveCancel              Permission = "objective:cancel"
	PermissionObjectiveConfigureCompletion Permission = "objective:configure_completion"
	PermissionTaskCreate                   Permission = "task:create"
	PermissionTaskUpdateDefinition         Permission = "task:update_definition"
	PermissionTaskActivate                 Permission = "task:activate"
	PermissionTaskDefer                    Permission = "task:defer"
	PermissionTaskAssign                   Permission = "task:assign"
	PermissionTaskClaim                    Permission = "task:claim"
	PermissionTaskRenew                    Permission = "task:renew"
	PermissionTaskRelease                  Permission = "task:release"
	PermissionTaskDeliver                  Permission = "task:deliver"
	PermissionTaskReview                   Permission = "task:review"
	PermissionTaskAccept                   Permission = "task:accept"
	PermissionTaskRequestChanges           Permission = "task:request_changes"
	PermissionTaskWithdrawDelivery         Permission = "task:withdraw_delivery"
	PermissionTaskCancel                   Permission = "task:cancel"
	PermissionTaskReopen                   Permission = "task:reopen"
	PermissionTaskIntervene                Permission = "task:intervene"
	PermissionCriterionDefine              Permission = "criterion:define"
	PermissionCriterionAssess              Permission = "criterion:assess"
	PermissionCriterionWaive               Permission = "criterion:waive"
	PermissionDependencyWrite              Permission = "dependency:write"
	PermissionIssueCreate                  Permission = "issue:create"
	PermissionIssueUpdate                  Permission = "issue:update"
	PermissionIssueAssign                  Permission = "issue:assign"
	PermissionIssueInvestigate             Permission = "issue:investigate"
	PermissionIssueResolve                 Permission = "issue:resolve"
	PermissionIssueReopen                  Permission = "issue:reopen"
	PermissionBlockerCreate                Permission = "blocker:create"
	PermissionBlockerRelease               Permission = "blocker:release"
	PermissionBlockerCancel                Permission = "blocker:cancel"
	PermissionCredentialsIssue             Permission = "credentials:issue"
	PermissionCredentialsUpdate            Permission = "credentials:update"
	PermissionCredentialsRevoke            Permission = "credentials:revoke"
	PermissionCredentialsRotate            Permission = "credentials:rotate"
	PermissionPermissionsManage            Permission = "permissions:manage"
	PermissionPolicyManage                 Permission = "policy:manage"
)

func (p Permission) Valid() bool {
	return domain.PermissionOperationValid(string(p))
}

// ValidatePermissionValues verifies an explicit grant list. An empty list is
// valid but confers no permissions; duplicates and unknown names are rejected.
func ValidatePermissionValues(values []Permission) error {
	if len(values) > MaxPermissionValues {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "permission list exceeds 128 values")
	}
	seen := make(map[Permission]struct{}, len(values))
	for _, permission := range values {
		if !permission.Valid() {
			return domain.NewError(domain.ErrorCodeInvalidArgument, "unknown permission")
		}
		if _, exists := seen[permission]; exists {
			return domain.NewError(domain.ErrorCodeInvalidArgument, "duplicate permission")
		}
		seen[permission] = struct{}{}
	}
	return nil
}

type AuthorizationRequest struct {
	NamespaceID domain.ID
	OutcomeID   domain.ID
	PrincipalID string
	Permission  Permission
	// ResourceRef optionally narrows authorization to the entity whose authority
	// is being exercised. Outcome-level permissions remain in Permission and
	// OutcomeID; callers must not substitute authority over another endpoint.
	ResourceRef *domain.EntityRef
	// LegacyPermission is populated only by the exact, versioned application
	// command compatibility map. Storage accepts it only for migrated grants and
	// credential policies; transports must never accept it from callers.
	LegacyPermission Permission
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
	if r.LegacyPermission != "" && !r.LegacyPermission.Valid() {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "authorization compatibility permission is invalid")
	}
	if r.ResourceRef != nil {
		if err := r.ResourceRef.Validate(); err != nil {
			return domain.WrapError(domain.ErrorCodeInvalidArgument, "authorization resource reference is invalid", err)
		}
		if r.ResourceRef.Scope.NamespaceID != r.NamespaceID || (r.OutcomeID != "" && r.ResourceRef.Scope.OutcomeID != r.OutcomeID) {
			return domain.NewError(domain.ErrorCodeInvalidArgument, "authorization resource reference is outside the requested scope")
		}
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

// AllPermissions returns a fresh, bounded list of supported grant values.
func AllPermissions() []Permission {
	names := domain.PermissionOperationNames()
	permissions := make([]Permission, len(names))
	for i, name := range names {
		permissions[i] = Permission(name)
	}
	return permissions
}
