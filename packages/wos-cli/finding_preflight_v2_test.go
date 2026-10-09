package woscli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/json"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"strings"
	"testing"
)

func TestInvalidFindingRejectedBeforeReturnFreeze(t *testing.T) {
	for _, both := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing", true: "both"}[both], func(t *testing.T) {
			project, profile, token := profileFixtureV2(t, "executor_a")
			_, file := contractFixtureV2(t)
			id := d.MustParseID("0199ac10-0000-7000-8000-000000000008")
			criterion := d.MustParseID("0199ac10-0000-7000-8000-000000000010")
			finding := d.SignedFinding{ID: criterion, Description: "Actual obligation was not met"}
			if both {
				finding.CriterionID = &criterion
				finding.RequirementRef = "task.description"
			}
			draft := ReviewDraftV2{Decision: "changes_requested", Material: a.SignedReviewMaterial{Findings: []d.SignedFinding{finding}}}
			draftRaw, err := json.Marshal(draft)
			if err != nil {
				t.Fatal(err)
			}
			var decoded ReviewDraftV2
			if err = json.Unmarshal(draftRaw, &decoded); err == nil || !strings.Contains(err.Error(), "exactly one existing obligation") {
				t.Fatalf("expected finding semantic rejection, got %v", err)
			}
			file.Execution = nil
			file.Review = &ReviewDraftV2{Decision: "changes_requested", Material: a.SignedReviewMaterial{Findings: []d.SignedFinding{finding}}}
			w, err := OpenWorkspace(t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			defer w.Close()
			path := contractPathV2(profile.Name, id)
			for name, doc := range map[string]any{".wos/project.yaml": project, profilePathV2(profile.Name): profile, path: file} {
				if err = w.CreateV2(name, doc); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := w.ReadV2(path)
			beforeProfile, _ := w.ReadV2(profilePathV2(profile.Name))
			// A nil client catches any attempted remote preflight; malformed editable
			// material must be rejected without touching immutable local intentions.
			if _, err = prepareReturnV2(context.Background(), w, profile, nil, token, a.SigningIdentityView{}, id, "review", ""); err == nil {
				t.Fatal("invalid finding reached signing")
			}
			after, _ := w.ReadV2(path)
			afterProfile, _ := w.ReadV2(profilePathV2(profile.Name))
			if !bytes.Equal(before, after) || !bytes.Equal(beforeProfile, afterProfile) {
				t.Fatal("invalid finding changed local state")
			}
		})
	}
}

func TestNonApprovalAssessmentsRejectedBeforeFreeze(t *testing.T) {
	for _, decision := range []string{"changes_requested", "inconclusive"} {
		var draft ReviewDraftV2
		input := ReviewDraftV2{Decision: decision, Material: a.SignedReviewMaterial{Reason: "Actual checks", Assessments: []a.SignedAssessmentInput{{CriterionID: d.MustParseID("0199ac10-0000-7000-8000-000000000010"), CriterionRevision: 1, Result: d.AssessmentResultMet, Rationale: "Passed"}}}}
		raw, err := json.Marshal(input)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(raw, &draft); err == nil || !strings.Contains(err.Error(), "non-approval cannot certify criteria") {
			t.Fatalf("expected semantic rejection, got %v", err)
		}
		approved := bytes.Replace(raw, []byte(decision), []byte("approved"), 1)
		if err := json.Unmarshal(approved, &draft); err != nil {
			t.Fatalf("approval assessments unexpectedly rejected: %v", err)
		}
	}
}

func TestReviewRecordsWithoutDeliveredSourceRejectedBeforeFreeze(t *testing.T) {
	project, profile, token := profileFixtureV2(t, "executor_a")
	_, file := contractFixtureV2(t)
	var original signing.SpecPayload[d.SignedWorkSpec]
	if err := json.Unmarshal(file.Issued.Specification.Payload, &original); err != nil {
		t.Fatal(err)
	}
	var authority signing.AuthorityPayload
	if err := json.Unmarshal(file.Issued.Authority.Payload, &authority); err != nil {
		t.Fatal(err)
	}
	authority.ContractKind = "review"
	payload := signing.SpecPayload[a.SignedReviewSpec]{Binding: original.Binding, ContractKind: "review", ContractID: original.ContractID, WorkItemID: original.WorkItemID, Spec: a.SignedReviewSpec{ReviewCaseID: d.ID(original.ContractID), SubmissionID: d.ID(original.ContractID), Submission: &d.SignedResultMaterial{ContractID: d.ID(original.ContractID), WorkItemID: d.ID(original.WorkItemID), Summary: "No delivered immutable artifact"}}}
	doc := func(purpose signing.PayloadType, value any) signing.Document {
		envelope, err := signing.Sign(purpose, value, original.SignerKeyID, ed25519.NewKeyFromSeed(make([]byte, 32)))
		if err != nil {
			_, detail := json.Marshal(value)
			t.Fatalf("purpose %s: %v (serialization %v)", purpose, err, detail)
		}
		document, err := signing.ToDocument(envelope)
		if err != nil {
			t.Fatal(err)
		}
		return document
	}
	payload.Spec.AcceptanceReceipt = doc(signing.AcceptanceReceipt, signing.ReceiptPayload{Binding: original.Binding, ContractID: original.ContractID, WorkItemID: original.WorkItemID, Accepted: true, LocalObligationClosed: true})
	file.Issued.Specification = doc(signing.ContractSpec, payload)
	authority.SpecDigest = signing.Digest(file.Issued.Specification.Payload)
	file.Issued.Authority = doc(signing.ContractAuthority, authority)
	file.Execution = nil
	file.Review = &ReviewDraftV2{Decision: "approved", Material: a.SignedReviewMaterial{ReviewedSourceVersion: "actual-commit", Evidence: []a.SyncEvidenceInput{{LocalKey: "review-tests", Evidence: a.RegisterEvidenceCommand{Scope: d.Scope{NamespaceID: profile.Binding.NamespaceID, OutcomeID: project.Scope.DefaultOutcomeID}, EvidenceType: d.EvidenceTypeTestResult, SourceVersion: "actual-commit"}}}}}
	if _, err := file.VerifyIssued(profile); err != nil {
		t.Fatal("fixture must have valid issued review proofs", err)
	}
	w, err := OpenWorkspace(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	id := d.ID(original.ContractID)
	path := contractPathV2(profile.Name, id)
	for name, value := range map[string]any{".wos/project.yaml": project, profilePathV2(profile.Name): profile, path: file} {
		if err = w.CreateV2(name, value); err != nil {
			t.Fatal(err)
		}
	}
	before, _ := w.ReadV2(path)
	beforeProfile, _ := w.ReadV2(profilePathV2(profile.Name))
	// Nil SDK proves this specific semantic guard precedes all remote work.
	if _, err = prepareReturnV2(context.Background(), w, profile, nil, token, a.SigningIdentityView{}, id, "review", ""); err == nil || !strings.Contains(err.Error(), "immutable reviewed source version/checksum") {
		t.Fatalf("expected source-binding rejection, got %v", err)
	}
	after, _ := w.ReadV2(path)
	afterProfile, _ := w.ReadV2(profilePathV2(profile.Name))
	if !bytes.Equal(before, after) || !bytes.Equal(beforeProfile, afterProfile) {
		t.Fatal("invalid source binding altered immutable local state")
	}
}
