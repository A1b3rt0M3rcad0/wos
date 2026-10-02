package domain

import (
	"strings"
	"time"
)

type ObjectiveLifecycle string

const (
	ObjectiveLifecyclePlanned    ObjectiveLifecycle = "planned"
	ObjectiveLifecycleInProgress ObjectiveLifecycle = "in_progress"
	ObjectiveLifecycleAchieved   ObjectiveLifecycle = "achieved"
	ObjectiveLifecycleCancelled  ObjectiveLifecycle = "cancelled"
)

type Objective struct {
	ID                 ID                 `json:"id"`
	Scope              Scope              `json:"scope"`
	Version            Version            `json:"version"`
	Title              string             `json:"title"`
	Description        string             `json:"description,omitempty"`
	ParentObjectiveID  *ID                `json:"parent_objective_id,omitempty"`
	Lifecycle          ObjectiveLifecycle `json:"lifecycle"`
	Priority           Priority           `json:"priority"`
	OwnerRefs          []ActorRef         `json:"owner_refs,omitempty"`
	RequiredForOutcome bool               `json:"required_for_outcome"`
	Criteria           CriterionSet       `json:"criteria"`
	CurrentConclusion  *Conclusion        `json:"conclusion,omitempty"`
	ConclusionHistory  []Conclusion       `json:"conclusion_history,omitempty"`
	CreatedAt          time.Time          `json:"created_at"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

func NewObjective(id ID, scope Scope, title, description string, priority Priority, requiredForOutcome bool, createdAt time.Time) (Objective, error) {
	o := Objective{
		ID:                 id,
		Scope:              scope,
		Version:            InitialVersion,
		Title:              strings.TrimSpace(title),
		Description:        strings.TrimSpace(description),
		Lifecycle:          ObjectiveLifecyclePlanned,
		Priority:           priority,
		RequiredForOutcome: requiredForOutcome,
		Criteria:           NewCriterionSet(),
		CreatedAt:          createdAt.UTC(),
		UpdatedAt:          createdAt.UTC(),
	}
	if err := o.Validate(); err != nil {
		return Objective{}, err
	}
	return o, nil
}

func (o Objective) Ref() EntityRef {
	return EntityRef{Scope: o.Scope, Kind: EntityKindObjective, ID: o.ID}
}

func (o Objective) Validate() error {
	if err := o.ID.Validate(); err != nil {
		return err
	}
	if err := o.Scope.Validate(); err != nil {
		return err
	}
	if err := o.Version.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(o.Title) == "" {
		return NewError(ErrorCodeInvalidArgument, "objective title is required")
	}
	if !o.Priority.Valid() {
		return NewError(ErrorCodeInvalidArgument, "objective priority is invalid")
	}
	if !o.Lifecycle.Valid() {
		return NewError(ErrorCodeInvalidArgument, "objective lifecycle is invalid")
	}
	if err := validateActorRefs(o.OwnerRefs, "objective owner"); err != nil {
		return err
	}
	if err := o.Criteria.ValidateForOwner(o.Ref()); err != nil {
		return err
	}
	if err := validateConclusionState(o.isTerminal(), o.CurrentConclusion, o.ConclusionHistory, "objective"); err != nil {
		return err
	}
	if o.Lifecycle == ObjectiveLifecycleAchieved && o.CurrentConclusion != nil && len(o.CurrentConclusion.Assessments) == 0 {
		return NewError(ErrorCodeInvalidArgument, "achieved objective must reference the assessments that justified achievement")
	}
	if o.CreatedAt.IsZero() || o.UpdatedAt.IsZero() {
		return NewError(ErrorCodeInvalidArgument, "objective timestamps are required")
	}
	return nil
}

func (o *Objective) AddCriterion(c SuccessCriterion, now time.Time) error {
	if o.isTerminal() {
		return NewError(ErrorCodeInvalidTransition, "cannot alter criteria on terminal objective")
	}
	if c.OwnerRef != o.Ref() {
		return NewError(ErrorCodeCriterion, "criterion owner does not match objective")
	}
	if err := o.Criteria.Add(c); err != nil {
		return err
	}
	return o.touch(now)
}

func (o *Objective) ReviseCriterion(id ID, title, description string, required bool, mode VerificationMode, now time.Time) error {
	if o.isTerminal() {
		return NewError(ErrorCodeInvalidTransition, "cannot revise criteria on terminal objective")
	}
	if err := o.Criteria.Revise(id, title, description, required, mode); err != nil {
		return err
	}
	return o.touch(now)
}

func (o *Objective) RetireCriterion(id ID, now time.Time) error {
	if o.isTerminal() {
		return NewError(ErrorCodeInvalidTransition, "cannot retire criteria on terminal objective")
	}
	if err := o.Criteria.Retire(id); err != nil {
		return err
	}
	return o.touch(now)
}

func (o *Objective) RecordCriterionAssessment(a CriterionAssessment, waiverAuthorized bool, now time.Time) error {
	if err := o.Criteria.RecordAssessment(a, waiverAuthorized); err != nil {
		return err
	}
	return o.touch(now)
}

func (o *Objective) AssessCriterionAttestation(a CriterionAssessment, now time.Time) error {
	if err := o.Criteria.AssessAttestation(a); err != nil {
		return err
	}
	return o.touch(now)
}

func (o *Objective) Start(now time.Time) error {
	if o.Lifecycle != ObjectiveLifecyclePlanned {
		return NewError(ErrorCodeInvalidTransition, "only planned objective can start")
	}
	if !o.Criteria.HasRequiredActive() {
		return NewError(ErrorCodePreconditionFailed, "objective requires at least one required active criterion")
	}
	o.Lifecycle = ObjectiveLifecycleInProgress
	return o.touch(now)
}

func (o *Objective) Achieve(conclusion Conclusion, now time.Time) error {
	if o.Lifecycle != ObjectiveLifecyclePlanned && o.Lifecycle != ObjectiveLifecycleInProgress {
		return NewError(ErrorCodeInvalidTransition, "objective cannot be achieved from current lifecycle")
	}
	refs, err := o.Criteria.RequiredSatisfied()
	if err != nil {
		return err
	}
	conclusion.Assessments = refs
	if err := conclusion.Validate(); err != nil {
		return err
	}
	o.Lifecycle = ObjectiveLifecycleAchieved
	o.CurrentConclusion = &conclusion
	return o.touch(now)
}

func (o *Objective) Cancel(conclusion Conclusion, now time.Time) error {
	if o.Lifecycle != ObjectiveLifecyclePlanned && o.Lifecycle != ObjectiveLifecycleInProgress {
		return NewError(ErrorCodeInvalidTransition, "objective cannot be cancelled from current lifecycle")
	}
	if err := conclusion.Validate(); err != nil {
		return err
	}
	o.Lifecycle = ObjectiveLifecycleCancelled
	o.CurrentConclusion = &conclusion
	return o.touch(now)
}

func (o *Objective) Reopen(reason string, now time.Time) error {
	if !o.isTerminal() {
		return NewError(ErrorCodeInvalidTransition, "only terminal objective can be reopened")
	}
	if strings.TrimSpace(reason) == "" {
		return NewError(ErrorCodeInvalidArgument, "reopen reason is required")
	}
	if o.CurrentConclusion != nil {
		o.ConclusionHistory = append(o.ConclusionHistory, *o.CurrentConclusion)
		o.CurrentConclusion = nil
	}
	o.Lifecycle = ObjectiveLifecyclePlanned
	return o.touch(now)
}

func (o Objective) isTerminal() bool {
	return o.Lifecycle == ObjectiveLifecycleAchieved || o.Lifecycle == ObjectiveLifecycleCancelled
}

func (o *Objective) touch(now time.Time) error {
	next, err := nextVersion(o.Version)
	if err != nil {
		return err
	}
	o.Version = next
	o.UpdatedAt = now.UTC()
	return nil
}
