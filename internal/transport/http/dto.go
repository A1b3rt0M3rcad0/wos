package httptransport

import (
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

type createOutcomeRequest struct {
	Title        string          `json:"title"`
	Description  string          `json:"description,omitempty"`
	DesiredState string          `json:"desired_state"`
	Priority     domain.Priority `json:"priority"`
}

type createObjectiveRequest struct {
	Title              string          `json:"title"`
	Description        string          `json:"description,omitempty"`
	Priority           domain.Priority `json:"priority"`
	RequiredForOutcome bool            `json:"required_for_outcome"`
	ParentObjectiveID  *string         `json:"parent_objective_id,omitempty"`
}

type createWorkItemRequest struct {
	Title       string                   `json:"title"`
	Description string                   `json:"description,omitempty"`
	Priority    domain.Priority          `json:"priority"`
	Lifecycle   domain.WorkItemLifecycle `json:"lifecycle,omitempty"`
	ObjectiveID *string                  `json:"objective_id,omitempty"`
	NotBefore   *time.Time               `json:"not_before,omitempty"`
}

type relationEndpointRequest struct {
	Kind domain.EntityKind `json:"kind"`
	ID   string            `json:"id"`
}

type createRelationRequest struct {
	SourceRef    relationEndpointRequest       `json:"source_ref"`
	RelationType domain.RelationType           `json:"relation_type"`
	TargetRef    relationEndpointRequest       `json:"target_ref"`
	Strength     domain.DependencyStrength     `json:"strength,omitempty"`
	Satisfaction domain.DependencySatisfaction `json:"satisfaction,omitempty"`
	Reason       string                        `json:"reason,omitempty"`
}

type criterionRequest struct {
	ExpectedVersion  *uint64                 `json:"expected_version,omitempty"`
	Title            string                  `json:"title"`
	Description      string                  `json:"description,omitempty"`
	Required         bool                    `json:"required"`
	VerificationMode domain.VerificationMode `json:"verification_mode"`
}

type assessmentRequest struct {
	ExpectedVersion   *uint64                  `json:"expected_version,omitempty"`
	CriterionRevision domain.CriterionRevision `json:"criterion_revision"`
	Result            domain.AssessmentResult  `json:"result"`
	Rationale         string                   `json:"rationale"`
}

type updateOutcomeRequest struct {
	ExpectedVersion *uint64          `json:"expected_version,omitempty"`
	Title           *string          `json:"title,omitempty"`
	Description     *string          `json:"description,omitempty"`
	DesiredState    *string          `json:"desired_state,omitempty"`
	Priority        *domain.Priority `json:"priority,omitempty"`
}

type updateObjectiveRequest struct {
	ExpectedVersion    *uint64          `json:"expected_version,omitempty"`
	Title              *string          `json:"title,omitempty"`
	Description        *string          `json:"description,omitempty"`
	Priority           *domain.Priority `json:"priority,omitempty"`
	RequiredForOutcome *bool            `json:"required_for_outcome,omitempty"`
}

type updateWorkItemRequest struct {
	ExpectedVersion *uint64          `json:"expected_version,omitempty"`
	Title           *string          `json:"title,omitempty"`
	Description     *string          `json:"description,omitempty"`
	Priority        *domain.Priority `json:"priority,omitempty"`
}

type versionRequest struct {
	ExpectedVersion *uint64 `json:"expected_version,omitempty"`
}

type reasonRequest struct {
	ExpectedVersion *uint64 `json:"expected_version,omitempty"`
	Reason          string  `json:"reason"`
}

type claimRequest struct {
	ExpectedVersion *uint64 `json:"expected_version,omitempty"`
	LeaseTTLSeconds int64   `json:"lease_ttl_seconds,omitempty"`
}

func (r claimRequest) leaseTTL() time.Duration {
	if r.LeaseTTLSeconds == 0 {
		return 0
	}
	return time.Duration(r.LeaseTTLSeconds) * time.Second
}

type releaseRequest struct {
	ExpectedVersion *uint64 `json:"expected_version,omitempty"`
	ClaimID         string  `json:"claim_id"`
	FencingToken    uint64  `json:"fencing_token"`
}

type completeRequest struct {
	ExpectedVersion *uint64 `json:"expected_version,omitempty"`
	ClaimID         string  `json:"claim_id"`
	FencingToken    uint64  `json:"fencing_token"`
	ResultSummary   string  `json:"result_summary"`
	Reason          string  `json:"reason"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	Details       any    `json:"details,omitempty"`
	Retryable     bool   `json:"retryable"`
	CorrelationID string `json:"correlation_id,omitempty"`
}

type collectionResponse[T any] struct {
	Items           []T                    `json:"items"`
	OutcomeRevision domain.OutcomeRevision `json:"outcome_revision"`
}
