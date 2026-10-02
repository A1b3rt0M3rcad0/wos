package application

import (
	"context"
	"sort"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
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
	if err := scope.Validate(); err != nil {
		return ReadyWork{}, err
	}
	snapshotTime := s.clock.Now().UTC()
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadyWork{}, err
	}
	defer uow.Rollback()

	outcome, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		return ReadyWork{}, err
	}
	workItems, err := uow.WorkItems().ListByOutcome(ctx, scope)
	if err != nil {
		return ReadyWork{}, err
	}
	objectives, err := uow.Objectives().ListByOutcome(ctx, scope)
	if err != nil {
		return ReadyWork{}, err
	}
	relations, err := uow.Relations().ListByOutcome(ctx, scope)
	if err != nil {
		return ReadyWork{}, err
	}
	coordination, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return ReadyWork{}, err
	}

	objectiveByID := make(map[domain.ID]domain.Objective, len(objectives))
	for _, objective := range objectives {
		objectiveByID[objective.ID] = objective
	}

	items := make([]ReadyWorkItem, 0)
	for _, item := range workItems {
		var objective *domain.Objective
		if item.ObjectiveID != nil {
			value, ok := objectiveByID[*item.ObjectiveID]
			if !ok {
				return ReadyWork{}, domain.NewError(domain.ErrorCodeNotFound, "work item objective is missing")
			}
			copyValue := value
			objective = &copyValue
		}
		evaluations, err := dependencyEvaluationsForSource(ctx, uow, item.Ref(), relations)
		if err != nil {
			return ReadyWork{}, err
		}
		readiness := domain.WorkItemReadiness(item, outcome, objective, evaluations, snapshotTime)
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
