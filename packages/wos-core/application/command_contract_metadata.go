package application

import "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"

// CommandPermissionCondition describes an existing authorization branch. It is
// discovery metadata; only CommandAuthorizationPermissions and the guarded
// Application command decide the caller's actual requirements and authority.
type CommandPermissionCondition struct {
	When        string             `json:"when"`
	Permissions []ports.Permission `json:"permissions"`
}

type CommandAuthorizationDescription struct {
	PrimaryPermission       ports.Permission             `json:"primary_permission"`
	AlwaysRequired          []ports.Permission           `json:"always_required_permissions"`
	ConditionalRequirements []CommandPermissionCondition `json:"conditional_requirements"`
	RecheckedOnExecution    bool                         `json:"rechecked_on_execution"`
	SnapshotChecks          string                       `json:"snapshot_checks"`
}

// CommandAuthorizationDescriptionForPresentation reads the same exact-name
// registry as execution. Descriptions never evaluate a credential, decode a
// secret, resolve a business resource or imply that a grant is sufficient.
func CommandAuthorizationDescriptionForPresentation(name string) (CommandAuthorizationDescription, bool) {
	entry, ok := commandAuthorizationCatalog[name]
	if !ok || !entry.primary.Valid() {
		return CommandAuthorizationDescription{}, false
	}
	description := CommandAuthorizationDescription{
		PrimaryPermission: entry.primary, AlwaysRequired: []ports.Permission{entry.primary}, ConditionalRequirements: []CommandPermissionCondition{}, RecheckedOnExecution: true,
		SnapshotChecks: "Current credential, grants, scope, protocol, optimistic versions, policy and any execution/review authority are rechecked in the Application transaction. Discovery grants no authority.",
	}
	if entry.additional != nil {
		// Unconditional composite requirements deliberately ignore their input.
		for _, permission := range entry.additional(nil) {
			if permission.Valid() && permission != entry.primary {
				description.AlwaysRequired = append(description.AlwaysRequired, permission)
			}
		}
	}
	condition := func(when string, permissions ...ports.Permission) {
		description.ConditionalRequirements = append(description.ConditionalRequirements, CommandPermissionCondition{When: when, Permissions: permissions})
	}
	switch name {
	case "CreateOutcome", "CreateObjective", "CreateWorkItem":
		condition("initial_criteria is nonempty", ports.PermissionCriterionDefine)
	case "CreatePlanItems":
		condition("an item creates an Objective", ports.PermissionObjectiveCreate)
		condition("an item creates a Task", ports.PermissionTaskCreate)
		condition("any item includes initial_criteria", ports.PermissionCriterionDefine)
	case "RecordCriterionAssessment":
		condition("result is waived", ports.PermissionCriterionWaive)
	case "SyncWorkContract":
		condition("documentary artifacts, evidence or evidence_links are included", ports.PermissionRecordsWrite)
	case "ReturnSignedWork":
		condition("a valid signed return intent is processed", ports.PermissionWorkContractReturn)
		condition("signed material includes artifacts, evidence or evidence_links", ports.PermissionRecordsWrite)
		condition("completion_intent is direct, or auto resolves to direct under the current policy", ports.PermissionTaskAccept, ports.PermissionWorkCompleteDirect, ports.PermissionCriterionAssess, ports.PermissionAssessmentWrite, ports.PermissionConclusionWrite)
		condition("a supplied criterion assessment is waived", ports.PermissionAssessmentWaive)
	case "ReturnSignedReview":
		condition("a valid signed review decision is processed", ports.PermissionWorkReviewDecide)
		condition("signed material includes artifacts, evidence or evidence_links", ports.PermissionRecordsWrite)
		condition("decision is approved", ports.PermissionTaskAccept, ports.PermissionCriterionAssess, ports.PermissionAssessmentWrite, ports.PermissionConclusionWrite)
		condition("decision is changes_requested", ports.PermissionTaskRequestChanges)
		condition("a supplied criterion assessment is waived", ports.PermissionAssessmentWaive)
	}
	return description, true
}

// CommandRequiresSignedHostForPresentation describes the existing signed
// command family without parsing a public name or changing protocol guards.
func CommandRequiresSignedHostForPresentation(name string) bool {
	return signedCommand(name)
}
