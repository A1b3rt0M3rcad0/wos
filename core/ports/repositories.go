package ports

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

type OutcomeRepository interface {
	Get(ctx context.Context, namespaceID, outcomeID domain.ID) (domain.Outcome, error)
	Insert(ctx context.Context, outcome domain.Outcome) error
	Save(ctx context.Context, outcome domain.Outcome, expected domain.Version) error
}

type ObjectiveRepository interface {
	Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Objective, error)
	ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.Objective, error)
	Insert(ctx context.Context, objective domain.Objective) error
	Save(ctx context.Context, objective domain.Objective, expected domain.Version) error
}

type WorkItemRepository interface {
	Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.WorkItem, error)
	ListByOutcome(ctx context.Context, scope domain.Scope) ([]domain.WorkItem, error)
	Insert(ctx context.Context, item domain.WorkItem) error
	Save(ctx context.Context, item domain.WorkItem, expected domain.Version) error
}
