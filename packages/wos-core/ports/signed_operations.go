package ports

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type SignedOperationRepository interface {
	Get(context.Context, domain.ID, string, string) (domain.SignedOperationResult, error)
	FindLegacy(context.Context, domain.ID, string, string) (domain.LegacySignedOperationResult, error)
	Insert(context.Context, domain.SignedOperationResult) error
}
type SignedOperationUnitOfWork interface {
	SignedOperations() SignedOperationRepository
}
