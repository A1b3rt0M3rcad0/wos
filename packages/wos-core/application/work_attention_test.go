package application

import (
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestAttentionKeepsDeclaredEvidenceSeparateFromCriterionApproval(t *testing.T) {
	now := time.Now().UTC()
	id := d.MustParseID("0199a555-0000-7000-8000-000000000001")
	scope := d.Scope{TenantID: id, OutcomeID: id}
	work, err := d.NewWorkItem(id, scope, "independently verify a delivery", "", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
	if err != nil {
		t.Fatal(err)
	}
	criterion, err := d.NewSuccessCriterion(d.MustParseID("0199a555-0000-7000-8000-000000000002"), work.Ref(), "Tests cover the delivery", "", true, d.VerificationModeEvidenceReview)
	if err != nil {
		t.Fatal(err)
	}
	if err = work.Criteria.Add(criterion); err != nil {
		t.Fatal(err)
	}
	context := WorkContext{Work: work, EvaluatedAt: now, EvidenceLinks: []d.EvidenceLink{{TargetRef: work.Ref(), CriterionID: &criterion.ID, Lifecycle: d.EvidenceLinkLifecycleActive}}, Omitted: map[string]int{}}
	attention := projectWorkAttention(context)
	if len(attention.Obligations) != 1 || !attention.Obligations[0].EvidenceLinked || attention.Obligations[0].Status != "awaiting_assessment" || attention.Obligations[0].AssessmentResult != "" {
		t.Fatalf("evidence silently approved criterion: %+v", attention)
	}
	// Approval for an older criterion revision cannot satisfy today's obligation.
	context.Work.Criteria.CurrentAssessments[criterion.ID] = d.CriterionAssessment{CriterionRevision: criterion.Revision + 1, Result: d.AssessmentResultMet}
	attention = projectWorkAttention(context)
	if attention.Obligations[0].Status != "awaiting_assessment" {
		t.Fatal("assessment inherited across criterion revisions")
	}
	context.Work.Criteria.CurrentAssessments[criterion.ID] = d.CriterionAssessment{CriterionRevision: criterion.Revision, Result: d.AssessmentResultNotMet}
	attention = projectWorkAttention(context)
	if attention.Obligations[0].Status != "not_met" {
		t.Fatal("negative current assessment disappeared")
	}
	context.Work.Criteria.CurrentAssessments[criterion.ID] = d.CriterionAssessment{CriterionRevision: criterion.Revision, Result: d.AssessmentResultMet}
	attention = projectWorkAttention(context)
	if attention.Obligations[0].Status != "satisfied" {
		t.Fatal("explicit current assessment was ignored")
	}
	context.Omitted["evidence_links"] = 1
	if projectWorkAttention(context).EvidenceSearchComplete {
		t.Fatal("partial link context presented as complete evidence search")
	}
	// Reopening or redelivering contracted work must not reuse the previous
	// delivery's approval merely because its criterion definition is unchanged.
	context.Work.ContractsEnabled = true
	if projectWorkAttention(context).Obligations[0].Status != "awaiting_assessment" {
		t.Fatal("contracted work inherited an unbound assessment")
	}
	oldSubmission := d.MustParseID("0199a555-0000-7000-8000-000000000003")
	newSubmission := d.MustParseID("0199a555-0000-7000-8000-000000000004")
	assessment := context.Work.Criteria.CurrentAssessments[criterion.ID]
	assessment.SubmissionID = &oldSubmission
	assessment.SubmissionDigest = "old-material"
	context.Work.Criteria.CurrentAssessments[criterion.ID] = assessment
	context.Contract = &WorkContractView{LatestSubmission: &d.WorkSubmission{ID: newSubmission, Digest: "new-material"}}
	if projectWorkAttention(context).Obligations[0].Status != "awaiting_assessment" {
		t.Fatal("superseding delivery inherited old submission approval")
	}
	context.Contract.LatestSubmission = &d.WorkSubmission{ID: oldSubmission, Digest: "old-material"}
	if projectWorkAttention(context).Obligations[0].Status != "satisfied" {
		t.Fatal("exact current submission assessment ignored")
	}
}

func TestAttentionReviewIsAnObligationWithoutExpiredExecutionAuthority(t *testing.T) {
	id := d.MustParseID("0199a555-0000-7000-8000-000000000001")
	context := WorkContext{Work: d.WorkItem{ID: id, PendingReviewCaseID: &id}, Operational: d.WorkItemOperationalState{ReadinessReasons: []d.ReadinessReason{d.ReadinessReasonLeaseExpired}}, Omitted: map[string]int{}}
	attention := projectWorkAttention(context)
	if !attention.AwaitingReview || attention.Status != "awaiting_review" || attention.ExecutionAllowed || attention.OperationalState.LeaseStatus != d.LeaseStatusNone {
		t.Fatalf("review represented as active execution: %+v", attention)
	}
	for _, obligation := range attention.Obligations {
		if obligation.Description == string(d.ReadinessReasonLeaseExpired) {
			t.Fatal("review waiting requires nonexistent execution lease")
		}
	}
	context.Work.PendingReviewCaseID = nil
	context.Work.CorrectionReviewCaseID = &id
	attention = projectWorkAttention(context)
	if !attention.CorrectionRequired || attention.Status != "corrections_requested" {
		t.Fatal("required correction disappeared after local contract closure")
	}
	context.Work.CorrectionReviewCaseID = nil
	context.Work.Lifecycle = d.WorkItemLifecycleDone
	context.Operational.ReadinessReasons = []d.ReadinessReason{d.ReadinessReasonLifecycleNotReady}
	if len(projectWorkAttention(context).Obligations) != 0 {
		t.Fatal("terminal Task readiness was presented as unfinished work")
	}
}
