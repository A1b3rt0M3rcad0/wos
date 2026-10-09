package woscli

import (
	"bytes"
	"context"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
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
		raw := []byte(`{"decision":"` + decision + `","progress":{},"material":{"reason":"Actual checks","criterion_assessments":[{"criterion_id":"0199ac10-0000-7000-8000-000000000010","criterion_revision":"1","result":"met","rationale":"Passed"}]}}`)
		if err := DecodeV2Document(raw, &draft); err == nil {
			t.Fatal("non-approval could freeze criterion assessments")
		}
	}
}
