package domain

// Read projections and criterion definition mutations use the same owner gate.
// This is independent of evidence assessment and execution authorization.
func (o Outcome) CheckCriterionDefinitionEdit() error {
	if o.IsArchived() {
		ref := o.Ref()
		return NewDetailedError(ErrorCodeInvalidTransition, "cannot alter criteria on archived outcome", ErrorDetails{ReasonCode: "outcome_archived", SubjectRef: &ref})
	}
	if o.isTerminal() {
		return criterionOwnerTerminal(o.Ref())
	}
	return nil
}
func (o Objective) CheckCriterionDefinitionEdit() error {
	if o.isTerminal() {
		return criterionOwnerTerminal(o.Ref())
	}
	return nil
}
func (w WorkItem) CheckCriterionDefinitionEdit() error {
	if err := w.requireUncontracted(); err != nil {
		return err
	}
	if w.isTerminal() {
		return criterionOwnerTerminal(w.Ref())
	}
	return nil
}
func criterionOwnerTerminal(ref EntityRef) error {
	return NewDetailedError(ErrorCodeInvalidTransition, "reopen the terminal owner before changing criterion definitions", ErrorDetails{ReasonCode: "entity_terminal", SubjectRef: &ref})
}
