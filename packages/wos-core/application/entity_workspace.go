package application

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sort"
)

func (v WorkspaceSubject) Validate(scope d.Scope) error {
	if e := v.Ref.Validate(); e != nil || v.Ref.Scope != scope {
		return d.NewError(d.ErrorCodeInvalidEntityRef, "workspace subject must belong to the Outcome")
	}
	switch v.Kind {
	case "entity":
		if v.CriterionID != nil || v.RevisionNumber != 0 || v.RecordID != nil || v.RecordKind != "" {
			return d.NewError(d.ErrorCodeInvalidArgument, "entity subject has incompatible selectors")
		}
	case "criterion":
		if v.CriterionID == nil || v.CriterionID.Validate() != nil || v.RevisionNumber != 0 || v.RecordID != nil || v.RecordKind != "" || (v.Ref.Kind != d.EntityKindOutcome && v.Ref.Kind != d.EntityKindObjective && v.Ref.Kind != d.EntityKindWorkItem) {
			return d.NewError(d.ErrorCodeInvalidArgument, "criterion subject requires its real owner and criterion_id")
		}
	case "roadmap_revision":
		if v.Ref.Kind != d.EntityKindRoadmap || v.RevisionNumber == 0 || v.CriterionID != nil || v.RecordID != nil || v.RecordKind != "" {
			return d.NewError(d.ErrorCodeInvalidArgument, "roadmap revision requires a Roadmap and revision_number")
		}
	case "execution_record":
		if v.Ref.Kind != d.EntityKindWorkItem || v.RecordID == nil || v.RecordID.Validate() != nil || v.CriterionID != nil || v.RevisionNumber != 0 || (v.RecordKind != "contract" && v.RecordKind != "submission" && v.RecordKind != "review_case") {
			return d.NewError(d.ErrorCodeInvalidArgument, "execution record requires Task, record_kind and record_id")
		}
	default:
		return d.NewError(d.ErrorCodeInvalidArgument, "unknown workspace subject kind")
	}
	return nil
}

