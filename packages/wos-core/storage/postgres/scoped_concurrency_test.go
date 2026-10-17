package postgres

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func TestSignedCoordinationHasIndependentPrincipalLanes(t *testing.T) {
	requirePostgres(t)
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "quota-lanes.db"))
	ns := testID("0199f383-0000-7000-8000-000000000001")
	if err := store.PutNamespace(ctx, ports.Namespace{ID: ns, Name: "Lanes"}); err != nil {
		t.Fatal(err)
	}
	first, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer first.Rollback()
	if _, err := first.(ports.WorkProtocolUnitOfWork).WorkProtocol().Lock(ctx, ns, false); err != nil {
		t.Fatal(err)
	}
	if err := first.(ports.SignedQuotaUnitOfWork).LockSignedQuota(ctx, ns, "one", "work", false); err != nil {
		t.Fatal(err)
	}
	second, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer second.Rollback()
	bounded, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if _, err := second.(ports.WorkProtocolUnitOfWork).WorkProtocol().Lock(bounded, ns, false); err != nil {
		t.Fatal("shared protocol barrier serialized independent commands", err)
	}
	if err := second.(ports.SignedQuotaUnitOfWork).LockSignedQuota(bounded, ns, "two", "work", false); err != nil {
		t.Fatal("independent Principal quota stalled", err)
	}
	blocked, cancelBlocked := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancelBlocked()
	third, err := store.Begin(blocked)
	if err != nil {
		t.Fatal(err)
	}
	defer third.Rollback()
	if err := third.(ports.SignedQuotaUnitOfWork).LockSignedQuota(blocked, ns, "one", "work", false); err == nil {
		t.Fatal("same Principal bypassed its acquisition lane")
	}
}

func TestSignedAcquisitionProgressesBesideIndependentOutcomeGuard(t *testing.T) {
	requirePostgres(t)
	f := newResolutionFixture(t, "sql")
	ctx := context.Background()
	ns := d.ID(f.request.NamespaceID)
	now := time.Now().UTC()
	u, err := f.store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := d.NewOutcome(testID("0199f386-0000-7000-8000-000000000001"), ns, "Independent Outcome", "", "Parallel signed work", d.PriorityNormal, now)
	if err != nil {
		t.Fatal(err)
	}
	outcome.Lifecycle = d.OutcomeLifecycleActive
	if err := u.Outcomes().Insert(ctx, outcome); err != nil {
		t.Fatal(err)
	}
	work, err := d.NewWorkItem(testID("0199f386-0000-7000-8000-000000000002"), outcome.Scope(), "Independent Task", "", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
	if err != nil {
		t.Fatal(err)
	}
	work.ContractsEnabled = true
	if err := u.WorkItems().Insert(ctx, work); err != nil {
		t.Fatal(err)
	}
	if err := u.Commit(); err != nil {
		t.Fatal(err)
	}
	// Hold the protocol/security barrier and another Outcome's mutation guard,
	// exactly the locks a simultaneous unrelated command must be able to hold.
	held, err := f.store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Rollback()
	if _, err := held.(ports.WorkProtocolUnitOfWork).WorkProtocol().Lock(ctx, ns, false); err != nil {
		t.Fatal(err)
	}
	if err := held.(ports.SignedQuotaUnitOfWork).LockSignedQuota(ctx, ns, "independent-principal", "work", false); err != nil {
		t.Fatal(err)
	}
	if _, err := held.Coordination().LockOutcome(ctx, d.Scope{NamespaceID: ns, OutcomeID: d.ID(f.request.OutcomeID)}); err != nil {
		t.Fatal(err)
	}
	bounded, cancel := context.WithTimeout(f.ctx, 2*time.Second)
	defer cancel()
	cc := f.cc
	cc.CommandID = testID("0199f386-0000-7000-8000-000000000003")
	cc.IdempotencyKey = "independent-outcome-acquisition"
	result, err := f.service.AcquireSignedWorkContract(bounded, cc, a.AcquireSignedWorkContractCommand{Scope: outcome.Scope(), WorkItemID: work.ID, ExpectedWorkItemVersion: work.Version, SignerKeyID: d.ID(f.request.SignerKeyID), TTLSeconds: 300})
	if err != nil {
		t.Fatalf("independent signed acquisition stalled behind another Outcome: %v; cause: %v", err, errors.Unwrap(err))
	}
	if result.Value.Contract.WorkItemID != work.ID {
		t.Fatal("wrong acquired work")
	}
	security := a.SecurityService{Store: f.store, Clock: sqliteFixedClock{time.Now().UTC()}, IDs: &loadIDs{}}
	if _, err := security.SigningIdentity(bounded, nil); err != nil {
		t.Fatal("identity diagnostic upgraded shared Namespace barrier", err)
	}
}

func TestReadSnapshotSecurityBarrierOrdersCredentialRevocation(t *testing.T) {
	requirePostgres(t)
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "snapshot-revoke.db"))
	now := time.Now().UTC()
	ns := testID("0199f384-0000-7000-8000-000000000001")
	actor := d.ActorRef{Kind: d.ActorKindHuman, Provider: "test", ID: "reader"}
	credential := ports.Credential{NamespaceID: ns, ID: testID("0199f384-0000-7000-8000-000000000002"), PrincipalID: "reader", Actor: actor, Digest: a.TokenDigest("snapshot-reader-token-at-least-32-bytes"), ExpiresAt: now.Add(time.Hour)}
	if err := store.BootstrapSecurity(ctx, ports.Namespace{ID: ns, Name: "Revocation"}, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "reader", Permissions: []ports.Permission{ports.PermissionStateRead}}, credential); err != nil {
		t.Fatal(err)
	}
	reader, err := store.BeginReadSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Rollback()
	request := ports.AccessSnapshotRequest{Authorization: ports.AuthorizationRequest{NamespaceID: ns, PrincipalID: "reader", Permission: ports.PermissionStateRead}, CredentialDigest: credential.Digest, Actor: actor, Now: now}
	if err := reader.(ports.AccessSnapshotUnitOfWork).AuthorizeAccessSnapshot(ctx, request); err != nil {
		t.Fatal(err)
	}
	revoked := make(chan error, 1)
	go func() { revoked <- store.RevokeCredential(ctx, credential.ID) }()
	select {
	case err := <-revoked:
		t.Fatal("revocation overtook an authorized snapshot", err)
	case <-time.After(50 * time.Millisecond):
	}
	reader.Rollback()
	select {
	case err := <-revoked:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("revocation remained blocked after snapshot closed")
	}
	fresh, err := store.BeginReadSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer fresh.Rollback()
	if err := fresh.(ports.AccessSnapshotUnitOfWork).AuthorizeAccessSnapshot(ctx, request); err == nil {
		t.Fatal("new snapshot retained revoked authority")
	}
}

