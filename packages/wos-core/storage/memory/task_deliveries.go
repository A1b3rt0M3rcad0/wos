package memory

import (
	"context"
	"sort"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func cloneTaskDeliveryMap(src map[string]d.TaskDelivery) map[string]d.TaskDelivery {
	out := make(map[string]d.TaskDelivery, len(src))
	for key, value := range src {
		out[key] = contractCopy(value)
	}
	return out
}

func cloneTaskDeliveryDecisionMap(src map[string]d.TaskDeliveryDecision) map[string]d.TaskDeliveryDecision {
	out := make(map[string]d.TaskDeliveryDecision, len(src))
	for key, value := range src {
		out[key] = contractCopy(value)
	}
	return out
}

func cloneTaskDeliveryWithdrawalMap(src map[string]d.TaskDeliveryWithdrawal) map[string]d.TaskDeliveryWithdrawal {
	out := make(map[string]d.TaskDeliveryWithdrawal, len(src))
	for key, value := range src {
		out[key] = value
	}
	return out
}

type taskDeliveryRepository struct{ tx *transaction }

func validateTaskDeliveryNavigation(item d.WorkItem, tx *transaction) error {
	if item.DeliveryRound == 0 && (item.PendingDeliveryID != nil || item.LatestDeliveryID != nil || item.LatestDeliveryDecisionID != nil || item.LatestDeliveryWithdrawalID != nil || item.TaskDeliveryReplanAuthorized) {
		return d.NewError(d.ErrorCodeInvalidScope, "WorkItem TaskDelivery navigation is inconsistent")
	}
	if item.PendingDeliveryID != nil {
		delivery, ok := tx.taskDeliveries[entityKey(item.Scope, *item.PendingDeliveryID)]
		if !ok || delivery.WorkItemID != item.ID || delivery.Round != item.DeliveryRound {
			return d.NewError(d.ErrorCodeInvalidScope, "WorkItem pending delivery reference is missing or mismatched")
		}
		if item.LatestDeliveryID == nil || *item.LatestDeliveryID != delivery.ID || item.LatestDeliveryDecisionID != nil || item.LatestDeliveryWithdrawalID != nil {
			return d.NewError(d.ErrorCodeInvalidScope, "WorkItem pending delivery is not its unresolved latest delivery")
		}
		for _, decision := range tx.taskDeliveryDecisions {
			if decision.Scope == item.Scope && decision.WorkItemID == item.ID && decision.DeliveryID == delivery.ID {
				return d.NewError(d.ErrorCodeInvalidScope, "WorkItem pending delivery already has a decision")
			}
		}
		for _, withdrawal := range tx.taskDeliveryWithdrawals {
			if withdrawal.Scope == item.Scope && withdrawal.WorkItemID == item.ID && withdrawal.DeliveryID == delivery.ID {
				return d.NewError(d.ErrorCodeInvalidScope, "WorkItem pending delivery already has a withdrawal")
			}
		}
	}
	if item.LatestDeliveryID != nil {
		delivery, ok := tx.taskDeliveries[entityKey(item.Scope, *item.LatestDeliveryID)]
		if !ok || delivery.WorkItemID != item.ID || delivery.Round != item.DeliveryRound {
			return d.NewError(d.ErrorCodeInvalidScope, "WorkItem latest delivery reference is missing or mismatched")
		}
		if item.PendingDeliveryID == nil && item.LatestDeliveryDecisionID != nil && item.LatestDeliveryWithdrawalID == nil {
			decision, ok := tx.taskDeliveryDecisions[entityKey(item.Scope, *item.LatestDeliveryDecisionID)]
			if !ok || decision.WorkItemID != item.ID || decision.DeliveryID != delivery.ID {
				return d.NewError(d.ErrorCodeInvalidScope, "WorkItem latest delivery decision reference is missing or mismatched")
			}
			if withdrawal, err := (taskDeliveryRepository{tx: tx}).WithdrawalForDelivery(context.Background(), item.Scope, delivery.ID); err != nil || withdrawal != nil {
				return d.NewError(d.ErrorCodeInvalidScope, "WorkItem delivery has both a decision and a withdrawal")
			}
		} else if item.PendingDeliveryID == nil && item.LatestDeliveryDecisionID == nil && item.LatestDeliveryWithdrawalID != nil {
			withdrawal, ok := tx.taskDeliveryWithdrawals[entityKey(item.Scope, *item.LatestDeliveryWithdrawalID)]
			if !ok || withdrawal.WorkItemID != item.ID || withdrawal.DeliveryID != delivery.ID || withdrawal.Round != delivery.Round || withdrawal.DeliveryDigest != delivery.Digest || withdrawal.ValidateForDelivery(delivery) != nil {
				return d.NewError(d.ErrorCodeInvalidScope, "WorkItem latest delivery withdrawal reference is missing or mismatched")
			}
			if decision, err := (taskDeliveryRepository{tx: tx}).DecisionForDelivery(context.Background(), item.Scope, delivery.ID); err != nil || decision != nil {
				return d.NewError(d.ErrorCodeInvalidScope, "WorkItem delivery has both a decision and a withdrawal")
			}
		} else if item.PendingDeliveryID == nil {
			return d.NewError(d.ErrorCodeInvalidScope, "WorkItem resolved delivery is missing its exact decision or withdrawal reference")
		}
	} else if item.LatestDeliveryDecisionID != nil || item.LatestDeliveryWithdrawalID != nil {
		return d.NewError(d.ErrorCodeInvalidScope, "WorkItem resolution reference has no latest delivery")
	}
	return nil
}

func (r taskDeliveryRepository) Insert(ctx context.Context, delivery d.TaskDelivery) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := delivery.Validate(); err != nil {
		return err
	}
	work, err := r.tx.workItemRepo.Get(ctx, delivery.Scope, delivery.WorkItemID)
	if err != nil {
		return err
	}
	if err := delivery.ValidateForWork(work); err != nil {
		return err
	}
	if !sameOptionalTaskDeliveryID(delivery.SupersedesDeliveryID, work.LatestDeliveryID) {
		return d.NewError(d.ErrorCodeInvalidTransition, "TaskDelivery must supersede the exact latest delivery")
	}
	if work.PendingDeliveryID != nil || delivery.Round != work.DeliveryRound+1 {
		return d.NewError(d.ErrorCodeInvalidTransition, "Task already has a pending delivery or delivery round is stale")
	}
	key := entityKey(delivery.Scope, delivery.ID)
	if _, exists := r.tx.taskDeliveries[key]; exists {
		return d.NewError(d.ErrorCodeAlreadyExists, "TaskDelivery already exists")
	}
	for _, prior := range r.tx.taskDeliveries {
		if prior.Scope == delivery.Scope && prior.WorkItemID == delivery.WorkItemID && prior.Round == delivery.Round {
			return d.NewError(d.ErrorCodeAlreadyExists, "TaskDelivery round already exists")
		}
	}
	if delivery.SubmissionRef != nil {
		if delivery.Authority.ContractID == nil || delivery.SubmissionRef.ContractID != *delivery.Authority.ContractID {
			return d.NewError(d.ErrorCodeContractSpecMismatch, "delivery submission does not reference its authorizing contract")
		}
		submission, err := r.tx.WorkContracts().GetSubmission(ctx, delivery.Scope, delivery.SubmissionRef.ID)
		if err != nil {
			return err
		}
		if submission.Material.WorkItemID != delivery.WorkItemID || submission.Material.ContractID != delivery.SubmissionRef.ContractID || submission.Digest != delivery.SubmissionRef.Digest {
			return d.NewError(d.ErrorCodeContractSpecMismatch, "delivery does not reference the exact scoped WorkSubmission")
		}
	} else if delivery.Authority.Kind != d.TaskDeliveryAuthorityLegacyLease || delivery.Material == nil {
		return d.NewError(d.ErrorCodeInvalidArgument, "inline delivery material is only valid for a legacy lease")
	}
	r.tx.taskDeliveries[key] = contractCopy(delivery)
	return nil
}