func (s *Service) GetEntityWorkspaceView(ctx context.Context, scope d.Scope, q WorkspaceViewQuery) (EntityWorkspaceView, error) {
	zero := EntityWorkspaceView{}
	if e := q.Subject.Validate(scope); e != nil {
		return zero, e
	}
	limit := q.Limit
	if limit == 0 {
		limit = 20
	}
	if limit < 1 || limit > 100 {
		return zero, d.NewError(d.ErrorCodeInvalidArgument, "workspace facet limit must be between 1 and 100")
	}
	frame, handled, e := s.openWorkspaceFrame(ctx, scope)
	if e != nil {
		return zero, e
	}
	if !handled {
		return zero, d.NewError(d.ErrorCodeInvalidConfig, "coherent workspace reads are unavailable")
	}
	defer frame.u.Rollback()
	focal, ok := frame.repo.(ports.FocalReadRepository)
	if !ok {
		return zero, d.NewError(d.ErrorCodeInvalidConfig, "focal workspace reads are unavailable")
	}
	repo, ok := frame.repo.(ports.WorkspaceReadRepository)
	if !ok {
		return zero, d.NewError(d.ErrorCodeInvalidConfig, "workspace reads are unavailable")
	}
	outcome, e := focal.FocalOutcome(ctx, scope)
	if e == nil && q.Subject.Ref.Kind == d.EntityKindOutcome {
		outcome, e = repo.WorkspaceOutcome(ctx, scope)
	}
	if e != nil {
		return zero, e
	}
	outHead := summary(outcome.Ref(), outcome.Version, outcome.Title, string(outcome.Lifecycle), outcome.Priority)
	v := EntityWorkspaceView{Subject: q.Subject, Rules: CanonicalWorkspaceRules(), OutcomeRevision: frame.revision, EvaluatedAt: frame.at, Placement: WorkspacePlacement{Outcome: outHead, Path: []EntitySummary{outHead}}, People: []WorkspacePerson{}, Relationships: WorkspaceRelationships{Items: []WorkspaceRelationship{}, Nodes: []EntitySummary{}}, Capabilities: []WorkspaceCapability{}, Coverage: map[string]WorkspaceCoverage{}}
	for _, key := range []string{"work", "criteria", "delivery", "attention"} {
		v.Coverage[key] = WorkspaceCoverage{Status: "not_applicable"}
	}
	v.Coverage["placement_preview"] = WorkspaceCoverage{Status: "not_loaded"}
	v.Coverage["blocker_preview"] = WorkspaceCoverage{Status: "not_loaded"}
	ref := q.Subject.Ref
	var selected any
	set := d.NewCriterionSet()
	var workContext *WorkContext
	var owners []d.ActorRef
	var parentID *d.ID
	switch ref.Kind {
	case d.EntityKindOutcome:
		selected = outcome
		set = outcome.Criteria
		owners = outcome.OwnerRefs
		v.EntityVersion = outcome.Version
		v.Header = WorkspaceHeader{Title: outcome.Title, Kind: ref.Kind.String(), Lifecycle: string(outcome.Lifecycle), Description: workspaceText(outcome.DesiredState, 1024)}
	case d.EntityKindObjective:
		current, e := focal.CurrentObjective(ctx, scope, ref.ID)
		if e != nil {
			return zero, e
		}
		o := current.Objective
		selected = o
		set = o.Criteria
		owners = o.OwnerRefs
		parentID = o.ParentObjectiveID
		v.EntityVersion = o.Version
		v.Header = WorkspaceHeader{Title: o.Title, Kind: ref.Kind.String(), Lifecycle: string(o.Lifecycle), Description: workspaceText(o.Description, 1024)}
	case d.EntityKindWorkItem:
		c, e := s.currentFocalWorkContextAt(ctx, frame.u, focal, scope, ref.ID, frame.revision, frame.at)
		if e != nil {
			return zero, e
		}
		workContext = &c
		w := c.Work
		selected = w
		set = w.Criteria
		owners = w.AssigneeRefs
		parentID = w.ObjectiveID
		v.EntityVersion = w.Version
		attention := projectWorkAttention(c)
		v.Attention = &attention
		v.Header = WorkspaceHeader{Title: w.Title, Kind: ref.Kind.String(), Lifecycle: string(w.Lifecycle), DisplayState: attention.Status, Description: workspaceText(w.Description, 1024)}
		v.Work = workspaceExecutionPreview(w.ExecutionSpec)
		v.Coverage["work"] = WorkspaceCoverage{Status: "complete"}
		if executionPreviewOmitted(w.ExecutionSpec) {
			v.Coverage["work"] = WorkspaceCoverage{Status: "partial", Reason: "execution_spec_preview; read the canonical Task for the full specification"}
		}
		v.Coverage["attention"] = WorkspaceCoverage{Status: "complete"}
		v.Coverage["delivery"] = WorkspaceCoverage{Status: "complete", ReturnedCount: workspaceCount(0)}
		if attention.Truncated {
			v.Coverage["attention"] = WorkspaceCoverage{Status: "partial", Reason: "focal_context_limit"}
		}
		if c.Contract != nil {
			contract := c.Contract.Contract
			state := "expired"
			if c.Contract.ExecutionAllowed {
				state = "current_authority"
			} else if contract.ValidAt(frame.at) {
				state = "authority_suspended"
			}
			actor := contract.Actor
			at := contract.AcquiredAt
			v.People = append(v.People, WorkspacePerson{Role: "executor", Actor: &actor, PrincipalID: contract.HolderPrincipalID, Source: "work_contract", AuthorityState: state, ContractID: &contract.ID, At: &at})
			v.Delivery = &WorkspaceDelivery{ContractID: &contract.ID, Status: string(c.Contract.EffectiveStatus), Current: true}
			if sub := c.Contract.LatestSubmission; sub != nil {
				v.Delivery.SubmissionID = &sub.ID
				v.Delivery.SubmissionDigest = sub.Digest
				v.Delivery.Summary = workspaceText(sub.Material.Summary, 1024)
			}
		} else if w.CurrentLease != nil {
			actor := w.CurrentLease.Actor
			state := "expired"
			if w.CurrentLease.ValidAt(frame.at) {
				state = "current_authority"
			}
			at := w.CurrentLease.AcquiredAt
			v.People = append(v.People, WorkspacePerson{Role: "executor", Actor: &actor, PrincipalID: w.CurrentLease.PrincipalID, Source: "legacy_work_lease", AuthorityState: state, At: &at})
		}
		if w.PendingReviewCaseID != nil {
			if v.Delivery == nil {
				v.Delivery = &WorkspaceDelivery{Current: true}
			}
			v.Delivery.ReviewCaseID = w.PendingReviewCaseID
			v.Delivery.Status = "awaiting_review"
			if signed, ok := frame.u.(ports.SignedContractUnitOfWork); ok {
				review, e := signed.SignedContracts().Case(ctx, scope, *w.PendingReviewCaseID)
				if e != nil {
					return zero, e
				}
				v.Delivery.SubmissionID = &review.SubmissionID
				v.Delivery.SubmissionDigest = review.SubmissionDigest
				v.Delivery.ContractID = &review.WorkContractID
			}
		}
		if c.Contract == nil && w.CurrentLease == nil {
			last, e := repo.WorkspaceLastExecution(ctx, scope, w.ID)
			if e != nil {
				return zero, e
			}
			if last != nil {
				actor := last.Actor
				at := last.AcquiredAt
				v.People = append(v.People, WorkspacePerson{Role: "last_executor", Actor: &actor, PrincipalID: last.HolderPrincipalID, Source: "work_contract_history", AuthorityState: "historical", ContractID: &last.ID, At: &at})
			}
		}
		if v.Delivery != nil {
			v.Coverage["delivery"] = WorkspaceCoverage{Status: "complete"}
		}
	case d.EntityKindRoadmap:
		selected, e = repo.WorkspaceRoadmap(ctx, scope, ref.ID, q.Subject.RevisionNumber)
		if e != nil {
			return zero, e
		}
		v.Header, v.EntityVersion, owners = workspaceDocumentHeader(selected, ref)
		plan := selected.(d.Roadmap)
		v.Roadmap = &WorkspaceRoadmapFacet{Draft: plan.Draft, Revisions: append([]d.RoadmapRevision{}, plan.Revisions[:min(20, len(plan.Revisions))]...), NodesStatus: "not_loaded"}
		if q.Subject.Kind == "roadmap_revision" && len(plan.Revisions) > 0 {
			revision := plan.Revisions[0]
			v.Roadmap.SelectedRevision = &revision
		}
		if p, ok := frame.u.(ports.PlanningUnitOfWork); ok {
			slot, e := p.RoadmapActivations().GetActive(ctx, scope, plan.PlanScope)
			if e != nil {
				return zero, e
			}
			if slot != nil && slot.RoadmapID == plan.ID {
				number := slot.RevisionNumber
				v.Roadmap.ActiveRevisionNumber = &number
				v.Roadmap.IsActive = true
			}
		}
		v.Coverage["roadmap"] = WorkspaceCoverage{Status: "partial", ReturnedCount: workspaceCount(len(v.Roadmap.Revisions)), Reason: "Bounded revision headers; open the canonical Roadmap to expand plan content"}
	default:
		selected, e = workspaceDocument(ctx, frame.u, ref)
		if e != nil {
			return zero, e
		}
		v.Header, v.EntityVersion, owners = workspaceDocumentHeader(selected, ref)
	}
	if evidence, ok := selected.(d.Evidence); ok {
		v.Observation, e = workspaceObservationSource(ctx, frame.u, scope, evidence.SourceRef)
		if e != nil {
			return zero, e
		}
	}
	if ref.Kind != d.EntityKindOutcome {
		p := outHead
		if parentID != nil {
			var parent d.Objective
			if workContext != nil && workContext.Objective != nil && workContext.Objective.ID == *parentID {
				parent = *workContext.Objective
			} else {
				parent, e = repo.WorkspaceObjectiveHeader(ctx, scope, *parentID)
				if e != nil {
					return zero, e
				}
			}
			p = summary(parent.Ref(), parent.Version, parent.Title, string(parent.Lifecycle), parent.Priority)
		}
		v.Placement.Parent = &p
		if parentID != nil {
			v.Placement.Path = append(v.Placement.Path, p)
			v.Coverage["placement"] = WorkspaceCoverage{Status: "partial", Reason: "Outcome and direct parent; ancestor path is not expanded"}
		} else {
			v.Coverage["placement"] = WorkspaceCoverage{Status: "complete"}
		}
	} else {
		v.Coverage["placement"] = WorkspaceCoverage{Status: "complete"}
	}
	for _, owner := range owners {
		actor := owner
		role := "declared_owner"
		source := "owner_refs"
		if ref.Kind == d.EntityKindWorkItem {
			role = "declared_assignee"
			source = "assignee_refs"
		} else if ref.Kind != d.EntityKindOutcome && ref.Kind != d.EntityKindObjective {
			role = "author"
			source = "recorded_authorship"
		}
		v.People = append(v.People, WorkspacePerson{Role: role, Actor: &actor, Source: source})
	}
	v.Coverage["people"] = WorkspaceCoverage{Status: "partial", ReturnedCount: workspaceCount(len(v.People)), Reason: "Declared responsibility and current recorded authority; no online presence is inferred"}
	ownerHeader := v.Header
	if q.Subject.Kind != "entity" {
		v.Placement.Path = append(v.Placement.Path, summary(ref, v.EntityVersion, ownerHeader.Title, ownerHeader.Lifecycle, ""))
	}
	if q.Subject.Kind == "criterion" {
		criterion, e := set.Find(*q.Subject.CriterionID)
		if e != nil {
			return zero, e
		}
		v.Header = WorkspaceHeader{Title: criterion.Title, Kind: "criterion", Lifecycle: string(criterion.Status), Description: workspaceText(criterion.Description, 1024)}
		selected = *criterion
	}
	if q.Subject.Kind == "roadmap_revision" {
		roadmap := selected.(d.Roadmap)
		found := false
		for _, revision := range roadmap.Revisions {
			if revision.RevisionNumber == q.Subject.RevisionNumber {
				selected = revision
				v.Header.Kind = "roadmap_revision"
				found = true
				break
			}
		}
		if !found {
			return zero, d.NewError(d.ErrorCodeNotFound, "Roadmap revision not found")
		}
	}
	if q.Subject.Kind == "execution_record" {
		selected, v.Delivery, e = workspaceExecutionRecord(ctx, frame.u, scope, q.Subject, frame.at)
		if e != nil {
			return zero, e
		}
		v.Header.Kind = q.Subject.RecordKind
		v.Header.Title = q.Subject.RecordKind + " / " + q.Subject.RecordID.String()
		v.Coverage["delivery"] = WorkspaceCoverage{Status: "complete"}
		v.Work = nil
		v.Attention = nil
		v.Coverage["work"] = WorkspaceCoverage{Status: "not_applicable"}
		v.Coverage["attention"] = WorkspaceCoverage{Status: "not_applicable"}
	}
	if ref.Kind == d.EntityKindOutcome && q.Subject.Kind == "entity" {
		if e = s.workspaceRequiredObjectives(ctx, frame, &v, q, limit); e != nil {
			return zero, e
		}
	} else {
		v.Coverage["required_objectives"] = WorkspaceCoverage{Status: "not_applicable"}
	}
	if q.Subject.Kind == "entity" && (ref.Kind == d.EntityKindOutcome || ref.Kind == d.EntityKindObjective) {
		if e = s.workspaceConclusionImpact(ctx, frame, &v, min(limit, 20)); e != nil {
			return zero, e
		}
	} else {
		v.Coverage["conclusion_impact"] = WorkspaceCoverage{Status: "not_applicable"}
	}
	v.Entity = workspaceRecordPreview(selected)
	v.Coverage["entity"] = WorkspaceCoverage{Status: "partial", Reason: "Selected record preview; load the canonical record before editing"}
	if q.Subject.Kind != "execution_record" && (ref.Kind == d.EntityKindOutcome || ref.Kind == d.EntityKindObjective || ref.Kind == d.EntityKindWorkItem) {
		if e = s.workspaceCriteria(ctx, frame, &v, set, workContext, q, limit); e != nil {
			return zero, e
		}
	}
	if e = s.workspaceRelationships(ctx, frame, &v, q, limit); e != nil {
		return zero, e
	}
	if ref.Kind == d.EntityKindRoadmap {
		v.Coverage["relationships"] = WorkspaceCoverage{Status: "not_loaded", Reason: "Plan nodes and historical snapshots are read from the canonical Roadmap revision resource"}
	}
	if e = s.workspaceSelectedRelationship(ctx, frame, &v, selected); e != nil {
		return zero, e
	}
	if issue, ok := selected.(d.Issue); ok {
		refs := []d.EntityRef{}
		known := map[d.EntityRef]bool{}
		for _, node := range v.Relationships.Nodes {
			known[node.Ref] = true
		}
		for _, target := range issue.AffectedRefs[:min(limit, len(issue.AffectedRefs))] {
			v.Relationships.Items = append(v.Relationships.Items, WorkspaceRelationship{GraphEdge: GraphEdge{ID: issue.ID.String() + ":affected:" + target.ID.String(), Source: issue.Ref(), Target: target, Kind: "affected", SourceOfTruth: "issue"}})
			if !known[target] {
				known[target] = true
				refs = append(refs, target)
			}
		}
		if len(refs) > 0 {
			if graph, ok := frame.repo.(ports.GraphReadRepository); ok {
				nodes, e := graph.GraphNodes(ctx, scope, refs)
				if e != nil {
					return zero, e
				}
				for _, node := range nodes {
					v.Relationships.Nodes = append(v.Relationships.Nodes, EntitySummary(node))
				}
			}
		}
		coverage := v.Coverage["relationships"]
		coverage.ReturnedCount = workspaceCount(len(v.Relationships.Items))
		if len(issue.AffectedRefs) > limit {
			coverage.Status = "partial"
			coverage.Reason = "affected_records_preview; open the canonical Issue for all affected records"
		}
		v.Coverage["relationships"] = coverage
	}
	activity, e := s.workspaceTimelineInUnitOfWork(ctx, frame.u, scope, WorkspaceTimelineQuery{SubjectRef: &ref, CriterionID: q.Subject.CriterionID, Limit: 5}, frame.revision, frame.at)
	if e != nil {
		return zero, e
	}
	v.Activity = &activity
	if len(activity.Items) > 0 {
		entry := activity.Items[0]
		actor := entry.Actor
		at := entry.RecordedAt
		v.People = append(v.People, WorkspacePerson{Role: "last_activity_author", Actor: &actor, PrincipalID: entry.PrincipalID, Source: entry.Origin, At: &at})
		peopleCoverage := v.Coverage["people"]
		peopleCoverage.ReturnedCount = workspaceCount(len(v.People))
		v.Coverage["people"] = peopleCoverage
	}
	v.Coverage["activity"] = WorkspaceCoverage{Status: "complete", ReturnedCount: workspaceCount(len(activity.Items)), NextCursor: activity.NextCursor}
	if activity.NextCursor != "" {
		v.Coverage["activity"] = WorkspaceCoverage{Status: "partial", ReturnedCount: workspaceCount(len(activity.Items)), NextCursor: activity.NextCursor, Reason: "recent_activity_page"}
	}
	v.Capabilities, e = s.workspaceCapabilities(ctx, frame, &v, selected, workContext, outcome)
	if e != nil {
		return zero, e
	}
	v.Coverage["capabilities"] = WorkspaceCoverage{Status: "complete", ReturnedCount: workspaceCount(len(v.Capabilities))}
	if q.PlacementPreview != nil {
		v.PlacementPreview, e = s.workspacePlacementPreview(ctx, frame, &v, q.PlacementPreview)
		if e != nil {
			return zero, e
		}
	}
	if q.BlockerPreview != nil {
		v.BlockerPreview, e = s.workspaceBlockerPreview(ctx, frame, &v, selected, outcome, q.BlockerPreview)
		if e != nil {
			return zero, e
		}
	}
	raw, e := json.Marshal(v)
	if e != nil {
		return zero, e
	}
	if len(raw) > MaxSnapshotBytes {
		return zero, d.NewError(d.ErrorCodeGraphLimitExceeded, "workspace view exceeds byte limit; reduce the facet limit")
	}
	return v, nil
}

