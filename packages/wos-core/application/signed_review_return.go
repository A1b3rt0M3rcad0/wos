package application

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"reflect"
	"strings"
	"time"
)

type ReturnSignedReviewCommand struct{ Envelope signing.Envelope }
type SignedReviewMaterial struct {
	Reason                 string                         `json:"reason"`
	Findings               []d.SignedFinding              `json:"findings,omitempty"`
	Assessments            []SignedAssessmentInput        `json:"criterion_assessments,omitempty"`
	Artifacts              []SyncArtifactInput            `json:"artifacts,omitempty"`
	Evidence               []SyncEvidenceInput            `json:"evidence,omitempty"`
	EvidenceLinks          []SyncEvidenceLinkInput        `json:"evidence_links,omitempty"`
	CriterionLocalEvidence []SignedLocalCriterionEvidence `json:"criterion_local_evidence,omitempty"`
	ReviewedSourceVersion  string                         `json:"reviewed_source_version,omitempty"`
}

func (s *Service) ReturnSignedReview(ctx context.Context, cc d.CommandContext, cmd ReturnSignedReviewCommand) (MutationResult[SignedReturnResult], error) {
	var hint signing.ReviewReturnPayload[SignedReviewMaterial]
	if err := signedRequestHint(cmd.Envelope, &hint); err != nil {
		return MutationResult[SignedReturnResult]{}, err
	}
	if _, err := signedRequestScope(hint.RequestBinding); err != nil {
		return MutationResult[SignedReturnResult]{}, err
	}
	if hint.ExpectedReviewCaseVersion == 0 || d.ID(hint.ReviewCaseID).Validate() != nil || d.ID(hint.SubmissionID).Validate() != nil || !d.ValidSignedDigest(hint.SubmissionDigest) {
		return MutationResult[SignedReturnResult]{}, d.NewError(d.ErrorCodeInvalidArgument, "signed review target/CAS required")
	}
	if cc.PrincipalID != hint.PrincipalID || cc.IdempotencyKey != hint.IdempotencyKey {
		return MutationResult[SignedReturnResult]{}, d.NewError(d.ErrorCodeForbidden, "review identity/idempotency differs")
	}
	cc.CommandID = d.ID(hint.RequestID)
	if err := cc.Validate(); err != nil {
		return MutationResult[SignedReturnResult]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (SignedReturnResult, d.OutcomeRevision, error) {
		var zero SignedReturnResult
		scope, err := signedRequestScope(hint.RequestBinding)
		if err != nil {
			return zero, 0, err
		}
		repo, workRepo, err := signedRepository(u)
		if err != nil {
			return zero, 0, err
		}
		registry, err := signingRepository(u)
		if err != nil {
			return zero, 0, err
		}
		key, err := registry.Key(ctx, scope.NamespaceID, d.ID(hint.SignerKeyID))
		if err != nil {
			return zero, 0, err
		}
		if key.Purpose != "agent" || key.PrincipalID != cc.PrincipalID {
			return zero, 0, d.NewError(d.ErrorCodeForbidden, "review signer role/Principal differs")
		}
		pub, err := key.PublicBytes()
		if err != nil {
			return zero, 0, err
		}
		raw, err := signing.Verify(cmd.Envelope, signing.ReviewReturn, key.ID.String(), pub)
		if err != nil {
			return zero, 0, signingError(err)
		}
		var request signing.ReviewReturnPayload[SignedReviewMaterial]
		if err = signing.DecodeStrict(raw, &request, 180*1024); err != nil {
			return zero, 0, signingError(err)
		}
		if request.OperationKind != "review_return" {
			return zero, 0, d.NewError(d.ErrorCodeInvalidArgument, "wrong signed review operation")
		}
		digest := signing.Digest(raw)
		if prior, e := repo.Acceptance(ctx, scope.NamespaceID, cc.PrincipalID, cc.IdempotencyKey); e == nil {
			if e = s.requireSignedAcceptedCredential(ctx, u, scope, "review", d.ID(request.ContractID)); e != nil {
				return zero, 0, e
			}
			if prior.Scope != scope || prior.ContractKind != "review" || prior.RequestDigest != digest || prior.RequestID == nil || *prior.RequestID != d.ID(request.RequestID) {
				return zero, 0, d.NewError(d.ErrorCodeIdempotencyConflict, "confirmed review intent differs")
			}
			doc, e := verifyIssuedFact(ctx, u, prior, signing.AcceptanceReceipt, request.ServerID, cc.PrincipalID)
			if e != nil {
				return zero, 0, e
			}
			var receipt signing.ReceiptPayload
			if e = signing.DecodeStrict(doc.Payload, &receipt, 180*1024); e != nil {
				return zero, 0, signingError(e)
			}
			return SignedReturnResult{Receipt: doc, durableReplay: true}, d.OutcomeRevision(receipt.OutcomeRevision), nil
		} else if code, _ := d.ErrorCodeOf(e); code != d.ErrorCodeNotFound {
			return zero, 0, e
		}
		identity, policy, err := s.signedAccess(ctx, u, scope, ports.PermissionWorkReviewDecide, key.ID)
		if err != nil {
			return zero, 0, err
		}
		server, err := s.requireSignedIssuer(ctx, u, scope.NamespaceID)
		if err != nil {
			return zero, 0, err
		}
		if request.ServerID != server.ID.String() || request.PolicyRevision != signing.Decimal(policy.Version) {
			return zero, 0, d.NewError(d.ErrorCodeVersionConflict, "review server/current policy changed")
		}
		if _, err = u.Coordination().LockOutcome(ctx, scope); err != nil {
			return zero, 0, err
		}
		if err = requireActiveOutcome(ctx, u, scope); err != nil {
			return zero, 0, err
		}
		contract, err := repo.ReviewContract(ctx, scope, d.ID(request.ContractID))
		if err != nil {
			return zero, 0, err
		}
		if contract.Binding.CredentialID != identity.CredentialID || contract.HolderPrincipalID != identity.PrincipalID || contract.WorkItemID != d.ID(request.WorkItemID) || contract.CaseID != d.ID(request.ReviewCaseID) || contract.SubmissionID != d.ID(request.SubmissionID) || contract.SubmissionDigest != request.SubmissionDigest {
			return zero, 0, d.NewError(d.ErrorCodeForbidden, "review return target/credential differs")
		}
		grant, err := requireIssuedReviewAuthority(ctx, u, contract, ContractAuthority{ExecutionID: d.ID(request.ExecutionID), FencingToken: d.FencingToken(request.FencingToken), SpecDigest: request.SpecDigest}, request.AuthorityDigest)
		if err != nil {
			return zero, 0, err
		}
		allowed := false
		for _, id := range grant.AllowedSigningKeyIDs {
			allowed = allowed || id == key.ID.String()
		}
		if !allowed {
			return zero, 0, d.NewError(d.ErrorCodeForbidden, "review signer outside issued grant")
		}
		if contract.Version != d.Version(request.ExpectedContractVersion) || contract.LeaseVersion != d.Version(request.ExpectedLeaseVersion) {
			return zero, 0, d.NewError(d.ErrorCodeVersionConflict, "review contract CAS changed")
		}
		now, err := securityTransactionTime(ctx, u, s.clock)
		if err != nil {
			return zero, 0, err
		}
		if err = contract.Authorize(identity.PrincipalID, d.ID(request.ExecutionID), d.FencingToken(request.FencingToken), now); err != nil {
			return zero, 0, err
		}
		review, err := repo.Case(ctx, scope, contract.CaseID)
		if err != nil {
			return zero, 0, err
		}
		if review.Version != d.Version(request.ExpectedReviewCaseVersion) || review.Status != d.ReviewInReview || review.CurrentContractID == nil || *review.CurrentContractID != contract.ID {
			return zero, 0, d.NewError(d.ErrorCodeVersionConflict, "review case CAS/authority changed")
		}
		group, err := registry.PrincipalGroup(ctx, scope.NamespaceID, identity.PrincipalID)
		if err != nil {
			return zero, 0, err
		}
		if err = s.requireSignedReviewIndependence(ctx, u, review, identity.PrincipalID, group); err != nil {
			return zero, 0, err
		}
		// Frozen separation remains a floor when an administrator later changes groups.
		if err = review.RequireIndependent(identity.PrincipalID, contract.SeparationGroup); err != nil {
			return zero, 0, err
		}
		target, work, err := s.authenticatedReviewTarget(ctx, u, review, server.ID.String())
		if err != nil {
			return zero, 0, err
		}
		if work.Version != d.Version(request.ExpectedWorkItemVersion) {
			return zero, 0, d.NewError(d.ErrorCodeVersionConflict, "reviewed Task CAS changed")
		}
		if contract.IssuedSpecificationID == nil {
			return zero, 0, d.NewError(d.ErrorCodeSignedProtocolRequired, "issued review specification required")
		}
		fact, err := repo.Fact(ctx, scope, *contract.IssuedSpecificationID)
		if err != nil {
			return zero, 0, err
		}
		doc, err := verifyIssuedFact(ctx, u, fact, signing.ContractSpec, server.ID.String(), identity.PrincipalID)
		if err != nil {
			return zero, 0, err
		}
		var spec signing.SpecPayload[SignedReviewSpec]
		if err = signing.DecodeStrict(doc.Payload, &spec, 128*1024); err != nil {
			return zero, 0, signingError(err)
		}
		expectedPayload := spec
		expectedPayload.Spec = target
		expectedReviewSpec, err := compactSignedReviewSpec(expectedPayload)
		if err != nil {
			return zero, 0, err
		}
		if fact.PayloadDigest != request.SpecDigest || spec.ContractKind != "review" || spec.ContractID != contract.ID.String() || spec.WorkItemID != work.ID.String() || !reflect.DeepEqual(spec.Spec, expectedReviewSpec) {
			return zero, 0, d.NewError(d.ErrorCodeContractSpecMismatch, "authenticated review obligations/target differ")
		}
		reason := strings.TrimSpace(request.Material.Reason)
		if reason == "" {
			return zero, 0, d.NewError(d.ErrorCodeInvalidArgument, "review decision needs an explicit rationale")
		}
		if request.Decision != "approved" && request.Decision != "changes_requested" && request.Decision != "inconclusive" {
			return zero, 0, d.NewError(d.ErrorCodeInvalidArgument, "review decision must approve/request changes/be inconclusive")
		}
		var workPayload signing.SpecPayload[d.SignedWorkSpec]
		if err = signing.DecodeStrict(target.IssuedWorkSpecification.Payload, &workPayload, 128*1024); err != nil {
			return zero, 0, signingError(err)
		}
		if len(request.Material.Findings) > 100 {
			return zero, 0, d.NewError(d.ErrorCodeInvalidArgument, "finding limit is 100")
		}
		seen := map[d.ID]bool{}
		for _, finding := range request.Material.Findings {
			if seen[finding.ID] {
				return zero, 0, d.NewError(d.ErrorCodeInvalidArgument, "duplicate finding identity")
			}
			seen[finding.ID] = true
			if err = finding.ValidateAgainst(workPayload.Spec.WorkSpec()); err != nil {
				return zero, 0, err
			}
		}
		if request.Decision == "changes_requested" && len(request.Material.Findings) == 0 {
			return zero, 0, d.NewError(d.ErrorCodeInvalidArgument, "changes requested needs addressable findings")
		}
		if request.Decision != "approved" && len(request.Material.Assessments) > 0 {
			return zero, 0, d.NewError(d.ErrorCodeForbidden, "non-approval cannot certify criteria")
		}
		if len(request.Material.Artifacts)+len(request.Material.Evidence)+len(request.Material.EvidenceLinks) > 0 {
			if _, _, err = s.signedAccess(ctx, u, scope, ports.PermissionRecordsWrite, key.ID); err != nil {
				return zero, 0, err
			}
			if err = validateComplementaryReviewSource(target.Submission.Material(), request.Material); err != nil {
				return zero, 0, err
			}
		}
		for _, link := range request.Material.EvidenceLinks {
			if link.Link.TargetRef != work.Ref() {
				return zero, 0, d.NewError(d.ErrorCodeSubmissionNotAccepted, "review evidence link must target the reviewed Task")
			}
			if link.Link.CriterionID != nil {
				found := false
				for _, criterion := range work.Criteria.Items {
					found = found || criterion.ID == *link.Link.CriterionID && criterion.Status == d.CriterionStatusActive
				}
				if !found {
					return zero, 0, d.NewError(d.ErrorCodeSubmissionNotAccepted, "review link criterion is not a reviewed obligation")
				}
			}
		}
		registered, err := s.registerContractRecords(ctx, u, cc, SyncWorkContractCommand{Scope: scope, Artifacts: request.Material.Artifacts, Evidence: request.Material.Evidence, EvidenceLinks: request.Material.EvidenceLinks}, now)
		if err != nil {
			return zero, 0, err
		}
		submission, err := workRepo.GetSubmission(ctx, scope, review.SubmissionID)
		if err != nil {
			return zero, 0, err
		}
		supplemental := d.WorkResultMaterial{ContractID: contract.ID, WorkItemID: work.ID, SpecDigest: request.SpecDigest, Summary: reason}
		supplementalInput := SignedReturnMaterial{CriterionLocalEvidence: request.Material.CriterionLocalEvidence}
		for _, e := range registered.Evidence {
			supplemental.EvidenceIDs = append(supplemental.EvidenceIDs, e.ID)
		}
		if err = resolveSignedMaterial(&supplemental, supplementalInput, registered); err != nil {
			return zero, 0, err
		}
		// These are decision-owned observations; never rewrite executor material.
		cv, lv, rv, wv := contract.Version, contract.LeaseVersion, review.Version, work.Version
		if request.Decision == "approved" {
			for _, permission := range []ports.Permission{ports.PermissionAssessmentWrite, ports.PermissionConclusionWrite} {
				if _, _, err = s.signedAccess(ctx, u, scope, permission, key.ID); err != nil {
					return zero, 0, err
				}
			}
			if err = s.recordSignedAssessments(ctx, u, cc, &work, submission, request.Material.Assessments, registered.LocalKeys, now, supplemental.CriterionEvidence); err != nil {
				return zero, 0, err
			}
			candidate := work
			candidate.Lifecycle = d.WorkItemLifecycleTodo
			candidate.CurrentLease = nil
			if err = requireWorkItemReady(ctx, u, candidate, now); err != nil {
				return zero, 0, err
			}
			if err = requireNotBlocked(ctx, u, work.Ref()); err != nil {
				return zero, 0, err
			}
			if err = requireHardDependenciesSatisfied(ctx, u, work.Ref()); err != nil {
				return zero, 0, err
			}
		}
		if err = contract.Close(d.ContractCompleted, identity.PrincipalID, reason, cv, now); err != nil {
			return zero, 0, err
		}
		decisionID, err := s.ids.NewID()
		if err != nil {
			return zero, 0, err
		}
		review.LatestDecisionID = &decisionID
		disposition := "review_inconclusive"
		switch request.Decision {
		case "approved":
			if err = review.Decide(contract, d.ReviewApproved, identity.PrincipalID, reason, now); err != nil {
				return zero, 0, err
			}
			id, e := s.ids.NewID()
			if e != nil {
				return zero, 0, e
			}
			if err = work.CompleteReviewedSubmission(review, submission, wv, conclusionFromContext(id, cc, reason, now), now); err != nil {
				return zero, 0, err
			}
			disposition = "review_approved"
		case "changes_requested":
			if err = review.Decide(contract, d.ReviewChangesRequested, identity.PrincipalID, reason, now); err != nil {
				return zero, 0, err
			}
			work.PendingReviewCaseID = nil
			work.CorrectionReviewCaseID = &review.ID
			next, e := work.Version.Next()
			if e != nil {
				return zero, 0, e
			}
			work.Version = next
			work.UpdatedAt = now
			disposition = "changes_requested"
		case "inconclusive":
			if err = review.Release(contract, now); err != nil {
				return zero, 0, err
			}
		}
		if err = repo.SaveReviewContract(ctx, contract, cv, lv); err != nil {
			return zero, 0, err
		}
		if err = repo.SaveCase(ctx, review, rv); err != nil {
			return zero, 0, err
		}
		if request.Decision != "inconclusive" {
			if err = u.WorkItems().Save(ctx, work, wv); err != nil {
				return zero, 0, err
			}
		}
		returned, err := signing.ToDocument(cmd.Envelope)
		if err != nil {
			return zero, 0, signingError(err)
		}
		proof, err := json.Marshal(returned.Proof)
		if err != nil {
			return zero, 0, err
		}
		requestID := d.ID(request.RequestID)
		decision := d.SignedFact{ID: decisionID, Scope: scope, ContractID: contract.ID, ContractKind: "review", Kind: "return", PrincipalID: identity.PrincipalID, KeyID: key.ID, RequestID: &requestID, IdempotencyKey: cc.IdempotencyKey, RequestDigest: digest, PayloadDigest: digest, Payload: raw, Proof: proof, RecordedAt: now}
		if err = repo.InsertFact(ctx, decision); err != nil {
			return zero, 0, err
		}
		revision, err := u.Coordination().AdvanceOutcome(ctx, scope)
		if err != nil {
			return zero, 0, err
		}
		local := map[string]string{}
		for k, v := range registered.LocalKeys {
			local[k] = v.String()
		}
		receipt := signing.ReceiptPayload{Binding: signing.Binding{ProtocolVersion: 2, ServerID: server.ID.String(), NamespaceID: scope.NamespaceID.String(), OutcomeID: scope.OutcomeID.String(), PrincipalID: identity.PrincipalID, SignerKeyID: server.IssuerKeyID.String()}, LocalKeys: local, RequestID: request.RequestID, IdempotencyKey: cc.IdempotencyKey, RequestDigest: digest, ContractID: contract.ID.String(), WorkItemID: work.ID.String(), SubmissionID: submission.ID.String(), SubmissionDigest: submission.Digest, ReviewCaseID: review.ID.String(), DecisionID: decisionID.String(), OutcomeRevision: signing.Decimal(revision), Accepted: true, Disposition: disposition, ContractStatus: string(contract.Status), WorkItemLifecycle: string(work.Lifecycle), LocalObligationClosed: true, AcceptedAt: now.UTC().Format(time.RFC3339Nano)}
		acceptance, receiptDoc, err := s.issueSignedFact(ctx, u, server, scope, contract.ID, "review", "acceptance", identity.PrincipalID, signing.AcceptanceReceipt, receipt, now)
		if err != nil {
			return zero, 0, err
		}
		acceptance.RequestID = &requestID
		acceptance.IdempotencyKey = cc.IdempotencyKey
		acceptance.RequestDigest = digest
		if err = repo.InsertFact(ctx, acceptance); err != nil {
			return zero, 0, err
		}
		registered.WorkItem = work
		registered.Submission = &submission
		registered.EvaluatedAt = now
		return SignedReturnResult{Receipt: receiptDoc, work: &registered, review: &review, reviewAuthority: &contract, reviewDecision: request.Decision}, revision, nil
	})
}
func validateComplementaryReviewSource(target d.WorkResultMaterial, input SignedReviewMaterial) error {
	exact := false
	for _, a := range target.Artifacts {
		exact = exact || input.ReviewedSourceVersion != "" && (input.ReviewedSourceVersion == a.SourceVersion || input.ReviewedSourceVersion == a.Checksum)
	}
	if !exact {
		return d.NewError(d.ErrorCodeSubmissionNotAccepted, "complementary review records need the immutable reviewed source version/checksum")
	}
	for _, a := range input.Artifacts {
		if a.Artifact.SourceVersion != input.ReviewedSourceVersion && a.Artifact.Checksum != input.ReviewedSourceVersion {
			return d.NewError(d.ErrorCodeSubmissionNotAccepted, "review artifact differs from reviewed commit")
		}
	}
	for _, e := range input.Evidence {
		if e.Evidence.SourceVersion != input.ReviewedSourceVersion && e.Evidence.Checksum != input.ReviewedSourceVersion {
			return d.NewError(d.ErrorCodeSubmissionNotAccepted, "review evidence differs from reviewed commit")
		}
	}
	return nil
}

