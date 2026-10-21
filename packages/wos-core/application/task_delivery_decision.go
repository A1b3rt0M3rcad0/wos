package application

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// AcceptTaskDeliveryCommand approves one exact immutable delivery after the
// required current assessments have been reviewed.
type AcceptTaskDeliveryCommand struct {
	Scope                   domain.Scope                    `json:"scope"`
	WorkItemID              domain.ID                       `json:"work_item_id"`
	ExpectedWorkItemVersion domain.Version                  `json:"expected_work_item_version,string"`
	DeliveryID              domain.ID                       `json:"delivery_id"`
	ExpectedDeliveryDigest  string                          `json:"expected_delivery_digest"`
	ExpectedDeliveryRound   uint64                          `json:"expected_delivery_round,string"`
	Reason                  string                          `json:"reason"`
	AssessmentRefs          []domain.CriterionAssessmentRef `json:"assessment_refs,omitempty"`
}

// RequestTaskDeliveryChangesCommand returns one exact pending delivery to To
// do. The next executor must acquire fresh authority.
type RequestTaskDeliveryChangesCommand struct {
	Scope                   domain.Scope   `json:"scope"`
	WorkItemID              domain.ID      `json:"work_item_id"`
	ExpectedWorkItemVersion domain.Version `json:"expected_work_item_version,string"`
	DeliveryID              domain.ID      `json:"delivery_id"`
	ExpectedDeliveryDigest  string         `json:"expected_delivery_digest"`
	ExpectedDeliveryRound   uint64         `json:"expected_delivery_round,string"`
	Reason                  string         `json:"reason"`
	Findings                []string       `wos:"required" json:"findings"`
}

type TaskDeliveryReviewResult struct {
	WorkItem    domain.WorkItem             `json:"work_item"`
	Delivery    domain.TaskDelivery         `json:"delivery"`
	Decision    domain.TaskDeliveryDecision `json:"decision"`
	EvaluatedAt time.Time                   `json:"evaluated_at"`
}

func (s *Service) AcceptTaskDelivery(ctx context.Context, cc domain.CommandContext, cmd AcceptTaskDeliveryCommand) (MutationResult[TaskDeliveryReviewResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[TaskDeliveryReviewResult]{}, err
	}
	if strings.TrimSpace(cc.IdempotencyKey) == "" {
		return MutationResult[TaskDeliveryReviewResult]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "Task delivery acceptance requires a non-empty idempotency key")
	}
	if err := validateTaskDeliveryDecisionTarget(cmd.Scope, cmd.WorkItemID, cmd.ExpectedWorkItemVersion, cmd.DeliveryID, cmd.ExpectedDeliveryDigest, cmd.ExpectedDeliveryRound, cmd.Reason); err != nil {
		return MutationResult[TaskDeliveryReviewResult]{}, err
	}
	if len(cmd.AssessmentRefs) > 100 {
		return MutationResult[TaskDeliveryReviewResult]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "delivery decision has too many assessment references")
	}
	if err := boundedWorkspaceCommand(cmd); err != nil {
		return MutationResult[TaskDeliveryReviewResult]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (TaskDeliveryReviewResult, domain.OutcomeRevision, error) {
		return s.decideTaskDelivery(ctx, u, cc, cmd.Scope, cmd.WorkItemID, cmd.ExpectedWorkItemVersion, cmd.DeliveryID, cmd.ExpectedDeliveryDigest, cmd.ExpectedDeliveryRound, domain.TaskDeliveryDecisionAccepted, cmd.Reason, nil, cmd.AssessmentRefs)
	})
}