func workspaceExecutionPreview(spec *d.ExecutionSpec) *d.ExecutionSpec {
	if spec == nil {
		return nil
	}
	v := *spec
	clip := func(items []string) []string {
		result := []string{}
		for _, item := range items[:min(len(items), 10)] {
			result = append(result, workspaceText(item, 1024))
		}
		return result
	}
	v.Instructions = clip(v.Instructions)
	v.Constraints = clip(v.Constraints)
	v.Deliverables = clip(v.Deliverables)
	v.ScopeHints = clip(v.ScopeHints)
	v.ContextRefs = append([]d.EntityRef{}, v.ContextRefs[:min(len(v.ContextRefs), 10)]...)
	return &v
}
func executionPreviewOmitted(spec *d.ExecutionSpec) bool {
	if spec == nil {
		return false
	}
	for _, list := range [][]string{spec.Instructions, spec.Constraints, spec.Deliverables, spec.ScopeHints} {
		if len(list) > 10 {
			return true
		}
		for _, s := range list {
			if len(s) > 1024 {
				return true
			}
		}
	}
	return len(spec.ContextRefs) > 10
}
func workspaceRecordPreview(v any) any {
	raw, _ := marshalWorkspaceProjection(v)
	var record map[string]json.RawMessage
	if json.Unmarshal(raw, &record) != nil {
		return v
	}
	for _, key := range []string{"criteria", "conclusion_history", "execution_spec", "signed_binding", "spec", "material", "payload", "proof"} {
		delete(record, key)
	}
	if raw, ok := record["conclusion"]; ok {
		var c map[string]json.RawMessage
		if json.Unmarshal(raw, &c) == nil {
			var reason string
			if json.Unmarshal(c["reason"], &reason) == nil {
				c["reason"], _ = json.Marshal(workspaceText(reason, 4096))
			}
			record["conclusion"], _ = json.Marshal(c)
		}
	}
	for key, value := range record {
		var text string
		if json.Unmarshal(value, &text) == nil && len(text) > 4096 {
			record[key], _ = json.Marshal(workspaceText(text, 4096))
		}
	}
	return record
}