func TestReadSnapshotCannotAuthorizeFromSnapshotPredatingRevocation(t *testing.T) {
	requirePostgres(t)
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "stale-snapshot-revoke.db"))
	now := time.Now().UTC()
	ns := testID("0199f385-0000-7000-8000-000000000001")
	actor := d.ActorRef{Kind: d.ActorKindHuman, Provider: "test", ID: "reader"}
	credential := ports.Credential{NamespaceID: ns, ID: testID("0199f385-0000-7000-8000-000000000002"), PrincipalID: "reader", Actor: actor, Digest: a.TokenDigest("stale-snapshot-reader-token-at-least-32"), ExpiresAt: now.Add(time.Hour)}
	if err := store.BootstrapSecurity(ctx, ports.Namespace{ID: ns, Name: "Revocation"}, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "reader", Permissions: []ports.Permission{ports.PermissionStateRead}}, credential); err != nil {
		t.Fatal(err)
	}
	reader, err := store.BeginReadSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Rollback()
	// Establish the MVCC snapshot without acquiring a credential row barrier.
	var count int
	if err := reader.(*unitOfWork).tx.QueryRowContext(ctx, `SELECT count(*) FROM credentials`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if err := store.RevokeCredential(ctx, credential.ID); err != nil {
		t.Fatal(err)
	}
	request := ports.AccessSnapshotRequest{Authorization: ports.AuthorizationRequest{NamespaceID: ns, PrincipalID: "reader", Permission: ports.PermissionStateRead}, CredentialDigest: credential.Digest, Actor: actor, Now: now}
	if err := reader.(ports.AccessSnapshotUnitOfWork).AuthorizeAccessSnapshot(ctx, request); err == nil {
		t.Fatal("stale MVCC snapshot authorized a credential revoked before its security barrier")
	} else if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodeTransactionConflict {
		t.Fatal("stale security snapshot hid its transaction conflict", err)
	}
}
