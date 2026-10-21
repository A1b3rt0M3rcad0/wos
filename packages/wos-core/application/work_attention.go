package application

import (
	"context"
	"encoding/json"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type WorkObligation struct {
	Kind              string              `json:"kind"`
	Description       string              `json:"description"`
	Status            string              `json:"status"`
	CriterionID       *d.ID               `json:"criterion_id,omitempty"`
	CriterionRevision d.CriterionRevision `json:"criterion_revision,omitempty,string"`
	AssessmentResult  d.AssessmentResult  `json:"assessment_result,omitempty"`
	EvidenceLinked    bool                `json:"evidence_linked"`
	ReasonCode        string              `json:"reason_code,omitempty"`
}

// WorkAttention derives current obligations; it never stores a second pending list
// or equates a holder's declaration with an independent assessment.
type WorkAttention struct {
	CurrentContractID      *d.ID                      `json:"current_contract_id,omitempty"`
	ReviewCaseID           *d.ID                      `json:"review_case_id,omitempty"`
	Verification           *TaskVerificationView      `json:"verification,omitempty"`
	CorrectionReviewCaseID *d.ID                      `json:"correction_review_case_id,omitempty"`
	WorkItemID             d.ID                       `json:"work_item_id"`
	Status                 string                     `json:"status"`
	OperationalState       d.WorkItemOperationalState `json:"operational_state"`
	ExecutionAllowed       bool                       `json:"execution_allowed"`
	AwaitingReview         bool                       `json:"awaiting_review"`
	CorrectionRequired     bool                       `json:"correction_required"`
	Obligations            []WorkObligation           `json:"obligations"`
	CorrectionFindings     []d.SignedFinding          `json:"correction_findings,omitempty"`
	EvidenceSearchComplete bool                       `json:"evidence_search_complete"`
	Truncated              bool                       `json:"truncated"`
	Omitted                map[string]int             `json:"omitted,omitempty"`
	OutcomeRevision        d.OutcomeRevision          `json:"outcome_revision,string"`
	EvaluatedAt            time.Time                  `json:"evaluated_at"`
}

func (s *Service) GetWorkAttention(ctx context.Context, scope d.Scope, id d.ID) (WorkAttention, error) {
	context, err := s.GetWorkContext(ctx, scope, id)
	if err != nil {
		return WorkAttention{}, err
	}
	value := projectWorkAttention(context)
	raw, err := json.Marshal(value)
	if err != nil {
		return WorkAttention{}, err
	}
	if len(raw) > MaxSnapshotBytes {
		return WorkAttention{}, d.NewError(d.ErrorCodeGraphLimitExceeded, "work attention exceeds byte limit; expand current criteria separately")
	}
	return value, nil
}

func projectWorkAttention(context WorkContext) WorkAttention {
	w := context.Work
	v := WorkAttention{CurrentContractID: w.CurrentContractID, Verification: context.Verification, ReviewCaseID: w.PendingReviewCaseID, CorrectionReviewCaseID: w.CorrectionReviewCaseID, WorkItemID: w.ID, Status: string(context.Operational.DisplayState), OperationalState: context.Operational, Obligations: []WorkObligation{}, EvidenceSearchComplete: context.Omitted["evidence_links"] == 0, OutcomeRevision: context.OutcomeRevision, EvaluatedAt: context.EvaluatedAt, Omitted: map[string]int{}}
	if context.Verification != nil {
		v.AwaitingReview = context.Verification.Pending
		v.CorrectionRequired = context.Verification.CorrectionRequired
	} else {
		// The aggregate's pending/correction navigation is still meaningful when
		// the richer verification projection is unavailable. In particular, a
		// reviewer handoff does not depend on an executor lease remaining active.
		v.AwaitingReview = w.PendingReviewCaseID != nil
		v.CorrectionRequired = w.CorrectionReviewCaseID != nil
	}
	if context.Contract != nil {
		v.ExecutionAllowed = context.Contract.ExecutionAllowed
		v.Status = string(context.Contract.EffectiveStatus)
		if context.Contract.Contract.SignedBinding != nil {
			v.CorrectionFindings = append([]d.SignedFinding{}, context.Contract.Contract.SignedBinding.CorrectionFindings...)
		}
		// Contract authority replaces the legacy WorkLease projection.
		v.OperationalState.ReadinessReasons = append([]d.ReadinessReason{}, context.Contract.Reasons...)
		v.OperationalState.LeaseStatus = d.LeaseStatusExpired
		if context.Contract.Contract.ValidAt(context.EvaluatedAt) {
			v.OperationalState.LeaseStatus = d.LeaseStatusActive
		}
		if v.ExecutionAllowed {
			v.OperationalState.DisplayState = d.WorkItemDisplayStateInProgress
		}
	}
	if context.Verification != nil {
		v.OperationalState = applyTaskVerificationOperational(v.OperationalState, context.Verification)
		v.Status = string(v.OperationalState.DisplayState)
		if v.AwaitingReview || taskVerificationCorrectionPending(*context.Verification) {
			v.ExecutionAllowed = false
		}
	}
	if v.AwaitingReview {
		v.Status = "awaiting_review"
		v.OperationalState.LeaseStatus = d.LeaseStatusNone
		currentReasons := make([]d.ReadinessReason, 0, len(v.OperationalState.ReadinessReasons))
		for _, reason := range v.OperationalState.ReadinessReasons {
			if reason != d.ReadinessReasonLeaseExpired {
				currentReasons = append(currentReasons, reason)
			}
		}
		v.OperationalState.ReadinessReasons = currentReasons
		v.Obligations = append(v.Obligations, WorkObligation{Kind: "review", Description: "An independent reviewer must evaluate the submitted result.", Status: "awaiting_independent_review"})
	} else if v.CorrectionRequired {
		if w.Lifecycle != d.WorkItemLifecycleInProgress {
			v.Status = "corrections_requested"
		}
		v.Obligations = append(v.Obligations, WorkObligation{Kind: "correction", Description: "Acquire correction authority and address the review findings.", Status: "corrections_requested"})
	}
	if w.Lifecycle == d.WorkItemLifecycleDone || w.Lifecycle == d.WorkItemLifecycleCancelled {
		v.Status, v.AwaitingReview, v.CorrectionRequired, v.ExecutionAllowed = string(w.Lifecycle), false, false, false
		v.OperationalState.DisplayState = d.WorkItemDisplayState(w.Lifecycle)
	}
	if !v.EvidenceSearchComplete {
		v.Truncated = true
		v.Omitted["evidence_links"] = context.Omitted["evidence_links"]
	}
	for _, reason := range v.OperationalState.ReadinessReasons {
		if reason == d.ReadinessReasonLeaseExpired && v.AwaitingReview {
			continue
		}
		if w.Lifecycle == d.WorkItemLifecycleDone || w.Lifecycle == d.WorkItemLifecycleCancelled {
			continue
		}
		v.Obligations = append(v.Obligations, WorkObligation{Kind: "readiness", Description: string(reason), Status: "requires_attention"})
	}
	linked := map[d.ID]bool{}
	for _, relation := range context.Dependencies {
		if d.IsOwnObjectiveHardDependency(w, relation) {
			v.Obligations = append(v.Obligations, WorkObligation{Kind: "structure", ReasonCode: "own_objective_hard_dependency", Description: "This Task has a hard dependency on its own Objective. Move the Task or explicitly remove the dependency; membership does not require a dependency.", Status: "requires_attention"})
			break
		}
	}
	for _, link := range context.EvidenceLinks {
		if link.TargetRef == w.Ref() && link.CriterionID != nil && link.Lifecycle == d.EvidenceLinkLifecycleActive {
			linked[*link.CriterionID] = true
		}
	}
	for _, c := range w.Criteria.Items {
		if !c.Required || c.Status != d.CriterionStatusActive {
			continue
		}
		id := c.ID
		obligation := WorkObligation{Kind: "criterion", Description: c.Title, Status: "awaiting_assessment", CriterionID: &id, CriterionRevision: c.Revision, EvidenceLinked: linked[c.ID]}
		if w.Lifecycle == d.WorkItemLifecycleCancelled {
			obligation.Status = "not_applicable"
			v.Obligations = append(v.Obligations, obligation)
			continue
		}
		if assessment, ok := w.Criteria.CurrentAssessments[c.ID]; ok && assessment.CriterionRevision == c.Revision && attentionAssessmentApplies(context, assessment) {
			obligation.AssessmentResult = assessment.Result
			switch assessment.Result {
			case d.AssessmentResultMet, d.AssessmentResultWaived:
				obligation.Status = "satisfied"
			default:
				obligation.Status = string(assessment.Result)
			}
		}
		v.Obligations = append(v.Obligations, obligation)
	}
	if context.Contract != nil {
		for _, deliverable := range context.Contract.Contract.Spec.ExecutionSpec.Deliverables {
			v.Obligations = append(v.Obligations, WorkObligation{Kind: "deliverable", Description: deliverable, Status: "requires_declared_result"})
		}
	}
	// Complete current criterion state is kept; only descriptive obligations are
	// bounded. A caller must expand when omitted, never infer absence of duties.
	if len(v.Obligations) > 100 {
		v.Omitted["obligations"] = len(v.Obligations) - 100
		v.Obligations = v.Obligations[:100]
		v.Truncated = true
	}
	if len(v.CorrectionFindings) > 25 {
		v.Omitted["correction_findings"] = len(v.CorrectionFindings) - 25
		v.CorrectionFindings = v.CorrectionFindings[:25]
		v.Truncated = true
	}
	return v
}

// A current criterion revision alone does not approve a fresh execution or a
// superseding delivery. Contract work must bind proof to its current submission.
func attentionAssessmentApplies(context WorkContext, assessment d.CriterionAssessment) bool {
	if !verificationAssessmentApplies(context.Verification, assessment) {
		return false
	}
	if fact, ok := context.workspaceCriterionFacts[assessment.CriterionID]; ok {
		return fact.AssessmentApplicable
	}
	if !context.Work.ContractsEnabled && context.Contract == nil {
		return true
	}
	var submissionID *d.ID
	var digest string
	if context.Contract != nil && context.Contract.LatestSubmission != nil {
		submissionID = &context.Contract.LatestSubmission.ID
		digest = context.Contract.LatestSubmission.Digest
	} else if context.Work.Lifecycle == d.WorkItemLifecycleDone && context.Work.CurrentConclusion != nil {
		submissionID = context.Work.CurrentConclusion.SubmissionID
		digest = context.Work.CurrentConclusion.SubmissionDigest
	}
	return submissionID != nil && assessment.SubmissionID != nil && *assessment.SubmissionID == *submissionID && digest != "" && assessment.SubmissionDigest == digest
}
