package memory

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sort"
)

func (r continuityReadRepository) Projection(ctx context.Context, scope d.Scope, options ports.ContinuityProjectionOptions) (ports.ContinuityProjectionInputs, error) {
	data := ports.ContinuityProjectionInputs{Objectives: []d.Objective{}, WorkItems: []d.WorkItem{}, Relations: []d.Relation{}, Blockers: []d.Blocker{}, Evidence: []d.Evidence{}, ActivePlans: []ports.ContinuityActivePlan{}}
	if _, e := r.Outcome(ctx, scope); e != nil {
		return data, e
	}
	original := r.tx.outcomes[outcomeKey(scope.TenantID, scope.OutcomeID)]
	data.Outcome = d.Outcome{ID: original.ID, TenantID: original.TenantID, Version: original.Version, Title: ports.Summarize(original.Ref(), original.Version, original.Title, "", "").Title, Lifecycle: original.Lifecycle, Priority: original.Priority, ExternalContext: original.ExternalContext.Clone(), Criteria: d.NewCriterionSet()}
	if original.ArchivedAt != nil {
		at := *original.ArchivedAt
		data.Outcome.ArchivedAt = &at
	}
	if options.Conclusions {
		data.Outcome.Criteria, data.Outcome.CurrentConclusion = memoryProjectionProof(original.Criteria, original.CurrentConclusion)
	}
	if options.Plans {
		for _, slot := range r.tx.roadmapSlots {
			if slot.Scope != scope {
				continue
			}
			plan, ok := r.tx.roadmaps[entityKey(scope, slot.RoadmapID)]
			if !ok || plan.PlanScope != slot.PlanScope || slot.RevisionNumber == 0 || slot.RevisionNumber > uint64(len(plan.Revisions)) {
				continue
			}
			revision := plan.Revisions[slot.RevisionNumber-1]
			copy := d.RoadmapRevision{RevisionNumber: revision.RevisionNumber, ContentHash: revision.ContentHash, Nodes: []d.RoadmapNode{}}
			for _, node := range revision.Nodes {
				n := d.RoadmapNode{NodeKey: node.NodeKey, Title: node.Title}
				if node.TargetRef != nil {
					ref := *node.TargetRef
					n.TargetRef = &ref
				}
				if node.ReferenceSnapshot != nil {
					n.ReferenceSnapshot = &d.RoadmapReferenceSnapshot{Title: node.ReferenceSnapshot.Title}
				}
				copy.Nodes = append(copy.Nodes, n)
			}
			data.ActivePlans = append(data.ActivePlans, ports.ContinuityActivePlan{Slot: slot, Revision: copy})
		}
	}
	needHeaders := options.Operational || options.Conclusions || len(data.ActivePlans) > 0
	if needHeaders {
		for _, v := range r.tx.objectives {
			if v.Scope != scope {
				continue
			}
			head := d.Objective{ID: v.ID, Scope: scope, Version: v.Version, Title: ports.Summarize(v.Ref(), v.Version, v.Title, "", "").Title, Lifecycle: v.Lifecycle, Priority: v.Priority, RequiredForOutcome: v.RequiredForOutcome, Criteria: d.NewCriterionSet()}
			if v.ParentObjectiveID != nil {
				id := *v.ParentObjectiveID
				head.ParentObjectiveID = &id
			}
			if options.Conclusions {
				head.Criteria, head.CurrentConclusion = memoryProjectionProof(v.Criteria, v.CurrentConclusion)
			}
			data.Objectives = append(data.Objectives, head)
		}
		for _, v := range r.tx.workItems {
			if v.Scope != scope {
				continue
			}
			if !options.Operational && !options.Progress && len(data.ActivePlans) == 0 && (!options.Conclusions || v.CurrentConclusion == nil) {
				continue
			}
			head := d.WorkItem{ID: v.ID, Scope: scope, Version: v.Version, Title: ports.Summarize(v.Ref(), v.Version, v.Title, "", "").Title, Lifecycle: v.Lifecycle, Priority: v.Priority, Criteria: d.NewCriterionSet()}
			if v.ObjectiveID != nil {
				id := *v.ObjectiveID
				head.ObjectiveID = &id
			}
			if v.NotBefore != nil {
				at := *v.NotBefore
				head.NotBefore = &at
			}
			if v.CurrentLease != nil {
				lease := *v.CurrentLease
				head.CurrentLease = &lease
			}
			if options.Conclusions {
				head.Criteria, head.CurrentConclusion = memoryProjectionProof(v.Criteria, v.CurrentConclusion)
			}
			data.WorkItems = append(data.WorkItems, head)
		}
	}
	if options.Operational || len(data.ActivePlans) > 0 {
		for _, v := range r.tx.relations {
			if v.Scope == scope && v.Lifecycle == d.RelationLifecycleActive && v.RelationType == d.RelationTypeDependsOn {
				data.Relations = append(data.Relations, d.Relation{ID: v.ID, Scope: scope, SourceRef: v.SourceRef, TargetRef: v.TargetRef, RelationType: v.RelationType, Strength: v.Strength, Lifecycle: v.Lifecycle})
			}
		}
		for _, v := range r.tx.blockers {
			if v.Scope != scope || v.Lifecycle != d.BlockerLifecycleActive {
				continue
			}
			data.Blockers = append(data.Blockers, d.Blocker{ID: v.ID, Scope: scope, BlockedRef: v.BlockedRef, Propagation: v.Propagation, Lifecycle: v.Lifecycle})
		}
	}
	if options.Conclusions {
		needed := map[d.ID]bool{}
		sets := []d.CriterionSet{data.Outcome.Criteria}
		for _, v := range data.Objectives {
			sets = append(sets, v.Criteria)
		}
		for _, v := range data.WorkItems {
			sets = append(sets, v.Criteria)
		}
		for _, set := range sets {
			for _, a := range set.Assessments {
				for _, id := range a.EvidenceIDs {
					needed[id] = true
				}
			}
		}
		for id := range needed {
			if v, ok := r.tx.evidence[entityKey(scope, id)]; ok {
				data.Evidence = append(data.Evidence, d.Evidence{ID: v.ID, Scope: scope, Lifecycle: v.Lifecycle})
			}
		}
	}
	if options.Progress {
		count := func(set d.CriterionSet) {
			for _, c := range set.Items {
				if !c.Required || c.Status != d.CriterionStatusActive {
					continue
				}
				data.RequiredCriteria++
				a, ok := set.CurrentAssessments[c.ID]
				if !ok || a.CriterionRevision != c.Revision {
					continue
				}
				if a.Result == d.AssessmentResultWaived {
					data.WaivedCriteria++
					continue
				}
				if a.Result != d.AssessmentResultMet {
					continue
				}
				usable := true
				for _, id := range a.EvidenceIDs {
					v, ok := r.tx.evidence[entityKey(scope, id)]
					if !ok || v.Lifecycle != d.EvidenceLifecycleRegistered {
						usable = false
						break
					}
				}
				if usable {
					data.MetCriteria++
				}
			}
		}
		count(original.Criteria)
		for _, v := range r.tx.objectives {
			if v.Scope == scope && v.Lifecycle != d.ObjectiveLifecycleCancelled {
				count(v.Criteria)
			}
		}
		for _, v := range r.tx.workItems {
			if v.Scope == scope && v.Lifecycle != d.WorkItemLifecycleCancelled {
				count(v.Criteria)
			}
		}
	}
	sort.Slice(data.Blockers, func(i, j int) bool { return data.Blockers[i].ID < data.Blockers[j].ID })
	sort.Slice(data.Relations, func(i, j int) bool { return data.Relations[i].ID < data.Relations[j].ID })
	return data, ctx.Err()
}
func memoryProjectionProof(original d.CriterionSet, current *d.Conclusion) (d.CriterionSet, *d.Conclusion) {
	set := d.NewCriterionSet()
	if current == nil {
		return set, nil
	}
	set.Items = append(set.Items, original.Items...)
	for id, a := range original.CurrentAssessments {
		set.CurrentAssessments[id] = d.CriterionAssessment{ID: a.ID, CriterionID: a.CriterionID, CriterionRevision: a.CriterionRevision, Result: a.Result}
	}
	c := &d.Conclusion{Obligations: d.ConclusionObligations{RequiredCriteria: append([]d.CriterionObligationSnapshot{}, current.Obligations.RequiredCriteria...), RequiredObjectiveIDs: append([]d.ID{}, current.Obligations.RequiredObjectiveIDs...)}, Assessments: append([]d.CriterionAssessmentRef{}, current.Assessments...)}
	needed := map[d.ID]bool{}
	for _, ref := range c.Assessments {
		needed[ref.AssessmentID] = true
	}
	for _, a := range original.Assessments {
		if needed[a.ID] {
			set.Assessments = append(set.Assessments, d.CriterionAssessment{ID: a.ID, CriterionID: a.CriterionID, CriterionRevision: a.CriterionRevision, Result: a.Result, EvidenceIDs: append([]d.ID{}, a.EvidenceIDs...)})
		}
	}
	return set, c
}

