package application_test

import (
	"context"
	"fmt"
	"sync"
	"testing"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func TestWithdrawTaskDeliveryRequiresExactCurrentPendingDeliveryAndIdempotency(t *testing.T) {
	permissions, registered := a.CommandAuthorizationPermissions("WithdrawTaskDelivery", a.WithdrawTaskDeliveryCommand{})
	if !registered || len(permissions) != 1 || permissions[0] != ports.PermissionTaskWithdrawDelivery {
		t.Fatalf("withdrawal authorization = %v (registered=%v), want task:withdraw_delivery", permissions, registered)
	}
	f, executor, _ := setupLegacyTaskForReview(t, d.VerificationModeAttestation)
	delivered := submitLegacyTaskDelivery(t, f, executor, d.TaskDeliveryMaterial{Summary: "Immutable result pending review"}, "e31-withdraw-delivery")
	command := a.WithdrawTaskDeliveryCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: delivered.WorkItem.Version,
		DeliveryID: delivered.Delivery.ID, ExpectedDeliveryDigest: delivered.Delivery.Digest,
		ExpectedDeliveryRound: delivered.Delivery.Round, Reason: "The result needs a planned correction",
	}
	beforeEvents := len(f.store.SnapshotDomainEvents(f.scope))
	beforeReceipts := f.store.IdempotencyRecordCount()
	beforeRevision := readTaskDeliveryRevision(t, f)
	if _, err := f.service.WithdrawTaskDelivery(f.identityContext(executor), f.cc, command); errorCode(err) != d.ErrorCodeInvalidArgument {
		t.Fatalf("empty idempotency key error = %v, want invalid argument", err)
	}
	if _, err := f.service.WithdrawTaskDelivery(context.Background(), freshCommandContext(f.cc, "e31-withdraw-unauthenticated"), command); errorCode(err) != d.ErrorCodeForbidden {
		t.Fatalf("unauthenticated withdrawal error = %v, want forbidden", err)
	}

	staleVersion := command
	staleVersion.ExpectedWorkItemVersion--
	if _, err := f.service.WithdrawTaskDelivery(f.identityContext(executor), freshCommandContext(f.cc, "e31-withdraw-stale-version"), staleVersion); errorCode(err) != d.ErrorCodeVersionConflict {
		t.Fatalf("stale Task version error = %v, want version conflict", err)
	}
	wrongDigest := command
	wrongDigest.ExpectedDeliveryDigest = "sha256:" + "0000000000000000000000000000000000000000000000000000000000000000"
	if _, err := f.service.WithdrawTaskDelivery(f.identityContext(executor), freshCommandContext(f.cc, "e31-withdraw-wrong-digest"), wrongDigest); errorCode(err) != d.ErrorCodeVersionConflict {
		t.Fatalf("wrong delivery digest error = %v, want version conflict", err)
	}
	wrongRound := command
	wrongRound.ExpectedDeliveryRound++
	if _, err := f.service.WithdrawTaskDelivery(f.identityContext(executor), freshCommandContext(f.cc, "e31-withdraw-wrong-round"), wrongRound); errorCode(err) != d.ErrorCodeVersionConflict {
		t.Fatalf("wrong delivery round error = %v, want version conflict", err)
	}
	if work, err := f.service.GetWorkItem(context.Background(), f.scope, f.work.ID); err != nil || work.Value.Version != delivered.WorkItem.Version || work.Value.PendingDeliveryID == nil || *work.Value.PendingDeliveryID != delivered.Delivery.ID {
		t.Fatalf("rejected withdrawals changed Task state: work=%+v err=%v", work.Value, err)
	}
	if events := f.store.SnapshotDomainEvents(f.scope); len(events) != beforeEvents {
		t.Fatalf("rejected withdrawals emitted events: before=%d after=%d", beforeEvents, len(events))
	}
	if receipts := f.store.IdempotencyRecordCount(); receipts != beforeReceipts {
		t.Fatalf("rejected withdrawals stored receipts: before=%d after=%d", beforeReceipts, receipts)
	}
	if revision := readTaskDeliveryRevision(t, f); revision != beforeRevision {
		t.Fatalf("rejected withdrawals advanced revision: before=%d after=%d", beforeRevision, revision)
	}
}

