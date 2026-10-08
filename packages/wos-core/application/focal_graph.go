package application

import (
	"context"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sort"
	"time"
)

type GraphEdge struct {
	ID            string           `json:"id"`
	Source        domain.EntityRef `json:"source_ref"`
	Target        domain.EntityRef `json:"target_ref"`
	Kind          string           `json:"kind"`
	SourceOfTruth string           `json:"source_of_truth"`
}
type GraphQuery struct {
	Root      *domain.EntityRef `json:"root,omitempty"`
	Direction string            `json:"direction,omitempty"`
	Kinds     []string          `json:"kinds,omitempty"`
	Depth     int               `json:"depth"`
	Limit     int               `json:"-"`
	Cursor    string            `json:"-"`
}
type GraphPage struct {
	Root            domain.EntityRef       `json:"root_ref"`
	Nodes           []EntitySummary        `json:"nodes"`
	Edges           []GraphEdge            `json:"edges"`
	OutcomeRevision domain.OutcomeRevision `json:"outcome_revision"`
	EvaluatedAt     time.Time              `json:"evaluated_at"`
	NextCursor      string                 `json:"next_cursor,omitempty"`
	Truncated       bool                   `json:"truncated"`
	Limit           int                    `json:"limit"`
	Depth           int                    `json:"depth"`
}

