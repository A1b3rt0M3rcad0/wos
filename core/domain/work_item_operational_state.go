package domain

import "time"

type LeaseStatus string

const (
	LeaseStatusNone    LeaseStatus = "none"
	LeaseStatusActive  LeaseStatus = "active"
	LeaseStatusExpired LeaseStatus = "expired"
)

type WorkItemDisplayState string

const (
	WorkItemDisplayStateBacklog             WorkItemDisplayState = "backlog"
	WorkItemDisplayStateWaitingScope        WorkItemDisplayState = "waiting_scope"
	WorkItemDisplayStateBlocked             WorkItemDisplayState = "blocked"
	WorkItemDisplayStateWaitingDependencies WorkItemDisplayState = "waiting_dependencies"
	WorkItemDisplayStateScheduled           WorkItemDisplayState = "scheduled"
	WorkItemDisplayStateReady               WorkItemDisplayState = "ready"
	WorkItemDisplayStateInProgress          WorkItemDisplayState = "in_progress"
	WorkItemDisplayStateAttentionNeeded     WorkItemDisplayState = "attention_needed"
	WorkItemDisplayStateDone                WorkItemDisplayState = "done"
	WorkItemDisplayStateCancelled           WorkItemDisplayState = "cancelled"
)

const ReadinessReasonLeaseExpired ReadinessReason = "lease_expired"

type WorkItemOperationalState struct {
	Ref              EntityRef            `json:"ref"`
	Lifecycle        WorkItemLifecycle    `json:"lifecycle"`
	DisplayState     WorkItemDisplayState `json:"display_state"`
	LeaseStatus      LeaseStatus          `json:"lease_status"`
	IsBlocked        bool                 `json:"is_blocked"`
	ReadinessReasons []ReadinessReason    `json:"readiness_reasons,omitempty"`
	EvaluatedAt      time.Time            `json:"evaluated_at"`
}

func (w WorkItem) LeaseStatusAt(now time.Time) LeaseStatus {
	if w.CurrentLease == nil {
		return LeaseStatusNone
	}
	if w.CurrentLease.ValidAt(now.UTC()) {
		return LeaseStatusActive
	}
	return LeaseStatusExpired
}

// ProjectWorkItemOperationalState derives presentation state from persisted
// lifecycle plus current coordination conditions. Time-dependent lease expiry
// changes this projection without mutating the WorkItem.
func ProjectWorkItemOperationalState(
	item WorkItem,
	outcome Outcome,
	objective *Objective,
	dependencies []DependencyEvaluation,
	blocked bool,
	now time.Time,
) WorkItemOperationalState {
	now = now.UTC()
	state := WorkItemOperationalState{
		Ref:         item.Ref(),
		Lifecycle:   item.Lifecycle,
		LeaseStatus: item.LeaseStatusAt(now),
		IsBlocked:   blocked,
		EvaluatedAt: now,
	}

	switch item.Lifecycle {
	case WorkItemLifecycleBacklog:
		state.DisplayState = WorkItemDisplayStateBacklog
		state.ReadinessReasons = []ReadinessReason{ReadinessReasonLifecycleNotReady}
		return state
	case WorkItemLifecycleDone:
		state.DisplayState = WorkItemDisplayStateDone
		state.ReadinessReasons = []ReadinessReason{ReadinessReasonLifecycleNotReady}
		return state
	case WorkItemLifecycleCancelled:
		state.DisplayState = WorkItemDisplayStateCancelled
		state.ReadinessReasons = []ReadinessReason{ReadinessReasonLifecycleNotReady}
		return state
	case WorkItemLifecycleTodo:
		readiness := WorkItemReadiness(item, outcome, objective, dependencies, now)
		readiness = ApplyBlockingToReadiness(readiness, blocked)
		state.ReadinessReasons = append([]ReadinessReason(nil), readiness.Reasons...)

		switch {
		case hasAnyReadinessReason(readiness.Reasons, ReadinessReasonOutcomeNotActive, ReadinessReasonOutcomeArchived, ReadinessReasonObjectiveTerminal):
			state.DisplayState = WorkItemDisplayStateWaitingScope
		case hasReadinessReason(readiness.Reasons, ReadinessReasonBlocked):
			state.DisplayState = WorkItemDisplayStateBlocked
		case hasReadinessReason(readiness.Reasons, ReadinessReasonHardDependencyUnsatisfied):
			state.DisplayState = WorkItemDisplayStateWaitingDependencies
		case hasReadinessReason(readiness.Reasons, ReadinessReasonNotBefore):
			state.DisplayState = WorkItemDisplayStateScheduled
		default:
			state.DisplayState = WorkItemDisplayStateReady
		}
		return state
	case WorkItemLifecycleInProgress:
		reasons := make([]ReadinessReason, 0, 6)
		if outcome.Lifecycle != OutcomeLifecycleActive {
			reasons = append(reasons, ReadinessReasonOutcomeNotActive)
		}
		if outcome.IsArchived() {
			reasons = append(reasons, ReadinessReasonOutcomeArchived)
		}
		if objective != nil && (objective.Lifecycle == ObjectiveLifecycleAchieved || objective.Lifecycle == ObjectiveLifecycleCancelled) {
			reasons = append(reasons, ReadinessReasonObjectiveTerminal)
		}
		if blocked {
			reasons = append(reasons, ReadinessReasonBlocked)
		}
		if hasUnsatisfiedHardDependency(dependencies) {
			reasons = append(reasons, ReadinessReasonHardDependencyUnsatisfied)
		}
		if state.LeaseStatus != LeaseStatusActive {
			reasons = append(reasons, ReadinessReasonLeaseExpired)
		}
		state.ReadinessReasons = reasons

		if blocked {
			state.DisplayState = WorkItemDisplayStateBlocked
		} else if len(reasons) > 0 {
			state.DisplayState = WorkItemDisplayStateAttentionNeeded
		} else {
			state.DisplayState = WorkItemDisplayStateInProgress
		}
		return state
	default:
		state.DisplayState = WorkItemDisplayStateAttentionNeeded
		state.ReadinessReasons = []ReadinessReason{ReadinessReasonLifecycleNotReady}
		return state
	}
}

func hasReadinessReason(reasons []ReadinessReason, target ReadinessReason) bool {
	for _, reason := range reasons {
		if reason == target {
			return true
		}
	}
	return false
}

func hasAnyReadinessReason(reasons []ReadinessReason, targets ...ReadinessReason) bool {
	for _, target := range targets {
		if hasReadinessReason(reasons, target) {
			return true
		}
	}
	return false
}
