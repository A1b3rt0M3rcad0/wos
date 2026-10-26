package ports

import (
	"context"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

const PermissionCompatibilityLegacyV1 uint32 = domain.PermissionCompatibilityLegacyV1

type PermissionMigrationSnapshot struct {
	NamespaceVersion domain.Version
	Grant            NamespaceGrant
	Credential       *Credential
	Policy           *domain.CredentialPolicy
}

type PermissionMigrationRead struct {
	NamespaceID  domain.ID
	PrincipalID  string
	CredentialID domain.ID
}

type PermissionMigrationApply struct {
	NamespaceID              domain.ID
	PrincipalID              string
	CredentialID             domain.ID
	ExpectedNamespaceVersion domain.Version
	ExpectedPolicyVersion    domain.Version
	PolicyVersionPresent     bool
	CallerPrincipalID        string
	CallerActor              domain.ActorRef
	CallerCredentialDigest   string
	IdempotencyKey           string
	Fingerprint              string
	Reason                   string
	Now                      time.Time
}

type PermissionMigrationResult struct {
	NamespaceID      domain.ID      `json:"namespace_id"`
	PrincipalID      string         `json:"principal_id"`
	CredentialID     domain.ID      `json:"credential_id,omitempty"`
	NamespaceVersion domain.Version `json:"namespace_version"`
	PolicyVersion    domain.Version `json:"policy_version,omitempty"`
	PolicyMigrated   bool           `json:"policy_migrated"`
	MapVersion       string         `json:"map_version"`
	AlreadyMigrated  bool           `json:"already_migrated,omitempty"`
	IdempotentReplay bool           `json:"idempotent_replay,omitempty"`
}

type PermissionMigrationRequest struct{ Command PermissionMigrationApply }

// PermissionCompatibilityStore provides a coherent, read-only preview and an
// atomic version-guarded compatibility cutover. It is deliberately not exposed
// by transports in this stage.
type PermissionCompatibilityStore interface {
	ReadPermissionMigration(context.Context, PermissionMigrationRead) (PermissionMigrationSnapshot, error)
	ApplyPermissionMigration(context.Context, PermissionMigrationRequest) (PermissionMigrationResult, error)
}
