package ports

import (
	"context"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// LocalContractSigner performs in-memory Ed25519 signing only. Load secrets at
// deployment composition, before transactions. This port must not call a model,
// network/KMS, subprocess, filesystem or user callback inside the UnitOfWork.
type LocalContractSigner interface {
	Identity() domain.ServerIdentity
	SignCanonical(purpose string, payload []byte) (json.RawMessage, error)
}
type ServerIdentityRepository interface {
	Server(context.Context) (domain.ServerIdentity, error)
	InsertServer(context.Context, domain.ServerIdentity) error
}
type ServerIdentityUnitOfWork interface {
	ServerIdentity() ServerIdentityRepository
}
