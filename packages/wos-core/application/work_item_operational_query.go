package application

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type WorkItemOperationalReadResult struct {
	State           domain.WorkItemOperationalState `json:"state"`
	OutcomeRevision domain.OutcomeRevision          `json:"outcome_revision"`
	Verification    *TaskVerificationView           `json:"verification,omitempty"`
}

func (s *Service) GetWorkItemOperationalState(
	ctx context.Context,
	scope domain.Scope,
	workItemID domain.ID,
) (WorkItemOperationalReadResult, error) {
	if err := s.authorizeScopedRead(ctx, scope); err != nil {
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
	if err := s.authorizeScopedReadInUnitOfWork(ctx, uow, scope); err != nil {
		return WorkItemOperationalReadResult{}, err
	}

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

	state := domain.ProjectWorkItemOperationalState(
		item,
		outcome,
		objective,
		dependencies,
		blocking.IsBlocked,
		evaluatedAt,
	)
	if item.Lifecycle == domain.WorkItemLifecycleBacklog && outcome.OperationalProfile.Effective() == domain.OperationalProfileOperationalV2 {
		// Backlog stays a persisted planning state. Its readiness projection uses
		// activation eligibility so it cannot imply that a structurally blocked
		// Task is ready to move into To Do.
		readiness, err := evaluateTaskActivation(ctx, uow, item, evaluatedAt)
		if err != nil {
			return WorkItemOperationalReadResult{}, err
		}
		state.ReadinessReasons = append([]domain.ReadinessReason(nil), readiness.Reasons...)
	}
	if item.Lifecycle == domain.WorkItemLifecycleTodo {
		// The list/read projection and acquisition capabilities share the same
		// profile-aware evaluator. Keep the existing display-state vocabulary,
		// but expose every current operational reason in the read contract.
		readiness, err := evaluateTaskAcquisition(ctx, uow, item, evaluatedAt)
		if err != nil {
			return WorkItemOperationalReadResult{}, err
		}
		state.ReadinessReasons = append([]domain.ReadinessReason(nil), readiness.Reasons...)
		state.DisplayState = workItemDisplayStateForReadiness(readiness)
	}
	verification, err := s.optionalTaskVerificationAt(ctx, uow, item, coordination.Revision, evaluatedAt)
	if err != nil {
		return WorkItemOperationalReadResult{}, err
	}
	state = applyTaskVerificationOperational(state, verification)
	annotateTaskVerificationReadiness(verification, state)
	return WorkItemOperationalReadResult{
		State:           state,
		OutcomeRevision: coordination.Revision,
		Verification:    verification,
	}, nil
}

func workItemDisplayStateForReadiness(readiness domain.Readiness) domain.WorkItemDisplayState {
	for _, reason := range readiness.Reasons {
		if reason == domain.ReadinessReasonBlocked {
			return domain.WorkItemDisplayStateBlocked
		}
	}
	for _, reason := range readiness.Reasons {
		if reason == domain.ReadinessReasonHardDependencyUnsatisfied {
			return domain.WorkItemDisplayStateWaitingDependencies
		}
	}
	for _, reason := range readiness.Reasons {
		if reason == domain.ReadinessReasonNotBefore {
			return domain.WorkItemDisplayStateScheduled
		}
	}
	for _, reason := range readiness.Reasons {
		switch reason {
		case domain.ReadinessReasonOutcomeNotActive, domain.ReadinessReasonOutcomeArchived,
			domain.ReadinessReasonObjectiveTerminal, domain.ReadinessReasonObjectiveNotStarted,
			domain.ReadinessReasonAncestorObjectiveNotStarted, domain.ReadinessReasonAncestorObjectiveTerminal,
			domain.ReadinessReasonTaskObjectiveRequired, domain.ReadinessReasonObjectiveAncestryMissing,
			domain.ReadinessReasonObjectiveAncestryCycle, domain.ReadinessReasonObjectiveAncestryCorrupt,
			domain.ReadinessReasonObjectiveAncestryIncomplete:
			return domain.WorkItemDisplayStateWaitingScope
		}
	}
	if len(readiness.Reasons) > 0 {
		return domain.WorkItemDisplayStateAttentionNeeded
	}
	return domain.WorkItemDisplayStateReady
}
