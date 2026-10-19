package memory

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sort"
	"strings"
)

func (r continuityReadRepository) WorkspaceOutcome(ctx context.Context, scope d.Scope) (d.Outcome, error) {
	if _, e := r.Outcome(ctx, scope); e != nil {
		return d.Outcome{}, e
	}
	v := cloneOutcome(r.tx.outcomes[outcomeKey(scope.NamespaceID, scope.OutcomeID)])
	v.Criteria.Assessments = nil
	v.Criteria.DefinitionRevisions = nil
	v.ConclusionHistory = nil
	return v, nil
}

func (r continuityReadRepository) WorkspaceRevision(ctx context.Context, scope d.Scope) (d.OutcomeRevision, error) {
	if _, e := r.Outcome(ctx, scope); e != nil {
		return 0, e
	}
	return r.tx.revisions[scopeKey(scope)], nil
}
func (r continuityReadRepository) WorkspaceObjectiveHeader(ctx context.Context, scope d.Scope, id d.ID) (d.Objective, error) {
	if e := (d.EntityRef{Scope: scope, Kind: d.EntityKindObjective, ID: id}).Validate(); e != nil {
		return d.Objective{}, e
	}
	for _, v := range r.tx.objectives {
		if v.Scope == scope && v.ID == id {
			v = cloneObjective(v)
			v.Criteria = d.NewCriterionSet()
			v.OwnerRefs = nil
			v.CurrentConclusion = nil
			v.ConclusionHistory = nil
			if len(v.Description) > 1024 {
				v.Description = v.Description[:1024]
			}
			return v, nil
		}
	}
	return d.Objective{}, d.NewError(d.ErrorCodeNotFound, "workspace Objective not found")
}

func (r continuityReadRepository) WorkspaceLastExecution(ctx context.Context, scope d.Scope, id d.ID) (*d.WorkContract, error) {
	if _, e := r.Outcome(ctx, scope); e != nil {
		return nil, e
	}
	var latest *d.WorkContract
	for _, records := range []map[string]d.WorkContract{r.tx.contracts, r.tx.signedContracts} {
		for _, v := range records {
			if v.Scope == scope && v.WorkItemID == id && (latest == nil || v.AcquiredAt.After(latest.AcquiredAt) || v.AcquiredAt.Equal(latest.AcquiredAt) && v.ID > latest.ID) {
				header := v
				header.Spec = d.WorkContractSpec{}
				latest = &header
			}
		}
	}
	return latest, nil
}

