package application

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// Objective hierarchy is validated on the authorized Outcome snapshot. The
// existing graph limit is also the fail-closed bound for a corrupt parent chain.
const operationalAncestryLimit = domain.DefaultDependencyGraphLimit

// evaluateObjectiveExecutionGate is the shared Objective start/readiness
// predicate. It does not mutate state or authorize a caller; callers still
// perform their normal permission checks and transaction writes.
func evaluateObjectiveExecutionGate(
	ctx context.Context,
	uow ports.UnitOfWork,
	objective domain.Objective,
	now time.Time,
) (domain.Readiness, error) {
	outcome, err := uow.Outcomes().Get(ctx, objective.Scope.NamespaceID, objective.Scope.OutcomeID)
	if err != nil {
		return domain.Readiness{}, err
	}
	if err := validateOperationalProfile(outcome); err != nil {
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
	readiness := domain.ObjectiveReadiness(objective, outcome, evaluations, now)
	addUnsatisfiedDependencyRefs(&readiness, evaluations)
	if outcome.OperationalProfile.Effective() == domain.OperationalProfileOperationalV2 {
		if objective.Lifecycle == domain.ObjectiveLifecyclePlanned {
			if err := objective.CheckStartDefinition(); err != nil {
				addDomainPrecondition(&readiness, domain.ReadinessStageDefinition, err)
			}
		}
		if err := addObjectiveAncestorReadiness(ctx, uow, outcome.Scope(), objective, &readiness); err != nil {
			return domain.Readiness{}, err
		}
	}
	blocking, err := blockingStateForRef(ctx, uow, objective.Ref())
	if err != nil {
		return domain.Readiness{}, err
	}
	addBlockingReasons(&readiness, blocking)
	return readiness, nil
}

// evaluateTaskActivation checks the authored definition separately from its
// execution scope. It deliberately ignores Task lifecycle, dependencies,
// schedule, and authority because these are acquisition-stage conditions.
func evaluateTaskActivation(
	ctx context.Context,
	uow ports.UnitOfWork,
	item domain.WorkItem,
	now time.Time,
) (domain.Readiness, error) {
	outcome, err := uow.Outcomes().Get(ctx, item.Scope.NamespaceID, item.Scope.OutcomeID)
	if err != nil {
		return domain.Readiness{}, err
	}
	if err := validateOperationalProfile(outcome); err != nil {
		return domain.Readiness{}, err
	}
	readiness := domain.NewReadiness(item.Ref(), now)
	if outcome.OperationalProfile.Effective() != domain.OperationalProfileOperationalV2 {
		// compatibility_v1 retains the historical activation contract. Migration
		// of these records is an explicit operation, not an implicit gate change.
		return readiness, nil
	}

	if err := item.CheckActivationDefinition(); err != nil {
		addDomainPrecondition(&readiness, domain.ReadinessStageDefinition, err)
	}
	addOutcomeEligibility(&readiness, outcome)
	if item.ObjectiveID == nil {
		readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonTaskObjectiveRequired)
	} else {
		objective, err := uow.Objectives().Get(ctx, item.Scope, *item.ObjectiveID)
		if err != nil {
			if code, _ := domain.ErrorCodeOf(err); code == domain.ErrorCodeNotFound {
				readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonObjectiveAncestryMissing)
			} else {
				return domain.Readiness{}, err
			}
		} else if objective.Scope != outcome.Scope() || objective.Validate() != nil {
			readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonObjectiveAncestryCorrupt)
		} else {
			if objective.Lifecycle == domain.ObjectiveLifecyclePlanned {
				readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonObjectiveNotStarted, objective.Ref())
			} else if objective.Lifecycle == domain.ObjectiveLifecycleAchieved || objective.Lifecycle == domain.ObjectiveLifecycleCancelled {
				readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonObjectiveTerminal, objective.Ref())
			}
			if err := addObjectiveAncestorReadiness(ctx, uow, item.Scope, objective, &readiness); err != nil {
				return domain.Readiness{}, err
			}
		}
	}
	blocking, err := blockingStateForRef(ctx, uow, item.Ref())
	if err != nil {
		return domain.Readiness{}, err
	}
	addBlockingReasons(&readiness, blocking)
	return readiness, nil
}

