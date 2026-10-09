package woscli

import (
	"bytes"
	"context"
	"encoding/json"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"testing"
)

func TestSignedEvidenceMismatchRejectedBeforeFreeze(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	project, profile, token := profileFixtureV2(t, "executor_a")
	_, file := contractFixtureV2(t)
	file.Execution.CompletionIntent = "auto"
	file.Execution.Material.Evidence = []a.SyncEvidenceInput{{LocalKey: "actual-tests", Evidence: a.RegisterEvidenceCommand{Scope: d.Scope{NamespaceID: profile.Binding.NamespaceID, OutcomeID: project.Scope.DefaultOutcomeID}, EvidenceType: d.EvidenceTypeTestResult, Measurement: &d.Measurement{Value: "5"}}}}
	w, e := OpenWorkspace(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	for path, document := range map[string]any{".wos/project.yaml": project, profilePathV2(profile.Name): profile} {
		if e = w.CreateV2(path, document); e != nil {
			t.Fatal(e)
		}
	}
	view, e := file.VerifyIssued(profile)
	if e != nil {
		t.Fatal(e)
	}
	id := d.ID(view.Authority.ContractID)
	path := contractPathV2(profile.Name, id)
	if e = w.CreateV2(path, file); e != nil {
		t.Fatal(e)
	}
	before, e := w.ReadV2(path)
	if e != nil {
		t.Fatal(e)
	}
	profileBefore, e := w.ReadV2(profilePathV2(profile.Name))
	if e != nil {
		t.Fatal(e)
	}
	// No SDK client: malformed editable material must fail before any network
	// preflight, signing or durable intent can occur.
	if _, e = prepareReturnV2(context.Background(), w, profile, nil, token, a.SigningIdentityView{}, id, "execution", ""); e == nil {
		t.Fatal("mismatched evidence reached immutable return preparation")
	}
	after, _ := w.ReadV2(path)
	profileAfter, _ := w.ReadV2(profilePathV2(profile.Name))
	if !bytes.Equal(before, after) || !bytes.Equal(profileBefore, profileAfter) {
		t.Fatal("invalid draft froze or modified original local state")
	}
}

func TestExecutionAndReviewEvidenceMeasurementSemantics(t *testing.T) {
	profile, file := contractFixtureV2(t)
	view, e := file.VerifyIssued(profile)
	if e != nil {
		t.Fatal(e)
	}
	scope := d.Scope{NamespaceID: profile.Binding.NamespaceID, OutcomeID: d.ID(view.Authority.OutcomeID)}
	for _, review := range []bool{false, true} {
		for _, input := range []struct {
			kind        d.EvidenceType
			measurement *d.Measurement
			valid       bool
		}{
			{d.EvidenceTypeTestResult, nil, true},
			{d.EvidenceTypeMeasurement, &d.Measurement{Value: "5", Unit: "passed tests"}, true},
			{d.EvidenceTypeTestResult, &d.Measurement{Value: "5"}, false},
			{d.EvidenceTypeMeasurement, nil, false},
			{d.EvidenceTypeMeasurement, &d.Measurement{}, false},
		} {
			evidence := []a.SyncEvidenceInput{{LocalKey: "tests", Evidence: a.RegisterEvidenceCommand{Scope: scope, EvidenceType: input.kind, Measurement: input.measurement}}}
			execution := *file.Execution
			execution.Material.Evidence = evidence
			var document any = execution
			if review {
				document = ReviewDraftV2{Material: a.SignedReviewMaterial{Evidence: evidence}}
			}
			raw, e := json.Marshal(document)
			if e != nil {
				t.Fatal(e)
			}
			if review {
				var target ReviewDraftV2
				e = json.Unmarshal(raw, &target)
			} else {
				var target ExecutionDraftV2
				e = json.Unmarshal(raw, &target)
			}
			if (e == nil) != input.valid {
				t.Fatalf("review=%t kind=%s valid=%t: %v", review, input.kind, input.valid, e)
			}
		}
	}
}
