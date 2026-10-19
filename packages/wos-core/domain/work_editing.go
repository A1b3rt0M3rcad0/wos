package domain

import (
	"slices"
	"time"
)

// Definition editing never grants execution authority or discards a contract.
// Application predicates additionally validate the current scoped repositories.
func (w WorkItem) ValidateDefinitionEdit(now time.Time) error {
	if w.isTerminal() {
		return NewError(ErrorCodeInvalidTransition, "terminal work must be reopened before editing its definition")
	}
	if w.PendingReviewCaseID != nil {
		return NewError(ErrorCodeSignedProtocolRequired, "open review freezes the work definition")
	}
	if err := w.requireUncontracted(); err != nil {
		return err
	}
	if w.CurrentLease != nil && w.CurrentLease.ValidAt(now) {
		return NewError(ErrorCodeWorkAlreadyClaimed, "release the current lease before editing the work definition")
	}
	return nil
}

func (w *WorkItem) SetExecutionSpec(spec *ExecutionSpec, now time.Time) (bool, error) {
	if err := w.ValidateDefinitionEdit(now); err != nil {
		return false, err
	}
	if spec != nil {
		if err := spec.Validate(); err != nil {
			return false, err
		}
	}
	if executionSpecsEqual(w.ExecutionSpec, spec) {
		return false, nil
	}
	w.ExecutionSpec = cloneExecutionSpec(spec)
	return true, w.touch(now)
}

func (w *WorkItem) SetNotBefore(value *time.Time, now time.Time) (bool, error) {
	if err := w.ValidateDefinitionEdit(now); err != nil {
		return false, err
	}
	if value != nil && value.IsZero() {
		return false, NewError(ErrorCodeInvalidArgument, "not_before must be a nonzero timestamp or null")
	}
	if w.NotBefore == nil && value == nil || w.NotBefore != nil && value != nil && w.NotBefore.Equal(*value) {
		return false, nil
	}
	w.NotBefore = nil
	if value != nil {
		instant := value.UTC()
		w.NotBefore = &instant
	}
	return true, w.touch(now)
}

func (w *WorkItem) SetObjective(value *ID, now time.Time) (bool, error) {
	if err := w.ValidateDefinitionEdit(now); err != nil {
		return false, err
	}
	if value != nil {
		if err := value.Validate(); err != nil {
			return false, err
		}
	}
	if optionalIDsEqual(w.ObjectiveID, value) {
		return false, nil
	}
	w.ObjectiveID = nil
	if value != nil {
		id := *value
		w.ObjectiveID = &id
	}
	return true, w.touch(now)
}

func (o *Objective) SetParent(value *ID, now time.Time) (bool, error) {
	if o.Lifecycle != ObjectiveLifecyclePlanned {
		return false, NewError(ErrorCodePreconditionFailed, "only a planned Objective can be reorganized")
	}
	if value != nil {
		if err := value.Validate(); err != nil {
			return false, err
		}
		if *value == o.ID {
			return false, NewError(ErrorCodeInvalidRelation, "an Objective cannot be its own parent")
		}
	}
	if optionalIDsEqual(o.ParentObjectiveID, value) {
		return false, nil
	}
	o.ParentObjectiveID = nil
	if value != nil {
		id := *value
		o.ParentObjectiveID = &id
	}
	return true, o.touch(now)
}

// IsOwnObjectiveHardDependency identifies an impossible readiness configuration:
// an incomplete parent fails the dependency and a completed parent closes scope.
// Existing records are diagnosed by readers rather than silently removed.
func IsOwnObjectiveHardDependency(item WorkItem, relation Relation) bool {
	return item.ObjectiveID != nil && relation.Scope == item.Scope &&
		relation.SourceRef == item.Ref() && relation.TargetRef.Scope == item.Scope &&
		relation.TargetRef.Kind == EntityKindObjective && relation.TargetRef.ID == *item.ObjectiveID &&
		relation.RelationType == RelationTypeDependsOn && relation.Strength == DependencyStrengthHard &&
		relation.Lifecycle == RelationLifecycleActive
}

func optionalIDsEqual(a, b *ID) bool {
	return a == nil && b == nil || a != nil && b != nil && *a == *b
}

func executionSpecsEqual(a, b *ExecutionSpec) bool {
	if a == nil || b == nil {
		return a == b
	}
	return slices.Equal(a.Instructions, b.Instructions) && slices.Equal(a.Constraints, b.Constraints) &&
		slices.Equal(a.Deliverables, b.Deliverables) && slices.Equal(a.ScopeHints, b.ScopeHints) && slices.Equal(a.ContextRefs, b.ContextRefs)
}

func cloneExecutionSpec(spec *ExecutionSpec) *ExecutionSpec {
	if spec == nil {
		return nil
	}
	result := *spec
	result.Instructions = slices.Clone(spec.Instructions)
	result.Constraints = slices.Clone(spec.Constraints)
	result.Deliverables = slices.Clone(spec.Deliverables)
	result.ScopeHints = slices.Clone(spec.ScopeHints)
	result.ContextRefs = slices.Clone(spec.ContextRefs)
	return &result
}