// evaluateTaskAcquisition composes activation eligibility with current
// dependency, schedule, lifecycle, blocker, and lease facts. A valid contract
// remains protocol-specific and is checked by its acquisition command.
func evaluateTaskAcquisition(
	ctx context.Context,
	uow ports.UnitOfWork,
	item domain.WorkItem,
	now time.Time,
) (domain.Readiness, error) {
	outcome, err := uow.Outcomes().Get(ctx, item.Scope.NamespaceID, item.Scope.OutcomeID)
	if err != nil {
		return domain.Readiness{}, err
	}
	if err := validateOperationalProfile(outcome); err != nil {
		return domain.Readiness{}, err
	}
	profile := outcome.OperationalProfile.Effective()
	var objective *domain.Objective
	if item.ObjectiveID != nil {
		value, err := uow.Objectives().Get(ctx, item.Scope, *item.ObjectiveID)
		if err != nil {
			if profile != domain.OperationalProfileOperationalV2 {
				return domain.Readiness{}, err
			}
			if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeNotFound {
				return domain.Readiness{}, err
			}
		} else {
			objective = &value
		}
	}
	relations, err := uow.Relations().ListByOutcome(ctx, item.Scope)
	if err != nil {
		return domain.Readiness{}, err
	}
	evaluations, err := dependencyEvaluationsForSource(ctx, uow, item.Ref(), relations)
	if err != nil {
		return domain.Readiness{}, err
	}
	readiness := domain.WorkItemReadiness(item, outcome, objective, evaluations, now)
	addUnsatisfiedDependencyRefs(&readiness, evaluations)
	if profile == domain.OperationalProfileOperationalV2 {
		if err := item.CheckActivationDefinition(); err != nil {
			addDomainPrecondition(&readiness, domain.ReadinessStageDefinition, err)
		}
		if item.ObjectiveID == nil {
			readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonTaskObjectiveRequired)
		} else if objective == nil {
			readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonObjectiveAncestryMissing)
		} else if objective.Scope != outcome.Scope() || objective.Validate() != nil {
			readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonObjectiveAncestryCorrupt)
		} else {
			if objective.Lifecycle == domain.ObjectiveLifecyclePlanned {
				readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonObjectiveNotStarted, objective.Ref())
			} else if objective.Lifecycle == domain.ObjectiveLifecycleAchieved || objective.Lifecycle == domain.ObjectiveLifecycleCancelled {
				readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonObjectiveTerminal, objective.Ref())
			}
			if err := addObjectiveAncestorReadiness(ctx, uow, item.Scope, *objective, &readiness); err != nil {
				return domain.Readiness{}, err
			}
		}
	}
	blocking, err := blockingStateForRef(ctx, uow, item.Ref())
	if err != nil {
		return domain.Readiness{}, err
	}
	addBlockingReasons(&readiness, blocking)
	return readiness, nil
}

func validateOperationalProfile(outcome domain.Outcome) error {
	if outcome.OperationalProfile.Valid() {
		return nil
	}
	ref := outcome.Ref()
	return domain.NewDetailedError(domain.ErrorCodeInvalidArgument, "Outcome has an unknown operational profile", domain.ErrorDetails{
		ReasonCode: string(domain.ReadinessReasonOperationalProfileUnknown), SubjectRef: &ref,
	})
}

func addOutcomeEligibility(readiness *domain.Readiness, outcome domain.Outcome) {
	if outcome.Lifecycle != domain.OutcomeLifecycleActive {
		readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonOutcomeNotActive, outcome.Ref())
	}
	if outcome.IsArchived() {
		readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonOutcomeArchived, outcome.Ref())
	}
}