func (s *Service) RequestTaskDeliveryChanges(ctx context.Context, cc domain.CommandContext, cmd RequestTaskDeliveryChangesCommand) (MutationResult[TaskDeliveryReviewResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[TaskDeliveryReviewResult]{}, err
	}
	if strings.TrimSpace(cc.IdempotencyKey) == "" {
		return MutationResult[TaskDeliveryReviewResult]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "Task delivery changes request requires a non-empty idempotency key")
	}
	if err := validateTaskDeliveryDecisionTarget(cmd.Scope, cmd.WorkItemID, cmd.ExpectedWorkItemVersion, cmd.DeliveryID, cmd.ExpectedDeliveryDigest, cmd.ExpectedDeliveryRound, cmd.Reason); err != nil {
		return MutationResult[TaskDeliveryReviewResult]{}, err
	}
	if len(cmd.Findings) == 0 || len(cmd.Findings) > 100 {
		return MutationResult[TaskDeliveryReviewResult]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "changes requested requires 1 to 100 actionable findings")
	}
	for _, finding := range cmd.Findings {
		if strings.TrimSpace(finding) == "" || len(finding) > 4096 {
			return MutationResult[TaskDeliveryReviewResult]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "each correction finding must be nonempty and bounded")
		}
	}
	if err := boundedWorkspaceCommand(cmd); err != nil {
		return MutationResult[TaskDeliveryReviewResult]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (TaskDeliveryReviewResult, domain.OutcomeRevision, error) {
		return s.decideTaskDelivery(ctx, u, cc, cmd.Scope, cmd.WorkItemID, cmd.ExpectedWorkItemVersion, cmd.DeliveryID, cmd.ExpectedDeliveryDigest, cmd.ExpectedDeliveryRound, domain.TaskDeliveryDecisionChangesRequested, cmd.Reason, cmd.Findings, nil)
	})
}

func validateTaskDeliveryDecisionTarget(scope domain.Scope, workID domain.ID, workVersion domain.Version, deliveryID domain.ID, digest string, round uint64, reason string) error {
	if err := scope.Validate(); err != nil {
		return err
	}
	if err := workID.Validate(); err != nil {
		return err
	}
	if err := workVersion.Validate(); err != nil {
		return err
	}
	if err := deliveryID.Validate(); err != nil {
		return err
	}
	if round == 0 || !domain.ValidSignedDigest(digest) || strings.TrimSpace(reason) == "" || len(reason) > 4096 {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "delivery decision requires exact delivery digest and round plus a bounded reason")
	}
	return nil
}

