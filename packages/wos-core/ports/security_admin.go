package ports

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"time"
)

// Security administration coordinates on the Namespace, rather than an Outcome.
type SecurityAdminCommand struct {
	NamespaceName         string                   `json:"namespace_name,omitempty"`
	NewNamespace          *Namespace               `json:"-"`
	NamespaceID           domain.ID                `json:"namespace_id"`
	ExpectedVersion       domain.Version           `json:"expected_namespace_version"`
	Operation             string                   `json:"operation"`
	Grant                 *NamespaceGrant          `json:"grant,omitempty"`
	Credential            *Credential              `json:"credential,omitempty"`
	CredentialPolicy      *domain.CredentialPolicy `json:"-"`
	CredentialMetadata    *CredentialMetadata      `json:"-"`
	CredentialName        string                   `json:"credential_name,omitempty"`
	CredentialDescription string                   `json:"credential_description,omitempty"`
	RoleTemplateID        PermissionTemplateID     `json:"role_template_id,omitempty"`
	RoleTemplateVersion   uint32                   `json:"role_template_version,omitempty"`
	CredentialPermissions []Permission             `json:"permissions,omitempty"`
	AllowedOutcomeIDs     []domain.ID              `json:"allowed_outcome_ids,omitempty"`
	AllowedSigningKeyIDs  []domain.ID              `json:"allowed_signing_key_ids,omitempty"`
	CredentialID          domain.ID                `json:"credential_id,omitempty"`
}

// SecurityAdminPermission binds each administrative operation to its specific
// authority. Namespace administration remains limited to namespace creation.
func SecurityAdminPermission(operation string) (Permission, bool) {
	switch operation {
	case "create_namespace":
		return PermissionNamespaceAdmin, true
	case "set_grant":
		return PermissionPermissionsManage, true
	case "issue_credential":
		return PermissionCredentialsIssue, true
	case "revoke_credential":
		return PermissionCredentialsRevoke, true
	default:
		return "", false
	}
}

type SecurityAdminRequest struct {
	Command          SecurityAdminCommand
	PrincipalID      string
	Actor            domain.ActorRef
	CredentialDigest string
	IdempotencyKey   string
	Fingerprint      string
	Now              time.Time
}
type SecurityAdminResult struct {
	CreatedNamespace   *Namespace          `json:"created_namespace,omitempty"`
	NamespaceVersion   domain.Version      `json:"namespace_version"`
	Credential         *Credential         `json:"credential,omitempty"`
	CredentialMetadata *CredentialMetadata `json:"credential_metadata,omitempty"`
	IdempotentReplay   bool                `json:"idempotent_replay"`
	TokenOmitted       bool                `json:"token_omitted,omitempty"`
}
type SecurityAdministrationStore interface {
	ApplySecurityAdmin(context.Context, SecurityAdminRequest) (SecurityAdminResult, error)
	// ReadSecurityAdminReceipt returns an authorized replay before operation
	// preparation. This lets retries recover safe metadata even when mutable
	// target state (such as a recipient's grant) changed after the first commit.
	ReadSecurityAdminReceipt(context.Context, SecurityAdminRequest) (SecurityAdminResult, bool, error)
	ReadSecurityAdmin(context.Context, domain.ID) (SecurityAdminSnapshot, error)
}

// SecurityAdminVersionReader supports legacy application methods that need
// only the Namespace CAS head; it avoids reading grants, credentials, or audit.
type SecurityAdminVersionReader interface {
	ReadSecurityAdminVersion(context.Context, domain.ID) (domain.Version, error)
}
type SecurityAudit struct {
	NamespaceVersion domain.Version  `json:"namespace_version"`
	PrincipalID      string          `json:"principal_id"`
	Actor            domain.ActorRef `json:"actor_ref"`
	Operation        string          `json:"operation"`
	Target           string          `json:"target"`
	RecordedAt       time.Time       `json:"recorded_at"`
}
type SecurityAdminSnapshot struct {
	Namespace          Namespace            `json:"namespace"`
	NamespaceVersion   domain.Version       `json:"namespace_version"`
	Grants             []NamespaceGrant     `json:"grants"`
	Credentials        []Credential         `json:"credentials"`
	CredentialMetadata []CredentialMetadata `json:"credential_metadata,omitempty"`
	Audit              []SecurityAudit      `json:"audit"`
	Truncated          bool                 `json:"truncated"`
}
