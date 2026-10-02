package application

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type CreateRoadmapCommand struct {
	Scope     domain.Scope
	PlanScope domain.RoadmapPlanScope
	Title     string
}

type OpenRoadmapDraftCommand struct {
	Scope              domain.Scope
	RoadmapID          domain.ID
	ExpectedVersion    domain.Version
	BaseRevisionNumber *uint64
}

type ReplaceRoadmapDraftCommand struct {
	Scope                domain.Scope
	RoadmapID            domain.ID
	ExpectedVersion      domain.Version
	ExpectedDraftVersion uint64
	Nodes                []domain.RoadmapNode
	AfterLinks           []domain.RoadmapAfterLink
}

type DiscardRoadmapDraftCommand struct {
	Scope                domain.Scope
	RoadmapID            domain.ID
	ExpectedVersion      domain.Version
	ExpectedDraftVersion uint64
}

func (s *Service) CreateRoadmap(
	ctx context.Context,
	cc domain.CommandContext,
	cmd CreateRoadmapCommand,
) (MutationResult[domain.Roadmap], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Roadmap]{}, err
	}
	id, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.Roadmap]{}, err
	}
	value, err := domain.NewRoadmap(id, cmd.Scope, cmd.PlanScope, cmd.Title, s.clock.Now().UTC())
	if err != nil {
		return MutationResult[domain.Roadmap]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (domain.Roadmap, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, cmd.Scope); err != nil {
			return domain.Roadmap{}, 0, err
		}
		if err := validateRoadmapPlanScope(ctx, uow, value); err != nil {
			return domain.Roadmap{}, 0, err
		}
		repo, err := planningRoadmaps(uow)
		if err != nil {
			return domain.Roadmap{}, 0, err
		}
		if err := repo.Insert(ctx, value); err != nil {
			return domain.Roadmap{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return value, revision, err
	})
}

func (s *Service) OpenRoadmapDraft(
	ctx context.Context,
	cc domain.CommandContext,
	cmd OpenRoadmapDraftCommand,
) (MutationResult[domain.Roadmap], error) {
	return s.mutateRoadmap(ctx, cc, cmd, cmd.Scope, cmd.RoadmapID, cmd.ExpectedVersion, func(
		ctx context.Context,
		uow ports.UnitOfWork,
		value *domain.Roadmap,
	) error {
		return value.OpenDraft(cmd.BaseRevisionNumber, s.clock.Now().UTC())
	})
}

func (s *Service) ReplaceRoadmapDraft(
	ctx context.Context,
	cc domain.CommandContext,
	cmd ReplaceRoadmapDraftCommand,
) (MutationResult[domain.Roadmap], error) {
	return s.mutateRoadmap(ctx, cc, cmd, cmd.Scope, cmd.RoadmapID, cmd.ExpectedVersion, func(
		ctx context.Context,
		uow ports.UnitOfWork,
		value *domain.Roadmap,
	) error {
		if err := validateRoadmapDraftReferences(ctx, uow, *value, cmd.Nodes); err != nil {
			return err
		}
		return value.ReplaceDraft(
			cmd.ExpectedDraftVersion,
			cmd.Nodes,
			cmd.AfterLinks,
			s.clock.Now().UTC(),
		)
	})
}

func (s *Service) DiscardRoadmapDraft(
	ctx context.Context,
	cc domain.CommandContext,
	cmd DiscardRoadmapDraftCommand,
) (MutationResult[domain.Roadmap], error) {
	return s.mutateRoadmap(ctx, cc, cmd, cmd.Scope, cmd.RoadmapID, cmd.ExpectedVersion, func(
		_ context.Context,
		_ ports.UnitOfWork,
		value *domain.Roadmap,
	) error {
		return value.DiscardDraft(cmd.ExpectedDraftVersion, s.clock.Now().UTC())
	})
}

func (s *Service) mutateRoadmap(
	ctx context.Context,
	cc domain.CommandContext,
	command any,
	scope domain.Scope,
	id domain.ID,
	expected domain.Version,
	mutate func(context.Context, ports.UnitOfWork, *domain.Roadmap) error,
) (MutationResult[domain.Roadmap], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Roadmap]{}, err
	}
	return transactCommand(ctx, s, cc, command, func(uow ports.UnitOfWork) (domain.Roadmap, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, uow, scope); err != nil {
			return domain.Roadmap{}, 0, err
		}
		repo, err := planningRoadmaps(uow)
		if err != nil {
			return domain.Roadmap{}, 0, err
		}
		value, err := repo.Get(ctx, scope, id)
		if err != nil {
			return domain.Roadmap{}, 0, err
		}
		if err := mutate(ctx, uow, &value); err != nil {
			return domain.Roadmap{}, 0, err
		}
		if err := repo.Save(ctx, value, expected); err != nil {
			return domain.Roadmap{}, 0, err
		}
		revision, err := uow.Coordination().AdvanceOutcome(ctx, scope)
		return value, revision, err
	})
}

