package ports

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"time"
)

type Credential struct {
	TenantID     domain.ID       `json:"tenant_id"`
	ID           domain.ID       `json:"id"`
	PrincipalID  string          `json:"principal_id"`
	Actor        domain.ActorRef `json:"actor_ref"`
	Digest       string          `json:"-"`
	ParentDigest string          `json:"-"`
	ExpiresAt    time.Time       `json:"expires_at"`
	Revoked      bool            `json:"revoked"`
}
type TenantGrant struct {
	TenantID    domain.ID    `json:"tenant_id"`
	PrincipalID string       `json:"principal_id"`
	Permissions []Permission `json:"permissions"`
}
type Tenant struct {
	ID   domain.ID `json:"id"`
	Name string    `json:"name"`
}
type SecurityStore interface {
	GetCredential(context.Context, string) (Credential, error)
	PutCredential(context.Context, Credential) error
	RevokeCredential(context.Context, domain.ID) error
	PutGrant(context.Context, TenantGrant) error
	GetGrant(context.Context, domain.ID, string) (TenantGrant, error)
	ListTenants(context.Context, string) ([]Tenant, error)
	PutTenant(context.Context, Tenant) error
}

// ScopedCredentialPrincipalReader exposes only the identity needed to protect
// operator credentials during trusted composition.
type ScopedCredentialPrincipalReader interface {
	ScopedCredentialPrincipal(context.Context, domain.ID, domain.ID) (string, error)
}
