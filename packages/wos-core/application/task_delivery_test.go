package application_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

func TestSubmitTaskForReviewLegacyRequiresExactBoundLeaseAndStoresProtectedReceipt(t *testing.T) {
	f := newCredentialLeaseFixture(t)
	root, session := f.addCredentialTree(t,
		"0199f281-0000-7000-8000-000000000001", "0199f282-0000-7000-8000-000000000001",
		"delivery-legacy-root", "delivery-legacy-session", f.cc.PrincipalID, f.cc.Actor)
	_, otherSession := f.addCredentialTree(t,
		"0199f281-0000-7000-8000-000000000002", "0199f282-0000-7000-8000-000000000002",
		"delivery-other-root", "delivery-other-session", f.cc.PrincipalID, f.cc.Actor)

	claimCC := f.cc
	claimCC.IdempotencyKey = "task-delivery-legacy-claim"
	claimed, err := f.service.ClaimWorkItem(f.identityContext(session), claimCC, a.ClaimWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: f.work.Version, TTL: d.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatalf("ClaimWorkItem() error = %v", err)
	}
	lease := claimed.Value.CurrentLease
	if lease == nil || lease.HolderCredentialID != root.ID {
		t.Fatalf("claim did not bind to root credential: %+v", lease)
	}

	command := a.SubmitTaskForReviewCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: claimed.Value.Version,
		ClaimID: &lease.ClaimID, FencingToken: lease.FencingToken,
		LegacyMaterial: ptrTaskDeliveryMaterial(d.TaskDeliveryMaterial{Summary: "Implemented the requested output"}),
	}
	invalid := command
	invalidCC := f.cc
	invalidCC.IdempotencyKey = "task-delivery-invalid-evidence"
	invalid.LegacyMaterial = ptrTaskDeliveryMaterial(d.TaskDeliveryMaterial{Summary: "references evidence outside the registry", EvidenceIDs: []d.ID{id("0199f283-0000-7000-8000-000000000099")}})
	if _, err := f.service.SubmitTaskForReview(f.identityContext(session), invalidCC, invalid); err == nil {
		t.Fatal("legacy delivery accepted evidence that is not registered in the current scope")
	}

	stale := command
	stale.ExpectedWorkItemVersion = f.work.Version
	staleCC := f.cc
	staleCC.IdempotencyKey = "task-delivery-stale-work-version"
	if _, err := f.service.SubmitTaskForReview(f.identityContext(session), staleCC, stale); errorCode(err) != d.ErrorCodeVersionConflict {
		t.Fatalf("stale WorkItem version error = %v, want version conflict", err)
	}
	staleClaim := command
	staleClaimID := id("0199f283-0000-7000-8000-000000000088")
	staleClaim.ClaimID = &staleClaimID
	if _, err := f.service.SubmitTaskForReview(f.identityContext(session), freshCommandContext(f.cc, "task-delivery-stale-claim"), staleClaim); errorCode(err) != d.ErrorCodeLease {
		t.Fatalf("stale claim error = %v, want lease error", err)
	}
	staleFence := command
	staleFence.FencingToken = lease.FencingToken + 1
	if _, err := f.service.SubmitTaskForReview(f.identityContext(session), freshCommandContext(f.cc, "task-delivery-stale-fence"), staleFence); errorCode(err) != d.ErrorCodeLease {
		t.Fatalf("stale fencing token error = %v, want lease error", err)
	}
	expiredService, err := a.NewService(f.store, fixedClock{now: lease.ExpiresAt}, &sequenceIDs{next: 9700})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := expiredService.SubmitTaskForReview(f.identityContext(session), freshCommandContext(f.cc, "task-delivery-expired-lease"), command); errorCode(err) != d.ErrorCodeLease {
		t.Fatalf("submission at lease expiry error = %v, want lease error", err)
	}
	blocked, err := f.service.CreateBlocker(f.identityContext(session), freshCommandContext(f.cc, "task-delivery-block-task"), a.CreateBlockerCommand{
		Scope: f.scope, BlockedRef: f.work.Ref(), Description: "Review prerequisite is blocked", Propagation: d.BlockerPropagationDirect,
	})
	if err != nil {
		t.Fatalf("CreateBlocker() error = %v", err)
	}
	if _, err := f.service.SubmitTaskForReview(f.identityContext(session), freshCommandContext(f.cc, "task-delivery-active-blocker"), command); errorCode(err) != d.ErrorCodePreconditionFailed {
		t.Fatalf("blocked Task submission error = %v, want precondition failure", err)
	}
	if _, err := f.service.ResolveBlocker(f.identityContext(session), freshCommandContext(f.cc, "task-delivery-resolve-blocker"), a.ResolveBlockerCommand{
		Scope: f.scope, BlockerID: blocked.Value.ID, ExpectedVersion: blocked.Value.Version, ResolutionSummary: "Unblocked for delivery review",
	}); err != nil {
		t.Fatalf("ResolveBlocker() error = %v", err)
	}

	wrongOwnerCC := f.cc
	wrongOwnerCC.IdempotencyKey = "task-delivery-wrong-credential"
	if _, err := f.service.SubmitTaskForReview(f.identityContext(otherSession), wrongOwnerCC, command); err == nil {
		t.Fatal("different root credential submitted another credential's legacy lease")
	}

	submitCC := f.cc
	submitCC.IdempotencyKey = "task-delivery-legacy-submit"
	submitted, err := f.service.SubmitTaskForReview(f.identityContext(session), submitCC, command)
	if err != nil {
		t.Fatalf("SubmitTaskForReview() error = %v", err)
	}
	delivery := submitted.Value.Delivery
	if delivery.Authority.Kind != d.TaskDeliveryAuthorityLegacyLease || delivery.Authority.CredentialID == nil || *delivery.Authority.CredentialID != root.ID || delivery.Material == nil {
		t.Fatalf("delivery authority/material = %+v", delivery)
	}
	if submitted.Value.WorkItem.CurrentLease != nil || submitted.Value.WorkItem.PendingDeliveryID == nil || *submitted.Value.WorkItem.PendingDeliveryID != delivery.ID {
		t.Fatal("delivery did not end the lease and become the pending review item")
	}

	replay, err := f.service.SubmitTaskForReview(f.identityContext(session), submitCC, command)
	if err != nil || !replay.IdempotentReplay || replay.Value.Delivery.ID != delivery.ID {
		t.Fatalf("same-root delivery receipt replay = (%v, %v)", replay.IdempotentReplay, err)
	}
	if _, err := f.service.SubmitTaskForReview(f.identityContext(otherSession), submitCC, command); errorCode(err) != d.ErrorCodeIdempotencyConflict {
		t.Fatalf("different root delivery receipt replay error = %v", err)
	}
	if _, err := f.service.SubmitTaskForReview(f.identityContext(session), freshCommandContext(f.cc, "task-delivery-already-pending"), command); errorCode(err) != d.ErrorCodeInvalidTransition {
		t.Fatalf("second pending delivery error = %v", err)
	}

	stored, err := readTaskDelivery(t, f.store, f.scope, delivery.ID)
	if err != nil || stored.Digest != delivery.Digest || stored.Authority.CredentialID == nil || *stored.Authority.CredentialID != root.ID {
		t.Fatalf("stored delivery = %+v, err=%v", stored, err)
	}
}

