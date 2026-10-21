package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func TestProtectedLeaseReceiptsBindReplayToCanonicalRootCredential(t *testing.T) {
	tests := []struct {
		name string
		run  func(t *testing.T, f credentialLeaseFixture, rootA, rootB, sessionA, sessionB ports.Credential)
	}{
		{
			name: "claim",
			run: func(t *testing.T, f credentialLeaseFixture, rootA, rootB, sessionA, _ ports.Credential) {
				cc := f.cc
				cc.IdempotencyKey = "e25-claim-replay-0001"
				cmd := application.ClaimWorkItemCommand{Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: f.work.Version, TTL: domain.DefaultLeaseTTL}
				first, err := f.service.ClaimWorkItem(f.identityContext(rootA), cc, cmd)
				if err != nil {
					t.Fatal(err)
				}
				replay, replayErr := f.service.ClaimWorkItem(f.identityContext(rootA), cc, cmd)
				assertLeaseReceiptReplay(t, replay, replayErr, first.Value)
				sessionReplay, sessionErr := f.service.ClaimWorkItem(f.identityContext(sessionA), cc, cmd)
				assertLeaseReceiptReplay(t, sessionReplay, sessionErr, first.Value)
				_, err = f.service.ClaimWorkItem(f.identityContext(rootB), cc, cmd)
				assertReceiptCredentialConflict(t, err)
			},
		},
		{
			name: "renew",
			run: func(t *testing.T, f credentialLeaseFixture, rootA, rootB, sessionA, _ ports.Credential) {
				claimed, err := f.service.ClaimWorkItem(f.identityContext(rootA), f.cc, application.ClaimWorkItemCommand{Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: f.work.Version, TTL: domain.DefaultLeaseTTL})
				if err != nil {
					t.Fatal(err)
				}
				cc := f.cc
				cc.IdempotencyKey = "e25-renew-replay-0001"
				lease := claimed.Value.CurrentLease
				cmd := application.RenewWorkItemLeaseCommand{Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: claimed.Value.Version, ClaimID: lease.ClaimID, FencingToken: lease.FencingToken, TTL: domain.DefaultLeaseTTL}
				first, err := f.service.RenewWorkItemLease(f.identityContext(rootA), cc, cmd)
				if err != nil {
					t.Fatal(err)
				}
				replay, replayErr := f.service.RenewWorkItemLease(f.identityContext(rootA), cc, cmd)
				assertLeaseReceiptReplay(t, replay, replayErr, first.Value)
				sessionReplay, sessionErr := f.service.RenewWorkItemLease(f.identityContext(sessionA), cc, cmd)
				assertLeaseReceiptReplay(t, sessionReplay, sessionErr, first.Value)
				_, err = f.service.RenewWorkItemLease(f.identityContext(rootB), cc, cmd)
				assertReceiptCredentialConflict(t, err)
			},
		},
		{
			name: "reclaim",
			run: func(t *testing.T, f credentialLeaseFixture, rootA, rootB, _, sessionB ports.Credential) {
				claimed, err := f.service.ClaimWorkItem(f.identityContext(rootA), f.cc, application.ClaimWorkItemCommand{Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: f.work.Version, TTL: domain.MinLeaseTTL})
				if err != nil {
					t.Fatal(err)
				}
				f.now = claimed.Value.CurrentLease.ExpiresAt
				f.service, err = application.NewService(f.store, fixedClock{now: f.now}, &sequenceIDs{next: 9600})
				if err != nil {
					t.Fatal(err)
				}
				cc := f.cc
				cc.IdempotencyKey = "e25-reclaim-replay-001"
				cmd := application.ReclaimWorkItemCommand{Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: claimed.Value.Version, TTL: domain.DefaultLeaseTTL}
				first, err := f.service.ReclaimWorkItem(f.identityContext(rootB), cc, cmd)
				if err != nil {
					t.Fatal(err)
				}
				replay, replayErr := f.service.ReclaimWorkItem(f.identityContext(rootB), cc, cmd)
				assertLeaseReceiptReplay(t, replay, replayErr, first.Value)
				sessionReplay, sessionErr := f.service.ReclaimWorkItem(f.identityContext(sessionB), cc, cmd)
				assertLeaseReceiptReplay(t, sessionReplay, sessionErr, first.Value)
				_, err = f.service.ReclaimWorkItem(f.identityContext(rootA), cc, cmd)
				assertReceiptCredentialConflict(t, err)
			},
		},
		{
			name: "release",
			run: func(t *testing.T, f credentialLeaseFixture, rootA, rootB, sessionA, _ ports.Credential) {
				claimed, err := f.service.ClaimWorkItem(f.identityContext(rootA), f.cc, application.ClaimWorkItemCommand{Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: f.work.Version, TTL: domain.DefaultLeaseTTL})
				if err != nil {
					t.Fatal(err)
				}
				lease := claimed.Value.CurrentLease
				cc := f.cc
				cc.IdempotencyKey = "e25-release-replay-001"
				cmd := application.ReleaseWorkItemCommand{Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: claimed.Value.Version, ClaimID: lease.ClaimID, FencingToken: lease.FencingToken}
				first, err := f.service.ReleaseWorkItem(f.identityContext(rootA), cc, cmd)
				if err != nil {
					t.Fatal(err)
				}
				replay, replayErr := f.service.ReleaseWorkItem(f.identityContext(rootA), cc, cmd)
				assertLeaseReceiptReplay(t, replay, replayErr, first.Value)
				sessionReplay, sessionErr := f.service.ReleaseWorkItem(f.identityContext(sessionA), cc, cmd)
				assertLeaseReceiptReplay(t, sessionReplay, sessionErr, first.Value)
				_, err = f.service.ReleaseWorkItem(f.identityContext(rootB), cc, cmd)
				assertReceiptCredentialConflict(t, err)
			},
		},
		{
			name: "complete",
			run: func(t *testing.T, f credentialLeaseFixture, rootA, rootB, sessionA, _ ports.Credential) {
				claimed, err := f.service.ClaimWorkItem(f.identityContext(rootA), f.cc, application.ClaimWorkItemCommand{Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: f.work.Version, TTL: domain.DefaultLeaseTTL})
				if err != nil {
					t.Fatal(err)
				}
				lease := claimed.Value.CurrentLease
				cc := f.cc
				cc.IdempotencyKey = "e25-complete-replay-001"
				cmd := application.CompleteWorkItemCommand{Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: claimed.Value.Version, ClaimID: lease.ClaimID, FencingToken: lease.FencingToken, ResultSummary: "The task is complete", Reason: "The requested work and result are recorded"}
				first, err := f.service.CompleteWorkItem(f.identityContext(rootA), cc, cmd)
				if err != nil {
					t.Fatal(err)
				}
				replay, replayErr := f.service.CompleteWorkItem(f.identityContext(rootA), cc, cmd)
				assertLeaseReceiptReplay(t, replay, replayErr, first.Value)
				sessionReplay, sessionErr := f.service.CompleteWorkItem(f.identityContext(sessionA), cc, cmd)
				assertLeaseReceiptReplay(t, sessionReplay, sessionErr, first.Value)
				_, err = f.service.CompleteWorkItem(f.identityContext(rootB), cc, cmd)
				assertReceiptCredentialConflict(t, err)
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newCredentialLeaseFixture(t)
			rootA := f.addRoot(t, "0199f261-0000-7000-8000-000000000080", "root-token-e25-a", f.cc.PrincipalID, f.cc.Actor)
			rootB := f.addRoot(t, "0199f261-0000-7000-8000-000000000081", "root-token-e25-b", f.cc.PrincipalID, f.cc.Actor)
			sessionA := f.addSession(t, "0199f262-0000-7000-8000-000000000080", "session-e25-a", rootA, f.cc.PrincipalID, f.cc.Actor)
			sessionB := f.addSession(t, "0199f262-0000-7000-8000-000000000081", "session-e25-b", rootB, f.cc.PrincipalID, f.cc.Actor)
			tc.run(t, f, rootA, rootB, sessionA, sessionB)
		})
	}
}

