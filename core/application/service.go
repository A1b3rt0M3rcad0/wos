package application

import (
	"context"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/core/ports"
)

type Service struct {
	tx    ports.TransactionManager
	clock ports.Clock
	ids   ports.IDGenerator
}

func NewService(tx ports.TransactionManager, clock ports.Clock, ids ports.IDGenerator) (*Service, error) {
	if tx == nil || clock == nil || ids == nil {
		return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "transaction manager, clock and id generator are required")
	}
	return &Service{tx: tx, clock: clock, ids: ids}, nil
}

func (s *Service) CreateOutcome(ctx context.Context, commandContext domain.CommandContext, cmd CreateOutcomeCommand) (MutationResult[domain.Outcome], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Outcome]{}, err
	}
	id, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.Outcome]{}, err
	}
	now := s.clock.Now().UTC()
	outcome, err := domain.NewOutcome(id, cmd.NamespaceID, cmd.Title, cmd.Description, cmd.DesiredState, cmd.Priority, now)
	if err != nil {
		return MutationResult[domain.Outcome]{}, err
	}

	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Outcome, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, outcome.Scope()); err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := uow.Outcomes().Insert(ctx, outcome); err != nil {
			return domain.Outcome{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, outcome.Scope())
		return outcome, revision, err
	})
}

func (s *Service) AddCriterion(ctx context.Context, commandContext domain.CommandContext, cmd AddCriterionCommand) (MutationResult[domain.SuccessCriterion], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.SuccessCriterion]{}, err
	}
	id, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.SuccessCriterion]{}, err
	}
	criterion, err := domain.NewSuccessCriterion(id, cmd.Owner, cmd.Title, cmd.Description, cmd.Required, cmd.VerificationMode)
	if err != nil {
		return MutationResult[domain.SuccessCriterion]{}, err
	}
	now := s.clock.Now().UTC()

	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.SuccessCriterion, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Owner.Scope); err != nil {
			return domain.SuccessCriterion{}, 0, err
		}
		if err := addCriterion(ctx, uow, cmd.Owner, cmd.ExpectedVersion, criterion, now); err != nil {
			return domain.SuccessCriterion{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Owner.Scope)
		return criterion, revision, err
	})
}

func (s *Service) ReviseCriterion(ctx context.Context, commandContext domain.CommandContext, cmd ReviseCriterionCommand) (MutationResult[domain.SuccessCriterion], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.SuccessCriterion]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.SuccessCriterion, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Owner.Scope); err != nil {
			return domain.SuccessCriterion{}, 0, err
		}
		criterion, err := reviseCriterion(ctx, uow, cmd, now)
		if err != nil {
			return domain.SuccessCriterion{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Owner.Scope)
		return criterion, revision, err
	})
}

func (s *Service) AttestCriterion(ctx context.Context, commandContext domain.CommandContext, cmd AttestCriterionCommand) (MutationResult[domain.CriterionAssessment], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.CriterionAssessment]{}, err
	}
	id, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.CriterionAssessment]{}, err
	}
	now := s.clock.Now().UTC()
	assessment := domain.CriterionAssessment{
		ID:                id,
		CriterionID:       cmd.CriterionID,
		CriterionRevision: cmd.CriterionRevision,
		Result:            cmd.Result,
		Rationale:         cmd.Rationale,
		PrincipalID:       commandContext.PrincipalID,
		Actor:             commandContext.Actor,
		AssessedAt:        now,
	}

	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.CriterionAssessment, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Owner.Scope); err != nil {
			return domain.CriterionAssessment{}, 0, err
		}
		if err := assessCriterion(ctx, uow, cmd.Owner, cmd.ExpectedVersion, assessment, now); err != nil {
			return domain.CriterionAssessment{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Owner.Scope)
		return assessment, revision, err
	})
}

func (s *Service) ActivateOutcome(ctx context.Context, commandContext domain.CommandContext, cmd ActivateOutcomeCommand) (MutationResult[domain.Outcome], error) {
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
		if err := outcome.Activate(now); err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := uow.Outcomes().Save(ctx, outcome, cmd.ExpectedVersion); err != nil {
			return domain.Outcome{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return outcome, revision, err
	})
}

func (s *Service) AchieveOutcome(ctx context.Context, commandContext domain.CommandContext, cmd AchieveOutcomeCommand) (MutationResult[domain.Outcome], error) {
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
		objectives, err := uow.Objectives().ListByOutcome(ctx, cmd.Scope)
		if err != nil {
			return domain.Outcome{}, 0, err
		}
		for _, objective := range objectives {
			if objective.RequiredForOutcome && objective.Lifecycle != domain.ObjectiveLifecycleAchieved {
				return domain.Outcome{}, 0, domain.NewError(domain.ErrorCodePreconditionFailed, "required objective is not achieved")
			}
		}
		if err := outcome.Achieve(conclusion, now); err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := uow.Outcomes().Save(ctx, outcome, cmd.ExpectedVersion); err != nil {
			return domain.Outcome{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return outcome, revision, err
	})
}

