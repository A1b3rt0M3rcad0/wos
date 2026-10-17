package ports

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"time"
)

// Security administration coordinates on the Tenant, rather than an Outcome.
type SecurityAdminCommand struct {
	TenantName      string         `json:"tenant_name,omitempty"`
	NewTenant       *Tenant        `json:"-"`
	TenantID        domain.ID      `json:"tenant_id"`
	ExpectedVersion domain.Version `json:"expected_tenant_version"`
	Operation       string         `json:"operation"`
	Grant           *TenantGrant   `json:"grant,omitempty"`
	Credential      *Credential    `json:"credential,omitempty"`
	CredentialID    domain.ID      `json:"credential_id,omitempty"`
}
type SecurityAdminRequest struct {
	// HostOperator is only set by explicit offline deployment composition. It
	// authorizes the separate operator-promotion operation, never remote intents.
	HostOperator     bool
	OperatorReason   string
	Command          SecurityAdminCommand
	PrincipalID      string
	Actor            domain.ActorRef
	CredentialDigest string
	IdempotencyKey   string
	Fingerprint      string
	Now              time.Time
}
type SecurityAdminResult struct {
	CreatedTenant    *Tenant        `json:"created_tenant,omitempty"`
	TenantVersion    domain.Version `json:"tenant_version"`
	Credential       *Credential    `json:"credential,omitempty"`
	IdempotentReplay bool           `json:"idempotent_replay"`
	TokenOmitted     bool           `json:"token_omitted,omitempty"`
}
type SecurityAdministrationStore interface {
	ApplySecurityAdmin(context.Context, SecurityAdminRequest) (SecurityAdminResult, error)
	ReadSecurityAdmin(context.Context, domain.ID) (SecurityAdminSnapshot, error)
}
type SecurityAudit struct {
	TenantVersion domain.Version  `json:"tenant_version"`
	PrincipalID   string          `json:"principal_id"`
	Actor         domain.ActorRef `json:"actor_ref"`
	Operation     string          `json:"operation"`
	Target        string          `json:"target"`
	RecordedAt    time.Time       `json:"recorded_at"`
}
type SecurityAdminSnapshot struct {
	Tenant        Tenant          `json:"tenant"`
	TenantVersion domain.Version  `json:"tenant_version"`
	Grants        []TenantGrant   `json:"grants"`
	Credentials   []Credential    `json:"credentials"`
	Audit         []SecurityAudit `json:"audit"`
	Truncated     bool            `json:"truncated"`
}
