package application

import (
	"context"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func taskVerificationProtocolCondition(protocol d.NamespaceWorkProtocol, command string) error {
	switch command {
	case "AcquireSignedWorkContract", "AcquireSignedReviewContract", "ReturnSignedWork", "ReturnSignedReview":
		if protocol.Phase != d.WorkProtocolSigned {
			return d.NewError(d.ErrorCodeSignedProtocolRequired, "Namespace has not activated signed-v2 protocol")
		}
		return nil
	default:
		return protectWorkProtocol(protocol, command)
	}
}

// Match the signed acquisition's readiness candidate without changing the Task
// or pretending that the previous delivered contract is new execution authority.
func taskVerificationCorrectionCandidate(w d.WorkItem, v *TaskVerificationView) (d.WorkItem, bool) {
	if v == nil || !taskVerificationCorrectionPending(*v) || v.SourceKind != "signed_review_case" ||
		v.signedReview == nil || w.CorrectionReviewCaseID == nil || v.signedReview.ID != *w.CorrectionReviewCaseID ||
		v.signedReview.Status != d.ReviewChangesRequested || v.signedReview.LatestDecisionID == nil ||
		w.CurrentContractID == nil || *w.CurrentContractID != v.signedReview.WorkContractID ||
		v.Executor == nil || v.Executor.Source != "work_contract" || v.Executor.ContractID == nil || *v.Executor.ContractID != *w.CurrentContractID ||
		w.Lifecycle != d.WorkItemLifecycleInProgress || !w.ContractsEnabled {
		return w, false
	}
	c := v.executionContract
	if c == nil || c.Status != d.ContractDelivered || c.SignedBinding == nil || c.ID != v.signedReview.WorkContractID ||
		c.LatestSubmissionID == nil || *c.LatestSubmissionID != v.signedReview.SubmissionID || c.SubmissionSpecDigest() != v.signedReview.IssuedSpecDigest {
		return w, false
	}
	candidate := w
	candidate.Lifecycle, candidate.CurrentLease, candidate.CurrentContractID = d.WorkItemLifecycleTodo, nil, nil
	return candidate, true
}

// This reads the same authoritative records and pure predicates as the native
// mutation. It allocates no IDs, ends no authority, and uses the frame's clock.
func (s *Service) nativeTaskVerificationConditions(ctx context.Context, u ports.UnitOfWork, w d.WorkItem, v *TaskVerificationView, at time.Time, canAccept, canChange, canWithdraw bool) (accept, changes, withdraw error) {
	if v == nil || !v.Pending || v.SourceKind != "task_delivery" || v.SourceID == nil {
		err := taskVerificationSourceError(w.Scope, w.ID, "there is no current native delivery to review")
		return err, err, err
	}
	identity, ok := IdentityFromContext(ctx)
	if !ok {
		err := d.NewError(d.ErrorCodeForbidden, "authenticated reviewer credential is required")
		return err, err, err
	}
	cc := d.CommandContext{PrincipalID: identity.PrincipalID, Actor: identity.Actor}
	reads, ok := u.(ports.TaskDeliveryUnitOfWork)
	if !ok {
		err := d.NewError(d.ErrorCodeInvalidConfig, "native delivery persistence is unavailable")
		return err, err, err
	}
	delivery, err := reads.TaskDeliveries().Get(ctx, w.Scope, *v.SourceID)
	if err != nil {
		return err, err, err
	}
	if delivery.WorkItemID != w.ID || delivery.Digest != v.DeliveryDigest || delivery.Round != v.Round || delivery.ValidateForWork(w) != nil {
		err = taskVerificationSourceError(w.Scope, w.ID, "delivery differs from the current verification snapshot")
		return err, err, err
	}
	if err = requireActiveOutcome(ctx, u, w.Scope); err != nil {
		return err, err, err
	}
	if canWithdraw {
		_, withdraw = s.resolveLeaseCredentialAuthority(ctx, u, cc, w.Scope, nil, "WithdrawTaskDelivery", at)
		if withdraw == nil && (identity.CredentialID.Validate() != nil || identity.CredentialDigest == "") {
			withdraw = d.NewError(d.ErrorCodeForbidden, "authenticated credential is required to withdraw a TaskDelivery")
		}
	}
	if !canAccept && !canChange {
		return nil, nil, withdraw
	}
	common := requireNotBlocked(ctx, u, w.Ref())
	if common == nil {
		common = requireHardDependenciesSatisfied(ctx, u, w.Ref())
	}
	if common == nil {
		_, common = s.resolveTaskDeliveryReviewerAuthority(ctx, u, cc, w.Scope, w, reads.TaskDeliveries(), at)
	}
	if common != nil {
		return common, common, withdraw
	}
	if !canAccept {
		return nil, nil, withdraw
	}
	var submission *d.WorkSubmission
	if delivery.SubmissionRef != nil {
		repo, err := contractRepository(u)
		if err != nil {
			return err, nil, withdraw
		}
		value, err := repo.GetSubmission(ctx, w.Scope, delivery.SubmissionRef.ID)
		if err != nil {
			return err, nil, withdraw
		}
		if value.Scope != w.Scope || value.Material.WorkItemID != w.ID || value.Material.ContractID != delivery.SubmissionRef.ContractID || value.Digest != delivery.SubmissionRef.Digest {
			return taskVerificationSourceError(w.Scope, w.ID, "native delivery submission binding differs"), nil, withdraw
		}
		submission = &value
	}
	refs := []d.CriterionAssessmentRef{}
	for _, c := range w.Criteria.Items {
		if c.Status == d.CriterionStatusActive && c.Required {
			if a, ok := w.Criteria.CurrentAssessments[c.ID]; ok {
				refs = append(refs, d.CriterionAssessmentRef{CriterionID: c.ID, CriterionRevision: a.CriterionRevision, AssessmentID: a.ID, Result: a.Result})
			}
		}
	}
	accept = validateTaskDeliveryAssessments(ctx, u, w, delivery, submission, refs)
	return accept, nil, withdraw
}
