package domain_test

import (
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"math"
	"reflect"
	"testing"
	"time"
)

func contractFixture(t *testing.T) (d.WorkItem, d.WorkContract, time.Time) {
	t.Helper()
	now := time.Date(2026, 10, 8, 12, 0, 0, 0, time.UTC)
	scope := d.Scope{NamespaceID: d.MustParseID("0199f200-0000-7000-8000-000000000001"), OutcomeID: d.MustParseID("0199f200-0000-7000-8000-000000000002")}
	w, err := d.NewWorkItem(d.MustParseID("0199f200-0000-7000-8000-000000000003"), scope, "Contract task", "Independent context", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
	if err != nil {
		t.Fatal(err)
	}
	w.ContractsEnabled = true
	c, err := d.NewWorkContract(d.MustParseID("0199f200-0000-7000-8000-000000000004"), d.MustParseID("0199f200-0000-7000-8000-000000000005"), w, "holder", d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "worker"}, d.DefaultContractLeasePolicy(), 30, d.WorkContractSpec{Title: w.Title, Description: w.Description}, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := w.BindContract(c, now); err != nil {
		t.Fatal(err)
	}
	return w, c, now
}
func TestContractTimeBoundaryAndTerminalCauses(t *testing.T) {
	for _, offset := range []time.Duration{-time.Nanosecond, 0, time.Nanosecond} {
		t.Run(offset.String(), func(t *testing.T) {
			_, c, _ := contractFixture(t)
			instant := c.ExpiresAt.Add(offset)
			err := c.Renew("holder", c.ExecutionID, c.FencingToken, c.LeaseVersion, 30, instant)
			if (err == nil) != (offset < 0) {
				t.Fatalf("boundary renew: %v", err)
			}
		})
	}
	for _, cause := range []d.ContractStatus{d.ContractExpired, d.ContractRevoked, d.ContractCompleted} {
		t.Run(string(cause), func(t *testing.T) {
			w, c, now := contractFixture(t)
			switch cause {
			case d.ContractExpired:
				if err := c.Expire(c.ExpiresAt); err != nil {
					t.Fatal(err)
				}
			case d.ContractRevoked:
				if err := c.Revoke("admin", "explicit intervention", c.Version, now); err != nil {
					t.Fatal(err)
				}
			case d.ContractCompleted:
				p := submissionFixture(t, c, now)
				if err := c.Submit(p, "holder", c.ExecutionID, c.FencingToken, c.Version, now); err != nil {
					t.Fatal(err)
				}
				conclusion := d.Conclusion{PrincipalID: "holder", Actor: c.Actor, Reason: p.Material.Summary, ConcludedAt: now}
				if err := c.Finalize(&w, p, "holder", c.ExecutionID, c.FencingToken, c.Version, w.Version, conclusion, now); err != nil {
					t.Fatal(err)
				}
			}
			if c.Status != cause {
				t.Fatal(c.Status)
			}
			before := c
			if err := c.Renew("holder", c.ExecutionID, c.FencingToken, c.LeaseVersion, 30, now); err == nil {
				t.Fatal("terminal renewal accepted")
			}
			if err := c.Revoke("admin", "rewrite cause", c.Version, now); err == nil {
				t.Fatal("terminal revoke accepted")
			}
			if !reflect.DeepEqual(before, c) {
				t.Fatal("rejected operation changed terminal record")
			}
			if err := c.Validate(); err != nil {
				t.Fatal(err)
			}
			if cause != d.ContractCompleted && w.Lifecycle != d.WorkItemLifecycleInProgress {
				t.Fatal("expiry/revoke altered work lifecycle")
			}
		})
	}
}
func submissionFixture(t *testing.T, c d.WorkContract, now time.Time) d.WorkSubmission {
	t.Helper()
	m := d.NormalizeResultMaterial(d.WorkResultMaterial{ContractID: c.ID, WorkItemID: c.WorkItemID, SpecDigest: c.SpecDigest, Summary: "Verified result"})
	digest, err := d.SemanticDigest(m)
	if err != nil {
		t.Fatal(err)
	}
	return d.WorkSubmission{ID: d.MustParseID("0199f200-0000-7000-8000-000000000006"), Scope: c.Scope, Material: m, Digest: digest, PrincipalID: c.HolderPrincipalID, Actor: c.Actor, SubmittedAt: now}
}
func TestContractTakeoverAndSeparateVersions(t *testing.T) {
	w, c, now := contractFixture(t)
	original := c
	newID := d.MustParseID("0199f200-0000-7000-8000-000000000007")
	if err := c.Resume("other", c.ExecutionID, c.FencingToken, c.LeaseVersion, newID, w.LastFencingToken, now); err == nil {
		t.Fatal("nonholder takeover accepted")
	}
	if err := c.Resume("holder", c.ExecutionID, c.FencingToken, c.LeaseVersion, newID, w.LastFencingToken, now); err != nil {
		t.Fatal(err)
	}
	if c.Version != original.Version || c.LeaseVersion != original.LeaseVersion+1 || c.FencingToken <= original.FencingToken || !c.ExpiresAt.Equal(original.ExpiresAt) {
		t.Fatal("takeover changed wrong authority/version")
	}
	if err := c.Authorize("holder", original.ExecutionID, original.FencingToken, now); err == nil {
		t.Fatal("stale generation accepted")
	}
	if err := c.Renew("holder", c.ExecutionID, c.FencingToken, c.LeaseVersion, 300, now); err != nil {
		t.Fatal(err)
	}
	if c.Version != original.Version || c.SpecDigest != original.SpecDigest {
		t.Fatal("renew modified progress/spec")
	}
}
func TestContractNoLegacyReleaseOrOverride(t *testing.T) {
	w, c, now := contractFixture(t)
	for _, call := range []func() error{func() error { return w.Release("holder", c.ID, uint64(c.FencingToken), now) }, func() error { return w.Claim(c.ID, "holder", c.Actor, 0, now) }, func() error { return w.CompleteAdministratively("bypass", d.Conclusion{}, now) }, func() error { return w.CancelAdministratively(d.Conclusion{}, now) }, func() error { _, err := w.UpdateDetails(nil, nil, nil, now); return err }} {
		if err := call(); err == nil {
			t.Fatal("legacy bypass accepted")
		}
	}
}
func TestFencingExactUnsignedAndOverflow(t *testing.T) {
	for _, value := range []uint64{1, 9007199254740993, math.MaxUint64} {
		f := d.FencingToken(value)
		b, err := json.Marshal(f)
		if err != nil {
			t.Fatal(err)
		}
		var decoded d.FencingToken
		if err := json.Unmarshal(b, &decoded); err != nil || decoded != f {
			t.Fatalf("roundtrip %s %v", b, err)
		}
	}
	for _, raw := range []string{`1`, `"01"`, `"+1"`, `"0"`, `"18446744073709551616"`} {
		var f d.FencingToken
		if err := json.Unmarshal([]byte(raw), &f); err == nil {
			t.Fatalf("accepted %s", raw)
		}
	}
	if _, err := d.NextFencing(math.MaxUint64); err == nil {
		t.Fatal("contract overflow accepted")
	}
	w, _, now := contractFixture(t)
	w.ContractsEnabled = false
	w.CurrentContractID = nil
	w.Lifecycle = d.WorkItemLifecycleTodo
	w.LastFencingToken = math.MaxUint64
	if err := w.Claim(d.MustParseID("0199f200-0000-7000-8000-000000000008"), "holder", d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "worker"}, 0, now); err == nil || w.LastFencingToken != math.MaxUint64 {
		t.Fatal("initial legacy claim wrapped fencing")
	}
}
func TestJCSDigestInteroperableOrdering(t *testing.T) {
	left := json.RawMessage(`{"\u20ac":1,"\r":2,"\ufb33":3,"1":4,"\ud83d\ude00":5,"\u0080":6,"\u00f6":7}`)
	right := json.RawMessage("{\"\\r\":2,\"1\":4,\"\u0080\":6,\"ö\":7,\"€\":1,\"😀\":5,\"דּ\":3}")
	a, err := d.SemanticDigest(left)
	if err != nil {
		t.Fatal(err)
	}
	b, err := d.SemanticDigest(right)
	if err != nil || a != b {
		t.Fatalf("RFC8785 UTF16 ordering mismatch %s %s %v", a, b, err)
	}
	a, _ = d.SemanticDigest(json.RawMessage(`{"n":1e+30,"s":"<>&","a":4.50}`))
	b, _ = d.SemanticDigest(json.RawMessage(`{"a":4.5,"s":"\u003c\u003e\u0026","n":1000000000000000000000000000000}`))
	if a != b {
		t.Fatal("JCS number/string equivalence mismatch")
	}
}
func TestSubmissionDoesNotCompleteAndDigestCannotBeForged(t *testing.T) {
	w, c, now := contractFixture(t)
	p := submissionFixture(t, c, now)
	if err := c.Submit(p, "holder", c.ExecutionID, c.FencingToken, c.Version, now); err != nil {
		t.Fatal(err)
	}
	if c.Status != d.ContractActive || w.Lifecycle != d.WorkItemLifecycleInProgress {
		t.Fatal("submit completed work")
	}
	p.Material.Summary = "changed"
	if err := p.Validate(); err == nil {
		t.Fatal("modified submission accepted old digest")
	}
	c.Spec.Title = "changed"
	if err := c.Validate(); err == nil {
		t.Fatal("spec mutation accepted old digest")
	}
}