func loadObjectiveAncestors(
	ctx context.Context,
	uow ports.UnitOfWork,
	scope domain.Scope,
	objective domain.Objective,
) ([]domain.Objective, domain.ReadinessReason, error) {
	if objective.Scope != scope || objective.ID.Validate() != nil || objective.Validate() != nil {
		return nil, domain.ReadinessReasonObjectiveAncestryCorrupt, nil
	}
	objectives, err := uow.Objectives().ListByOutcome(ctx, scope)
	if err != nil {
		return nil, "", err
	}
	byID := make(map[domain.ID]domain.Objective, len(objectives))
	invalidByID := make(map[domain.ID]bool)
	duplicateIDs := make(map[domain.ID]bool)
	for _, value := range objectives {
		if value.Scope != scope || value.ID.Validate() != nil {
			continue
		}
		if _, exists := byID[value.ID]; exists {
			duplicateIDs[value.ID] = true
			continue
		}
		byID[value.ID] = value
		invalidByID[value.ID] = value.Validate() != nil
	}
	if _, exists := byID[objective.ID]; !exists {
		return nil, domain.ReadinessReasonObjectiveAncestryCorrupt, nil
	}

	ancestors := make([]domain.Objective, 0)
	visited := map[domain.ID]struct{}{objective.ID: {}}
	current := objective
	for depth := 0; current.ParentObjectiveID != nil; depth++ {
		if depth >= operationalAncestryLimit {
			return ancestors, domain.ReadinessReasonObjectiveAncestryIncomplete, nil
		}
		parentID := *current.ParentObjectiveID
		if parentID.Validate() != nil {
			return ancestors, domain.ReadinessReasonObjectiveAncestryCorrupt, nil
		}
		if _, seen := visited[parentID]; seen {
			if parent, ok := byID[parentID]; ok {
				ancestors = append(ancestors, parent)
			}
			return ancestors, domain.ReadinessReasonObjectiveAncestryCycle, nil
		}
		parent, exists := byID[parentID]
		if !exists {
			return ancestors, domain.ReadinessReasonObjectiveAncestryMissing, nil
		}
		if duplicateIDs[parentID] || invalidByID[parentID] || parent.ID != parentID || parent.Scope != scope || parent.Validate() != nil {
			return ancestors, domain.ReadinessReasonObjectiveAncestryCorrupt, nil
		}
		ancestors = append(ancestors, parent)
		visited[parentID] = struct{}{}
		current = parent
	}
	return ancestors, "", nil
}

func addObjectiveAncestorReadiness(
	ctx context.Context,
	uow ports.UnitOfWork,
	scope domain.Scope,
	objective domain.Objective,
	readiness *domain.Readiness,
) error {
	ancestors, issue, err := loadObjectiveAncestors(ctx, uow, scope, objective)
	if err != nil {
		return err
	}
	if issue != "" {
		readiness.AddReason(domain.ReadinessStageScope, issue)
	}
	for _, ancestor := range ancestors {
		if ancestor.Lifecycle == domain.ObjectiveLifecycleInProgress {
			continue
		}
		reason := domain.ReadinessReasonAncestorObjectiveNotStarted
		if ancestor.Lifecycle == domain.ObjectiveLifecycleAchieved || ancestor.Lifecycle == domain.ObjectiveLifecycleCancelled {
			reason = domain.ReadinessReasonAncestorObjectiveTerminal
		}
		readiness.AddReason(domain.ReadinessStageScope, reason, ancestor.Ref())
	}
	return nil
}

func addDomainPrecondition(readiness *domain.Readiness, stage domain.ReadinessStage, err error) {
	var typed *domain.Error
	if !errors.As(err, &typed) || typed == nil || typed.Details == nil || typed.Details.ReasonCode == "" {
		readiness.AddReason(stage, domain.ReadinessReasonTaskDefinitionIncomplete)
		return
	}
	readiness.AddReason(stage, domain.ReadinessReason(typed.Details.ReasonCode))
}

