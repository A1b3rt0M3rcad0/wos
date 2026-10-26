package application

import (
	"errors"
	"sort"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// ObjectiveCompletionReport contains factual obligations read from one
// authenticated workspace frame. It does not grant command authority: scoped
// read access and factual eligibility are deliberately separate from the
// caller's permission to achieve the Objective.
type ObjectiveCompletionReport struct {
	Objective                      d.EntityRef                         `json:"objective"`
	ObjectiveVersion               d.Version                           `json:"objective_version,string"`
	OutcomeRevision                d.OutcomeRevision                   `json:"outcome_revision,string"`
	EvaluatedAt                    time.Time                           `json:"evaluated_at"`
	RequiredForOutcome             bool                                `json:"required_for_outcome"`
	CompletionMode                 d.ObjectiveCompletionMode           `json:"completion_mode"`
	CompletionCycle                uint64                              `json:"completion_cycle,string"`
	CompletionPolicy               d.CompletionPolicy                  `json:"completion_policy"`
	FactsComplete                  bool                                `json:"facts_complete"`
	FactualEligible                bool                                `json:"factual_eligible"`
	EligibilityReasonCodes         []string                            `json:"eligibility_reason_codes"`
	Authorization                  ObjectiveCompletionAuthorization    `json:"authorization"`
	DescendantObjectives           ObjectiveCompletionLifecycleCounts  `json:"descendant_objectives"`
	Tasks                          ObjectiveCompletionTaskCounts       `json:"tasks"`
	StrictWork                     ObjectiveCompletionStrictWork       `json:"strict_work"`
	Criteria                       ObjectiveCompletionCriteriaCounts   `json:"criteria"`
	PendingIssues                  ObjectiveCompletionIssueCounts      `json:"pending_issues"`
	HardDependencies               ObjectiveCompletionDependencyCounts `json:"hard_dependencies"`
	ActiveBlockers                 ObjectiveCompletionBlockerCounts    `json:"active_blockers"`
	CycleBoundAssessmentsAvailable bool                                `json:"cycle_bound_assessments_available"`
}

// objectiveCompletionEvaluation keeps the complete facts used by guarded
// mutations alongside the bounded public report. In particular, manual
// conclusion acknowledgements must be checked against every cancelled child,
// never against the report's sample.
type objectiveCompletionEvaluation struct {
	Report          ObjectiveCompletionReport
	StrictOpen      []d.EntityRef
	StrictCancelled []d.EntityRef
	Issues          objectiveIssueReport
}

// ObjectiveCompletionAuthorization makes the boundary explicit. A successful
// report read authenticates access to these facts, not permission to execute
// AchieveObjective. The command path remains responsible for authorization.
type ObjectiveCompletionAuthorization struct {
	Status string `json:"status"`
	Reason string `json:"reason"`
}

type ObjectiveCompletionRefs struct {
	Total   int           `json:"total"`
	Samples []d.EntityRef `json:"samples"`
}

type ObjectiveCompletionLifecycleCounts struct {
	Total      int                     `json:"total"`
	Planned    int                     `json:"planned"`
	InProgress int                     `json:"in_progress"`
	Achieved   int                     `json:"achieved"`
	Cancelled  int                     `json:"cancelled"`
	Open       ObjectiveCompletionRefs `json:"open"`
}

type ObjectiveCompletionTaskCounts struct {
	Total                      int                                  `json:"total"`
	Backlog                    int                                  `json:"backlog"`
	Todo                       int                                  `json:"todo"`
	InProgress                 int                                  `json:"in_progress"`
	Done                       int                                  `json:"done"`
	Cancelled                  int                                  `json:"cancelled"`
	Open                       ObjectiveCompletionRefs              `json:"open"`
	AwaitingVerification       ObjectiveCompletionRefs              `json:"awaiting_verification"`
	PendingReview              ObjectiveCompletionRefs              `json:"pending_review"`
	CorrectionRequired         ObjectiveCompletionRefs              `json:"correction_required"`
	ActiveAuthority            ObjectiveCompletionRefs              `json:"active_authority"`
	IncompleteCriteria         int                                  `json:"incomplete_criteria"`
	IncompleteCriterionSamples []ObjectiveCompletionCriterionSample `json:"incomplete_criterion_samples"`
}

type ObjectiveCompletionStrictWork struct {
	Applies                 bool                    `json:"applies"`
	Open                    ObjectiveCompletionRefs `json:"open"`
	Cancelled               ObjectiveCompletionRefs `json:"cancelled"`
	AcknowledgementRequired bool                    `json:"acknowledgement_required"`
}

type ObjectiveCompletionCriteriaCounts struct {
	ActiveRequired int                                  `json:"active_required"`
	Met            int                                  `json:"met"`
	Waived         int                                  `json:"waived"`
	Pending        int                                  `json:"pending"`
	ProofInvalid   int                                  `json:"proof_invalid"`
	Samples        []ObjectiveCompletionCriterionSample `json:"pending_samples"`
}

type ObjectiveCompletionCriterionSample struct {
	OwnerRef          d.EntityRef         `json:"owner_ref"`
	CriterionID       d.ID                `json:"criterion_id"`
	CriterionRevision d.CriterionRevision `json:"criterion_revision,string"`
	State             string              `json:"state"`
}

type ObjectiveCompletionIssueCounts struct {
	Pending  ObjectiveCompletionRefs `json:"pending"`
	Complete bool                    `json:"complete"`
}

type ObjectiveCompletionDependencyCounts struct {
	Total       int                     `json:"total"`
	Satisfied   int                     `json:"satisfied"`
	Unsatisfied ObjectiveCompletionRefs `json:"unsatisfied"`
}

type ObjectiveCompletionBlockerCounts struct {
	Total    int                     `json:"total"`
	Blockers ObjectiveCompletionRefs `json:"blockers"`
}

func newObjectiveCompletionReport(objective d.Objective, revision d.OutcomeRevision, at time.Time, sampleLimit int, policy d.CompletionPolicy) ObjectiveCompletionReport {
	return ObjectiveCompletionReport{
		Objective: objective.Ref(), ObjectiveVersion: objective.Version,
		OutcomeRevision: revision, EvaluatedAt: at,
		RequiredForOutcome: objective.RequiredForOutcome,
		CompletionMode:     objective.CompletionMode.Effective(),
		CompletionCycle:    objective.EffectiveCompletionCycle(),
		CompletionPolicy:   policy.Effective(), FactsComplete: true,
		EligibilityReasonCodes:         []string{},
		Authorization:                  ObjectiveCompletionAuthorization{Status: "not_evaluated", Reason: "read access does not imply permission to achieve this Objective"},
		DescendantObjectives:           ObjectiveCompletionLifecycleCounts{Open: emptyCompletionRefs()},
		Tasks:                          ObjectiveCompletionTaskCounts{Open: emptyCompletionRefs(), AwaitingVerification: emptyCompletionRefs(), PendingReview: emptyCompletionRefs(), CorrectionRequired: emptyCompletionRefs(), ActiveAuthority: emptyCompletionRefs(), IncompleteCriterionSamples: []ObjectiveCompletionCriterionSample{}},
		StrictWork:                     ObjectiveCompletionStrictWork{Applies: policy.Effective() == d.CompletionPolicyStrictWork, Open: emptyCompletionRefs(), Cancelled: emptyCompletionRefs()},
		Criteria:                       ObjectiveCompletionCriteriaCounts{Samples: []ObjectiveCompletionCriterionSample{}},
		PendingIssues:                  ObjectiveCompletionIssueCounts{Pending: emptyCompletionRefs(), Complete: false},
		HardDependencies:               ObjectiveCompletionDependencyCounts{Unsatisfied: emptyCompletionRefs()},
		ActiveBlockers:                 ObjectiveCompletionBlockerCounts{Blockers: emptyCompletionRefs()},
		CycleBoundAssessmentsAvailable: false,
	}
}

func emptyCompletionRefs() ObjectiveCompletionRefs {
	return ObjectiveCompletionRefs{Samples: []d.EntityRef{}}
}

func (r *ObjectiveCompletionReport) addReason(code string) {
	if code == "" {
		return
	}
	for _, existing := range r.EligibilityReasonCodes {
		if existing == code {
			return
		}
	}
	r.EligibilityReasonCodes = append(r.EligibilityReasonCodes, code)
}

func (r *ObjectiveCompletionReport) markIncomplete(reason string) {
	r.FactsComplete = false
	r.addReason(reason)
}

func (r *ObjectiveCompletionReport) finish() {
	sort.Strings(r.EligibilityReasonCodes)
	r.FactualEligible = r.FactsComplete && len(r.EligibilityReasonCodes) == 0
}

func addCompletionRef(target *ObjectiveCompletionRefs, ref d.EntityRef, limit int) {
	if target == nil {
		return
	}
	target.Total++
	key := ref.ID.String()
	index := sort.Search(len(target.Samples), func(i int) bool { return target.Samples[i].ID.String() >= key })
	if index < len(target.Samples) && target.Samples[index] == ref {
		return
	}
	if index >= limit {
		return
	}
	if len(target.Samples) < limit {
		target.Samples = append(target.Samples, d.EntityRef{})
	}
	copy(target.Samples[index+1:], target.Samples[index:])
	target.Samples[index] = ref
}

func addCompletionCriterion(target *[]ObjectiveCompletionCriterionSample, sample ObjectiveCompletionCriterionSample, limit int) {
	values := *target
	key := sample.OwnerRef.ID.String() + "/" + sample.CriterionID.String()
	index := sort.Search(len(values), func(i int) bool {
		return values[i].OwnerRef.ID.String()+"/"+values[i].CriterionID.String() >= key
	})
	if index >= limit {
		return
	}
	if len(values) < limit {
		values = append(values, ObjectiveCompletionCriterionSample{})
	}
	copy(values[index+1:], values[index:])
	values[index] = sample
	*target = values
}

func objectiveCompletionReason(err error, fallback string) string {
	if err == nil {
		return fallback
	}
	var typed *d.Error
	if errors.As(err, &typed) && typed != nil {
		if typed.Details != nil && typed.Details.ReasonCode != "" {
			return typed.Details.ReasonCode
		}
		return string(typed.Code)
	}
	return fallback
}

func objectiveCompletionTaskCriteria(w d.WorkItem, verification *TaskVerificationView, limit int, report *ObjectiveCompletionReport) {
	for _, criterion := range w.Criteria.Items {
		if criterion.Status != d.CriterionStatusActive || !criterion.Required {
			continue
		}
		assessment, ok := w.Criteria.CurrentAssessments[criterion.ID]
		state := "pending"
		if ok && assessment.CriterionRevision == criterion.Revision && verificationAssessmentApplies(verification, assessment) {
			switch assessment.Result {
			case d.AssessmentResultMet:
				continue
			case d.AssessmentResultWaived:
				continue
			}
		} else if ok && assessment.CriterionRevision != criterion.Revision {
			state = "stale"
		} else if ok && !verificationAssessmentApplies(verification, assessment) {
			state = "not_current_for_delivery"
		}
		report.Tasks.IncompleteCriteria++
		addCompletionCriterion(&report.Tasks.IncompleteCriterionSamples, ObjectiveCompletionCriterionSample{
			OwnerRef: w.Ref(), CriterionID: criterion.ID, CriterionRevision: criterion.Revision, State: state,
		}, limit)
	}
}

func objectiveCompletionRequiredCriteria(objective d.Objective, limit int, report *ObjectiveCompletionReport) {
	for _, criterion := range objective.Criteria.Items {
		if criterion.Status != d.CriterionStatusActive || !criterion.Required {
			continue
		}
		report.Criteria.ActiveRequired++
		assessment, exists := objective.Criteria.CurrentAssessments[criterion.ID]
		state := "pending"
		if exists {
			switch {
			case assessment.CriterionRevision != criterion.Revision:
				state = "stale"
			case assessment.Result == d.AssessmentResultMet:
				state = "met"
			case assessment.Result == d.AssessmentResultWaived:
				state = "waived"
			default:
				state = "not_met"
			}
		}
		switch state {
		case "met":
			report.Criteria.Met++
		case "waived":
			report.Criteria.Waived++
		default:
			report.Criteria.Pending++
			addCompletionCriterion(&report.Criteria.Samples, ObjectiveCompletionCriterionSample{
				OwnerRef: objective.Ref(), CriterionID: criterion.ID, CriterionRevision: criterion.Revision, State: state,
			}, limit)
		}
	}
	if report.Criteria.ActiveRequired == 0 {
		report.addReason("missing_required_criterion")
	} else if report.Criteria.Pending > 0 {
		for _, sample := range report.Criteria.Samples {
			if sample.OwnerRef == objective.Ref() && sample.State == "stale" {
				report.addReason("criterion_assessment_stale")
			} else if sample.OwnerRef == objective.Ref() && sample.State == "not_met" {
				report.addReason("criterion_not_satisfied")
			} else if sample.OwnerRef == objective.Ref() && sample.State == "pending" {
				report.addReason("criterion_assessment_missing")
			}
		}
		// Samples are bounded; totals, not samples, establish that an obligation
		// remains. Keep a general reason even when the omitted row has the detail.
		report.addReason("required_criterion_pending")
	}
}

func objectiveCompletionCycleProofUnverifiable(objective d.Objective) bool {
	if objective.EffectiveCompletionCycle() <= 1 {
		return false
	}
	for _, criterion := range objective.Criteria.Items {
		if criterion.Status != d.CriterionStatusActive || !criterion.Required {
			continue
		}
		assessment, ok := objective.Criteria.CurrentAssessments[criterion.ID]
		if ok && assessment.CriterionRevision == criterion.Revision &&
			(assessment.Result == d.AssessmentResultMet || assessment.Result == d.AssessmentResultWaived) {
			// Criterion assessments do not yet carry an Objective completion-cycle
			// binding. A current revision may therefore still be prior-cycle proof.
			return true
		}
	}
	return false
}

func objectiveCompletionBusinessFailure(err error) bool {
	if err == nil {
		return false
	}
	var typed *d.Error
	if !errors.As(err, &typed) || typed == nil {
		return false
	}
	switch typed.Code {
	case d.ErrorCodePreconditionFailed, d.ErrorCodeInvalidTransition, d.ErrorCodeNotFound, d.ErrorCodeEvidence, d.ErrorCodeEvidenceLink, d.ErrorCodeCriterion, d.ErrorCodeAssessment:
		return true
	default:
		return false
	}
}

func objectiveCompletionTaskScope(work []d.WorkItem, objectives map[d.ID]d.Objective, owner d.Objective) []d.WorkItem {
	result := make([]d.WorkItem, 0)
	for _, item := range work {
		if item.ObjectiveID == nil {
			continue
		}
		if item.ObjectiveID != nil && objectiveDescendsFrom(*item.ObjectiveID, owner.ID, objectives) {
			result = append(result, item)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].ID.String() < result[j].ID.String() })
	return result
}

