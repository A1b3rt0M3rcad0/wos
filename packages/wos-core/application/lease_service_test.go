package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

func TestLeaseRenewAndReclaimAreAuditedAndFenceStaleClaimant(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	start := time.Date(2026, 10, 2, 18, 0, 0, 0, time.UTC)
	ids := &sequenceIDs{next: 200}
	service, err := application.NewService(store, fixedClock{now: start}, ids)
	if err != nil {
		t.Fatal(err)
	}

	ccOld := domain.CommandContext{
		PrincipalID: "worker-old",
		Actor:       domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "worker-old"},
		CommandID:   id("0199f210-0000-7000-8000-000000000001"),
	}
	namespaceID := id("0199f210-0000-7000-8000-000000000002")

	created, err := service.CreateOutcome(ctx, ccOld, application.CreateOutcomeCommand{
		NamespaceID: namespaceID, Title: "Lease coordination", DesiredState: "work fenced", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value
	if _, err := service.AddCriterion(ctx, ccOld, application.AddCriterionCommand{
		Owner: outcome.Ref(), ExpectedVersion: outcome.Version, Title: "verified",
		Required: true, VerificationMode: domain.VerificationModeAttestation,
	}); err != nil {
		t.Fatal(err)
	}
	outcome.Version++
	activated, err := service.ActivateOutcome(ctx, ccOld, application.ActivateOutcomeCommand{
		Scope: outcome.Scope(), ExpectedVersion: outcome.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome = activated.Value

	createdWork, err := service.CreateWorkItem(ctx, ccOld, application.CreateWorkItemCommand{
		Scope: outcome.Scope(), Title: "leased work", Priority: domain.PriorityNormal,
		Lifecycle: domain.WorkItemLifecycleTodo,
	})
	if err != nil {
		t.Fatal(err)
	}
	work := createdWork.Value
	claimed, err := service.ClaimWorkItem(ctx, ccOld, application.ClaimWorkItemCommand{
		Scope: outcome.Scope(), WorkItemID: work.ID, ExpectedVersion: work.Version, TTL: domain.MinLeaseTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	work = claimed.Value
	oldClaimID := work.CurrentLease.ClaimID
	oldToken := work.CurrentLease.FencingToken

	renewAt := start.Add(10 * time.Second)
	renewService, err := application.NewService(store, fixedClock{now: renewAt}, &sequenceIDs{next: 400})
	if err != nil {
		t.Fatal(err)
	}
	renewed, err := renewService.RenewWorkItemLease(ctx, ccOld, application.RenewWorkItemLeaseCommand{
		Scope: outcome.Scope(), WorkItemID: work.ID, ExpectedVersion: work.Version,
		ClaimID: oldClaimID, FencingToken: oldToken, TTL: domain.MinLeaseTTL,
	})
	if err != nil {
		t.Fatalf("RenewWorkItemLease() error = %v", err)
	}
	work = renewed.Value
	if work.CurrentLease.FencingToken != oldToken {
		t.Fatalf("renew changed fencing token: got %d want %d", work.CurrentLease.FencingToken, oldToken)
	}

	reclaimAt := renewAt.Add(domain.MinLeaseTTL)
	ccNew := domain.CommandContext{
		PrincipalID: "worker-new",
		Actor:       domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "worker-new"},
		CommandID:   id("0199f210-0000-7000-8000-000000000003"),
	}
	reclaimService, err := application.NewService(store, fixedClock{now: reclaimAt}, &sequenceIDs{next: 600})
	if err != nil {
		t.Fatal(err)
	}
	reclaimed, err := reclaimService.ReclaimWorkItem(ctx, ccNew, application.ReclaimWorkItemCommand{
		Scope: outcome.Scope(), WorkItemID: work.ID, ExpectedVersion: work.Version, TTL: domain.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatalf("ReclaimWorkItem() error = %v", err)
	}
	work = reclaimed.Value
	if work.CurrentLease.PrincipalID != "worker-new" {
		t.Fatalf("lease principal = %q, want worker-new", work.CurrentLease.PrincipalID)
	}
	if work.CurrentLease.ClaimID == oldClaimID {
		t.Fatal("reclaim reused old claim id")
	}
	if work.CurrentLease.FencingToken != oldToken+1 {
		t.Fatalf("reclaim fencing token = %d, want %d", work.CurrentLease.FencingToken, oldToken+1)
	}

	staleService, err := application.NewService(store, fixedClock{now: reclaimAt.Add(time.Second)}, &sequenceIDs{next: 800})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := staleService.CompleteWorkItem(ctx, ccOld, application.CompleteWorkItemCommand{
		Scope: outcome.Scope(), WorkItemID: work.ID, ExpectedVersion: work.Version,
		ClaimID: oldClaimID, FencingToken: oldToken, ResultSummary: "stale", Reason: "stale worker",
	}); err == nil {
		t.Fatal("CompleteWorkItem() unexpectedly accepted stale claimant after reclaim")
	}

	events := store.SnapshotDomainEvents(outcome.Scope())
	var sawRenew, sawReclaim bool
	for _, event := range events {
		switch event.EventType {
		case "work_item.lease_renewed":
			sawRenew = true
		case "work_item.reclaimed":
			sawReclaim = true
		}
	}
	if !sawRenew || !sawReclaim {
		t.Fatalf("lease events missing: renew=%v reclaim=%v", sawRenew, sawReclaim)
	}
}
