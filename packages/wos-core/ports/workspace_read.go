package ports

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// WorkspaceCardFacts deliberately excludes aggregate history, execution specs,
// signed payloads and documentary bodies. SQL adapters resolve one bounded batch,
// rather than opening an inspector (or a transaction) for every card.
type WorkspaceCardFacts struct {
	Ref                d.EntityRef
	ParentRef          *d.EntityRef
	Description        string
	RequiredForOutcome bool
	Actors             []d.ActorRef
	Criteria           []WorkspaceCriterionFact
	PendingReview      bool
	CorrectionRequired bool
}
type WorkspaceCriterionFact struct {
	ID                   d.ID
	Revision             d.CriterionRevision
	Required             bool
	Status               d.CriterionStatus
	AssessmentResult     d.AssessmentResult
	AssessmentApplicable bool
	EvidenceLinked       bool
}

// WorkspaceConclusionImpactFacts contains current-placement metadata only.
// Counts include every open descendant; Items is a bounded stable sample.
// A terminal Objective does not prune its still-open descendants.
type WorkspaceConclusionImpactFacts struct {
	Items          []ContinuitySummary
	OpenObjectives int
	OpenWorkItems  int
}

type WorkspaceReadRepository interface {
	WorkspaceConclusionImpact(context.Context, d.Scope, d.EntityRef, int) (WorkspaceConclusionImpactFacts, error)
	WorkspaceOutcome(context.Context, d.Scope) (d.Outcome, error)
	WorkspaceRevision(context.Context, d.Scope) (d.OutcomeRevision, error)
	WorkspaceObjectiveHeader(context.Context, d.Scope, d.ID) (d.Objective, error)
	WorkspaceRoadmap(context.Context, d.Scope, d.ID, uint64) (d.Roadmap, error)
	WorkspaceGuardWork(context.Context, d.Scope, *d.ID) ([]d.WorkItem, error)
	WorkspaceRequiredObjectives(context.Context, d.Scope) ([]d.Objective, error)
	WorkspaceRequiredObjectivePage(context.Context, d.Scope, string, int) ([]d.Objective, int, error)
	WorkspaceStructure(context.Context, d.Scope, int) ([]d.Objective, []d.WorkItem, error)
	WorkspaceLastExecution(context.Context, d.Scope, d.ID) (*d.WorkContract, error)
	WorkspaceCards(context.Context, d.Scope, []d.EntityRef) ([]WorkspaceCardFacts, error)
	WorkspaceChildren(context.Context, d.Scope, d.EntityRef, string, int) ([]d.EntityRef, error)
}
