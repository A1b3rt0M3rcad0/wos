package domain

import (
	"reflect"
	"strings"
	"time"
)

type TaskDeliveryDecisionResult string

const (
	TaskDeliveryDecisionAccepted         TaskDeliveryDecisionResult = "accepted"
	TaskDeliveryDecisionChangesRequested TaskDeliveryDecisionResult = "changes_requested"
)

// TaskDeliveryDecision is an append-only reviewer decision against one exact
// immutable delivery round. It is separate from criterion assessment and the
// Task Conclusion.
type TaskDeliveryDecision struct {
	ID                  ID                         `json:"id"`
	Scope               Scope                      `json:"scope"`
	WorkItemID          ID                         `json:"work_item_id"`
	DeliveryID          ID                         `json:"delivery_id"`
	DeliveryDigest      string                     `json:"delivery_digest"`
	Round               uint64                     `json:"round,string"`
	Result              TaskDeliveryDecisionResult `json:"result"`
	ReviewerPrincipalID string                     `json:"reviewer_principal_id"`
	ReviewerActor       ActorRef                   `json:"reviewer_actor_ref"`
	Reason              string                     `json:"reason"`
	Findings            []string                   `json:"findings,omitempty"`
	DecidedAt           time.Time                  `json:"decided_at"`
}

func NewTaskDeliveryDecision(id ID, delivery TaskDelivery, result TaskDeliveryDecisionResult, reviewerPrincipalID string, reviewerActor ActorRef, reason string, findings []string, decidedAt time.Time) (TaskDeliveryDecision, error) {
	decision := TaskDeliveryDecision{
		ID: id, Scope: delivery.Scope, WorkItemID: delivery.WorkItemID,
		DeliveryID: delivery.ID, DeliveryDigest: delivery.Digest, Round: delivery.Round,
		Result: result, ReviewerPrincipalID: strings.TrimSpace(reviewerPrincipalID),
		ReviewerActor: reviewerActor, Reason: strings.TrimSpace(reason),
		Findings: append([]string(nil), findings...), DecidedAt: decidedAt.UTC(),
	}
	if err := decision.ValidateForDelivery(delivery); err != nil {
		return TaskDeliveryDecision{}, err
	}
	return decision, nil
}

func (d TaskDeliveryDecision) Validate() error {
	for _, id := range []ID{d.ID, d.WorkItemID, d.DeliveryID} {
		if err := id.Validate(); err != nil {
			return err
		}
	}
	if err := d.Scope.Validate(); err != nil {
		return err
	}
	if !ValidSignedDigest(d.DeliveryDigest) || d.Round == 0 || strings.TrimSpace(d.ReviewerPrincipalID) == "" || strings.TrimSpace(d.Reason) == "" || d.DecidedAt.IsZero() {
		return NewError(ErrorCodeInvalidArgument, "delivery decision requires exact target, reviewer, rationale and timestamp")
	}
	if err := d.ReviewerActor.Validate(); err != nil {
		return WrapError(ErrorCodeInvalidArgument, "delivery decision reviewer actor is invalid", err)
	}
	if len(d.Findings) > 100 {
		return NewError(ErrorCodeInvalidArgument, "delivery decision has too many findings")
	}
	for _, finding := range d.Findings {
		if strings.TrimSpace(finding) == "" || len(finding) > 4096 {
			return NewError(ErrorCodeInvalidArgument, "delivery decision finding must be nonempty and bounded")
		}
	}
	switch d.Result {
	case TaskDeliveryDecisionAccepted:
		if len(d.Findings) != 0 {
			return NewError(ErrorCodeInvalidArgument, "accepted delivery cannot carry correction findings")
		}
	case TaskDeliveryDecisionChangesRequested:
		if len(d.Findings) == 0 {
			return NewError(ErrorCodeInvalidArgument, "changes_requested decision requires at least one finding")
		}
	default:
		return NewError(ErrorCodeInvalidArgument, "unknown TaskDelivery decision")
	}
	return nil
}

func (d TaskDeliveryDecision) ValidateForDelivery(delivery TaskDelivery) error {
	if err := delivery.Validate(); err != nil {
		return err
	}
	if err := d.Validate(); err != nil {
		return err
	}
	if d.Scope != delivery.Scope || d.WorkItemID != delivery.WorkItemID || d.DeliveryID != delivery.ID || d.DeliveryDigest != delivery.Digest || d.Round != delivery.Round || d.DecidedAt.Before(delivery.DeliveredAt) {
		return NewError(ErrorCodeSubmissionNotAccepted, "decision does not target the exact TaskDelivery round")
	}
	return nil
}

// ValidateTaskDeliveryDecisionMutation prevents a stored decision from being
// rewritten after it has entered history.
func ValidateTaskDeliveryDecisionMutation(old, next TaskDeliveryDecision) error {
	if !reflect.DeepEqual(old, next) {
		return NewError(ErrorCodeInvalidTransition, "TaskDeliveryDecision is immutable")
	}
	return next.Validate()
}