func (s *Service) CreateObjective(ctx context.Context, commandContext domain.CommandContext, cmd CreateObjectiveCommand) (MutationResult[domain.Objective], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Objective]{}, err
	}
	id, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.Objective]{}, err
	}
	now := s.clock.Now().UTC()
	objective, err := domain.NewObjective(id, cmd.Scope, cmd.Title, cmd.Description, cmd.Priority, cmd.RequiredForOutcome, now)
	if err != nil {
		return MutationResult[domain.Objective]{}, err
	}
	objective.ParentObjectiveID = cloneIDPtr(cmd.ParentObjectiveID)

	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Objective, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return domain.Objective{}, 0, err
		}
		outcome, err := uow.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
		if err != nil {
			return domain.Objective{}, 0, err
		}
		if outcome.IsArchived() || outcome.Lifecycle == domain.OutcomeLifecycleAchieved || outcome.Lifecycle == domain.OutcomeLifecycleFailed || outcome.Lifecycle == domain.OutcomeLifecycleAbandoned {
			return domain.Objective{}, 0, domain.NewError(domain.ErrorCodeInvalidTransition, "cannot create objective in archived or terminal outcome")
		}
		if cmd.ParentObjectiveID != nil {
			if _, err := uow.Objectives().Get(ctx, cmd.Scope, *cmd.ParentObjectiveID); err != nil {
				return domain.Objective{}, 0, err
			}
		}
		if err := uow.Objectives().Insert(ctx, objective); err != nil {
			return domain.Objective{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return objective, revision, err
	})
}

func (s *Service) StartObjective(ctx context.Context, commandContext domain.CommandContext, cmd StartObjectiveCommand) (MutationResult[domain.Objective], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Objective]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Objective, domain.OutcomeRevision, error) {
		if err := requireActiveOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Objective{}, 0, err
		}
		objective, err := uow.Objectives().Get(ctx, cmd.Scope, cmd.ObjectiveID)
		if err != nil {
			return domain.Objective{}, 0, err
		}
		if err := objective.Start(now); err != nil {
			return domain.Objective{}, 0, err
		}
		if err := uow.Objectives().Save(ctx, objective, cmd.ExpectedVersion); err != nil {
			return domain.Objective{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return objective, revision, err
	})
}

func (s *Service) AchieveObjective(ctx context.Context, commandContext domain.CommandContext, cmd AchieveObjectiveCommand) (MutationResult[domain.Objective], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Objective]{}, err
	}
	now := s.clock.Now().UTC()
	conclusion := conclusionFromContext(commandContext, cmd.Reason, now)
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Objective, domain.OutcomeRevision, error) {
		if err := requireActiveOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Objective{}, 0, err
		}
		objective, err := uow.Objectives().Get(ctx, cmd.Scope, cmd.ObjectiveID)
		if err != nil {
			return domain.Objective{}, 0, err
		}
		if err := requireHardDependenciesSatisfied(ctx, uow, objective.Ref()); err != nil {
			return domain.Objective{}, 0, err
		}
		if err := objective.Achieve(conclusion, now); err != nil {
			return domain.Objective{}, 0, err
		}
		if err := uow.Objectives().Save(ctx, objective, cmd.ExpectedVersion); err != nil {
			return domain.Objective{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return objective, revision, err
	})
}

func (s *Service) CreateWorkItem(ctx context.Context, commandContext domain.CommandContext, cmd CreateWorkItemCommand) (MutationResult[domain.WorkItem], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	id, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	now := s.clock.Now().UTC()
	item, err := domain.NewWorkItem(id, cmd.Scope, cmd.Title, cmd.Description, cmd.Priority, cmd.Lifecycle, now)
	if err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	item.ObjectiveID = cloneIDPtr(cmd.ObjectiveID)
	item.NotBefore = cloneTimePtrUTC(cmd.NotBefore)

	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		outcome, err := uow.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if outcome.IsArchived() || outcome.Lifecycle == domain.OutcomeLifecycleAchieved || outcome.Lifecycle == domain.OutcomeLifecycleFailed || outcome.Lifecycle == domain.OutcomeLifecycleAbandoned {
			return domain.WorkItem{}, 0, domain.NewError(domain.ErrorCodeInvalidTransition, "cannot create work in archived or terminal outcome")
		}
		if cmd.ObjectiveID != nil {
			objective, err := uow.Objectives().Get(ctx, cmd.Scope, *cmd.ObjectiveID)
			if err != nil {
				return domain.WorkItem{}, 0, err
			}
			if objective.Lifecycle == domain.ObjectiveLifecycleAchieved || objective.Lifecycle == domain.ObjectiveLifecycleCancelled {
				return domain.WorkItem{}, 0, domain.NewError(domain.ErrorCodeInvalidTransition, "cannot attach new work to terminal objective")
			}
		}
		if err := uow.WorkItems().Insert(ctx, item); err != nil {
			return domain.WorkItem{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return item, revision, err
	})
}