func signedReviewReturnEvents(s *Service, cc d.CommandContext, meta commandMetadata, result SignedReturnResult, revision d.OutcomeRevision) ([]d.DomainEvent, error) {
	work := result.work
	typ := "work_review.inconclusive"
	if result.reviewDecision == "approved" {
		typ = "work_review.approved"
	}
	if result.reviewDecision == "changes_requested" {
		typ = "work_review.changes_requested"
	}
	specs := []compoundEventSpec{}
	for _, a := range work.Artifacts {
		specs = append(specs, compoundEventSpec{eventType: "artifact.registered", ref: a.Ref(), after: versionPtr(a.Version)})
	}
	for _, e := range work.Evidence {
		specs = append(specs, compoundEventSpec{eventType: "evidence.registered", ref: e.Ref(), after: versionPtr(e.Version)})
	}
	for _, l := range work.EvidenceLinks {
		specs = append(specs, compoundEventSpec{eventType: "evidence.link_created", ref: l.Ref(), after: versionPtr(l.Version)})
	}
	if result.reviewDecision == "approved" {
		for range work.WorkItem.Criteria.CurrentAssessments {
			specs = append(specs, compoundEventSpec{eventType: "work_item.assessment_recorded", ref: work.WorkItem.Ref(), after: versionPtr(work.WorkItem.Version)})
		}
	}
	specs = append(specs, compoundEventSpec{eventType: typ, ref: work.WorkItem.Ref(), after: versionPtr(work.WorkItem.Version)})
	if result.reviewDecision == "approved" {
		specs = append(specs, compoundEventSpec{eventType: "work_item.completed", ref: work.WorkItem.Ref(), after: versionPtr(work.WorkItem.Version)}, compoundEventSpec{eventType: "work_item.conclusion_recorded", ref: work.WorkItem.Ref(), after: versionPtr(work.WorkItem.Version)})
	}
	events, err := buildCompoundEvents(s, cc, meta, revision, specs)
	if err != nil {
		return nil, err
	}
	for i := range events {
		raw, e := json.Marshal(map[string]any{"review_case_id": result.review.ID, "review_contract_id": result.reviewAuthority.ID, "decision_id": result.review.LatestDecisionID, "submission_id": work.Submission.ID, "submission_digest": work.Submission.Digest, "decision": result.reviewDecision, "status": result.review.Status, "receipt_digest": signing.Digest(result.Receipt.Payload)})
		if events[i].EventType == "work_item.conclusion_recorded" {
			raw, e = conclusionRecordedPayload(work.WorkItem)
		}
		if e != nil {
			return nil, e
		}
		events[i].Payload = raw
		events[i].RecordedAt = work.EvaluatedAt
		if e = events[i].Validate(); e != nil {
			return nil, e
		}
	}
	return events, nil
}
