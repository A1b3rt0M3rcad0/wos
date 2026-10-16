package woscli

import (
	"context"
	"strings"
	"testing"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

func TestAssessmentShapeRejectedBeforeSigningOrNetwork(t *testing.T) {
	project, profile, token, file, intent, _ := returnFixtureV2(t)
	file.Local.Pending = nil
	profile.Local.PendingOperations = nil
	if e := profile.SealBinding(token); e != nil {
		t.Fatal(e)
	}
	file.Execution.Material.Assessments = []a.SignedAssessmentInput{{CriterionID: d.ID("0199ac10-0000-7000-8000-000000000010"), Result: d.AssessmentResultMet}}
	w, e := OpenWorkspace(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	path := contractPathV2(profile.Name, *intent.ContractID)
	for p, v := range map[string]any{".wos/project.yaml": project, profilePathV2(profile.Name): profile, path: file} {
		if e = w.CreateV2(p, v); e != nil {
			t.Fatal(e)
		}
	}
	before, e := w.ReadV2(path)
	if e != nil {
		t.Fatal(e)
	}
	_, e = prepareReturnV2(context.Background(), w, profile, nil, token, a.SigningIdentityView{}, *intent.ContractID, "execution", "")
	if e == nil || !strings.Contains(e.Error(), "assessment needs") {
		t.Fatalf("invalid assessment reached remote preflight: %v", e)
	}
	after, e := w.ReadV2(path)
	if e != nil {
		t.Fatal(e)
	}
	if string(before) != string(after) {
		t.Fatal("invalid material altered draft")
	}
}

func TestFindingReferenceUsesAuthenticatedOriginalObligations(t *testing.T) {
	profile, file := contractFixtureV2(t)
	view, e := file.VerifyIssued(profile)
	if e != nil {
		t.Fatal(e)
	}
	work := file.Issued.Specification
	view.ReviewSpecification = &signing.SpecPayload[a.SignedReviewSpec]{Spec: a.SignedReviewSpec{IssuedWorkSpecification: &work, IssuedWorkSpecDigest: signing.Digest(work.Payload)}}
	file.Execution = nil
	file.Review = &ReviewDraftV2{Decision: "changes_requested", Material: a.SignedReviewMaterial{Reason: "observed a missing obligation", Findings: []d.SignedFinding{{ID: d.ID("0199ac10-0000-7000-8000-000000000010"), RequirementRef: "uncontracted prose", Description: "actual check"}}}}
	if e = validateReturnDraftV2(profile, file, view); e == nil || !strings.Contains(e.Error(), "existing contracted obligation") {
		t.Fatalf("unplanned finding accepted: %v", e)
	}
	file.Review.Material.Findings[0].RequirementRef = "task.title"
	if e = validateReturnDraftV2(profile, file, view); e != nil {
		t.Fatal(e)
	}
}
