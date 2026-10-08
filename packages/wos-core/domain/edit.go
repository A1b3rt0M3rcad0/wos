package domain

import (
	"strings"
	"time"
)

func (o *Outcome) UpdateDetails(
	title, description, desiredState *string,
	priority *Priority,
	now time.Time,
) (bool, error) {
	if o.IsArchived() || o.isTerminal() {
		return false, NewError(ErrorCodeInvalidTransition, "cannot edit archived or terminal outcome")
	}

	nextTitle := o.Title
	nextDescription := o.Description
	nextDesiredState := o.DesiredState
	nextPriority := o.Priority

	if title != nil {
		nextTitle = strings.TrimSpace(*title)
	}
	if description != nil {
		nextDescription = strings.TrimSpace(*description)
	}
	if desiredState != nil {
		nextDesiredState = strings.TrimSpace(*desiredState)
	}
	if priority != nil {
		nextPriority = *priority
	}

	if nextTitle == "" {
		return false, NewError(ErrorCodeInvalidArgument, "outcome title is required")
	}
	if nextDesiredState == "" {
		return false, NewError(ErrorCodeInvalidArgument, "outcome desired_state is required")
	}
	if !nextPriority.Valid() {
		return false, NewError(ErrorCodeInvalidArgument, "outcome priority is invalid")
	}

	if nextTitle == o.Title &&
		nextDescription == o.Description &&
		nextDesiredState == o.DesiredState &&
		nextPriority == o.Priority {
		return false, nil
	}

	o.Title = nextTitle
	o.Description = nextDescription
	o.DesiredState = nextDesiredState
	o.Priority = nextPriority
	return true, o.touch(now)
}

func (o *Objective) UpdateDetails(
	title, description *string,
	priority *Priority,
	requiredForOutcome *bool,
	now time.Time,
) (bool, error) {
	if o.isTerminal() {
		return false, NewError(ErrorCodeInvalidTransition, "cannot edit terminal objective")
	}

	nextTitle := o.Title
	nextDescription := o.Description
	nextPriority := o.Priority
	nextRequired := o.RequiredForOutcome

	if title != nil {
		nextTitle = strings.TrimSpace(*title)
	}
	if description != nil {
		nextDescription = strings.TrimSpace(*description)
	}
	if priority != nil {
		nextPriority = *priority
	}
	if requiredForOutcome != nil {
		nextRequired = *requiredForOutcome
	}

	if nextTitle == "" {
		return false, NewError(ErrorCodeInvalidArgument, "objective title is required")
	}
	if !nextPriority.Valid() {
		return false, NewError(ErrorCodeInvalidArgument, "objective priority is invalid")
	}

	if nextTitle == o.Title &&
		nextDescription == o.Description &&
		nextPriority == o.Priority &&
		nextRequired == o.RequiredForOutcome {
		return false, nil
	}

	o.Title = nextTitle
	o.Description = nextDescription
	o.Priority = nextPriority
	o.RequiredForOutcome = nextRequired
	return true, o.touch(now)
}

func (w *WorkItem) UpdateDetails(
	title, description *string,
	priority *Priority,
	now time.Time,
) (bool, error) {
	if err := w.requireUncontracted(); err != nil {
		return false, err
	}
	if w.isTerminal() {
		return false, NewError(ErrorCodeInvalidTransition, "cannot edit terminal work item")
	}

	nextTitle := w.Title
	nextDescription := w.Description
	nextPriority := w.Priority

	if title != nil {
		nextTitle = strings.TrimSpace(*title)
	}
	if description != nil {
		nextDescription = strings.TrimSpace(*description)
	}
	if priority != nil {
		nextPriority = *priority
	}

	if nextTitle == "" {
		return false, NewError(ErrorCodeInvalidArgument, "work item title is required")
	}
	if !nextPriority.Valid() {
		return false, NewError(ErrorCodeInvalidArgument, "work item priority is invalid")
	}

	if nextTitle == w.Title &&
		nextDescription == w.Description &&
		nextPriority == w.Priority {
		return false, nil
	}

	w.Title = nextTitle
	w.Description = nextDescription
	w.Priority = nextPriority
	return true, w.touch(now)
}