func (s *Service) decideTaskDelivery(
	ctx context.Context,
	u ports.UnitOfWork,
	cc domain.CommandContext,
	scope domain.Scope,
	workItemID domain.ID,
	expectedWorkVersion domain.Version,
	deliveryID domain.ID,
	expectedDigest string,
	expectedRound uint64,
	result domain.TaskDeliveryDecisionResult,
	reason string,
	findings []string,
	assessmentRefs []domain.CriterionAssessmentRef,
) (TaskDeliveryReviewResult, domain.OutcomeRevision, error) {
	var zero TaskDeliveryReviewResult
	if err := requireActiveOutcome(ctx, u, scope); err != nil {
		return zero, 0, err
	}
	if _, err := u.Coordination().LockOutcome(ctx, scope); err != nil {
		return zero, 0, err
	}
	deliveryUOW, ok := u.(ports.TaskDeliveryUnitOfWork)
	if !ok {
		return zero, 0, domain.NewError(domain.ErrorCodeInvalidConfig, "adapter lacks TaskDelivery persistence")
	}
	w, err := u.WorkItems().Get(ctx, scope, workItemID)
	if err != nil {
		return zero, 0, err
	}
	if w.Version != expectedWorkVersion {
		return zero, 0, domain.NewError(domain.ErrorCodeVersionConflict, "Task version conflict")
	}
	delivery, err := deliveryUOW.TaskDeliveries().Get(ctx, scope, deliveryID)
	if err != nil {
		return zero, 0, err
	}
	if delivery.WorkItemID != workItemID || delivery.Digest != expectedDigest || delivery.Round != expectedRound {
		return zero, 0, domain.NewError(domain.ErrorCodeVersionConflict, "delivery digest, round, or Task binding changed")
	}
	pending, err := deliveryUOW.TaskDeliveries().Pending(ctx, scope, workItemID)
	if err != nil {
		return zero, 0, err
	}
	if pending == nil || pending.ID != delivery.ID || pending.Digest != delivery.Digest || pending.Round != delivery.Round {
		return zero, 0, domain.NewError(domain.ErrorCodeSubmissionNotAccepted, "delivery is no longer the current pending review")
	}
	if err := delivery.ValidateForWork(w); err != nil {
		return zero, 0, err
	}
	if err := requireNotBlocked(ctx, u, w.Ref()); err != nil {
		return zero, 0, err
	}
	if err := requireHardDependenciesSatisfied(ctx, u, w.Ref()); err != nil {
		return zero, 0, err
	}
	now, err := s.transactionTime(ctx, u)
	if err != nil {
		return zero, 0, err
	}
	reviewerAuthority, err := s.resolveTaskDeliveryReviewerAuthority(ctx, u, cc, scope, w, deliveryUOW.TaskDeliveries(), now)
	if err != nil {
		return zero, 0, err
	}
	_ = reviewerAuthority // The resolved canonical credential binds the protected receipt; the decision records principal/actor.

	var submission *domain.WorkSubmission
	if delivery.SubmissionRef != nil {
		repository, err := contractRepository(u)
		if err != nil {
			return zero, 0, err
		}
		value, err := repository.GetSubmission(ctx, scope, delivery.SubmissionRef.ID)
		if err != nil {
			return zero, 0, err
		}
		if value.ID != delivery.SubmissionRef.ID || value.Material.ContractID != delivery.SubmissionRef.ContractID || value.Digest != delivery.SubmissionRef.Digest || value.Material.WorkItemID != workItemID || value.Scope != scope {
			return zero, 0, domain.NewError(domain.ErrorCodeSubmissionNotAccepted, "delivery no longer references the exact contracts_v1 submission")
		}
		submission = &value
	}

	if result == domain.TaskDeliveryDecisionAccepted {
		if err := validateTaskDeliveryAssessments(ctx, u, w, delivery, submission, assessmentRefs); err != nil {
			return zero, 0, err
		}
	} else if len(assessmentRefs) != 0 {
		return zero, 0, domain.NewError(domain.ErrorCodeInvalidArgument, "changes request cannot include accepted assessment references")
	}
	decisionID, err := s.ids.NewID()
	if err != nil {
		return zero, 0, err
	}
	decision, err := domain.NewTaskDeliveryDecision(decisionID, delivery, result, cc.PrincipalID, cc.Actor, reason, findings, now)
	if err != nil {
		return zero, 0, err
	}
	before := w.Version
	if result == domain.TaskDeliveryDecisionAccepted {
		conclusionID, err := s.ids.NewID()
		if err != nil {
			return zero, 0, err
		}
		conclusion := conclusionFromContext(conclusionID, cc, reason, now)
		if err := w.AcceptTaskDelivery(delivery, decision, submission, assessmentRefs, before, conclusion, now); err != nil {
			return zero, 0, err
		}
	} else if err := w.RequestTaskDeliveryChanges(delivery, decision, before, now); err != nil {
		return zero, 0, err
	}
	if err := deliveryUOW.TaskDeliveries().InsertDecision(ctx, decision); err != nil {
		return zero, 0, err
	}
	if err := u.WorkItems().Save(ctx, w, before); err != nil {
		return zero, 0, err
	}
	revision, err := u.Coordination().AdvanceOutcome(ctx, scope)
	return TaskDeliveryReviewResult{WorkItem: w, Delivery: delivery, Decision: decision, EvaluatedAt: now}, revision, err
}