func (s *Service) workspaceCriteria(ctx context.Context, f *continuityReadFrame, v *EntityWorkspaceView, set d.CriterionSet, work *WorkContext, q WorkspaceViewQuery, limit int) error {
	items := append([]d.SuccessCriterion{}, set.Items...)
	sort.Slice(items, func(i, j int) bool { return items[i].ID < items[j].ID })
	if q.Subject.Kind == "criterion" {
		c, e := set.Find(*q.Subject.CriterionID)
		if e != nil {
			return e
		}
		items = []d.SuccessCriterion{*c}
	}
	after := ""
	hash := workspaceCursorFilter(ctx, q.Subject)
	if q.CriteriaCursor != "" {
		c, e := decodeCursor(q.CriteriaCursor, v.Subject.Ref.NamespaceID, v.Subject.Ref.OutcomeID, "workspace_criteria", hash)
		if e != nil {
			return e
		}
		if c.Revision != f.revision {
			return d.NewError(d.ErrorCodePreconditionFailed, "snapshot_changed")
		}
		after = c.Key
	}
	total := len(items)
	start := sort.Search(len(items), func(i int) bool { return items[i].ID.String() > after })
	end := min(start+limit, len(items))
	v.Criteria = []WorkspaceCriterion{}
	facts := map[d.ID]ports.WorkspaceCriterionFact{}
	if work != nil {
		for _, fact := range work.workspaceCriterionFacts {
			facts[fact.ID] = fact
		}
	} else if repo, ok := f.repo.(ports.WorkspaceReadRepository); ok {
		batch, e := repo.WorkspaceCards(ctx, q.Subject.Ref.Scope, []d.EntityRef{q.Subject.Ref})
		if e != nil {
			return e
		}
		for _, owner := range batch {
			for _, fact := range owner.Criteria {
				facts[fact.ID] = fact
			}
		}
	}
	for _, criterion := range items[start:end] {
		criterion.Description = workspaceText(criterion.Description, 2048)
		c := WorkspaceCriterion{Definition: criterion}
		if a, ok := set.CurrentAssessments[criterion.ID]; ok {
			applies := criterion.Status == d.CriterionStatusActive && a.CriterionRevision == criterion.Revision
			if fact, ok := facts[criterion.ID]; ok {
				applies = applies && fact.AssessmentApplicable
			} else {
				applies = false
			}
			a.Rationale = workspaceText(a.Rationale, 2048)
			c.Assessment = &a
			c.AssessmentApplicable = applies
		}
		if fact, ok := facts[criterion.ID]; ok {
			c.EvidenceLinked = fact.EvidenceLinked
		}
		v.Criteria = append(v.Criteria, c)
	}
	coverage := WorkspaceCoverage{Status: "complete", ReturnedCount: workspaceCount(len(v.Criteria)), TotalCount: workspaceCount(total)}
	if end < len(items) {
		coverage.Status = "partial"
		coverage.Reason = "current_criteria_page"
		coverage.NextCursor = encodeCursor(queryCursor{Namespace: v.Subject.Ref.NamespaceID, Outcome: v.Subject.Ref.OutcomeID, Section: "workspace_criteria", Filter: hash, Revision: f.revision, Key: items[end-1].ID.String()})
	}
	v.Coverage["criteria"] = coverage
	return nil
}

