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

func TestDirectCompletionMustRespectIssuedAcceptanceFloor(t *testing.T) {
	profile, file := contractFixtureV2WithFloor(t, d.AcceptanceIndependentReview)
	view, err := file.VerifyIssued(profile)
	if err != nil {
		t.Fatal(err)
	}
	if view.Authority.AcceptanceMode != string(d.AcceptanceIndependentReview) {
		t.Fatal("review-floor authority did not verify from its issued signature")
	}
	issuedAuthorityBytes, err := signing.Canonical(file.Issued.Authority)
	if err != nil {
		t.Fatal(err)
	}
	file.Execution.CompletionIntent = "direct"
	if err = validateReturnDraftV2(profile, file, view); err == nil || !strings.Contains(err.Error(), "requires independent review") {
		t.Fatalf("direct intent below the issued review floor was not rejected locally: %v", err)
	}
	file.Execution.CompletionIntent = "auto"
	if err = validateReturnDraftV2(profile, file, view); err != nil {
		t.Fatalf("auto intent should be allowed to resolve to a real review: %v", err)
	}
	directProfile, directFile := contractFixtureV2(t)
	directView, err := directFile.VerifyIssued(directProfile)
	if err != nil {
		t.Fatal(err)
	}
	directFile.Execution.CompletionIntent = "direct"
	if err = validateReturnDraftV2(directProfile, directFile, directView); err != nil {
		t.Fatalf("direct intent should remain available for an issued direct floor: %v", err)
	}
	issuedAuthorityAfter, err := signing.Canonical(file.Issued.Authority)
	if err != nil || string(issuedAuthorityAfter) != string(issuedAuthorityBytes) {
		t.Fatal("local intent validation rewrote frozen issued authority bytes", err)
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
