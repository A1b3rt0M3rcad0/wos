package application

import (
	"context"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type OutcomeState struct {
	ExternalReferences        []domain.ExternalReference        `json:"external_references"`
	ActiveRoadmaps            []ActivePlan                      `json:"active_roadmaps"`
	Outcome                   domain.Outcome                    `json:"outcome"`
	Objectives                []domain.Objective                `json:"objectives"`
	WorkItems                 []domain.WorkItem                 `json:"work_items"`
	WorkItemOperationalStates []domain.WorkItemOperationalState `json:"work_item_operational_states"`
	Relations                 []domain.Relation                 `json:"relations"`
	Issues                    []domain.Issue                    `json:"issues"`
	Blockers                  []domain.Blocker                  `json:"blockers"`
	Artifacts                 []domain.Artifact                 `json:"artifacts"`
	Evidence                  []domain.Evidence                 `json:"evidence"`
	EvidenceLinks             []domain.EvidenceLink             `json:"evidence_links"`
	Decisions                 []domain.Decision                 `json:"decisions"`
	ConclusionContested       bool                              `json:"conclusion_contested"`
	ConclusionContestations   []domain.ConclusionContestation   `json:"conclusion_contestations"`
	BlockingStates            []BlockingState                   `json:"blocking_states"`
	EvaluatedAt               time.Time                         `json:"evaluated_at"`
	OutcomeRevision           domain.OutcomeRevision            `json:"outcome_revision"`
}

type ReadResult[T any] struct {
	Value           T                      `json:"value"`
	OutcomeRevision domain.OutcomeRevision `json:"outcome_revision"`
}

type CriterionValidationHistory struct {
	Criterion           domain.SuccessCriterion              `json:"criterion"`
	DefinitionRevisions []domain.CriterionDefinitionRevision `json:"definition_revisions"`
	Assessments         []domain.CriterionAssessment         `json:"assessments"`
	CurrentAssessment   *domain.CriterionAssessment          `json:"current_assessment,omitempty"`
}

type ConclusionHistory struct {
	Current *domain.Conclusion  `json:"current,omitempty"`
	History []domain.Conclusion `json:"history"`
}

func (s *Service) GetOutcome(ctx context.Context, scope domain.Scope) (ReadResult[domain.Outcome], error) {
	if err := s.authorizeRead(ctx, scope.NamespaceID); err != nil {
		return ReadResult[domain.Outcome]{}, err
	}

	if err := scope.Validate(); err != nil {
		return ReadResult[domain.Outcome]{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[domain.Outcome]{}, err
	}
	defer uow.Rollback()

	value, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		return ReadResult[domain.Outcome]{}, err
	}
	coordination, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return ReadResult[domain.Outcome]{}, err
	}
	return ReadResult[domain.Outcome]{
		Value:           value,
		OutcomeRevision: coordination.Revision,
	}, nil
}

func (s *Service) GetObjective(ctx context.Context, scope domain.Scope, id domain.ID) (ReadResult[domain.Objective], error) {
	if err := s.authorizeRead(ctx, scope.NamespaceID); err != nil {
		return ReadResult[domain.Objective]{}, err
	}

	if err := scope.Validate(); err != nil {
		return ReadResult[domain.Objective]{}, err
	}
	if err := id.Validate(); err != nil {
		return ReadResult[domain.Objective]{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[domain.Objective]{}, err
	}
	defer uow.Rollback()

	value, err := uow.Objectives().Get(ctx, scope, id)
	if err != nil {
		return ReadResult[domain.Objective]{}, err
	}
	coordination, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return ReadResult[domain.Objective]{}, err
	}
	return ReadResult[domain.Objective]{
		Value:           value,
		OutcomeRevision: coordination.Revision,
	}, nil
}