func (s *Service) GetOutcomeGraph(ctx context.Context, scope domain.Scope, q GraphQuery) (GraphPage, error) {
	n, err := queryLimit(q.Limit)
	if err != nil {
		return GraphPage{}, err
	}
	if q.Depth < 0 || q.Depth > 8 {
		return GraphPage{}, domain.NewError(domain.ErrorCodeInvalidArgument, "depth must be between 0 and 8")
	}
	if q.Direction == "" {
		q.Direction = "both"
	}
	if q.Direction != "both" && q.Direction != "out" && q.Direction != "in" {
		return GraphPage{}, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid graph direction")
	}
	sort.Strings(q.Kinds)
	hash := filterHash(q)
	state, err := s.GetOutcomeState(ctx, scope)
	if err != nil {
		return GraphPage{}, err
	}
	root := state.Outcome.Ref()
	if q.Root != nil {
		root = *q.Root
		if err := root.Validate(); err != nil || root.Scope != scope {
			return GraphPage{}, domain.NewError(domain.ErrorCodeInvalidEntityRef, "graph root must belong to the Outcome")
		}
	}
	nodes := map[domain.EntityRef]EntitySummary{}
	edges := []GraphEdge{}
	add := func(id string, source, target domain.EntityRef, kind, truth string) {
		if len(q.Kinds) > 0 {
			ok := false
			for _, k := range q.Kinds {
				if k == kind {
					ok = true
				}
			}
			if !ok {
				return
			}
		}
		edges = append(edges, GraphEdge{ID: id, Source: source, Target: target, Kind: kind, SourceOfTruth: truth})
	}
	nodes[state.Outcome.Ref()] = summary(state.Outcome.Ref(), state.Outcome.Version, state.Outcome.Title, string(state.Outcome.Lifecycle), state.Outcome.Priority)
	for _, v := range state.Objectives {
		nodes[v.Ref()] = summary(v.Ref(), v.Version, v.Title, string(v.Lifecycle), v.Priority)
		parent := state.Outcome.Ref()
		if v.ParentObjectiveID != nil {
			parent = domain.EntityRef{Scope: scope, Kind: domain.EntityKindObjective, ID: *v.ParentObjectiveID}
		}
		add("objective:"+v.ID.String(), parent, v.Ref(), "hierarchy", "objective")
	}
	for _, v := range state.WorkItems {
		nodes[v.Ref()] = summary(v.Ref(), v.Version, v.Title, string(v.Lifecycle), v.Priority)
		owner := state.Outcome.Ref()
		if v.ObjectiveID != nil {
			owner = domain.EntityRef{Scope: scope, Kind: domain.EntityKindObjective, ID: *v.ObjectiveID}
		}
		add("work:"+v.ID.String(), owner, v.Ref(), "ownership", "work_item")
	}
	for _, v := range state.Issues {
		nodes[v.Ref()] = summary(v.Ref(), v.Version, v.Title, string(v.Lifecycle), "")
	}
	for _, v := range state.Blockers {
		nodes[v.Ref()] = summary(v.Ref(), v.Version, v.Description, string(v.Lifecycle), "")
		if v.Lifecycle == domain.BlockerLifecycleActive {
			add(v.ID.String(), v.Ref(), v.BlockedRef, "blocks", "blocker")
			if v.CauseRef != nil {
				add(v.ID.String()+":cause", *v.CauseRef, v.Ref(), "causes", "blocker")
			}
		}
	}
	for _, v := range state.Artifacts {
		nodes[v.Ref()] = summary(v.Ref(), v.Version, v.Name, string(v.Lifecycle), "")
	}
	for _, v := range state.Evidence {
		nodes[v.Ref()] = summary(v.Ref(), v.Version, v.Description, string(v.Lifecycle), "")
		if v.ArtifactID != nil {
			add(v.ID.String()+":artifact", v.Ref(), domain.EntityRef{Scope: scope, Kind: domain.EntityKindArtifact, ID: *v.ArtifactID}, "artifact_source", "evidence")
		}
	}
	for _, v := range state.Decisions {
		nodes[v.Ref()] = summary(v.Ref(), v.Version, v.Title, string(v.Lifecycle), "")
		if v.SupersedesDecisionID != nil {
			add(v.ID.String()+":supersedes", v.Ref(), domain.EntityRef{Scope: scope, Kind: domain.EntityKindDecision, ID: *v.SupersedesDecisionID}, "supersedes", "decision")
		}
	}
	for _, v := range state.Relations {
		if v.Lifecycle == domain.RelationLifecycleActive {
			add(v.ID.String(), v.SourceRef, v.TargetRef, string(v.RelationType), "relation")
		}
	}
	for _, v := range state.EvidenceLinks {
		if v.Lifecycle == domain.EvidenceLinkLifecycleActive {
			add(v.ID.String(), domain.EntityRef{Scope: scope, Kind: domain.EntityKindEvidence, ID: v.EvidenceID}, v.TargetRef, string(v.Stance), "evidence_link")
		}
	}
	if _, ok := nodes[root]; !ok {
		return GraphPage{}, domain.NewError(domain.ErrorCodeNotFound, "graph root not found")
	}
	// Bounded BFS uses batch-loaded edges. Relation, hierarchy and documentary graphs keep distinct edge kinds.
	reached := map[domain.EntityRef]int{root: 0}
	frontier := []domain.EntityRef{root}
	for len(frontier) > 0 {
		current := frontier[0]
		frontier = frontier[1:]
		depth := reached[current]
		if depth >= q.Depth {
			continue
		}
		for _, e := range edges {
			var other domain.EntityRef
			match := false
			if e.Source == current && (q.Direction == "out" || q.Direction == "both") {
				other = e.Target
				match = true
			} else if e.Target == current && (q.Direction == "in" || q.Direction == "both") {
				other = e.Source
				match = true
			}
			if match {
				if _, exists := reached[other]; !exists {
					reached[other] = depth + 1
					frontier = append(frontier, other)
				}
			}
		}
	}
	selected := []GraphEdge{}
	for _, e := range edges {
		_, a := reached[e.Source]
		_, b := reached[e.Target]
		if a && b {
			selected = append(selected, e)
		}
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].ID < selected[j].ID })
	after := ""
	if q.Cursor != "" {
		c, err := decodeCursor(q.Cursor, scope.NamespaceID, scope.OutcomeID, "graph", hash)
		if err != nil {
			return GraphPage{}, err
		}
		if c.Revision != state.OutcomeRevision {
			return GraphPage{}, domain.NewError(domain.ErrorCodePreconditionFailed, "snapshot_changed")
		}
		after = c.Key
	}
	start := sort.Search(len(selected), func(i int) bool { return selected[i].ID > after })
	end := min(start+n, len(selected))
	page := GraphPage{Root: root, Nodes: []EntitySummary{}, Edges: selected[start:end], OutcomeRevision: state.OutcomeRevision, EvaluatedAt: state.EvaluatedAt, Limit: n, Depth: q.Depth}
	refs := map[domain.EntityRef]bool{root: true}
	for _, e := range page.Edges {
		refs[e.Source] = true
		refs[e.Target] = true
	}
	for ref := range refs {
		if v, ok := nodes[ref]; ok {
			page.Nodes = append(page.Nodes, v)
		}
	}
	sort.Slice(page.Nodes, func(i, j int) bool { return page.Nodes[i].Ref.ID < page.Nodes[j].Ref.ID })
	if end < len(selected) {
		page.Truncated = true
		page.NextCursor = encodeCursor(queryCursor{Namespace: scope.NamespaceID, Outcome: scope.OutcomeID, Section: "graph", Filter: hash, Revision: state.OutcomeRevision, Key: selected[end-1].ID})
	}
	raw, _ := json.Marshal(page)
	if len(raw) > MaxSnapshotBytes {
		return GraphPage{}, domain.NewError(domain.ErrorCodeGraphLimitExceeded, "graph exceeds byte limit; reduce limit")
	}
	return page, nil
}