func TestSubmitTaskForReviewRequiresIdempotencyKeyWithoutMutation(t *testing.T) {
	f := newCredentialLeaseFixture(t)
	_, session := f.addCredentialTree(t,
		"0199f28b-0000-7000-8000-000000000001", "0199f28c-0000-7000-8000-000000000001",
		"delivery-no-key-root", "delivery-no-key-session", f.cc.PrincipalID, f.cc.Actor)
	claimed, err := f.service.ClaimWorkItem(f.identityContext(session), freshCommandContext(f.cc, "delivery-no-key-claim"), a.ClaimWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: f.work.Version, TTL: d.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatalf("ClaimWorkItem() error = %v", err)
	}
	beforeEvents := len(f.store.SnapshotDomainEvents(f.scope))
	beforeReceipts := f.store.IdempotencyRecordCount()
	beforeRevision := readTaskDeliveryRevision(t, f)
	command := a.SubmitTaskForReviewCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: claimed.Value.Version,
		ClaimID: &claimed.Value.CurrentLease.ClaimID, FencingToken: claimed.Value.CurrentLease.FencingToken,
		LegacyMaterial: ptrTaskDeliveryMaterial(d.TaskDeliveryMaterial{Summary: "Must not be recorded without a key"}),
	}
	if _, err := f.service.SubmitTaskForReview(f.identityContext(session), f.cc, command); errorCode(err) != d.ErrorCodeInvalidArgument {
		t.Fatalf("empty idempotency key error = %v, want invalid argument", err)
	}
	after, err := f.service.GetWorkItem(context.Background(), f.scope, f.work.ID)
	if err != nil || after.Value.Version != claimed.Value.Version || after.Value.CurrentLease == nil || after.Value.PendingDeliveryID != nil {
		t.Fatalf("empty-key submission mutated Task state: value=%+v err=%v", after.Value, err)
	}
	if deliveries := listTaskDeliveries(t, f); len(deliveries) != 0 {
		t.Fatalf("empty-key submission stored %d deliveries", len(deliveries))
	}
	if events := f.store.SnapshotDomainEvents(f.scope); len(events) != beforeEvents {
		t.Fatalf("empty-key submission emitted events: before=%d after=%d", beforeEvents, len(events))
	}
	if revision := readTaskDeliveryRevision(t, f); revision != beforeRevision {
		t.Fatalf("empty-key submission advanced Outcome revision: before=%d after=%d", beforeRevision, revision)
	}
	if receipts := f.store.IdempotencyRecordCount(); receipts != beforeReceipts {
		t.Fatalf("empty-key submission wrote a receipt: before=%d after=%d", beforeReceipts, receipts)
	}
}

