package domain

import (
	"strings"
	"time"
)

type OutcomeLifecycle string

const (
	OutcomeLifecycleDraft     OutcomeLifecycle = "draft"
	OutcomeLifecycleActive    OutcomeLifecycle = "active"
	OutcomeLifecycleAchieved  OutcomeLifecycle = "achieved"
	OutcomeLifecycleFailed    OutcomeLifecycle = "failed"
	OutcomeLifecycleAbandoned OutcomeLifecycle = "abandoned"
)

type Outcome struct {
	ID                ID               `json:"id"`
	NamespaceID       ID               `json:"namespace_id"`
	Version           Version          `json:"version"`
	Title             string           `json:"title"`
	Description       string           `json:"description,omitempty"`
	DesiredState      string           `json:"desired_state"`
	Lifecycle         OutcomeLifecycle `json:"lifecycle"`
	Priority          Priority         `json:"priority"`
	OwnerRefs         []ActorRef       `json:"owner_refs,omitempty"`
	ArchivedAt        *time.Time       `json:"archived_at,omitempty"`
	Criteria          CriterionSet     `json:"criteria"`
	CurrentConclusion *Conclusion      `json:"conclusion,omitempty"`
	ConclusionHistory []Conclusion     `json:"conclusion_history,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	UpdatedAt         time.Time        `json:"updated_at"`
}

func NewOutcome(id, namespaceID ID, title, description, desiredState string, priority Priority, createdAt time.Time) (Outcome, error) {
	o := Outcome{
		ID:           id,
		NamespaceID:  namespaceID,
		Version:      InitialVersion,
		Title:        strings.TrimSpace(title),
		Description:  strings.TrimSpace(description),
		DesiredState: strings.TrimSpace(desiredState),
		Lifecycle:    OutcomeLifecycleDraft,
		Priority:     priority,
		Criteria:     NewCriterionSet(),
		CreatedAt:    createdAt.UTC(),
		UpdatedAt:    createdAt.UTC(),
	}
	if err := o.Validate(); err != nil {
		return Outcome{}, err
	}
	return o, nil
}

func (o Outcome) Scope() Scope {
	return Scope{NamespaceID: o.NamespaceID, OutcomeID: o.ID}
}

func (o Outcome) Ref() EntityRef {
	return EntityRef{Scope: o.Scope(), Kind: EntityKindOutcome, ID: o.ID}
}

func (o Outcome) IsArchived() bool { return o.ArchivedAt != nil }

func (o Outcome) Validate() error {
	if err := o.ID.Validate(); err != nil {
		return err
	}
	if err := o.NamespaceID.Validate(); err != nil {
		return err
	}
	if err := o.Version.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(o.Title) == "" {
		return NewError(ErrorCodeInvalidArgument, "outcome title is required")
	}
	if strings.TrimSpace(o.DesiredState) == "" {
		return NewError(ErrorCodeInvalidArgument, "outcome desired_state is required")
	}
	if !o.Priority.Valid() {
		return NewError(ErrorCodeInvalidArgument, "outcome priority is invalid")
	}
	if !o.Lifecycle.Valid() {
		return NewError(ErrorCodeInvalidArgument, "outcome lifecycle is invalid")
	}
	if err := validateActorRefs(o.OwnerRefs, "outcome owner"); err != nil {
		return err
	}
	if err := o.Criteria.ValidateForOwner(o.Ref()); err != nil {
		return err
	}
	if err := validateConclusionState(o.isTerminal(), o.CurrentConclusion, o.ConclusionHistory, "outcome"); err != nil {
		return err
	}
	if o.Lifecycle == OutcomeLifecycleAchieved && o.CurrentConclusion != nil && len(o.CurrentConclusion.Assessments) == 0 {
		return NewError(ErrorCodeInvalidArgument, "achieved outcome must reference the assessments that justified achievement")
	}
	if o.CreatedAt.IsZero() || o.UpdatedAt.IsZero() {
		return NewError(ErrorCodeInvalidArgument, "outcome timestamps are required")
	}
	return nil
}

func (o *Outcome) AddCriterion(c SuccessCriterion, now time.Time) error {
	if o.IsArchived() || o.isTerminal() {
		return NewError(ErrorCodeInvalidTransition, "cannot alter criteria on archived or terminal outcome")
	}
	if c.OwnerRef != o.Ref() {
		return NewError(ErrorCodeCriterion, "criterion owner does not match outcome")
	}
	if err := o.Criteria.Add(c); err != nil {
		return err
	}
	return o.touch(now)
}

func (o *Outcome) ReviseCriterion(id ID, title, description string, required bool, mode VerificationMode, now time.Time) error {
	if o.IsArchived() || o.isTerminal() {
		return NewError(ErrorCodeInvalidTransition, "cannot revise criteria on archived or terminal outcome")
	}
	if err := o.Criteria.Revise(id, title, description, required, mode); err != nil {
		return err
	}
	return o.touch(now)
}

func (o *Outcome) RetireCriterion(id ID, now time.Time) error {
	if o.IsArchived() || o.isTerminal() {
		return NewError(ErrorCodeInvalidTransition, "cannot retire criteria on archived or terminal outcome")
	}
	if err := o.Criteria.Retire(id); err != nil {
		return err
	}
	return o.touch(now)
}

func (o *Outcome) AssessCriterionAttestation(a CriterionAssessment, now time.Time) error {
	if o.IsArchived() {
		return NewError(ErrorCodeInvalidTransition, "cannot assess archived outcome")
	}
	if err := o.Criteria.AssessAttestation(a); err != nil {
		return err
	}
	return o.touch(now)
}

func (o *Outcome) Activate(now time.Time) error {
	if o.IsArchived() {
		return NewError(ErrorCodeInvalidTransition, "archived outcome cannot be activated")
	}
	if o.Lifecycle != OutcomeLifecycleDraft {
		return NewError(ErrorCodeInvalidTransition, "only draft outcome can be activated")
	}
	if !o.Criteria.HasRequiredActive() {
		return NewError(ErrorCodePreconditionFailed, "outcome requires at least one required active criterion")
	}
	o.Lifecycle = OutcomeLifecycleActive
	return o.touch(now)
}

func (o *Outcome) Achieve(conclusion Conclusion, now time.Time) error {
	if o.IsArchived() || o.Lifecycle != OutcomeLifecycleActive {
		return NewError(ErrorCodeInvalidTransition, "only active non-archived outcome can be achieved")
	}
	refs, err := o.Criteria.RequiredSatisfied()
	if err != nil {
		return err
	}
	conclusion.Assessments = refs
	if err := conclusion.Validate(); err != nil {
		return err
	}
	o.Lifecycle = OutcomeLifecycleAchieved
	o.CurrentConclusion = &conclusion
	return o.touch(now)
}

func (o *Outcome) Fail(conclusion Conclusion, now time.Time) error {
	if o.IsArchived() || o.Lifecycle != OutcomeLifecycleActive {
		return NewError(ErrorCodeInvalidTransition, "only active non-archived outcome can fail")
	}
	if err := conclusion.Validate(); err != nil {
		return err
	}
	o.Lifecycle = OutcomeLifecycleFailed
	o.CurrentConclusion = &conclusion
	return o.touch(now)
}

func (o *Outcome) Abandon(conclusion Conclusion, now time.Time) error {
	if o.IsArchived() || (o.Lifecycle != OutcomeLifecycleDraft && o.Lifecycle != OutcomeLifecycleActive) {
		return NewError(ErrorCodeInvalidTransition, "only draft or active non-archived outcome can be abandoned")
	}
	if err := conclusion.Validate(); err != nil {
		return err
	}
	o.Lifecycle = OutcomeLifecycleAbandoned
	o.CurrentConclusion = &conclusion
	return o.touch(now)
}

func (o *Outcome) Reopen(reason string, now time.Time) error {
	if o.IsArchived() {
		return NewError(ErrorCodeInvalidTransition, "archived outcome cannot be reopened")
	}
	if !o.isTerminal() {
		return NewError(ErrorCodeInvalidTransition, "only terminal outcome can be reopened")
	}
	if strings.TrimSpace(reason) == "" {
		return NewError(ErrorCodeInvalidArgument, "reopen reason is required")
	}
	if o.CurrentConclusion != nil {
		o.ConclusionHistory = append(o.ConclusionHistory, *o.CurrentConclusion)
		o.CurrentConclusion = nil
	}
	o.Lifecycle = OutcomeLifecycleActive
	return o.touch(now)
}

func (o *Outcome) Archive(now time.Time) error {
	if o.IsArchived() {
		return NewError(ErrorCodeInvalidTransition, "outcome is already archived")
	}
	t := now.UTC()
	o.ArchivedAt = &t
	return o.touch(now)
}

func (o *Outcome) Unarchive(reason string, now time.Time) error {
	if !o.IsArchived() {
		return NewError(ErrorCodeInvalidTransition, "outcome is not archived")
	}
	if strings.TrimSpace(reason) == "" {
		return NewError(ErrorCodeInvalidArgument, "unarchive reason is required")
	}
	o.ArchivedAt = nil
	return o.touch(now)
}

func (o Outcome) isTerminal() bool {
	return o.Lifecycle == OutcomeLifecycleAchieved || o.Lifecycle == OutcomeLifecycleFailed || o.Lifecycle == OutcomeLifecycleAbandoned
}

func (o *Outcome) touch(now time.Time) error {
	next, err := nextVersion(o.Version)
	if err != nil {
		return err
	}
	o.Version = next
	o.UpdatedAt = now.UTC()
	return nil
}
