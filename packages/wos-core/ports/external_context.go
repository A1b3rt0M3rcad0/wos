package ports

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type ExternalContextRepository interface {
	List(context.Context, domain.Scope) ([]domain.ExternalReference, error)
	Add(context.Context, domain.Scope, domain.ExternalReference) error
	Remove(context.Context, domain.Scope, domain.ExternalReference) error
}
type ExternalContextUnitOfWork interface {
	ExternalContexts() ExternalContextRepository
}