func (r continuityReadRepository) WorkspaceGuardWork(ctx context.Context, scope d.Scope, objective *d.ID) ([]d.WorkItem, error) {
	if _, e := r.Outcome(ctx, scope); e != nil {
		return nil, e
	}
	result := []d.WorkItem{}
	for _, v := range r.tx.workItems {
		if v.Scope != scope || objective != nil && (v.ObjectiveID == nil || *v.ObjectiveID != *objective) {
			continue
		}
		hasReview := v.PendingReviewCaseID != nil
		if !hasReview {
			pending, e := r.tx.SignedContracts().OpenCase(ctx, scope, v.ID)
			if e != nil {
				return nil, e
			}
			hasReview = pending != nil
		}
		if v.CurrentContractID == nil && !hasReview {
			continue
		}
		result = append(result, d.WorkItem{ID: v.ID, Scope: scope, ObjectiveID: v.ObjectiveID, CurrentContractID: v.CurrentContractID, PendingReviewCaseID: v.PendingReviewCaseID})
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}
func (r continuityReadRepository) WorkspaceRequiredObjectives(ctx context.Context, scope d.Scope) ([]d.Objective, error) {
	if _, e := r.Outcome(ctx, scope); e != nil {
		return nil, e
	}
	result := []d.Objective{}
	for _, v := range r.tx.objectives {
		if v.Scope == scope && v.RequiredForOutcome {
			result = append(result, d.Objective{ID: v.ID, Scope: scope, Lifecycle: v.Lifecycle, RequiredForOutcome: true})
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	return result, nil
}

func (r continuityReadRepository) WorkspaceRequiredObjectivePage(ctx context.Context, scope d.Scope, after string, limit int) ([]d.Objective, int, error) {
	if _, e := r.Outcome(ctx, scope); e != nil {
		return nil, 0, e
	}
	if limit < 1 || limit > 101 {
		return nil, 0, d.NewError(d.ErrorCodeInvalidArgument, "invalid required Objective page limit")
	}
	result := []d.Objective{}
	total := 0
	for _, v := range r.tx.objectives {
		if v.Scope == scope && v.RequiredForOutcome {
			total++
			if v.ID.String() > after {
				result = append(result, d.Objective{ID: v.ID, Scope: scope, Version: v.Version, Title: v.Title, Lifecycle: v.Lifecycle, Priority: v.Priority, RequiredForOutcome: true})
			}
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID < result[j].ID })
	if len(result) > limit {
		result = result[:limit]
	}
	return result, total, nil
}

func (r continuityReadRepository) WorkspaceStructure(ctx context.Context, scope d.Scope, limit int) ([]d.Objective, []d.WorkItem, error) {
	if _, e := r.Outcome(ctx, scope); e != nil {
		return nil, nil, e
	}
	if limit < 1 || limit > d.DefaultDependencyGraphLimit+1 {
		return nil, nil, d.NewError(d.ErrorCodeInvalidArgument, "invalid structural validation limit")
	}
	objectives := []d.Objective{}
	items := []d.WorkItem{}
	for _, v := range r.tx.objectives {
		if v.Scope == scope {
			objectives = append(objectives, d.Objective{ID: v.ID, Scope: scope, ParentObjectiveID: v.ParentObjectiveID, Lifecycle: v.Lifecycle})
		}
	}
	for _, v := range r.tx.workItems {
		if v.Scope == scope {
			review := v.PendingReviewCaseID
			if review == nil {
				c, e := r.tx.SignedContracts().OpenCase(ctx, scope, v.ID)
				if e != nil {
					return nil, nil, e
				}
				if c != nil {
					review = &c.ID
				}
			}
			items = append(items, d.WorkItem{ID: v.ID, Scope: scope, ObjectiveID: v.ObjectiveID, Lifecycle: v.Lifecycle, CurrentContractID: v.CurrentContractID, PendingReviewCaseID: review, CurrentLease: v.CurrentLease})
		}
	}
	sort.Slice(objectives, func(i, j int) bool { return objectives[i].ID < objectives[j].ID })
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	if len(objectives) > limit {
		objectives = objectives[:limit]
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return objectives, items, nil
}

func (r continuityReadRepository) WorkspaceRoadmap(ctx context.Context, scope d.Scope, id d.ID, revision uint64) (d.Roadmap, error) {
	if _, e := r.Outcome(ctx, scope); e != nil {
		return d.Roadmap{}, e
	}
	v, ok := r.tx.roadmaps[entityKey(scope, id)]
	if !ok {
		return d.Roadmap{}, d.NewError(d.ErrorCodeNotFound, "Roadmap not found")
	}
	if v.Draft != nil {
		draft := *v.Draft
		draft.Nodes = nil
		draft.AfterLinks = nil
		v.Draft = &draft
	}
	revisions := v.Revisions
	v.Revisions = nil
	if revision > 0 {
		for _, header := range revisions {
			if header.RevisionNumber == revision {
				header.Nodes = nil
				header.AfterLinks = nil
				header.DependencySnapshots = nil
				v.Revisions = []d.RoadmapRevision{header}
				break
			}
		}
		if len(v.Revisions) == 0 {
			return d.Roadmap{}, d.NewError(d.ErrorCodeNotFound, "Roadmap revision not found")
		}
	} else {
		for _, header := range revisions {
			header.Nodes = nil
			header.AfterLinks = nil
			header.DependencySnapshots = nil
			v.Revisions = append(v.Revisions, header)
		}
		sort.Slice(v.Revisions, func(i, j int) bool { return v.Revisions[i].RevisionNumber > v.Revisions[j].RevisionNumber })
		if len(v.Revisions) > 21 {
			v.Revisions = v.Revisions[:21]
		}
	}
	return v, nil
}
func (r continuityReadRepository) WorkspaceChildren(ctx context.Context, scope d.Scope, parent d.EntityRef, after string, limit int) ([]d.EntityRef, error) {
	if _, e := r.Outcome(ctx, scope); e != nil {
		return nil, e
	}
	if e := parent.Validate(); e != nil || parent.Scope != scope || (parent.Kind != d.EntityKindOutcome && parent.Kind != d.EntityKindObjective) || limit < 1 || limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "invalid structural page")
	}
	refs := []d.EntityRef{}
	add := func(ref d.EntityRef, p *d.ID) {
		matches := parent.Kind == d.EntityKindOutcome && p == nil || parent.Kind == d.EntityKindObjective && p != nil && *p == parent.ID
		if matches && ref.Kind.String()+":"+ref.ID.String() > after {
			refs = append(refs, ref)
		}
	}
	for _, v := range r.tx.objectives {
		if v.Scope == scope {
			add(v.Ref(), v.ParentObjectiveID)
		}
	}
	for _, v := range r.tx.workItems {
		if v.Scope == scope {
			add(v.Ref(), v.ObjectiveID)
		}
	}
	sort.Slice(refs, func(i, j int) bool {
		return strings.Compare(refs[i].Kind.String()+":"+refs[i].ID.String(), refs[j].Kind.String()+":"+refs[j].ID.String()) < 0
	})
	if len(refs) > limit {
		refs = refs[:limit]
	}
	return refs, nil
}
func (r continuityReadRepository) WorkspaceCards(ctx context.Context, scope d.Scope, refs []d.EntityRef) ([]ports.WorkspaceCardFacts, error) {
	if _, e := r.Outcome(ctx, scope); e != nil {
		return nil, e
	}
	if len(refs) < 1 || len(refs) > 100 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "invalid card batch")
	}
	result := []ports.WorkspaceCardFacts{}
	linked := map[d.EntityRef]map[d.ID]bool{}
	for _, link := range r.tx.evidenceLinks {
		if link.Scope == scope && link.Lifecycle == d.EvidenceLinkLifecycleActive && link.CriterionID != nil {
			if linked[link.TargetRef] == nil {
				linked[link.TargetRef] = map[d.ID]bool{}
			}
			linked[link.TargetRef][*link.CriterionID] = true
		}
	}
	for _, ref := range refs {
		if e := ref.Validate(); e != nil || ref.Scope != scope {
			return nil, d.NewError(d.ErrorCodeInvalidEntityRef, "card references must belong to the Outcome")
		}
		f := ports.WorkspaceCardFacts{Ref: ref, Actors: []d.ActorRef{}, Criteria: []ports.WorkspaceCriterionFact{}}
		set := d.NewCriterionSet()
		var w *d.WorkItem
		switch ref.Kind {
		case d.EntityKindOutcome:
			v := r.tx.outcomes[outcomeKey(scope.NamespaceID, ref.ID)]
			set = v.Criteria
			f.Actors = v.OwnerRefs
		case d.EntityKindObjective:
			v := r.tx.objectives[entityKey(scope, ref.ID)]
			set = v.Criteria
			f.Description = v.Description
			f.Actors = v.OwnerRefs
			f.RequiredForOutcome = v.RequiredForOutcome
			p := d.EntityRef{Scope: scope, Kind: d.EntityKindOutcome, ID: scope.OutcomeID}
			if v.ParentObjectiveID != nil {
				p.Kind = d.EntityKindObjective
				p.ID = *v.ParentObjectiveID
			}
			f.ParentRef = &p
		case d.EntityKindWorkItem:
			v := r.tx.workItems[entityKey(scope, ref.ID)]
			w = &v
			set = v.Criteria
			f.Description = v.Description
			f.Actors = v.AssigneeRefs
			f.PendingReview = v.PendingReviewCaseID != nil
			f.CorrectionRequired = v.CorrectionReviewCaseID != nil
			p := d.EntityRef{Scope: scope, Kind: d.EntityKindOutcome, ID: scope.OutcomeID}
			if v.ObjectiveID != nil {
				p.Kind = d.EntityKindObjective
				p.ID = *v.ObjectiveID
			}
			f.ParentRef = &p
		}
		for _, c := range set.Items {
			a, ok := set.CurrentAssessments[c.ID]
			valid := ok && a.CriterionRevision == c.Revision
			for _, eid := range a.EvidenceIDs {
				e, ok := r.tx.evidence[entityKey(scope, eid)]
				if !ok || e.Lifecycle != d.EvidenceLifecycleRegistered {
					valid = false
				}
			}
			if w != nil && w.ContractsEnabled {
				var sid *d.ID
				digest := ""
				if w.CurrentContractID != nil {
					contract, ok := r.tx.contracts[entityKey(scope, *w.CurrentContractID)]
					if !ok {
						contract = r.tx.signedContracts[entityKey(scope, *w.CurrentContractID)]
					}
					sid = contract.LatestSubmissionID
					if sid != nil {
						sub, ok := r.tx.submissions[entityKey(scope, *sid)]
						if !ok {
							sub = r.tx.signedSubmissions[entityKey(scope, *sid)]
						}
						digest = sub.Digest
					}
				} else if w.CurrentConclusion != nil {
					sid = w.CurrentConclusion.SubmissionID
					digest = w.CurrentConclusion.SubmissionDigest
				}
				valid = valid && sid != nil && a.SubmissionID != nil && *sid == *a.SubmissionID && digest != "" && digest == a.SubmissionDigest
			}
			f.Criteria = append(f.Criteria, ports.WorkspaceCriterionFact{ID: c.ID, Revision: c.Revision, Required: c.Required, Status: c.Status, AssessmentResult: a.Result, AssessmentApplicable: valid, EvidenceLinked: linked[ref][c.ID]})
		}
		result = append(result, f)
	}
	return result, nil
}

