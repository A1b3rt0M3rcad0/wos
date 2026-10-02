package application

import (
	"context"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func (s *Service) FailOutcome(ctx context.Context, commandContext domain.CommandContext, cmd FailOutcomeCommand) (MutationResult[domain.Outcome], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Outcome]{}, err
	}
	now := s.clock.Now().UTC()
	conclusion := conclusionFromContext(commandContext, cmd.Reason, now)
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Outcome, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return domain.Outcome{}, 0, err
		}
		outcome, err := uow.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
		if err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := outcome.Fail(conclusion, now); err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := uow.Outcomes().Save(ctx, outcome, cmd.ExpectedVersion); err != nil {
			return domain.Outcome{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return outcome, revision, err
	})
}

func (s *Service) AbandonOutcome(ctx context.Context, commandContext domain.CommandContext, cmd AbandonOutcomeCommand) (MutationResult[domain.Outcome], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Outcome]{}, err
	}
	now := s.clock.Now().UTC()
	conclusion := conclusionFromContext(commandContext, cmd.Reason, now)
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Outcome, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return domain.Outcome{}, 0, err
		}
		outcome, err := uow.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
		if err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := outcome.Abandon(conclusion, now); err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := uow.Outcomes().Save(ctx, outcome, cmd.ExpectedVersion); err != nil {
			return domain.Outcome{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return outcome, revision, err
	})
}

func (s *Service) ReopenOutcome(ctx context.Context, commandContext domain.CommandContext, cmd ReopenOutcomeCommand) (MutationResult[domain.Outcome], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Outcome]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Outcome, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return domain.Outcome{}, 0, err
		}
		outcome, err := uow.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
		if err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := outcome.Reopen(cmd.Reason, now); err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := uow.Outcomes().Save(ctx, outcome, cmd.ExpectedVersion); err != nil {
			return domain.Outcome{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return outcome, revision, err
	})
}

func (s *Service) ArchiveOutcome(ctx context.Context, commandContext domain.CommandContext, cmd ArchiveOutcomeCommand) (MutationResult[domain.Outcome], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Outcome]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Outcome, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return domain.Outcome{}, 0, err
		}
		outcome, err := uow.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
		if err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := outcome.Archive(now); err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := uow.Outcomes().Save(ctx, outcome, cmd.ExpectedVersion); err != nil {
			return domain.Outcome{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return outcome, revision, err
	})
}

func (s *Service) UnarchiveOutcome(ctx context.Context, commandContext domain.CommandContext, cmd UnarchiveOutcomeCommand) (MutationResult[domain.Outcome], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Outcome]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Outcome, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return domain.Outcome{}, 0, err
		}
		outcome, err := uow.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
		if err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := outcome.Unarchive(cmd.Reason, now); err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := uow.Outcomes().Save(ctx, outcome, cmd.ExpectedVersion); err != nil {
			return domain.Outcome{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return outcome, revision, err
	})
}

func (s *Service) CancelObjective(ctx context.Context, commandContext domain.CommandContext, cmd CancelObjectiveCommand) (MutationResult[domain.Objective], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Objective]{}, err
	}
	now := s.clock.Now().UTC()
	conclusion := conclusionFromContext(commandContext, cmd.Reason, now)
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Objective, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Objective{}, 0, err
		}
		objective, err := uow.Objectives().Get(ctx, cmd.Scope, cmd.ObjectiveID)
		if err != nil {
			return domain.Objective{}, 0, err
		}
		if err := objective.Cancel(conclusion, now); err != nil {
			return domain.Objective{}, 0, err
		}
		if err := uow.Objectives().Save(ctx, objective, cmd.ExpectedVersion); err != nil {
			return domain.Objective{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return objective, revision, err
	})
}

