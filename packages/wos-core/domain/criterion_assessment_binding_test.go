package domain_test

import (
	"strings"
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func validBoundAssessment() d.CriterionAssessment {
	return d.CriterionAssessment{
		ID:                d.MustParseID("0199f2b0-0000-7000-8000-000000000001"),
		CriterionID:       d.MustParseID("0199f2b0-0000-7000-8000-000000000002"),
		CriterionRevision: d.InitialCriterionRevision,
		Result:            d.AssessmentResultMet,
		Rationale:         "The exact delivery was reviewed",
		PrincipalID:       "reviewer",
		Actor:             d.ActorRef{Kind: d.ActorKindHuman, Provider: "local", ID: "reviewer"},
		AssessedAt:        time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC),
		TaskDeliveryRef: &d.TaskDeliveryAssessmentRef{
			ID: d.MustParseID("0199f2b0-0000-7000-8000-000000000003"), Digest: "sha256:" + strings.Repeat("a", 64), Round: 1,
		},
	}
}

func TestTaskDeliveryAssessmentRefRequiresExactValidIdentityDigestAndRound(t *testing.T) {
	valid := d.TaskDeliveryAssessmentRef{ID: d.MustParseID("0199f2b0-0000-7000-8000-000000000003"), Digest: "sha256:" + strings.Repeat("b", 64), Round: 1}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid TaskDelivery assessment reference rejected: %v", err)
	}
	for name, mutate := range map[string]func(*d.TaskDeliveryAssessmentRef){
		"invalid id":       func(ref *d.TaskDeliveryAssessmentRef) { ref.ID = "not-an-id" },
		"empty digest":     func(ref *d.TaskDeliveryAssessmentRef) { ref.Digest = "" },
		"malformed digest": func(ref *d.TaskDeliveryAssessmentRef) { ref.Digest = "sha256:" + strings.Repeat("z", 64) },
		"zero round":       func(ref *d.TaskDeliveryAssessmentRef) { ref.Round = 0 },
	} {
		t.Run(name, func(t *testing.T) {
			ref := valid
			mutate(&ref)
			if err := ref.Validate(); err == nil {
				t.Fatal("malformed TaskDelivery reference was accepted")
			}
		})
	}
}

func TestCriterionAssessmentCanBindBothReviewDeliveryAndContractsSubmission(t *testing.T) {
	assessment := validBoundAssessment()
	submissionID := d.MustParseID("0199f2b0-0000-7000-8000-000000000004")
	assessment.SubmissionID = &submissionID
	assessment.SubmissionDigest = "sha256:" + strings.Repeat("c", 64)
	if err := assessment.Validate(); err != nil {
		t.Fatalf("contracts_v1 assessment with separate delivery and submission bindings rejected: %v", err)
	}
}