func TestConcurrentTaskDeliverySubmissionsWithDifferentKeysCreateOnePendingDelivery(t *testing.T) {
	f := newCredentialLeaseFixture(t)
	_, session := f.addCredentialTree(t,
		"0199f28d-0000-7000-8000-000000000001", "0199f28e-0000-7000-8000-000000000001",
		"delivery-race-root", "delivery-race-session", f.cc.PrincipalID, f.cc.Actor)
	claimed, err := f.service.ClaimWorkItem(f.identityContext(session), freshCommandContext(f.cc, "task-delivery-race-claim"), a.ClaimWorkItemCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: f.work.Version, TTL: d.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatalf("ClaimWorkItem() error = %v", err)
	}
	command := a.SubmitTaskForReviewCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: claimed.Value.Version,
		ClaimID: &claimed.Value.CurrentLease.ClaimID, FencingToken: claimed.Value.CurrentLease.FencingToken,
		LegacyMaterial: ptrTaskDeliveryMaterial(d.TaskDeliveryMaterial{Summary: "One of the competing submissions should be recorded"}),
	}
	serviceTwo, err := a.NewService(f.store, fixedClock{now: f.now}, &sequenceIDs{next: 41000})
	if err != nil {
		t.Fatal(err)
	}
	services := []*a.Service{f.service, serviceTwo}
	keys := []string{"task-delivery-race-submit-one", "task-delivery-race-submit-two"}
	start := make(chan struct{})
	results := make(chan error, len(services))
	var group sync.WaitGroup
	group.Add(len(services))
	for index, service := range services {
		index, service := index, service
		go func() {
			defer group.Done()
			<-start
			_, err := service.SubmitTaskForReview(f.identityContext(session), freshCommandContext(f.cc, keys[index]), command)
			results <- err
		}()
	}
	close(start)
	group.Wait()
	close(results)
	successes, failures := 0, 0
	for err := range results {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("different-key concurrent submissions = %d successes, %d failures; want one each", successes, failures)
	}
	deliveries := listTaskDeliveries(t, f)
	if len(deliveries) != 1 {
		t.Fatalf("different-key race stored %d deliveries, want one", len(deliveries))
	}
	work, err := f.service.GetWorkItem(context.Background(), f.scope, f.work.ID)
	if err != nil || work.Value.PendingDeliveryID == nil || *work.Value.PendingDeliveryID != deliveries[0].ID || work.Value.CurrentLease != nil {
		t.Fatalf("different-key race left an inconsistent pending delivery: value=%+v err=%v", work.Value, err)
	}
}

