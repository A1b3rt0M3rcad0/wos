package memory

import (
	"context"
	"sort"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func (r continuityReadRepository) WorkspacePlacementBlockers(ctx context.Context, scope d.Scope, limit int) ([]d.Blocker, error) {
	if _, err := r.Outcome(ctx, scope); err != nil {
		return nil, err
	}
	if limit < 1 || limit > d.DefaultDependencyGraphLimit+1 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "invalid structural Blocker limit")
	}
	items := []d.Blocker{}
	for _, item := range r.tx.blockers {
		if item.Scope == scope && item.Lifecycle == d.BlockerLifecycleActive {
			item.Description = ports.Summarize(item.Ref(), item.Version, item.Description, "", "").Title
			item.ExternalCause = nil
			item.ResolutionSummary = ""
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

var _ ports.WorkspacePlacementReadRepository = continuityReadRepository{}