var _ ports.ContinuityProjectionRepository = continuityReadRepository{}

func (r continuityReadRepository) BlockerDetails(ctx context.Context, scope d.Scope, ids []d.ID, byteBudget int) ([]d.Blocker, error) {
	if e := ctx.Err(); e != nil {
		return nil, e
	}
	if e := scope.Validate(); e != nil {
		return nil, e
	}
	if e := r.tx.ensureOpen(); e != nil {
		return nil, e
	}
	if byteBudget < 1 || byteBudget > 1<<20 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "invalid blocker detail byte budget")
	}
	used := 0
	result := []d.Blocker{}
	seen := map[d.ID]bool{}
	for _, id := range ids {
		if seen[id] {
			continue
		}
		if e := id.Validate(); e != nil {
			return nil, e
		}
		v, ok := r.tx.blockers[entityKey(scope, id)]
		if !ok {
			return nil, d.NewError(d.ErrorCodeNotFound, "projected blocker missing")
		}
		raw, e := json.Marshal(v)
		if e != nil {
			return nil, e
		}
		used += len(raw)
		if used > byteBudget {
			return nil, d.NewError(d.ErrorCodeGraphLimitExceeded, "projected blocker details exceed byte budget; reduce limit")
		}
		result = append(result, cloneBlocker(v))
		seen[id] = true
	}
	return result, nil
}
