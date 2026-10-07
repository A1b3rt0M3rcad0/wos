package application

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type WorkItemOperationalReadResult struct {
	State           domain.WorkItemOperationalState `json:"state"`
	OutcomeRevision domain.OutcomeRevision          `json:"outcome_revision"`
}

func (s *Service) GetWorkItemOperationalState(
	ctx context.Context,
	scope domain.Scope,
	workItemID domain.ID,
) (WorkItemOperationalReadResult, error) {
	if err := s.authorizeRead(ctx, scope.NamespaceID); err != nil {
		return WorkItemOperationalReadResult{}, err
	}

	if err := scope.Validate(); err != nil {
		return WorkItemOperationalReadResult{}, err
	}
	if err := workItemID.Validate(); err != nil {
		return WorkItemOperationalReadResult{}, err
	}

	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return WorkItemOperationalReadResult{}, err
	}
	defer uow.Rollback()

	outcome, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		return WorkItemOperationalReadResult{}, err
	}
	item, err := uow.WorkItems().Get(ctx, scope, workItemID)
	if err != nil {
		return WorkItemOperationalReadResult{}, err
	}

	var objective *domain.Objective
	if item.ObjectiveID != nil {
		value, err := uow.Objectives().Get(ctx, scope, *item.ObjectiveID)
		if err != nil {
			return WorkItemOperationalReadResult{}, err
		}
		objective = &value
	}

	relations, err := uow.Relations().ListByOutcome(ctx, scope)
	if err != nil {
		return WorkItemOperationalReadResult{}, err
	}
	dependencies, err := dependencyEvaluationsForSource(ctx, uow, item.Ref(), relations)
	if err != nil {
		return WorkItemOperationalReadResult{}, err
	}
	blocking, err := blockingStateForRef(ctx, uow, item.Ref())
	if err != nil {
		return WorkItemOperationalReadResult{}, err
	}
	coordination, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return WorkItemOperationalReadResult{}, err
	}

	evaluatedAt, err := s.transactionTime(ctx, uow)
	if err != nil {
		return WorkItemOperationalReadResult{}, err
	}

	return WorkItemOperationalReadResult{
		State: domain.ProjectWorkItemOperationalState(
			item,
			outcome,
			objective,
			dependencies,
			blocking.IsBlocked,
			evaluatedAt,
		),
		OutcomeRevision: coordination.Revision,
	}, nil
}