func (s *Service) GetRoadmap(
	ctx context.Context,
	scope domain.Scope,
	id domain.ID,
) (ReadResult[domain.Roadmap], error) {
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[domain.Roadmap]{}, err
	}
	defer uow.Rollback()
	if err := lockExistingOutcome(ctx, uow, scope); err != nil {
		return ReadResult[domain.Roadmap]{}, err
	}
	repo, err := planningRoadmaps(uow)
	if err != nil {
		return ReadResult[domain.Roadmap]{}, err
	}
	value, err := repo.Get(ctx, scope, id)
	if err != nil {
		return ReadResult[domain.Roadmap]{}, err
	}
	coordination, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return ReadResult[domain.Roadmap]{}, err
	}
	return ReadResult[domain.Roadmap]{
		Value:           value,
		OutcomeRevision: coordination.Revision,
	}, nil
}

func (s *Service) ListRoadmaps(
	ctx context.Context,
	scope domain.Scope,
) ([]domain.Roadmap, domain.OutcomeRevision, error) {
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer uow.Rollback()
	if err := lockExistingOutcome(ctx, uow, scope); err != nil {
		return nil, 0, err
	}
	repo, err := planningRoadmaps(uow)
	if err != nil {
		return nil, 0, err
	}
	values, err := repo.ListByOutcome(ctx, scope)
	if err != nil {
		return nil, 0, err
	}
	coordination, err := uow.Coordination().LockOutcome(ctx, scope)
	return values, coordination.Revision, err
}

func planningRoadmaps(uow ports.UnitOfWork) (ports.RoadmapRepository, error) {
	planning, ok := uow.(ports.PlanningUnitOfWork)
	if !ok {
		return nil, domain.NewError(domain.ErrorCodeRoadmap, "storage does not expose planning repositories")
	}
	return planning.Roadmaps(), nil
}

func validateRoadmapPlanScope(ctx context.Context, uow ports.UnitOfWork, roadmap domain.Roadmap) error {
	if roadmap.PlanScope.Kind == domain.RoadmapScopeOutcome {
		return nil
	}
	objective, err := uow.Objectives().Get(ctx, roadmap.Scope, roadmap.PlanScope.ID)
	if err != nil {
		return err
	}
	if objective.Scope != roadmap.Scope {
		return domain.NewError(domain.ErrorCodeRoadmap, "roadmap Objective scope must belong to the same Outcome")
	}
	return nil
}

