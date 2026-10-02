package domain

import (
	"strings"
	"time"
)

type DecisionLifecycle string

const (
	DecisionLifecycleProposed   DecisionLifecycle = "proposed"
	DecisionLifecycleAccepted   DecisionLifecycle = "accepted"
	DecisionLifecycleRejected   DecisionLifecycle = "rejected"
	DecisionLifecycleSuperseded DecisionLifecycle = "superseded"
)

func (l DecisionLifecycle) Valid() bool {
	switch l {
	case DecisionLifecycleProposed, DecisionLifecycleAccepted, DecisionLifecycleRejected, DecisionLifecycleSuperseded:
		return true
	default:
		return false
	}
}

type Decision struct {
	ID                   ID                `json:"id"`
	Scope                Scope             `json:"scope"`
	Version              Version           `json:"version"`
	Title                string            `json:"title"`
	Proposal             string            `json:"proposal"`
	ChosenAlternative    string            `json:"chosen_alternative,omitempty"`
	Rationale            string            `json:"rationale,omitempty"`
	Alternatives         []string          `json:"alternatives"`
	Lifecycle            DecisionLifecycle `json:"lifecycle"`
	DecidedBy            *ActorRef         `json:"decided_by,omitempty"`
	DecidedAt            *time.Time        `json:"decided_at,omitempty"`
	SupersedesDecisionID *ID               `json:"supersedes_decision_id,omitempty"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
}

func NewDecision(
	id ID,
	scope Scope,
	title, proposal, rationale string,
	alternatives []string,
	supersedesDecisionID *ID,
	now time.Time,
) (Decision, error) {
	normalized, err := normalizedUniqueStrings(alternatives, "decision alternatives")
	if err != nil {
		return Decision{}, err
	}
	value := Decision{
		ID:                   id,
		Scope:                scope,
		Version:              InitialVersion,
		Title:                strings.TrimSpace(title),
		Proposal:             strings.TrimSpace(proposal),
		Rationale:            strings.TrimSpace(rationale),
		Alternatives:         normalized,
		Lifecycle:            DecisionLifecycleProposed,
		SupersedesDecisionID: cloneIDPointerValue(supersedesDecisionID),
		CreatedAt:            now.UTC(),
		UpdatedAt:            now.UTC(),
	}
	if err := value.Validate(); err != nil {
		return Decision{}, err
	}
	return value, nil
}

func (d Decision) Ref() EntityRef {
	return EntityRef{Scope: d.Scope, Kind: EntityKindDecision, ID: d.ID}
}

func (d Decision) Validate() error {
	if err := d.ID.Validate(); err != nil {
		return err
	}
	if err := d.Scope.Validate(); err != nil {
		return err
	}
	if err := d.Version.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(d.Title) == "" {
		return NewError(ErrorCodeDecision, "decision title is required")
	}
	if strings.TrimSpace(d.Proposal) == "" {
		return NewError(ErrorCodeDecision, "decision proposal is required")
	}
	normalized, err := normalizedUniqueStrings(d.Alternatives, "decision alternatives")
	if err != nil {
		return err
	}
	if len(normalized) == 0 {
		return NewError(ErrorCodeDecision, "decision requires at least one alternative")
	}
	if !d.Lifecycle.Valid() {
		return NewError(ErrorCodeDecision, "decision lifecycle is invalid")
	}
	if d.SupersedesDecisionID != nil {
		if err := d.SupersedesDecisionID.Validate(); err != nil {
			return WrapError(ErrorCodeDecision, "supersedes_decision_id is invalid", err)
		}
		if *d.SupersedesDecisionID == d.ID {
			return NewError(ErrorCodeDecision, "decision cannot supersede itself")
		}
	}
	switch d.Lifecycle {
	case DecisionLifecycleProposed:
		if d.DecidedBy != nil || d.DecidedAt != nil || strings.TrimSpace(d.ChosenAlternative) != "" {
			return NewError(ErrorCodeDecision, "proposed decision cannot contain final decision fields")
		}
	case DecisionLifecycleAccepted:
		if !containsString(normalized, strings.TrimSpace(d.ChosenAlternative)) {
			return NewError(ErrorCodeDecision, "accepted decision chosen_alternative must be one of alternatives")
		}
		if strings.TrimSpace(d.Rationale) == "" {
			return NewError(ErrorCodeDecision, "accepted decision requires rationale")
		}
		if err := validateDecisionActorAndTime(d.DecidedBy, d.DecidedAt); err != nil {
			return err
		}
	case DecisionLifecycleRejected:
		if strings.TrimSpace(d.ChosenAlternative) != "" {
			return NewError(ErrorCodeDecision, "rejected decision cannot have chosen_alternative")
		}
		if strings.TrimSpace(d.Rationale) == "" {
			return NewError(ErrorCodeDecision, "rejected decision requires rationale")
		}
		if err := validateDecisionActorAndTime(d.DecidedBy, d.DecidedAt); err != nil {
			return err
		}
	case DecisionLifecycleSuperseded:
		if strings.TrimSpace(d.ChosenAlternative) == "" || strings.TrimSpace(d.Rationale) == "" {
			return NewError(ErrorCodeDecision, "superseded decision must preserve accepted content")
		}
		if err := validateDecisionActorAndTime(d.DecidedBy, d.DecidedAt); err != nil {
			return err
		}
	}
	if d.CreatedAt.IsZero() || d.UpdatedAt.IsZero() {
		return NewError(ErrorCodeDecision, "decision timestamps are required")
	}
	return nil
}

func (d *Decision) ReviseProposal(title, proposal, rationale string, alternatives []string, now time.Time) error {
	if d.Lifecycle != DecisionLifecycleProposed {
		return NewError(ErrorCodeInvalidTransition, "only proposed decision can be revised")
	}
	normalized, err := normalizedUniqueStrings(alternatives, "decision alternatives")
	if err != nil {
		return err
	}
	d.Title = strings.TrimSpace(title)
	d.Proposal = strings.TrimSpace(proposal)
	d.Rationale = strings.TrimSpace(rationale)
	d.Alternatives = normalized
	return d.touch(now)
}

func (d *Decision) Accept(chosenAlternative, rationale string, actor ActorRef, now time.Time) error {
	if d.Lifecycle != DecisionLifecycleProposed {
		return NewError(ErrorCodeInvalidTransition, "only proposed decision can be accepted")
	}
	chosenAlternative = strings.TrimSpace(chosenAlternative)
	if !containsString(d.Alternatives, chosenAlternative) {
		return NewError(ErrorCodeDecision, "chosen_alternative must be one of decision alternatives")
	}
	if err := actor.Validate(); err != nil {
		return WrapError(ErrorCodeDecision, "decided_by is invalid", err)
	}
	rationale = strings.TrimSpace(rationale)
	if rationale == "" {
		return NewError(ErrorCodeDecision, "decision acceptance rationale is required")
	}
	decidedAt := now.UTC()
	d.ChosenAlternative = chosenAlternative
	d.Rationale = rationale
	d.Lifecycle = DecisionLifecycleAccepted
	d.DecidedBy = cloneActorRefPointer(&actor)
	d.DecidedAt = &decidedAt
	return d.touch(now)
}

func (d *Decision) Reject(reason string, actor ActorRef, now time.Time) error {
	if d.Lifecycle != DecisionLifecycleProposed {
		return NewError(ErrorCodeInvalidTransition, "only proposed decision can be rejected")
	}
	if err := actor.Validate(); err != nil {
		return WrapError(ErrorCodeDecision, "decided_by is invalid", err)
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return NewError(ErrorCodeDecision, "decision rejection reason is required")
	}
	decidedAt := now.UTC()
	d.ChosenAlternative = ""
	d.Rationale = reason
	d.Lifecycle = DecisionLifecycleRejected
	d.DecidedBy = cloneActorRefPointer(&actor)
	d.DecidedAt = &decidedAt
	return d.touch(now)
}

func (d *Decision) MarkSuperseded(now time.Time) error {
	if d.Lifecycle != DecisionLifecycleAccepted {
		return NewError(ErrorCodeInvalidTransition, "only accepted decision can be superseded")
	}
	d.Lifecycle = DecisionLifecycleSuperseded
	return d.touch(now)
}

func (d *Decision) touch(now time.Time) error {
	next, err := nextVersion(d.Version)
	if err != nil {
		return err
	}
	d.Version = next
	d.UpdatedAt = now.UTC()
	return d.Validate()
}

func validateDecisionActorAndTime(actor *ActorRef, at *time.Time) error {
	if actor == nil || at == nil || at.IsZero() {
		return NewError(ErrorCodeDecision, "final decision requires decided_by and decided_at")
	}
	if err := actor.Validate(); err != nil {
		return WrapError(ErrorCodeDecision, "decided_by is invalid", err)
	}
	return nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func cloneActorRefPointer(value *ActorRef) *ActorRef {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
