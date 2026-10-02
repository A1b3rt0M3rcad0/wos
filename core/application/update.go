package application

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/core/ports"
)

type UpdateOutcomeCommand struct {
	Scope           domain.Scope
	ExpectedVersion domain.Version
	Title           *string
	Description     *string
	DesiredState    *string
	Priority        *domain.Priority
}

type UpdateObjectiveCommand struct {
	Scope              domain.Scope
	ObjectiveID        domain.ID
	ExpectedVersion    domain.Version
	Title              *string
	Description        *string
	Priority           *domain.Priority
	RequiredForOutcome *bool
}

type UpdateWorkItemCommand struct {
	Scope           domain.Scope
	WorkItemID      domain.ID
	ExpectedVersion domain.Version
	Title           *string
	Description     *string
	Priority        *domain.Priority
}

func (s *Service) UpdateOutcome(
	ctx context.Context,
	commandContext domain.CommandContext,
	cmd UpdateOutcomeCommand,
) (MutationResult[domain.Outcome], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Outcome]{}, err
	}
	if cmd.Title == nil && cmd.Description == nil && cmd.DesiredState == nil && cmd.Priority == nil {
		return MutationResult[domain.Outcome]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "outcome patch has no editable fields")
	}

	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Outcome, domain.OutcomeRevision, error) {
		coordination, err := uow.Coordination().LockOutcome(ctx, cmd.Scope)
		if err != nil {
			return domain.Outcome{}, 0, err
		}
		outcome, err := uow.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
		if err != nil {
			return domain.Outcome{}, 0, err
		}
		changed, err := outcome.UpdateDetails(cmd.Title, cmd.Description, cmd.DesiredState, cmd.Priority, now)
		if err != nil {
			return domain.Outcome{}, 0, err
		}
		if !changed {
			return outcome, coordination.Revision, nil
		}
		if err := uow.Outcomes().Save(ctx, outcome, cmd.ExpectedVersion); err != nil {
			return domain.Outcome{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return outcome, revision, err
	})
}

func (s *Service) UpdateObjective(
	ctx context.Context,
	commandContext domain.CommandContext,
	cmd UpdateObjectiveCommand,
) (MutationResult[domain.Objective], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Objective]{}, err
	}
	if cmd.Title == nil && cmd.Description == nil && cmd.Priority == nil && cmd.RequiredForOutcome == nil {
		return MutationResult[domain.Objective]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "objective patch has no editable fields")
	}

	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Objective, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Objective{}, 0, err
		}
		coordination, err := uow.Coordination().LockOutcome(ctx, cmd.Scope)
		if err != nil {
			return domain.Objective{}, 0, err
		}
		objective, err := uow.Objectives().Get(ctx, cmd.Scope, cmd.ObjectiveID)
		if err != nil {
			return domain.Objective{}, 0, err
		}
		changed, err := objective.UpdateDetails(cmd.Title, cmd.Description, cmd.Priority, cmd.RequiredForOutcome, now)
		if err != nil {
			return domain.Objective{}, 0, err
		}
		if !changed {
			return objective, coordination.Revision, nil
		}
		if err := uow.Objectives().Save(ctx, objective, cmd.ExpectedVersion); err != nil {
			return domain.Objective{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return objective, revision, err
	})
}

func (s *Service) UpdateWorkItem(
	ctx context.Context,
	commandContext domain.CommandContext,
	cmd UpdateWorkItemCommand,
) (MutationResult[domain.WorkItem], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	if cmd.Title == nil && cmd.Description == nil && cmd.Priority == nil {
		return MutationResult[domain.WorkItem]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "work item patch has no editable fields")
	}

	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		coordination, err := uow.Coordination().LockOutcome(ctx, cmd.Scope)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		item, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		changed, err := item.UpdateDetails(cmd.Title, cmd.Description, cmd.Priority, now)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if !changed {
			return item, coordination.Revision, nil
		}
		if err := uow.WorkItems().Save(ctx, item, cmd.ExpectedVersion); err != nil {
			return domain.WorkItem{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return item, revision, err
	})
}