func (s *Service) ReopenObjective(ctx context.Context, commandContext domain.CommandContext, cmd ReopenObjectiveCommand) (MutationResult[domain.Objective], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Objective]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Objective, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Objective{}, 0, err
		}
		objective, err := uow.Objectives().Get(ctx, cmd.Scope, cmd.ObjectiveID)
		if err != nil {
			return domain.Objective{}, 0, err
		}
		if err := objective.Reopen(cmd.Reason, now); err != nil {
			return domain.Objective{}, 0, err
		}
		if err := uow.Objectives().Save(ctx, objective, cmd.ExpectedVersion); err != nil {
			return domain.Objective{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return objective, revision, err
	})
}

func (s *Service) DeferWorkItem(ctx context.Context, commandContext domain.CommandContext, cmd DeferWorkItemCommand) (MutationResult[domain.WorkItem], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		item, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := item.Defer(now); err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := uow.WorkItems().Save(ctx, item, cmd.ExpectedVersion); err != nil {
			return domain.WorkItem{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return item, revision, err
	})
}

func (s *Service) ReleaseWorkItem(ctx context.Context, commandContext domain.CommandContext, cmd ReleaseWorkItemCommand) (MutationResult[domain.WorkItem], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		item, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := item.Release(commandContext.PrincipalID, cmd.ClaimID, cmd.FencingToken, now); err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := uow.WorkItems().Save(ctx, item, cmd.ExpectedVersion); err != nil {
			return domain.WorkItem{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return item, revision, err
	})
}

func (s *Service) CancelWorkItem(ctx context.Context, commandContext domain.CommandContext, cmd CancelWorkItemCommand) (MutationResult[domain.WorkItem], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	now := s.clock.Now().UTC()
	conclusion := conclusionFromContext(commandContext, cmd.Reason, now)
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		item, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := item.Cancel(conclusion, now); err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := uow.WorkItems().Save(ctx, item, cmd.ExpectedVersion); err != nil {
			return domain.WorkItem{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return item, revision, err
	})
}

func (s *Service) ReopenWorkItem(ctx context.Context, commandContext domain.CommandContext, cmd ReopenWorkItemCommand) (MutationResult[domain.WorkItem], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		item, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := item.Reopen(cmd.Reason, now); err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := uow.WorkItems().Save(ctx, item, cmd.ExpectedVersion); err != nil {
			return domain.WorkItem{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return item, revision, err
	})
}

func (s *Service) RetireCriterion(ctx context.Context, commandContext domain.CommandContext, cmd RetireCriterionCommand) (MutationResult[domain.SuccessCriterion], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.SuccessCriterion]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.SuccessCriterion, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Owner.Scope); err != nil {
			return domain.SuccessCriterion{}, 0, err
		}
		criterion, err := retireCriterion(ctx, uow, cmd.Owner, cmd.CriterionID, cmd.ExpectedVersion, now)
		if err != nil {
			return domain.SuccessCriterion{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Owner.Scope)
		return criterion, revision, err
	})
}

func (s *Service) SetOutcomeOwners(ctx context.Context, commandContext domain.CommandContext, cmd SetOutcomeOwnersCommand) (MutationResult[domain.Outcome], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Outcome]{}, err
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
		if actorRefsEqual(outcome.OwnerRefs, cmd.OwnerRefs) {
			return outcome, coordination.Revision, nil
		}
		if err := outcome.SetOwners(cmd.OwnerRefs, now); err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := uow.Outcomes().Save(ctx, outcome, cmd.ExpectedVersion); err != nil {
			return domain.Outcome{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return outcome, revision, err
	})
}

func (s *Service) SetObjectiveOwners(ctx context.Context, commandContext domain.CommandContext, cmd SetObjectiveOwnersCommand) (MutationResult[domain.Objective], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Objective]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Objective, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Objective{}, 0, err
		}
		objective, err := uow.Objectives().Get(ctx, cmd.Scope, cmd.ObjectiveID)
		if err != nil {
			return domain.Objective{}, 0, err
		}
		if actorRefsEqual(objective.OwnerRefs, cmd.OwnerRefs) {
			coordination, err := uow.Coordination().LockOutcome(ctx, cmd.Scope)
			if err != nil {
				return domain.Objective{}, 0, err
			}
			return objective, coordination.Revision, nil
		}
		if err := objective.SetOwners(cmd.OwnerRefs, now); err != nil {
			return domain.Objective{}, 0, err
		}
		if err := uow.Objectives().Save(ctx, objective, cmd.ExpectedVersion); err != nil {
			return domain.Objective{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return objective, revision, err
	})
}

