package application

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

const (
	DependencyImpactUpstream   = "upstream"
	DependencyImpactDownstream = "downstream"

	DependencyImpactDirectBlocked   = "direct_blocked"
	DependencyImpactHardWaiting     = "hard_waiting"
	DependencyImpactAdvisoryRelated = "advisory_related"

	DependencyImpactCurrent       = "current"
	DependencyImpactPotential     = "potential"
	DependencyImpactInformational = "informational"

	DependencyImpactGateTaskActivation         = "task_activation"
	DependencyImpactGateTaskAcquisition        = "task_acquisition"
	DependencyImpactGateTaskDeliveryCompletion = "task_delivery_or_completion"
	DependencyImpactGateObjectiveStart         = "objective_start"
	DependencyImpactGateObjectiveAcceptance    = "objective_acceptance"

	dependencyImpactDefaultDepth = 4
	dependencyImpactMaxDepth     = 8
	dependencyImpactMaxNodes     = 500
	dependencyImpactMaxEdges     = 2000
	dependencyImpactMaxBlockers  = 500
	dependencyImpactSampleSize   = 5
)

type DependencyImpactQuery struct {
	Subject   domain.EntityRef `json:"subject"`
	Direction string           `json:"direction,omitempty"`
	Depth     int              `json:"depth,omitempty"`
	Limit     int              `json:"-"`
	Cursor    string           `json:"-"`
}

type DependencyImpactStep struct {
	RelationID domain.ID                 `json:"relation_id"`
	SourceRef  domain.EntityRef          `json:"source_ref"`
	TargetRef  domain.EntityRef          `json:"target_ref"`
	Strength   domain.DependencyStrength `json:"strength"`
}

// DependencyImpactDependency is one direct, observed dependency of the node.
// Dependencies is a bounded sample; its total and satisfaction counters are
// separate fields and are omitted when the direct relation scan is incomplete.
type DependencyImpactDependency struct {
	RelationID domain.ID                 `json:"relation_id"`
	TargetRef  domain.EntityRef          `json:"target_ref"`
	Strength   domain.DependencyStrength `json:"strength"`
	Satisfied  bool                      `json:"satisfied"`
	Gate       string                    `json:"gate,omitempty"`
}

type DependencyImpactEffect struct {
	Class  string `json:"class"`
	Status string `json:"status"`
	Gate   string `json:"gate,omitempty"`
}

// DependencyImpactNode is a bounded operational summary. Direct Blockers and
// unsatisfied dependencies are reported independently; neither is inferred
// from the other. Counts are nil when the relevant scan is incomplete.
type DependencyImpactNode struct {
	Summary                             EntitySummary                `json:"summary"`
	Depth                               int                          `json:"depth"`
	Path                                []DependencyImpactStep       `json:"path"`
	Effects                             []DependencyImpactEffect     `json:"effects"`
	Dependencies                        []DependencyImpactDependency `json:"dependencies"`
	DependencyCount                     *int                         `json:"dependency_count,omitempty"`
	HardDependencyCount                 *int                         `json:"hard_dependency_count,omitempty"`
	AdvisoryDependencyCount             *int                         `json:"advisory_dependency_count,omitempty"`
	SatisfiedHardDependencyCount        *int                         `json:"satisfied_hard_dependency_count,omitempty"`
	SatisfiedAdvisoryDependencyCount    *int                         `json:"satisfied_advisory_dependency_count,omitempty"`
	DependencyCoverageComplete          bool                         `json:"dependency_coverage_complete"`
	DirectBlocked                       bool                         `json:"direct_blocked"`
	DirectBlockerCount                  *int                         `json:"direct_blocker_count,omitempty"`
	DirectBlockerSample                 []domain.EntityRef           `json:"direct_blocker_sample"`
	UnsatisfiedHardDependencyCount      *int                         `json:"unsatisfied_hard_dependency_count,omitempty"`
	UnsatisfiedHardDependencySample     []domain.EntityRef           `json:"unsatisfied_hard_dependency_sample"`
	UnsatisfiedAdvisoryDependencyCount  *int                         `json:"unsatisfied_advisory_dependency_count,omitempty"`
	UnsatisfiedAdvisoryDependencySample []domain.EntityRef           `json:"unsatisfied_advisory_dependency_sample"`
	StateCoverageComplete               bool                         `json:"state_coverage_complete"`
}