func sameOptionalTaskDeliveryID(left, right *d.ID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

func (r taskDeliveryRepository) Get(ctx context.Context, scope d.Scope, id d.ID) (d.TaskDelivery, error) {
	if err := ctx.Err(); err != nil {
		return d.TaskDelivery{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return d.TaskDelivery{}, err
	}
	value, ok := r.tx.taskDeliveries[entityKey(scope, id)]
	if !ok {
		return d.TaskDelivery{}, d.NewError(d.ErrorCodeNotFound, "TaskDelivery not found")
	}
	value = contractCopy(value)
	return value, value.Validate()
}

func (r taskDeliveryRepository) Pending(ctx context.Context, scope d.Scope, workItemID d.ID) (*d.TaskDelivery, error) {
	work, err := r.tx.workItemRepo.Get(ctx, scope, workItemID)
	if err != nil {
		return nil, err
	}
	if work.PendingDeliveryID == nil {
		return nil, nil
	}
	delivery, err := r.Get(ctx, scope, *work.PendingDeliveryID)
	if err != nil {
		return nil, err
	}
	if delivery.WorkItemID != workItemID || work.LatestDeliveryID == nil || *work.LatestDeliveryID != delivery.ID || work.DeliveryRound != delivery.Round {
		return nil, d.NewError(d.ErrorCodeInvalidScope, "pending TaskDelivery navigation differs from its WorkItem")
	}
	return &delivery, nil
}

func (r taskDeliveryRepository) List(ctx context.Context, scope d.Scope, filter ports.TaskDeliveryFilter) ([]d.TaskDelivery, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	if filter.Limit < 1 || filter.Limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "TaskDelivery limit outside 1..101")
	}
	values := make([]d.TaskDelivery, 0)
	for _, stored := range r.tx.taskDeliveries {
		if stored.Scope != scope || stored.ID <= filter.After || (filter.WorkItemID != "" && stored.WorkItemID != filter.WorkItemID) {
			continue
		}
		if filter.Pending {
			work, err := r.tx.workItemRepo.Get(ctx, scope, stored.WorkItemID)
			if err != nil {
				return nil, err
			}
			if work.PendingDeliveryID == nil || *work.PendingDeliveryID != stored.ID {
				continue
			}
		}
		if err := stored.Validate(); err != nil {
			return nil, err
		}
		values = append(values, contractCopy(stored))
	}
	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	if len(values) > filter.Limit {
		values = values[:filter.Limit]
	}
	return values, nil
}