func TestSubmitTaskForReviewRechecksAuthorityAfterWaitingForOutcomeLock(t *testing.T) {
	t.Run("legacy lease", func(t *testing.T) {
		f := newCredentialLeaseFixture(t)
		_, session := f.addCredentialTree(t,
			"0199f28f-0000-7000-8000-000000000001", "0199f290-0000-7000-8000-000000000002",
			"delivery-wait-legacy-root", "delivery-wait-legacy-session", f.cc.PrincipalID, f.cc.Actor)
		claimed, err := f.service.ClaimWorkItem(f.identityContext(session), freshCommandContext(f.cc, "delivery-wait-legacy-claim"), a.ClaimWorkItemCommand{
			Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: f.work.Version, TTL: d.DefaultLeaseTTL,
		})
		if err != nil {
			t.Fatalf("ClaimWorkItem() error = %v", err)
		}
		lease := claimed.Value.CurrentLease
		command := a.SubmitTaskForReviewCommand{
			Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: claimed.Value.Version,
			ClaimID: &lease.ClaimID, FencingToken: lease.FencingToken,
			LegacyMaterial: ptrTaskDeliveryMaterial(d.TaskDeliveryMaterial{Summary: "Must be rejected after lease expiry"}),
		}
		testSubmitWaitsPastAuthorityExpiry(t, f, session, command, lease.ExpiresAt, d.ErrorCodeLease, nil)
	})

	t.Run("contracts_v1 lease", func(t *testing.T) {
		f := newCredentialLeaseFixture(t)
		markWorkContractEnabled(t, &f)
		_, session := f.addCredentialTree(t,
			"0199f291-0000-7000-8000-000000000002", "0199f292-0000-7000-8000-000000000002",
			"delivery-wait-contract-root", "delivery-wait-contract-session", f.cc.PrincipalID, f.cc.Actor)
		acquired, err := f.service.AcquireWorkContract(f.identityContext(session), freshCommandContext(f.cc, "delivery-wait-contract-acquire"), a.AcquireWorkContractCommand{
			Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: f.work.Version, TTLSeconds: 60,
		})
		if err != nil {
			t.Fatalf("AcquireWorkContract() error = %v", err)
		}
		contract := acquired.Value.Contract
		command := a.SubmitTaskForReviewCommand{
			Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: acquired.Value.WorkItem.Version,
			ContractID: &contract.ID, Authority: &a.ContractAuthority{ExecutionID: contract.ExecutionID, FencingToken: contract.FencingToken, SpecDigest: contract.SpecDigest},
			ExpectedContractVersion: ptrVersion(contract.Version),
			SubmissionMaterial:      &d.WorkResultMaterial{ContractID: contract.ID, WorkItemID: f.work.ID, SpecDigest: contract.SpecDigest, Summary: "Must not close an expired contract"},
		}
		testSubmitWaitsPastAuthorityExpiry(t, f, session, command, contract.ExpiresAt, d.ErrorCodeContractExpired, &contract.ID)
	})
}

