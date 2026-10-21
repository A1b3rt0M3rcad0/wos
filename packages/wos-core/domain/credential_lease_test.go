package domain_test

import (
	"math"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestCredentialBoundLeaseRequiresSameCredentialPrincipalClaimAndFence(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	work, actor, claim, holder, session := credentialLeaseFixture(t, now)
	if err := work.ClaimForCredential(claim, "principal-a", holder, &session, actor, domain.DefaultLeaseTTL, now); err != nil {
		t.Fatalf("ClaimForCredential() error = %v", err)
	}
	lease := work.CurrentLease
	if lease.CredentialBindingVersion != domain.WorkLeaseCredentialBindingV1 || lease.HolderCredentialID != holder {
		t.Fatalf("lease binding = version %d holder %q, want v1 holder %q", lease.CredentialBindingVersion, lease.HolderCredentialID, holder)
	}
	if lease.SessionCredentialID == nil || *lease.SessionCredentialID != session {
		t.Fatalf("session provenance = %v, want %q", lease.SessionCredentialID, session)
	}

	if err := work.RenewLeaseForCredential("principal-a", holder, claim, lease.FencingToken, 10*time.Minute, now.Add(time.Minute)); err != nil {
		t.Fatalf("same credential could not renew its lease: %v", err)
	}

	wrongClaim := domain.MustParseID("019a3200-0000-7000-8000-000000000010")
	for _, tc := range []struct {
		name       string
		principal  string
		credential domain.ID
		claim      domain.ID
		fence      uint64
	}{
		{name: "another credential for same principal", principal: "principal-a", credential: domain.MustParseID("019a3200-0000-7000-8000-000000000011"), claim: claim, fence: lease.FencingToken},
		{name: "different principal", principal: "principal-b", credential: holder, claim: claim, fence: lease.FencingToken},
		{name: "wrong claim", principal: "principal-a", credential: holder, claim: wrongClaim, fence: lease.FencingToken},
		{name: "stale fence", principal: "principal-a", credential: holder, claim: claim, fence: lease.FencingToken - 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := work.CheckCompletionAuthorityForCredential(tc.principal, tc.credential, tc.claim, tc.fence, now.Add(2*time.Minute)); err == nil {
				t.Fatal("CheckCompletionAuthorityForCredential() unexpectedly authorized mismatched lease identity")
			}
		})
	}
	if err := work.CheckCompletionAuthorityForCredential("principal-a", holder, claim, lease.FencingToken, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("matching credential, claim and fence were rejected: %v", err)
	}
}

func TestCredentialBoundLeaseCanBeCompletedByItsHolder(t *testing.T) {
	now := time.Date(2026, 10, 21, 12, 30, 0, 0, time.UTC)
	work, actor, claim, holder, _ := credentialLeaseFixture(t, now)
	if err := work.ClaimForCredential(claim, "principal-a", holder, nil, actor, domain.DefaultLeaseTTL, now); err != nil {
		t.Fatal(err)
	}
	fence := work.CurrentLease.FencingToken
	wrongHolder := domain.MustParseID("019a3200-0000-7000-8000-000000000040")
	if err := work.CompleteForCredential("principal-a", wrongHolder, claim, fence, "should remain active", domain.Conclusion{}, now.Add(time.Minute)); err == nil {
		t.Fatal("different credential completed the lease")
	}
	if work.Lifecycle != domain.WorkItemLifecycleInProgress {
		t.Fatalf("rejected completion mutated lifecycle to %q", work.Lifecycle)
	}
	decision := domain.Conclusion{PrincipalID: "principal-a", Actor: actor, Reason: "verified result", ConcludedAt: now.Add(time.Minute)}
	if err := work.CompleteForCredential("principal-a", holder, claim, fence, "verified result", decision, now.Add(time.Minute)); err != nil {
		t.Fatalf("holder could not complete its lease: %v", err)
	}
	if work.Lifecycle != domain.WorkItemLifecycleDone || work.CurrentLease != nil || work.CurrentConclusion == nil {
		t.Fatalf("completion did not persist the terminal result: %#v", work)
	}
}

