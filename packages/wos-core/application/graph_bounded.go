package application

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sort"
)

func (s *Service) boundedOutcomeGraph(ctx context.Context, scope d.Scope, q GraphQuery, n int, hash string) (GraphPage, bool, error) {
	var zero GraphPage
	frame, handled, e := s.openContinuityFrame(ctx, scope)
	if !handled || e != nil {
		return zero, handled, e
	}
	defer frame.u.Rollback()
	repo, ok := frame.repo.(ports.GraphReadRepository)
	if !ok {
		return zero, false, nil
	}
	root := d.EntityRef{Scope: scope, Kind: d.EntityKindOutcome, ID: scope.OutcomeID}
	if q.Root != nil {
		root = *q.Root
		if e := root.Validate(); e != nil || root.Scope != scope {
			return zero, true, d.NewError(d.ErrorCodeInvalidEntityRef, "graph root must belong to the Outcome")
		}
	}
	rootNodes, e := repo.GraphNodes(ctx, scope, []d.EntityRef{root})
	if e != nil {
		return zero, true, e
	}
	if len(rootNodes) == 0 {
		return zero, true, d.NewError(d.ErrorCodeNotFound, "graph root not found")
	}
	after := ""
	if q.Cursor != "" {
		c, e := decodeCursor(q.Cursor, scope.TenantID, scope.OutcomeID, "graph", hash)
		if e != nil {
			return zero, true, e
		}
		if c.Revision != frame.revision {
			return zero, true, d.NewError(d.ErrorCodePreconditionFailed, "snapshot_changed")
		}
		after = c.Key
	}
	// Stream metadata pages; retain only IDs needed for reachability. Even a wide
	// frontier never hydrates aggregate proof, history or documentary payloads.
	visit := func(refs []d.EntityRef, direction, start string, fn func(ports.GraphReadEdge)) error {
		for offset := 0; offset < len(refs); offset += 100 {
			batch := refs[offset:min(offset+100, len(refs))]
			key := start
			for {
				edges, e := repo.GraphAdjacent(ctx, scope, batch, direction, q.Kinds, key, 101)
				if e != nil {
					return e
				}
				for _, edge := range edges {
					fn(edge)
				}
				if len(edges) < 101 {
					break
				}
				key = edges[len(edges)-1].ID
			}
		}
		return nil
	}
	reached := map[d.EntityRef]bool{root: true}
	frontier := []d.EntityRef{root}
	for depth := 0; depth < q.Depth && len(frontier) > 0; depth++ {
		next := []d.EntityRef{}
		frontierSet := map[d.EntityRef]bool{}
		for _, ref := range frontier {
			frontierSet[ref] = true
		}
		e = visit(frontier, q.Direction, "", func(edge ports.GraphReadEdge) {
			add := func(ref d.EntityRef) {
				if !reached[ref] {
					reached[ref] = true
					next = append(next, ref)
				}
			}
			// A batch may have both endpoints; each permitted orientation must traverse.
			inFrontier := func(ref d.EntityRef) bool { return frontierSet[ref] }
			if q.Direction != "in" && inFrontier(edge.Source) {
				add(edge.Target)
			}
			if q.Direction != "out" && inFrontier(edge.Target) {
				add(edge.Source)
			}
		})
		if e != nil {
			return zero, true, e
		}
		frontier = next
	}
	refs := make([]d.EntityRef, 0, len(reached))
	for ref := range reached {
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].ID < refs[j].ID })
	// Legacy graphs include every induced edge, including edges between nodes at
	// the depth boundary. Retain only the smallest N+1 unique keys, never all edges.
	selected := map[string]ports.GraphReadEdge{}
	e = visit(refs, "both", after, func(edge ports.GraphReadEdge) {
		if !reached[edge.Source] || !reached[edge.Target] {
			return
		}
		selected[edge.ID] = edge
		if len(selected) > n+1 {
			maxKey := ""
			for key := range selected {
				if key > maxKey {
					maxKey = key
				}
			}
			delete(selected, maxKey)
		}
	})
	if e != nil {
		return zero, true, e
	}
	edges := make([]GraphEdge, 0, len(selected))
	for _, edge := range selected {
		edges = append(edges, GraphEdge(edge))
	}
	sort.Slice(edges, func(i, j int) bool { return edges[i].ID < edges[j].ID })
	page := GraphPage{Root: root, Nodes: []EntitySummary{}, Edges: edges[:min(n, len(edges))], OutcomeRevision: frame.revision, EvaluatedAt: frame.at, Limit: n, Depth: q.Depth}
	if len(edges) > n {
		page.Truncated = true
		page.NextCursor = encodeCursor(queryCursor{Tenant: scope.TenantID, Outcome: scope.OutcomeID, Section: "graph", Filter: hash, Revision: frame.revision, Key: edges[n-1].ID})
	}
	nodeSet := map[d.EntityRef]bool{root: true}
	for _, edge := range page.Edges {
		nodeSet[edge.Source] = true
		nodeSet[edge.Target] = true
	}
	nodeRefs := make([]d.EntityRef, 0, len(nodeSet))
	for ref := range nodeSet {
		nodeRefs = append(nodeRefs, ref)
	}
	nodes, e := repo.GraphNodes(ctx, scope, nodeRefs)
	if e != nil {
		return zero, true, e
	}
	for _, v := range nodes {
		page.Nodes = append(page.Nodes, EntitySummary(v))
	}
	sort.Slice(page.Nodes, func(i, j int) bool { return page.Nodes[i].Ref.ID < page.Nodes[j].Ref.ID })
	raw, e := json.Marshal(page)
	if e != nil {
		return zero, true, e
	}
	if len(raw) > MaxSnapshotBytes {
		return zero, true, d.NewError(d.ErrorCodeGraphLimitExceeded, "graph exceeds byte limit; reduce limit")
	}
	return page, true, nil
}
