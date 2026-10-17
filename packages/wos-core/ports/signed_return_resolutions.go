package ports

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// Calls share the signed security barrier and Outcome coordination with
// acceptance in a Serializable command transaction. A not-found snapshot alone
// never proves abandonment; immutable uniqueness remains authoritative at insert.
type SignedReturnResolutionRepository interface {
	GetByOriginalIntent(context.Context, d.ID, string, d.ID, string, string) (d.SignedReturnIntentResolution, error)
	InsertTerminal(context.Context, d.SignedReturnIntentResolution) error
}
type SignedReturnResolutionUnitOfWork interface {
	SignedReturnResolutions() SignedReturnResolutionRepository
}
