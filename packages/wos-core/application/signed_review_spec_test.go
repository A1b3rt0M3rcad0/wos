package application

import (
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"strings"
	"testing"
)

func TestReviewSpecificationExplicitlyReferencesOversizeMaterial(t *testing.T) {
	contract := d.ID("0199a555-0000-7000-8000-000000000001")
	submission := d.ID("0199a555-0000-7000-8000-000000000002")
	raw, err := json.Marshal(map[string]string{"contract_id": contract.String(), "description": strings.Repeat("x", 140*1024)})
	if err != nil {
		t.Fatal(err)
	}
	material := d.SignedResultMaterial{ContractID: contract, WorkItemID: contract, Summary: strings.Repeat("y", 140*1024)}
	payload := signing.SpecPayload[SignedReviewSpec]{Spec: SignedReviewSpec{ReviewCaseID: contract, SubmissionID: submission, SubmissionDigest: "accepted-material-digest", IssuedWorkSpecDigest: signing.Digest(raw), Submission: &material, IssuedWorkSpecification: &signing.Document{Payload: raw}}}
	spec, err := compactSignedReviewSpec(payload)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Submission != nil || spec.IssuedWorkSpecification != nil || len(spec.Omitted) != 2 {
		t.Fatal("oversize material must be replaced with explicit authenticated references")
	}
	if spec.Omitted[0].ContractID != contract || spec.Omitted[0].SubmissionID == nil || *spec.Omitted[0].SubmissionID != submission || spec.Omitted[0].Digest != payload.Spec.SubmissionDigest || spec.Omitted[1].Digest != payload.Spec.IssuedWorkSpecDigest {
		t.Fatal("reference lost its exact target or digest")
	}
	payload.Spec = spec
	encoded, err := json.Marshal(payload)
	if err != nil || len(encoded) > 128*1024 {
		t.Fatal("compact specification exceeded its budget", err)
	}
	material.Summary = "small accepted result"
	payload.Spec.Submission = &material
	payload.Spec.IssuedWorkSpecification = &signing.Document{Payload: json.RawMessage(`{"contract_id":"0199a555-0000-7000-8000-000000000001"}`)}
	payload.Spec.Omitted = nil
	small, err := compactSignedReviewSpec(payload)
	if err != nil || small.Submission == nil || small.IssuedWorkSpecification == nil || len(small.Omitted) != 0 {
		t.Fatal("fitting material was omitted", err)
	}
}