func TestWithdrawTaskDeliveryIsAppendOnlyAndRequiresExplicitBacklogReturnForReplanning(t *testing.T) {
	f, executor, _ := setupLegacyTaskForReview(t, d.VerificationModeAttestation)
	first := submitLegacyTaskDelivery(t, f, executor, d.TaskDeliveryMaterial{Summary: "First immutable result"}, "e31-withdraw-first-delivery")
	commandContext := freshCommandContext(f.cc, "e31-withdraw-first")
	command := a.WithdrawTaskDeliveryCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: first.WorkItem.Version,
		DeliveryID: first.Delivery.ID, ExpectedDeliveryDigest: first.Delivery.Digest,
		ExpectedDeliveryRound: first.Delivery.Round, Reason: "Replan the requested work before a corrected delivery",
	}
	withdrawn, err := f.service.WithdrawTaskDelivery(f.identityContext(executor), commandContext, command)
	if err != nil {
		t.Fatalf("WithdrawTaskDelivery() error = %v", err)
	}
	if withdrawn.Value.WorkItem.Lifecycle != d.WorkItemLifecycleTodo || withdrawn.Value.WorkItem.PendingDeliveryID != nil || withdrawn.Value.WorkItem.CurrentLease != nil || withdrawn.Value.WorkItem.CurrentContractID != nil {
		t.Fatalf("withdrawal did not return the Task to To do without execution authority: %+v", withdrawn.Value.WorkItem)
	}
	if withdrawn.Value.WorkItem.LatestDeliveryID == nil || *withdrawn.Value.WorkItem.LatestDeliveryID != first.Delivery.ID || withdrawn.Value.WorkItem.LatestDeliveryDecisionID != nil || withdrawn.Value.WorkItem.LatestDeliveryWithdrawalID == nil || *withdrawn.Value.WorkItem.LatestDeliveryWithdrawalID != withdrawn.Value.Withdrawal.ID {
		t.Fatalf("withdrawal did not preserve and resolve the latest delivery pointer: %+v", withdrawn.Value.WorkItem)
	}
	if withdrawn.Value.Withdrawal.PrincipalID != f.cc.PrincipalID || withdrawn.Value.Withdrawal.Actor != f.cc.Actor || withdrawn.Value.Withdrawal.Reason != command.Reason || withdrawn.Value.Withdrawal.DeliveryDigest != first.Delivery.Digest || withdrawn.Value.Withdrawal.Round != first.Delivery.Round {
		t.Fatalf("withdrawal audit record lost actor or delivery binding: %+v", withdrawn.Value.Withdrawal)
	}
	var withdrawalEvent *d.DomainEvent
	for _, event := range f.store.SnapshotDomainEvents(f.scope) {
		if event.EventType == "task_delivery.withdrawn" && event.AggregateRef.ID == f.work.ID {
			copy := event
			withdrawalEvent = &copy
			break
		}
	}
	if withdrawalEvent == nil {
		t.Fatal("successful withdrawal did not append a TaskDelivery withdrawal event")
	}
	if err := first.Delivery.Validate(); err != nil {
		t.Fatalf("withdrawal mutated immutable delivery: %v", err)
	}
	replayed, err := f.service.WithdrawTaskDelivery(f.identityContext(executor), commandContext, command)
	if err != nil || !replayed.IdempotentReplay || replayed.Value.Withdrawal.ID != withdrawn.Value.Withdrawal.ID {
		t.Fatalf("withdrawal receipt replay = (%+v, %v)", replayed, err)
	}
	_, otherCredential := f.addCredentialTree(t,
		"0199f2e1-0000-7000-8000-000000000001", "0199f2e2-0000-7000-8000-000000000001",
		"e31-withdraw-replay-other-root", "e31-withdraw-replay-other-session", f.cc.PrincipalID, f.cc.Actor)
	if _, err := f.service.WithdrawTaskDelivery(f.identityContext(otherCredential), commandContext, command); errorCode(err) != d.ErrorCodeIdempotencyConflict {
		t.Fatalf("different credential replay error = %v, want idempotency conflict", err)
	}
	duplicateCommand := command
	duplicateCommand.ExpectedWorkItemVersion = withdrawn.Value.WorkItem.Version
	if _, err := f.service.WithdrawTaskDelivery(f.identityContext(executor), freshCommandContext(f.cc, "e31-withdraw-second-attempt"), duplicateCommand); errorCode(err) != d.ErrorCodeSubmissionNotAccepted {
		t.Fatalf("new-key duplicate withdrawal error = %v, want submission not accepted", err)
	}

	stored, err := readTaskDeliveryWithdrawal(t, f, withdrawn.Value.Withdrawal.ID)
	if err != nil || stored.ID != withdrawn.Value.Withdrawal.ID || stored.Reason != command.Reason {
		t.Fatalf("persisted withdrawal = %+v, err=%v", stored, err)
	}
	byDelivery, err := readTaskDeliveryWithdrawalForDelivery(t, f, first.Delivery.ID)
	if err != nil || byDelivery == nil || byDelivery.ID != stored.ID {
		t.Fatalf("withdrawal lookup by delivery = %+v, err=%v", byDelivery, err)
	}

	changedDefinition := a.AddCriterionCommand{
		Owner: f.work.Ref(), ExpectedVersion: withdrawn.Value.WorkItem.Version,
		Title: "A planned correction criterion", Required: true, VerificationMode: d.VerificationModeAttestation,
	}
	if _, err := f.service.AddCriterion(context.Background(), freshCommandContext(f.cc, "e31-withdraw-reject-edit-todo"), changedDefinition); err == nil {
		t.Fatal("changed the Task definition before explicitly returning it to Backlog")
	}
	if _, err := f.service.ClaimWorkItem(f.identityContext(executor), freshCommandContext(f.cc, "e31-withdraw-reject-direct-claim"), a.ClaimWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: withdrawn.Value.WorkItem.Version, TTL: d.DefaultLeaseTTL,
	}); errorCode(err) != d.ErrorCodeInvalidTransition {
		t.Fatalf("ClaimWorkItem() before explicit Backlog replan error = %v, want invalid transition", err)
	}
	stillWithdrawn, err := f.service.GetWorkItem(context.Background(), f.scope, f.work.ID)
	if err != nil || stillWithdrawn.Value.Version != withdrawn.Value.WorkItem.Version || stillWithdrawn.Value.Lifecycle != d.WorkItemLifecycleTodo || stillWithdrawn.Value.CurrentLease != nil || stillWithdrawn.Value.TaskDeliveryReplanAuthorized {
		t.Fatalf("rejected direct claim changed withdrawn Task state: work=%+v err=%v", stillWithdrawn.Value, err)
	}
	backlog, err := f.service.DeferWorkItem(context.Background(), freshCommandContext(f.cc, "e31-withdraw-return-backlog"), a.DeferWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: withdrawn.Value.WorkItem.Version,
	})
	if err != nil {
		t.Fatalf("DeferWorkItem() after withdrawal = %v", err)
	}
	if backlog.Value.Lifecycle != d.WorkItemLifecycleBacklog || !backlog.Value.TaskDeliveryReplanAuthorized {
		t.Fatalf("explicit Backlog defer did not authorize replanning: %+v", backlog.Value)
	}
	changedDefinition.ExpectedVersion = backlog.Value.Version
	if _, err := f.service.AddCriterion(context.Background(), freshCommandContext(f.cc, "e31-withdraw-edit-after-backlog"), changedDefinition); err != nil {
		t.Fatalf("AddCriterion() after explicit Backlog return = %v", err)
	}
	current, err := f.service.GetWorkItem(context.Background(), f.scope, f.work.ID)
	if err != nil {
		t.Fatal(err)
	}
	activated, err := f.service.ActivateWorkItem(context.Background(), freshCommandContext(f.cc, "e31-withdraw-reactivate"), a.ActivateWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: current.Value.Version,
	})
	if err != nil {
		t.Fatalf("ActivateWorkItem() after replanning = %v", err)
	}
	claimed, err := f.service.ClaimWorkItem(f.identityContext(executor), freshCommandContext(f.cc, "e31-withdraw-new-claim"), a.ClaimWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: activated.Value.Version, TTL: d.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatalf("ClaimWorkItem() after replanning = %v", err)
	}
	if !claimed.Value.TaskDeliveryReplanAuthorized || claimed.Value.LatestDeliveryWithdrawalID == nil || *claimed.Value.LatestDeliveryWithdrawalID != withdrawn.Value.Withdrawal.ID {
		t.Fatalf("claim did not preserve replan authorization and withdrawal history: %+v", claimed.Value)
	}
	f.work = claimed.Value
	second := submitLegacyTaskDelivery(t, f, executor, d.TaskDeliveryMaterial{Summary: "Corrected result after withdrawal"}, "e31-withdraw-second-delivery")
	if second.Delivery.Round != first.Delivery.Round+1 || second.Delivery.SupersedesDeliveryID == nil || *second.Delivery.SupersedesDeliveryID != first.Delivery.ID {
		t.Fatalf("new delivery chain = round %d supersedes %v; want round %d superseding %s", second.Delivery.Round, second.Delivery.SupersedesDeliveryID, first.Delivery.Round+1, first.Delivery.ID)
	}
	if second.WorkItem.TaskDeliveryReplanAuthorized || second.WorkItem.LatestDeliveryWithdrawalID != nil || second.WorkItem.LatestDeliveryID == nil || *second.WorkItem.LatestDeliveryID != second.Delivery.ID {
		t.Fatalf("new delivery did not close replanning marker while advancing delivery navigation: %+v", second.WorkItem)
	}
}

