package domain

import "time"

type ReadinessReason string

const (
	ReadinessReasonLifecycleNotReady           ReadinessReason = "lifecycle_not_ready"
	ReadinessReasonOutcomeNotActive            ReadinessReason = "outcome_not_active"
	ReadinessReasonOutcomeArchived             ReadinessReason = "outcome_archived"
	ReadinessReasonObjectiveTerminal           ReadinessReason = "objective_terminal"
	ReadinessReasonObjectiveNotStarted         ReadinessReason = "objective_not_started"
	ReadinessReasonAncestorObjectiveNotStarted ReadinessReason = "ancestor_objective_not_started"
	ReadinessReasonAncestorObjectiveTerminal   ReadinessReason = "ancestor_objective_terminal"
	ReadinessReasonTaskObjectiveRequired       ReadinessReason = "task_objective_required"
	ReadinessReasonTaskDefinitionIncomplete    ReadinessReason = "task_definition_incomplete"
	ReadinessReasonObjectiveAncestryMissing    ReadinessReason = "objective_ancestry_missing"
	ReadinessReasonObjectiveAncestryCycle      ReadinessReason = "objective_ancestry_cycle"
	ReadinessReasonObjectiveAncestryCorrupt    ReadinessReason = "objective_ancestry_corrupt"
	ReadinessReasonObjectiveAncestryIncomplete ReadinessReason = "objective_ancestry_incomplete"
	ReadinessReasonOperationalProfileUnknown   ReadinessReason = "operational_profile_unknown"
	ReadinessReasonBlocked                     ReadinessReason = "blocked"
	ReadinessReasonHardDependencyUnsatisfied   ReadinessReason = "hard_dependency_unsatisfied"
	ReadinessReasonNotBefore                   ReadinessReason = "not_before"
	ReadinessReasonActiveLease                 ReadinessReason = "active_lease"
)

// ReadinessStage keeps planning completeness, scope eligibility, and exclusive
// authority separate. These dimensions are internal to Core until public query
// contracts deliberately expose them; the legacy JSON shape remains unchanged.
type ReadinessStage string

const (
	ReadinessStageDefinition ReadinessStage = "definition"
	ReadinessStageScope      ReadinessStage = "scope"
	ReadinessStageAuthority  ReadinessStage = "authority"
)

type ReadinessDimension struct {
	Ready   bool
	Reasons []ReadinessReason
}

type DependencyEvaluation struct {
	Relation  Relation `json:"relation"`
	Satisfied bool     `json:"satisfied"`
}

type Readiness struct {
	Ref         EntityRef          `json:"ref"`
	Ready       bool               `json:"ready"`
	Reasons     []ReadinessReason  `json:"reasons,omitempty"`
	EvaluatedAt time.Time          `json:"evaluated_at"`
	Definition  ReadinessDimension `json:"-"`
	Scope       ReadinessDimension `json:"-"`
	Authority   ReadinessDimension `json:"-"`
	RelatedRefs []EntityRef        `json:"-"`
}

func NewReadiness(ref EntityRef, now time.Time) Readiness {
	return Readiness{
		Ref: ref, Ready: true, EvaluatedAt: now.UTC(),
		Definition: ReadinessDimension{Ready: true},
		Scope:      ReadinessDimension{Ready: true},
		Authority:  ReadinessDimension{Ready: true},
	}
}

// AddReason records a stable reason in its operational dimension and preserves
// the historic flat Reasons field used by existing projections.
func (r *Readiness) AddReason(stage ReadinessStage, reason ReadinessReason, related ...EntityRef) {
	if r == nil || reason == "" {
		return
	}
	var dimension *ReadinessDimension
	switch stage {
	case ReadinessStageDefinition:
		dimension = &r.Definition
	case ReadinessStageScope:
		dimension = &r.Scope
	case ReadinessStageAuthority:
		dimension = &r.Authority
	default:
		return
	}
	dimension.Ready = false
	if !containsReadinessReason(dimension.Reasons, reason) {
		dimension.Reasons = append(dimension.Reasons, reason)
	}
	if !containsReadinessReason(r.Reasons, reason) {
		r.Reasons = append(r.Reasons, reason)
	}
	for _, ref := range related {
		// Readiness is evaluated inside an authorized Outcome scope. Never attach
		// a related object from another scope or a malformed reference.
		if ref.Scope != r.Ref.Scope || ref.Validate() != nil {
			continue
		}
		found := false
		for _, existing := range r.RelatedRefs {
			if existing == ref {
				found = true
				break
			}
		}
		if !found {
			r.RelatedRefs = append(r.RelatedRefs, ref)
		}
	}
	r.Ready = r.Definition.Ready && r.Scope.Ready && r.Authority.Ready
}

func containsReadinessReason(values []ReadinessReason, target ReadinessReason) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func WorkItemReadiness(item WorkItem, outcome Outcome, objective *Objective, dependencies []DependencyEvaluation, now time.Time) Readiness {
	now = now.UTC()
	readiness := NewReadiness(item.Ref(), now)
	if item.Lifecycle != WorkItemLifecycleTodo {
		readiness.AddReason(ReadinessStageScope, ReadinessReasonLifecycleNotReady)
	}
	if outcome.Lifecycle != OutcomeLifecycleActive {
		readiness.AddReason(ReadinessStageScope, ReadinessReasonOutcomeNotActive)
	}
	if outcome.IsArchived() {
		readiness.AddReason(ReadinessStageScope, ReadinessReasonOutcomeArchived)
	}
	if objective != nil && (objective.Lifecycle == ObjectiveLifecycleAchieved || objective.Lifecycle == ObjectiveLifecycleCancelled) {
		readiness.AddReason(ReadinessStageScope, ReadinessReasonObjectiveTerminal, objective.Ref())
	}
	if hasUnsatisfiedHardDependency(dependencies) {
		readiness.AddReason(ReadinessStageScope, ReadinessReasonHardDependencyUnsatisfied)
	}
	if item.NotBefore != nil && item.NotBefore.After(now) {
		readiness.AddReason(ReadinessStageScope, ReadinessReasonNotBefore)
	}
	if item.CurrentLease != nil && item.CurrentLease.ValidAt(now) {
		readiness.AddReason(ReadinessStageAuthority, ReadinessReasonActiveLease)
	}
	return readiness
}

func ObjectiveReadiness(objective Objective, outcome Outcome, dependencies []DependencyEvaluation, now time.Time) Readiness {
	now = now.UTC()
	readiness := NewReadiness(objective.Ref(), now)
	if objective.Lifecycle != ObjectiveLifecyclePlanned {
		readiness.AddReason(ReadinessStageScope, ReadinessReasonLifecycleNotReady)
	}
	if outcome.Lifecycle != OutcomeLifecycleActive {
		readiness.AddReason(ReadinessStageScope, ReadinessReasonOutcomeNotActive)
	}
	if outcome.IsArchived() {
		readiness.AddReason(ReadinessStageScope, ReadinessReasonOutcomeArchived)
	}
	if hasUnsatisfiedHardDependency(dependencies) {
		readiness.AddReason(ReadinessStageScope, ReadinessReasonHardDependencyUnsatisfied)
	}
	return readiness
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

func ApplyBlockingToReadiness(value Readiness, blocked bool) Readiness {
	if !blocked {
		return value
	}
	value.AddReason(ReadinessStageScope, ReadinessReasonBlocked)
	return value
}
