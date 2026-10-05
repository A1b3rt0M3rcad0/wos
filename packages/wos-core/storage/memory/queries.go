package memory

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sort"
	"strings"
)

func (r outcomeRepository) ListOutcomes(ctx context.Context, ns domain.ID, f ports.OutcomeFilter) ([]ports.OutcomeIndexEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	values := []ports.OutcomeIndexEntry{}
	for _, v := range r.tx.outcomes {
		if v.NamespaceID != ns {
			continue
		}
		if f.ExternalProvider != "" || f.ExternalKind != "" || f.ExternalID != "" {
			found := false
			for _, ref := range r.tx.externalContexts[contextKey(v.Scope())] {
				if ref.Provider == f.ExternalProvider && ref.Kind == f.ExternalKind && ref.ExternalID == f.ExternalID {
					found = true
				}
			}
			if !found {
				continue
			}
		}
		if f.Text != "" && !strings.Contains(strings.ToLower(v.Title+"\n"+v.Description), strings.ToLower(f.Text)) {
			continue
		}
		if f.Lifecycle != "" && string(v.Lifecycle) != f.Lifecycle {
			continue
		}
		if f.Priority != "" && v.Priority != f.Priority {
			continue
		}
		if f.Archived != nil && v.IsArchived() != *f.Archived {
			continue
		}
		if !f.AfterID.IsZero() && (v.CreatedAt.Before(f.AfterCreated) || (v.CreatedAt.Equal(f.AfterCreated) && v.ID <= f.AfterID)) {
			continue
		}
		if f.Owner != nil {
			found := false
			for _, actor := range v.OwnerRefs {
				if actor.Kind == f.Owner.Kind && actor.Provider == f.Owner.Provider && actor.ID == f.Owner.ID {
					found = true
				}
			}
			if !found {
				continue
			}
		}
		values = append(values, ports.OutcomeIndexEntry{ID: v.ID, NamespaceID: ns, Version: v.Version, Title: v.Title, Description: v.Description, Lifecycle: v.Lifecycle, Priority: v.Priority, ArchivedAt: cloneOutcome(v).ArchivedAt, CreatedAt: v.CreatedAt, UpdatedAt: v.UpdatedAt, DetailOmitted: true})
	}
	sort.Slice(values, func(i, j int) bool {
		if values[i].CreatedAt.Equal(values[j].CreatedAt) {
			return values[i].ID < values[j].ID
		}
		return values[i].CreatedAt.Before(values[j].CreatedAt)
	})
	if len(values) > f.Limit {
		values = values[:f.Limit]
	}
	return values, nil
}
func (l eventLog) ListEvents(ctx context.Context, scope domain.Scope, f ports.EventFilter) ([]domain.DomainEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := []domain.DomainEvent{}
	for _, v := range cloneEvents(l.tx.events) {
		if v.Scope() != scope || v.OutcomeRevision < f.AfterRevision || (v.OutcomeRevision == f.AfterRevision && v.EventIndex <= f.AfterIndex) {
			continue
		}
		if f.PrincipalID != "" && v.PrincipalID != f.PrincipalID {
			continue
		}
		if !f.CommandID.IsZero() && v.CommandID != f.CommandID {
			continue
		}
		if !f.EntityID.IsZero() && v.AggregateRef.ID != f.EntityID {
			continue
		}
		if f.EventType != "" && v.EventType != f.EventType {
			continue
		}
		out = append(out, v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].OutcomeRevision == out[j].OutcomeRevision {
			return out[i].EventIndex < out[j].EventIndex
		}
		return out[i].OutcomeRevision < out[j].OutcomeRevision
	})
	if len(out) > f.Limit {
		out = out[:f.Limit]
	}
	return out, nil
}
