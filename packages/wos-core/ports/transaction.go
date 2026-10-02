package ports

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type OutcomeCoordination struct {
	Scope    domain.Scope
	Revision domain.OutcomeRevision
}

type CoordinationStore interface {
	LockOutcome(ctx context.Context, scope domain.Scope) (OutcomeCoordination, error)
	AdvanceOutcome(ctx context.Context, scope domain.Scope) (domain.OutcomeRevision, error)
}

type UnitOfWork interface {
	Outcomes() OutcomeRepository
	Objectives() ObjectiveRepository
	WorkItems() WorkItemRepository
	Relations() RelationRepository
	Coordination() CoordinationStore
	Events() DomainEventLog
	Idempotency() IdempotencyStore
	Commit() error
	Rollback() error
}

type TransactionManager interface {
	Begin(ctx context.Context) (UnitOfWork, error)
}

type IssueBlockerUnitOfWork interface {
	UnitOfWork
	Issues() IssueRepository
	Blockers() BlockerRepository
}

type DocumentaryUnitOfWork interface {
	Artifacts() ArtifactRepository
	Evidence() EvidenceRepository
	EvidenceLinks() EvidenceLinkRepository
	Decisions() DecisionRepository
}

type PlanningUnitOfWork interface {
	Roadmaps() RoadmapRepository
	RoadmapActivations() RoadmapActivationStore
}
