package domain

import (
	"strings"
	"time"
)

const (
	DefaultLeaseTTL = 5 * time.Minute
	MinLeaseTTL     = 30 * time.Second
	MaxLeaseTTL     = 60 * time.Minute
)

type WorkItemLifecycle string

const (
	WorkItemLifecycleBacklog    WorkItemLifecycle = "backlog"
	WorkItemLifecycleTodo       WorkItemLifecycle = "todo"
	WorkItemLifecycleInProgress WorkItemLifecycle = "in_progress"
	WorkItemLifecycleDone       WorkItemLifecycle = "done"
	WorkItemLifecycleCancelled  WorkItemLifecycle = "cancelled"
)

type WorkLease struct {
	ClaimID       ID        `json:"claim_id"`
	PrincipalID   string    `json:"principal_id"`
	Actor         ActorRef  `json:"actor_ref"`
	FencingToken  uint64    `json:"fencing_token"`
	AcquiredAt    time.Time `json:"acquired_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}

func (l WorkLease) ValidAt(now time.Time) bool {
	return !now.Before(l.AcquiredAt) && now.Before(l.ExpiresAt)
}

type WorkItem struct {
	ID                ID                `json:"id"`
	Scope             Scope             `json:"scope"`
	Version           Version           `json:"version"`
	Title             string            `json:"title"`
	Description       string            `json:"description,omitempty"`
	ObjectiveID       *ID               `json:"objective_id,omitempty"`
	Lifecycle         WorkItemLifecycle `json:"lifecycle"`
	Priority          Priority          `json:"priority"`
	ResultSummary     string            `json:"result_summary,omitempty"`
	CurrentLease      *WorkLease        `json:"current_lease,omitempty"`
	LastFencingToken  uint64            `json:"last_fencing_token"`
	Criteria          CriterionSet      `json:"criteria"`
	CurrentConclusion *Conclusion       `json:"conclusion,omitempty"`
	ConclusionHistory []Conclusion      `json:"conclusion_history,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

func NewWorkItem(id ID, scope Scope, title, description string, priority Priority, lifecycle WorkItemLifecycle, createdAt time.Time) (WorkItem, error) {
	if lifecycle == "" {
		lifecycle = WorkItemLifecycleBacklog
	}
	if lifecycle != WorkItemLifecycleBacklog && lifecycle != WorkItemLifecycleTodo {
		return WorkItem{}, NewError(ErrorCodeInvalidArgument, "new work item lifecycle must be backlog or todo")
	}
	w := WorkItem{
		ID:        id,
		Scope:     scope,
		Version:   InitialVersion,
		Title:     strings.TrimSpace(title),
		Description: strings.TrimSpace(description),
		Lifecycle: lifecycle,
		Priority:  priority,
		Criteria:  NewCriterionSet(),
		CreatedAt: createdAt.UTC(),
		UpdatedAt: createdAt.UTC(),
	}
	if err := w.Validate(); err != nil {
		return WorkItem{}, err
	}
	return w, nil
}

func (w WorkItem) Ref() EntityRef {
	return EntityRef{Scope: w.Scope, Kind: EntityKindWorkItem, ID: w.ID}
}

func (w WorkItem) Validate() error {
	if err := w.ID.Validate(); err != nil {
		return err
	}
	if err := w.Scope.Validate(); err != nil {
		return err
	}
	if err := w.Version.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(w.Title) == "" {
		return NewError(ErrorCodeInvalidArgument, "work item title is required")
	}
	if !w.Priority.Valid() {
		return NewError(ErrorCodeInvalidArgument, "work item priority is invalid")
	}
	return nil
}

func (w *WorkItem) AddCriterion(c SuccessCriterion, now time.Time) error {
	if w.isTerminal() {
		return NewError(ErrorCodeInvalidTransition, "cannot alter criteria on terminal work item")
	}
	if c.OwnerRef != w.Ref() {
		return NewError(ErrorCodeCriterion, "criterion owner does not match work item")
	}
	if err := w.Criteria.Add(c); err != nil {
		return err
	}
	return w.touch(now)
}

func (w *WorkItem) ReviseCriterion(id ID, title, description string, required bool, mode VerificationMode, now time.Time) error {
	if w.isTerminal() {
		return NewError(ErrorCodeInvalidTransition, "cannot revise criteria on terminal work item")
	}
	if err := w.Criteria.Revise(id, title, description, required, mode); err != nil {
		return err
	}
	return w.touch(now)
}

func (w *WorkItem) RetireCriterion(id ID, now time.Time) error {
	if w.isTerminal() {
		return NewError(ErrorCodeInvalidTransition, "cannot retire criteria on terminal work item")
	}
	if err := w.Criteria.Retire(id); err != nil {
		return err
	}
	return w.touch(now)
}

func (w *WorkItem) AssessCriterionAttestation(a CriterionAssessment, now time.Time) error {
	if err := w.Criteria.AssessAttestation(a); err != nil {
		return err
	}
	return w.touch(now)
}

func (w *WorkItem) Activate(now time.Time) error {
	if w.Lifecycle != WorkItemLifecycleBacklog {
		return NewError(ErrorCodeInvalidTransition, "only backlog work item can be activated")
	}
	w.Lifecycle = WorkItemLifecycleTodo
	return w.touch(now)
}

func (w *WorkItem) Defer(now time.Time) error {
	if w.Lifecycle != WorkItemLifecycleTodo {
		return NewError(ErrorCodeInvalidTransition, "only todo work item can be deferred")
	}
	w.Lifecycle = WorkItemLifecycleBacklog
	return w.touch(now)
}

func (w *WorkItem) Claim(claimID ID, principalID string, actor ActorRef, ttl time.Duration, now time.Time) error {
	if w.Lifecycle != WorkItemLifecycleTodo {
		return NewError(ErrorCodeInvalidTransition, "only todo work item can be claimed")
	}
	if err := claimID.Validate(); err != nil {
		return WrapError(ErrorCodeLease, "claim id is invalid", err)
	}
	if strings.TrimSpace(principalID) == "" {
		return NewError(ErrorCodeLease, "lease principal_id is required")
	}
	if err := actor.Validate(); err != nil {
		return WrapError(ErrorCodeLease, "lease actor is invalid", err)
	}
	if ttl == 0 {
		ttl = DefaultLeaseTTL
	}
	if ttl < MinLeaseTTL || ttl > MaxLeaseTTL {
		return NewError(ErrorCodeLease, "lease ttl is outside supported range")
	}
	w.LastFencingToken++
	w.CurrentLease = &WorkLease{
		ClaimID:      claimID,
		PrincipalID:  principalID,
		Actor:        actor,
		FencingToken: w.LastFencingToken,
		AcquiredAt:   now.UTC(),
		ExpiresAt:    now.UTC().Add(ttl),
	}
	w.Lifecycle = WorkItemLifecycleInProgress
	return w.touch(now)
}

func (w *WorkItem) Release(principalID string, claimID ID, fencingToken uint64, now time.Time) error {
	if w.Lifecycle != WorkItemLifecycleInProgress || w.CurrentLease == nil {
		return NewError(ErrorCodeInvalidTransition, "work item is not currently claimed")
	}
	if err := w.validateLeaseOwnership(principalID, claimID, fencingToken); err != nil {
		return err
	}
	w.CurrentLease = nil
	w.Lifecycle = WorkItemLifecycleTodo
	return w.touch(now)
}

func (w *WorkItem) Complete(principalID string, claimID ID, fencingToken uint64, resultSummary string, conclusion Conclusion, now time.Time) error {
	if w.Lifecycle != WorkItemLifecycleInProgress || w.CurrentLease == nil {
		return NewError(ErrorCodeInvalidTransition, "work item is not in progress")
	}
	if err := w.validateLeaseOwnership(principalID, claimID, fencingToken); err != nil {
		return err
	}
	if !w.CurrentLease.ValidAt(now.UTC()) {
		return NewError(ErrorCodeLease, "lease has expired")
	}

	var refs []CriterionAssessmentRef
	var err error
	if w.Criteria.HasRequiredActive() {
		refs, err = w.Criteria.RequiredSatisfied()
		if err != nil {
			return err
		}
	} else if strings.TrimSpace(resultSummary) == "" {
		return NewError(ErrorCodePreconditionFailed, "work item without required criteria needs result summary")
	}

	conclusion.Assessments = refs
	if err := conclusion.Validate(); err != nil {
		return err
	}
	w.ResultSummary = strings.TrimSpace(resultSummary)
	w.CurrentConclusion = &conclusion
	w.CurrentLease = nil
	w.Lifecycle = WorkItemLifecycleDone
	return w.touch(now)
}

func (w *WorkItem) Cancel(conclusion Conclusion, now time.Time) error {
	if w.Lifecycle != WorkItemLifecycleBacklog && w.Lifecycle != WorkItemLifecycleTodo && w.Lifecycle != WorkItemLifecycleInProgress {
		return NewError(ErrorCodeInvalidTransition, "work item cannot be cancelled from current lifecycle")
	}
	if err := conclusion.Validate(); err != nil {
		return err
	}
	w.CurrentConclusion = &conclusion
	w.CurrentLease = nil
	w.Lifecycle = WorkItemLifecycleCancelled
	return w.touch(now)
}

func (w *WorkItem) Reopen(reason string, now time.Time) error {
	if !w.isTerminal() {
		return NewError(ErrorCodeInvalidTransition, "only terminal work item can be reopened")
	}
	if strings.TrimSpace(reason) == "" {
		return NewError(ErrorCodeInvalidArgument, "reopen reason is required")
	}
	if w.CurrentConclusion != nil {
		w.ConclusionHistory = append(w.ConclusionHistory, *w.CurrentConclusion)
		w.CurrentConclusion = nil
	}
	w.ResultSummary = ""
	w.CurrentLease = nil
	w.Lifecycle = WorkItemLifecycleTodo
	return w.touch(now)
}

func (w WorkItem) isTerminal() bool {
	return w.Lifecycle == WorkItemLifecycleDone || w.Lifecycle == WorkItemLifecycleCancelled
}

func (w WorkItem) validateLeaseOwnership(principalID string, claimID ID, fencingToken uint64) error {
	if w.CurrentLease == nil {
		return NewError(ErrorCodeLease, "work item has no active lease")
	}
	if w.CurrentLease.PrincipalID != principalID || w.CurrentLease.ClaimID != claimID || w.CurrentLease.FencingToken != fencingToken {
		return NewError(ErrorCodeLease, "lease ownership or fencing token does not match")
	}
	return nil
}

func (w *WorkItem) touch(now time.Time) error {
	next, err := nextVersion(w.Version)
	if err != nil {
		return err
	}
	w.Version = next
	w.UpdatedAt = now.UTC()
	return nil
}
