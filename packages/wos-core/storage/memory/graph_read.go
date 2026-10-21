package memory

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func (r continuityReadRepository) GraphAdjacent(ctx context.Context, scope d.Scope, refs []d.EntityRef, direction string, kinds []string, after string, limit int) ([]ports.GraphReadEdge, error) {
	if _, e := r.Outcome(ctx, scope); e != nil {
		return nil, e
	}
	if len(refs) == 0 || len(refs) > 100 || limit < 1 || limit > 101 || (direction != "out" && direction != "in" && direction != "both") {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "invalid graph adjacency page")
	}
	frontier := map[d.EntityRef]bool{}
	for _, ref := range refs {
		if e := ref.Validate(); e != nil || ref.Scope != scope {
			return nil, d.NewError(d.ErrorCodeInvalidEntityRef, "graph references must belong to the Outcome")
		}
		frontier[ref] = true
	}
	b, e := ports.NewContinuityPageBuilder(after, limit)
	if e != nil {
		return nil, e
	}
	add := func(id string, source, target d.EntityRef, kind, truth string, strength d.DependencyStrength) error {
		if id <= after || !(direction != "in" && frontier[source] || direction != "out" && frontier[target]) {
			return nil
		}
		if len(kinds) > 0 {
			found := false
			for _, k := range kinds {
				if k == kind {
					found = true
				}
			}
			if !found {
				return nil
			}
		}
		return b.Add(id, ports.GraphReadEdge{ID: id, Source: source, Target: target, Kind: kind, Strength: strength, SourceOfTruth: truth})
	}
	outcome := d.EntityRef{Scope: scope, Kind: d.EntityKindOutcome, ID: scope.OutcomeID}
	for _, v := range r.tx.objectives {
		if v.Scope != scope {
			continue
		}
		parent := outcome
		if v.ParentObjectiveID != nil {
			parent = d.EntityRef{Scope: scope, Kind: d.EntityKindObjective, ID: *v.ParentObjectiveID}
		}
		if e = add("objective:"+v.ID.String(), parent, v.Ref(), "hierarchy", "objective", ""); e != nil {
			return nil, e
		}
	}
	for _, v := range r.tx.workItems {
		if v.Scope != scope {
			continue
		}
		owner := outcome
		if v.ObjectiveID != nil {
			owner = d.EntityRef{Scope: scope, Kind: d.EntityKindObjective, ID: *v.ObjectiveID}
		}
		if e = add("work:"+v.ID.String(), owner, v.Ref(), "ownership", "work_item", ""); e != nil {
			return nil, e
		}
	}
	for _, v := range r.tx.blockers {
		if v.Scope != scope || v.Lifecycle != d.BlockerLifecycleActive {
			continue
		}
		if e = add(v.ID.String(), v.Ref(), v.BlockedRef, "blocks", "blocker", ""); e != nil {
			return nil, e
		}
		if v.CauseRef != nil {
			if e = add(v.ID.String()+":cause", *v.CauseRef, v.Ref(), "causes", "blocker", ""); e != nil {
				return nil, e
			}
		}
	}
	for _, v := range r.tx.evidence {
		if v.Scope != scope || v.ArtifactID == nil {
			continue
		}
		if e = add(v.ID.String()+":artifact", v.Ref(), d.EntityRef{Scope: scope, Kind: d.EntityKindArtifact, ID: *v.ArtifactID}, "artifact_source", "evidence", ""); e != nil {
			return nil, e
		}
	}
	for _, v := range r.tx.decisions {
		if v.Scope != scope || v.SupersedesDecisionID == nil {
			continue
		}
		if e = add(v.ID.String()+":supersedes", v.Ref(), d.EntityRef{Scope: scope, Kind: d.EntityKindDecision, ID: *v.SupersedesDecisionID}, "supersedes", "decision", ""); e != nil {
			return nil, e
		}
	}
	for _, v := range r.tx.relations {
		if v.Scope != scope || v.Lifecycle != d.RelationLifecycleActive {
			continue
		}
		if e = add(v.ID.String(), v.SourceRef, v.TargetRef, string(v.RelationType), "relation", v.Strength); e != nil {
			return nil, e
		}
	}
	for _, v := range r.tx.evidenceLinks {
		if v.Scope != scope || v.Lifecycle != d.EvidenceLinkLifecycleActive {
			continue
		}
		if e = add(v.ID.String(), d.EntityRef{Scope: scope, Kind: d.EntityKindEvidence, ID: v.EvidenceID}, v.TargetRef, string(v.Stance), "evidence_link", ""); e != nil {
			return nil, e
		}
	}
	result := []ports.GraphReadEdge{}
	for _, item := range b.Page().Items {
		var edge ports.GraphReadEdge
		if e = json.Unmarshal(item.Value, &edge); e != nil {
			return nil, e
		}
		result = append(result, edge)
	}
	return result, nil
}
func (r continuityReadRepository) GraphNodes(ctx context.Context, scope d.Scope, refs []d.EntityRef) ([]ports.ContinuitySummary, error) {
	if _, e := r.Outcome(ctx, scope); e != nil {
		return nil, e
	}
	if len(refs) == 0 || len(refs) > 201 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "invalid graph reference batch")
	}
	result := []ports.ContinuitySummary{}
	for _, ref := range refs {
		if e := ref.Validate(); e != nil || ref.Scope != scope {
			return nil, d.NewError(d.ErrorCodeInvalidEntityRef, "graph references must belong to the Outcome")
		}
		key := entityKey(scope, ref.ID)
		add := func(version d.Version, title, life string, priority d.Priority) {
			result = append(result, ports.Summarize(ref, version, title, life, priority))
		}
		switch ref.Kind {
		case d.EntityKindOutcome:
			if ref.ID == scope.OutcomeID {
				v, e := r.Outcome(ctx, scope)
				if e != nil {
					return nil, e
				}
				result = append(result, v)
			}
		case d.EntityKindWorkItem:
			if v, ok := r.tx.workItems[key]; ok {
				add(v.Version, v.Title, string(v.Lifecycle), v.Priority)
			}
		case d.EntityKindObjective:
			if v, ok := r.tx.objectives[key]; ok {
				add(v.Version, v.Title, string(v.Lifecycle), v.Priority)
			}
		case d.EntityKindIssue:
			if v, ok := r.tx.issues[key]; ok {
				add(v.Version, v.Title, string(v.Lifecycle), "")
			}
		case d.EntityKindBlocker:
			if v, ok := r.tx.blockers[key]; ok {
				add(v.Version, v.Description, string(v.Lifecycle), "")
			}
		case d.EntityKindArtifact:
			if v, ok := r.tx.artifacts[key]; ok {
				add(v.Version, v.Name, string(v.Lifecycle), "")
			}
		case d.EntityKindEvidence:
			if v, ok := r.tx.evidence[key]; ok {
				add(v.Version, v.Description, string(v.Lifecycle), "")
			}
		case d.EntityKindDecision:
			if v, ok := r.tx.decisions[key]; ok {
				add(v.Version, v.Title, string(v.Lifecycle), "")
			}
		}
	}
	return result, nil
}

var _ ports.GraphReadRepository = continuityReadRepository{}