func (r taskDeliveryRepository) InsertDecision(ctx context.Context, decision d.TaskDeliveryDecision) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := decision.Validate(); err != nil {
		return err
	}
	delivery, err := r.Get(ctx, decision.Scope, decision.DeliveryID)
	if err != nil {
		return err
	}
	if err := decision.ValidateForDelivery(delivery); err != nil {
		return err
	}
	work, err := r.tx.workItemRepo.Get(ctx, decision.Scope, decision.WorkItemID)
	if err != nil {
		return err
	}
	if work.PendingDeliveryID == nil || *work.PendingDeliveryID != delivery.ID || work.LatestDeliveryID == nil || *work.LatestDeliveryID != delivery.ID {
		return d.NewError(d.ErrorCodeSubmissionNotAccepted, "decision target is not the pending TaskDelivery")
	}
	withdrawal, err := r.WithdrawalForDelivery(ctx, decision.Scope, delivery.ID)
	if err != nil {
		return err
	}
	if withdrawal != nil {
		return d.NewError(d.ErrorCodeSubmissionNotAccepted, "TaskDelivery has already been withdrawn")
	}
	key := entityKey(decision.Scope, decision.ID)
	if _, exists := r.tx.taskDeliveryDecisions[key]; exists {
		return d.NewError(d.ErrorCodeAlreadyExists, "TaskDeliveryDecision already exists")
	}
	for _, prior := range r.tx.taskDeliveryDecisions {
		if prior.Scope == decision.Scope && prior.WorkItemID == decision.WorkItemID && prior.DeliveryID == decision.DeliveryID {
			return d.NewError(d.ErrorCodeAlreadyExists, "TaskDelivery already has a decision")
		}
	}
	r.tx.taskDeliveryDecisions[key] = contractCopy(decision)
	return nil
}