func TestHistoricalUnboundLeaseReceiptFailsClosedForCredentialsAndAllowsTrustedEmbeddedReplay(t *testing.T) {
	f := newCredentialLeaseFixture(t)
	root := f.addRoot(t, "0199f261-0000-7000-8000-000000000082", "root-token-e25-historical", f.cc.PrincipalID, f.cc.Actor)
	cc := f.cc
	cc.IdempotencyKey = "e25-historical-claim-0001"
	cmd := application.ClaimWorkItemCommand{Scope: f.scope, WorkItemID: f.work.ID, ExpectedVersion: f.work.Version, TTL: domain.DefaultLeaseTTL}

	// A trusted embedded caller represents a pre-E25 receipt: it has no root
	// credential to record, and its request fingerprint/key remain unchanged.
	original, err := f.service.ClaimWorkItem(context.Background(), cc, cmd)
	if err != nil {
		t.Fatalf("embedded ClaimWorkItem() error = %v", err)
	}
	_, err = f.service.ClaimWorkItem(f.identityContext(root), cc, cmd)
	assertReceiptCredentialConflict(t, err)

	replay, err := f.service.ClaimWorkItem(context.Background(), cc, cmd)
	assertLeaseReceiptReplay(t, replay, err, original.Value)
}

func TestContractsV1ReceiptsBindHolderAndAdministrativeIssuerRoots(t *testing.T) {
	ctx := context.Background()
	service, store, clock, work := contractServiceFixture(t)
	if err := store.PutNamespace(ctx, ports.Namespace{ID: work.Scope.NamespaceID, Name: "contract receipt replay"}); err != nil {
		t.Fatal(err)
	}
	cc := commandContext()
	credential := func(id, digest string) ports.Credential {
		return ports.Credential{
			ID: domain.ID(id), NamespaceID: work.Scope.NamespaceID, PrincipalID: cc.PrincipalID,
			Actor: cc.Actor, Digest: digest, ExpiresAt: clock.Now().Add(time.Hour),
		}
	}
	rootA := credential("0199f261-0000-7000-8000-000000000083", "root-token-e25-contract-holder")
	rootB := credential("0199f261-0000-7000-8000-000000000084", "root-token-e25-contract-admin")
	sessionA := credential("0199f262-0000-7000-8000-000000000083", "session-token-e25-contract-holder")
	sessionA.ParentDigest = rootA.Digest
	sessionB := credential("0199f262-0000-7000-8000-000000000084", "session-token-e25-contract-admin")
	sessionB.ParentDigest = rootB.Digest
	for _, item := range []ports.Credential{rootA, rootB, sessionA, sessionB} {
		if err := store.PutCredential(ctx, item); err != nil {
			t.Fatal(err)
		}
	}
	identityContext := func(item ports.Credential) context.Context {
		return application.WithIdentity(ctx, application.Identity{
			CredentialID: item.ID, PrincipalID: item.PrincipalID, NamespaceID: item.NamespaceID,
			CredentialDigest: item.Digest, Actor: item.Actor,
		})
	}

	cc.IdempotencyKey = "e25-contract-acquire-replay-01"
	acquire := application.AcquireWorkContractCommand{Scope: work.Scope, WorkItemID: work.ID, ExpectedWorkItemVersion: work.Version}
	acquired, err := service.AcquireWorkContract(identityContext(rootA), cc, acquire)
	if err != nil {
		t.Fatalf("AcquireWorkContract() error = %v", err)
	}
	contractReplay, err := service.AcquireWorkContract(identityContext(sessionA), cc, acquire)
	if err != nil || !contractReplay.IdempotentReplay || contractReplay.Value.Contract.ID != acquired.Value.Contract.ID {
		t.Fatalf("same-root contracts_v1 acquisition replay = %+v err=%v", contractReplay, err)
	}
	_, err = service.AcquireWorkContract(identityContext(rootB), cc, acquire)
	assertReceiptCredentialConflict(t, err)

	// A separately authorized administrator credential may revoke a contract
	// held by rootA. Its receipt belongs to rootB as issuer, not to the holder.
	cc.IdempotencyKey = "e25-contract-revoke-issuer-01"
	revoke := application.RevokeWorkContractCommand{
		Scope: work.Scope, ContractID: acquired.Value.Contract.ID,
		ExpectedContractVersion: acquired.Value.Contract.Version, Reason: "Authorized recovery action",
	}
	revoked, err := service.RevokeWorkContract(identityContext(rootB), cc, revoke)
	if err != nil {
		t.Fatalf("RevokeWorkContract() error = %v", err)
	}
	revokeReplay, err := service.RevokeWorkContract(identityContext(sessionB), cc, revoke)
	if err != nil || !revokeReplay.IdempotentReplay || revokeReplay.Value.Contract.ID != revoked.Value.Contract.ID {
		t.Fatalf("same issuer-root revoke replay = %+v err=%v", revokeReplay, err)
	}
	_, err = service.RevokeWorkContract(identityContext(rootA), cc, revoke)
	assertReceiptCredentialConflict(t, err)
}

func assertLeaseReceiptReplay(t *testing.T, replay application.MutationResult[domain.WorkItem], err error, original domain.WorkItem) {
	t.Helper()
	if err != nil || !replay.IdempotentReplay || replay.Value.ID != original.ID || replay.Value.Version != original.Version || (replay.Value.CurrentLease == nil) != (original.CurrentLease == nil) {
		t.Fatalf("same-root receipt replay = %+v err=%v", replay, err)
	}
	if replay.Value.CurrentLease != nil && (original.CurrentLease == nil || replay.Value.CurrentLease.ClaimID != original.CurrentLease.ClaimID || replay.Value.CurrentLease.FencingToken != original.CurrentLease.FencingToken) {
		t.Fatalf("same-root replay changed lease receipt: got %+v want %+v", replay.Value.CurrentLease, original.CurrentLease)
	}
}

func assertReceiptCredentialConflict(t *testing.T, err error) {
	t.Helper()
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeIdempotencyConflict {
		t.Fatalf("different-root receipt replay error code = %s err=%v, want idempotency conflict", code, err)
	}
}
