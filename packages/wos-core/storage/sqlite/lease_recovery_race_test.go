package sqlite

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func TestSQLiteProtectedReceiptSurvivesRestartAndRejectsDifferentRoot(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "lease-receipt.db")
	store := openTestStore(t, path)
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	service, scope, work, cc := createSQLiteTodoWork(t, store, now, "0199ec10")
	rootA := putSQLiteReceiptCredential(t, store, scope, cc, "0199ec11-0000-7000-8000-000000000001", "e25-root-a")
	rootB := putSQLiteReceiptCredential(t, store, scope, cc, "0199ec11-0000-7000-8000-000000000002", "e25-root-b")
	cc.IdempotencyKey = "e25-sqlite-claim-receipt-01"
	command := application.ClaimWorkItemCommand{Scope: scope, WorkItemID: work.ID, ExpectedVersion: work.Version, TTL: domain.DefaultLeaseTTL}
	first, err := service.ClaimWorkItem(sqliteReceiptIdentityContext(ctx, rootA), cc, command)
	if err != nil {
		t.Fatal(err)
	}
	var receiptCredentialID string
	if err := store.db.QueryRowContext(ctx, `SELECT authority_credential_id FROM idempotency_records WHERE namespace_id=? AND principal_id=? AND command_name='ClaimWorkItem' AND idempotency_key=?`, scope.NamespaceID.String(), cc.PrincipalID, cc.IdempotencyKey).Scan(&receiptCredentialID); err != nil || receiptCredentialID != rootA.ID.String() {
		t.Fatalf("receipt credential binding was not persisted: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store = openTestStore(t, path)
	service, err = application.NewService(store, sqliteFixedClock{now: now}, &sqliteSequenceIDs{prefix: "0199ec12", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	replayed, err := service.ClaimWorkItem(sqliteReceiptIdentityContext(ctx, rootA), cc, command)
	if err != nil || !replayed.IdempotentReplay || replayed.Value.CurrentLease == nil || replayed.Value.CurrentLease.ClaimID != first.Value.CurrentLease.ClaimID || replayed.Value.Version != first.Value.Version {
		t.Fatalf("same-root retry after restart = %+v err=%v", replayed, err)
	}
	if _, err := service.ClaimWorkItem(sqliteReceiptIdentityContext(ctx, rootB), cc, command); e25StorageErrorCode(err) != domain.ErrorCodeIdempotencyConflict {
		t.Fatalf("different root received stored claim result: code=%s err=%v", e25StorageErrorCode(err), err)
	}
	current, err := service.GetWorkItem(ctx, scope, work.ID)
	if err != nil || current.Value.Version != first.Value.Version || current.Value.CurrentLease == nil || current.Value.CurrentLease.ClaimID != first.Value.CurrentLease.ClaimID {
		t.Fatalf("replay mutated recovered lease: item=%+v err=%v", current, err)
	}
}

func TestSQLiteConcurrentExpiredLeaseReclaimHasOneFenceWinner(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "lease-reclaim-race.db"))
	claimedAt := time.Date(2026, 10, 21, 13, 0, 0, 0, time.UTC)
	outcome, work := createSQLiteClaimedWork(t, store, claimedAt, "0199ec20", testID("0199ec21-0000-7000-8000-000000000001"))
	oldLease := *work.CurrentLease
	reclaimAt := oldLease.ExpiresAt
	services := make([]*application.Service, 2)
	for i, prefix := range []string{"0199ec22", "0199ec23"} {
		service, err := application.NewService(store, sqliteFixedClock{now: reclaimAt}, &sqliteSequenceIDs{prefix: prefix, next: 1})
		if err != nil {
			t.Fatal(err)
		}
		services[i] = service
	}
	type result struct {
		work domain.WorkItem
		err  error
	}
	results := make(chan result, len(services))
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i, service := range services {
		wg.Add(1)
		go func(i int, service *application.Service) {
			defer wg.Done()
			<-start
			cc := sqliteLeaseCommandContext(fmt.Sprintf("recovery-worker-%c", 'a'+i), testID(fmt.Sprintf("0199ec24-0000-7000-8000-%012x", i+1)).String())
			cc.IdempotencyKey = fmt.Sprintf("e25-concurrent-reclaim-%02d", i+1)
			reclaimed, err := service.ReclaimWorkItem(ctx, cc, application.ReclaimWorkItemCommand{Scope: outcome.Scope(), WorkItemID: work.ID, ExpectedVersion: work.Version, TTL: domain.DefaultLeaseTTL})
			results <- result{work: reclaimed.Value, err: err}
		}(i, service)
	}
	close(start)
	wg.Wait()
	close(results)
	var winner *domain.WorkItem
	var successes int
	for result := range results {
		if result.err == nil {
			successes++
			copy := result.work
			winner = &copy
		}
	}
	if successes != 1 || winner == nil {
		t.Fatalf("concurrent reclaim successes = %d, want exactly one", successes)
	}
	if winner.CurrentLease == nil || winner.CurrentLease.ClaimID == oldLease.ClaimID || winner.CurrentLease.FencingToken != oldLease.FencingToken+1 || winner.Version != work.Version+1 {
		t.Fatalf("winning reclaim did not advance one fence/version: %+v", winner)
	}
	current, err := services[0].GetWorkItem(ctx, outcome.Scope(), work.ID)
	if err != nil || current.Value.CurrentLease == nil || current.Value.CurrentLease.ClaimID != winner.CurrentLease.ClaimID || current.Value.Version != winner.Version {
		t.Fatalf("persisted reclaim differs from winner: item=%+v err=%v", current, err)
	}
}

type e25RenewBarrier struct {
	security application.SecurityService
	entered  chan struct{}
	resume   chan struct{}
}

func (a e25RenewBarrier) Authorize(ctx context.Context, request ports.AuthorizationRequest) error {
	if err := a.security.Authorize(ctx, request); err != nil {
		return err
	}
	close(a.entered)
	<-a.resume
	return nil
}

func TestSQLiteRenewRechecksCredentialAfterConcurrentRevocation(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "lease-revoke-renew.db"))
	now := time.Date(2026, 10, 21, 14, 0, 0, 0, time.UTC)
	_, scope, work, cc := createSQLiteTodoWork(t, store, now, "0199ec30")
	if err := store.PutGrant(ctx, ports.NamespaceGrant{NamespaceID: scope.NamespaceID, PrincipalID: cc.PrincipalID, Permissions: []ports.Permission{ports.PermissionTaskClaim, ports.PermissionTaskRenew}}); err != nil {
		t.Fatal(err)
	}
	credential := putSQLiteReceiptCredential(t, store, scope, cc, "0199ec31-0000-7000-8000-000000000001", "e25-revoke-renew")
	clock := sqliteFixedClock{now: now}
	security := application.SecurityService{Store: store, Clock: clock, IDs: &sqliteSequenceIDs{prefix: "0199ec32", next: 1}}
	service, err := application.NewAuthorizedService(store, clock, &sqliteSequenceIDs{prefix: "0199ec33", next: 1}, security)
	if err != nil {
		t.Fatal(err)
	}
	identityContext := sqliteReceiptIdentityContext(ctx, credential)
	claimed, err := service.ClaimWorkItem(identityContext, cc, application.ClaimWorkItemCommand{Scope: scope, WorkItemID: work.ID, ExpectedVersion: work.Version, TTL: domain.DefaultLeaseTTL})
	if err != nil {
		t.Fatal(err)
	}
	lease := claimed.Value.CurrentLease
	barrier := e25RenewBarrier{security: security, entered: make(chan struct{}), resume: make(chan struct{})}
	barrierService, err := application.NewAuthorizedService(store, clock, &sqliteSequenceIDs{prefix: "0199ec34", next: 1}, barrier)
	if err != nil {
		t.Fatal(err)
	}
	cc.IdempotencyKey = "e25-revoke-during-renew-01"
	result := make(chan error, 1)
	go func() {
		_, renewErr := barrierService.RenewWorkItemLease(identityContext, cc, application.RenewWorkItemLeaseCommand{Scope: scope, WorkItemID: work.ID, ExpectedVersion: claimed.Value.Version, ClaimID: lease.ClaimID, FencingToken: lease.FencingToken, TTL: domain.DefaultLeaseTTL})
		result <- renewErr
	}()
	<-barrier.entered
	if err := store.RevokeCredential(ctx, credential.ID); err != nil {
		t.Fatal(err)
	}
	close(barrier.resume)
	if err := <-result; e25StorageErrorCode(err) != domain.ErrorCodeForbidden {
		t.Fatalf("renew after revocation preflight race: code=%s err=%v", e25StorageErrorCode(err), err)
	}
	uow, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	current, err := uow.WorkItems().Get(ctx, scope, work.ID)
	_ = uow.Rollback()
	if err != nil || current.Version != claimed.Value.Version || current.CurrentLease == nil || !current.CurrentLease.ExpiresAt.Equal(lease.ExpiresAt) {
		t.Fatalf("revoked credential extended or changed lease: item=%+v err=%v", current, err)
	}
}