type DependencyImpactCoverage struct {
	Status       string `json:"status"`
	Complete     bool   `json:"complete"`
	Reason       string `json:"reason,omitempty"`
	VisitedNodes int    `json:"visited_nodes"`
	ScannedEdges int    `json:"scanned_edges"`
	LimitNodes   int    `json:"limit_nodes"`
	LimitEdges   int    `json:"limit_edges"`
	DepthLimit   int    `json:"depth_limit"`
}

type DependencyImpactPage struct {
	Root            DependencyImpactNode     `json:"root"`
	Direction       string                   `json:"direction"`
	Items           []DependencyImpactNode   `json:"items"`
	Coverage        DependencyImpactCoverage `json:"coverage"`
	OutcomeRevision domain.OutcomeRevision   `json:"outcome_revision"`
	EvaluatedAt     time.Time                `json:"evaluated_at"`
	NextCursor      string                   `json:"next_cursor,omitempty"`
	Limit           int                      `json:"limit"`
}

type dependencyImpactWalkNode struct {
	depth    int
	path     []DependencyImpactStep
	pathRefs []domain.EntityRef
	pathKey  string
	advisory bool
}

func (s *Service) GetDependencyImpact(ctx context.Context, scope domain.Scope, q DependencyImpactQuery) (DependencyImpactPage, error) {
	var zero DependencyImpactPage
	if err := q.Subject.Validate(); err != nil || q.Subject.Scope != scope || (q.Subject.Kind != domain.EntityKindObjective && q.Subject.Kind != domain.EntityKindWorkItem) {
		return zero, domain.NewError(domain.ErrorCodeInvalidEntityRef, "dependency impact subject must be an Objective or Task in this Outcome")
	}
	if q.Direction == "" {
		q.Direction = DependencyImpactDownstream
	}
	if q.Direction != DependencyImpactUpstream && q.Direction != DependencyImpactDownstream {
		return zero, domain.NewError(domain.ErrorCodeInvalidArgument, "dependency impact direction must be upstream or downstream")
	}
	if q.Depth == 0 {
		q.Depth = dependencyImpactDefaultDepth
	}
	if q.Depth < 1 || q.Depth > dependencyImpactMaxDepth {
		return zero, domain.NewError(domain.ErrorCodeInvalidArgument, "dependency impact depth must be between 1 and 8")
	}
	limit, err := queryLimit(q.Limit)
	if err != nil {
		return zero, err
	}
	frame, handled, err := s.openContinuityFrame(ctx, scope)
	if err != nil {
		return zero, err
	}
	if !handled {
		return zero, domain.NewError(domain.ErrorCodeInvalidConfig, "dependency impact read adapter is unavailable")
	}
	defer frame.u.Rollback()
	graph, ok := frame.repo.(ports.GraphReadRepository)
	if !ok {
		return zero, domain.NewError(domain.ErrorCodeInvalidConfig, "dependency graph read adapter is unavailable")
	}
	filter := filterHash(struct {
		Subject   domain.EntityRef `json:"subject"`
		Direction string           `json:"direction"`
		Depth     int              `json:"depth"`
	}{q.Subject, q.Direction, q.Depth})
	after := ""
	if q.Cursor != "" {
		cursor, decodeErr := decodeCursor(q.Cursor, scope.NamespaceID, scope.OutcomeID, "dependency_impact", filter)
		if decodeErr != nil {
			return zero, decodeErr
		}
		if cursor.Revision != frame.revision {
			return zero, domain.NewError(domain.ErrorCodePreconditionFailed, "snapshot_changed")
		}
		after = cursor.Key
	}

	rootSummaries, err := graph.GraphNodes(ctx, scope, []domain.EntityRef{q.Subject})
	if err != nil {
		return zero, err
	}
	if len(rootSummaries) != 1 || rootSummaries[0].Ref != q.Subject {
		return zero, domain.NewError(domain.ErrorCodeNotFound, "dependency impact subject was not found")
	}
	coverage := DependencyImpactCoverage{Status: "complete", Complete: true, LimitNodes: dependencyImpactMaxNodes, LimitEdges: dependencyImpactMaxEdges, DepthLimit: q.Depth}
	walk := map[domain.EntityRef]*dependencyImpactWalkNode{
		q.Subject: {pathRefs: []domain.EntityRef{q.Subject}, path: []DependencyImpactStep{}},
	}
	dependencyEdges := map[string]ports.GraphReadEdge{}
	dependencyComplete := map[domain.EntityRef]bool{q.Subject: true}
	blockerComplete := map[domain.EntityRef]bool{}
	blockersByTarget := map[domain.EntityRef][]domain.EntityRef{}
	usedEdges := 0
	markPartial := func(reason string) {
		coverage.Complete = false
		coverage.Status = "partial"
		if coverage.Reason == "" {
			coverage.Reason = reason
		}
	}
	walkDirection := "in"
	if q.Direction == DependencyImpactUpstream {
		walkDirection = "out"
	}
	frontier := []domain.EntityRef{q.Subject}
	for depth := 0; depth <= q.Depth && len(frontier) > 0; depth++ {
		edges, complete, err := dependencyImpactAdjacent(ctx, graph, scope, frontier, walkDirection, "depends_on", &usedEdges, dependencyImpactMaxEdges)
		if err != nil {
			return zero, err
		}
		if !complete {
			markPartial("edge_limit")
			for _, ref := range frontier {
				dependencyComplete[ref] = false
			}
		}
		for _, edge := range edges {
			if err := validateDependencyImpactEdge(scope, edge); err != nil {
				return zero, err
			}
			dependencyEdges[edge.ID] = edge
		}
		coverage.ScannedEdges += len(edges)
		if depth == q.Depth {
			for _, edge := range edges {
				current := edge.Source
				next := edge.Target
				if walkDirection == "in" {
					current, next = edge.Target, edge.Source
				}
				if current == next {
					return zero, domain.NewError(domain.ErrorCodeDependencyCycle, "dependency impact graph contains a cycle")
				}
				if currentNode := walk[current]; currentNode != nil && currentNode.hasPathTo(next) {
					return zero, domain.NewError(domain.ErrorCodeDependencyCycle, "dependency impact graph contains a cycle")
				}
				if _, reached := walk[next]; !reached {
					markPartial("depth_limit")
					break
				}
			}
			break
		}
		type candidate struct {
			from domain.EntityRef
			to   domain.EntityRef
			edge ports.GraphReadEdge
		}
		candidates := make([]candidate, 0, len(edges))
		frontierSet := map[domain.EntityRef]bool{}
		for _, ref := range frontier {
			frontierSet[ref] = true
		}
		for _, edge := range edges {
			from, to := edge.Source, edge.Target
			if walkDirection == "in" {
				from, to = edge.Target, edge.Source
			}
			if !frontierSet[from] {
				continue
			}
			candidates = append(candidates, candidate{from: from, to: to, edge: edge})
		}
		sort.Slice(candidates, func(i, j int) bool {
			if dependencyImpactRefKey(candidates[i].to) != dependencyImpactRefKey(candidates[j].to) {
				return dependencyImpactRefKey(candidates[i].to) < dependencyImpactRefKey(candidates[j].to)
			}
			return candidates[i].edge.ID < candidates[j].edge.ID
		})
		nextFrontier := []domain.EntityRef{}
		for _, candidate := range candidates {
			parent := walk[candidate.from]
			if parent == nil {
				return zero, domain.NewError(domain.ErrorCodeInvalidRelation, "dependency graph returned an edge outside the traversal frontier")
			}
			if parent.hasPathTo(candidate.to) {
				return zero, domain.NewError(domain.ErrorCodeDependencyCycle, "dependency impact graph contains a cycle")
			}
			step := DependencyImpactStep{RelationID: domain.ID(candidate.edge.ID), SourceRef: candidate.edge.Source, TargetRef: candidate.edge.Target, Strength: candidate.edge.Strength}
			path := append(append([]DependencyImpactStep(nil), parent.path...), step)
			pathRefs := append(append([]domain.EntityRef(nil), parent.pathRefs...), candidate.to)
			pathKey := dependencyImpactPathKey(path)
			advisory := parent.advisory || candidate.edge.Strength == domain.DependencyStrengthAdvisory
			if existing, found := walk[candidate.to]; found {
				if existing.depth == depth+1 && pathKey < existing.pathKey {
					existing.path, existing.pathRefs, existing.pathKey, existing.advisory = path, pathRefs, pathKey, advisory
				}
				continue
			}
			if len(walk) >= dependencyImpactMaxNodes {
				markPartial("node_limit")
				continue
			}
			walk[candidate.to] = &dependencyImpactWalkNode{depth: depth + 1, path: path, pathRefs: pathRefs, pathKey: pathKey, advisory: advisory}
			dependencyComplete[candidate.to] = true
			nextFrontier = append(nextFrontier, candidate.to)
		}
		if !complete {
			break
		}
		sort.Slice(nextFrontier, func(i, j int) bool {
			return dependencyImpactRefKey(nextFrontier[i]) < dependencyImpactRefKey(nextFrontier[j])
		})
		frontier = nextFrontier
	}
	coverage.VisitedNodes = len(walk)

	allRefs := make([]domain.EntityRef, 0, len(walk))
	for ref := range walk {
		allRefs = append(allRefs, ref)
	}
	sort.Slice(allRefs, func(i, j int) bool { return dependencyImpactRefKey(allRefs[i]) < dependencyImpactRefKey(allRefs[j]) })
	// The traversal direction answers who is upstream/downstream. Gate facts
	// always describe the complete direct dependency set of each visible node,
	// so load outgoing relations in both directions within this same snapshot.
	outgoing, complete, err := dependencyImpactAdjacent(ctx, graph, scope, allRefs, "out", "depends_on", &usedEdges, dependencyImpactMaxEdges)
	if err != nil {
		return zero, err
	}
	for _, edge := range outgoing {
		if err := validateDependencyImpactEdge(scope, edge); err != nil {
			return zero, err
		}
		dependencyEdges[edge.ID] = edge
	}
	coverage.ScannedEdges += len(outgoing)
	if !complete {
		markPartial("edge_limit")
		for _, ref := range allRefs {
			dependencyComplete[ref] = false
		}
	}
	blockerEdges, blockersReadComplete, err := dependencyImpactAdjacent(ctx, graph, scope, allRefs, "both", "blocks", new(int), dependencyImpactMaxBlockers)
	if err != nil {
		return zero, err
	}
	for _, edge := range blockerEdges {
		if edge.Source.Scope != scope || edge.Target.Scope != scope || edge.Kind != "blocks" || edge.SourceOfTruth != "blocker" || edge.Source.Kind != domain.EntityKindBlocker || (edge.Target.Kind != domain.EntityKindObjective && edge.Target.Kind != domain.EntityKindWorkItem) {
			return zero, domain.NewError(domain.ErrorCodeInvalidRelation, "dependency impact graph returned an invalid Blocker edge")
		}
		blockersByTarget[edge.Target] = append(blockersByTarget[edge.Target], edge.Source)
	}
	for _, ref := range allRefs {
		blockerComplete[ref] = blockersReadComplete
	}
	if !blockersReadComplete {
		markPartial("blocker_limit")
	}

	dependencyTargets := map[domain.EntityRef]bool{}
	for _, edge := range dependencyEdges {
		dependencyTargets[edge.Target] = true
	}
	missingRefs := make([]domain.EntityRef, 0)
	for ref := range dependencyTargets {
		if _, ok := walk[ref]; !ok {
			missingRefs = append(missingRefs, ref)
		}
	}
	statusRefs := append(append([]domain.EntityRef(nil), allRefs...), missingRefs...)
	statusRefs = uniqueDependencyImpactRefs(statusRefs)
	summaries := make(map[domain.EntityRef]ports.ContinuitySummary, len(statusRefs))
	for start := 0; start < len(statusRefs); start += 100 {
		batch := statusRefs[start:min(start+100, len(statusRefs))]
		values, err := graph.GraphNodes(ctx, scope, batch)
		if err != nil {
			return zero, err
		}
		for _, value := range values {
			if value.Ref.Scope != scope || !containsDependencyImpactRef(batch, value.Ref) {
				return zero, domain.NewError(domain.ErrorCodeInvalidRelation, "dependency impact node read returned an unexpected reference")
			}
			summaries[value.Ref] = value
		}
	}
	for _, ref := range statusRefs {
		if _, ok := summaries[ref]; !ok {
			return zero, domain.NewError(domain.ErrorCodeNotFound, "dependency impact graph references a missing node")
		}
	}

	dependenciesBySource := map[domain.EntityRef][]ports.GraphReadEdge{}
	for _, edge := range dependencyEdges {
		dependenciesBySource[edge.Source] = append(dependenciesBySource[edge.Source], edge)
	}
	for source := range dependenciesBySource {
		sort.Slice(dependenciesBySource[source], func(i, j int) bool { return dependenciesBySource[source][i].ID < dependenciesBySource[source][j].ID })
	}
	buildNode := func(ref domain.EntityRef) DependencyImpactNode {
		value := walk[ref]
		summary := summaries[ref]
		node := DependencyImpactNode{Summary: EntitySummary(summary), Depth: value.depth, Path: append([]DependencyImpactStep(nil), value.path...), Effects: []DependencyImpactEffect{}, Dependencies: []DependencyImpactDependency{}, DirectBlockerSample: []domain.EntityRef{}, UnsatisfiedHardDependencySample: []domain.EntityRef{}, UnsatisfiedAdvisoryDependencySample: []domain.EntityRef{}, DependencyCoverageComplete: dependencyComplete[ref], StateCoverageComplete: dependencyComplete[ref] && blockerComplete[ref]}
		blockerRefs := append([]domain.EntityRef(nil), blockersByTarget[ref]...)
		sort.Slice(blockerRefs, func(i, j int) bool {
			return dependencyImpactRefKey(blockerRefs[i]) < dependencyImpactRefKey(blockerRefs[j])
		})
		if blockerComplete[ref] {
			count := len(blockerRefs)
			node.DirectBlockerCount = &count
			node.DirectBlocked = count > 0
			node.DirectBlockerSample = blockerRefs[:min(len(blockerRefs), dependencyImpactSampleSize)]
		}
		hardRefs, advisoryRefs := []domain.EntityRef{}, []domain.EntityRef{}
		satisfiedHard, satisfiedAdvisory := 0, 0
		directDependencies := dependenciesBySource[ref]
		for _, edge := range directDependencies {
			target := summaries[edge.Target]
			completed := target.Lifecycle == string(domain.WorkItemLifecycleDone) || target.Lifecycle == string(domain.ObjectiveLifecycleAchieved)
			dependency := DependencyImpactDependency{RelationID: domain.ID(edge.ID), TargetRef: edge.Target, Strength: edge.Strength, Satisfied: completed}
			if edge.Strength == domain.DependencyStrengthHard {
				dependency.Gate = dependencyImpactGate(summary, DependencyImpactHardWaiting)
				if completed {
					satisfiedHard++
				}
			} else if completed {
				satisfiedAdvisory++
			}
			node.Dependencies = append(node.Dependencies, dependency)
		}
		if len(node.Dependencies) > dependencyImpactSampleSize {
			node.Dependencies = node.Dependencies[:dependencyImpactSampleSize]
		}
		for _, edge := range dependenciesBySource[ref] {
			target := summaries[edge.Target]
			completed := target.Lifecycle == string(domain.WorkItemLifecycleDone) || target.Lifecycle == string(domain.ObjectiveLifecycleAchieved)
			if completed {
				continue
			}
			if edge.Strength == domain.DependencyStrengthHard {
				hardRefs = append(hardRefs, edge.Target)
			} else {
				advisoryRefs = append(advisoryRefs, edge.Target)
			}
		}
		sort.Slice(hardRefs, func(i, j int) bool { return dependencyImpactRefKey(hardRefs[i]) < dependencyImpactRefKey(hardRefs[j]) })
		sort.Slice(advisoryRefs, func(i, j int) bool {
			return dependencyImpactRefKey(advisoryRefs[i]) < dependencyImpactRefKey(advisoryRefs[j])
		})
		if dependencyComplete[ref] {
			hardCount, advisoryCount := 0, 0
			for _, edge := range directDependencies {
				if edge.Strength == domain.DependencyStrengthHard {
					hardCount++
				} else {
					advisoryCount++
				}
			}
			dependencyCount := len(directDependencies)
			node.DependencyCount = &dependencyCount
			node.HardDependencyCount = &hardCount
			node.AdvisoryDependencyCount = &advisoryCount
			node.SatisfiedHardDependencyCount = &satisfiedHard
			node.SatisfiedAdvisoryDependencyCount = &satisfiedAdvisory
			unsatisfiedHardCount, unsatisfiedAdvisoryCount := len(hardRefs), len(advisoryRefs)
			node.UnsatisfiedHardDependencyCount = &unsatisfiedHardCount
			node.UnsatisfiedAdvisoryDependencyCount = &unsatisfiedAdvisoryCount
			node.UnsatisfiedHardDependencySample = hardRefs[:min(len(hardRefs), dependencyImpactSampleSize)]
			node.UnsatisfiedAdvisoryDependencySample = advisoryRefs[:min(len(advisoryRefs), dependencyImpactSampleSize)]
		}
		terminal := summaries[ref].Lifecycle == string(domain.WorkItemLifecycleDone) || summaries[ref].Lifecycle == string(domain.WorkItemLifecycleCancelled) || summaries[ref].Lifecycle == string(domain.ObjectiveLifecycleAchieved) || summaries[ref].Lifecycle == string(domain.ObjectiveLifecycleCancelled)
		if blockerComplete[ref] && len(blockerRefs) > 0 {
			status := DependencyImpactCurrent
			if terminal {
				status = DependencyImpactPotential
			}
			node.Effects = append(node.Effects, DependencyImpactEffect{Class: DependencyImpactDirectBlocked, Status: status, Gate: dependencyImpactGate(summary, DependencyImpactDirectBlocked)})
		}
		if dependencyComplete[ref] && len(hardRefs) > 0 {
			status := DependencyImpactPotential
			if !terminal && dependencyImpactCanWait(summaries[ref]) {
				status = DependencyImpactCurrent
			}
			node.Effects = append(node.Effects, DependencyImpactEffect{Class: DependencyImpactHardWaiting, Status: status, Gate: dependencyImpactGate(summary, DependencyImpactHardWaiting)})
		}
		if value.advisory || dependencyComplete[ref] && len(advisoryRefs) > 0 {
			node.Effects = append(node.Effects, DependencyImpactEffect{Class: DependencyImpactAdvisoryRelated, Status: DependencyImpactInformational})
		}
		return node
	}
	root := buildNode(q.Subject)
	items := make([]DependencyImpactNode, 0, len(walk)-1)
	for _, ref := range allRefs {
		if ref != q.Subject {
			items = append(items, buildNode(ref))
		}
	}
	sort.Slice(items, func(i, j int) bool {
		return dependencyImpactRefKey(items[i].Summary.Ref) < dependencyImpactRefKey(items[j].Summary.Ref)
	})
	start := 0
	for start < len(items) && dependencyImpactRefKey(items[start].Summary.Ref) <= after {
		start++
	}
	pageItems := items[start:min(start+limit, len(items))]
	page := DependencyImpactPage{Root: root, Direction: q.Direction, Items: pageItems, Coverage: coverage, OutcomeRevision: frame.revision, EvaluatedAt: frame.at, Limit: limit}
	if start+len(pageItems) < len(items) && len(pageItems) > 0 {
		last := pageItems[len(pageItems)-1]
		page.NextCursor = encodeCursor(queryCursor{Namespace: scope.NamespaceID, Outcome: scope.OutcomeID, Section: "dependency_impact", Filter: filter, Revision: frame.revision, Key: dependencyImpactRefKey(last.Summary.Ref)})
	}
	return page, nil
}

