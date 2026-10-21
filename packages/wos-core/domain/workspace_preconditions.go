package domain

func criterionPreparationError(c SuccessCriterion, reason, message string) error {
	ref := c.OwnerRef
	return NewDetailedError(ErrorCodePreconditionFailed, message, ErrorDetails{ReasonCode: reason, SubjectRef: &ref, FieldPaths: []string{"criteria." + c.ID.String()}, Remedies: []ErrorRemedy{{ActionID: "record_criterion_assessment", SubjectRef: &ref, CriterionID: &c.ID}}})
}

// These predicates are shared by mutations and workspace capability reads.
// They inspect current canonical facts; they never grant permission or mutate.
func (o Outcome) CheckActivation() error {
	ref := o.Ref()
	if o.IsArchived() {
		return NewDetailedError(ErrorCodeInvalidTransition, "archived outcome cannot be activated", ErrorDetails{ReasonCode: "outcome_archived", SubjectRef: &ref})
	}
	if o.Lifecycle != OutcomeLifecycleDraft {
		return NewDetailedError(ErrorCodeInvalidTransition, "only draft outcome can be activated", ErrorDetails{ReasonCode: "invalid_lifecycle", SubjectRef: &ref})
	}
	if !o.Criteria.HasRequiredActive() {
		return missingRequiredCriterion(ref)
	}
	return nil
}
func (o Outcome) CheckAchievement() error {
	ref := o.Ref()
	if o.IsArchived() || o.Lifecycle != OutcomeLifecycleActive {
		return NewDetailedError(ErrorCodeInvalidTransition, "only active non-archived outcome can be achieved", ErrorDetails{ReasonCode: "invalid_lifecycle", SubjectRef: &ref})
	}
	_, err := o.Criteria.RequiredSatisfied()
	return err
}
func (o Objective) CheckStart() error {
	ref := o.Ref()
	if o.Lifecycle != ObjectiveLifecyclePlanned {
		return NewDetailedError(ErrorCodeInvalidTransition, "only planned objective can start", ErrorDetails{ReasonCode: "invalid_lifecycle", SubjectRef: &ref})
	}
	return o.CheckStartDefinition()
}

// CheckStartDefinition evaluates authored planning completeness independently
// from the Objective's lifecycle, parent scope, dependencies, and blockers.
func (o Objective) CheckStartDefinition() error {
	ref := o.Ref()
	if !o.Criteria.HasRequiredActive() {
		return missingRequiredCriterion(ref)
	}
	return nil
}

// CheckActivationDefinition checks the Task definition needed before it may
// become executable. Compatibility-profile records may continue using their
// historical rules; operational_v2 callers opt into this stricter predicate.
func (w WorkItem) CheckActivationDefinition() error {
	ref := w.Ref()
	if w.ObjectiveID == nil {
		return NewDetailedError(ErrorCodePreconditionFailed, "Task requires an owning Objective before activation", ErrorDetails{
			ReasonCode: string(ReadinessReasonTaskObjectiveRequired), SubjectRef: &ref,
			FieldPaths: []string{"objective_id"}, Remedies: []ErrorRemedy{{ActionID: "select_objective"}},
		})
	}
	if !w.Criteria.HasRequiredActive() {
		return NewDetailedError(ErrorCodePreconditionFailed, "Task requires at least one active required criterion before activation", ErrorDetails{
			ReasonCode: string(ReadinessReasonTaskDefinitionIncomplete), SubjectRef: &ref,
			FieldPaths: []string{"criteria"}, Remedies: []ErrorRemedy{{ActionID: "add_criterion", SubjectRef: &ref}},
		})
	}
	return nil
}
func (o Objective) CheckAchievement() error {
	ref := o.Ref()
	if o.Lifecycle != ObjectiveLifecyclePlanned && o.Lifecycle != ObjectiveLifecycleInProgress {
		return NewDetailedError(ErrorCodeInvalidTransition, "objective cannot be achieved from current lifecycle", ErrorDetails{ReasonCode: "invalid_lifecycle", SubjectRef: &ref})
	}
	_, err := o.Criteria.RequiredSatisfied()
	return err
}
func missingRequiredCriterion(ref EntityRef) error {
	return NewDetailedError(ErrorCodePreconditionFailed, "at least one required active criterion is required", ErrorDetails{ReasonCode: "missing_required_criterion", SubjectRef: &ref, FieldPaths: []string{"criteria"}, Remedies: []ErrorRemedy{{ActionID: "add_criterion", SubjectRef: &ref}}})
}
