package application

import (
	"context"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type Service struct {
	tx                  ports.TransactionManager
	clock               ports.Clock
	ids                 ports.IDGenerator
	authorizer          ports.Authorizer
	requireIdentity     bool
	independentReviewer bool
}

func NewService(tx ports.TransactionManager, clock ports.Clock, ids ports.IDGenerator) (*Service, error) {
	return NewServiceWithAuthorizer(tx, clock, ids, ports.DenyPrivilegedAuthorizer{})
}

func NewServiceWithAuthorizer(
	tx ports.TransactionManager,
	clock ports.Clock,
	ids ports.IDGenerator,
	authorizer ports.Authorizer,
) (*Service, error) {
	if tx == nil || clock == nil || ids == nil || authorizer == nil {
		return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "transaction manager, clock, id generator and authorizer are required")
	}
	return &Service{tx: tx, clock: clock, ids: ids, authorizer: authorizer}, nil
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

func (s *Service) RecordCriterionAssessment(ctx context.Context, commandContext domain.CommandContext, cmd RecordCriterionAssessmentCommand) (MutationResult[domain.CriterionAssessment], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.CriterionAssessment]{}, err
	}

	waiverAuthorized := false
	if cmd.Result == domain.AssessmentResultWaived {
		if err := s.authorizer.Authorize(ctx, ports.AuthorizationRequest{
			NamespaceID: cmd.Owner.Scope.NamespaceID,
			PrincipalID: commandContext.PrincipalID,
			Permission:  ports.PermissionAssessmentWaive,
		}); err != nil {
			return MutationResult[domain.CriterionAssessment]{}, err
		}
		waiverAuthorized = true
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
		EvidenceIDs:       append([]domain.ID(nil), cmd.EvidenceIDs...),
		EvaluatorRef:      cloneEvaluatorRef(cmd.EvaluatorRef),
		PrincipalID:       commandContext.PrincipalID,
		Actor:             commandContext.Actor,
		AssessedAt:        now,
	}

	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.CriterionAssessment, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Owner.Scope); err != nil {
			return domain.CriterionAssessment{}, 0, err
		}
		if err := s.requireIndependentReview(ctx, uow, cmd.Owner.Scope, commandContext.PrincipalID); err != nil {
			return domain.CriterionAssessment{}, 0, err
		}
		if err := validateAssessmentEvidence(ctx, uow, cmd.Owner.Scope, assessment.EvidenceIDs); err != nil {
			return domain.CriterionAssessment{}, 0, err
		}
		if err := recordCriterionAssessment(ctx, uow, cmd.Owner, cmd.ExpectedVersion, assessment, waiverAuthorized, now); err != nil {
			return domain.CriterionAssessment{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Owner.Scope)
		return assessment, revision, err
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
		if err := s.requireIndependentReview(ctx, uow, cmd.Owner.Scope, commandContext.PrincipalID); err != nil {
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
		if err := requireNotBlocked(ctx, uow, outcome.Ref()); err != nil {
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
	conclusionID, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.Outcome]{}, err
	}
	conclusion := conclusionFromContext(conclusionID, commandContext, cmd.Reason, now)

	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Outcome, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := s.requireIndependentReview(ctx, uow, cmd.Scope, commandContext.PrincipalID); err != nil {
			return domain.Outcome{}, 0, err
		}
		outcome, err := uow.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
		if err != nil {
			return domain.Outcome{}, 0, err
		}
		if err := requireNotBlocked(ctx, uow, outcome.Ref()); err != nil {
			return domain.Outcome{}, 0, err
		}
		objectives, err := uow.Objectives().ListByOutcome(ctx, cmd.Scope)
		if err != nil {
			return domain.Outcome{}, 0, err
		}
		requiredObjectiveIDs := make([]domain.ID, 0)
		for _, objective := range objectives {
			if !objective.RequiredForOutcome {
				continue
			}
			if objective.Lifecycle != domain.ObjectiveLifecycleAchieved {
				return domain.Outcome{}, 0, domain.NewError(domain.ErrorCodePreconditionFailed, "required objective is not achieved")
			}
			requiredObjectiveIDs = append(requiredObjectiveIDs, objective.ID)
		}
		if err := outcome.AchieveWithObligations(conclusion, requiredObjectiveIDs, now); err != nil {
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
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Objective, domain.OutcomeRevision, error) {
		if err := requireActiveOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Objective{}, 0, err
		}
		now := s.clock.Now().UTC()
		objective, err := uow.Objectives().Get(ctx, cmd.Scope, cmd.ObjectiveID)
		if err != nil {
			return domain.Objective{}, 0, err
		}
		if err := requireObjectiveReady(ctx, uow, objective, now); err != nil {
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
	conclusionID, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.Objective]{}, err
	}
	conclusion := conclusionFromContext(conclusionID, commandContext, cmd.Reason, now)
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Objective, domain.OutcomeRevision, error) {
		if err := requireActiveOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Objective{}, 0, err
		}
		if err := s.requireIndependentReview(ctx, uow, cmd.Scope, commandContext.PrincipalID); err != nil {
			return domain.Objective{}, 0, err
		}
		objective, err := uow.Objectives().Get(ctx, cmd.Scope, cmd.ObjectiveID)
		if err != nil {
			return domain.Objective{}, 0, err
		}
		if err := requireNotBlocked(ctx, uow, objective.Ref()); err != nil {
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
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if err := requireActiveOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.WorkItem{}, 0, err
		}
		item, err := uow.WorkItems().Get(ctx, cmd.Scope, cmd.WorkItemID)
		if err != nil {
			return domain.WorkItem{}, 0, err
		}
		now := s.clock.Now().UTC()
		if err := requireWorkItemReady(ctx, uow, item, now); err != nil {
			return domain.WorkItem{}, 0, err
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
	conclusionID, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.WorkItem]{}, err
	}
	conclusion := conclusionFromContext(conclusionID, commandContext, cmd.Reason, now)
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.WorkItem, domain.OutcomeRevision, error) {
		if _, err := uow.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
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

func validateAssessmentEvidence(ctx context.Context, uow ports.UnitOfWork, scope domain.Scope, evidenceIDs []domain.ID) error {
	if len(evidenceIDs) == 0 {
		return nil
	}
	_, evidenceRepo, _, _, err := documentaryRepositories(uow)
	if err != nil {
		return err
	}
	seen := make(map[domain.ID]struct{}, len(evidenceIDs))
	for _, evidenceID := range evidenceIDs {
		if _, exists := seen[evidenceID]; exists {
			return domain.NewError(domain.ErrorCodeAssessment, "assessment evidence_ids cannot contain duplicates")
		}
		seen[evidenceID] = struct{}{}
		evidence, err := evidenceRepo.Get(ctx, scope, evidenceID)
		if err != nil {
			return err
		}
		if evidence.Lifecycle != domain.EvidenceLifecycleRegistered {
			return domain.NewError(domain.ErrorCodePreconditionFailed, "retracted evidence cannot support a new assessment")
		}
	}
	return nil
}

func recordCriterionAssessment(
	ctx context.Context,
	uow ports.UnitOfWork,
	owner domain.EntityRef,
	expected domain.Version,
	assessment domain.CriterionAssessment,
	waiverAuthorized bool,
	now time.Time,
) error {
	switch owner.Kind {
	case domain.EntityKindOutcome:
		value, err := uow.Outcomes().Get(ctx, owner.Scope.NamespaceID, owner.Scope.OutcomeID)
		if err != nil {
			return err
		}
		if err := value.RecordCriterionAssessment(assessment, waiverAuthorized, now); err != nil {
			return err
		}
		return uow.Outcomes().Save(ctx, value, expected)
	case domain.EntityKindObjective:
		value, err := uow.Objectives().Get(ctx, owner.Scope, owner.ID)
		if err != nil {
			return err
		}
		if err := value.RecordCriterionAssessment(assessment, waiverAuthorized, now); err != nil {
			return err
		}
		return uow.Objectives().Save(ctx, value, expected)
	case domain.EntityKindWorkItem:
		value, err := uow.WorkItems().Get(ctx, owner.Scope, owner.ID)
		if err != nil {
			return err
		}
		if err := value.RecordCriterionAssessment(assessment, waiverAuthorized, now); err != nil {
			return err
		}
		return uow.WorkItems().Save(ctx, value, expected)
	default:
		return domain.NewError(domain.ErrorCodeInvalidArgument, "owner kind does not support assessments")
	}
}

func cloneEvaluatorRef(value *domain.EvaluatorRef) *domain.EvaluatorRef {
	if value == nil {
		return nil
	}
	copyValue := *value
	return &copyValue
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

func conclusionFromContext(id domain.ID, ctx domain.CommandContext, reason string, now time.Time) domain.Conclusion {
	return domain.Conclusion{
		ID:          id,
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