func validateRoadmapDraftReferences(
	ctx context.Context,
	uow ports.UnitOfWork,
	roadmap domain.Roadmap,
	nodes []domain.RoadmapNode,
) error {
	outcome, err := uow.Outcomes().Get(ctx, roadmap.Scope.NamespaceID, roadmap.Scope.OutcomeID)
	if err != nil {
		return err
	}
	objectives, err := uow.Objectives().ListByOutcome(ctx, roadmap.Scope)
	if err != nil {
		return err
	}
	workItems, err := uow.WorkItems().ListByOutcome(ctx, roadmap.Scope)
	if err != nil {
		return err
	}

	objectiveByID := make(map[domain.ID]domain.Objective, len(objectives))
	for _, objective := range objectives {
		objectiveByID[objective.ID] = objective
	}
	workByID := make(map[domain.ID]domain.WorkItem, len(workItems))
	for _, item := range workItems {
		workByID[item.ID] = item
	}

	allowedObjectives := make(map[domain.ID]struct{}, len(objectives))
	if roadmap.PlanScope.Kind == domain.RoadmapScopeOutcome {
		for _, objective := range objectives {
			allowedObjectives[objective.ID] = struct{}{}
		}
	} else {
		if _, exists := objectiveByID[roadmap.PlanScope.ID]; !exists {
			return domain.NewError(domain.ErrorCodeRoadmap, "roadmap Objective scope does not exist")
		}
		allowedObjectives[roadmap.PlanScope.ID] = struct{}{}
		for changed := true; changed; {
			changed = false
			for _, objective := range objectives {
				if _, already := allowedObjectives[objective.ID]; already || objective.ParentObjectiveID == nil {
					continue
				}
				if _, parentAllowed := allowedObjectives[*objective.ParentObjectiveID]; parentAllowed {
					allowedObjectives[objective.ID] = struct{}{}
					changed = true
				}
			}
		}
	}

	for _, node := range nodes {
		if node.TargetRef != nil {
			if err := validateRoadmapTargetRef(*node.TargetRef, roadmap, objectiveByID, workByID, allowedObjectives); err != nil {
				return err
			}
		}
		for _, criterionRef := range node.CriterionRefs {
			if err := validateRoadmapCriterionRef(
				criterionRef,
				roadmap,
				outcome,
				objectiveByID,
				workByID,
				allowedObjectives,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateRoadmapTargetRef(
	ref domain.EntityRef,
	roadmap domain.Roadmap,
	objectiveByID map[domain.ID]domain.Objective,
	workByID map[domain.ID]domain.WorkItem,
	allowedObjectives map[domain.ID]struct{},
) error {
	if ref.Scope != roadmap.Scope {
		return domain.NewError(domain.ErrorCodeRoadmap, "roadmap target must belong to the same Outcome")
	}
	switch ref.Kind {
	case domain.EntityKindObjective:
		if _, exists := objectiveByID[ref.ID]; !exists {
			return domain.NewError(domain.ErrorCodeNotFound, "roadmap target Objective not found")
		}
		if _, allowed := allowedObjectives[ref.ID]; !allowed {
			return domain.NewError(domain.ErrorCodeRoadmap, "Objective is outside the Roadmap scope")
		}
	case domain.EntityKindWorkItem:
		item, exists := workByID[ref.ID]
		if !exists {
			return domain.NewError(domain.ErrorCodeNotFound, "roadmap target WorkItem not found")
		}
		if roadmap.PlanScope.Kind == domain.RoadmapScopeObjective {
			if item.ObjectiveID == nil {
				return domain.NewError(domain.ErrorCodeRoadmap, "unscoped WorkItem is outside Objective Roadmap scope")
			}
			if _, allowed := allowedObjectives[*item.ObjectiveID]; !allowed {
				return domain.NewError(domain.ErrorCodeRoadmap, "WorkItem is outside the Roadmap Objective subtree")
			}
		}
	default:
		return domain.NewError(domain.ErrorCodeRoadmap, "Roadmap reference target kind is invalid")
	}
	return nil
}

func validateRoadmapCriterionRef(
	ref domain.RoadmapCriterionRef,
	roadmap domain.Roadmap,
	outcome domain.Outcome,
	objectiveByID map[domain.ID]domain.Objective,
	workByID map[domain.ID]domain.WorkItem,
	allowedObjectives map[domain.ID]struct{},
) error {
	switch ref.OwnerRef.Kind {
	case domain.EntityKindOutcome:
		if roadmap.PlanScope.Kind != domain.RoadmapScopeOutcome || ref.OwnerRef.ID != roadmap.Scope.OutcomeID {
			return domain.NewError(domain.ErrorCodeRoadmap, "Outcome criterion is outside the Roadmap scope")
		}
		if !criterionExists(outcome.Criteria, ref.CriterionID) {
			return domain.NewError(domain.ErrorCodeRoadmap, "milestone criterion does not exist on Outcome")
		}
	case domain.EntityKindObjective:
		objective, exists := objectiveByID[ref.OwnerRef.ID]
		if !exists {
			return domain.NewError(domain.ErrorCodeNotFound, "milestone criterion owner Objective not found")
		}
		if _, allowed := allowedObjectives[objective.ID]; !allowed {
			return domain.NewError(domain.ErrorCodeRoadmap, "milestone criterion owner is outside the Roadmap scope")
		}
		if !criterionExists(objective.Criteria, ref.CriterionID) {
			return domain.NewError(domain.ErrorCodeRoadmap, "milestone criterion does not exist on Objective")
		}
	case domain.EntityKindWorkItem:
		item, exists := workByID[ref.OwnerRef.ID]
		if !exists {
			return domain.NewError(domain.ErrorCodeNotFound, "milestone criterion owner WorkItem not found")
		}
		if roadmap.PlanScope.Kind == domain.RoadmapScopeObjective {
			if item.ObjectiveID == nil {
				return domain.NewError(domain.ErrorCodeRoadmap, "milestone WorkItem criterion is outside Objective Roadmap scope")
			}
			if _, allowed := allowedObjectives[*item.ObjectiveID]; !allowed {
				return domain.NewError(domain.ErrorCodeRoadmap, "milestone WorkItem criterion is outside the Roadmap scope")
			}
		}
		if !criterionExists(item.Criteria, ref.CriterionID) {
			return domain.NewError(domain.ErrorCodeRoadmap, "milestone criterion does not exist on WorkItem")
		}
	default:
		return domain.NewError(domain.ErrorCodeRoadmap, "milestone criterion owner kind is invalid")
	}
	return nil
}

func criterionExists(criteria domain.CriterionSet, id domain.ID) bool {
	value, err := criteria.Find(id)
	return err == nil && value.Status == domain.CriterionStatusActive
}
