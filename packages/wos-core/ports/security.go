package ports

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"time"
)

type Credential struct {
	NamespaceID domain.ID       `json:"namespace_id"`
	ID          domain.ID       `json:"id"`
	PrincipalID string          `json:"principal_id"`
	Actor       domain.ActorRef `json:"actor_ref"`
	Digest      string          `json:"-"`
	ExpiresAt   time.Time       `json:"expires_at"`
	Revoked     bool            `json:"revoked"`
}
type NamespaceGrant struct {
	NamespaceID domain.ID    `json:"namespace_id"`
	PrincipalID string       `json:"principal_id"`
	Permissions []Permission `json:"permissions"`
}
type Namespace struct {
	ID   domain.ID `json:"id"`
	Name string    `json:"name"`
}
type SecurityStore interface {
	GetCredential(context.Context, string) (Credential, error)
	PutCredential(context.Context, Credential) error
	RevokeCredential(context.Context, domain.ID) error
	PutGrant(context.Context, NamespaceGrant) error
	GetGrant(context.Context, domain.ID, string) (NamespaceGrant, error)
	ListNamespaces(context.Context, string) ([]Namespace, error)
	PutNamespace(context.Context, Namespace) error
}