func objectiveCompletionDescendants(objectives []d.Objective, owner d.Objective, limit int, report *ObjectiveCompletionReport) map[d.ID]d.Objective {
	byID := make(map[d.ID]d.Objective, len(objectives))
	for _, value := range objectives {
		if value.Scope != owner.Scope || value.ID.Validate() != nil || byID[value.ID].ID != "" {
			report.markIncomplete("objective_scan_inconsistent")
			continue
		}
		byID[value.ID] = value
	}
	if current, ok := byID[owner.ID]; !ok || current.ParentObjectiveID == nil != (owner.ParentObjectiveID == nil) || current.ParentObjectiveID != nil && owner.ParentObjectiveID != nil && *current.ParentObjectiveID != *owner.ParentObjectiveID || current.Lifecycle != owner.Lifecycle {
		report.markIncomplete("objective_scan_inconsistent")
	}
	for _, value := range objectives {
		if value.ParentObjectiveID != nil {
			if _, ok := byID[*value.ParentObjectiveID]; !ok {
				report.markIncomplete("objective_hierarchy_incomplete")
			}
		}
		seen := map[d.ID]bool{}
		current := value.ID
		for steps := 0; ; steps++ {
			if steps > len(byID) || seen[current] {
				report.markIncomplete("objective_hierarchy_cycle")
				break
			}
			seen[current] = true
			entry, ok := byID[current]
			if !ok || entry.ParentObjectiveID == nil {
				break
			}
			current = *entry.ParentObjectiveID
		}
	}
	for _, value := range objectives {
		if value.ID == owner.ID || !objectiveDescendsFrom(value.ID, owner.ID, byID) {
			continue
		}
		report.DescendantObjectives.Total++
		switch value.Lifecycle {
		case d.ObjectiveLifecyclePlanned:
			report.DescendantObjectives.Planned++
			addCompletionRef(&report.DescendantObjectives.Open, value.Ref(), limit)
		case d.ObjectiveLifecycleInProgress:
			report.DescendantObjectives.InProgress++
			addCompletionRef(&report.DescendantObjectives.Open, value.Ref(), limit)
		case d.ObjectiveLifecycleAchieved:
			report.DescendantObjectives.Achieved++
		case d.ObjectiveLifecycleCancelled:
			report.DescendantObjectives.Cancelled++
			if value.CurrentConclusion == nil || value.CurrentConclusion.Reason == "" {
				report.markIncomplete("objective_conclusion_incomplete")
			}
		default:
			report.markIncomplete("objective_lifecycle_invalid")
		}
	}
	return byID
}
