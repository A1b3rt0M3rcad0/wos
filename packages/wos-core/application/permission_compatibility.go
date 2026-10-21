package application

import (
	"sort"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

const PermissionCompatibilityMapVersion = "legacy-v1"

// legacyCommandPermissions is a frozen, exact-name compatibility crosswalk
// from the pre-granular authorization rules to current capabilities. This is
// intentionally not inferred from command names or permission prefixes.
// Adding an application command does not add it to this table.
var legacyCommandPermissions = map[string]map[ports.Permission]ports.Permission{}

func init() {
	add := func(legacy ports.Permission, current ports.Permission, commands ...string) {
		for _, command := range commands {
			if legacyCommandPermissions[command] == nil {
				legacyCommandPermissions[command] = map[ports.Permission]ports.Permission{}
			}
			legacyCommandPermissions[command][current] = legacy
		}
	}

	// Former OutcomeWrite fallback/default operations that now have specific
	// capabilities. OutcomeWrite itself remains an exact current permission.
	add(ports.PermissionOutcomeWrite, ports.PermissionObjectiveCreate, "CreateObjective", "CreatePlanItems")
	add(ports.PermissionOutcomeWrite, ports.PermissionObjectiveUpdate, "UpdateObjective", "SetObjectiveOwners", "SetObjectiveParent")
	add(ports.PermissionOutcomeWrite, ports.PermissionObjectiveStart, "StartObjective")
	add(ports.PermissionOutcomeWrite, ports.PermissionObjectiveCancel, "CancelObjective")
	add(ports.PermissionOutcomeWrite, ports.PermissionCriterionDefine, "AddCriterion", "ReviseCriterion", "RetireCriterion", "CreateOutcome", "CreateObjective", "CreateWorkItem", "CreatePlanItems")
	add(ports.PermissionOutcomeWrite, ports.PermissionDependencyWrite, "AddDependency", "RemoveDependency", "ApplyDependencyChanges")

	// Former WorkWrite fallback operations. Composite command requirements are
	// mapped too, so a legacy grant never partially authorizes the old operation.
	add(ports.PermissionWorkWrite, ports.PermissionTaskCreate, "CreateWorkItem")
	add(ports.PermissionWorkWrite, ports.PermissionTaskCreate, "CreatePlanItems")
	add(ports.PermissionWorkWrite, ports.PermissionTaskUpdateDefinition, "UpdateWorkItem", "SetWorkItemExecutionSpec", "SetWorkItemNotBefore", "SetWorkItemObjective")
	add(ports.PermissionWorkWrite, ports.PermissionTaskActivate, "ActivateWorkItem")
	add(ports.PermissionWorkWrite, ports.PermissionTaskDefer, "DeferWorkItem")
	add(ports.PermissionWorkWrite, ports.PermissionTaskAssign, "SetWorkItemAssignees")
	add(ports.PermissionWorkWrite, ports.PermissionTaskClaim, "ClaimWorkItem", "ReclaimWorkItem")
	add(ports.PermissionWorkWrite, ports.PermissionTaskRenew, "RenewWorkItemLease", "RenewWorkContract", "ResumeWorkContract", "RenewSignedWorkContract", "ResumeSignedWorkContract")
	add(ports.PermissionWorkWrite, ports.PermissionTaskRelease, "ReleaseWorkItem")
	add(ports.PermissionWorkWrite, ports.PermissionTaskDeliver, "SubmitTaskForReview", "SubmitWorkResult", "SyncWorkContract")
	add(ports.PermissionWorkWrite, ports.PermissionTaskWithdrawDelivery, "WithdrawTaskDelivery")
	add(ports.PermissionTaskReview, ports.PermissionWorkReviewDecide, "AcceptTaskDelivery", "RequestTaskDeliveryChanges")
	add(ports.PermissionWorkWrite, ports.PermissionTaskAccept, "AcceptTaskDelivery")
	add(ports.PermissionWorkWrite, ports.PermissionConclusionWrite, "AcceptTaskDelivery")
	add(ports.PermissionWorkWrite, ports.PermissionTaskRequestChanges, "RequestTaskDeliveryChanges")
	add(ports.PermissionWorkWrite, ports.PermissionTaskAccept, "CompleteWorkItem")
	add(ports.PermissionWorkWrite, ports.PermissionTaskCancel, "CancelWorkItem")
	add(ports.PermissionWorkWrite, ports.PermissionTaskReopen, "ReopenWorkItem")
	add(ports.PermissionWorkWrite, ports.PermissionIssueCreate, "CreateIssue", "ReportIssueWithBlocker", "ReportIssueWithBlockers")
	add(ports.PermissionWorkWrite, ports.PermissionIssueUpdate, "UpdateIssue", "RelateIssueToWorkItem", "RelateIssueWorkItem", "RemoveIssueWorkItemRelation")
	add(ports.PermissionWorkWrite, ports.PermissionIssueAssign, "AssignIssue", "UnassignIssue")
	add(ports.PermissionWorkWrite, ports.PermissionIssueInvestigate, "InvestigateIssue", "StartIssueInvestigation")
	add(ports.PermissionWorkWrite, ports.PermissionIssueResolve, "ResolveIssue", "ResolveIssueAndBlockers", "MarkIssueDuplicate", "MarkIssueWontFix")
	add(ports.PermissionWorkWrite, ports.PermissionIssueReopen, "ReopenIssue")
	add(ports.PermissionWorkWrite, ports.PermissionBlockerCreate, "CreateBlocker", "ReportIssueWithBlocker", "ReportIssueWithBlockers")
	add(ports.PermissionWorkWrite, ports.PermissionBlockerRelease, "ResolveBlocker", "ResolveIssueAndBlockers")
	add(ports.PermissionWorkWrite, ports.PermissionBlockerCancel, "CancelBlocker")
	add(ports.PermissionWorkWrite, ports.PermissionRecordsWrite, "AppendWorkActivity")
	// Former exact contract/review permissions remain usable only on their old
	// protocol commands. No key-management permission is expanded.
	add(ports.PermissionWorkContractAcquire, ports.PermissionTaskClaim, "AcquireSignedWorkContract", "AcquireNextSignedWorkContract", "AcquireWorkContract", "AcquireNextWorkContract")
	add(ports.PermissionWorkContractReturn, ports.PermissionTaskDeliver, "ReturnSignedWork")
	add(ports.PermissionWorkCompleteDirect, ports.PermissionTaskAccept, "ReturnSignedWork")
	add(ports.PermissionAssessmentWrite, ports.PermissionCriterionAssess, "ReturnSignedWork", "ReturnSignedReview")
	add(ports.PermissionAssessmentWaive, ports.PermissionCriterionWaive, "ReturnSignedWork", "ReturnSignedReview")
	add(ports.PermissionWorkReviewAcquire, ports.PermissionTaskReview, "AcquireSignedReviewContract", "AcquireNextSignedReviewContract", "RenewSignedReviewContract", "ResumeSignedReviewContract")
	add(ports.PermissionWorkReviewDecide, ports.PermissionTaskReview, "ReturnSignedReview")
	add(ports.PermissionWorkReviewDecide, ports.PermissionTaskAccept, "ReturnSignedReview")
	add(ports.PermissionWorkReviewDecide, ports.PermissionTaskRequestChanges, "ReturnSignedReview")
	add(ports.PermissionAssessmentWrite, ports.PermissionCriterionAssess, "AttestCriterion", "RecordCriterionAssessment")
	add(ports.PermissionAssessmentWaive, ports.PermissionCriterionWaive, "RecordCriterionAssessment")
	add(ports.PermissionConclusionWrite, ports.PermissionObjectiveConclude, "AchieveObjective")
	add(ports.PermissionConclusionWrite, ports.PermissionObjectiveReopen, "ReopenObjective")
	add(ports.PermissionConclusionWrite, ports.PermissionTaskAccept, "FinalizeWorkContract")
	add(ports.PermissionNamespaceAdmin, ports.PermissionPolicyManage, "SetNamespaceWorkProtocol", "SetCredentialPolicy", "SetAcceptancePolicy", "SetReviewGroup")
	add(ports.PermissionNamespaceAdmin, ports.PermissionTaskIntervene, "InterveneSignedReviewCase", "ReconcileExpiredWorkContracts")
}

func legacyPermissionForCommand(command string, current ports.Permission) ports.Permission {
	return legacyCommandPermissions[command][current]
}

func legacyCompatibilityCapabilities(permissions []ports.Permission) []ports.Permission {
	legacySet := make(map[ports.Permission]struct{}, len(permissions))
	for _, permission := range permissions {
		legacySet[permission] = struct{}{}
	}
	set := make(map[ports.Permission]struct{}, len(legacySet))
	for permission := range legacySet {
		set[permission] = struct{}{}
	}
	for _, byCapability := range legacyCommandPermissions {
		for current, legacy := range byCapability {
			// Expand only from the source permissions. A newly derived capability
			// must not be treated as another legacy grant in the same pass.
			if _, exists := legacySet[legacy]; exists {
				set[current] = struct{}{}
			}
		}
	}
	result := make([]ports.Permission, 0, len(set))
	for permission := range set {
		result = append(result, permission)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func effectivePermissionIntersection(grant, policy []ports.Permission, policyPresent bool) []ports.Permission {
	return effectivePermissionSet(grant, ports.PermissionCompatibilityLegacyV1, policy, ports.PermissionCompatibilityLegacyV1, policyPresent)
}

func effectivePermissionSet(grant []ports.Permission, grantCompatibility uint32, policy []ports.Permission, policyCompatibility uint32, policyPresent bool) []ports.Permission {
	contains := func(values []ports.Permission, sought ports.Permission) bool {
		for _, value := range values {
			if value == sought {
				return true
			}
		}
		return false
	}
	set := map[ports.Permission]struct{}{}
	for _, current := range ports.AllPermissions() {
		if contains(grant, current) && (!policyPresent || contains(policy, current)) {
			set[current] = struct{}{}
		}
	}
	for _, byCapability := range legacyCommandPermissions {
		for current, legacy := range byCapability {
			grantMigrated := grantCompatibility == ports.PermissionCompatibilityLegacyV1
			policyMigrated := !policyPresent || policyCompatibility == ports.PermissionCompatibilityLegacyV1
			if grantMigrated && policyMigrated && contains(grant, legacy) && (!policyPresent || contains(policy, legacy)) {
				set[current] = struct{}{}
			}
		}
	}
	result := make([]ports.Permission, 0, len(set))
	for permission := range set {
		result = append(result, permission)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}

func strictPermissionIntersection(grant, policy []ports.Permission, policyPresent bool) []ports.Permission {
	set := map[ports.Permission]struct{}{}
	for _, current := range ports.AllPermissions() {
		grantAllows, policyAllows := false, !policyPresent
		for _, permission := range grant {
			grantAllows = grantAllows || permission == current
		}
		for _, permission := range policy {
			policyAllows = policyAllows || permission == current
		}
		if grantAllows && policyAllows {
			set[current] = struct{}{}
		}
	}
	result := make([]ports.Permission, 0, len(set))
	for permission := range set {
		result = append(result, permission)
	}
	sort.Slice(result, func(i, j int) bool { return result[i] < result[j] })
	return result
}
