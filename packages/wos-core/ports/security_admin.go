package ports

import (
	"context"
	"encoding/json"
	"sort"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// Security administration coordinates on the Namespace, rather than an Outcome.
type SecurityAdminCommand struct {
	NamespaceName            string                   `json:"namespace_name,omitempty"`
	NewNamespace             *Namespace               `json:"-"`
	NamespaceID              domain.ID                `json:"namespace_id"`
	ExpectedVersion          domain.Version           `json:"expected_namespace_version"`
	Operation                string                   `json:"operation"`
	Grant                    *NamespaceGrant          `json:"grant,omitempty"`
	Credential               *Credential              `json:"credential,omitempty"`
	CredentialPolicy         *domain.CredentialPolicy `json:"-"`
	CredentialMetadata       *CredentialMetadata      `json:"-"`
	CredentialName           string                   `json:"credential_name,omitempty"`
	CredentialDescription    string                   `json:"credential_description,omitempty"`
	RoleTemplateID           PermissionTemplateID     `json:"role_template_id,omitempty"`
	RoleTemplateVersion      uint32                   `json:"role_template_version,omitempty"`
	CredentialPermissions    []Permission             `json:"permissions,omitempty"`
	AllowedOutcomeIDs        []domain.ID              `json:"allowed_outcome_ids,omitempty"`
	AllowedSigningKeyIDs     []domain.ID              `json:"allowed_signing_key_ids,omitempty"`
	CredentialID             domain.ID                `json:"credential_id,omitempty"`
	ExpectedMetadataRevision domain.Version           `json:"expected_metadata_revision,omitempty"`
	ExpectedPolicyRevision   domain.Version           `json:"expected_policy_revision,omitempty"`
	CredentialExpiresAt      *time.Time               `json:"expires_at,omitempty"`
	OverlapUntil             *time.Time               `json:"overlap_until,omitempty"`
	Reason                   string                   `json:"reason,omitempty"`
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
	case "update_credential":
		return PermissionCredentialsUpdate, true
	case "rotate_credential":
		return PermissionCredentialsRotate, true
	default:
		return "", false
	}
}

type SecurityAdminRequest struct {
	Command            SecurityAdminCommand
	PrincipalID        string
	Actor              domain.ActorRef
	CallerCredentialID domain.ID
	CredentialDigest   string
	IdempotencyKey     string
	Fingerprint        string
	Now                time.Time
}
type SecurityAdminResult struct {
	CreatedNamespace   *Namespace                  `json:"created_namespace,omitempty"`
	NamespaceVersion   domain.Version              `json:"namespace_version"`
	Credential         *Credential                 `json:"credential,omitempty"`
	CredentialMetadata *CredentialMetadata         `json:"credential_metadata,omitempty"`
	RevocationImpact   *CredentialRevocationImpact `json:"revocation_impact,omitempty"`
	IdempotentReplay   bool                        `json:"idempotent_replay"`
	TokenOmitted       bool                        `json:"token_omitted,omitempty"`
}
type SecurityAdministrationStore interface {
	ApplySecurityAdmin(context.Context, SecurityAdminRequest) (SecurityAdminResult, error)
	// ReadSecurityAdminReceipt returns an authorized replay before operation
	// preparation. This lets retries recover safe metadata even when mutable
	// target state (such as a recipient's grant) changed after the first commit.
	ReadSecurityAdminReceipt(context.Context, SecurityAdminRequest) (SecurityAdminResult, bool, error)
	ReadSecurityAdmin(context.Context, domain.ID) (SecurityAdminSnapshot, error)
}

// SecurityAdminReadAccess is trusted application context for a secret-free
// administrative read. The credential digest is never persisted or returned.
type SecurityAdminReadAccess struct {
	NamespaceID      domain.ID
	PrincipalID      string
	CredentialID     domain.ID
	CredentialDigest string
	Actor            domain.ActorRef
	Permissions      []Permission
	Now              time.Time
}

type AuthorizedSecurityAdministrationReader interface {
	ReadSecurityAdminAuthorized(context.Context, SecurityAdminReadAccess) (SecurityAdminSnapshot, error)
}

// SecurityAdminVersionReader supports legacy application methods that need
// only the Namespace CAS head; it avoids reading grants, credentials, or audit.
type SecurityAdminVersionReader interface {
	ReadSecurityAdminVersion(context.Context, domain.ID) (domain.Version, error)
}

// CredentialManagementQuery is a bounded, secret-free administration query.
// Cursor is the last credential ID returned by the previous page.
type CredentialManagementQuery struct {
	NamespaceID domain.ID
	Limit       int
	Cursor      domain.ID
	PrincipalID string
	Status      string
	ActorKind   domain.ActorKind
	ActorID     string
	Now         time.Time
}

// CredentialAdministrativeRecord contains no bearer token, digest, or parent
// digest. EffectivePolicy is the credential policy intersected with the
// Principal's current Namespace grant.
type CredentialAdministrativeRecord struct {
	Credential                          Credential               `json:"credential"`
	Metadata                            CredentialMetadata       `json:"metadata"`
	Policy                              *domain.CredentialPolicy `json:"policy,omitempty"`
	EffectivePolicy                     domain.CredentialPolicy  `json:"effective_policy"`
	GrantPermissionCompatibilityVersion uint32                   `json:"-"`
	PolicyExplicit                      bool                     `json:"policy_explicit"`
	Status                              string                   `json:"status"`
}

type CredentialManagementPage struct {
	Items      []CredentialAdministrativeRecord `json:"items"`
	NextCursor domain.ID                        `json:"next_cursor,omitempty"`
}

