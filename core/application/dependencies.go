package application

import (
	"context"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/core/ports"
)

type AddDependencyCommand struct {
	Scope     domain.Scope
	SourceRef domain.EntityRef
	TargetRef domain.EntityRef
	Strength  domain.DependencyStrength
	Reason    string
}

type RemoveDependencyCommand struct {
	Scope           domain.Scope
	RelationID      domain.ID
	ExpectedVersion domain.Version
	Reason          string
}

func (s *Service) AddDependency(
	ctx context.Context,
	commandContext domain.CommandContext,
	cmd AddDependencyCommand,
) (MutationResult[domain.Relation], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Relation]{}, err
	}
	if cmd.SourceRef.Scope != cmd.Scope || cmd.TargetRef.Scope != cmd.Scope {
		return MutationResult[domain.Relation]{}, domain.NewError(domain.ErrorCodeInvalidRelation, "dependency endpoints must match command scope")
	}
	id, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.Relation]{}, err
	}
	now := s.clock.Now().UTC()
	relation, err := domain.NewDependencyRelation(id, cmd.SourceRef, cmd.TargetRef, cmd.Strength, now)
	if err != nil {
		return MutationResult[domain.Relation]{}, err
	}
	relation.Reason = strings.TrimSpace(cmd.Reason)

	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Relation, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Relation{}, 0, err
		}
		sourceState, err := dependencyEndpointState(ctx, uow, cmd.SourceRef)
		if err != nil {
			return domain.Relation{}, 0, err
		}
		if _, err := dependencyEndpointState(ctx, uow, cmd.TargetRef); err != nil {
			return domain.Relation{}, 0, err
		}
		if sourceState.terminal {
			return domain.Relation{}, 0, domain.NewError(domain.ErrorCodeInvalidTransition, "terminal dependency source must be reopened before changing dependencies")
		}
		if sourceState.inProgress && strings.TrimSpace(cmd.Reason) == "" {
			return domain.Relation{}, 0, domain.NewError(domain.ErrorCodeInvalidArgument, "adding dependency to in-progress source requires reason")
		}
		existing, err := uow.Relations().ListByOutcome(ctx, cmd.Scope)
		if err != nil {
			return domain.Relation{}, 0, err
		}
		if err := domain.ValidateDependencyAcyclic(existing, relation, domain.DefaultDependencyGraphLimit); err != nil {
			return domain.Relation{}, 0, err
		}
		if err := uow.Relations().Insert(ctx, relation); err != nil {
			return domain.Relation{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return relation, revision, err
	})
}

