package ports

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// GraphReadEdge contains only adjacency, never proof or documentary bodies.
type GraphReadEdge struct {
	ID     string      `json:"id"`
	Source d.EntityRef `json:"source_ref"`
	Target d.EntityRef `json:"target_ref"`
	Kind   string      `json:"kind"`
	// Strength is populated only for depends_on relations. Other graph edges
	// have no dependency semantics and leave it empty.
	Strength      d.DependencyStrength `json:"strength,omitempty"`
	SourceOfTruth string               `json:"source_of_truth"`
}

// GraphReadRepository pages adjacency for at most 100 frontier references. The
// caller retains reachability IDs, then requests only returned node summaries.
type GraphReadRepository interface {
	GraphAdjacent(context.Context, d.Scope, []d.EntityRef, string, []string, string, int) ([]GraphReadEdge, error)
	GraphNodes(context.Context, d.Scope, []d.EntityRef) ([]ContinuitySummary, error)
}