func dependencyImpactAdjacent(ctx context.Context, repo ports.GraphReadRepository, scope domain.Scope, refs []domain.EntityRef, direction, kind string, used *int, maxEdges int) ([]ports.GraphReadEdge, bool, error) {
	result := []ports.GraphReadEdge{}
	for offset := 0; offset < len(refs); offset += 100 {
		batch := refs[offset:min(offset+100, len(refs))]
		after := ""
		for {
			remaining := maxEdges - *used
			pageLimit := min(101, remaining+1)
			if pageLimit < 1 {
				return result, false, nil
			}
			edges, err := repo.GraphAdjacent(ctx, scope, batch, direction, []string{kind}, after, pageLimit)
			if err != nil {
				return nil, false, err
			}
			if len(edges) > remaining {
				edges = edges[:max(0, remaining)]
				result = append(result, edges...)
				*used += len(edges)
				return result, false, nil
			}
			result = append(result, edges...)
			*used += len(edges)
			if len(edges) < pageLimit {
				break
			}
			if len(edges) == 0 {
				break
			}
			after = edges[len(edges)-1].ID
		}
	}
	return result, true, nil
}

func validateDependencyImpactEdge(scope domain.Scope, edge ports.GraphReadEdge) error {
	if edge.ID == "" || edge.SourceOfTruth != "relation" || edge.Kind != string(domain.RelationTypeDependsOn) || edge.Source.Scope != scope || edge.Target.Scope != scope || (edge.Source.Kind != domain.EntityKindObjective && edge.Source.Kind != domain.EntityKindWorkItem) || (edge.Target.Kind != domain.EntityKindObjective && edge.Target.Kind != domain.EntityKindWorkItem) || !edge.Strength.Valid() {
		return domain.NewError(domain.ErrorCodeInvalidRelation, "dependency impact graph returned an invalid dependency edge")
	}
	if _, err := domain.ParseID(edge.ID); err != nil {
		return domain.WrapError(domain.ErrorCodeInvalidRelation, "dependency impact relation ID is invalid", err)
	}
	return nil
}