func (s *Service) RemoveDependency(
	ctx context.Context,
	commandContext domain.CommandContext,
	cmd RemoveDependencyCommand,
) (MutationResult[domain.Relation], error) {
	if err := commandContext.Validate(); err != nil {
		return MutationResult[domain.Relation]{}, err
	}
	now := s.clock.Now().UTC()
	return transactCommand(ctx, s, commandContext, cmd, func(uow ports.UnitOfWork) (domain.Relation, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Relation{}, 0, err
		}
		relation, err := uow.Relations().Get(ctx, cmd.Scope, cmd.RelationID)
		if err != nil {
			return domain.Relation{}, 0, err
		}
		if relation.RelationType != domain.RelationTypeDependsOn {
			return domain.Relation{}, 0, domain.NewError(domain.ErrorCodeInvalidRelation, "relation is not a dependency")
		}
		sourceState, err := dependencyEndpointState(ctx, uow, relation.SourceRef)
		if err != nil {
			return domain.Relation{}, 0, err
		}
		if sourceState.terminal {
			return domain.Relation{}, 0, domain.NewError(domain.ErrorCodeInvalidTransition, "terminal dependency source must be reopened before changing dependencies")
		}
		if err := relation.Remove(cmd.Reason, now); err != nil {
			return domain.Relation{}, 0, err
		}
		if err := uow.Relations().Save(ctx, relation, cmd.ExpectedVersion); err != nil {
			return domain.Relation{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return relation, revision, err
	})
}

type endpointState struct {
	completed  bool
	terminal   bool
	inProgress bool
}

func dependencyEndpointState(ctx context.Context, uow ports.UnitOfWork, ref domain.EntityRef) (endpointState, error) {
	if err := ref.Validate(); err != nil {
		return endpointState{}, err
	}
	switch ref.Kind {
	case domain.EntityKindObjective:
		value, err := uow.Objectives().Get(ctx, ref.Scope, ref.ID)
		if err != nil {
			return endpointState{}, err
		}
		return endpointState{
			completed:  value.Lifecycle == domain.ObjectiveLifecycleAchieved,
			terminal:   value.Lifecycle == domain.ObjectiveLifecycleAchieved || value.Lifecycle == domain.ObjectiveLifecycleCancelled,
			inProgress: value.Lifecycle == domain.ObjectiveLifecycleInProgress,
		}, nil
	case domain.EntityKindWorkItem:
		value, err := uow.WorkItems().Get(ctx, ref.Scope, ref.ID)
		if err != nil {
			return endpointState{}, err
		}
		return endpointState{
			completed:  value.Lifecycle == domain.WorkItemLifecycleDone,
			terminal:   value.Lifecycle == domain.WorkItemLifecycleDone || value.Lifecycle == domain.WorkItemLifecycleCancelled,
			inProgress: value.Lifecycle == domain.WorkItemLifecycleInProgress,
		}, nil
	default:
		return endpointState{}, domain.NewError(domain.ErrorCodeInvalidRelation, "dependency endpoint must be objective or work_item")
	}
}

func dependencyEvaluationsForSource(
	ctx context.Context,
	uow ports.UnitOfWork,
	source domain.EntityRef,
	relations []domain.Relation,
) ([]domain.DependencyEvaluation, error) {
	result := make([]domain.DependencyEvaluation, 0)
	for _, relation := range relations {
		if relation.Lifecycle != domain.RelationLifecycleActive ||
			relation.RelationType != domain.RelationTypeDependsOn ||
			relation.SourceRef != source {
			continue
		}
		state, err := dependencyEndpointState(ctx, uow, relation.TargetRef)
		if err != nil {
			return nil, err
		}
		result = append(result, domain.DependencyEvaluation{
			Relation:  relation,
			Satisfied: state.completed,
		})
	}
	return result, nil
}

func requireHardDependenciesSatisfied(
	ctx context.Context,
	uow ports.UnitOfWork,
	source domain.EntityRef,
) error {
	relations, err := uow.Relations().ListByOutcome(ctx, source.Scope)
	if err != nil {
		return err
	}
	evaluations, err := dependencyEvaluationsForSource(ctx, uow, source, relations)
	if err != nil {
		return err
	}
	for _, evaluation := range evaluations {
		if evaluation.Relation.Strength == domain.DependencyStrengthHard && !evaluation.Satisfied {
			return domain.NewError(domain.ErrorCodePreconditionFailed, "hard dependency is not satisfied")
		}
	}
	return nil
}

func evaluateObjectiveReadiness(
	ctx context.Context,
	uow ports.UnitOfWork,
	objective domain.Objective,
	now time.Time,
) (domain.Readiness, error) {
	outcome, err := uow.Outcomes().Get(ctx, objective.Scope.NamespaceID, objective.Scope.OutcomeID)
	if err != nil {
		return domain.Readiness{}, err
	}
	relations, err := uow.Relations().ListByOutcome(ctx, objective.Scope)
	if err != nil {
		return domain.Readiness{}, err
	}
	evaluations, err := dependencyEvaluationsForSource(ctx, uow, objective.Ref(), relations)
	if err != nil {
		return domain.Readiness{}, err
	}
	return domain.ObjectiveReadiness(objective, outcome, evaluations, now), nil
}

func evaluateWorkItemReadiness(
	ctx context.Context,
	uow ports.UnitOfWork,
	item domain.WorkItem,
	now time.Time,
) (domain.Readiness, error) {
	outcome, err := uow.Outcomes().Get(ctx, item.Scope.NamespaceID, item.Scope.OutcomeID)
	if err != nil {
		return domain.Readiness{}, err
	}
	var objective *domain.Objective
	if item.ObjectiveID != nil {
		value, err := uow.Objectives().Get(ctx, item.Scope, *item.ObjectiveID)
		if err != nil {
			return domain.Readiness{}, err
		}
		objective = &value
	}
	relations, err := uow.Relations().ListByOutcome(ctx, item.Scope)
	if err != nil {
		return domain.Readiness{}, err
	}
	evaluations, err := dependencyEvaluationsForSource(ctx, uow, item.Ref(), relations)
	if err != nil {
		return domain.Readiness{}, err
	}
	return domain.WorkItemReadiness(item, outcome, objective, evaluations, now), nil
}

func requireObjectiveReady(ctx context.Context, uow ports.UnitOfWork, objective domain.Objective, now time.Time) error {
	readiness, err := evaluateObjectiveReadiness(ctx, uow, objective, now)
	if err != nil {
		return err
	}
	if readiness.Ready {
		return nil
	}
	reasons := make([]string, len(readiness.Reasons))
	for i, reason := range readiness.Reasons {
		reasons[i] = string(reason)
	}
	return domain.NewError(domain.ErrorCodePreconditionFailed, "objective is not ready: "+strings.Join(reasons, ","))
}

func requireWorkItemReady(ctx context.Context, uow ports.UnitOfWork, item domain.WorkItem, now time.Time) error {
	readiness, err := evaluateWorkItemReadiness(ctx, uow, item, now)
	if err != nil {
		return err
	}
	if readiness.Ready {
		return nil
	}
	reasons := make([]string, len(readiness.Reasons))
	for i, reason := range readiness.Reasons {
		reasons[i] = string(reason)
	}
	return domain.NewError(domain.ErrorCodePreconditionFailed, "work item is not ready: "+strings.Join(reasons, ","))
}
