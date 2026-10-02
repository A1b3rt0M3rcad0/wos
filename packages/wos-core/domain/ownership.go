package domain

import (
	"fmt"
	"time"
)

// OwnerRefs and AssigneeRefs declare responsibility or allocation intent.
// They are descriptive domain state and never grant authorization. Execution
// ownership remains represented by the WorkItem lease Principal.
func (o *Outcome) SetOwners(refs []ActorRef, now time.Time) error {
	if o.IsArchived() || o.isTerminal() {
		return NewError(ErrorCodeInvalidTransition, "cannot change owners on archived or terminal outcome")
	}
	if err := validateActorRefs(refs, "outcome owner"); err != nil {
		return err
	}
	o.OwnerRefs = cloneActorRefs(refs)
	return o.touch(now)
}

func (o *Objective) SetOwners(refs []ActorRef, now time.Time) error {
	if o.isTerminal() {
		return NewError(ErrorCodeInvalidTransition, "cannot change owners on terminal objective")
	}
	if err := validateActorRefs(refs, "objective owner"); err != nil {
		return err
	}
	o.OwnerRefs = cloneActorRefs(refs)
	return o.touch(now)
}

func (w *WorkItem) SetAssignees(refs []ActorRef, now time.Time) error {
	if w.isTerminal() {
		return NewError(ErrorCodeInvalidTransition, "cannot change assignees on terminal work item")
	}
	if err := validateActorRefs(refs, "work item assignee"); err != nil {
		return err
	}
	w.AssigneeRefs = cloneActorRefs(refs)
	return w.touch(now)
}

func validateActorRefs(refs []ActorRef, label string) error {
	seen := make(map[string]struct{}, len(refs))
	for _, ref := range refs {
		if err := ref.Validate(); err != nil {
			return WrapError(ErrorCodeInvalidActorRef, fmt.Sprintf("%s is invalid", label), err)
		}
		key := string(ref.Kind) + "\x00" + ref.Provider + "\x00" + ref.ID
		if _, exists := seen[key]; exists {
			return NewError(ErrorCodeInvalidArgument, fmt.Sprintf("duplicate %s", label))
		}
		seen[key] = struct{}{}
	}
	return nil
}

func cloneActorRefs(refs []ActorRef) []ActorRef {
	if refs == nil {
		return nil
	}
	return append([]ActorRef(nil), refs...)
}