func addUnsatisfiedDependencyRefs(readiness *domain.Readiness, values []domain.DependencyEvaluation) {
	for _, value := range values {
		if value.Relation.Lifecycle == domain.RelationLifecycleActive &&
			value.Relation.RelationType == domain.RelationTypeDependsOn &&
			value.Relation.Strength == domain.DependencyStrengthHard && !value.Satisfied {
			readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonHardDependencyUnsatisfied, value.Relation.TargetRef)
		}
	}
}

func addBlockingReasons(readiness *domain.Readiness, state BlockingState) {
	if !state.IsBlocked {
		return
	}
	for _, value := range state.ActiveBlockers {
		refs := []domain.EntityRef{value.Blocker.Ref()}
		if value.InheritedFrom != nil {
			refs = append(refs, *value.InheritedFrom)
		}
		readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonBlocked, refs...)
	}
	if len(state.ActiveBlockers) == 0 {
		readiness.AddReason(domain.ReadinessStageScope, domain.ReadinessReasonBlocked)
	}
}

func operationalReadinessError(readiness domain.Readiness, prefix string) error {
	if readiness.Ready {
		return nil
	}
	reason := domain.ReadinessReasonLifecycleNotReady
	if len(readiness.Reasons) > 0 {
		reason = readiness.Reasons[0]
	}
	message := prefix + ": " + strings.Join(readinessReasonStrings(readiness.Reasons), ",")
	remedies := readinessRemedies(reason, readiness)
	subject := readiness.Ref
	return domain.NewDetailedError(domain.ErrorCodePreconditionFailed, message, domain.ErrorDetails{
		ReasonCode: string(reason), SubjectRef: &subject, RelatedRefs: append([]domain.EntityRef(nil), readiness.RelatedRefs...), Remedies: remedies,
	})
}

func readinessReasonStrings(values []domain.ReadinessReason) []string {
	result := make([]string, len(values))
	for i, value := range values {
		result[i] = string(value)
	}
	return result
}

func readinessRemedies(reason domain.ReadinessReason, readiness domain.Readiness) []domain.ErrorRemedy {
	var target *domain.EntityRef
	for i := range readiness.RelatedRefs {
		ref := readiness.RelatedRefs[i]
		if ref.Kind == domain.EntityKindObjective && (reason == domain.ReadinessReasonObjectiveNotStarted || reason == domain.ReadinessReasonAncestorObjectiveNotStarted) {
			target = &ref
			break
		}
	}
	switch reason {
	case domain.ReadinessReasonObjectiveNotStarted, domain.ReadinessReasonAncestorObjectiveNotStarted:
		if target != nil {
			return []domain.ErrorRemedy{{ActionID: "StartObjective", SubjectRef: target}}
		}
		return []domain.ErrorRemedy{{ActionID: "inspect", SubjectRef: &readiness.Ref}}
	case domain.ReadinessReasonTaskObjectiveRequired:
		return []domain.ErrorRemedy{{ActionID: "select_objective", SubjectRef: &readiness.Ref}}
	case domain.ReadinessReasonTaskDefinitionIncomplete:
		return []domain.ErrorRemedy{{ActionID: "add_criterion", SubjectRef: &readiness.Ref}}
	case domain.ReadinessReason("missing_required_criterion"):
		return []domain.ErrorRemedy{{ActionID: "add_criterion", SubjectRef: &readiness.Ref}}
	case domain.ReadinessReasonHardDependencyUnsatisfied, domain.ReadinessReasonBlocked,
		domain.ReadinessReasonObjectiveAncestryMissing, domain.ReadinessReasonObjectiveAncestryCycle,
		domain.ReadinessReasonObjectiveAncestryCorrupt, domain.ReadinessReasonObjectiveAncestryIncomplete:
		return []domain.ErrorRemedy{{ActionID: "inspect", SubjectRef: &readiness.Ref}}
	default:
		return nil
	}
}