func (s *Service) resolveTaskDeliveryReviewerAuthority(ctx context.Context, u ports.UnitOfWork, cc domain.CommandContext, scope domain.Scope, w domain.WorkItem, deliveries ports.TaskDeliveryRepository, now time.Time) (*leaseCredentialAuthority, error) {
	identity, ok := IdentityFromContext(ctx)
	if !ok || identity.CredentialDigest == "" || identity.CredentialID.Validate() != nil || identity.NamespaceID != scope.NamespaceID || identity.PrincipalID != cc.PrincipalID || identity.Actor != cc.Actor {
		return nil, domain.NewError(domain.ErrorCodeForbidden, "authenticated reviewer credential is required")
	}
	authority, err := s.resolveLeaseCredentialAuthority(ctx, u, cc, scope, nil, "", now)
	if err != nil {
		return nil, err
	}
	if authority == nil {
		return nil, domain.NewError(domain.ErrorCodeForbidden, "authenticated reviewer credential could not be resolved")
	}
	if !s.independentReviewer {
		return authority, nil
	}
	for after := domain.ID(""); ; {
		page, err := deliveries.List(ctx, scope, ports.TaskDeliveryFilter{WorkItemID: w.ID, After: after, Limit: 100})
		if err != nil {
			return nil, err
		}
		for _, prior := range page {
			if prior.Authority.PrincipalID == cc.PrincipalID {
				return nil, domain.NewError(domain.ErrorCodeForbidden, "deployment requires a reviewer principal independent from every executor in this Task's delivery history")
			}
		}
		if len(page) < 100 {
			break
		}
		after = page[len(page)-1].ID
	}
	if err := s.requireIndependentReview(ctx, u, scope, cc.PrincipalID); err != nil {
		return nil, err
	}
	return authority, nil
}

func validateTaskDeliveryAssessments(ctx context.Context, u ports.UnitOfWork, w domain.WorkItem, delivery domain.TaskDelivery, submission *domain.WorkSubmission, refs []domain.CriterionAssessmentRef) error {
	if err := validateRequiredAssessmentProof(ctx, u, w.Ref(), w.Criteria); err != nil {
		return err
	}
	expected := make(map[domain.ID]domain.CriterionAssessmentRef, len(refs))
	for _, ref := range refs {
		if _, exists := expected[ref.CriterionID]; exists {
			return domain.NewError(domain.ErrorCodeAssessment, "assessment references contain a duplicate criterion")
		}
		expected[ref.CriterionID] = ref
	}
	for _, criterion := range w.Criteria.Items {
		if criterion.Status != domain.CriterionStatusActive || !criterion.Required {
			continue
		}
		assessment, exists := w.Criteria.CurrentAssessments[criterion.ID]
		if !exists || assessment.CriterionRevision != criterion.Revision || assessment.Result != domain.AssessmentResultMet {
			return domain.NewError(domain.ErrorCodeAssessment, "every required active criterion needs a current met assessment before acceptance")
		}
		ref, present := expected[criterion.ID]
		if !present || ref.AssessmentID != assessment.ID || ref.CriterionRevision != assessment.CriterionRevision || ref.Result != assessment.Result {
			return domain.NewError(domain.ErrorCodeAssessment, "acceptance must reference the exact current assessment for every required criterion")
		}
		expectedDeliveryRef := domain.TaskDeliveryAssessmentRef{ID: delivery.ID, Digest: delivery.Digest, Round: delivery.Round}
		if assessment.TaskDeliveryRef == nil || *assessment.TaskDeliveryRef != expectedDeliveryRef {
			return domain.NewError(domain.ErrorCodeSubmissionNotAccepted, "criterion assessment is not bound to the exact pending delivery and correction round")
		}
		delete(expected, criterion.ID)
		if delivery.SubmissionRef != nil {
			if submission == nil || assessment.SubmissionID == nil || *assessment.SubmissionID != submission.ID || assessment.SubmissionDigest != submission.Digest {
				return domain.NewError(domain.ErrorCodeSubmissionNotAccepted, "contracts_v1 assessment is not bound to the exact delivered submission")
			}
			if err := assessmentEvidenceMatchesCriterionMaterial(criterion.ID, assessment.EvidenceIDs, submission.Material.EvidenceIDs, submission.Material.CriterionEvidence); err != nil {
				return err
			}
		} else {
			if assessment.SubmissionID != nil {
				return domain.NewError(domain.ErrorCodeSubmissionNotAccepted, "legacy TaskDelivery cannot accept a contracts_v1-bound assessment")
			}
			if delivery.Material == nil {
				return domain.NewError(domain.ErrorCodeSubmissionNotAccepted, "legacy TaskDelivery has no inline proof material")
			}
			if err := assessmentEvidenceMatchesCriterionMaterial(criterion.ID, assessment.EvidenceIDs, delivery.Material.EvidenceIDs, delivery.Material.CriterionEvidence); err != nil {
				return err
			}
		}
	}
	if len(expected) != 0 {
		return domain.NewError(domain.ErrorCodeAssessment, "acceptance references must match current required criterion assessments only")
	}
	if w.Criteria.HasRequiredActive() {
		if _, err := w.Criteria.RequiredSatisfied(); err != nil {
			return err
		}
	}
	return nil
}

