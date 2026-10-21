package application

import (
	"context"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// TaskDeliveryDetail is a scoped read of one immutable native TaskDelivery and
// its exact append-only resolution, if one exists. It does not fold delivery,
// review decision, withdrawal, or Task acceptance into a synthetic status.
type TaskDeliveryDetail struct {
	Delivery        d.TaskDelivery            `json:"delivery"`
	Decision        *d.TaskDeliveryDecision   `json:"decision,omitempty"`
	Withdrawal      *d.TaskDeliveryWithdrawal `json:"withdrawal,omitempty"`
	TaskVersion     d.Version                 `json:"task_version,string"`
	OutcomeRevision d.OutcomeRevision         `json:"outcome_revision,string"`
	Pending         bool                      `json:"pending"`
	Latest          bool                      `json:"latest"`
}

// GetTaskDelivery reads a native TaskDelivery inside the same authenticated,
// scoped snapshot as the Task and Outcome revision that navigate it. Signed
// review submissions remain available through their own protocol queries.
func (s *Service) GetTaskDelivery(ctx context.Context, scope d.Scope, deliveryID d.ID) (ReadResult[TaskDeliveryDetail], error) {
	if err := scope.Validate(); err != nil {
		return ReadResult[TaskDeliveryDetail]{}, err
	}
	if deliveryID.Validate() != nil {
		return ReadResult[TaskDeliveryDetail]{}, d.NewError(d.ErrorCodeInvalidArgument, "invalid TaskDelivery ID")
	}

	frame, handled, err := s.openWorkspaceFrame(ctx, scope)
	if err != nil {
		return ReadResult[TaskDeliveryDetail]{}, err
	}
	if !handled {
		return ReadResult[TaskDeliveryDetail]{}, d.NewError(d.ErrorCodeInvalidConfig, "coherent TaskDelivery reads are unavailable")
	}
	defer frame.u.Rollback()

	deliveryUOW, ok := frame.u.(ports.TaskDeliveryUnitOfWork)
	if !ok {
		return ReadResult[TaskDeliveryDetail]{}, d.NewError(d.ErrorCodeInvalidConfig, "TaskDelivery persistence is unavailable")
	}
	repository := deliveryUOW.TaskDeliveries()
	delivery, err := repository.Get(ctx, scope, deliveryID)
	if err != nil {
		return ReadResult[TaskDeliveryDetail]{}, err
	}
	if delivery.ID != deliveryID || delivery.Scope != scope || delivery.Validate() != nil {
		return ReadResult[TaskDeliveryDetail]{}, d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery source does not match the requested scope")
	}

	task, err := frame.u.WorkItems().Get(ctx, scope, delivery.WorkItemID)
	if err != nil {
		return ReadResult[TaskDeliveryDetail]{}, err
	}
	if task.ID != delivery.WorkItemID || task.Scope != scope || task.Validate() != nil {
		return ReadResult[TaskDeliveryDetail]{}, d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery navigation does not match its Task")
	}

	decision, err := repository.DecisionForDelivery(ctx, scope, delivery.ID)
	if err != nil {
		return ReadResult[TaskDeliveryDetail]{}, err
	}
	withdrawal, err := repository.WithdrawalForDelivery(ctx, scope, delivery.ID)
	if err != nil {
		return ReadResult[TaskDeliveryDetail]{}, err
	}
	if decision != nil && (decision.ValidateForDelivery(delivery) != nil || decision.Scope != scope || decision.WorkItemID != task.ID) {
		return ReadResult[TaskDeliveryDetail]{}, d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery decision does not match its exact delivery")
	}
	if withdrawal != nil && (withdrawal.ValidateForDelivery(delivery) != nil || withdrawal.Scope != scope || withdrawal.WorkItemID != task.ID) {
		return ReadResult[TaskDeliveryDetail]{}, d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery withdrawal does not match its exact delivery")
	}
	if decision != nil && withdrawal != nil {
		return ReadResult[TaskDeliveryDetail]{}, d.NewError(d.ErrorCodeInvalidConfig, "TaskDelivery has conflicting immutable resolutions")
	}

	latest := task.LatestDeliveryID != nil && *task.LatestDeliveryID == delivery.ID
	if delivery.Round > task.DeliveryRound || (delivery.Round == task.DeliveryRound) != latest {
		return ReadResult[TaskDeliveryDetail]{}, d.NewError(d.ErrorCodeInvalidScope, "TaskDelivery round differs from current Task navigation")
	}
	pending := task.PendingDeliveryID != nil && *task.PendingDeliveryID == delivery.ID
	if latest {
		if task.DeliveryRound != delivery.Round {
			return ReadResult[TaskDeliveryDetail]{}, d.NewError(d.ErrorCodeInvalidScope, "latest TaskDelivery round differs from its Task")
		}
		switch {
		case pending:
			if decision != nil || withdrawal != nil || task.LatestDeliveryDecisionID != nil || task.LatestDeliveryWithdrawalID != nil {
				return ReadResult[TaskDeliveryDetail]{}, d.NewError(d.ErrorCodeInvalidScope, "pending TaskDelivery has an immutable resolution")
			}
		case decision != nil:
			if task.LatestDeliveryDecisionID == nil || *task.LatestDeliveryDecisionID != decision.ID || task.LatestDeliveryWithdrawalID != nil {
				return ReadResult[TaskDeliveryDetail]{}, d.NewError(d.ErrorCodeInvalidScope, "Task decision pointer differs from its exact delivery resolution")
			}
		case withdrawal != nil:
			if task.LatestDeliveryWithdrawalID == nil || *task.LatestDeliveryWithdrawalID != withdrawal.ID || task.LatestDeliveryDecisionID != nil {
				return ReadResult[TaskDeliveryDetail]{}, d.NewError(d.ErrorCodeInvalidScope, "Task withdrawal pointer differs from its exact delivery resolution")
			}
		default:
			return ReadResult[TaskDeliveryDetail]{}, d.NewError(d.ErrorCodeInvalidScope, "latest TaskDelivery has no pending or resolved navigation")
		}
	} else if pending || (decision == nil && withdrawal == nil) {
		return ReadResult[TaskDeliveryDetail]{}, d.NewError(d.ErrorCodeInvalidScope, "historical TaskDelivery is not resolved")
	}

	value := TaskDeliveryDetail{
		Delivery: delivery, Decision: decision, Withdrawal: withdrawal,
		TaskVersion: task.Version, OutcomeRevision: frame.revision,
		Pending: pending, Latest: latest,
	}
	return ReadResult[TaskDeliveryDetail]{Value: value, OutcomeRevision: frame.revision}, nil
}
