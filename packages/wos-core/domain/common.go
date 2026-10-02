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

type CriterionObligationSnapshot struct {
	CriterionID      ID                `json:"criterion_id"`
	CriterionRevision CriterionRevision `json:"criterion_revision"`
	VerificationMode VerificationMode  `json:"verification_mode"`
}

func (o CriterionObligationSnapshot) Validate() error {
	if err := o.CriterionID.Validate(); err != nil {
		return WrapError(ErrorCodeInvalidArgument, "criterion obligation id is invalid", err)
	}
	if o.CriterionRevision < InitialCriterionRevision {
		return NewError(ErrorCodeInvalidArgument, "criterion obligation revision is invalid")
	}
	if !o.VerificationMode.Valid() {
		return NewError(ErrorCodeInvalidArgument, "criterion obligation verification mode is invalid")
	}
	return nil
}

type ConclusionObligations struct {
	RequiredCriteria     []CriterionObligationSnapshot `json:"required_criteria,omitempty"`
	RequiredObjectiveIDs []ID                          `json:"required_objective_ids,omitempty"`
	ResultSummaryRequired bool                         `json:"result_summary_required,omitempty"`
}

func (o ConclusionObligations) Validate() error {
	criteria := make(map[ID]struct{}, len(o.RequiredCriteria))
	for _, criterion := range o.RequiredCriteria {
		if err := criterion.Validate(); err != nil {
			return err
		}
		if _, exists := criteria[criterion.CriterionID]; exists {
			return NewError(ErrorCodeInvalidArgument, "duplicate criterion obligation")
		}
		criteria[criterion.CriterionID] = struct{}{}
	}
	objectives := make(map[ID]struct{}, len(o.RequiredObjectiveIDs))
	for _, objectiveID := range o.RequiredObjectiveIDs {
		if err := objectiveID.Validate(); err != nil {
			return WrapError(ErrorCodeInvalidArgument, "required objective id is invalid", err)
		}
		if _, exists := objectives[objectiveID]; exists {
			return NewError(ErrorCodeInvalidArgument, "duplicate required objective obligation")
		}
		objectives[objectiveID] = struct{}{}
	}
	return nil
}

type Conclusion struct {
	PrincipalID     string                   `json:"principal_id"`
	Actor           ActorRef                 `json:"actor_ref"`
	Reason          string                   `json:"reason"`
	ConcludedAt     time.Time                `json:"concluded_at"`
	Assessments     []CriterionAssessmentRef `json:"assessments,omitempty"`
	OwnerRef        *EntityRef               `json:"owner_ref,omitempty"`
	OwnerVersion    *Version                 `json:"owner_version,omitempty"`
	LifecycleResult string                   `json:"lifecycle_result,omitempty"`
	Obligations     ConclusionObligations    `json:"obligations"`
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
	if err := c.Obligations.Validate(); err != nil {
		return err
	}
	bound := c.OwnerRef != nil || c.OwnerVersion != nil || strings.TrimSpace(c.LifecycleResult) != ""
	if bound {
		if c.OwnerRef == nil || c.OwnerVersion == nil || strings.TrimSpace(c.LifecycleResult) == "" {
			return NewError(ErrorCodeInvalidArgument, "bound conclusion requires owner_ref, owner_version and lifecycle_result")
		}
		if err := c.OwnerRef.Validate(); err != nil {
			return WrapError(ErrorCodeInvalidArgument, "conclusion owner_ref is invalid", err)
		}
		if err := c.OwnerVersion.Validate(); err != nil {
			return WrapError(ErrorCodeInvalidArgument, "conclusion owner_version is invalid", err)
		}
	}
	return nil
}

func (c *Conclusion) Bind(owner EntityRef, ownerVersion Version, lifecycleResult string, obligations ConclusionObligations) error {
	if err := owner.Validate(); err != nil {
		return err
	}
	if err := ownerVersion.Validate(); err != nil {
		return err
	}
	lifecycleResult = strings.TrimSpace(lifecycleResult)
	if lifecycleResult == "" {
		return NewError(ErrorCodeInvalidArgument, "conclusion lifecycle_result is required")
	}
	if err := obligations.Validate(); err != nil {
		return err
	}
	ownerCopy := owner
	versionCopy := ownerVersion
	c.OwnerRef = &ownerCopy
	c.OwnerVersion = &versionCopy
	c.LifecycleResult = lifecycleResult
	c.Obligations = obligations
	return c.Validate()
}

func nextVersion(v Version) (Version, error) {
	return v.Next()
}