func (s *Service) GetWorkItem(ctx context.Context, scope domain.Scope, id domain.ID) (ReadResult[domain.WorkItem], error) {
	if err := s.authorizeRead(ctx, scope.NamespaceID); err != nil {
		return ReadResult[domain.WorkItem]{}, err
	}

	if err := scope.Validate(); err != nil {
		return ReadResult[domain.WorkItem]{}, err
	}
	if err := id.Validate(); err != nil {
		return ReadResult[domain.WorkItem]{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[domain.WorkItem]{}, err
	}
	defer uow.Rollback()

	value, err := uow.WorkItems().Get(ctx, scope, id)
	if err != nil {
		return ReadResult[domain.WorkItem]{}, err
	}
	coordination, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return ReadResult[domain.WorkItem]{}, err
	}
	return ReadResult[domain.WorkItem]{
		Value:           value,
		OutcomeRevision: coordination.Revision,
	}, nil
}

func (s *Service) GetCriterionHistory(
	ctx context.Context,
	owner domain.EntityRef,
	criterionID domain.ID,
) (ReadResult[CriterionValidationHistory], error) {
	if err := s.authorizeRead(ctx, owner.NamespaceID); err != nil {
		return ReadResult[CriterionValidationHistory]{}, err
	}

	if err := owner.Validate(); err != nil {
		return ReadResult[CriterionValidationHistory]{}, err
	}
	if err := criterionID.Validate(); err != nil {
		return ReadResult[CriterionValidationHistory]{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[CriterionValidationHistory]{}, err
	}
	defer uow.Rollback()

	criteria, _, _, err := validationStateForOwner(ctx, uow, owner)
	if err != nil {
		return ReadResult[CriterionValidationHistory]{}, err
	}
	criterion, err := criteria.Find(criterionID)
	if err != nil {
		return ReadResult[CriterionValidationHistory]{}, err
	}
	value := CriterionValidationHistory{
		Criterion:           *criterion,
		DefinitionRevisions: make([]domain.CriterionDefinitionRevision, 0),
		Assessments:         make([]domain.CriterionAssessment, 0),
	}
	for _, revision := range criteria.DefinitionRevisions {
		if revision.CriterionID == criterionID {
			value.DefinitionRevisions = append(value.DefinitionRevisions, revision)
		}
	}
	for _, assessment := range criteria.Assessments {
		if assessment.CriterionID == criterionID {
			value.Assessments = append(value.Assessments, assessment)
		}
	}
	if current, ok := criteria.CurrentAssessments[criterionID]; ok {
		copyValue := current
		value.CurrentAssessment = &copyValue
	}
	coordination, err := uow.Coordination().LockOutcome(ctx, owner.Scope)
	if err != nil {
		return ReadResult[CriterionValidationHistory]{}, err
	}
	return ReadResult[CriterionValidationHistory]{
		Value:           value,
		OutcomeRevision: coordination.Revision,
	}, nil
}

func (s *Service) ListConclusions(
	ctx context.Context,
	owner domain.EntityRef,
) (ReadResult[ConclusionHistory], error) {
	if err := s.authorizeRead(ctx, owner.NamespaceID); err != nil {
		return ReadResult[ConclusionHistory]{}, err
	}

	if err := owner.Validate(); err != nil {
		return ReadResult[ConclusionHistory]{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[ConclusionHistory]{}, err
	}
	defer uow.Rollback()

	_, current, history, err := validationStateForOwner(ctx, uow, owner)
	if err != nil {
		return ReadResult[ConclusionHistory]{}, err
	}
	value := ConclusionHistory{
		Current: current,
		History: append([]domain.Conclusion(nil), history...),
	}
	coordination, err := uow.Coordination().LockOutcome(ctx, owner.Scope)
	if err != nil {
		return ReadResult[ConclusionHistory]{}, err
	}
	return ReadResult[ConclusionHistory]{
		Value:           value,
		OutcomeRevision: coordination.Revision,
	}, nil
}

func (s *Service) GetConclusion(
	ctx context.Context,
	owner domain.EntityRef,
	conclusionID domain.ID,
) (ReadResult[domain.Conclusion], error) {
	if err := s.authorizeRead(ctx, owner.NamespaceID); err != nil {
		return ReadResult[domain.Conclusion]{}, err
	}

	if err := owner.Validate(); err != nil {
		return ReadResult[domain.Conclusion]{}, err
	}
	if err := conclusionID.Validate(); err != nil {
		return ReadResult[domain.Conclusion]{}, err
	}
	history, err := s.ListConclusions(ctx, owner)
	if err != nil {
		return ReadResult[domain.Conclusion]{}, err
	}
	if history.Value.Current != nil && history.Value.Current.ID == conclusionID {
		return ReadResult[domain.Conclusion]{
			Value:           *history.Value.Current,
			OutcomeRevision: history.OutcomeRevision,
		}, nil
	}
	for _, conclusion := range history.Value.History {
		if conclusion.ID == conclusionID {
			return ReadResult[domain.Conclusion]{
				Value:           conclusion,
				OutcomeRevision: history.OutcomeRevision,
			}, nil
		}
	}
	return ReadResult[domain.Conclusion]{}, domain.NewError(domain.ErrorCodeNotFound, "conclusion not found")
}

func validationStateForOwner(
	ctx context.Context,
	uow ports.UnitOfWork,
	owner domain.EntityRef,
) (domain.CriterionSet, *domain.Conclusion, []domain.Conclusion, error) {
	switch owner.Kind {
	case domain.EntityKindOutcome:
		value, err := uow.Outcomes().Get(ctx, owner.Scope.NamespaceID, owner.Scope.OutcomeID)
		if err != nil {
			return domain.CriterionSet{}, nil, nil, err
		}
		return value.Criteria, value.CurrentConclusion, value.ConclusionHistory, nil
	case domain.EntityKindObjective:
		value, err := uow.Objectives().Get(ctx, owner.Scope, owner.ID)
		if err != nil {
			return domain.CriterionSet{}, nil, nil, err
		}
		return value.Criteria, value.CurrentConclusion, value.ConclusionHistory, nil
	case domain.EntityKindWorkItem:
		value, err := uow.WorkItems().Get(ctx, owner.Scope, owner.ID)
		if err != nil {
			return domain.CriterionSet{}, nil, nil, err
		}
		return value.Criteria, value.CurrentConclusion, value.ConclusionHistory, nil
	default:
		return domain.CriterionSet{}, nil, nil, domain.NewError(
			domain.ErrorCodeInvalidArgument,
			"owner kind does not expose validation history",
		)
	}
}

func (s *Service) GetRelation(ctx context.Context, scope domain.Scope, id domain.ID) (ReadResult[domain.Relation], error) {
	if err := s.authorizeRead(ctx, scope.NamespaceID); err != nil {
		return ReadResult[domain.Relation]{}, err
	}

	if err := scope.Validate(); err != nil {
		return ReadResult[domain.Relation]{}, err
	}
	if err := id.Validate(); err != nil {
		return ReadResult[domain.Relation]{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return ReadResult[domain.Relation]{}, err
	}
	defer uow.Rollback()

	value, err := uow.Relations().Get(ctx, scope, id)
	if err != nil {
		return ReadResult[domain.Relation]{}, err
	}
	coordination, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return ReadResult[domain.Relation]{}, err
	}
	return ReadResult[domain.Relation]{
		Value:           value,
		OutcomeRevision: coordination.Revision,
	}, nil
}

func (s *Service) ListRelations(ctx context.Context, scope domain.Scope) ([]domain.Relation, domain.OutcomeRevision, error) {
	if err := s.authorizeRead(ctx, scope.NamespaceID); err != nil {
		return nil, 0, err
	}

	if err := scope.Validate(); err != nil {
		return nil, 0, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer uow.Rollback()

	values, err := uow.Relations().ListByOutcome(ctx, scope)
	if err != nil {
		return nil, 0, err
	}
	coordination, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return nil, 0, err
	}
	return values, coordination.Revision, nil
}

func (s *Service) GetOutcomeState(ctx context.Context, scope domain.Scope) (result OutcomeState, queryErr error) {
	defer s.observeQuery(ctx, "outcome_state", scope, &result.OutcomeRevision, &queryErr, time.Now())
	if err := s.authorizeRead(ctx, scope.NamespaceID); err != nil {
		return OutcomeState{}, err
	}

	if err := scope.Validate(); err != nil {
		return OutcomeState{}, err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return OutcomeState{}, err
	}
	defer uow.Rollback()

	outcome, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		return OutcomeState{}, err
	}
	objectives, err := uow.Objectives().ListByOutcome(ctx, scope)
	if err != nil {
		return OutcomeState{}, err
	}
	workItems, err := uow.WorkItems().ListByOutcome(ctx, scope)
	if err != nil {
		return OutcomeState{}, err
	}
	relations, err := uow.Relations().ListByOutcome(ctx, scope)
	if err != nil {
		return OutcomeState{}, err
	}
	issuesRepo, blockersRepo, err := issueBlockerRepositories(uow)
	if err != nil {
		return OutcomeState{}, err
	}
	issues, err := issuesRepo.ListByOutcome(ctx, scope)
	if err != nil {
		return OutcomeState{}, err
	}
	blockers, err := blockersRepo.ListByOutcome(ctx, scope)
	if err != nil {
		return OutcomeState{}, err
	}
	artifactsRepo, evidenceRepo, evidenceLinksRepo, decisionsRepo, err := documentaryRepositories(uow)
	if err != nil {
		return OutcomeState{}, err
	}
	artifacts, err := artifactsRepo.ListByOutcome(ctx, scope)
	if err != nil {
		return OutcomeState{}, err
	}
	evidence, err := evidenceRepo.ListByOutcome(ctx, scope)
	if err != nil {
		return OutcomeState{}, err
	}
	evidenceLinks, err := evidenceLinksRepo.ListByOutcome(ctx, scope)
	if err != nil {
		return OutcomeState{}, err
	}
	decisions, err := decisionsRepo.ListByOutcome(ctx, scope)
	if err != nil {
		return OutcomeState{}, err
	}

	objectiveByID := make(map[domain.ID]domain.Objective, len(objectives))
	workByID := make(map[domain.ID]domain.WorkItem, len(workItems))
	completed := make(map[domain.EntityRef]bool, len(objectives)+len(workItems))
	for _, objective := range objectives {
		objectiveByID[objective.ID] = objective
		completed[objective.Ref()] = objective.Lifecycle == domain.ObjectiveLifecycleAchieved
	}
	for _, item := range workItems {
		workByID[item.ID] = item
		completed[item.Ref()] = item.Lifecycle == domain.WorkItemLifecycleDone
	}
	dependenciesBySource := make(map[domain.EntityRef][]domain.DependencyEvaluation)
	for _, relation := range relations {
		if relation.Lifecycle != domain.RelationLifecycleActive || relation.RelationType != domain.RelationTypeDependsOn {
			continue
		}
		satisfied, exists := completed[relation.TargetRef]
		if !exists {
			return OutcomeState{}, domain.NewError(domain.ErrorCodeNotFound, "dependency target is missing from outcome snapshot")
		}
		dependenciesBySource[relation.SourceRef] = append(dependenciesBySource[relation.SourceRef], domain.DependencyEvaluation{Relation: relation, Satisfied: satisfied})
	}
	now, err := s.transactionTime(ctx, uow)
	if err != nil {
		return OutcomeState{}, err
	}
	evaluatedAt := evaluationTime(ctx, now)
	blockingStates := make([]BlockingState, 0, 1+len(objectives)+len(workItems))
	blockingByRef := make(map[domain.EntityRef]BlockingState, 1+len(objectives)+len(workItems))
	refs := make([]domain.EntityRef, 0, 1+len(objectives)+len(workItems))
	refs = append(refs, outcome.Ref())
	for _, objective := range objectives {
		refs = append(refs, objective.Ref())
	}
	for _, workItem := range workItems {
		refs = append(refs, workItem.Ref())
	}
	for _, ref := range refs {
		var parent *domain.ID
		if ref.Kind == domain.EntityKindObjective {
			id := ref.ID
			parent = &id
		}
		if ref.Kind == domain.EntityKindWorkItem {
			parent = workByID[ref.ID].ObjectiveID
		}
		state := projectBlockingState(ref, parent, blockers, objectiveByID)
		blockingStates = append(blockingStates, state)
		blockingByRef[ref] = state
	}

	workOperationalStates := make([]domain.WorkItemOperationalState, 0, len(workItems))
	for _, item := range workItems {
		var objective *domain.Objective
		if item.ObjectiveID != nil {
			value, ok := objectiveByID[*item.ObjectiveID]
			if !ok {
				return OutcomeState{}, domain.NewError(domain.ErrorCodeNotFound, "work item objective is missing")
			}
			copyValue := value
			objective = &copyValue
		}
		dependencies := dependenciesBySource[item.Ref()]
		blocking := blockingByRef[item.Ref()]
		workOperationalStates = append(workOperationalStates, domain.ProjectWorkItemOperationalState(
			item,
			outcome,
			objective,
			dependencies,
			blocking.IsBlocked,
			evaluatedAt,
		))
	}

	evidenceByID := make(map[domain.ID]domain.Evidence, len(evidence))
	for _, item := range evidence {
		evidenceByID[item.ID] = item
	}
	contestations := make([]domain.ConclusionContestation, 0)
	contestations = append(contestations, conclusionContestations(
		outcome.Ref(), outcome.CurrentConclusion, outcome.Criteria, evidenceByID,
	)...)
	if outcome.CurrentConclusion != nil {
		recorded := map[domain.ID]bool{}
		for _, id := range outcome.CurrentConclusion.Obligations.RequiredObjectiveIDs {
			recorded[id] = true
			objective, ok := objectiveByID[id]
			if !ok || objective.Lifecycle != domain.ObjectiveLifecycleAchieved {
				copyID := id
				contestations = append(contestations, domain.ConclusionContestation{OwnerRef: outcome.Ref(), Kind: domain.ConclusionContestationRequiredObjective, ObjectiveID: &copyID})
			}
		}
		for _, objective := range objectives {
			if objective.RequiredForOutcome && !recorded[objective.ID] {
				id := objective.ID
				contestations = append(contestations, domain.ConclusionContestation{OwnerRef: outcome.Ref(), Kind: domain.ConclusionContestationObligationsChanged, ObjectiveID: &id})
			}
		}
	}
	for _, objective := range objectives {
		contestations = append(contestations, conclusionContestations(
			objective.Ref(), objective.CurrentConclusion, objective.Criteria, evidenceByID,
		)...)
	}
	for _, item := range workItems {
		contestations = append(contestations, conclusionContestations(
			item.Ref(), item.CurrentConclusion, item.Criteria, evidenceByID,
		)...)
	}

	activePlans := []ActivePlan{}
	if planning, ok := uow.(ports.PlanningUnitOfWork); ok {
		plans, err := planning.Roadmaps().ListByOutcome(ctx, scope)
		if err != nil {
			return OutcomeState{}, err
		}
		for _, plan := range plans {
			slot, err := planning.RoadmapActivations().GetActive(ctx, scope, plan.PlanScope)
			if err != nil {
				return OutcomeState{}, err
			}
			if slot != nil && slot.RoadmapID == plan.ID {
				if slot.RevisionNumber == 0 || slot.RevisionNumber > uint64(len(plan.Revisions)) {
					return OutcomeState{}, domain.NewError(domain.ErrorCodeRoadmap, "active roadmap revision is missing")
				}
				activePlans = append(activePlans, ActivePlan{Slot: *slot, Revision: plan.Revisions[slot.RevisionNumber-1], LiveReferences: projectPlanReferences(*slot, plan.Revisions[slot.RevisionNumber-1], objectives, workItems, workOperationalStates)})
			}
		}
	}
	coordination, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return OutcomeState{}, err
	}
	var externalReferences []domain.ExternalReference
	if ext, ok := uow.(ports.ExternalContextUnitOfWork); ok {
		externalReferences, err = ext.ExternalContexts().List(ctx, scope)
		if err != nil {
			return OutcomeState{}, err
		}
	}
	return OutcomeState{
		ExternalReferences:        externalReferences,
		ActiveRoadmaps:            activePlans,
		Outcome:                   outcome,
		Objectives:                objectives,
		WorkItems:                 workItems,
		WorkItemOperationalStates: workOperationalStates,
		Relations:                 relations,
		Issues:                    issues,
		Blockers:                  blockers,
		Artifacts:                 artifacts,
		Evidence:                  evidence,
		EvidenceLinks:             evidenceLinks,
		Decisions:                 decisions,
		ConclusionContested:       len(contestations) > 0,
		ConclusionContestations:   contestations,
		BlockingStates:            blockingStates,
		EvaluatedAt:               evaluatedAt,
		OutcomeRevision:           coordination.Revision,
	}, nil
}

