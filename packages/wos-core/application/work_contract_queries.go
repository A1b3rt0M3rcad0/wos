package application

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"time"
)

type WorkContractView struct {
	Contract         d.WorkContract      `json:"contract"`
	EffectiveStatus  d.ContractStatus    `json:"effective_status"`
	EvaluatedAt      time.Time           `json:"evaluated_at"`
	ExecutionAllowed bool                `json:"execution_allowed"`
	Recoverable      bool                `json:"recoverable"`
	Reasons          []d.ReadinessReason `json:"reasons"`
}

func (s *Service) GetWorkContract(ctx context.Context, scope d.Scope, id d.ID) (ReadResult[WorkContractView], error) {
	if err := s.authorizeRead(ctx, scope.NamespaceID); err != nil {
		return ReadResult[WorkContractView]{}, err
	}
	if err := scope.Validate(); err != nil {
		return ReadResult[WorkContractView]{}, err
	}
	if err := id.Validate(); err != nil {
		return ReadResult[WorkContractView]{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[WorkContractView]{}, err
	}
	defer uow.Rollback()
	coord, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return ReadResult[WorkContractView]{}, err
	}
	repo, err := contractRepository(uow)
	if err != nil {
		return ReadResult[WorkContractView]{}, err
	}
	c, err := repo.Get(ctx, scope, id)
	if err != nil {
		return ReadResult[WorkContractView]{}, err
	}
	now, err := s.transactionTime(ctx, uow)
	if err != nil {
		return ReadResult[WorkContractView]{}, err
	}
	w, err := uow.WorkItems().Get(ctx, scope, c.WorkItemID)
	if err != nil {
		return ReadResult[WorkContractView]{}, err
	}
	candidate := w
	candidate.CurrentLease = nil
	candidate.Lifecycle = d.WorkItemLifecycleTodo
	ready, err := evaluateWorkItemReadiness(ctx, uow, candidate, now)
	if err != nil {
		return ReadResult[WorkContractView]{}, err
	}
	live := w.CurrentContractID != nil && *w.CurrentContractID == c.ID
	v := WorkContractView{Contract: c, EffectiveStatus: c.EffectiveStatus(now), EvaluatedAt: now, ExecutionAllowed: live && c.ValidAt(now) && ready.Ready, Recoverable: w.Lifecycle == d.WorkItemLifecycleInProgress && !c.ValidAt(now) && ready.Ready, Reasons: ready.Reasons}
	return ReadResult[WorkContractView]{Value: v, OutcomeRevision: coord.Revision}, nil
}
func (s *Service) ListWorkContracts(ctx context.Context, scope d.Scope, f ports.ContractFilter) (ReadResult[[]d.WorkContract], error) {
	if err := s.authorizeRead(ctx, scope.NamespaceID); err != nil {
		return ReadResult[[]d.WorkContract]{}, err
	}
	if err := scope.Validate(); err != nil {
		return ReadResult[[]d.WorkContract]{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[[]d.WorkContract]{}, err
	}
	defer uow.Rollback()
	coord, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return ReadResult[[]d.WorkContract]{}, err
	}
	repo, err := contractRepository(uow)
	if err != nil {
		return ReadResult[[]d.WorkContract]{}, err
	}
	list, err := repo.List(ctx, scope, f)
	return ReadResult[[]d.WorkContract]{Value: list, OutcomeRevision: coord.Revision}, err
}