func TestTaskDeliveryReviewDecisionAndWithdrawalCannotBothResolveSameDelivery(t *testing.T) {
	f, executor, _ := setupLegacyTaskForReview(t, d.VerificationModeAttestation)
	delivered := submitLegacyTaskDelivery(t, f, executor, d.TaskDeliveryMaterial{Summary: "Competing review/withdrawal target"}, "e31-race-delivery")
	reviewer, reviewerCC := newReviewPrincipal(t, f, "e31-race-reviewer", "reviewer-e31-race")
	serviceTwo, err := a.NewService(f.store, fixedClock{now: f.now}, &sequenceIDs{next: 9900})
	if err != nil {
		t.Fatal(err)
	}
	withdrawCommand := a.WithdrawTaskDeliveryCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: delivered.WorkItem.Version,
		DeliveryID: delivered.Delivery.ID, ExpectedDeliveryDigest: delivered.Delivery.Digest,
		ExpectedDeliveryRound: delivered.Delivery.Round, Reason: "Withdraw the exact pending handoff",
	}
	decisionCommand := a.RequestTaskDeliveryChangesCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: delivered.WorkItem.Version,
		DeliveryID: delivered.Delivery.ID, ExpectedDeliveryDigest: delivered.Delivery.Digest,
		ExpectedDeliveryRound: delivered.Delivery.Round, Reason: "Review the exact pending handoff",
		Findings: []string{"Revise the result"},
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var group sync.WaitGroup
	group.Add(2)
	go func() {
		defer group.Done()
		<-start
		_, err := f.service.WithdrawTaskDelivery(f.identityContext(executor), freshCommandContext(f.cc, "e31-race-withdraw"), withdrawCommand)
		results <- err
	}()
	go func() {
		defer group.Done()
		_, err := serviceTwo.RequestTaskDeliveryChanges(f.identityContext(reviewer), freshCommandContext(reviewerCC, "e31-race-review"), decisionCommand)
		results <- err
	}()
	close(start)
	group.Wait()
	close(results)
	successes := 0
	for err := range results {
		if err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("competing decision/withdrawal produced %d successful resolutions; want exactly one", successes)
	}
	current, err := f.service.GetWorkItem(context.Background(), f.scope, f.work.ID)
	if err != nil {
		t.Fatal(err)
	}
	decisionSet := current.Value.LatestDeliveryDecisionID != nil
	withdrawalSet := current.Value.LatestDeliveryWithdrawalID != nil
	if decisionSet == withdrawalSet || current.Value.PendingDeliveryID != nil {
		t.Fatalf("current delivery has conflicting or missing resolution pointers: %+v", current.Value)
	}
	uow, err := f.store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	repository, ok := uow.(ports.TaskDeliveryUnitOfWork)
	if !ok {
		t.Fatal("Memory UoW lacks TaskDelivery persistence")
	}
	decision, err := repository.TaskDeliveries().DecisionForDelivery(context.Background(), f.scope, delivered.Delivery.ID)
	if err != nil {
		t.Fatal(err)
	}
	withdrawal, err := repository.TaskDeliveries().WithdrawalForDelivery(context.Background(), f.scope, delivered.Delivery.ID)
	if err != nil {
		t.Fatal(err)
	}
	if (decision != nil) != decisionSet || (withdrawal != nil) != withdrawalSet || (decision != nil && withdrawal != nil) {
		t.Fatalf("persisted delivery resolutions conflict: decision=%+v withdrawal=%+v Task=%+v", decision, withdrawal, current.Value)
	}
}