func (s *Service) ActivateWorkItem(ctx context.Context, commandContext domain.CommandContext, cmd ActivateWorkItemCommand) (MutationResult[domain.WorkItem], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		item, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := item.Activate(now); err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := uow.WorkItems().Save(ctx, item, cmd.ExpectedVersion); err != nil {
			return domain.WorkItem{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return item, revision, err
	})
}

func (s *Service) ClaimWorkItem(ctx context.Context, commandContext domain.CommandContext, cmd ClaimWorkItemCommand) (MutationResult[domain.WorkItem], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	claimID, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if err := requireActiveOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		item, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if item.ObjectiveID != nil {
			objective, err := uow.Objectives().Get(ctx, cmd.Scope, *item.ObjectiveID)
			if err != nil {
				return domain.WorkItem{}, 0, err
			}
			if objective.Lifecycle == domain.ObjectiveLifecycleAchieved || objective.Lifecycle == domain.ObjectiveLifecycleCancelled {
				return domain.WorkItem{}, 0, domain.NewError(domain.ErrorCodePreconditionFailed, "associated objective is terminal")
			}
		}
		if err := item.Claim(claimID, commandContext.PrincipalID, commandContext.Actor, cmd.TTL, now); err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := uow.WorkItems().Save(ctx, item, cmd.ExpectedVersion); err != nil {
			return domain.WorkItem{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return item, revision, err
	})
}

func (s *Service) CompleteWorkItem(ctx context.Context, commandContext domain.CommandContext, cmd CompleteWorkItemCommand) (MutationResult[domain.WorkItem], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	now := s.clock.Now().UTC()
	conclusion := conclusionFromContext(commandContext, cmd.Reason, now)
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		item, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := requireHardDependenciesSatisfied(ctx, uow, item.Ref()); err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := item.Complete(commandContext.PrincipalID, cmd.ClaimID, cmd.FencingToken, cmd.ResultSummary, conclusion, now); err != nil {
			return domain.WorkItem{}, 0, err
		}
		if err := uow.WorkItems().Save(ctx, item, cmd.ExpectedVersion); err != nil {
			return domain.WorkItem{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return item, revision, err
	})
}

func transact[T any](ctx context.Context, manager ports.TransactionManager, fn func(ports.UnitOfWork) (T, domain.OutcomeRevision, error)) (MutationResult[T], error) {
	var zero T
	uow, err := manager.Begin(ctx)
	if err != nil {
		return MutationResult[T]{}, err
	}
	committed := false
	defer func() {
		if !committed {
			_ = uow.Rollback()
		}
	}()

	value, revision, err := fn(uow)
	if err != nil {
		return MutationResult[T]{Value: zero}, err
	}
	if err := uow.Commit(); err != nil {
		return MutationResult[T]{Value: zero}, err
	}
	committed = true
	return MutationResult[T]{Value: value, OutcomeRevision: revision}, nil
}

func requireActiveOutcome(ctx context.Context, uow ports.UnitOfWork, scope domain.Scope) error {
	if _, err := uow.Coordination().LockOutcome(ctx, scope); err != nil {
		return err
	}
	outcome, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		return err
	}
	if outcome.IsArchived() || outcome.Lifecycle != domain.OutcomeLifecycleActive {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "outcome must be active and not archived")
	}
	return nil
}

func addCriterion(ctx context.Context, uow ports.UnitOfWork, owner domain.EntityRef, expected domain.Version, criterion domain.SuccessCriterion, now time.Time) error {
	switch owner.Kind {
	case domain.EntityKindOutcome:
		value, err := uow.Outcomes().Get(ctx, owner.Scope.NamespaceID, owner.Scope.OutcomeID)
		if err != nil {
			return err
		}
		if err := value.AddCriterion(criterion, now); err != nil {
			return err
		}
		return uow.Outcomes().Save(ctx, value, expected)
	case domain.EntityKindObjective:
		value, err := uow.Objectives().Get(ctx, owner.Scope, owner.ID)
		if err != nil {
			return err
		}
		if err := value.AddCriterion(criterion, now); err != nil {
			return err
		}
		return uow.Objectives().Save(ctx, value, expected)
	case domain.EntityKindWorkItem:
		value, err := uow.WorkItems().Get(ctx, owner.Scope, owner.ID)
		if err != nil {
			return err
		}
		if err := value.AddCriterion(criterion, now); err != nil {
			return err
		}
		return uow.WorkItems().Save(ctx, value, expected)
	default:
		return domain.NewError(domain.ErrorCodeInvalidArgument, "owner kind does not support criteria")
	}
}

