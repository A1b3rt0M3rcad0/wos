package ports

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"time"
)

type WorkProtocolRepository interface {
	Lock(context.Context, domain.ID, bool) (domain.NamespaceWorkProtocol, error)
	Scopes(context.Context, domain.ID) ([]domain.Scope, error)
	ValidLegacyLeases(context.Context, domain.ID, time.Time) (int, error)
	ValidUnsignedContracts(context.Context, domain.ID, time.Time) (int, error)
	EnableContracts(context.Context, domain.ID, time.Time) error
	Save(context.Context, domain.NamespaceWorkProtocol, domain.Version) error
}
type WorkProtocolUnitOfWork interface{ WorkProtocol() WorkProtocolRepository }