func (r taskDeliveryRepository) GetDecision(ctx context.Context, scope d.Scope, id d.ID) (d.TaskDeliveryDecision, error) {
	if err := ctx.Err(); err != nil {
		return d.TaskDeliveryDecision{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return d.TaskDeliveryDecision{}, err
	}
	value, ok := r.tx.taskDeliveryDecisions[entityKey(scope, id)]
	if !ok {
		return d.TaskDeliveryDecision{}, d.NewError(d.ErrorCodeNotFound, "TaskDeliveryDecision not found")
	}
	value = contractCopy(value)
	delivery, err := r.Get(ctx, scope, value.DeliveryID)
	if err != nil {
		return value, err
	}
	return value, value.ValidateForDelivery(delivery)
}

func (r taskDeliveryRepository) InsertWithdrawal(ctx context.Context, withdrawal d.TaskDeliveryWithdrawal) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := withdrawal.Validate(); err != nil {
		return err
	}
	delivery, err := r.Get(ctx, withdrawal.Scope, withdrawal.DeliveryID)
	if err != nil {
		return err
	}
	if err := withdrawal.ValidateForDelivery(delivery); err != nil {
		return err
	}
	work, err := r.tx.workItemRepo.Get(ctx, withdrawal.Scope, withdrawal.WorkItemID)
	if err != nil {
		return err
	}
	if work.PendingDeliveryID == nil || *work.PendingDeliveryID != delivery.ID || work.LatestDeliveryID == nil || *work.LatestDeliveryID != delivery.ID {
		return d.NewError(d.ErrorCodeSubmissionNotAccepted, "withdrawal target is not the pending TaskDelivery")
	}
	decision, err := r.DecisionForDelivery(ctx, withdrawal.Scope, delivery.ID)
	if err != nil {
		return err
	}
	if decision != nil {
		return d.NewError(d.ErrorCodeSubmissionNotAccepted, "TaskDelivery already has a decision")
	}
	for _, existing := range r.tx.taskDeliveryWithdrawals {
		if existing.Scope == withdrawal.Scope && existing.WorkItemID == withdrawal.WorkItemID && existing.DeliveryID == delivery.ID {
			return d.NewError(d.ErrorCodeAlreadyExists, "TaskDelivery already has a withdrawal")
		}
	}
	key := entityKey(withdrawal.Scope, withdrawal.ID)
	if _, exists := r.tx.taskDeliveryWithdrawals[key]; exists {
		return d.NewError(d.ErrorCodeAlreadyExists, "TaskDeliveryWithdrawal already exists")
	}
	r.tx.taskDeliveryWithdrawals[key] = withdrawal
	return nil
}

func (r taskDeliveryRepository) GetWithdrawal(ctx context.Context, scope d.Scope, id d.ID) (d.TaskDeliveryWithdrawal, error) {
	if err := ctx.Err(); err != nil {
		return d.TaskDeliveryWithdrawal{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return d.TaskDeliveryWithdrawal{}, err
	}
	value, ok := r.tx.taskDeliveryWithdrawals[entityKey(scope, id)]
	if !ok {
		return d.TaskDeliveryWithdrawal{}, d.NewError(d.ErrorCodeNotFound, "TaskDeliveryWithdrawal not found")
	}
	delivery, err := r.Get(ctx, scope, value.DeliveryID)
	if err != nil {
		return value, err
	}
	return value, value.ValidateForDelivery(delivery)
}

func (r taskDeliveryRepository) WithdrawalForDelivery(ctx context.Context, scope d.Scope, deliveryID d.ID) (*d.TaskDeliveryWithdrawal, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	for _, withdrawal := range r.tx.taskDeliveryWithdrawals {
		if withdrawal.Scope == scope && withdrawal.DeliveryID == deliveryID {
			value, err := r.GetWithdrawal(ctx, scope, withdrawal.ID)
			if err != nil {
				return nil, err
			}
			return &value, nil
		}
	}
	return nil, nil
}

func (r taskDeliveryRepository) DecisionForDelivery(ctx context.Context, scope d.Scope, deliveryID d.ID) (*d.TaskDeliveryDecision, error) {
	for _, decision := range r.tx.taskDeliveryDecisions {
		if decision.Scope == scope && decision.DeliveryID == deliveryID {
			value, err := r.GetDecision(ctx, scope, decision.ID)
			return &value, err
		}
	}
	return nil, nil
}

func (r taskDeliveryRepository) ListDecisions(ctx context.Context, scope d.Scope, deliveryID, after d.ID, limit int) ([]d.TaskDeliveryDecision, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "TaskDeliveryDecision limit outside 1..101")
	}
	values := make([]d.TaskDeliveryDecision, 0)
	for _, decision := range r.tx.taskDeliveryDecisions {
		if decision.Scope == scope && decision.DeliveryID == deliveryID && decision.ID > after {
			value, err := r.GetDecision(ctx, scope, decision.ID)
			if err != nil {
				return nil, err
			}
			values = append(values, value)
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}
