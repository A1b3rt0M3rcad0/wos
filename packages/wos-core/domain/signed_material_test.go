package domain_test

import (
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"math"
	"strings"
	"testing"
)

func TestSignedMaterialExactCountersAndIndependentFingerprints(t *testing.T) {
	w, c, now := contractFixture(t)
	criterion, err := d.NewSuccessCriterion(d.MustParseID("0199f202-0000-7000-8000-000000000001"), w.Ref(), "max revision", "", true, d.VerificationModeEvidenceReview)
	if err != nil {
		t.Fatal(err)
	}
	criterion.Revision = d.CriterionRevision(math.MaxUint64)
	c.Spec.Criteria = []d.SuccessCriterion{criterion}
	b := reviewBinding()
	b.PolicyRevision = d.Version(math.MaxUint64)
	wire := d.NewSignedWorkSpec(c.Spec, b)
	raw, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"criterion_revision":"18446744073709551615"`) || !strings.Contains(string(raw), `"policy_revision":"18446744073709551615"`) {
		t.Fatal("signed specification lost exact counter wire form")
	}
	var decoded d.SignedWorkSpec
	if err = json.Unmarshal(raw, &decoded); err != nil || decoded.Criteria[0].Revision != criterion.Revision {
		t.Fatal("exact revision round trip failed", err)
	}
	material := d.NormalizeResultMaterial(d.WorkResultMaterial{ContractID: c.ID, WorkItemID: w.ID, SpecDigest: c.SpecDigest, Summary: "max revision", CriterionEvidence: []d.SubmissionCriterionEvidence{{CriterionID: criterion.ID, CriterionRevision: criterion.Revision, EvidenceIDs: []d.ID{}}}})
	first, err := d.SignedResultDigest(material)
	if err != nil {
		t.Fatal(err)
	}
	material.CriterionEvidence[0].CriterionRevision--
	second, err := d.SignedResultDigest(material)
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("signed material collapsed adjacent uint64 revisions")
	}
	submission := d.WorkSubmission{ProtocolVersion: 2, ID: d.MustParseID("0199f202-0000-7000-8000-000000000002"), Scope: w.Scope, Material: material, Digest: second, PrincipalID: "holder", Actor: c.Actor, SubmittedAt: now}
	if err = submission.Validate(); err != nil {
		t.Fatal(err)
	}
	submission.ProtocolVersion = 0
	if err = submission.Validate(); err == nil {
		t.Fatal("new digest was accepted as a legacy fingerprint")
	}
}
