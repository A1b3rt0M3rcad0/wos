package ports

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"time"
)

// Security administration coordinates on the Namespace, rather than an Outcome.
type SecurityAdminCommand struct {
	NamespaceName   string          `json:"namespace_name,omitempty"`
	NewNamespace    *Namespace      `json:"-"`
	NamespaceID     domain.ID       `json:"namespace_id"`
	ExpectedVersion domain.Version  `json:"expected_namespace_version"`
	Operation       string          `json:"operation"`
	Grant           *NamespaceGrant `json:"grant,omitempty"`
	Credential      *Credential     `json:"credential,omitempty"`
	CredentialID    domain.ID       `json:"credential_id,omitempty"`
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
	CreatedNamespace *Namespace     `json:"created_namespace,omitempty"`
	NamespaceVersion domain.Version `json:"namespace_version"`
	Credential       *Credential    `json:"credential,omitempty"`
	IdempotentReplay bool           `json:"idempotent_replay"`
	TokenOmitted     bool           `json:"token_omitted,omitempty"`
}
type SecurityAdministrationStore interface {
	ApplySecurityAdmin(context.Context, SecurityAdminRequest) (SecurityAdminResult, error)
	ReadSecurityAdmin(context.Context, domain.ID) (SecurityAdminSnapshot, error)
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
	Namespace        Namespace        `json:"namespace"`
	NamespaceVersion domain.Version   `json:"namespace_version"`
	Grants           []NamespaceGrant `json:"grants"`
	Credentials      []Credential     `json:"credentials"`
	Audit            []SecurityAudit  `json:"audit"`
	Truncated        bool             `json:"truncated"`
}
