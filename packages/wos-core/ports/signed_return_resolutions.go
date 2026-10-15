package ports

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// Calls share the signed protocol Namespace lock then the Outcome guard with
// acceptance. A not-found read outside that transaction never proves abandonment.
type SignedReturnResolutionRepository interface {
	GetByOriginalIntent(context.Context, d.ID, string, d.ID, string, string) (d.SignedReturnIntentResolution, error)
	InsertTerminal(context.Context, d.SignedReturnIntentResolution) error
}
type SignedReturnResolutionUnitOfWork interface {
	SignedReturnResolutions() SignedReturnResolutionRepository
}
