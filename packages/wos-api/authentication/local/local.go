package local

import (
	"context"
	"strings"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type Resolver struct {
	PrincipalID string
}

func New(principalID string) (Resolver, error) {
	principalID = strings.TrimSpace(principalID)
	if principalID == "" {
		return Resolver{}, domain.NewError(domain.ErrorCodeInvalidConfig, "local principal id is required")
	}
	return Resolver{PrincipalID: principalID}, nil
}

func (r Resolver) Principal() string {
	return r.PrincipalID
}

func (r Resolver) Actor() domain.ActorRef {
	return domain.ActorRef{
		Kind:     domain.ActorKindHuman,
		Provider: "local",
		ID:       r.PrincipalID,
	}
}

type Authorizer struct {
	PrincipalID                  string
	AllowAdministrativeOverrides bool
}

func NewAuthorizer(principalID string, allowAdministrativeOverrides bool) (Authorizer, error) {
	principalID = strings.TrimSpace(principalID)
	if principalID == "" {
		return Authorizer{}, domain.NewError(domain.ErrorCodeInvalidConfig, "local authorizer principal id is required")
	}
	return Authorizer{
		PrincipalID:                  principalID,
		AllowAdministrativeOverrides: allowAdministrativeOverrides,
	}, nil
}

func (a Authorizer) Authorize(_ context.Context, request ports.AuthorizationRequest) error {
	if err := request.Validate(); err != nil {
		return err
	}
	if request.PrincipalID != a.PrincipalID {
		return domain.NewError(domain.ErrorCodeForbidden, "local principal does not match authorization principal")
	}
	switch request.Permission {
	case ports.PermissionWorkContractAcquire, ports.PermissionStateRead, ports.PermissionOutcomeWrite, ports.PermissionPlanningWrite, ports.PermissionWorkWrite, ports.PermissionRecordsWrite, ports.PermissionAssessmentWrite, ports.PermissionConclusionWrite,
		ports.PermissionObjectiveCreate, ports.PermissionObjectiveUpdate, ports.PermissionObjectiveStart, ports.PermissionObjectiveConclude, ports.PermissionObjectiveReopen, ports.PermissionObjectiveCancel,
		ports.PermissionTaskCreate, ports.PermissionTaskUpdateDefinition, ports.PermissionTaskActivate, ports.PermissionTaskDefer, ports.PermissionTaskAssign, ports.PermissionTaskClaim, ports.PermissionTaskRenew, ports.PermissionTaskRelease, ports.PermissionTaskDeliver, ports.PermissionTaskReview, ports.PermissionTaskAccept, ports.PermissionTaskRequestChanges, ports.PermissionTaskWithdrawDelivery, ports.PermissionTaskCancel, ports.PermissionTaskReopen,
		ports.PermissionCriterionDefine, ports.PermissionCriterionAssess, ports.PermissionDependencyWrite,
		ports.PermissionIssueCreate, ports.PermissionIssueUpdate, ports.PermissionIssueAssign, ports.PermissionIssueInvestigate, ports.PermissionIssueResolve, ports.PermissionIssueReopen,
		ports.PermissionBlockerCreate, ports.PermissionBlockerRelease, ports.PermissionBlockerCancel:
		return nil
	}
	if !a.AllowAdministrativeOverrides {
		return domain.NewError(domain.ErrorCodeForbidden, "local administrative overrides are disabled")
	}
	switch request.Permission {
	case ports.PermissionWorkContractRevoke, ports.PermissionWorkAdminCancel, ports.PermissionWorkAdminComplete,
		ports.PermissionObjectiveConfigureCompletion, ports.PermissionTaskIntervene, ports.PermissionCriterionWaive,
		ports.PermissionCredentialsRead, ports.PermissionAuditRead, ports.PermissionCredentialsIssue, ports.PermissionCredentialsUpdate, ports.PermissionCredentialsRevoke, ports.PermissionCredentialsRotate,
		ports.PermissionPermissionsManage, ports.PermissionPolicyManage:
		return nil
	default:
		return domain.NewError(domain.ErrorCodeForbidden, "permission is not granted by local authorizer")
	}
}
