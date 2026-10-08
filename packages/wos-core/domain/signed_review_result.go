package domain

import "time"

// Review approval depends only on the immutable delivered submission and the
// review case. It does not reactivate or require executor authority.
func (w *WorkItem) CompleteReviewedSubmission(r ReviewCase, p WorkSubmission, expected Version, conclusion Conclusion, now time.Time) error {
	if w.Version != expected {
		return NewError(ErrorCodeVersionConflict, "reviewed Task CAS differs")
	}
	if r.Status != ReviewApproved || r.Scope != w.Scope || r.WorkItemID != w.ID || r.SubmissionID != p.ID || r.SubmissionDigest != p.Digest || p.ProtocolVersion != 2 || p.Material.WorkItemID != w.ID || p.Material.ContractID != r.WorkContractID || p.Material.SpecDigest != r.IssuedSpecDigest || w.Lifecycle != WorkItemLifecycleInProgress || w.PendingReviewCaseID == nil || *w.PendingReviewCaseID != r.ID || w.CurrentContractID == nil || *w.CurrentContractID != r.WorkContractID {
		return NewError(ErrorCodeSubmissionNotAccepted, "review does not approve this pending exact submission")
	}
	if err := p.Validate(); err != nil {
		return err
	}
	refs, err := w.Criteria.RequiredSatisfied()
	if err != nil && w.Criteria.HasRequiredActive() {
		return err
	}
	for _, ref := range refs {
		assessment := w.Criteria.CurrentAssessments[ref.CriterionID]
		if assessment.SubmissionID == nil || *assessment.SubmissionID != p.ID || assessment.SubmissionDigest != p.Digest {
			return NewError(ErrorCodeSubmissionNotAccepted, "review assessment targets different material")
		}
	}
	next, err := w.Version.Next()
	if err != nil {
		return err
	}
	conclusion.Assessments = refs
	id := p.ID
	conclusion.SubmissionID = &id
	conclusion.SubmissionDigest = p.Digest
	if err = conclusion.Bind(w.Ref(), next, string(WorkItemLifecycleDone), ConclusionObligations{RequiredCriteria: w.Criteria.RequiredObligations(), ResultSummaryRequired: !w.Criteria.HasRequiredActive()}); err != nil {
		return err
	}
	if err = conclusion.Validate(); err != nil {
		return err
	}
	w.Version = next
	w.Lifecycle = WorkItemLifecycleDone
	w.ResultSummary = p.Material.Summary
	w.CurrentConclusion = &conclusion
	w.CurrentLease = nil
	w.CurrentContractID = nil
	w.PendingReviewCaseID = nil
	w.CorrectionReviewCaseID = nil
	w.UpdatedAt = now.UTC()
	return w.Validate()
}