func (s *Service) SetWorkItemAssignees(ctx context.Context, commandContext domain.CommandContext, cmd SetWorkItemAssigneesCommand) (MutationResult[domain.WorkItem], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		item, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if actorRefsEqual(item.AssigneeRefs, cmd.AssigneeRefs) {
			coordination, err := uow.Coordination().LockOutcome(ctx, cmd.Scope)
			if err != nil {
				return domain.WorkItem{}, 0, err
			}
			return item, coordination.Revision, nil
		}
		if err := item.SetAssignees(cmd.AssigneeRefs, now); err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := uow.WorkItems().Save(ctx, item, cmd.ExpectedVersion); err != nil {
			return domain.WorkItem{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return item, revision, err
	})
}

func retireCriterion(ctx context.Context, uow ports.UnitOfWork, owner domain.EntityRef, criterionID domain.ID, expected domain.Version, now time.Time) (domain.SuccessCriterion, error) {
	switch owner.Kind {
	case domain.EntityKindOutcome:
		value, err := uow.Outcomes().Get(ctx, owner.Scope.NamespaceID, owner.Scope.OutcomeID)
		if err != nil {
			return domain.SuccessCriterion{}, err
		}
		if err := value.RetireCriterion(criterionID, now); err != nil {
			return domain.SuccessCriterion{}, err
		}
		if err := uow.Outcomes().Save(ctx, value, expected); err != nil {
			return domain.SuccessCriterion{}, err
		}
		c, err := value.Criteria.Find(criterionID)
		if err != nil {
			return domain.SuccessCriterion{}, err
		}
		return *c, nil
	case domain.EntityKindObjective:
		value, err := uow.Objectives().Get(ctx, owner.Scope, owner.ID)
		if err != nil {
			return domain.SuccessCriterion{}, err
		}
		if err := value.RetireCriterion(criterionID, now); err != nil {
			return domain.SuccessCriterion{}, err
		}
		if err := uow.Objectives().Save(ctx, value, expected); err != nil {
			return domain.SuccessCriterion{}, err
		}
		c, err := value.Criteria.Find(criterionID)
		if err != nil {
			return domain.SuccessCriterion{}, err
		}
		return *c, nil
	case domain.EntityKindWorkItem:
		value, err := uow.WorkItems().Get(ctx, owner.Scope, owner.ID)
		if err != nil {
			return domain.SuccessCriterion{}, err
		}
		if err := value.RetireCriterion(criterionID, now); err != nil {
			return domain.SuccessCriterion{}, err
		}
		if err := uow.WorkItems().Save(ctx, value, expected); err != nil {
			return domain.SuccessCriterion{}, err
		}
		c, err := value.Criteria.Find(criterionID)
		if err != nil {
			return domain.SuccessCriterion{}, err
		}
		return *c, nil
	default:
		return domain.SuccessCriterion{}, domain.NewError(domain.ErrorCodeInvalidArgument, "owner kind does not support criteria")
	}
}

func requireOpenOutcome(ctx context.Context, uow ports.UnitOfWork, scope domain.Scope) error {
	if _, err := uow.Coordination().LockOutcome(ctx, scope); err != nil {
		return err
	}
	outcome, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		return err
	}
	if outcome.IsArchived() {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "outcome must not be archived")
	}
	switch outcome.Lifecycle {
	case domain.OutcomeLifecycleAchieved, domain.OutcomeLifecycleFailed, domain.OutcomeLifecycleAbandoned:
		return domain.NewError(domain.ErrorCodePreconditionFailed, "outcome must not be terminal")
	default:
		return nil
	}
}
