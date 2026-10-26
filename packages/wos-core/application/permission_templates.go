package application

import (
	"sort"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// permissionTemplateDefinitions is an immutable-by-convention registry of
// versioned, explicit permission lists. When a template changes, add a new
// version and keep prior definitions available for provenance resolution.
// Never derive these lists from permission names or prefixes.
var permissionTemplateDefinitions = [...]ports.PermissionTemplate{
	{
		ID:             ports.PermissionTemplateReader,
		Version:        ports.PermissionTemplateVersion1,
		NameKey:        "permissionTemplateReader",
		DescriptionKey: "permissionTemplateReaderDescription",
		Permissions:    []ports.Permission{ports.PermissionStateRead},
	},
	{
		ID:             ports.PermissionTemplatePlanner,
		Version:        ports.PermissionTemplateVersion1,
		NameKey:        "permissionTemplatePlanner",
		DescriptionKey: "permissionTemplatePlannerDescription",
		Permissions: []ports.Permission{
			ports.PermissionStateRead,
			ports.PermissionOutcomeWrite,
			ports.PermissionPlanningWrite,
			ports.PermissionObjectiveCreate,
			ports.PermissionObjectiveUpdate,
			ports.PermissionObjectiveStart,
			ports.PermissionObjectiveConfigureCompletion,
			ports.PermissionTaskCreate,
			ports.PermissionTaskUpdateDefinition,
			ports.PermissionTaskActivate,
			ports.PermissionTaskDefer,
			ports.PermissionTaskAssign,
			ports.PermissionCriterionDefine,
			ports.PermissionDependencyWrite,
		},
	},
	{
		ID:             ports.PermissionTemplateExecutor,
		Version:        ports.PermissionTemplateVersion1,
		NameKey:        "permissionTemplateExecutor",
		DescriptionKey: "permissionTemplateExecutorDescription",
		Permissions: []ports.Permission{
			ports.PermissionStateRead,
			ports.PermissionTaskClaim,
			ports.PermissionTaskRenew,
			ports.PermissionTaskRelease,
			ports.PermissionTaskDeliver,
			ports.PermissionRecordsWrite,
		},
	},
	{
		ID:             ports.PermissionTemplateReviewer,
		Version:        ports.PermissionTemplateVersion1,
		NameKey:        "permissionTemplateReviewer",
		DescriptionKey: "permissionTemplateReviewerDescription",
		Permissions: []ports.Permission{
			ports.PermissionStateRead,
			ports.PermissionTaskReview,
			ports.PermissionTaskAccept,
			ports.PermissionTaskRequestChanges,
			ports.PermissionCriterionAssess,
			ports.PermissionRecordsWrite,
		},
	},
	{
		ID:             ports.PermissionTemplateIssueOperator,
		Version:        ports.PermissionTemplateVersion1,
		NameKey:        "permissionTemplateIssueOperator",
		DescriptionKey: "permissionTemplateIssueOperatorDescription",
		Permissions: []ports.Permission{
			ports.PermissionStateRead,
			ports.PermissionIssueCreate,
			ports.PermissionIssueUpdate,
			ports.PermissionIssueAssign,
			ports.PermissionIssueInvestigate,
			ports.PermissionIssueResolve,
			ports.PermissionIssueReopen,
			ports.PermissionBlockerCreate,
			ports.PermissionBlockerRelease,
			ports.PermissionBlockerCancel,
			ports.PermissionRecordsWrite,
		},
	},
	{
		ID:             ports.PermissionTemplateAdministrator,
		Version:        ports.PermissionTemplateVersion1,
		NameKey:        "permissionTemplateAdministrator",
		DescriptionKey: "permissionTemplateAdministratorDescription",
		Permissions: []ports.Permission{
			ports.PermissionStateRead,
			ports.PermissionCredentialsRead,
			ports.PermissionCredentialsIssue,
			ports.PermissionCredentialsUpdate,
			ports.PermissionCredentialsRevoke,
			ports.PermissionCredentialsRotate,
			ports.PermissionAuditRead,
			ports.PermissionNamespaceAdmin,
			ports.PermissionActorDelegate,
			ports.PermissionPermissionsManage,
			ports.PermissionPolicyManage,
		},
	},
}

// permissionTemplateCurrentVersions is an explicit display order and current
// version index. Historical versions remain resolvable but are not offered as
// current choices when a new version is added.
var permissionTemplateCurrentVersions = [...]struct {
	id      ports.PermissionTemplateID
	version uint32
}{
	{ports.PermissionTemplateReader, ports.PermissionTemplateVersion1},
	{ports.PermissionTemplatePlanner, ports.PermissionTemplateVersion1},
	{ports.PermissionTemplateExecutor, ports.PermissionTemplateVersion1},
	{ports.PermissionTemplateReviewer, ports.PermissionTemplateVersion1},
	{ports.PermissionTemplateIssueOperator, ports.PermissionTemplateVersion1},
	{ports.PermissionTemplateAdministrator, ports.PermissionTemplateVersion1},
}

// ListPermissionTemplates returns the current, built-in templates in stable
// product order. Callers receive independent permission slices.
func ListPermissionTemplates() []ports.PermissionTemplate {
	templates := make([]ports.PermissionTemplate, 0, len(permissionTemplateCurrentVersions))
	for _, current := range permissionTemplateCurrentVersions {
		if template, ok := ResolvePermissionTemplate(current.id, current.version); ok {
			templates = append(templates, template)
		}
	}
	return templates
}

// ResolvePermissionTemplate returns the exact version requested. Unknown IDs
// and versions are not silently upgraded to the current template.
func ResolvePermissionTemplate(id ports.PermissionTemplateID, version uint32) (ports.PermissionTemplate, bool) {
	for _, definition := range permissionTemplateDefinitions {
		if definition.ID == id && definition.Version == version {
			return clonePermissionTemplate(definition), true
		}
	}
	return ports.PermissionTemplate{}, false
}

// PermissionTemplateSelection describes an exact template selection and an
// optional edited list. A nil EditedPermissions means the selected preset is
// used verbatim; a non-nil list is validated and materialized as a fixed set.
// With no TemplateID/Version, a non-nil list represents a custom selection.
type PermissionTemplateSelection struct {
	TemplateID        ports.PermissionTemplateID
	TemplateVersion   uint32
	EditedPermissions []ports.Permission
}

// PermissionTemplateMaterialization is a frozen permission list suitable for
// an issuer to persist. TemplateID/Version are present only when the exact
// selected preset remains unchanged. Custom lists have no template identity so
// a future catalog update cannot alter them through inheritance.
type PermissionTemplateMaterialization struct {
	TemplateID      ports.PermissionTemplateID `json:"role_template_id,omitempty"`
	TemplateVersion uint32                     `json:"role_template_version,omitempty"`
	Custom          bool                       `json:"custom"`
	Permissions     []ports.Permission         `json:"permissions"`
}

// MaterializePermissionTemplate resolves an exact template version and
// produces a detached list of concrete permissions. Edited lists are sorted
// and marked custom; a custom list cannot retain a misleading preset ID.
func MaterializePermissionTemplate(selection PermissionTemplateSelection) (PermissionTemplateMaterialization, error) {
	if selection.TemplateID == "" {
		if selection.TemplateVersion != 0 || selection.EditedPermissions == nil {
			return PermissionTemplateMaterialization{}, domain.NewError(domain.ErrorCodeInvalidArgument, "custom permissions require an explicit permission list and no template version")
		}
		if err := ports.ValidatePermissionValues(selection.EditedPermissions); err != nil {
			return PermissionTemplateMaterialization{}, err
		}
		return PermissionTemplateMaterialization{
			Custom:      true,
			Permissions: sortedPermissionCopy(selection.EditedPermissions),
		}, nil
	}

	template, ok := ResolvePermissionTemplate(selection.TemplateID, selection.TemplateVersion)
	if !ok {
		return PermissionTemplateMaterialization{}, domain.NewError(domain.ErrorCodeInvalidArgument, "permission template ID or version is unsupported")
	}
	if selection.EditedPermissions == nil {
		return PermissionTemplateMaterialization{
			TemplateID:      template.ID,
			TemplateVersion: template.Version,
			Permissions:     append([]ports.Permission(nil), template.Permissions...),
		}, nil
	}
	if err := ports.ValidatePermissionValues(selection.EditedPermissions); err != nil {
		return PermissionTemplateMaterialization{}, err
	}
	if samePermissionSet(selection.EditedPermissions, template.Permissions) {
		return PermissionTemplateMaterialization{
			TemplateID:      template.ID,
			TemplateVersion: template.Version,
			Permissions:     append([]ports.Permission(nil), template.Permissions...),
		}, nil
	}
	return PermissionTemplateMaterialization{
		Custom:      true,
		Permissions: sortedPermissionCopy(selection.EditedPermissions),
	}, nil
}

// PermissionTemplatePreviewInput carries the authorization bounds observed by
// the caller. NamespaceGrantPermissions is the Principal's explicit grant in
// the target Namespace. CredentialPolicyPermissions is a second, optional
// per-credential ceiling; when present, an empty list denies every permission.
// This helper does not authorize a request or replace server-side checks.
type PermissionTemplatePreviewInput struct {
	Selection                   PermissionTemplateSelection
	NamespaceGrantPermissions   []ports.Permission
	CredentialPolicyPermissions []ports.Permission
	CredentialPolicyPresent     bool
}

// PermissionTemplatePreview explains how the selected concrete list is bounded
// by the Principal's Namespace grant and, when present, the credential policy.
type PermissionTemplatePreview struct {
	Materialization             PermissionTemplateMaterialization `json:"materialization"`
	NamespaceGrantPermissions   []ports.Permission                `json:"namespace_grant_permissions"`
	CredentialPolicyPresent     bool                              `json:"credential_policy_present"`
	CredentialPolicyPermissions []ports.Permission                `json:"credential_policy_permissions,omitempty"`
	EffectivePermissions        []ports.Permission                `json:"effective_permissions"`
	ExcludedByNamespaceGrant    []ports.Permission                `json:"excluded_by_namespace_grant"`
	ExcludedByCredentialPolicy  []ports.Permission                `json:"excluded_by_credential_policy,omitempty"`
}

// PreviewPermissionTemplate deterministically intersects a materialized
// selection with the Principal's Namespace grant and optional credential
// restrictions. It returns explanatory exclusions but does not infer, add, or
// inherit permissions from any other template or role.
func PreviewPermissionTemplate(input PermissionTemplatePreviewInput) (PermissionTemplatePreview, error) {
	materialized, err := MaterializePermissionTemplate(input.Selection)
	if err != nil {
		return PermissionTemplatePreview{}, err
	}
	if err := ports.ValidatePermissionValues(input.NamespaceGrantPermissions); err != nil {
		return PermissionTemplatePreview{}, domain.WrapError(domain.ErrorCodeInvalidArgument, "namespace grant permissions are invalid", err)
	}
	if err := ports.ValidatePermissionValues(input.CredentialPolicyPermissions); err != nil {
		return PermissionTemplatePreview{}, domain.WrapError(domain.ErrorCodeInvalidArgument, "credential policy permissions are invalid", err)
	}
	if !input.CredentialPolicyPresent && len(input.CredentialPolicyPermissions) != 0 {
		return PermissionTemplatePreview{}, domain.NewError(domain.ErrorCodeInvalidArgument, "credential policy permissions require a present policy")
	}

	grant := permissionSet(input.NamespaceGrantPermissions)
	policy := permissionSet(input.CredentialPolicyPermissions)
	selected := materialized.Permissions
	preview := PermissionTemplatePreview{
		Materialization:             materialized,
		NamespaceGrantPermissions:   sortedPermissionCopy(input.NamespaceGrantPermissions),
		CredentialPolicyPresent:     input.CredentialPolicyPresent,
		CredentialPolicyPermissions: sortedPermissionCopy(input.CredentialPolicyPermissions),
		EffectivePermissions:        make([]ports.Permission, 0, len(selected)),
		ExcludedByNamespaceGrant:    make([]ports.Permission, 0),
		ExcludedByCredentialPolicy:  make([]ports.Permission, 0),
	}
	for _, permission := range selected {
		if _, ok := grant[permission]; !ok {
			preview.ExcludedByNamespaceGrant = append(preview.ExcludedByNamespaceGrant, permission)
			continue
		}
		if input.CredentialPolicyPresent {
			if _, ok := policy[permission]; !ok {
				preview.ExcludedByCredentialPolicy = append(preview.ExcludedByCredentialPolicy, permission)
				continue
			}
		}
		preview.EffectivePermissions = append(preview.EffectivePermissions, permission)
	}
	preview.EffectivePermissions = sortedPermissionCopy(preview.EffectivePermissions)
	preview.ExcludedByNamespaceGrant = sortedPermissionCopy(preview.ExcludedByNamespaceGrant)
	preview.ExcludedByCredentialPolicy = sortedPermissionCopy(preview.ExcludedByCredentialPolicy)
	return preview, nil
}

func clonePermissionTemplate(template ports.PermissionTemplate) ports.PermissionTemplate {
	template.Permissions = append([]ports.Permission(nil), template.Permissions...)
	return template
}

func permissionSet(values []ports.Permission) map[ports.Permission]struct{} {
	set := make(map[ports.Permission]struct{}, len(values))
	for _, value := range values {
		set[value] = struct{}{}
	}
	return set
}

func samePermissionSet(a, b []ports.Permission) bool {
	if len(a) != len(b) {
		return false
	}
	set := permissionSet(a)
	for _, permission := range b {
		if _, ok := set[permission]; !ok {
			return false
		}
	}
	return true
}

func sortedPermissionCopy(values []ports.Permission) []ports.Permission {
	copy := append([]ports.Permission{}, values...)
	sort.Slice(copy, func(i, j int) bool { return copy[i] < copy[j] })
	return copy
}
