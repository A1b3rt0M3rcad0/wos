package domain_test

import (
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"reflect"
	"strings"
	"testing"
	"time"
)

func reviewBinding() d.SignedContractBinding {
	key := d.MustParseID("0199f201-0000-7000-8000-000000000001")
	return d.SignedContractBinding{ProtocolVersion: 2, ServerID: "persistent-server", CredentialID: d.MustParseID("0199f201-0000-7000-8000-000000000002"), SignerKeyID: key, AcceptanceFloor: d.AcceptanceIndependentReview, PolicyRevision: 1, AllowedSigningKeyIDs: []d.ID{key}}
}
func reviewFixture(t *testing.T) (d.ReviewCase, time.Time) {
	t.Helper()
	_, work, now := contractFixture(t)
	r := d.ReviewCase{ID: d.MustParseID("0199f201-0000-7000-8000-000000000003"), Scope: work.Scope, WorkItemID: work.WorkItemID, WorkContractID: work.ID, SubmissionID: d.MustParseID("0199f201-0000-7000-8000-000000000004"), SubmissionDigest: "sha256:" + strings.Repeat("1", 64), IssuedSpecDigest: "sha256:" + strings.Repeat("2", 64), PolicyRevision: 1, AcceptanceFloor: d.AcceptanceIndependentReview, Round: 1, Version: 1, Status: d.ReviewPending, ExecutionPrincipals: []string{"executor", "correction-executor"}, ExecutionGroups: []string{"engineering"}, CreatedAt: now, UpdatedAt: now}
	if err := r.Validate(); err != nil {
		t.Fatal(err)
	}
	return r, now
}
func TestSignedDeliveryClosesExecutorWithoutCompletingTask(t *testing.T) {
	w, c, now := contractFixture(t)
	p := submissionFixture(t, c, now)
	if err := c.Deliver(p, "holder", c.ExecutionID, c.FencingToken, c.Version, now); err == nil {
		t.Fatal("unsigned v1 delivered")
	}
	binding := reviewBinding()
	c.SignedBinding = &binding
	old := c
	if err := c.Deliver(p, "holder", c.ExecutionID, c.FencingToken, c.Version, now); err != nil {
		t.Fatal(err)
	}
	if c.Status != d.ContractDelivered || c.ValidAt(now) || w.Lifecycle != d.WorkItemLifecycleInProgress {
		t.Fatal("delivery retained executor authority or certified task")
	}
	if err := d.ValidateContractUpdate(old, c); err != nil {
		t.Fatal(err)
	}
	if err := c.Renew("holder", c.ExecutionID, c.FencingToken, c.LeaseVersion, 30, now); err == nil {
		t.Fatal("delivered authority renewed")
	}
	if err := c.Deliver(p, "holder", c.ExecutionID, c.FencingToken, c.Version, now); err == nil {
		t.Fatal("delivered authority reopened")
	}
}
func TestReviewIndependentAuthorityExpiryTakeoverAndCompletion(t *testing.T) {
	r, now := reviewFixture(t)
	b := reviewBinding()
	actor := d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "reviewer"}
	contractID := d.MustParseID("0199f201-0000-7000-8000-000000000005")
	execution := d.MustParseID("0199f201-0000-7000-8000-000000000006")
	for _, p := range []string{"executor", "correction-executor"} {
		if _, err := d.NewReviewContract(contractID, execution, r, p, actor, "", b, d.DefaultContractLeasePolicy(), 30, now); err == nil {
			t.Fatal("historical executor reviewed", p)
		}
	}
	if _, err := d.NewReviewContract(contractID, execution, r, "reviewer", actor, "engineering", b, d.DefaultContractLeasePolicy(), 30, now); err == nil {
		t.Fatal("separation group bypass")
	}
	c, err := d.NewReviewContract(contractID, execution, r, "reviewer", actor, "quality", b, d.DefaultContractLeasePolicy(), 30, now)
	if err != nil {
		t.Fatal(err)
	}
	old := r
	if err = r.Bind(c, now); err != nil {
		t.Fatal(err)
	}
	if err = d.ValidateReviewCaseUpdate(old, r); err != nil {
		t.Fatal(err)
	}
	nextExecution := d.MustParseID("0199f201-0000-7000-8000-000000000007")
	before := c
	if err = c.Resume("reviewer", execution, c.FencingToken, c.LeaseVersion, nextExecution, now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if !c.ExpiresAt.Equal(before.ExpiresAt) || c.FencingToken <= before.FencingToken {
		t.Fatal("takeover extended TTL or reused fence")
	}
	if err = d.ValidateReviewContractUpdate(before, c); err != nil {
		t.Fatal(err)
	}
	if err = c.Authorize("reviewer", execution, before.FencingToken, now); err == nil {
		t.Fatal("old review execution remained authoritative")
	}
	expired := c
	if err = expired.Renew("reviewer", c.ExecutionID, c.FencingToken, c.LeaseVersion, 30, c.ExpiresAt); err == nil {
		t.Fatal("renewed at expiry boundary")
	}
	if err = expired.Close(d.ContractExpired, "system", "review lease expired", expired.Version, expired.ExpiresAt); err != nil {
		t.Fatal(err)
	}
	old = r
	if err = r.Release(expired, expired.ExpiresAt); err != nil {
		t.Fatal(err)
	}
	if err = d.ValidateReviewCaseUpdate(old, r); err != nil {
		t.Fatal(err)
	}
	if r.Status != d.ReviewPending || r.LastFencingToken != uint64(c.FencingToken) {
		t.Fatal("expiration closed case or lost takeover high water")
	}
	later := c.ExpiresAt.Add(time.Second)
	replacement, err := d.NewReviewContract(d.MustParseID("0199f201-0000-7000-8000-000000000008"), nextExecution, r, "reviewer-two", actor, "quality", b, d.DefaultContractLeasePolicy(), 30, later)
	if err != nil {
		t.Fatal(err)
	}
	if replacement.FencingToken <= c.FencingToken {
		t.Fatal("review reacquisition reused fencing")
	}
	if err = r.Bind(replacement, later); err != nil {
		t.Fatal(err)
	}
	// Reviewer completes its own authority with no executor credential or lease.
	if err = replacement.Close(d.ContractCompleted, "executor", "approve", replacement.Version, later); err == nil {
		t.Fatal("executor closed review authority")
	}
	before = replacement
	if err = replacement.Close(d.ContractCompleted, "reviewer-two", "approved evidence", replacement.Version, later); err != nil {
		t.Fatal(err)
	}
	if err = d.ValidateReviewContractUpdate(before, replacement); err != nil {
		t.Fatal(err)
	}
	old = r
	if err = r.Decide(replacement, d.ReviewApproved, "reviewer-two", "approved evidence", later); err != nil {
		t.Fatal(err)
	}
	if err = d.ValidateReviewCaseUpdate(old, r); err != nil {
		t.Fatal(err)
	}
	terminal := r
	if err = r.Release(replacement, later); err == nil || !reflect.DeepEqual(terminal, r) {
		t.Fatal("terminal review reactivated/mutated")
	}
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var decoded d.ReviewCase
	if err = json.Unmarshal(raw, &decoded); err != nil || decoded.Validate() != nil {
		t.Fatal("persisted review metadata lost exact counters", err)
	}
}
func TestReviewTargetPolicyAndRoundAreImmutable(t *testing.T) {
	r, now := reviewFixture(t)
	raw, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var pending d.ReviewCase
	if err = json.Unmarshal(raw, &pending); err != nil || pending.LastFencingToken != 0 {
		t.Fatal("pending zero high water failed round trip", err)
	}
	for _, change := range []func(*d.ReviewCase){func(n *d.ReviewCase) { n.SubmissionDigest = "sha256:" + strings.Repeat("3", 64) }, func(n *d.ReviewCase) { n.PolicyRevision++ }, func(n *d.ReviewCase) { n.Round++ }, func(n *d.ReviewCase) { n.ExecutionPrincipals = []string{"someone-else"} }} {
		next := r
		next.Version++
		next.Status = d.ReviewCancelled
		next.ClosedAt = &now
		next.CloseReason = "admin"
		next.ClosedBy = "admin"
		change(&next)
		if err = d.ValidateReviewCaseUpdate(r, next); err == nil {
			t.Fatal("review target/policy/history rewritten")
		}
	}
}
