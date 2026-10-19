package sqlite

import (
	"context"
	"database/sql"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func (r continuityReadRepository) WorkspacePlacementBlockers(ctx context.Context, scope d.Scope, limit int) ([]d.Blocker, error) {
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if limit < 1 || limit > d.DefaultDependencyGraphLimit+1 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "invalid structural Blocker limit")
	}
	rows, err := r.rows(ctx, `SELECT id,version,blocked_kind,blocked_id,cause_kind,cause_id,substr(description,1,512),propagation FROM blockers WHERE namespace_id=? AND outcome_id=? AND lifecycle='active' ORDER BY id LIMIT ?`, scope.NamespaceID.String(), scope.OutcomeID.String(), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []d.Blocker{}
	for rows.Next() {
		var id, kind, target, propagation string
		var causeKind, causeID sql.NullString
		item := d.Blocker{Scope: scope, Lifecycle: d.BlockerLifecycleActive}
		if err = rows.Scan(&id, &item.Version, &kind, &target, &causeKind, &causeID, &item.Description, &propagation); err != nil {
			return nil, err
		}
		item.ID, err = d.ParseID(id)
		if err != nil {
			return nil, err
		}
		targetID, err := d.ParseID(target)
		if err != nil {
			return nil, err
		}
		item.BlockedRef = d.EntityRef{Scope: scope, Kind: d.EntityKind(kind), ID: targetID}
		item.Propagation = d.BlockerPropagation(propagation)
		if causeKind.Valid && causeID.Valid {
			parsed, err := d.ParseID(causeID.String)
			if err != nil {
				return nil, err
			}
			item.CauseRef = &d.EntityRef{Scope: scope, Kind: d.EntityKind(causeKind.String), ID: parsed}
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

var _ ports.WorkspacePlacementReadRepository = continuityReadRepository{}