func (s *Service) workspaceRelationships(ctx context.Context, f *continuityReadFrame, v *EntityWorkspaceView, q WorkspaceViewQuery, limit int) error {
	graph, ok := f.repo.(ports.GraphReadRepository)
	if !ok {
		v.Coverage["relationships"] = WorkspaceCoverage{Status: "unavailable", Reason: "graph_adapter_unavailable"}
		return nil
	}
	after := ""
	hash := workspaceCursorFilter(ctx, q.Subject)
	if q.RelationCursor != "" {
		c, e := decodeCursor(q.RelationCursor, q.Subject.Ref.NamespaceID, q.Subject.Ref.OutcomeID, "workspace_relationships", hash)
		if e != nil {
			return e
		}
		if c.Revision != f.revision {
			return d.NewError(d.ErrorCodePreconditionFailed, "snapshot_changed")
		}
		after = c.Key
	}
	edges, e := graph.GraphAdjacent(ctx, q.Subject.Ref.Scope, []d.EntityRef{q.Subject.Ref}, "both", nil, after, limit+1)
	if e != nil {
		return e
	}
	coverage := WorkspaceCoverage{Status: "complete"}
	if len(edges) > limit {
		edges = edges[:limit]
		coverage.Status = "partial"
		coverage.Reason = "direct_relationships_page"
		coverage.NextCursor = encodeCursor(queryCursor{Namespace: q.Subject.Ref.NamespaceID, Outcome: q.Subject.Ref.OutcomeID, Section: "workspace_relationships", Filter: hash, Revision: f.revision, Key: edges[len(edges)-1].ID})
	}
	refs := []d.EntityRef{}
	seen := map[d.EntityRef]bool{}
	for _, edge := range edges {
		relationship := WorkspaceRelationship{GraphEdge: GraphEdge{ID: edge.ID, Source: edge.Source, Target: edge.Target, Kind: edge.Kind, SourceOfTruth: edge.SourceOfTruth}}
		if edge.SourceOfTruth == "relation" {
			id, e := d.ParseID(edge.ID)
			if e != nil {
				return e
			}
			record, e := f.u.Relations().Get(ctx, q.Subject.Ref.Scope, id)
			if e != nil {
				return e
			}
			relationship.Strength = record.Strength
			relationship.Rationale = workspaceText(record.Reason, 512)
		}
		if edge.SourceOfTruth == "evidence_link" {
			id, e := d.ParseID(edge.ID)
			if e != nil {
				return e
			}
			records, ok := f.u.(ports.DocumentaryUnitOfWork)
			if !ok {
				return d.NewError(d.ErrorCodeInvalidConfig, "documentary read adapter is unavailable")
			}
			record, e := records.EvidenceLinks().Get(ctx, q.Subject.Ref.Scope, id)
			if e != nil {
				return e
			}
			relationship.Stance = record.Stance
			relationship.Rationale = workspaceText(record.Rationale, 512)
			relationship.CriterionID = record.CriterionID
			if q.Subject.CriterionID != nil && (record.CriterionID == nil || *record.CriterionID != *q.Subject.CriterionID) {
				continue
			}
		}
		if q.Subject.Kind == "criterion" && edge.SourceOfTruth != "evidence_link" {
			continue
		}
		v.Relationships.Items = append(v.Relationships.Items, relationship)
		for _, ref := range []d.EntityRef{edge.Source, edge.Target} {
			if !seen[ref] {
				seen[ref] = true
				refs = append(refs, ref)
			}
		}
	}
	if len(refs) > 0 {
		nodes, e := graph.GraphNodes(ctx, q.Subject.Ref.Scope, refs)
		if e != nil {
			return e
		}
		for _, node := range nodes {
			v.Relationships.Nodes = append(v.Relationships.Nodes, EntitySummary(node))
		}
	}
	coverage.ReturnedCount = workspaceCount(len(v.Relationships.Items))
	v.Coverage["relationships"] = coverage
	return nil
}

