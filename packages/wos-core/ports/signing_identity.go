package ports

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"time"
)

// Signing identities belong to Namespace security, not a fabricated Outcome.
type SigningIdentityRepository interface {
	LockNamespace(context.Context, d.ID) (d.Version, error)
	Credential(context.Context, d.ID, d.ID) (Credential, error)
	CredentialByDigest(context.Context, string) (Credential, error)
	Key(context.Context, d.ID, d.ID) (d.SigningKey, error)
	Keys(context.Context, d.ID, string, int) ([]d.SigningKey, error)
	SaveKey(context.Context, d.SigningKey, d.Version) error
	Enrollment(context.Context, d.ID, d.ID) (d.SigningEnrollment, error)
	SaveEnrollment(context.Context, d.SigningEnrollment, d.Version) error
	CredentialPolicy(context.Context, d.ID, d.ID) (d.CredentialPolicy, error)
	SaveCredentialPolicy(context.Context, d.CredentialPolicy, d.Version) error
	AcceptancePolicy(context.Context, d.ID, *d.ID, *d.ID) (d.WorkAcceptancePolicy, error)
	SaveAcceptancePolicy(context.Context, d.WorkAcceptancePolicy, d.Version) error
	PrincipalGroup(context.Context, d.ID, string) (string, error)
	SetPrincipalGroup(context.Context, d.ID, string, string) error
	Receipt(context.Context, d.ID, string, string) (string, json.RawMessage, error)
	SaveReceipt(context.Context, d.ID, string, string, string, json.RawMessage) error
	Audit(context.Context, d.ID, string, d.ActorRef, string, string, time.Time) (d.Version, error)
}
type SigningIdentityUnitOfWork interface {
	SigningIdentity() SigningIdentityRepository
}

// CredentialPolicyReader is a best-effort preflight reader; mutations still
// revalidate against SigningIdentity inside the coherent UnitOfWork.
type CredentialPolicyReader interface {
	CredentialPolicy(context.Context, d.ID, d.ID) (d.CredentialPolicy, error)
}