func TestPrincipalOnlyLeaseMethodsCannotOperateCredentialBoundLease(t *testing.T) {
	now := time.Date(2026, 10, 21, 13, 0, 0, 0, time.UTC)
	operations := []struct {
		name string
		call func(*domain.WorkItem, string, domain.ID, uint64, time.Time) error
	}{
		{name: "renew", call: func(w *domain.WorkItem, principal string, claim domain.ID, fence uint64, at time.Time) error {
			return w.RenewLease(principal, claim, fence, domain.DefaultLeaseTTL, at)
		}},
		{name: "release", call: func(w *domain.WorkItem, principal string, claim domain.ID, fence uint64, at time.Time) error {
			return w.Release(principal, claim, fence, at)
		}},
		{name: "complete", call: func(w *domain.WorkItem, principal string, claim domain.ID, fence uint64, at time.Time) error {
			return w.Complete(principal, claim, fence, "result", domain.Conclusion{}, at)
		}},
	}
	for i, operation := range operations {
		t.Run(operation.name, func(t *testing.T) {
			work, actor, claim, holder, session := credentialLeaseFixture(t, now)
			if err := work.ClaimForCredential(claim, "principal-a", holder, &session, actor, domain.DefaultLeaseTTL, now); err != nil {
				t.Fatal(err)
			}
			if err := operation.call(work, "principal-a", claim, work.CurrentLease.FencingToken, now.Add(time.Minute)); err == nil {
				t.Fatalf("legacy principal-only operation %d unexpectedly accepted credential-bound lease", i)
			}
		})
	}
}

func TestLegacyLeaseRemainsReadableAndUsesLegacyPrincipalAuthority(t *testing.T) {
	now := time.Date(2026, 10, 21, 14, 0, 0, 0, time.UTC)
	work, actor, claim, _, _ := credentialLeaseFixture(t, now)
	if err := work.Claim(claim, "legacy-principal", actor, domain.DefaultLeaseTTL, now); err != nil {
		t.Fatal(err)
	}
	if work.CurrentLease.CredentialBindingVersion != 0 || !work.CurrentLease.HolderCredentialID.IsZero() {
		t.Fatalf("legacy claim unexpectedly gained a credential binding: %#v", work.CurrentLease)
	}
	if err := work.CurrentLease.Validate(); err != nil {
		t.Fatalf("unbound historical lease was rejected: %v", err)
	}
	if err := work.RenewLease("legacy-principal", claim, work.CurrentLease.FencingToken, domain.DefaultLeaseTTL, now.Add(time.Minute)); err != nil {
		t.Fatalf("legacy principal-based renewal stopped working for an unbound lease: %v", err)
	}
}

func TestCredentialBoundLeaseReclaimFencesOldHolderAndAllowsNewHolder(t *testing.T) {
	now := time.Date(2026, 10, 21, 15, 0, 0, 0, time.UTC)
	work, actor, oldClaim, oldHolder, session := credentialLeaseFixture(t, now)
	if err := work.ClaimForCredential(oldClaim, "principal-a", oldHolder, &session, actor, domain.MinLeaseTTL, now); err != nil {
		t.Fatal(err)
	}
	oldFence := work.CurrentLease.FencingToken
	newClaim := domain.MustParseID("019a3200-0000-7000-8000-000000000020")
	newHolder := domain.MustParseID("019a3200-0000-7000-8000-000000000021")
	if err := work.Reclaim(newClaim, "principal-b", actor, domain.DefaultLeaseTTL, now.Add(domain.MinLeaseTTL)); err == nil {
		t.Fatal("legacy principal-only reclaim unexpectedly took over a credential-bound lease")
	}
	if err := work.ReclaimForCredential(newClaim, "principal-b", newHolder, nil, actor, domain.MaxLeaseTTL, now.Add(domain.MinLeaseTTL)); err != nil {
		t.Fatalf("ReclaimForCredential() at the expiry boundary failed: %v", err)
	}
	if work.CurrentLease.FencingToken != oldFence+1 || work.CurrentLease.HolderCredentialID != newHolder || work.CurrentLease.SessionCredentialID != nil {
		t.Fatalf("reclaimed lease did not carry new fenced holder: %#v", work.CurrentLease)
	}
	if err := work.CheckCompletionAuthorityForCredential("principal-a", oldHolder, oldClaim, oldFence, now.Add(domain.MinLeaseTTL+time.Second)); err == nil {
		t.Fatal("old holder remained authorized after credential-bound reclaim")
	}
	if err := work.CheckCompletionAuthorityForCredential("principal-b", newHolder, newClaim, oldFence+1, now.Add(domain.MinLeaseTTL+time.Second)); err != nil {
		t.Fatalf("new holder cannot use reclaimed lease: %v", err)
	}
}

