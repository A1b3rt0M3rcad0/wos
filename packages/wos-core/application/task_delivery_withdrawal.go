package application

import (
	"context"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// WithdrawTaskDeliveryCommand withdraws only the exact current unsigned
// delivery awaiting review. It does not record a review decision or return the
// Task to Backlog; callers must explicitly defer the Task before changing its
// definition or planning another execution.
type WithdrawTaskDeliveryCommand struct {
	Scope                   domain.Scope   `json:"scope"`
	WorkItemID              domain.ID      `json:"work_item_id"`
	ExpectedWorkItemVersion domain.Version `json:"expected_work_item_version,string"`
	DeliveryID              domain.ID      `json:"delivery_id"`
	ExpectedDeliveryDigest  string         `json:"expected_delivery_digest"`
	ExpectedDeliveryRound   uint64         `json:"expected_delivery_round,string"`
	Reason                  string         `json:"reason"`
}

type TaskDeliveryWithdrawalResult struct {
	WorkItem    domain.WorkItem               `json:"work_item"`
	Delivery    domain.TaskDelivery           `json:"delivery"`
	Withdrawal  domain.TaskDeliveryWithdrawal `json:"withdrawal"`
	EvaluatedAt time.Time                     `json:"evaluated_at"`
}

func (s *Service) WithdrawTaskDelivery(ctx context.Context, cc domain.CommandContext, cmd WithdrawTaskDeliveryCommand) (MutationResult[TaskDeliveryWithdrawalResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[TaskDeliveryWithdrawalResult]{}, err
	}
	if strings.TrimSpace(cc.IdempotencyKey) == "" {
		return MutationResult[TaskDeliveryWithdrawalResult]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "TaskDelivery withdrawal requires a non-empty idempotency key")
	}
	if err := validateTaskDeliveryWithdrawalCommand(cmd); err != nil {
		return MutationResult[TaskDeliveryWithdrawalResult]{}, err
	}
	if err := boundedWorkspaceCommand(cmd); err != nil {
		return MutationResult[TaskDeliveryWithdrawalResult]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (TaskDeliveryWithdrawalResult, domain.OutcomeRevision, error) {
		var zero TaskDeliveryWithdrawalResult
		if err := requireActiveOutcome(ctx, u, cmd.Scope); err != nil {
			return zero, 0, err
		}
		if _, err := u.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return zero, 0, err
		}
		deliveryUOW, ok := u.(ports.TaskDeliveryUnitOfWork)
		if !ok {
			return zero, 0, domain.NewError(domain.ErrorCodeInvalidConfig, "adapter lacks TaskDelivery persistence")
		}
		w, err := u.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return zero, 0, err
		}
		if w.Version != cmd.ExpectedWorkItemVersion {
			return zero, 0, domain.NewError(domain.ErrorCodeVersionConflict, "Task version conflict")
		}
		delivery, err := deliveryUOW.TaskDeliveries().Get(ctx, cmd.Scope, cmd.DeliveryID)
		if err != nil {
			return zero, 0, err
		}
		if delivery.WorkItemID != cmd.WorkItemID || delivery.Digest != cmd.ExpectedDeliveryDigest || delivery.Round != cmd.ExpectedDeliveryRound {
			return zero, 0, domain.NewError(domain.ErrorCodeVersionConflict, "delivery digest, round, or Task binding changed")
		}
		pending, err := deliveryUOW.TaskDeliveries().Pending(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return zero, 0, err
		}
		if pending == nil || pending.ID != delivery.ID || pending.Digest != delivery.Digest || pending.Round != delivery.Round {
			return zero, 0, domain.NewError(domain.ErrorCodeSubmissionNotAccepted, "delivery is no longer the current pending review")
		}
		if err := delivery.ValidateForWork(w); err != nil {
			return zero, 0, err
		}
		decision, err := deliveryUOW.TaskDeliveries().DecisionForDelivery(ctx, cmd.Scope, delivery.ID)
		if err != nil {
			return zero, 0, err
		}
		if decision != nil {
			return zero, 0, domain.NewError(domain.ErrorCodeSubmissionNotAccepted, "a decided TaskDelivery cannot be withdrawn")
		}
		now, err := s.transactionTime(ctx, u)
		if err != nil {
			return zero, 0, err
		}
		identity, authenticated := IdentityFromContext(ctx)
		if !authenticated || identity.CredentialID.Validate() != nil || identity.CredentialDigest == "" || identity.NamespaceID != cmd.Scope.NamespaceID || identity.PrincipalID != cc.PrincipalID || identity.Actor != cc.Actor {
			return zero, 0, domain.NewError(domain.ErrorCodeForbidden, "authenticated credential is required to withdraw a TaskDelivery")
		}
		authority, err := s.resolveLeaseCredentialAuthority(ctx, u, cc, cmd.Scope, nil, "WithdrawTaskDelivery", now)
		if err != nil {
			return zero, 0, err
		}
		if authority == nil {
			return zero, 0, domain.NewError(domain.ErrorCodeForbidden, "authenticated credential could not be resolved for TaskDelivery withdrawal")
		}
		priorWithdrawal, err := deliveryUOW.TaskDeliveries().WithdrawalForDelivery(ctx, cmd.Scope, delivery.ID)
		if err != nil {
			return zero, 0, err
		}
		if priorWithdrawal != nil {
			return zero, 0, domain.NewError(domain.ErrorCodeSubmissionNotAccepted, "TaskDelivery has already been withdrawn")
		}
		withdrawalID, err := s.ids.NewID()
		if err != nil {
			return zero, 0, err
		}
		withdrawal := domain.TaskDeliveryWithdrawal{
			ID: withdrawalID, Scope: cmd.Scope, WorkItemID: cmd.WorkItemID,
			DeliveryID: delivery.ID, DeliveryDigest: delivery.Digest, Round: delivery.Round,
			PrincipalID: cc.PrincipalID, Actor: cc.Actor, Reason: strings.TrimSpace(cmd.Reason), WithdrawnAt: now,
		}
		before := w.Version
		if err := w.WithdrawTaskDelivery(delivery, withdrawal, before, now); err != nil {
			return zero, 0, err
		}
		if err := deliveryUOW.TaskDeliveries().InsertWithdrawal(ctx, withdrawal); err != nil {
			return zero, 0, err
		}
		if err := u.WorkItems().Save(ctx, w, before); err != nil {
			return zero, 0, err
		}
		revision, err := u.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return TaskDeliveryWithdrawalResult{WorkItem: w, Delivery: delivery, Withdrawal: withdrawal, EvaluatedAt: now}, revision, err
	})
}

func validateTaskDeliveryWithdrawalCommand(cmd WithdrawTaskDeliveryCommand) error {
	if err := cmd.Scope.Validate(); err != nil {
		return err
	}
	if err := cmd.WorkItemID.Validate(); err != nil {
		return err
	}
	if err := cmd.ExpectedWorkItemVersion.Validate(); err != nil {
		return err
	}
	if err := cmd.DeliveryID.Validate(); err != nil {
		return err
	}
	if cmd.ExpectedDeliveryRound == 0 || !domain.ValidSignedDigest(cmd.ExpectedDeliveryDigest) || strings.TrimSpace(cmd.Reason) == "" || len(cmd.Reason) > 4096 {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "TaskDelivery withdrawal requires an exact delivery digest and round plus a bounded reason")
	}
	return nil
}
