package ports

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type FocalHistoryCounts struct {
	Assessments int
	Definitions int
	Conclusions int
}
type FocalCurrentWork struct {
	Work    d.WorkItem
	History FocalHistoryCounts
}
type FocalCurrentObjective struct {
	Objective d.Objective
	History   FocalHistoryCounts
}
type FocalContextRecords struct {
	Dependencies          []d.Relation
	DependencyCount       int
	DependencyEvaluations []d.DependencyEvaluation
	Blockers              []d.Blocker
	ObjectiveAncestors    []d.Objective
	Decisions             []d.Decision
	DecisionCount         int
	EvidenceLinks         []d.EvidenceLink
	EvidenceLinkCount     int
}

// Full current proof is returned for the selected owner. Historical rows are
// counted, never loaded solely to discard them from the response. These are
// read DTOs and must not be saved as complete aggregates.
type FocalReadRepository interface {
	CurrentWork(context.Context, d.Scope, d.ID) (FocalCurrentWork, error)
	CurrentObjective(context.Context, d.Scope, d.ID) (FocalCurrentObjective, error)
	FocalOutcome(context.Context, d.Scope) (d.Outcome, error)
	FocalRecords(context.Context, d.Scope, d.ID, *d.ID) (FocalContextRecords, error)
}