type CredentialManagementReader interface {
	ListManagedCredentials(context.Context, CredentialManagementQuery) (CredentialManagementPage, error)
	ReadManagedCredential(context.Context, domain.ID, domain.ID, time.Time) (CredentialAdministrativeRecord, error)
}

// AuthorizedCredentialManagementReader guarantees that the caller's current
// root/session, Namespace grant, and credential policy are checked in the same
// read snapshot that returns credential administration data.
type AuthorizedCredentialManagementReader interface {
	ListManagedCredentialsAuthorized(context.Context, CredentialManagementQuery, SecurityAdminReadAccess) (CredentialManagementPage, error)
	ReadManagedCredentialAuthorized(context.Context, domain.ID, domain.ID, time.Time, SecurityAdminReadAccess) (CredentialAdministrativeRecord, error)
	PreviewCredentialRevocation(context.Context, domain.ID, domain.ID, time.Time, SecurityAdminReadAccess) (CredentialRevocationImpact, error)
}

// CredentialRevocationImpact describes persisted authority linked directly to
// a root credential. It intentionally does not claim that external workers
// stopped or attribute legacy leases that carry no credential binding.
type CredentialRevocationImpact struct {
	NamespaceID                      domain.ID `json:"namespace_id"`
	CredentialID                     domain.ID `json:"credential_id"`
	PrincipalID                      string    `json:"principal_id"`
	ActiveSessions                   int       `json:"active_sessions"`
	ActiveSignedWorkContracts        int       `json:"active_signed_work_contracts"`
	ActiveSignedReviewContracts      int       `json:"active_signed_review_contracts"`
	LegacyLeasesMayBeUnattributed    bool      `json:"legacy_leases_may_be_unattributed"`
	LegacyContractsMayBeUnattributed bool      `json:"legacy_contracts_may_be_unattributed"`
	ExternalProcessesStopped         bool      `json:"external_processes_stopped"`
	AuthorityRemovedOnCommit         bool      `json:"authority_removed_on_commit"`
}

// CredentialUseObserver stores a coalesced last-used hint separately from
// domain transactions. Implementations skip writes while the observation is
// still within the provided interval.
type CredentialUseObserver interface {
	ObserveCredentialUse(context.Context, domain.ID, domain.ID, time.Time, time.Duration) error
}
type SecurityAudit struct {
	NamespaceVersion   domain.Version  `json:"namespace_version"`
	PrincipalID        string          `json:"principal_id"`
	Actor              domain.ActorRef `json:"actor_ref"`
	AuthorCredentialID domain.ID       `json:"author_credential_id,omitempty"`
	RootCredentialID   domain.ID       `json:"root_credential_id,omitempty"`
	Operation          string          `json:"operation"`
	Target             string          `json:"target"`
	Reason             string          `json:"reason,omitempty"`
	Before             json.RawMessage `json:"before,omitempty"`
	After              json.RawMessage `json:"after,omitempty"`
	RecordedAt         time.Time       `json:"recorded_at"`
}

// SecurityAuditGrantState is the canonical, secret-free audit projection of a
// Namespace grant. Permission compatibility is security-relevant because it
// controls which legacy command aliases remain authorized.
func SecurityAuditGrantState(exists bool, permissions []Permission, compatibilityVersion uint32) map[string]any {
	copyPermissions := append([]Permission(nil), permissions...)
	sort.Slice(copyPermissions, func(i, j int) bool { return copyPermissions[i] < copyPermissions[j] })
	if copyPermissions == nil {
		copyPermissions = []Permission{}
	}
	return map[string]any{
		"exists":                           exists,
		"permissions":                      copyPermissions,
		"permission_compatibility_version": compatibilityVersion,
	}
}

// SecurityAuditCredentialState projects the full effective authorization
// policy without recording bearer material, digests, or parent digests. It is
// shared by every storage adapter so audit records have the same shape.
func SecurityAuditCredentialState(record CredentialAdministrativeRecord) map[string]any {
	policy := record.EffectivePolicy
	operations := append([]string{}, policy.PermittedOperations...)
	sort.Strings(operations)
	outcomes := make([]string, 0, len(policy.AllowedOutcomeIDs))
	for _, id := range policy.AllowedOutcomeIDs {
		outcomes = append(outcomes, id.String())
	}
	sort.Strings(outcomes)
	keys := make([]string, 0, len(policy.AllowedSigningKeyIDs))
	for _, id := range policy.AllowedSigningKeyIDs {
		keys = append(keys, id.String())
	}
	sort.Strings(keys)
	return map[string]any{
		"exists":                                 true,
		"credential_id":                          record.Credential.ID,
		"principal_id":                           record.Credential.PrincipalID,
		"actor_ref":                              record.Credential.Actor,
		"status":                                 record.Status,
		"name":                                   record.Metadata.Name,
		"description":                            record.Metadata.Description,
		"metadata_revision":                      record.Metadata.MetadataRevision,
		"policy_revision":                        policy.Version,
		"grant_permission_compatibility_version": record.GrantPermissionCompatibilityVersion,
		"expires_at":                             record.Credential.ExpiresAt.UTC(),
		"permissions":                            operations,
		"policy_explicit":                        record.PolicyExplicit,
		"effective_policy": map[string]any{
			"permission_compatibility_version": policy.PermissionCompatibilityVersion,
			"permitted_operations":             operations,
			"acceptance_floor":                 policy.AcceptanceFloor,
			"allowed_outcome_ids":              outcomes,
			"allowed_signing_key_ids":          keys,
			"max_active_work_contracts":        policy.MaxActiveWorkContracts,
			"max_active_review_contracts":      policy.MaxActiveReviewContracts,
		},
	}
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