func testSubmitWaitsPastAuthorityExpiry(t *testing.T, f credentialLeaseFixture, session ports.Credential, command a.SubmitTaskForReviewCommand, expiry time.Time, expectedCode d.ErrorCode, contractID *d.ID) {
	t.Helper()
	beforeWork, err := f.service.GetWorkItem(context.Background(), f.scope, f.work.ID)
	if err != nil {
		t.Fatal(err)
	}
	beforeEvents := len(f.store.SnapshotDomainEvents(f.scope))
	beforeReceipts := f.store.IdempotencyRecordCount()
	beforeRevision := readTaskDeliveryRevision(t, f)
	var beforeContract d.WorkContract
	if contractID != nil {
		loaded, loadErr := f.service.GetWorkContract(context.Background(), f.scope, *contractID)
		if loadErr != nil {
			t.Fatal(loadErr)
		}
		beforeContract = loaded.Value.Contract
	}

	// Keep a transaction holding the Outcome coordination lock. The instrumented
	// manager signals exactly when the command reaches its transaction begin.
	lockHolder, err := f.store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := lockHolder.Coordination().LockOutcome(context.Background(), f.scope); err != nil {
		_ = lockHolder.Rollback()
		t.Fatal(err)
	}
	released := false
	defer func() {
		if released {
			return
		}
		if rollbackErr := lockHolder.Rollback(); rollbackErr != nil && errorCode(rollbackErr) != d.ErrorCodeInvalidTransition {
			t.Errorf("release held Outcome transaction: %v", rollbackErr)
		}
	}()

	clock := &mutableTaskDeliveryClock{now: f.now}
	manager := &signalingTaskDeliveryTransactionManager{inner: f.store, beginCalled: make(chan struct{}), beginReturned: make(chan struct{})}
	service, err := a.NewService(manager, clock, &sequenceIDs{next: 52000})
	if err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		_, submitErr := service.SubmitTaskForReview(f.identityContext(session), freshCommandContext(f.cc, "task-delivery-wait-expiry-submit"), command)
		result <- submitErr
	}()
	select {
	case <-manager.beginCalled:
	case <-time.After(5 * time.Second):
		t.Fatal("SubmitTaskForReview did not reach transaction begin")
	}
	select {
	case <-manager.beginReturned:
		t.Fatal("SubmitTaskForReview acquired a transaction while the Outcome lock was held")
	case <-time.After(20 * time.Millisecond):
	}
	clock.Set(expiry)
	if err := lockHolder.Rollback(); err != nil {
		t.Fatal(err)
	}
	released = true
	select {
	case submitErr := <-result:
		if errorCode(submitErr) != expectedCode {
			t.Fatalf("submission after authority expiry error = %v, want %s", submitErr, expectedCode)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("SubmitTaskForReview did not finish after releasing the Outcome lock")
	}

	afterWork, err := f.service.GetWorkItem(context.Background(), f.scope, f.work.ID)
	if err != nil || afterWork.Value.Version != beforeWork.Value.Version || afterWork.Value.Lifecycle != beforeWork.Value.Lifecycle || afterWork.Value.PendingDeliveryID != nil || beforeWork.Value.PendingDeliveryID != nil {
		t.Fatalf("expired submission mutated Task state: before=%+v after=%+v err=%v", beforeWork.Value, afterWork.Value, err)
	}
	if beforeWork.Value.CurrentLease != nil && (afterWork.Value.CurrentLease == nil || afterWork.Value.CurrentLease.ClaimID != beforeWork.Value.CurrentLease.ClaimID || afterWork.Value.CurrentLease.FencingToken != beforeWork.Value.CurrentLease.FencingToken) {
		t.Fatalf("expired legacy submission changed its lease: before=%+v after=%+v", beforeWork.Value.CurrentLease, afterWork.Value.CurrentLease)
	}
	if deliveries := listTaskDeliveries(t, f); len(deliveries) != 0 {
		t.Fatalf("expired submission stored %d TaskDeliveries", len(deliveries))
	}
	if events := f.store.SnapshotDomainEvents(f.scope); len(events) != beforeEvents {
		t.Fatalf("expired submission emitted events: before=%d after=%d", beforeEvents, len(events))
	}
	if revision := readTaskDeliveryRevision(t, f); revision != beforeRevision {
		t.Fatalf("expired submission advanced Outcome revision: before=%d after=%d", beforeRevision, revision)
	}
	if receipts := f.store.IdempotencyRecordCount(); receipts != beforeReceipts {
		t.Fatalf("expired submission wrote a receipt: before=%d after=%d", beforeReceipts, receipts)
	}
	if contractID != nil {
		loaded, loadErr := f.service.GetWorkContract(context.Background(), f.scope, *contractID)
		if loadErr != nil || loaded.Value.Contract.Status != d.ContractActive || loaded.Value.Contract.CloseReason != beforeContract.CloseReason || loaded.Value.Contract.Version != beforeContract.Version || loaded.Value.Contract.LeaseVersion != beforeContract.LeaseVersion {
			t.Fatalf("expired submission closed or changed its contract: before=%+v after=%+v err=%v", beforeContract, loaded.Value.Contract, loadErr)
		}
		uow, beginErr := f.store.Begin(context.Background())
		if beginErr != nil {
			t.Fatal(beginErr)
		}
		submissions, listErr := uow.(ports.WorkContractUnitOfWork).WorkContracts().ListSubmissions(context.Background(), f.scope, *contractID, "", 100)
		if rollbackErr := uow.Rollback(); rollbackErr != nil {
			t.Fatal(rollbackErr)
		}
		if listErr != nil || len(submissions) != 0 {
			t.Fatalf("expired submission created WorkSubmissions: values=%+v err=%v", submissions, listErr)
		}
	}
}

type mutableTaskDeliveryClock struct {
	mu  sync.RWMutex
	now time.Time
}

func (c *mutableTaskDeliveryClock) Now() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.now
}

func (c *mutableTaskDeliveryClock) Set(now time.Time) {
	c.mu.Lock()
	c.now = now
	c.mu.Unlock()
}

type signalingTaskDeliveryTransactionManager struct {
	inner         ports.TransactionManager
	beginCalled   chan struct{}
	beginReturned chan struct{}
	once          sync.Once
}

func (m *signalingTaskDeliveryTransactionManager) Begin(ctx context.Context) (ports.UnitOfWork, error) {
	m.once.Do(func() { close(m.beginCalled) })
	uow, err := m.inner.Begin(ctx)
	close(m.beginReturned)
	return uow, err
}