func (n *dependencyImpactWalkNode) hasPathTo(ref domain.EntityRef) bool {
	if n == nil {
		return false
	}
	for _, value := range n.pathRefs {
		if value == ref {
			return true
		}
	}
	return false
}

func dependencyImpactRefKey(ref domain.EntityRef) string {
	return ref.Kind.String() + ":" + ref.ID.String()
}

func dependencyImpactPathKey(path []DependencyImpactStep) string {
	parts := make([]string, 0, len(path))
	for _, step := range path {
		parts = append(parts, step.RelationID.String())
	}
	return strings.Join(parts, "/")
}

func uniqueDependencyImpactRefs(refs []domain.EntityRef) []domain.EntityRef {
	seen := map[domain.EntityRef]bool{}
	result := make([]domain.EntityRef, 0, len(refs))
	for _, ref := range refs {
		if !seen[ref] {
			seen[ref] = true
			result = append(result, ref)
		}
	}
	sort.Slice(result, func(i, j int) bool { return dependencyImpactRefKey(result[i]) < dependencyImpactRefKey(result[j]) })
	return result
}

func containsDependencyImpactRef(refs []domain.EntityRef, wanted domain.EntityRef) bool {
	for _, ref := range refs {
		if ref == wanted {
			return true
		}
	}
	return false
}