type WorkContext struct {
	Contract        *WorkContractView               `json:"contract,omitempty"`
	Work            domain.WorkItem                 `json:"work_item"`
	Outcome         EntitySummary                   `json:"outcome"`
	Objective       *domain.Objective               `json:"objective,omitempty"`
	Operational     domain.WorkItemOperationalState `json:"operational_state"`
	Blocking        *BlockingState                  `json:"blocking,omitempty"`
	Dependencies    []domain.Relation               `json:"dependencies"`
	Decisions       []domain.Decision               `json:"current_decisions"`
	EvidenceLinks   []domain.EvidenceLink           `json:"evidence_links"`
	OutcomeRevision domain.OutcomeRevision          `json:"outcome_revision"`
	EvaluatedAt     time.Time                       `json:"evaluated_at"`
	Truncated       bool                            `json:"truncated"`
	Omitted         map[string]int                  `json:"omitted"`
}

func (s *Service) GetWorkContext(ctx context.Context, scope domain.Scope, id domain.ID) (WorkContext, error) {
	if err := scope.Validate(); err != nil {
		return WorkContext{}, err
	}
	if err := id.Validate(); err != nil {
		return WorkContext{}, err
	}
	if err := s.authorizeRead(ctx, scope.NamespaceID); err != nil {
		return WorkContext{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return WorkContext{}, err
	}
	defer uow.Rollback()
	coord, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return WorkContext{}, err
	}
	work, err := uow.WorkItems().Get(ctx, scope, id)
	if err != nil {
		return WorkContext{}, err
	}
	outcome, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		return WorkContext{}, err
	}
	now, err := s.transactionTime(ctx, uow)
	if err != nil {
		return WorkContext{}, err
	}
	value := WorkContext{Work: work, Outcome: summary(outcome.Ref(), outcome.Version, outcome.Title, string(outcome.Lifecycle), outcome.Priority), OutcomeRevision: coord.Revision, EvaluatedAt: now, Dependencies: []domain.Relation{}, Decisions: []domain.Decision{}, EvidenceLinks: []domain.EvidenceLink{}, Omitted: map[string]int{}}
	relevant := map[domain.EntityRef]bool{outcome.Ref(): true, work.Ref(): true}
	if work.ObjectiveID != nil {
		objective, e := uow.Objectives().Get(ctx, scope, *work.ObjectiveID)
		if e != nil {
			return WorkContext{}, e
		}
		value.Objective = &objective
		relevant[objective.Ref()] = true
	}
	relations, err := uow.Relations().ListByOutcome(ctx, scope)
	if err != nil {
		return WorkContext{}, err
	}
	evaluations, err := dependencyEvaluationsForSource(ctx, uow, work.Ref(), relations)
	if err != nil {
		return WorkContext{}, err
	}
	blocking, err := blockingStateForRef(ctx, uow, work.Ref())
	if err != nil {
		return WorkContext{}, err
	}
	value.Blocking = &blocking
	value.Operational = domain.ProjectWorkItemOperationalState(work, outcome, value.Objective, evaluations, blocking.IsBlocked, now)
	for _, rel := range relations {
		if rel.SourceRef == work.Ref() && rel.Lifecycle == domain.RelationLifecycleActive {
			value.Dependencies = append(value.Dependencies, rel)
		}
	}
	if work.CurrentContractID != nil {
		repo, e := contractRepository(uow)
		if e != nil {
			return WorkContext{}, e
		}
		c, e := repo.Get(ctx, scope, *work.CurrentContractID)
		if e != nil {
			return WorkContext{}, e
		}
		view, e := workContractView(ctx, uow, scope, c, work, coord.Revision, now)
		if e != nil {
			return WorkContext{}, e
		}
		value.Contract = &view.Value
	}
	if records, ok := uow.(ports.DocumentaryUnitOfWork); ok {
		decisions, e := records.Decisions().ListByOutcome(ctx, scope)
		if e != nil {
			return WorkContext{}, e
		}
		for _, v := range decisions {
			if v.Lifecycle == domain.DecisionLifecycleAccepted {
				value.Decisions = append(value.Decisions, v)
			}
		}
		links, e := records.EvidenceLinks().ListByOutcome(ctx, scope)
		if e != nil {
			return WorkContext{}, e
		}
		for _, v := range links {
			if relevant[v.TargetRef] && v.Lifecycle == domain.EvidenceLinkLifecycleActive {
				value.EvidenceLinks = append(value.EvidenceLinks, v)
			}
		}
	}

	if len(value.Decisions) > 25 {
		value.Omitted["decisions"] = len(value.Decisions) - 25
		value.Decisions = value.Decisions[:25]
		value.Truncated = true
	}
	if len(value.EvidenceLinks) > 25 {
		value.Omitted["evidence_links"] = len(value.EvidenceLinks) - 25
		value.EvidenceLinks = value.EvidenceLinks[:25]
		value.Truncated = true
	}
	if len(value.Dependencies) > 100 {
		value.Omitted["dependencies"] = len(value.Dependencies) - 100
		value.Dependencies = value.Dependencies[:100]
		value.Truncated = true
	}
	// Current proof remains; historical proof expands through criterion/conclusion resources.
	for key, count := range map[string]int{"assessment_history": len(value.Work.Criteria.Assessments), "criterion_definition_history": len(value.Work.Criteria.DefinitionRevisions), "conclusion_history": len(value.Work.ConclusionHistory)} {
		if count > 0 {
			value.Omitted[key] = count
			value.Truncated = true
		}
	}
	value.Work.Criteria.Assessments = nil
	value.Work.Criteria.DefinitionRevisions = nil
	value.Work.ConclusionHistory = nil
	if value.Objective != nil {
		value.Omitted["objective_assessment_history"] = len(value.Objective.Criteria.Assessments)
		value.Omitted["objective_definition_history"] = len(value.Objective.Criteria.DefinitionRevisions)
		value.Omitted["objective_conclusion_history"] = len(value.Objective.ConclusionHistory)
		if len(value.Objective.Criteria.Assessments)+len(value.Objective.Criteria.DefinitionRevisions)+len(value.Objective.ConclusionHistory) > 0 {
			value.Truncated = true
		}
		value.Objective.Criteria.Assessments = nil
		value.Objective.Criteria.DefinitionRevisions = nil
		value.Objective.ConclusionHistory = nil
	}
	raw, _ := json.Marshal(value)
	if len(raw) > MaxSnapshotBytes {
		return WorkContext{}, domain.NewError(domain.ErrorCodeGraphLimitExceeded, "work context exceeds byte limit; read individual sections")
	}
	return value, nil
}
