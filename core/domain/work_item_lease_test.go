package domain_test

import (
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func TestRenewWorkItemLeasePreservesClaimAndFencingToken(t *testing.T) {
	now := time.Date(2026, 10, 2, 15, 0, 0, 0, time.UTC)
	scope := domain.Scope{
		NamespaceID: domain.MustParseID("0199f200-0000-7000-8000-000000000001"),
		OutcomeID:   domain.MustParseID("0199f200-0000-7000-8000-000000000002"),
	}
	workID := domain.MustParseID("0199f200-0000-7000-8000-000000000003")
	claimID := domain.MustParseID("0199f200-0000-7000-8000-000000000004")
	actor := domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "worker-a"}

	work, err := domain.NewWorkItem(workID, scope, "renew lease", "", domain.PriorityNormal, domain.WorkItemLifecycleTodo, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := work.Claim(claimID, "principal-a", actor, domain.DefaultLeaseTTL, now); err != nil {
		t.Fatal(err)
	}
	versionBefore := work.Version
	tokenBefore := work.CurrentLease.FencingToken
	renewedAt := now.Add(time.Minute)

	if err := work.RenewLease("principal-a", claimID, tokenBefore, 10*time.Minute, renewedAt); err != nil {
		t.Fatalf("RenewLease() error = %v", err)
	}
	if work.CurrentLease.ClaimID != claimID {
		t.Fatalf("claim id changed on renew: %s", work.CurrentLease.ClaimID)
	}
	if work.CurrentLease.FencingToken != tokenBefore {
		t.Fatalf("fencing token changed on renew: got %d want %d", work.CurrentLease.FencingToken, tokenBefore)
	}
	if want := renewedAt.Add(10 * time.Minute); !work.CurrentLease.ExpiresAt.Equal(want) {
		t.Fatalf("expires_at = %v, want %v", work.CurrentLease.ExpiresAt, want)
	}
	if work.Version != versionBefore+1 {
		t.Fatalf("version = %d, want %d", work.Version, versionBefore+1)
	}
}

func TestExpiredLeaseMustBeReclaimedAndFencesStaleClaimant(t *testing.T) {
	now := time.Date(2026, 10, 2, 16, 0, 0, 0, time.UTC)
	scope := domain.Scope{
		NamespaceID: domain.MustParseID("0199f200-0000-7000-8000-000000000011"),
		OutcomeID:   domain.MustParseID("0199f200-0000-7000-8000-000000000012"),
	}
	workID := domain.MustParseID("0199f200-0000-7000-8000-000000000013")
	oldClaimID := domain.MustParseID("0199f200-0000-7000-8000-000000000014")
	newClaimID := domain.MustParseID("0199f200-0000-7000-8000-000000000015")
	oldActor := domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "worker-old"}
	newActor := domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "worker-new"}

	work, err := domain.NewWorkItem(workID, scope, "reclaim lease", "", domain.PriorityNormal, domain.WorkItemLifecycleTodo, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := work.Claim(oldClaimID, "principal-old", oldActor, domain.MinLeaseTTL, now); err != nil {
		t.Fatal(err)
	}
	oldToken := work.CurrentLease.FencingToken
	expiredAt := work.CurrentLease.ExpiresAt

	if err := work.RenewLease("principal-old", oldClaimID, oldToken, domain.DefaultLeaseTTL, expiredAt); err == nil {
		t.Fatal("RenewLease() unexpectedly renewed an expired lease")
	}
	if err := work.Reclaim(newClaimID, "principal-new", newActor, domain.DefaultLeaseTTL, expiredAt); err != nil {
		t.Fatalf("Reclaim() error = %v", err)
	}
	if work.Lifecycle != domain.WorkItemLifecycleInProgress {
		t.Fatalf("lifecycle = %q, want in_progress", work.Lifecycle)
	}
	if work.CurrentLease.ClaimID != newClaimID || work.CurrentLease.PrincipalID != "principal-new" {
		t.Fatalf("unexpected reclaimed lease %#v", work.CurrentLease)
	}
	if work.CurrentLease.FencingToken != oldToken+1 {
		t.Fatalf("fencing token = %d, want %d", work.CurrentLease.FencingToken, oldToken+1)
	}
	if err := work.Complete("principal-old", oldClaimID, oldToken, "stale completion", domain.Conclusion{}, expiredAt.Add(time.Second)); err == nil {
		t.Fatal("Complete() unexpectedly accepted stale claimant after reclaim")
	}
}

func TestReclaimRejectsLiveLease(t *testing.T) {
	now := time.Date(2026, 10, 2, 17, 0, 0, 0, time.UTC)
	scope := domain.Scope{
		NamespaceID: domain.MustParseID("0199f200-0000-7000-8000-000000000021"),
		OutcomeID:   domain.MustParseID("0199f200-0000-7000-8000-000000000022"),
	}
	workID := domain.MustParseID("0199f200-0000-7000-8000-000000000023")
	oldClaimID := domain.MustParseID("0199f200-0000-7000-8000-000000000024")
	newClaimID := domain.MustParseID("0199f200-0000-7000-8000-000000000025")
	actor := domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "worker"}

	work, err := domain.NewWorkItem(workID, scope, "live lease", "", domain.PriorityNormal, domain.WorkItemLifecycleTodo, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := work.Claim(oldClaimID, "principal-a", actor, domain.DefaultLeaseTTL, now); err != nil {
		t.Fatal(err)
	}
	if err := work.Reclaim(newClaimID, "principal-b", actor, domain.DefaultLeaseTTL, now.Add(time.Minute)); err == nil {
		t.Fatal("Reclaim() unexpectedly replaced a live lease")
	}
}
