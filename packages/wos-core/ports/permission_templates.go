package ports

// PermissionTemplateID identifies a stable, named set of explicit permissions.
// A template is descriptive provenance only; it does not grant authority.
type PermissionTemplateID string

const (
	PermissionTemplateReader        PermissionTemplateID = "reader"
	PermissionTemplatePlanner       PermissionTemplateID = "planner"
	PermissionTemplateExecutor      PermissionTemplateID = "executor"
	PermissionTemplateReviewer      PermissionTemplateID = "reviewer"
	PermissionTemplateIssueOperator PermissionTemplateID = "issue_operator"
	PermissionTemplateAdministrator PermissionTemplateID = "administrator"

	PermissionTemplateVersion1 uint32 = 1
)

// PermissionTemplate is a localized catalog descriptor plus a concrete list of
// permissions. Consumers must materialize Permissions into the relevant grant
// and credential policy; ID and Version are provenance, not an authorization
// source. Each value returned by the application catalog owns its Permissions
// slice.
type PermissionTemplate struct {
	ID             PermissionTemplateID `json:"role_template_id"`
	Version        uint32               `json:"role_template_version"`
	NameKey        string               `json:"name_key"`
	DescriptionKey string               `json:"description_key"`
	Permissions    []Permission         `json:"permissions"`
}
