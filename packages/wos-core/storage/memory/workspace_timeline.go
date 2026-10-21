package memory

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sort"
)

func (l eventLog) WorkspaceStructureChanged(ctx context.Context, scope d.Scope, after d.OutcomeRevision) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := l.tx.ensureOpen(); err != nil {
		return false, err
	}
	for _, event := range l.tx.events {
		if event.Scope() == scope && event.OutcomeRevision > after && (event.EventType == "work_item.objective_changed" || event.EventType == "objective.parent_changed") {
			return true, nil
		}
	}
	return false, nil
}
func (l eventLog) ListWorkspaceEvents(ctx context.Context, scope d.Scope, f ports.WorkspaceTimelineFilter) ([]d.DomainEvent, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := l.tx.ensureOpen(); err != nil {
		return nil, err
	}
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if f.SubjectRef.Scope != scope || f.SubjectRef.Validate() != nil || f.ThroughRevision == 0 || f.Limit < 1 || f.Limit > 21 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "invalid workspace timeline bounds")
	}
	subjects := map[d.EntityRef]bool{f.SubjectRef: true}
	if f.IncludeDescendants && f.SubjectRef.Kind == d.EntityKindObjective {
		branch := map[d.ID]bool{f.SubjectRef.ID: true}
		for changed := true; changed; {
			changed = false
			for _, o := range l.tx.objectives {
				if o.Scope == scope && o.ParentObjectiveID != nil && branch[*o.ParentObjectiveID] && !branch[o.ID] {
					branch[o.ID] = true
					changed = true
					subjects[o.Ref()] = true
				}
			}
		}
		for _, w := range l.tx.workItems {
			if w.Scope == scope && w.ObjectiveID != nil && branch[*w.ObjectiveID] {
				subjects[w.Ref()] = true
			}
		}
	}
	out := []d.DomainEvent{}
	for _, event := range l.tx.events {
		if event.Scope() != scope || event.OutcomeRevision > f.ThroughRevision || f.BeforeRevision != 0 && (event.OutcomeRevision > f.BeforeRevision || event.OutcomeRevision == f.BeforeRevision && event.EventIndex >= f.BeforeIndex) {
			continue
		}
		matches := subjects[event.AggregateRef] || f.IncludeDescendants && f.SubjectRef.Kind == d.EntityKindOutcome
		var link *d.EvidenceLink
		if event.AggregateRef.Kind == d.EntityKindEvidenceLink {
			for _, candidate := range l.tx.evidenceLinks {
				if candidate.Scope == scope && candidate.ID == event.AggregateRef.ID {
					copy := candidate
					link = &copy
					break
				}
			}
			if link != nil && subjects[link.TargetRef] {
				matches = true
			}
		}
		if !matches {
			continue
		}
		if f.CriterionID != nil {
			var payload struct {
				CriterionID       *d.ID                        `json:"criterion_id"`
				LegacyCriterionID *d.ID                        `json:"CriterionID"`
				CriterionRefs     []d.WorkActivityCriterionRef `json:"criterion_refs"`
			}
			if err := json.Unmarshal(event.Payload, &payload); err != nil {
				return nil, err
			}
			criterionMatches := payload.CriterionID != nil && *payload.CriterionID == *f.CriterionID || payload.LegacyCriterionID != nil && *payload.LegacyCriterionID == *f.CriterionID || link != nil && link.CriterionID != nil && *link.CriterionID == *f.CriterionID
			if event.EventType == "work_activity.recorded" {
				for _, ref := range payload.CriterionRefs {
					if ref.CriterionID == *f.CriterionID {
						criterionMatches = true
					}
				}
			}
			if !criterionMatches {
				continue
			}
		}
		out = append(out, event)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].OutcomeRevision == out[j].OutcomeRevision {
			return out[i].EventIndex > out[j].EventIndex
		}
		return out[i].OutcomeRevision > out[j].OutcomeRevision
	})
	return cloneEvents(out[:min(f.Limit, len(out))]), nil
}

var _ ports.WorkspaceTimelineRepository = eventLog{}
