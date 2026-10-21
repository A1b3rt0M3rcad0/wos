package application

import (
	"context"
	"sort"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type ReadyWorkItem struct {
	WorkItem  domain.WorkItem  `json:"work_item"`
	Readiness domain.Readiness `json:"readiness"`
}

type ReadyWork struct {
	Items           []ReadyWorkItem        `json:"items"`
	EvaluatedAt     time.Time              `json:"evaluated_at"`
	OutcomeRevision domain.OutcomeRevision `json:"outcome_revision"`
}

func (s *Service) ListReadyWork(ctx context.Context, scope domain.Scope) (ReadyWork, error) {
	if err := s.authorizeScopedRead(ctx, scope); err != nil {
		return ReadyWork{}, err
	}

	if err := scope.Validate(); err != nil {
		return ReadyWork{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadyWork{}, err
	}
	defer uow.Rollback()
	if err := s.authorizeScopedReadInUnitOfWork(ctx, uow, scope); err != nil {
		return ReadyWork{}, err
	}
	// Serialize the readiness snapshot with lifecycle/coordination mutations
	// before loading the Outcome, Tasks or their Objective hierarchy.
	coordination, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return ReadyWork{}, err
	}

	outcome, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		return ReadyWork{}, err
	}
	if err := validateOperationalProfile(outcome); err != nil {
		return ReadyWork{}, err
	}
	workItems, err := uow.WorkItems().ListByOutcome(ctx, scope)
	if err != nil {
		return ReadyWork{}, err
	}

	snapshotTime, err := s.transactionTime(ctx, uow)
	if err != nil {
		return ReadyWork{}, err
	}

	items := make([]ReadyWorkItem, 0)
	for _, item := range workItems {
		readiness, err := evaluateTaskAcquisition(ctx, uow, item, snapshotTime)
		if err != nil {
			return ReadyWork{}, err
		}
		if readiness.Ready {
			items = append(items, ReadyWorkItem{WorkItem: item, Readiness: readiness})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		left, right := items[i].WorkItem, items[j].WorkItem
		if priorityRank(left.Priority) != priorityRank(right.Priority) {
			return priorityRank(left.Priority) < priorityRank(right.Priority)
		}
		if left.CreatedAt.Equal(right.CreatedAt) {
			return left.ID.String() < right.ID.String()
		}
		return left.CreatedAt.Before(right.CreatedAt)
	})

	return ReadyWork{
		Items:           items,
		EvaluatedAt:     snapshotTime,
		OutcomeRevision: coordination.Revision,
	}, nil
}

func priorityRank(value domain.Priority) int {
	switch value {
	case domain.PriorityCritical:
		return 0
	case domain.PriorityHigh:
		return 1
	case domain.PriorityNormal:
		return 2
	case domain.PriorityLow:
		return 3
	default:
		return 4
	}
}
