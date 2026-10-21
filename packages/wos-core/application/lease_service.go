package application

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func (s *Service) RenewWorkItemLease(
	ctx context.Context,
	commandContext domain.CommandContext,
	cmd RenewWorkItemLeaseCommand,
) (MutationResult[domain.WorkItem], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if err := requireActiveOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		item, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		now, err := s.transactionTime(ctx, uow)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		authority, err := s.resolveLeaseCredentialAuthority(ctx, uow, commandContext, cmd.Scope, item.CurrentLease, "RenewWorkItemLease", now)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		candidate := item
		candidate.Lifecycle = domain.WorkItemLifecycleTodo
		candidate.CurrentLease = nil
		if err := requireWorkItemReady(ctx, uow, candidate, now); err != nil {
			return domain.WorkItem{}, 0, err
		}
		if authority != nil {
			err = item.RenewLeaseForCredential(commandContext.PrincipalID, authority.holderID, cmd.ClaimID, cmd.FencingToken, cmd.TTL, now)
		} else {
			err = item.RenewLease(commandContext.PrincipalID, cmd.ClaimID, cmd.FencingToken, cmd.TTL, now)
		}
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := uow.WorkItems().Save(ctx, item, cmd.ExpectedVersion); err != nil {
			return domain.WorkItem{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return item, revision, err
	})
}

func (s *Service) ReclaimWorkItem(
	ctx context.Context,
	commandContext domain.CommandContext,
	cmd ReclaimWorkItemCommand,
) (MutationResult[domain.WorkItem], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	claimID, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if err := requireActiveOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		item, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		now, err := s.transactionTime(ctx, uow)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		authority, err := s.resolveLeaseCredentialAuthority(ctx, uow, commandContext, cmd.Scope, item.CurrentLease, "ReclaimWorkItem", now)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		candidate := item
		candidate.Lifecycle = domain.WorkItemLifecycleTodo
		candidate.CurrentLease = nil
		if err := requireWorkItemReady(ctx, uow, candidate, now); err != nil {
			return domain.WorkItem{}, 0, err
		}
		if authority != nil {
			err = item.ReclaimForCredential(claimID, commandContext.PrincipalID, authority.holderID, authority.sessionID, commandContext.Actor, cmd.TTL, now)
		} else {
			err = item.Reclaim(claimID, commandContext.PrincipalID, commandContext.Actor, cmd.TTL, now)
		}
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := uow.WorkItems().Save(ctx, item, cmd.ExpectedVersion); err != nil {
			return domain.WorkItem{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return item, revision, err
	})
}
