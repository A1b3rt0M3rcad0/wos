package domain

import "math"

// ObjectiveCompletionMode controls whether Objective achievement is always an
// explicit operation or may be derived from a separately authorized rule.
// An empty value preserves the historical manual behavior.
type ObjectiveCompletionMode string

const (
	ObjectiveCompletionModeManual    ObjectiveCompletionMode = "manual"
	ObjectiveCompletionModeAutomatic ObjectiveCompletionMode = "automatic"
)

func (m ObjectiveCompletionMode) Effective() ObjectiveCompletionMode {
	if m == "" {
		return ObjectiveCompletionModeManual
	}
	return m
}

func (m ObjectiveCompletionMode) Valid() bool {
	switch m.Effective() {
	case ObjectiveCompletionModeManual, ObjectiveCompletionModeAutomatic:
		return true
	default:
		return false
	}
}

// EffectiveCompletionCycle returns the first cycle for legacy Objectives that
// predate explicit cycle persistence.
func (o Objective) EffectiveCompletionCycle() uint64 {
	if o.CompletionCycle == 0 {
		return 1
	}
	return o.CompletionCycle
}

// ValidateCompletionConfiguration checks the policy combination that cannot
// be validated by either aggregate in isolation.
func (o Objective) ValidateCompletionConfiguration(outcome Outcome) error {
	if outcome.Scope() != o.Scope {
		return NewError(ErrorCodeInvalidArgument, "objective completion policy must use its owning outcome")
	}
	if !outcome.OperationalProfile.Valid() || !outcome.CompletionPolicy.Valid() || !o.CompletionMode.Valid() {
		return NewError(ErrorCodeInvalidArgument, "invalid operational completion policy")
	}
	if o.CompletionMode.Effective() == ObjectiveCompletionModeAutomatic &&
		(outcome.OperationalProfile.Effective() != OperationalProfileOperationalV2 ||
			outcome.CompletionPolicy.Effective() != CompletionPolicyStrictWork) {
		return NewError(ErrorCodeInvalidArgument, "automatic objective completion requires operational_v2 and strict_work")
	}
	return nil
}

func nextObjectiveCompletionCycle(cycle uint64) (uint64, error) {
	if cycle == math.MaxUint64 {
		return 0, NewError(ErrorCodeInvalidVersion, "objective completion cycle overflow")
	}
	return cycle + 1, nil
}
