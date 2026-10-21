package domain_test

import (
	"errors"
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestTaskDeliveryWithdrawalRemainsResolvedThroughExplicitBacklogReturn(t *testing.T) {
	now := time.Date(2026, 10, 21, 14, 0, 0, 0, time.UTC)
	work, _, credentialID, actor := taskDeliveryFixture(t, now)
	claimTaskDeliveryFixture(t, &work, credentialID, actor, now.Add(time.Second), d.MustParseID("01a11774-0000-7000-8000-000000000001"))
	deliveredAt := now.Add(2 * time.Second)
	delivery, err := d.NewLegacyTaskDelivery(d.MustParseID("01a11774-0000-7000-8000-000000000002"), work, d.TaskDeliveryMaterial{Summary: "Immutable work result"}, "executor", &credentialID, deliveredAt)
	if err != nil {
		t.Fatal(err)
	}
	if err := work.RecordTaskDelivery(delivery, nil, nil, work.Version, deliveredAt); err != nil {
		t.Fatal(err)
	}
	withdrawnAt := deliveredAt.Add(time.Second)
	withdrawal := d.TaskDeliveryWithdrawal{
		ID: d.MustParseID("01a11774-0000-7000-8000-000000000003"), Scope: work.Scope, WorkItemID: work.ID,
		DeliveryID: delivery.ID, DeliveryDigest: delivery.Digest, Round: delivery.Round,
		PrincipalID: "planner", Actor: actor, Reason: "Return the work to planning before another execution", WithdrawnAt: withdrawnAt,
	}
	before := work.Version
	if err := work.WithdrawTaskDelivery(delivery, withdrawal, before, withdrawnAt); err != nil {
		t.Fatal(err)
	}
	if work.Lifecycle != d.WorkItemLifecycleTodo || work.PendingDeliveryID != nil || work.CurrentLease != nil || work.LatestDeliveryWithdrawalID == nil || *work.LatestDeliveryWithdrawalID != withdrawal.ID {
		t.Fatalf("withdrawal state = %+v", work)
	}
	if err := work.Claim(d.MustParseID("01a11774-0000-7000-8000-000000000004"), "executor", actor, d.MinLeaseTTL, withdrawnAt.Add(500*time.Millisecond)); errorCode(err) != d.ErrorCodeInvalidTransition {
		t.Fatalf("claim before explicit replan = %v, want invalid transition", err)
	}
	if err := work.Defer(withdrawnAt.Add(time.Second)); err != nil {
		t.Fatalf("explicit return to Backlog = %v", err)
	}
	if !work.TaskDeliveryReplanAuthorized {
		t.Fatal("explicit Defer did not set the durable replan authorization marker")
	}
	if err := work.Validate(); err != nil {
		t.Fatalf("withdrawal history is invalid after Backlog return: %v", err)
	}
	if work.Lifecycle != d.WorkItemLifecycleBacklog || work.LatestDeliveryID == nil || *work.LatestDeliveryID != delivery.ID || work.LatestDeliveryWithdrawalID == nil || *work.LatestDeliveryWithdrawalID != withdrawal.ID {
		t.Fatalf("Backlog return lost withdrawal resolution history: %+v", work)
	}
	if err := delivery.Validate(); err != nil {
		t.Fatalf("withdrawal changed the immutable delivery: %v", err)
	}
	if err := work.Activate(withdrawnAt.Add(2 * time.Second)); err != nil {
		t.Fatalf("activate after explicit replan = %v", err)
	}
	if err := work.Claim(d.MustParseID("01a11774-0000-7000-8000-000000000005"), "executor", actor, d.MinLeaseTTL, withdrawnAt.Add(3*time.Second)); err != nil {
		t.Fatalf("claim after explicit replan = %v", err)
	}
	if !work.TaskDeliveryReplanAuthorized || work.LatestDeliveryWithdrawalID == nil || *work.LatestDeliveryWithdrawalID != withdrawal.ID || work.CurrentLease == nil {
		t.Fatalf("replanned execution lost authorization or withdrawal history: %+v", work)
	}
}

func TestTaskDeliveryReplanAuthorizationRequiresExactWithdrawnHistory(t *testing.T) {
	now := time.Date(2026, 10, 21, 14, 30, 0, 0, time.UTC)
	work, _, _, _ := taskDeliveryFixture(t, now)
	work.TaskDeliveryReplanAuthorized = true
	if err := work.Validate(); errorCode(err) != d.ErrorCodeInvalidArgument {
		t.Fatalf("replan marker without withdrawn delivery = %v, want invalid argument", err)
	}
}

func TestTaskDeliveryWithdrawalMustBindExactDeliveryRound(t *testing.T) {
	now := time.Date(2026, 10, 21, 15, 0, 0, 0, time.UTC)
	work, _, credentialID, actor := taskDeliveryFixture(t, now)
	claimTaskDeliveryFixture(t, &work, credentialID, actor, now.Add(time.Second), d.MustParseID("01a11775-0000-7000-8000-000000000001"))
	deliveredAt := now.Add(2 * time.Second)
	delivery, err := d.NewLegacyTaskDelivery(d.MustParseID("01a11775-0000-7000-8000-000000000002"), work, d.TaskDeliveryMaterial{Summary: "Immutable work result"}, "executor", &credentialID, deliveredAt)
	if err != nil {
		t.Fatal(err)
	}
	withdrawal := d.TaskDeliveryWithdrawal{
		ID: d.MustParseID("01a11775-0000-7000-8000-000000000003"), Scope: work.Scope, WorkItemID: work.ID,
		DeliveryID: delivery.ID, DeliveryDigest: "sha256:0000000000000000000000000000000000000000000000000000000000000000", Round: delivery.Round,
		PrincipalID: "planner", Actor: actor, Reason: "Exact delivery binding is required", WithdrawnAt: deliveredAt.Add(time.Second),
	}
	if err := withdrawal.ValidateForDelivery(delivery); err == nil {
		t.Fatal("withdrawal accepted a digest that differs from the immutable delivery")
	}
}

func errorCode(err error) d.ErrorCode {
	var coded *d.Error
	if errors.As(err, &coded) {
		return coded.Code
	}
	return ""
}
