package memory

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sort"
)

func (r continuityReadRepository) CurrentWork(ctx context.Context, scope d.Scope, id d.ID) (ports.FocalCurrentWork, error) {
	var result ports.FocalCurrentWork
	if _, e := r.Outcome(ctx, scope); e != nil {
		return result, e
	}
	if e := id.Validate(); e != nil {
		return result, e
	}
	v, ok := r.tx.workItems[entityKey(scope, id)]
	if !ok {
		return result, d.NewError(d.ErrorCodeNotFound, "work item not found")
	}
	result.History = ports.FocalHistoryCounts{Assessments: len(v.Criteria.Assessments), Definitions: len(v.Criteria.DefinitionRevisions), Conclusions: len(v.ConclusionHistory)}
	v.Criteria.Assessments = nil
	v.Criteria.DefinitionRevisions = nil
	v.ConclusionHistory = nil
	result.Work = cloneWorkItem(v)
	return result, nil
}
func (r continuityReadRepository) CurrentObjective(ctx context.Context, scope d.Scope, id d.ID) (ports.FocalCurrentObjective, error) {
	var result ports.FocalCurrentObjective
	if _, e := r.Outcome(ctx, scope); e != nil {
		return result, e
	}
	if e := id.Validate(); e != nil {
		return result, e
	}
	v, ok := r.tx.objectives[entityKey(scope, id)]
	if !ok {
		return result, d.NewError(d.ErrorCodeNotFound, "objective not found")
	}
	result.History = ports.FocalHistoryCounts{Assessments: len(v.Criteria.Assessments), Definitions: len(v.Criteria.DefinitionRevisions), Conclusions: len(v.ConclusionHistory)}
	v.Criteria.Assessments = nil
	v.Criteria.DefinitionRevisions = nil
	v.ConclusionHistory = nil
	result.Objective = cloneObjective(v)
	return result, nil
}
func (r continuityReadRepository) FocalOutcome(ctx context.Context, scope d.Scope) (d.Outcome, error) {
	if _, e := r.Outcome(ctx, scope); e != nil {
		return d.Outcome{}, e
	}
	v := r.tx.outcomes[outcomeKey(scope.NamespaceID, scope.OutcomeID)]
	head := d.Outcome{ID: v.ID, NamespaceID: v.NamespaceID, Version: v.Version, Title: ports.Summarize(v.Ref(), v.Version, v.Title, "", "").Title, Lifecycle: v.Lifecycle, Priority: v.Priority}
	if v.ArchivedAt != nil {
		at := *v.ArchivedAt
		head.ArchivedAt = &at
	}
	return head, nil
}
func (r continuityReadRepository) FocalRecords(ctx context.Context, scope d.Scope, workID d.ID, objectiveID *d.ID) (ports.FocalContextRecords, error) {
	result := ports.FocalContextRecords{Dependencies: []d.Relation{}, DependencyEvaluations: []d.DependencyEvaluation{}, Blockers: []d.Blocker{}, ObjectiveAncestors: []d.Objective{}, Decisions: []d.Decision{}, EvidenceLinks: []d.EvidenceLink{}}
	if _, e := r.Outcome(ctx, scope); e != nil {
		return result, e
	}
	if e := workID.Validate(); e != nil {
		return result, e
	}
	workRef := d.EntityRef{Scope: scope, Kind: d.EntityKindWorkItem, ID: workID}
	for _, v := range r.tx.relations {
		if v.Scope != scope || v.SourceRef != workRef || v.Lifecycle != d.RelationLifecycleActive {
			continue
		}
		result.Dependencies = append(result.Dependencies, v)
		if v.RelationType == d.RelationTypeDependsOn {
			done := false
			switch v.TargetRef.Kind {
			case d.EntityKindWorkItem:
				target, ok := r.tx.workItems[entityKey(scope, v.TargetRef.ID)]
				if !ok {
					return result, d.NewError(d.ErrorCodeNotFound, "dependency target missing")
				}
				done = target.Lifecycle == d.WorkItemLifecycleDone
			case d.EntityKindObjective:
				target, ok := r.tx.objectives[entityKey(scope, v.TargetRef.ID)]
				if !ok {
					return result, d.NewError(d.ErrorCodeNotFound, "dependency target missing")
				}
				done = target.Lifecycle == d.ObjectiveLifecycleAchieved
			default:
				return result, d.NewError(d.ErrorCodeNotFound, "dependency target missing")
			}
			if v.Strength == d.DependencyStrengthHard && !done && len(result.DependencyEvaluations) == 0 {
				result.DependencyEvaluations = append(result.DependencyEvaluations, d.DependencyEvaluation{Relation: v, Satisfied: false})
			}
		}
	}
	sort.Slice(result.Dependencies, func(i, j int) bool { return result.Dependencies[i].ID < result.Dependencies[j].ID })
	result.DependencyCount = len(result.Dependencies)
	if len(result.Dependencies) > 101 {
		result.Dependencies = result.Dependencies[:101]
	}
	ancestors := map[d.ID]bool{}
	if objectiveID != nil {
		if e := objectiveID.Validate(); e != nil {
			return result, e
		}
		next := objectiveID
		for next != nil && !ancestors[*next] {
			v, ok := r.tx.objectives[entityKey(scope, *next)]
			if !ok {
				return result, d.NewError(d.ErrorCodeNotFound, "focal Objective ancestry missing")
			}
			ancestors[v.ID] = true
			head := d.Objective{ID: v.ID, Scope: scope}
			if v.ParentObjectiveID != nil {
				id := *v.ParentObjectiveID
				head.ParentObjectiveID = &id
			}
			result.ObjectiveAncestors = append(result.ObjectiveAncestors, head)
			next = v.ParentObjectiveID
		}
	}
	bytes := 0
	for _, v := range r.tx.blockers {
		if v.Scope != scope || v.Lifecycle != d.BlockerLifecycleActive {
			continue
		}
		applies := v.BlockedRef == workRef
		if v.Propagation == d.BlockerPropagationSubtree {
			applies = applies || v.BlockedRef.Kind == d.EntityKindOutcome && v.BlockedRef.ID == scope.OutcomeID || v.BlockedRef.Kind == d.EntityKindObjective && ancestors[v.BlockedRef.ID]
		}
		if !applies {
			continue
		}
		raw, e := json.Marshal(v)
		if e != nil {
			return result, e
		}
		bytes += len(raw)
		if bytes > 256<<10 {
			return result, d.NewError(d.ErrorCodeGraphLimitExceeded, "work context blocking proof exceeds byte limit")
		}
		result.Blockers = append(result.Blockers, cloneBlocker(v))
	}
	sort.Slice(result.Blockers, func(i, j int) bool { return result.Blockers[i].ID < result.Blockers[j].ID })
	decisions, e := ports.NewContinuityPageBuilder("", 26)
	if e != nil {
		return result, e
	}
	links, e := ports.NewContinuityPageBuilder("", 26)
	if e != nil {
		return result, e
	}
	for _, v := range r.tx.decisions {
		if v.Scope == scope && v.Lifecycle == d.DecisionLifecycleAccepted {
			if e = decisions.Add(v.ID.String(), v); e != nil {
				return result, e
			}
		}
	}
	relevant := map[d.EntityRef]bool{workRef: true, {Scope: scope, Kind: d.EntityKindOutcome, ID: scope.OutcomeID}: true}
	if objectiveID != nil {
		relevant[d.EntityRef{Scope: scope, Kind: d.EntityKindObjective, ID: *objectiveID}] = true
	}
	for _, v := range r.tx.evidenceLinks {
		if v.Scope == scope && v.Lifecycle == d.EvidenceLinkLifecycleActive && relevant[v.TargetRef] {
			if e = links.Add(v.ID.String(), v); e != nil {
				return result, e
			}
		}
	}
	page := decisions.Page()
	result.DecisionCount = page.Count
	for _, item := range page.Items {
		var v d.Decision
		if e = json.Unmarshal(item.Value, &v); e != nil {
			return result, e
		}
		result.Decisions = append(result.Decisions, v)
	}
	page = links.Page()
	result.EvidenceLinkCount = page.Count
	for _, item := range page.Items {
		var v d.EvidenceLink
		if e = json.Unmarshal(item.Value, &v); e != nil {
			return result, e
		}
		result.EvidenceLinks = append(result.EvidenceLinks, v)
	}
	return result, ctx.Err()
}

var _ ports.FocalReadRepository = continuityReadRepository{}
