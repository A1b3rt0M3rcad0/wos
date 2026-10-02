package domain

import "time"

type RoadmapRevisionPointer struct {
	RoadmapID      ID     `json:"roadmap_id"`
	RevisionNumber uint64 `json:"revision_number"`
}

func (p RoadmapRevisionPointer) Validate() error {
	if err := p.RoadmapID.Validate(); err != nil {
		return WrapError(ErrorCodeRoadmap, "roadmap revision pointer id is invalid", err)
	}
	if p.RevisionNumber == 0 {
		return NewError(ErrorCodeRoadmap, "roadmap revision pointer number must be at least 1")
	}
	return nil
}

type RoadmapActiveSlot struct {
	Scope          Scope                  `json:"scope"`
	PlanScope      RoadmapPlanScope       `json:"plan_scope"`
	RoadmapID      ID                     `json:"roadmap_id"`
	RevisionNumber uint64                 `json:"revision_number"`
	ActivatedBy    ActorRef               `json:"activated_by"`
	ActivatedAt    time.Time              `json:"activated_at"`
}

func (s RoadmapActiveSlot) Validate() error {
	if err := s.Scope.Validate(); err != nil {
		return err
	}
	if err := s.PlanScope.Validate(s.Scope); err != nil {
		return err
	}
	if err := s.RoadmapID.Validate(); err != nil {
		return WrapError(ErrorCodeRoadmap, "active roadmap id is invalid", err)
	}
	if s.RevisionNumber == 0 {
		return NewError(ErrorCodeRoadmap, "active roadmap revision must be at least 1")
	}
	if err := s.ActivatedBy.Validate(); err != nil {
		return WrapError(ErrorCodeRoadmap, "roadmap activation actor is invalid", err)
	}
	if s.ActivatedAt.IsZero() {
		return NewError(ErrorCodeRoadmap, "roadmap activated_at is required")
	}
	return nil
}

func (s RoadmapActiveSlot) Pointer() RoadmapRevisionPointer {
	return RoadmapRevisionPointer{RoadmapID: s.RoadmapID, RevisionNumber: s.RevisionNumber}
}

type RoadmapActivationAction string

const (
	RoadmapActivationActivated   RoadmapActivationAction = "activated"
	RoadmapActivationSuperseded  RoadmapActivationAction = "superseded"
	RoadmapActivationDeactivated RoadmapActivationAction = "deactivated"
)

func (a RoadmapActivationAction) Valid() bool {
	switch a {
	case RoadmapActivationActivated, RoadmapActivationSuperseded, RoadmapActivationDeactivated:
		return true
	default:
		return false
	}
}

type RoadmapActivationRecord struct {
	ID             ID                      `json:"id"`
	Scope          Scope                   `json:"scope"`
	PlanScope      RoadmapPlanScope        `json:"plan_scope"`
	Action         RoadmapActivationAction `json:"action"`
	RoadmapID      ID                      `json:"roadmap_id"`
	RevisionNumber uint64                  `json:"revision_number"`
	Replacement    *RoadmapRevisionPointer `json:"replacement,omitempty"`
	Actor          ActorRef                `json:"actor_ref"`
	RecordedAt     time.Time               `json:"recorded_at"`
}

func (r RoadmapActivationRecord) Validate() error {
	if err := r.ID.Validate(); err != nil {
		return WrapError(ErrorCodeRoadmap, "roadmap activation id is invalid", err)
	}
	if err := r.Scope.Validate(); err != nil {
		return err
	}
	if err := r.PlanScope.Validate(r.Scope); err != nil {
		return err
	}
	if !r.Action.Valid() {
		return NewError(ErrorCodeRoadmap, "roadmap activation action is invalid")
	}
	if err := (RoadmapRevisionPointer{RoadmapID: r.RoadmapID, RevisionNumber: r.RevisionNumber}).Validate(); err != nil {
		return err
	}
	if err := r.Actor.Validate(); err != nil {
		return WrapError(ErrorCodeRoadmap, "roadmap activation actor is invalid", err)
	}
	if r.RecordedAt.IsZero() {
		return NewError(ErrorCodeRoadmap, "roadmap activation recorded_at is required")
	}
	if r.Action == RoadmapActivationSuperseded {
		if r.Replacement == nil {
			return NewError(ErrorCodeRoadmap, "superseded roadmap activation requires replacement")
		}
		if err := r.Replacement.Validate(); err != nil {
			return err
		}
		if r.Replacement.RoadmapID == r.RoadmapID && r.Replacement.RevisionNumber == r.RevisionNumber {
			return NewError(ErrorCodeRoadmap, "roadmap activation cannot supersede itself")
		}
	} else if r.Replacement != nil {
		return NewError(ErrorCodeRoadmap, "only superseded roadmap activation may contain replacement")
	}
	return nil
}

func (r RoadmapActivationRecord) RoadmapRef() EntityRef {
	return EntityRef{Scope: r.Scope, Kind: EntityKindRoadmap, ID: r.RoadmapID}
}
