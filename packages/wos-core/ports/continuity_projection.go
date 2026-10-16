package ports

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// These inputs contain only fields consumed by read projections. They are not
// validated writable aggregates and must never be passed to repository Save.
// Documentary body/history fields are deliberately absent. Evaluation time and
// revision are owned by the caller's same authorized UnitOfWork snapshot.
type ContinuityProjectionOptions struct {
	Operational bool
	Conclusions bool
	Progress    bool
	Plans       bool
}
type ContinuityActivePlan struct {
	Slot     d.RoadmapActiveSlot
	Revision d.RoadmapRevision
}
type ContinuityProjectionInputs struct {
	Outcome          d.Outcome
	Objectives       []d.Objective
	WorkItems        []d.WorkItem
	Relations        []d.Relation
	Blockers         []d.Blocker
	Evidence         []d.Evidence
	ActivePlans      []ContinuityActivePlan
	RequiredCriteria int
	MetCriteria      int
	WaivedCriteria   int
}
type ContinuityProjectionRepository interface {
	Projection(context.Context, d.Scope, ContinuityProjectionOptions) (ContinuityProjectionInputs, error)
	BlockerDetails(context.Context, d.Scope, []d.ID, int) ([]d.Blocker, error)
}