func listTaskDeliveries(t *testing.T, f credentialLeaseFixture) []d.TaskDelivery {
	t.Helper()
	uow, err := f.store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	deliveryUOW, ok := uow.(ports.TaskDeliveryUnitOfWork)
	if !ok {
		t.Fatal("Memory UoW lacks TaskDelivery persistence")
	}
	deliveries, err := deliveryUOW.TaskDeliveries().List(context.Background(), f.scope, ports.TaskDeliveryFilter{WorkItemID: f.work.ID, Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	return deliveries
}

func readTaskDeliveryRevision(t *testing.T, f credentialLeaseFixture) d.OutcomeRevision {
	t.Helper()
	uow, err := f.store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	coordination, err := uow.Coordination().LockOutcome(context.Background(), f.scope)
	if err != nil {
		t.Fatal(err)
	}
	return coordination.Revision
}

func TestSubmitTaskForReviewContractsV1SubmitsAndClosesExecutionAuthority(t *testing.T) {
	f := newCredentialLeaseFixture(t)
	markWorkContractEnabled(t, &f)
	root, session := f.addCredentialTree(t,
		"0199f284-0000-7000-8000-000000000001", "0199f285-0000-7000-8000-000000000001",
		"delivery-contract-root", "delivery-contract-session", f.cc.PrincipalID, f.cc.Actor)
	_, otherSession := f.addCredentialTree(t,
		"0199f284-0000-7000-8000-000000000002", "0199f285-0000-7000-8000-000000000002",
		"delivery-contract-other-root", "delivery-contract-other-session", f.cc.PrincipalID, f.cc.Actor)

	acquireCC := freshCommandContext(f.cc, "task-delivery-contract-acquire")
	acquired, err := f.service.AcquireWorkContract(f.identityContext(session), acquireCC, a.AcquireWorkContractCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: f.work.Version, TTLSeconds: 60,
	})
	if err != nil {
		t.Fatalf("AcquireWorkContract() error = %v", err)
	}
	contract := acquired.Value.Contract
	if contract.CredentialBindingVersion != d.WorkContractCredentialBindingV1 || contract.HolderCredentialID != root.ID || contract.SignedBinding != nil {
		t.Fatalf("unsigned contract credential binding = %+v", contract)
	}
	authority := a.ContractAuthority{ExecutionID: contract.ExecutionID, FencingToken: contract.FencingToken, SpecDigest: contract.SpecDigest}
	command := a.SubmitTaskForReviewCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: acquired.Value.WorkItem.Version,
		ContractID: &contract.ID, Authority: &authority, ExpectedContractVersion: &contract.Version,
		SubmissionMaterial: &d.WorkResultMaterial{
			ContractID: contract.ID, WorkItemID: f.work.ID, SpecDigest: contract.SpecDigest,
			Summary: "Implemented and submitted through the active contracts_v1 authority",
		},
	}

	wrongCredentialCC := freshCommandContext(f.cc, "task-delivery-contract-wrong-owner")
	if _, err := f.service.SubmitTaskForReview(f.identityContext(otherSession), wrongCredentialCC, command); err == nil {
		t.Fatal("different root credential submitted another credential's contracts_v1 authority")
	}
	stale := command
	stale.ExpectedContractVersion = ptrVersion(contract.Version + 1)
	staleCC := freshCommandContext(f.cc, "task-delivery-contract-stale-version")
	if _, err := f.service.SubmitTaskForReview(f.identityContext(session), staleCC, stale); errorCode(err) != d.ErrorCodeVersionConflict {
		t.Fatalf("stale contract version error = %v, want version conflict", err)
	}
	invalid := command
	invalid.SubmissionMaterial = &d.WorkResultMaterial{
		ContractID: contract.ID, WorkItemID: f.work.ID, SpecDigest: "sha256:wrong-spec",
		Summary: "wrong snapshot",
	}
	invalidCC := freshCommandContext(f.cc, "task-delivery-contract-invalid-material")
	if _, err := f.service.SubmitTaskForReview(f.identityContext(session), invalidCC, invalid); err == nil {
		t.Fatal("contracts_v1 delivery accepted material for a different contract snapshot")
	}

	submitCC := freshCommandContext(f.cc, "task-delivery-contract-submit")
	submitted, err := f.service.SubmitTaskForReview(f.identityContext(session), submitCC, command)
	if err != nil {
		t.Fatalf("SubmitTaskForReview(contracts_v1) error = %v", err)
	}
	result := submitted.Value
	if result.Submission == nil || result.Delivery.SubmissionRef == nil || result.Delivery.SubmissionRef.ID != result.Submission.ID || result.Delivery.SubmissionRef.ContractID != contract.ID || result.Delivery.SubmissionRef.Digest != result.Submission.Digest {
		t.Fatalf("delivery does not freeze the exact WorkSubmission: %+v", result.Delivery)
	}
	if result.Contract == nil || result.Contract.Status != d.ContractCompleted || result.Contract.CloseReason != d.ContractV1DeliveryCloseReason || result.WorkItem.CurrentContractID != nil || result.WorkItem.CurrentLease != nil || result.WorkItem.PendingDeliveryID == nil || *result.WorkItem.PendingDeliveryID != result.Delivery.ID || result.WorkItem.Lifecycle != d.WorkItemLifecycleInProgress {
		t.Fatalf("contracts_v1 handoff did not close execution and await review: %+v", result)
	}
	if result.ExpectedContractVersion != contract.Version || result.Contract.Version != contract.Version+2 {
		t.Fatalf("contract version transition: started %d ended %d", contract.Version, result.Contract.Version)
	}

	replay, err := f.service.SubmitTaskForReview(f.identityContext(session), submitCC, command)
	if err != nil || !replay.IdempotentReplay || replay.Value.Delivery.ID != result.Delivery.ID {
		t.Fatalf("contracts_v1 receipt replay = (%v, %v)", replay.IdempotentReplay, err)
	}
	if _, err := f.service.SubmitTaskForReview(f.identityContext(otherSession), submitCC, command); errorCode(err) != d.ErrorCodeIdempotencyConflict {
		t.Fatalf("contracts_v1 receipt disclosure error = %v", err)
	}
}

