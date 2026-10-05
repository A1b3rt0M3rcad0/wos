package sqlite

import (
	"context"
	"database/sql"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func (r outcomeRepository) ListOutcomes(ctx context.Context, ns domain.ID, f ports.OutcomeFilter) ([]ports.OutcomeIndexEntry, error) {
	q := `SELECT id,version,title,description,lifecycle,priority,archived_at,created_at,updated_at FROM outcomes WHERE namespace_id=?`
	args := []any{ns.String()}
	if f.ExternalProvider != "" || f.ExternalKind != "" || f.ExternalID != "" {
		q += ` AND EXISTS(SELECT 1 FROM outcome_external_references e WHERE e.namespace_id=outcomes.namespace_id AND e.outcome_id=outcomes.id AND e.provider=? AND e.context_kind=? AND e.external_id=?)`
		args = append(args, f.ExternalProvider, f.ExternalKind, f.ExternalID)
	}
	if f.Text != "" {
		q += ` AND (instr(lower(title),lower(?))>0 OR instr(lower(description),lower(?))>0)`
		args = append(args, f.Text, f.Text)
	}
	if f.Lifecycle != "" {
		q += ` AND lifecycle=?`
		args = append(args, f.Lifecycle)
	}
	if f.Priority != "" {
		q += ` AND priority=?`
		args = append(args, string(f.Priority))
	}
	if f.Archived != nil {
		if *f.Archived {
			q += ` AND archived_at IS NOT NULL`
		} else {
			q += ` AND archived_at IS NULL`
		}
	}
	if f.Owner != nil {
		q += ` AND EXISTS(SELECT 1 FROM entity_actor_links a WHERE a.namespace_id=outcomes.namespace_id AND a.outcome_id=outcomes.id AND a.entity_id=outcomes.id AND a.role='owner' AND a.actor_kind=? AND a.actor_provider=? AND a.actor_id=?)`
		args = append(args, string(f.Owner.Kind), f.Owner.Provider, f.Owner.ID)
	}
	if !f.AfterID.IsZero() {
		q += ` AND (created_at>? OR (created_at=? AND id>?))`
		args = append(args, encodeTime(f.AfterCreated), encodeTime(f.AfterCreated), f.AfterID.String())
	}
	q += ` ORDER BY created_at,id LIMIT ?`
	args = append(args, f.Limit)
	rows, err := r.uow.tx.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	values := []ports.OutcomeIndexEntry{}
	for rows.Next() {
		var v ports.OutcomeIndexEntry
		var rawID, lifecycle, priority string
		var archived sql.NullInt64
		var created, updated int64
		err = rows.Scan(&rawID, &v.Version, &v.Title, &v.Description, &lifecycle, &priority, &archived, &created, &updated)
		if err != nil {
			return nil, err
		}
		v.ID, err = domain.ParseID(rawID)
		if err != nil {
			return nil, err
		}
		v.NamespaceID = ns
		v.Lifecycle = domain.OutcomeLifecycle(lifecycle)
		v.Priority = domain.Priority(priority)
		if archived.Valid {
			at := decodeTime(archived.Int64)
			v.ArchivedAt = &at
		}
		v.CreatedAt = decodeTime(created)
		v.UpdatedAt = decodeTime(updated)
		v.DetailOmitted = true
		values = append(values, v)
	}
	return values, rows.Err()
}

func (l eventLog) ListEvents(ctx context.Context, scope domain.Scope, f ports.EventFilter) ([]domain.DomainEvent, error) {
	q := `SELECT id,event_type,schema_version,outcome_revision,event_index,aggregate_id,aggregate_kind,aggregate_version_before,aggregate_version_after,command_id,principal_id,actor_json,recorded_at,correlation_id,causation_id,execution_context_json,payload_json FROM domain_events WHERE namespace_id=? AND outcome_id=? AND (outcome_revision>? OR (outcome_revision=? AND event_index>?))`
	args := []any{scope.NamespaceID.String(), scope.OutcomeID.String(), int64(f.AfterRevision), int64(f.AfterRevision), int64(f.AfterIndex)}
	for _, v := range []struct{ column, value string }{{"principal_id", f.PrincipalID}, {"command_id", string(f.CommandID)}, {"aggregate_id", string(f.EntityID)}, {"event_type", f.EventType}} {
		if v.value != "" {
			q += ` AND ` + v.column + `=?`
			args = append(args, v.value)
		}
	}
	q += ` ORDER BY outcome_revision,event_index LIMIT ?`
	args = append(args, f.Limit)
	rows, err := l.uow.tx.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []domain.DomainEvent{}
	for rows.Next() {
		v, err := scanEvent(rows, scope)
		if err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

var _ ports.OutcomeDiscoveryRepository = outcomeRepository{}
var _ ports.TimelineRepository = eventLog{}
