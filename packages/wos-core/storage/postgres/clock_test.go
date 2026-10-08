package postgres

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"path/filepath"
	"testing"
	"time"
)

type clockTestAuthorizer struct{}

func (clockTestAuthorizer) Authorize(context.Context, ports.AuthorizationRequest) error { return nil }
func TestRemoteLeasesUseDatabaseAuthorityDespiteReplicaClockSkew(t *testing.T) {
	requirePostgres(t)
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "authority.db"))
	trusted, err := application.NewService(store, sqliteFixedClock{now: time.Now().UTC()}, &sqliteSequenceIDs{prefix: "01a11760", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	cc := sqliteCommandContext("01a11743-0000-7000-8000-00000000003c", "")
	outcome, err := trusted.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{NamespaceID: testID("01a11760-0000-7000-8000-000000000001"), Title: "Shared authority", DesiredState: "replica skew cannot alter leases", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	_, err = trusted.AddCriterion(ctx, cc, application.AddCriterionCommand{Owner: outcome.Value.Ref(), ExpectedVersion: 1, Title: "Verified", Required: true, VerificationMode: domain.VerificationModeAttestation})
	if err != nil {
		t.Fatal(err)
	}
	_, err = trusted.ActivateOutcome(ctx, cc, application.ActivateOutcomeCommand{Scope: outcome.Value.Scope(), ExpectedVersion: 2})
	if err != nil {
		t.Fatal(err)
	}
	work, err := trusted.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{Scope: outcome.Value.Scope(), Title: "Shared work", Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleTodo})
	if err != nil {
		t.Fatal(err)
	}
	remote, err := application.NewAuthorizedService(store, sqliteFixedClock{now: time.Now().Add(365 * 24 * time.Hour)}, &sqliteSequenceIDs{prefix: "01a11761", next: 1}, clockTestAuthorizer{})
	if err != nil {
		t.Fatal(err)
	}
	ctx = application.WithIdentity(ctx, application.Identity{PrincipalID: cc.PrincipalID, Actor: cc.Actor})
	before := time.Now().UTC()
	claimed, err := remote.ClaimWorkItem(ctx, cc, application.ClaimWorkItemCommand{Scope: outcome.Value.Scope(), WorkItemID: work.Value.ID, ExpectedVersion: 1, TTL: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	if claimed.Value.CurrentLease.AcquiredAt.Before(before.Add(-time.Second)) || claimed.Value.CurrentLease.AcquiredAt.After(time.Now().Add(time.Second)) {
		t.Fatal("lease used replica clock instead of database instant")
	}
	// The skewed local clock must not be allowed to reclaim a still-valid lease.
	_, err = remote.ReclaimWorkItem(ctx, cc, application.ReclaimWorkItemCommand{Scope: outcome.Value.Scope(), WorkItemID: work.Value.ID, ExpectedVersion: claimed.Value.Version, TTL: time.Minute})
	if err == nil {
		t.Fatal("replica's future clock reclaimed a valid lease")
	}
	snapshot, err := remote.GetOutcomeState(ctx, outcome.Value.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.EvaluatedAt.After(time.Now().Add(time.Second)) {
		t.Fatal("snapshot used skewed local time")
	}
	renewed, err := remote.RenewWorkItemLease(ctx, cc, application.RenewWorkItemLeaseCommand{Scope: outcome.Value.Scope(), WorkItemID: work.Value.ID, ExpectedVersion: claimed.Value.Version, ClaimID: claimed.Value.CurrentLease.ClaimID, FencingToken: claimed.Value.CurrentLease.FencingToken, TTL: time.Minute})
	if err != nil {
		t.Fatal(err)
	}
	_, err = remote.CompleteWorkItem(ctx, cc, application.CompleteWorkItemCommand{Scope: outcome.Value.Scope(), WorkItemID: work.Value.ID, ExpectedVersion: renewed.Value.Version, ClaimID: renewed.Value.CurrentLease.ClaimID, FencingToken: renewed.Value.CurrentLease.FencingToken, ResultSummary: "Delivered", Reason: "Valid lease at database time"})
	if err != nil {
		t.Fatal(err)
	}
}

func TestRemoteContractsUseDatabaseAuthorityDespiteReplicaClockSkew(t *testing.T) {
	requirePostgres(t)
	store := openTestStore(t, filepath.Join(t.TempDir(), "contract-authority.db"))
	w := contractStorageFixture(t, store, time.Now().UTC())
	remote, err := application.NewAuthorizedService(store, sqliteFixedClock{now: time.Now().Add(365 * 24 * time.Hour)}, &sqliteSequenceIDs{prefix: "01a11762", next: 1}, clockTestAuthorizer{})
	if err != nil {
		t.Fatal(err)
	}
	cc := sqliteCommandContext("01a11743-0000-7000-8000-00000000003d", "contract-database-time-1")
	ctx := application.WithIdentity(context.Background(), application.Identity{PrincipalID: cc.PrincipalID, Actor: cc.Actor})
	before := time.Now().UTC()
	got, err := remote.AcquireWorkContract(ctx, cc, application.AcquireWorkContractCommand{Scope: w.Scope, WorkItemID: w.ID, ExpectedWorkItemVersion: w.Version, TTLSeconds: 30})
	if err != nil {
		t.Fatal(err)
	}
	c := got.Value.Contract
	if c.AcquiredAt.Before(before.Add(-time.Second)) || c.AcquiredAt.After(time.Now().Add(time.Second)) {
		t.Fatal("contract acquisition used replica clock")
	}
	cc.IdempotencyKey = "contract-database-time-2"
	_, err = remote.RenewWorkContract(ctx, cc, application.RenewWorkContractCommand{Scope: w.Scope, ContractID: c.ID, Authority: application.ContractAuthority{ExecutionID: c.ExecutionID, FencingToken: c.FencingToken, SpecDigest: c.SpecDigest}, ExpectedLeaseVersion: c.LeaseVersion, TTLSeconds: 30})
	if err != nil {
		t.Fatal(err)
	}
	view, err := remote.GetWorkContract(ctx, w.Scope, c.ID)
	if err != nil || view.Value.EvaluatedAt.After(time.Now().Add(time.Second)) || view.Value.EffectiveStatus != domain.ContractActive {
		t.Fatalf("contract query clock %v %v", view, err)
	}
}
