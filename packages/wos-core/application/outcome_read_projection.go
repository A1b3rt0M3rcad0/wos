package application

import "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"

// Shared pure classification over one coherent read. Projection adapters may
// supply minimal headers and relevant current proof; this is not a write path.
func projectOutcomeReadState(state OutcomeState) (OutcomeState, error) {
	outcome, objectives, workItems := state.Outcome, state.Objectives, state.WorkItems
	relations, blockers, evidence := state.Relations, state.Blockers, state.Evidence
	evaluatedAt := state.EvaluatedAt
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

	state.WorkItemOperationalStates = workOperationalStates
	state.BlockingStates = blockingStates
	state.ConclusionContested = len(contestations) > 0
	state.ConclusionContestations = contestations
	return state, nil
}
