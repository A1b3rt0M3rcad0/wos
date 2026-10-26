package domain

import "strings"

func (l OutcomeLifecycle) Valid() bool {
	switch l {
	case OutcomeLifecycleDraft, OutcomeLifecycleActive, OutcomeLifecycleAchieved, OutcomeLifecycleFailed, OutcomeLifecycleAbandoned:
		return true
	default:
		return false
	}
}

func (l ObjectiveLifecycle) Valid() bool {
	switch l {
	case ObjectiveLifecyclePlanned, ObjectiveLifecycleInProgress, ObjectiveLifecycleAchieved, ObjectiveLifecycleCancelled:
		return true
	default:
		return false
	}
}

func (l WorkItemLifecycle) Valid() bool {
	switch l {
	case WorkItemLifecycleBacklog, WorkItemLifecycleTodo, WorkItemLifecycleInProgress, WorkItemLifecycleDone, WorkItemLifecycleCancelled:
		return true
	default:
		return false
	}
}

func (l WorkLease) Validate() error {
	if err := l.ClaimID.Validate(); err != nil {
		return WrapError(ErrorCodeLease, "lease claim id is invalid", err)
	}
	if strings.TrimSpace(l.PrincipalID) == "" {
		return NewError(ErrorCodeLease, "lease principal_id is required")
	}
	if err := l.Actor.Validate(); err != nil {
		return WrapError(ErrorCodeLease, "lease actor is invalid", err)
	}
	if l.FencingToken == 0 {
		return NewError(ErrorCodeLease, "lease fencing token must be positive")
	}
	if l.AcquiredAt.IsZero() || l.ExpiresAt.IsZero() {
		return NewError(ErrorCodeLease, "lease timestamps are required")
	}
	if !l.ExpiresAt.After(l.AcquiredAt) {
		return NewError(ErrorCodeLease, "lease expiration must be after acquisition")
	}
	if err := l.validateCredentialBinding(); err != nil {
		return err
	}
	return nil
}

func (l WorkLease) validateCredentialBinding() error {
	switch l.CredentialBindingVersion {
	case 0:
		if !l.HolderCredentialID.IsZero() || l.SessionCredentialID != nil {
			return NewError(ErrorCodeLease, "unversioned lease cannot contain credential binding")
		}
	case WorkLeaseCredentialBindingV1:
		if err := l.HolderCredentialID.Validate(); err != nil {
			return WrapError(ErrorCodeLease, "lease holder credential id is invalid", err)
		}
		if l.SessionCredentialID != nil {
			if err := l.SessionCredentialID.Validate(); err != nil {
				return WrapError(ErrorCodeLease, "lease session credential id is invalid", err)
			}
		}
	default:
		return NewError(ErrorCodeLease, "lease credential binding version is unsupported")
	}
	return nil
}

func validateConclusionState(terminal bool, current *Conclusion, history []Conclusion, label string) error {
	for _, conclusion := range history {
		if err := conclusion.Validate(); err != nil {
			return WrapError(ErrorCodeInvalidArgument, label+" conclusion history is invalid", err)
		}
	}
	if terminal {
		if current == nil {
			return NewError(ErrorCodeInvalidArgument, label+" terminal lifecycle requires current conclusion")
		}
		if err := current.Validate(); err != nil {
			return WrapError(ErrorCodeInvalidArgument, label+" current conclusion is invalid", err)
		}
		return nil
	}
	if current != nil {
		return NewError(ErrorCodeInvalidArgument, label+" non-terminal lifecycle cannot retain current conclusion")
	}
	return nil
}
