package application

import (
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"testing"
)

func TestProofProgressDoesNotTreatWaiverOrRetractedEvidenceAsProof(t *testing.T) {
	id := func(n string) domain.ID { return domain.MustParseID("0199d098-0000-7000-8000-" + n) }
	evidenceID := id("000000000010")
	set := domain.CriterionSet{CurrentAssessments: map[domain.ID]domain.CriterionAssessment{}}
	for i, c := range []struct {
		required bool
		revision domain.CriterionRevision
		result   domain.AssessmentResult
		proof    bool
	}{{true, 1, domain.AssessmentResultMet, true}, {true, 1, domain.AssessmentResultWaived, false}, {true, 2, domain.AssessmentResultMet, false}, {false, 1, domain.AssessmentResultMet, false}} {
		criterionID := []domain.ID{id("000000000001"), id("000000000002"), id("000000000003"), id("000000000004")}[i]
		set.Items = append(set.Items, domain.SuccessCriterion{ID: criterionID, Required: c.required, Revision: c.revision, Status: domain.CriterionStatusActive})
		assessment := domain.CriterionAssessment{CriterionID: criterionID, CriterionRevision: 1, Result: c.result}
		if c.proof {
			assessment.EvidenceIDs = []domain.ID{evidenceID}
		}
		set.CurrentAssessments[criterionID] = assessment
	}
	state := OutcomeState{Outcome: domain.Outcome{Criteria: set}, Evidence: []domain.Evidence{{ID: evidenceID, Lifecycle: domain.EvidenceLifecycleRegistered}}}
	got := proofProgress(state)
	if got["required_criteria_met"].Numerator != 1 || got["required_criteria_met"].Denominator != 3 || got["required_criteria_waived"].Numerator != 1 {
		t.Fatalf("waiver, old revision or optional criterion counted as proof: %+v", got)
	}
	state.Evidence[0].Lifecycle = domain.EvidenceLifecycleRetracted
	got = proofProgress(state)
	if got["required_criteria_met"].Numerator != 0 || got["required_criteria_waived"].Numerator != 1 {
		t.Fatalf("retracted evidence remained proven: %+v", got)
	}
	empty := proofProgress(OutcomeState{})["required_criteria_met"]
	if empty.Value != nil || empty.Denominator != 0 {
		t.Fatal("empty obligations produced a completion percentage")
	}
}