// A selected relation is a record with two endpoints, not an adjacency root.
// Resolve its canonical endpoints under this same snapshot so labels retain
// meaning even when the record itself has no graph-adjacent edges.
func (s *Service) workspaceSelectedRelationship(ctx context.Context, f *continuityReadFrame, v *EntityWorkspaceView, selected any) error {
	var edge WorkspaceRelationship
	switch record := selected.(type) {
	case d.Relation:
		edge = WorkspaceRelationship{GraphEdge: GraphEdge{ID: record.ID.String(), Source: record.SourceRef, Target: record.TargetRef, Kind: string(record.RelationType), SourceOfTruth: "relation"}, Strength: record.Strength, Rationale: workspaceText(record.Reason, 512)}
	case d.EvidenceLink:
		edge = WorkspaceRelationship{GraphEdge: GraphEdge{ID: record.ID.String(), Source: d.EntityRef{Scope: record.Scope, Kind: d.EntityKindEvidence, ID: record.EvidenceID}, Target: record.TargetRef, Kind: "evidence_link", SourceOfTruth: "evidence_link"}, Stance: record.Stance, CriterionID: record.CriterionID, Rationale: workspaceText(record.Rationale, 512)}
	default:
		return nil
	}
	for _, known := range v.Relationships.Items {
		if known.ID == edge.ID {
			return nil
		}
	}
	v.Relationships.Items = append(v.Relationships.Items, edge)
	graph, ok := f.repo.(ports.GraphReadRepository)
	if !ok {
		return nil
	}
	nodes, e := graph.GraphNodes(ctx, v.Subject.Ref.Scope, []d.EntityRef{edge.Source, edge.Target})
	if e != nil {
		return e
	}
	known := map[d.EntityRef]bool{}
	for _, n := range v.Relationships.Nodes {
		known[n.Ref] = true
	}
	for _, node := range nodes {
		if !known[node.Ref] {
			v.Relationships.Nodes = append(v.Relationships.Nodes, EntitySummary(node))
			known[node.Ref] = true
		}
	}
	coverage := v.Coverage["relationships"]
	coverage.ReturnedCount = workspaceCount(len(v.Relationships.Items))
	v.Coverage["relationships"] = coverage
	return nil
}