func assessmentEvidenceMatchesCriterionMaterial(criterionID domain.ID, assessmentEvidence, deliveryEvidence []domain.ID, criterionLinks []domain.SubmissionCriterionEvidence) error {
	deliverySet := make(map[domain.ID]struct{}, len(deliveryEvidence))
	for _, id := range deliveryEvidence {
		deliverySet[id] = struct{}{}
	}
	var criterionEvidence map[domain.ID]struct{}
	for _, link := range criterionLinks {
		if link.CriterionID == criterionID {
			criterionEvidence = make(map[domain.ID]struct{}, len(link.EvidenceIDs))
			for _, id := range link.EvidenceIDs {
				criterionEvidence[id] = struct{}{}
			}
			break
		}
	}
	for _, id := range assessmentEvidence {
		if _, exists := deliverySet[id]; !exists {
			return domain.NewError(domain.ErrorCodeSubmissionNotAccepted, "criterion assessment references evidence outside the current delivery")
		}
		if criterionEvidence != nil {
			if _, linked := criterionEvidence[id]; !linked {
				return domain.NewError(domain.ErrorCodeSubmissionNotAccepted, "criterion assessment evidence is not linked to that criterion in the current delivery")
			}
		} else {
			return domain.NewError(domain.ErrorCodeSubmissionNotAccepted, "criterion assessment evidence lacks a criterion-specific link in the current delivery")
		}
	}
	return nil
}

func taskDeliveryDecisionCommandEvents(s *Service, cc domain.CommandContext, meta commandMetadata, result TaskDeliveryReviewResult, revision domain.OutcomeRevision) ([]domain.DomainEvent, bool, error) {
	if meta.Name != "AcceptTaskDelivery" && meta.Name != "RequestTaskDeliveryChanges" {
		return nil, false, nil
	}
	before := result.WorkItem.Version - 1
	eventType := "work_review.changes_requested"
	if result.Decision.Result == domain.TaskDeliveryDecisionAccepted {
		eventType = "work_review.approved"
	}
	specs := []compoundEventSpec{{eventType: eventType, ref: result.WorkItem.Ref(), before: versionPtr(before), after: versionPtr(result.WorkItem.Version)}}
	if result.Decision.Result == domain.TaskDeliveryDecisionAccepted {
		specs = append(specs,
			compoundEventSpec{eventType: "work_item.completed", ref: result.WorkItem.Ref(), before: versionPtr(before), after: versionPtr(result.WorkItem.Version)},
			compoundEventSpec{eventType: "work_item.conclusion_recorded", ref: result.WorkItem.Ref(), before: versionPtr(before), after: versionPtr(result.WorkItem.Version)},
		)
	}
	events, err := buildCompoundEvents(s, cc, meta, revision, specs)
	if err != nil {
		return nil, true, err
	}
	for i := range events {
		payload := map[string]any{
			"delivery_id": result.Delivery.ID, "delivery_digest": result.Delivery.Digest,
			"delivery_round": result.Delivery.Round, "decision_id": result.Decision.ID,
			"decision": result.Decision.Result, "reviewer_principal_id": result.Decision.ReviewerPrincipalID,
		}
		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, true, err
		}
		if events[i].EventType == "work_item.conclusion_recorded" {
			raw, err = conclusionRecordedPayload(result.WorkItem)
			if err != nil {
				return nil, true, err
			}
		}
		events[i].Payload = raw
		events[i].RecordedAt = result.EvaluatedAt
		if err := events[i].Validate(); err != nil {
			return nil, true, err
		}
	}
	return events, true, nil
}
