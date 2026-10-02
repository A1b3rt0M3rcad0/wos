package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"

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

type PublishRoadmapDraftCommand struct {
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

func (s *Service) PublishRoadmapDraft(
	ctx context.Context,
	cc domain.CommandContext,
	cmd PublishRoadmapDraftCommand,
) (MutationResult[domain.Roadmap], error) {
	return s.mutateRoadmap(ctx, cc, cmd, cmd.Scope, cmd.RoadmapID, cmd.ExpectedVersion, func(
		ctx context.Context,
		uow ports.UnitOfWork,
		value *domain.Roadmap,
	) error {
		if value.Draft == nil || value.Draft.Lifecycle != domain.RoadmapDraftOpen {
			return domain.NewError(domain.ErrorCodeRoadmap, "roadmap has no publishable draft")
		}
		if value.Draft.DraftVersion != cmd.ExpectedDraftVersion {
			return domain.NewError(domain.ErrorCodeVersionConflict, "roadmap draft version conflict")
		}
		publishedNodes, dependencySnapshots, err := buildRoadmapPublicationSnapshot(ctx, uow, *value)
		if err != nil {
			return err
		}
		contentHash, err := roadmapRevisionContentHash(
			publishedNodes,
			value.Draft.AfterLinks,
			dependencySnapshots,
		)
		if err != nil {
			return err
		}
		_, err = value.PublishDraft(
			cmd.ExpectedDraftVersion,
			contentHash,
			publishedNodes,
			dependencySnapshots,
			cc.Actor,
			s.clock.Now().UTC(),
		)
		return err
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

func buildRoadmapPublicationSnapshot(
	ctx context.Context,
	uow ports.UnitOfWork,
	roadmap domain.Roadmap,
) ([]domain.RoadmapNode, []domain.RoadmapDependencySnapshot, error) {
	if roadmap.Draft == nil {
		return nil, nil, domain.NewError(domain.ErrorCodeRoadmap, "roadmap draft is required")
	}
	if err := validateRoadmapDraftReferences(ctx, uow, roadmap, roadmap.Draft.Nodes); err != nil {
		return nil, nil, err
	}

	outcome, err := uow.Outcomes().Get(ctx, roadmap.Scope.NamespaceID, roadmap.Scope.OutcomeID)
	if err != nil {
		return nil, nil, err
	}
	objectives, err := uow.Objectives().ListByOutcome(ctx, roadmap.Scope)
	if err != nil {
		return nil, nil, err
	}
	workItems, err := uow.WorkItems().ListByOutcome(ctx, roadmap.Scope)
	if err != nil {
		return nil, nil, err
	}
	objectiveByID := make(map[domain.ID]domain.Objective, len(objectives))
	for _, objective := range objectives {
		objectiveByID[objective.ID] = objective
	}
	workByID := make(map[domain.ID]domain.WorkItem, len(workItems))
	for _, item := range workItems {
		workByID[item.ID] = item
	}

	nodes := make([]domain.RoadmapNode, len(roadmap.Draft.Nodes))
	referenced := make(map[domain.EntityRef]struct{})
	for i, node := range roadmap.Draft.Nodes {
		nodes[i] = cloneRoadmapNode(node)
		if node.TargetRef != nil {
			title, err := roadmapTargetTitle(*node.TargetRef, objectiveByID, workByID)
			if err != nil {
				return nil, nil, err
			}
			nodes[i].ReferenceSnapshot = &domain.RoadmapReferenceSnapshot{
				TargetRef: *node.TargetRef,
				Title:     title,
			}
			referenced[*node.TargetRef] = struct{}{}
		}
		if node.NodeType == domain.RoadmapNodeMilestone {
			nodes[i].CriterionSnapshots = make([]domain.RoadmapCriterionSnapshot, 0, len(node.CriterionRefs))
			for _, ref := range node.CriterionRefs {
				criterion, err := roadmapCriterion(ref, outcome, objectiveByID, workByID)
				if err != nil {
					return nil, nil, err
				}
				nodes[i].CriterionSnapshots = append(nodes[i].CriterionSnapshots, domain.RoadmapCriterionSnapshot{
					OwnerRef:          ref.OwnerRef,
					CriterionID:       criterion.ID,
					CriterionRevision: criterion.Revision,
					Title:             criterion.Title,
					Required:          criterion.Required,
					VerificationMode:  criterion.VerificationMode,
				})
			}
		}
	}

	relations, err := uow.Relations().ListByOutcome(ctx, roadmap.Scope)
	if err != nil {
		return nil, nil, err
	}
	dependencies := make([]domain.RoadmapDependencySnapshot, 0)
	for _, relation := range relations {
		if relation.Lifecycle != domain.RelationLifecycleActive ||
			relation.RelationType != domain.RelationTypeDependsOn {
			continue
		}
		if _, ok := referenced[relation.SourceRef]; !ok {
			continue
		}
		if _, ok := referenced[relation.TargetRef]; !ok {
			continue
		}
		dependencies = append(dependencies, domain.RoadmapDependencySnapshot{
			DependentRef:    relation.SourceRef,
			PrerequisiteRef: relation.TargetRef,
			Strength:        relation.Strength,
			Satisfaction:    relation.Satisfaction,
		})
	}
	sort.Slice(dependencies, func(i, j int) bool {
		left := dependencies[i].DependentRef.Kind.String() + "/" + dependencies[i].DependentRef.ID.String() + "/" +
			dependencies[i].PrerequisiteRef.Kind.String() + "/" + dependencies[i].PrerequisiteRef.ID.String()
		right := dependencies[j].DependentRef.Kind.String() + "/" + dependencies[j].DependentRef.ID.String() + "/" +
			dependencies[j].PrerequisiteRef.Kind.String() + "/" + dependencies[j].PrerequisiteRef.ID.String()
		return left < right
	})
	return nodes, dependencies, nil
}

func roadmapRevisionContentHash(
	nodes []domain.RoadmapNode,
	afterLinks []domain.RoadmapAfterLink,
	dependencies []domain.RoadmapDependencySnapshot,
) (string, error) {
	canonicalNodes := append([]domain.RoadmapNode(nil), nodes...)
	sort.Slice(canonicalNodes, func(i, j int) bool { return canonicalNodes[i].NodeKey < canonicalNodes[j].NodeKey })
	canonicalLinks := append([]domain.RoadmapAfterLink(nil), afterLinks...)
	sort.Slice(canonicalLinks, func(i, j int) bool {
		if canonicalLinks[i].NodeKey == canonicalLinks[j].NodeKey {
			return canonicalLinks[i].AfterNodeKey < canonicalLinks[j].AfterNodeKey
		}
		return canonicalLinks[i].NodeKey < canonicalLinks[j].NodeKey
	})
	canonicalDependencies := append([]domain.RoadmapDependencySnapshot(nil), dependencies...)
	sort.Slice(canonicalDependencies, func(i, j int) bool {
		left := canonicalDependencies[i].DependentRef.Kind.String() + "/" + canonicalDependencies[i].DependentRef.ID.String() + "/" +
			canonicalDependencies[i].PrerequisiteRef.Kind.String() + "/" + canonicalDependencies[i].PrerequisiteRef.ID.String()
		right := canonicalDependencies[j].DependentRef.Kind.String() + "/" + canonicalDependencies[j].DependentRef.ID.String() + "/" +
			canonicalDependencies[j].PrerequisiteRef.Kind.String() + "/" + canonicalDependencies[j].PrerequisiteRef.ID.String()
		return left < right
	})
	payload, err := json.Marshal(struct {
		Nodes        []domain.RoadmapNode               `json:"nodes"`
		AfterLinks   []domain.RoadmapAfterLink          `json:"after_links"`
		Dependencies []domain.RoadmapDependencySnapshot `json:"dependency_snapshots"`
	}{
		Nodes:        canonicalNodes,
		AfterLinks:   canonicalLinks,
		Dependencies: canonicalDependencies,
	})
	if err != nil {
		return "", domain.WrapError(domain.ErrorCodeRoadmap, "roadmap revision content cannot be encoded", err)
	}
	sum := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func cloneRoadmapNode(node domain.RoadmapNode) domain.RoadmapNode {
	result := node
	result.CriterionRefs = append([]domain.RoadmapCriterionRef(nil), node.CriterionRefs...)
	result.CriterionSnapshots = append([]domain.RoadmapCriterionSnapshot(nil), node.CriterionSnapshots...)
	if node.TargetRef != nil {
		value := *node.TargetRef
		result.TargetRef = &value
	}
	if node.PlannedStart != nil {
		value := *node.PlannedStart
		result.PlannedStart = &value
	}
	if node.PlannedEnd != nil {
		value := *node.PlannedEnd
		result.PlannedEnd = &value
	}
	if node.ReferenceSnapshot != nil {
		value := *node.ReferenceSnapshot
		result.ReferenceSnapshot = &value
	}
	return result
}

func roadmapTargetTitle(
	ref domain.EntityRef,
	objectives map[domain.ID]domain.Objective,
	workItems map[domain.ID]domain.WorkItem,
) (string, error) {
	switch ref.Kind {
	case domain.EntityKindObjective:
		value, ok := objectives[ref.ID]
		if !ok {
			return "", domain.NewError(domain.ErrorCodeNotFound, "roadmap target Objective not found")
		}
		return value.Title, nil
	case domain.EntityKindWorkItem:
		value, ok := workItems[ref.ID]
		if !ok {
			return "", domain.NewError(domain.ErrorCodeNotFound, "roadmap target WorkItem not found")
		}
		return value.Title, nil
	default:
		return "", domain.NewError(domain.ErrorCodeRoadmap, "roadmap target kind is invalid")
	}
}

func roadmapCriterion(
	ref domain.RoadmapCriterionRef,
	outcome domain.Outcome,
	objectives map[domain.ID]domain.Objective,
	workItems map[domain.ID]domain.WorkItem,
) (domain.SuccessCriterion, error) {
	var criteria domain.CriterionSet
	switch ref.OwnerRef.Kind {
	case domain.EntityKindOutcome:
		criteria = outcome.Criteria
	case domain.EntityKindObjective:
		value, ok := objectives[ref.OwnerRef.ID]
		if !ok {
			return domain.SuccessCriterion{}, domain.NewError(domain.ErrorCodeNotFound, "roadmap criterion Objective not found")
		}
		criteria = value.Criteria
	case domain.EntityKindWorkItem:
		value, ok := workItems[ref.OwnerRef.ID]
		if !ok {
			return domain.SuccessCriterion{}, domain.NewError(domain.ErrorCodeNotFound, "roadmap criterion WorkItem not found")
		}
		criteria = value.Criteria
	default:
		return domain.SuccessCriterion{}, domain.NewError(domain.ErrorCodeRoadmap, "roadmap criterion owner kind is invalid")
	}
	criterion, err := criteria.Find(ref.CriterionID)
	if err != nil {
		return domain.SuccessCriterion{}, err
	}
	if criterion.Status != domain.CriterionStatusActive {
		return domain.SuccessCriterion{}, domain.NewError(domain.ErrorCodeRoadmap, "roadmap criterion must be active at publication")
	}
	return *criterion, nil
}

func criterionExists(criteria domain.CriterionSet, id domain.ID) bool {
	value, err := criteria.Find(id)
	return err == nil && value.Status == domain.CriterionStatusActive
}