func (s *Service) workspaceRequiredObjectives(ctx context.Context, f *continuityReadFrame, v *EntityWorkspaceView, q WorkspaceViewQuery, limit int) error {
	repo := f.repo.(ports.WorkspaceReadRepository)
	after := ""
	hash := workspaceCursorFilter(ctx, q.Subject)
	if q.RequiredObjectivesCursor != "" {
		cursor, e := decodeCursor(q.RequiredObjectivesCursor, q.Subject.Ref.NamespaceID, q.Subject.Ref.OutcomeID, "workspace_required_objectives", hash)
		if e != nil {
			return e
		}
		if cursor.Revision != f.revision {
			return d.NewError(d.ErrorCodePreconditionFailed, "snapshot_changed")
		}
		after = cursor.Key
	}
	items, total, e := repo.WorkspaceRequiredObjectivePage(ctx, q.Subject.Ref.Scope, after, limit+1)
	if e != nil {
		return e
	}
	coverage := WorkspaceCoverage{Status: "complete", TotalCount: workspaceCount(total)}
	if len(items) > limit {
		items = items[:limit]
		coverage.Status = "partial"
		coverage.Reason = "global_required_objectives_page"
		coverage.NextCursor = encodeCursor(queryCursor{Namespace: q.Subject.Ref.NamespaceID, Outcome: q.Subject.Ref.OutcomeID, Section: "workspace_required_objectives", Filter: hash, Revision: f.revision, Key: items[len(items)-1].ID.String()})
	}
	v.RequiredObjectives = []EntitySummary{}
	for _, item := range items {
		v.RequiredObjectives = append(v.RequiredObjectives, summary(item.Ref(), item.Version, item.Title, string(item.Lifecycle), item.Priority))
	}
	coverage.ReturnedCount = workspaceCount(len(items))
	v.Coverage["required_objectives"] = coverage
	return nil
}