func createSQLiteTodoWork(t *testing.T, store *Store, now time.Time, prefix string) (*application.Service, domain.Scope, domain.WorkItem, domain.CommandContext) {
	t.Helper()
	ctx := context.Background()
	service, err := application.NewService(store, sqliteFixedClock{now: now}, &sqliteSequenceIDs{prefix: prefix, next: 1})
	if err != nil {
		t.Fatal(err)
	}
	cc := sqliteLeaseCommandContext("e25-worker", testID(prefix+"-0000-7000-8000-000000000101").String())
	namespace := testID(prefix + "-0000-7000-8000-000000000001")
	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{NamespaceID: namespace, Title: "E25 lease", DesiredState: "Credential-bound lease receipts", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = service.AddCriterion(ctx, cc, application.AddCriterionCommand{Owner: created.Value.Ref(), ExpectedVersion: created.Value.Version, Title: "Receipt and execution remain consistent", Required: true, VerificationMode: domain.VerificationModeAttestation}); err != nil {
		t.Fatal(err)
	}
	activated, err := service.ActivateOutcome(ctx, cc, application.ActivateOutcomeCommand{Scope: created.Value.Scope(), ExpectedVersion: created.Value.Version + 1})
	if err != nil {
		t.Fatal(err)
	}
	work, err := service.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{Scope: activated.Value.Scope(), ObjectiveID: sqliteObjectiveForWorkFixture(t, service, activated.Value.Scope()), Title: "Receipt-safe task", Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleTodo})
	if err != nil {
		t.Fatal(err)
	}
	return service, activated.Value.Scope(), work.Value, cc
}

func putSQLiteReceiptCredential(t *testing.T, store *Store, scope domain.Scope, cc domain.CommandContext, id, digestSuffix string) ports.Credential {
	t.Helper()
	credential := ports.Credential{ID: testID(id), NamespaceID: scope.NamespaceID, PrincipalID: cc.PrincipalID, Actor: cc.Actor, Digest: "e25-token-" + digestSuffix, ExpiresAt: time.Date(2027, 10, 21, 12, 0, 0, 0, time.UTC)}
	if err := store.PutCredential(context.Background(), credential); err != nil {
		t.Fatal(err)
	}
	return credential
}

func sqliteReceiptIdentityContext(ctx context.Context, credential ports.Credential) context.Context {
	return application.WithIdentity(ctx, application.Identity{CredentialID: credential.ID, PrincipalID: credential.PrincipalID, NamespaceID: credential.NamespaceID, CredentialDigest: credential.Digest, Actor: credential.Actor})
}

func e25StorageErrorCode(err error) domain.ErrorCode {
	code, _ := domain.ErrorCodeOf(err)
	return code
}
