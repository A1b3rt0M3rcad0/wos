package application

import (
	"context"
	"strings"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type AdministrativeCancelWorkItemCommand struct {
	Scope           domain.Scope
	WorkItemID      domain.ID
	ExpectedVersion domain.Version
	Reason          string
}

type AdministrativeCompleteWorkItemCommand struct {
	Scope           domain.Scope
	WorkItemID      domain.ID
	ExpectedVersion domain.Version
	ResultSummary   string
	Reason          string
}

func (s *Service) AdministrativeCancelWorkItem(
	ctx context.Context,
	commandContext domain.CommandContext,
	cmd AdministrativeCancelWorkItemCommand,
) (MutationResult[domain.WorkItem], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	if strings.TrimSpace(cmd.Reason) == "" {
		return MutationResult[domain.WorkItem]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "administrative cancellation reason is required")
	}
	if err := s.authorizer.Authorize(ctx, ports.AuthorizationRequest{
		NamespaceID: cmd.Scope.NamespaceID,
		PrincipalID: commandContext.PrincipalID,
		Permission:  ports.PermissionWorkAdminCancel,
	}); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}

	now := s.clock.Now().UTC()
	conclusionID, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	conclusion := conclusionFromContext(conclusionID, commandContext, cmd.Reason, now)
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		item, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := item.CancelAdministratively(conclusion, now); err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := uow.WorkItems().Save(ctx, item, cmd.ExpectedVersion); err != nil {
			return domain.WorkItem{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return item, revision, err
	})
}

func (s *Service) AdministrativeCompleteWorkItem(
	ctx context.Context,
	commandContext domain.CommandContext,
	cmd AdministrativeCompleteWorkItemCommand,
) (MutationResult[domain.WorkItem], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	if strings.TrimSpace(cmd.Reason) == "" {
		return MutationResult[domain.WorkItem]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "administrative completion reason is required")
	}
	if err := s.authorizer.Authorize(ctx, ports.AuthorizationRequest{
		NamespaceID: cmd.Scope.NamespaceID,
		PrincipalID: commandContext.PrincipalID,
		Permission:  ports.PermissionWorkAdminComplete,
	}); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}

	now := s.clock.Now().UTC()
	conclusionID, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	conclusion := conclusionFromContext(conclusionID, commandContext, cmd.Reason, now)
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		item, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := requireNotBlocked(ctx, uow, item.Ref()); err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := requireHardDependenciesSatisfied(ctx, uow, item.Ref()); err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := item.CompleteAdministratively(cmd.ResultSummary, conclusion, now); err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := uow.WorkItems().Save(ctx, item, cmd.ExpectedVersion); err != nil {
			return domain.WorkItem{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return item, revision, err
	})
}
