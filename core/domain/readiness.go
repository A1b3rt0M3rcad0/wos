package domain

import "time"

type ReadinessReason string

const (
	ReadinessReasonLifecycleNotReady         ReadinessReason = "lifecycle_not_ready"
	ReadinessReasonOutcomeNotActive          ReadinessReason = "outcome_not_active"
	ReadinessReasonOutcomeArchived           ReadinessReason = "outcome_archived"
	ReadinessReasonObjectiveTerminal         ReadinessReason = "objective_terminal"
	ReadinessReasonHardDependencyUnsatisfied ReadinessReason = "hard_dependency_unsatisfied"
	ReadinessReasonNotBefore                 ReadinessReason = "not_before"
	ReadinessReasonActiveLease               ReadinessReason = "active_lease"
)

type DependencyEvaluation struct {
	Relation  Relation `json:"relation"`
	Satisfied bool     `json:"satisfied"`
}

type Readiness struct {
	Ref         EntityRef         `json:"ref"`
	Ready       bool              `json:"ready"`
	Reasons     []ReadinessReason `json:"reasons,omitempty"`
	EvaluatedAt time.Time         `json:"evaluated_at"`
}

func WorkItemReadiness(item WorkItem, outcome Outcome, objective *Objective, dependencies []DependencyEvaluation, now time.Time) Readiness {
	now = now.UTC()
	reasons := make([]ReadinessReason, 0, 7)
	if item.Lifecycle != WorkItemLifecycleTodo {
		reasons = append(reasons, ReadinessReasonLifecycleNotReady)
	}
	if outcome.Lifecycle != OutcomeLifecycleActive {
		reasons = append(reasons, ReadinessReasonOutcomeNotActive)
	}
	if outcome.IsArchived() {
		reasons = append(reasons, ReadinessReasonOutcomeArchived)
	}
	if objective != nil && (objective.Lifecycle == ObjectiveLifecycleAchieved || objective.Lifecycle == ObjectiveLifecycleCancelled) {
		reasons = append(reasons, ReadinessReasonObjectiveTerminal)
	}
	if hasUnsatisfiedHardDependency(dependencies) {
		reasons = append(reasons, ReadinessReasonHardDependencyUnsatisfied)
	}
	if item.NotBefore != nil && item.NotBefore.After(now) {
		reasons = append(reasons, ReadinessReasonNotBefore)
	}
	if item.CurrentLease != nil && item.CurrentLease.ValidAt(now) {
		reasons = append(reasons, ReadinessReasonActiveLease)
	}
	return Readiness{Ref: item.Ref(), Ready: len(reasons) == 0, Reasons: reasons, EvaluatedAt: now}
}

func ObjectiveReadiness(objective Objective, outcome Outcome, dependencies []DependencyEvaluation, now time.Time) Readiness {
	now = now.UTC()
	reasons := make([]ReadinessReason, 0, 4)
	if objective.Lifecycle != ObjectiveLifecyclePlanned {
		reasons = append(reasons, ReadinessReasonLifecycleNotReady)
	}
	if outcome.Lifecycle != OutcomeLifecycleActive {
		reasons = append(reasons, ReadinessReasonOutcomeNotActive)
	}
	if outcome.IsArchived() {
		reasons = append(reasons, ReadinessReasonOutcomeArchived)
	}
	if hasUnsatisfiedHardDependency(dependencies) {
		reasons = append(reasons, ReadinessReasonHardDependencyUnsatisfied)
	}
	return Readiness{Ref: objective.Ref(), Ready: len(reasons) == 0, Reasons: reasons, EvaluatedAt: now}
}

func hasUnsatisfiedHardDependency(values []DependencyEvaluation) bool {
	for _, value := range values {
		if value.Relation.Lifecycle == RelationLifecycleActive &&
			value.Relation.RelationType == RelationTypeDependsOn &&
			value.Relation.Strength == DependencyStrengthHard &&
			!value.Satisfied {
			return true
		}
	}
	return false
}