func TestSubmitTaskForReviewContractsV1RollsBackSubmissionWhenDeliveryCannotBeCreated(t *testing.T) {
	f := newCredentialLeaseFixture(t)
	markWorkContractEnabled(t, &f)
	_, session := f.addCredentialTree(t,
		"0199f286-0000-7000-8000-000000000001", "0199f287-0000-7000-8000-000000000001",
		"delivery-rollback-root", "delivery-rollback-session", f.cc.PrincipalID, f.cc.Actor)
	acquired, err := f.service.AcquireWorkContract(f.identityContext(session), freshCommandContext(f.cc, "task-delivery-rollback-acquire"), a.AcquireWorkContractCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: f.work.Version, TTLSeconds: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	contract := acquired.Value.Contract
	submissionID := id("0199f288-0000-7000-8000-000000000001")
	service, err := a.NewService(f.store, fixedClock{now: f.now}, &taskDeliveryFailingIDs{submissionID: submissionID})
	if err != nil {
		t.Fatal(err)
	}
	command := a.SubmitTaskForReviewCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: acquired.Value.WorkItem.Version,
		ContractID: &contract.ID, Authority: &a.ContractAuthority{ExecutionID: contract.ExecutionID, FencingToken: contract.FencingToken, SpecDigest: contract.SpecDigest},
		ExpectedContractVersion: ptrVersion(contract.Version),
		SubmissionMaterial:      &d.WorkResultMaterial{ContractID: contract.ID, WorkItemID: f.work.ID, SpecDigest: contract.SpecDigest, Summary: "must roll back as a unit"},
	}
	_, err = service.SubmitTaskForReview(f.identityContext(session), freshCommandContext(f.cc, "task-delivery-forced-rollback"), command)
	if err == nil || !strings.Contains(err.Error(), "configured TaskDelivery ID failure") {
		t.Fatalf("expected failure while allocating the delivery after its submission, got %v", err)
	}

	work, err := service.GetWorkItem(context.Background(), f.scope, f.work.ID)
	if err != nil || work.Value.CurrentContractID == nil || *work.Value.CurrentContractID != contract.ID || work.Value.PendingDeliveryID != nil || work.Value.CurrentLease != nil {
		t.Fatalf("failed handoff left a partially updated WorkItem: value=%+v err=%v", work.Value, err)
	}
	loadedContract, err := service.GetWorkContract(context.Background(), f.scope, contract.ID)
	if err != nil || loadedContract.Value.Contract.LatestSubmissionID != nil || loadedContract.Value.Contract.Status != d.ContractActive {
		t.Fatalf("failed handoff left a submitted/closed contract: value=%+v err=%v", loadedContract.Value.Contract, err)
	}
	tx, err := f.store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	repo := tx.(ports.WorkContractUnitOfWork).WorkContracts()
	if _, err := repo.GetSubmission(context.Background(), f.scope, submissionID); errorCode(err) != d.ErrorCodeNotFound {
		t.Fatalf("rolled-back WorkSubmission lookup error = %v", err)
	}
}