func dependencyImpactCanWait(summary ports.ContinuitySummary) bool {
	switch domain.EntityKind(summary.Ref.Kind) {
	case domain.EntityKindObjective:
		return summary.Lifecycle == string(domain.ObjectiveLifecyclePlanned) || summary.Lifecycle == string(domain.ObjectiveLifecycleInProgress)
	case domain.EntityKindWorkItem:
		return summary.Lifecycle == string(domain.WorkItemLifecycleTodo) || summary.Lifecycle == string(domain.WorkItemLifecycleInProgress)
	default:
		return false
	}
}

// dependencyImpactGate mirrors the concrete operations whose existing guards
// consume hard dependencies or active Blockers. Advisory relations never call
// this helper because they do not participate in readiness or execution gates.
func dependencyImpactGate(summary ports.ContinuitySummary, effect string) string {
	switch domain.EntityKind(summary.Ref.Kind) {
	case domain.EntityKindObjective:
		switch summary.Lifecycle {
		case string(domain.ObjectiveLifecyclePlanned):
			return DependencyImpactGateObjectiveStart
		case string(domain.ObjectiveLifecycleInProgress):
			return DependencyImpactGateObjectiveAcceptance
		}
	case domain.EntityKindWorkItem:
		switch summary.Lifecycle {
		case string(domain.WorkItemLifecycleBacklog):
			if effect == DependencyImpactDirectBlocked {
				return DependencyImpactGateTaskActivation
			}
			return DependencyImpactGateTaskAcquisition
		case string(domain.WorkItemLifecycleTodo):
			return DependencyImpactGateTaskAcquisition
		case string(domain.WorkItemLifecycleInProgress):
			return DependencyImpactGateTaskDeliveryCompletion
		}
	}
	return ""
}