var _ ports.WorkspaceReadRepository = continuityReadRepository{}

func (r continuityReadRepository) WorkspaceConclusionImpact(ctx context.Context, scope d.Scope, subject d.EntityRef, limit int) (ports.WorkspaceConclusionImpactFacts, error) {
	result := ports.WorkspaceConclusionImpactFacts{Items: []ports.ContinuitySummary{}}
	if e := subject.Validate(); e != nil {
		return result, e
	}
	if subject.Scope != scope {
		return result, d.NewError(d.ErrorCodeInvalidScope, "conclusion impact subject belongs to another Outcome")
	}
	if limit < 1 || limit > 20 {
		return result, d.NewError(d.ErrorCodeInvalidArgument, "conclusion impact sample limit must be between 1 and 20")
	}
	if subject.Kind != d.EntityKindOutcome && subject.Kind != d.EntityKindObjective {
		return result, d.NewError(d.ErrorCodeInvalidArgument, "only Outcome and Objective have conclusion scope impacts")
	}
	if subject.Kind == d.EntityKindOutcome && subject.ID != scope.OutcomeID {
		return result, d.NewError(d.ErrorCodeInvalidScope, "conclusion impact Outcome does not match scope")
	}
	if _, e := r.Outcome(ctx, scope); e != nil {
		return result, e
	}
	descendants := map[d.ID]bool{}
	children := map[d.ID][]d.ID{}
	found := subject.Kind == d.EntityKindOutcome
	if subject.Kind == d.EntityKindObjective {
		for _, o := range r.tx.objectives {
			if o.Scope != scope {
				continue
			}
			if o.ID == subject.ID {
				found = true
			}
			if o.ParentObjectiveID != nil {
				children[*o.ParentObjectiveID] = append(children[*o.ParentObjectiveID], o.ID)
			}
		}
		if !found {
			return result, d.NewError(d.ErrorCodeNotFound, "conclusion impact Objective not found")
		}
		descendants[subject.ID] = true
		queue := []d.ID{subject.ID}
		for len(queue) > 0 {
			id := queue[0]
			queue = queue[1:]
			for _, child := range children[id] {
				if !descendants[child] {
					descendants[child] = true
					queue = append(queue, child)
				}
			}
		}
	}
	// Retain only the smallest `limit` keys while counting every scoped match.
	add := func(item ports.ContinuitySummary) {
		key := item.Ref.Kind.String() + ":" + item.Ref.ID.String()
		at := sort.Search(len(result.Items), func(i int) bool { return result.Items[i].Ref.Kind.String()+":"+result.Items[i].Ref.ID.String() >= key })
		if at >= limit {
			return
		}
		result.Items = append(result.Items, ports.ContinuitySummary{})
		copy(result.Items[at+1:], result.Items[at:])
		result.Items[at] = item
		if len(result.Items) > limit {
			result.Items = result.Items[:limit]
		}
	}
	for _, o := range r.tx.objectives {
		if o.Scope != scope || o.Lifecycle == d.ObjectiveLifecycleAchieved || o.Lifecycle == d.ObjectiveLifecycleCancelled || subject.Kind == d.EntityKindObjective && (o.ID == subject.ID || !descendants[o.ID]) {
			continue
		}
		result.OpenObjectives++
		add(ports.Summarize(o.Ref(), o.Version, o.Title, string(o.Lifecycle), o.Priority))
	}
	for _, w := range r.tx.workItems {
		if w.Scope != scope || w.Lifecycle == d.WorkItemLifecycleDone || w.Lifecycle == d.WorkItemLifecycleCancelled || subject.Kind == d.EntityKindObjective && (w.ObjectiveID == nil || !descendants[*w.ObjectiveID]) {
			continue
		}
		result.OpenWorkItems++
		add(ports.Summarize(w.Ref(), w.Version, w.Title, string(w.Lifecycle), w.Priority))
	}
	return result, nil
}