func reviseCriterion(ctx context.Context, uow ports.UnitOfWork, cmd ReviseCriterionCommand, now time.Time) (domain.SuccessCriterion, error) {
	switch cmd.Owner.Kind {
	case domain.EntityKindOutcome:
		value, err := uow.Outcomes().Get(ctx, cmd.Owner.Scope.NamespaceID, cmd.Owner.Scope.OutcomeID)
		if err != nil {
			return domain.SuccessCriterion{}, err
		}
		if err := value.ReviseCriterion(cmd.CriterionID, cmd.Title, cmd.Description, cmd.Required, cmd.VerificationMode, now); err != nil {
			return domain.SuccessCriterion{}, err
		}
		if err := uow.Outcomes().Save(ctx, value, cmd.ExpectedVersion); err != nil {
			return domain.SuccessCriterion{}, err
		}
		c, err := value.Criteria.Find(cmd.CriterionID)
		if err != nil {
			return domain.SuccessCriterion{}, err
		}
		return *c, nil
	case domain.EntityKindObjective:
		value, err := uow.Objectives().Get(ctx, cmd.Owner.Scope, cmd.Owner.ID)
		if err != nil {
			return domain.SuccessCriterion{}, err
		}
		if err := value.ReviseCriterion(cmd.CriterionID, cmd.Title, cmd.Description, cmd.Required, cmd.VerificationMode, now); err != nil {
			return domain.SuccessCriterion{}, err
		}
		if err := uow.Objectives().Save(ctx, value, cmd.ExpectedVersion); err != nil {
			return domain.SuccessCriterion{}, err
		}
		c, err := value.Criteria.Find(cmd.CriterionID)
		if err != nil {
			return domain.SuccessCriterion{}, err
		}
		return *c, nil
	case domain.EntityKindWorkItem:
		value, err := uow.WorkItems().Get(ctx, cmd.Owner.Scope, cmd.Owner.ID)
		if err != nil {
			return domain.SuccessCriterion{}, err
		}
		if err := value.ReviseCriterion(cmd.CriterionID, cmd.Title, cmd.Description, cmd.Required, cmd.VerificationMode, now); err != nil {
			return domain.SuccessCriterion{}, err
		}
		if err := uow.WorkItems().Save(ctx, value, cmd.ExpectedVersion); err != nil {
			return domain.SuccessCriterion{}, err
		}
		c, err := value.Criteria.Find(cmd.CriterionID)
		if err != nil {
			return domain.SuccessCriterion{}, err
		}
		return *c, nil
	default:
		return domain.SuccessCriterion{}, domain.NewError(domain.ErrorCodeInvalidArgument, "owner kind does not support criteria")
	}
}

func assessCriterion(ctx context.Context, uow ports.UnitOfWork, owner domain.EntityRef, expected domain.Version, assessment domain.CriterionAssessment, now time.Time) error {
	switch owner.Kind {
	case domain.EntityKindOutcome:
		value, err := uow.Outcomes().Get(ctx, owner.Scope.NamespaceID, owner.Scope.OutcomeID)
		if err != nil {
			return err
		}
		if err := value.AssessCriterionAttestation(assessment, now); err != nil {
			return err
		}
		return uow.Outcomes().Save(ctx, value, expected)
	case domain.EntityKindObjective:
		value, err := uow.Objectives().Get(ctx, owner.Scope, owner.ID)
		if err != nil {
			return err
		}
		if err := value.AssessCriterionAttestation(assessment, now); err != nil {
			return err
		}
		return uow.Objectives().Save(ctx, value, expected)
	case domain.EntityKindWorkItem:
		value, err := uow.WorkItems().Get(ctx, owner.Scope, owner.ID)
		if err != nil {
			return err
		}
		if err := value.AssessCriterionAttestation(assessment, now); err != nil {
			return err
		}
		return uow.WorkItems().Save(ctx, value, expected)
	default:
		return domain.NewError(domain.ErrorCodeInvalidArgument, "owner kind does not support assessments")
	}
}

func conclusionFromContext(ctx domain.CommandContext, reason string, now time.Time) domain.Conclusion {
	return domain.Conclusion{
		PrincipalID: ctx.PrincipalID,
		Actor:       ctx.Actor,
		Reason:      strings.TrimSpace(reason),
		ConcludedAt: now,
	}
}

func cloneIDPtr(id *domain.ID) *domain.ID {
	if id == nil {
		return nil
	}
	value := *id
	return &value
}

func cloneTimePtrUTC(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}
