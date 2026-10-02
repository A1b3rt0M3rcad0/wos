package ports

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
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
	Coordination() CoordinationStore
	Commit() error
	Rollback() error
}

type TransactionManager interface {
	Begin(ctx context.Context) (UnitOfWork, error)
}
