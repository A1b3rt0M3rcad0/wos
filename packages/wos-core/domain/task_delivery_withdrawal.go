package domain

import (
	"reflect"
	"strings"
	"time"
)

// TaskDeliveryWithdrawal is an authorized, append-only withdrawal of one
// pending unsigned delivery. It does not revise the delivery or make a review
// decision; its history remains distinct from both records.
type TaskDeliveryWithdrawal struct {
	ID             ID        `json:"id"`
	Scope          Scope     `json:"scope"`
	WorkItemID     ID        `json:"work_item_id"`
	DeliveryID     ID        `json:"delivery_id"`
	DeliveryDigest string    `json:"delivery_digest"`
	Round          uint64    `json:"round,string"`
	PrincipalID    string    `json:"principal_id"`
	Actor          ActorRef  `json:"actor_ref"`
	Reason         string    `json:"reason"`
	WithdrawnAt    time.Time `json:"withdrawn_at"`
}

func (w TaskDeliveryWithdrawal) Validate() error {
	for _, id := range []ID{w.ID, w.WorkItemID, w.DeliveryID} {
		if err := id.Validate(); err != nil {
			return err
		}
	}
	if err := w.Scope.Validate(); err != nil {
		return err
	}
	if !ValidSignedDigest(w.DeliveryDigest) || w.Round == 0 || strings.TrimSpace(w.PrincipalID) == "" || strings.TrimSpace(w.Reason) == "" || len(w.Reason) > 4096 || w.WithdrawnAt.IsZero() {
		return NewError(ErrorCodeInvalidArgument, "TaskDelivery withdrawal requires an exact delivery, actor, reason and timestamp")
	}
	if err := w.Actor.Validate(); err != nil {
		return WrapError(ErrorCodeInvalidArgument, "TaskDelivery withdrawal actor is invalid", err)
	}
	return nil
}

func (w TaskDeliveryWithdrawal) ValidateForDelivery(delivery TaskDelivery) error {
	if err := delivery.Validate(); err != nil {
		return err
	}
	if err := w.Validate(); err != nil {
		return err
	}
	if w.Scope != delivery.Scope || w.WorkItemID != delivery.WorkItemID || w.DeliveryID != delivery.ID || w.DeliveryDigest != delivery.Digest || w.Round != delivery.Round || w.WithdrawnAt.Before(delivery.DeliveredAt) {
		return NewError(ErrorCodeSubmissionNotAccepted, "withdrawal does not target the exact TaskDelivery")
	}
	return nil
}

func ValidateTaskDeliveryWithdrawalMutation(old, next TaskDeliveryWithdrawal) error {
	if !reflect.DeepEqual(old, next) {
		return NewError(ErrorCodeInvalidTransition, "TaskDeliveryWithdrawal is immutable")
	}
	return next.Validate()
}
