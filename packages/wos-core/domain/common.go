package domain

import (
	"strings"
	"time"
)

type Priority string

const (
	PriorityCritical Priority = "critical"
	PriorityHigh     Priority = "high"
	PriorityNormal   Priority = "normal"
	PriorityLow      Priority = "low"
)

func (p Priority) Valid() bool {
	switch p {
	case PriorityCritical, PriorityHigh, PriorityNormal, PriorityLow:
		return true
	default:
		return false
	}
}

type Conclusion struct {
	PrincipalID string                   `json:"principal_id"`
	Actor       ActorRef                 `json:"actor_ref"`
	Reason      string                   `json:"reason"`
	ConcludedAt time.Time                `json:"concluded_at"`
	Assessments []CriterionAssessmentRef `json:"assessments,omitempty"`
}

func (c Conclusion) Validate() error {
	if strings.TrimSpace(c.PrincipalID) == "" {
		return NewError(ErrorCodeInvalidArgument, "conclusion principal_id is required")
	}
	if err := c.Actor.Validate(); err != nil {
		return WrapError(ErrorCodeInvalidArgument, "conclusion actor is invalid", err)
	}
	if strings.TrimSpace(c.Reason) == "" {
		return NewError(ErrorCodeInvalidArgument, "conclusion reason is required")
	}
	if c.ConcludedAt.IsZero() {
		return NewError(ErrorCodeInvalidArgument, "conclusion time is required")
	}
	return nil
}

func nextVersion(v Version) (Version, error) {
	return v.Next()
}
