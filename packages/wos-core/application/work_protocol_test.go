package application_test

import (
	"context"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"testing"
	"time"
)

func TestNamespaceCutoverPreservesLegacyReceiptAndRejectsOldMutations(t *testing.T) {
	s, store, clock, w := contractServiceFixture(t)
	ctx := context.Background()
	tx, _ := store.Begin(ctx)
	old, _ := tx.WorkItems().Get(ctx, w.Scope, w.ID)
	old.ContractsEnabled = false
	old.Version++
	if err := tx.WorkItems().Save(ctx, old, w.Version); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	cc := commandContext()
	cc.IdempotencyKey = "legacy-protocol-claim-1"
	claim := a.ClaimWorkItemCommand{Scope: w.Scope, WorkItemID: w.ID, ExpectedVersion: old.Version, TTL: time.Minute}
	claimed, err := s.ClaimWorkItem(ctx, cc, claim)
	if err != nil {
		t.Fatal(err)
	}
	change := a.SetNamespaceWorkProtocolCommand{Scope: w.Scope, ExpectedProtocolVersion: 1, Phase: d.WorkProtocolDraining, Reason: "retiring legacy writers"}
	admin := commandContext()
	admin.IdempotencyKey = "protocol-drain-1"
	draining, err := s.SetNamespaceWorkProtocol(ctx, admin, change)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.ClaimWorkItem(ctx, cc, claim)
	if err != nil || !replay.IdempotentReplay || replay.Value.CurrentLease.ClaimID != claimed.Value.CurrentLease.ClaimID {
		t.Fatalf("old receipt changed: %v %v", replay, err)
	}
	cc.IdempotencyKey = "legacy-protocol-new-claim"
	if _, err = s.ClaimWorkItem(ctx, cc, claim); err == nil {
		t.Fatal("draining permitted a new legacy claim")
	}
	change.ExpectedProtocolVersion = draining.Value.Protocol.Version
	change.Phase = d.WorkProtocolContracts
	admin.IdempotencyKey = "protocol-cutover-1"
	if _, err = s.SetNamespaceWorkProtocol(ctx, admin, change); err == nil {
		t.Fatal("cutover without old writer acknowledgement")
	}
	change.WritersDrained = true
	if _, err = s.SetNamespaceWorkProtocol(ctx, admin, change); err == nil {
		t.Fatal("cutover with valid legacy lease")
	}
	clock.set(clock.Now().Add(time.Minute))
	cutover, err := s.SetNamespaceWorkProtocol(ctx, admin, change)
	if err != nil {
		t.Fatal(err)
	}
	tx, _ = store.Begin(ctx)
	migrated, err := tx.WorkItems().Get(ctx, w.Scope, w.ID)
	tx.Rollback()
	if err != nil {
		t.Fatal(err)
	}
	if !migrated.ContractsEnabled || migrated.CurrentLease != nil || migrated.Version != claimed.Value.Version || migrated.Lifecycle != d.WorkItemLifecycleInProgress {
		t.Fatalf("migration rewrote canonical work: %+v", migrated)
	}
	cc.IdempotencyKey = "legacy-protocol-claim-1"
	replay, err = s.ClaimWorkItem(ctx, cc, claim)
	if err != nil || !replay.IdempotentReplay {
		t.Fatal("legacy replay failed after activation", err)
	}
	admin.IdempotencyKey = "protocol-downgrade"
	change.Phase = d.WorkProtocolLegacy
	change.ExpectedProtocolVersion = cutover.Value.Protocol.Version
	if _, err = s.SetNamespaceWorkProtocol(ctx, admin, change); err == nil {
		t.Fatal("silent downgrade accepted")
	}
	cc.IdempotencyKey = "contract-post-cutover"
	got, err := s.AcquireWorkContract(ctx, cc, a.AcquireWorkContractCommand{Scope: w.Scope, WorkItemID: w.ID, ExpectedWorkItemVersion: migrated.Version, TTLSeconds: 30})
	if err != nil {
		t.Fatal(err)
	}
	clock.set(got.Value.Contract.ExpiresAt)
	admin.IdempotencyKey = "reconcile-expiry"
	reconciled, err := s.ReconcileExpiredWorkContracts(ctx, admin, a.ReconcileExpiredWorkContractsCommand{Scope: w.Scope, Limit: 1})
	if err != nil || len(reconciled.Value.Contracts) != 1 {
		t.Fatal("expiry reconciliation", err, reconciled)
	}
	view, err := s.GetWorkContract(ctx, w.Scope, got.Value.Contract.ID)
	if err != nil || view.Value.Contract.Status != d.ContractExpired || !view.Value.Recoverable {
		t.Fatal("expiry not recoverable", err, view)
	}
	admin.IdempotencyKey = "cancel-reconciled"
	tx, _ = store.Begin(ctx)
	current, _ := tx.WorkItems().Get(ctx, w.Scope, w.ID)
	tx.Rollback()
	if _, err = s.AdministrativeCancelWorkItem(ctx, admin, a.AdministrativeCancelWorkItemCommand{Scope: w.Scope, WorkItemID: w.ID, ExpectedVersion: current.Version, Reason: "operator cancellation after expiry"}); err != nil {
		t.Fatal(err)
	}
}