func TestCredentialLeaseTTLBoundariesAndFencingOverflow(t *testing.T) {
	now := time.Date(2026, 10, 21, 16, 0, 0, 0, time.UTC)
	for _, ttl := range []time.Duration{domain.MinLeaseTTL, domain.MaxLeaseTTL} {
		t.Run(ttl.String(), func(t *testing.T) {
			work, actor, claim, holder, _ := credentialLeaseFixture(t, now)
			if err := work.ClaimForCredential(claim, "principal-a", holder, nil, actor, ttl, now); err != nil {
				t.Fatalf("boundary TTL %s rejected: %v", ttl, err)
			}
		})
	}
	for _, ttl := range []time.Duration{domain.MinLeaseTTL - time.Nanosecond, domain.MaxLeaseTTL + time.Nanosecond} {
		work, actor, claim, holder, _ := credentialLeaseFixture(t, now)
		if err := work.ClaimForCredential(claim, "principal-a", holder, nil, actor, ttl, now); err == nil {
			t.Fatalf("out-of-range TTL %s unexpectedly accepted", ttl)
		}
		if work.CurrentLease != nil || work.Lifecycle != domain.WorkItemLifecycleTodo {
			t.Fatalf("invalid TTL %s mutated the work item", ttl)
		}
	}

	work, actor, claim, holder, _ := credentialLeaseFixture(t, now)
	work.LastFencingToken = math.MaxUint64
	if err := work.ClaimForCredential(claim, "principal-a", holder, nil, actor, domain.DefaultLeaseTTL, now); err == nil {
		t.Fatal("credential-bound claim accepted exhausted fencing sequence")
	}
	if work.CurrentLease != nil || work.Lifecycle != domain.WorkItemLifecycleTodo {
		t.Fatal("fencing overflow mutated work item")
	}
}

func TestWorkLeaseCredentialBindingValidationFailsClosed(t *testing.T) {
	now := time.Date(2026, 10, 21, 17, 0, 0, 0, time.UTC)
	claim := domain.MustParseID("019a3200-0000-7000-8000-000000000030")
	holder := domain.MustParseID("019a3200-0000-7000-8000-000000000031")
	session := domain.MustParseID("019a3200-0000-7000-8000-000000000032")
	base := domain.WorkLease{ClaimID: claim, PrincipalID: "principal-a", Actor: domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "worker"}, FencingToken: 1, AcquiredAt: now, ExpiresAt: now.Add(time.Minute)}
	valid := base
	valid.CredentialBindingVersion = domain.WorkLeaseCredentialBindingV1
	valid.HolderCredentialID = holder
	valid.SessionCredentialID = &session
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid v1 binding rejected: %v", err)
	}

	invalid := []struct {
		name  string
		lease domain.WorkLease
	}{
		{name: "unversioned holder", lease: func() domain.WorkLease { v := base; v.HolderCredentialID = holder; return v }()},
		{name: "missing holder", lease: func() domain.WorkLease {
			v := base
			v.CredentialBindingVersion = domain.WorkLeaseCredentialBindingV1
			return v
		}()},
		{name: "malformed holder", lease: func() domain.WorkLease { v := valid; v.HolderCredentialID = "credential"; return v }()},
		{name: "malformed session provenance", lease: func() domain.WorkLease {
			v := valid
			bad := domain.ID("session")
			v.SessionCredentialID = &bad
			return v
		}()},
		{name: "future binding version", lease: func() domain.WorkLease { v := valid; v.CredentialBindingVersion++; return v }()},
	}
	for _, tc := range invalid {
		t.Run(tc.name, func(t *testing.T) {
			if err := tc.lease.Validate(); err == nil {
				t.Fatal("invalid credential binding unexpectedly validated")
			}
		})
	}
}

func credentialLeaseFixture(t *testing.T, now time.Time) (*domain.WorkItem, domain.ActorRef, domain.ID, domain.ID, domain.ID) {
	t.Helper()
	base := "019a32" + "00"
	work, err := domain.NewWorkItem(
		domain.MustParseID(base+"-0000-7000-8000-000000000001"),
		domain.Scope{NamespaceID: domain.MustParseID(base + "-0000-7000-8000-000000000002"), OutcomeID: domain.MustParseID(base + "-0000-7000-8000-000000000003")},
		"Credential-bound work", "", domain.PriorityNormal, domain.WorkItemLifecycleTodo, now,
	)
	if err != nil {
		t.Fatal(err)
	}
	actor := domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "worker"}
	claim := domain.MustParseID(base + "-0000-7000-8000-000000000004")
	holder := domain.MustParseID(base + "-0000-7000-8000-000000000005")
	session := domain.MustParseID(base + "-0000-7000-8000-000000000006")
	return &work, actor, claim, holder, session
}