func readTaskDeliveryWithdrawal(t *testing.T, f credentialLeaseFixture, withdrawalID d.ID) (d.TaskDeliveryWithdrawal, error) {
	t.Helper()
	uow, err := f.store.Begin(context.Background())
	if err != nil {
		return d.TaskDeliveryWithdrawal{}, err
	}
	defer uow.Rollback()
	deliveryUOW, ok := uow.(ports.TaskDeliveryUnitOfWork)
	if !ok {
		return d.TaskDeliveryWithdrawal{}, fmt.Errorf("Memory UoW lacks TaskDelivery persistence")
	}
	return deliveryUOW.TaskDeliveries().GetWithdrawal(context.Background(), f.scope, withdrawalID)
}

func readTaskDeliveryWithdrawalForDelivery(t *testing.T, f credentialLeaseFixture, deliveryID d.ID) (*d.TaskDeliveryWithdrawal, error) {
	t.Helper()
	uow, err := f.store.Begin(context.Background())
	if err != nil {
		return nil, err
	}
	defer uow.Rollback()
	deliveryUOW, ok := uow.(ports.TaskDeliveryUnitOfWork)
	if !ok {
		return nil, fmt.Errorf("Memory UoW lacks TaskDelivery persistence")
	}
	return deliveryUOW.TaskDeliveries().WithdrawalForDelivery(context.Background(), f.scope, deliveryID)
}