func TestSubmitTaskForReviewIsRejectedAfterSignedProtocolCutover(t *testing.T) {
	f := newCredentialLeaseFixture(t)
	markWorkContractEnabled(t, &f)
	_, session := f.addCredentialTree(t,
		"0199f289-0000-7000-8000-000000000001", "0199f28a-0000-7000-8000-000000000001",
		"delivery-signed-root", "delivery-signed-session", f.cc.PrincipalID, f.cc.Actor)
	acquired, err := f.service.AcquireWorkContract(f.identityContext(session), freshCommandContext(f.cc, "task-delivery-signed-acquire"), a.AcquireWorkContractCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: f.work.Version, TTLSeconds: 60,
	})
	if err != nil {
		t.Fatal(err)
	}
	contract := acquired.Value.Contract

	ctx := context.Background()
	tx, err := f.store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	protocolRepo, ok := tx.(ports.WorkProtocolUnitOfWork)
	if !ok {
		t.Fatal("memory UoW lacks Namespace work protocol persistence")
	}
	protocol := d.DefaultWorkProtocol(f.scope.NamespaceID)
	protocol.Version = 2
	protocol.Phase = d.WorkProtocolSigned
	protocol.WriterEpoch = 2
	protocol.WritersDrained = true
	protocol.UpdatedAt = f.now
	protocol.UpdatedBy = f.cc.PrincipalID
	protocol.Reason = "exercise unsigned delivery gate"
	if err := protocolRepo.WorkProtocol().Save(ctx, protocol, 1); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	command := a.SubmitTaskForReviewCommand{
		Scope: f.scope, WorkItemID: f.work.ID, ExpectedWorkItemVersion: acquired.Value.WorkItem.Version,
		ContractID:              &contract.ID,
		Authority:               &a.ContractAuthority{ExecutionID: contract.ExecutionID, FencingToken: contract.FencingToken, SpecDigest: contract.SpecDigest},
		ExpectedContractVersion: ptrVersion(contract.Version),
		SubmissionMaterial:      &d.WorkResultMaterial{ContractID: contract.ID, WorkItemID: f.work.ID, SpecDigest: contract.SpecDigest, Summary: "unsigned submission after signed cutover"},
	}
	_, err = f.service.SubmitTaskForReview(f.identityContext(session), freshCommandContext(f.cc, "task-delivery-signed-rejected"), command)
	if errorCode(err) != d.ErrorCodeSignedProtocolRequired {
		t.Fatalf("unsigned delivery after signed cutover error = %v, want signed-protocol-required", err)
	}
}

type taskDeliveryFailingIDs struct {
	submissionID d.ID
	used         bool
}

func (g *taskDeliveryFailingIDs) NewID() (d.ID, error) {
	if !g.used {
		g.used = true
		return g.submissionID, nil
	}
	return "", errors.New("configured TaskDelivery ID failure")
}

func ptrTaskDeliveryMaterial(value d.TaskDeliveryMaterial) *d.TaskDeliveryMaterial { return &value }

func ptrVersion(value d.Version) *d.Version { return &value }

func freshCommandContext(cc d.CommandContext, key string) d.CommandContext {
	cc.IdempotencyKey = key
	cc.CommandID = d.MustParseID(fmt.Sprintf("0199f280-0000-7000-8000-%012x", commandOrdinal(key)))
	return cc
}

func commandOrdinal(key string) uint64 {
	var value uint64 = 1
	for _, r := range key {
		value = value*33 + uint64(r)
	}
	return value & 0xffffffffffff
}

func errorCode(err error) d.ErrorCode {
	code, _ := d.ErrorCodeOf(err)
	return code
}

func readTaskDelivery(t *testing.T, store *memory.Store, scope d.Scope, deliveryID d.ID) (d.TaskDelivery, error) {
	t.Helper()
	tx, err := store.Begin(context.Background())
	if err != nil {
		return d.TaskDelivery{}, err
	}
	defer tx.Rollback()
	repo, ok := tx.(ports.TaskDeliveryUnitOfWork)
	if !ok {
		return d.TaskDelivery{}, fmt.Errorf("memory UoW lacks TaskDelivery persistence")
	}
	return repo.TaskDeliveries().Get(context.Background(), scope, deliveryID)
}