func conclusionContestations(
	owner domain.EntityRef,
	conclusion *domain.Conclusion,
	criteria domain.CriterionSet,
	evidenceByID map[domain.ID]domain.Evidence,
) []domain.ConclusionContestation {
	if conclusion == nil {
		return nil
	}
	assessmentByID := make(map[domain.ID]domain.CriterionAssessment, len(criteria.Assessments))
	for _, assessment := range criteria.Assessments {
		assessmentByID[assessment.ID] = assessment
	}
	result := make([]domain.ConclusionContestation, 0)
	recorded := map[domain.ID]domain.CriterionRevision{}
	for _, obligation := range conclusion.Obligations.RequiredCriteria {
		recorded[obligation.CriterionID] = obligation.CriterionRevision
		criterion, err := criteria.Find(obligation.CriterionID)
		if err != nil || criterion.Revision != obligation.CriterionRevision || criterion.Status != domain.CriterionStatusActive || !criterion.Required {
			result = append(result, domain.ConclusionContestation{OwnerRef: owner, Kind: domain.ConclusionContestationObligationsChanged, CriterionID: obligation.CriterionID})
		}
	}
	for _, criterion := range criteria.Items {
		if criterion.Required && criterion.Status == domain.CriterionStatusActive {
			if _, ok := recorded[criterion.ID]; !ok {
				result = append(result, domain.ConclusionContestation{OwnerRef: owner, Kind: domain.ConclusionContestationObligationsChanged, CriterionID: criterion.ID})
			}
		}
	}
	for _, ref := range conclusion.Assessments {
		if current, ok := criteria.CurrentAssessments[ref.CriterionID]; ok &&
			current.ID != ref.AssessmentID &&
			(current.Result == domain.AssessmentResultNotMet || current.Result == domain.AssessmentResultInconclusive) {
			result = append(result, domain.ConclusionContestation{
				OwnerRef:     owner,
				Kind:         domain.ConclusionContestationAssessmentContradiction,
				CriterionID:  ref.CriterionID,
				AssessmentID: current.ID,
			})
		}

		assessment, ok := assessmentByID[ref.AssessmentID]
		if !ok {
			continue
		}
		for _, evidenceID := range assessment.EvidenceIDs {
			evidence, ok := evidenceByID[evidenceID]
			if !ok || evidence.Lifecycle != domain.EvidenceLifecycleRetracted {
				continue
			}
			id := evidenceID
			result = append(result, domain.ConclusionContestation{
				OwnerRef:     owner,
				Kind:         domain.ConclusionContestationEvidenceRetracted,
				CriterionID:  ref.CriterionID,
				AssessmentID: ref.AssessmentID,
				EvidenceID:   &id,
			})
		}
	}
	return result
}
