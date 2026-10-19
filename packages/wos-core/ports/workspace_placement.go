package ports

import (
	"context"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// Structural impact is an explicit, bounded query. Normal inspector and card
// reads do not load the Outcome's active Blocker graph.
type WorkspacePlacementReadRepository interface {
	WorkspacePlacementBlockers(context.Context, d.Scope, int) ([]d.Blocker, error)
}
