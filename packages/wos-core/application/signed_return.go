package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"reflect"
	"slices"
	"time"
)

// The envelope is the entire command. Scope hints only select the transaction;
// no hint is executed until the same payload bytes have been verified in it.
type ReturnSignedWorkCommand struct{ Envelope signing.Envelope }
type SignedAssessmentInput struct {
	CriterionID       d.ID               `json:"criterion_id"`
	CriterionRevision signing.Decimal    `json:"criterion_revision"`
	Result            d.AssessmentResult `json:"result"`
	Rationale         string             `json:"rationale"`
	EvidenceIDs       []d.ID             `json:"evidence_ids,omitempty"`
	EvidenceLocalKeys []string           `json:"evidence_local_keys,omitempty"`
	EvaluatorRef      *d.EvaluatorRef    `json:"evaluator_ref,omitempty"`
}
type SignedLocalCriterionEvidence struct {
	CriterionID       d.ID            `json:"criterion_id"`
	CriterionRevision signing.Decimal `json:"criterion_revision"`
	EvidenceLocalKeys []string        `json:"evidence_local_keys"`
}
type SignedReturnMaterial struct {
	CorrectionResponses    []SignedFindingResponse        `json:"correction_responses,omitempty"`
	Result                 d.SignedResultMaterial         `json:"result"`
	Artifacts              []SyncArtifactInput            `json:"artifacts,omitempty"`
	Evidence               []SyncEvidenceInput            `json:"evidence,omitempty"`
	EvidenceLinks          []SyncEvidenceLinkInput        `json:"evidence_links,omitempty"`
	ArtifactLocalKeys      []string                       `json:"artifact_local_keys,omitempty"`
	EvidenceLocalKeys      []string                       `json:"evidence_local_keys,omitempty"`
	CriterionLocalEvidence []SignedLocalCriterionEvidence `json:"criterion_local_evidence,omitempty"`
	Assessments            []SignedAssessmentInput        `json:"criterion_assessments,omitempty"`
	Reason                 string                         `json:"reason,omitempty"`
}
type SignedReturnResult struct {
	reviewAuthority *d.ReviewContract
	reviewDecision  string
	Receipt         signing.Document `json:"acceptance_receipt"`
	// Runtime-only event composition; never expose unsigned copies as proof.
	work          *WorkContractResult
	review        *d.ReviewCase
	durableReplay bool
}

func (r SignedReturnResult) IsDurableReplay() bool { return r.durableReplay }
func signedRequestHint[T any](envelope signing.Envelope, target *T) error {
	raw, err := json.Marshal(envelope)
	if err != nil {
		return err
	}
	if len(raw) > 256*1024 {
		return d.NewError(d.ErrorCodeInvalidArgument, "signed envelope exceeds 256 KiB")
	}
	payload, err := base64.StdEncoding.Strict().DecodeString(envelope.Payload)
	if err != nil {
		return signingError(err)
	}
	return signingError(signing.DecodeStrict(payload, target, 180*1024))
}
func signedRequestScope(b signing.RequestBinding) (d.Scope, error) {
	scope := d.Scope{NamespaceID: d.ID(b.NamespaceID), OutcomeID: d.ID(b.OutcomeID)}
	if err := scope.Validate(); err != nil {
		return scope, err
	}
	if b.ProtocolVersion != 2 || b.PrincipalID == "" || d.ID(b.RequestID).Validate() != nil || d.ID(b.ContractID).Validate() != nil || d.ID(b.WorkItemID).Validate() != nil || d.ID(b.ExecutionID).Validate() != nil || d.ID(b.SignerKeyID).Validate() != nil || d.ValidateIdempotencyKey(b.IdempotencyKey) != nil || b.FencingToken == 0 || b.ExpectedContractVersion == 0 || b.ExpectedLeaseVersion == 0 || b.ExpectedWorkItemVersion == 0 || b.PolicyRevision == 0 || !d.ValidSignedDigest(b.SpecDigest) || !d.ValidSignedDigest(b.AuthorityDigest) {
		return scope, d.NewError(d.ErrorCodeInvalidArgument, "signed request bindings are incomplete")
	}
	return scope, nil
}
func (s *Service) ReturnSignedWork(ctx context.Context, cc d.CommandContext, cmd ReturnSignedWorkCommand) (MutationResult[SignedReturnResult], error) {
	var hint signing.WorkReturnPayload[SignedReturnMaterial]
	if err := signedRequestHint(cmd.Envelope, &hint); err != nil {
		return MutationResult[SignedReturnResult]{}, err
	}
	if _, err := signedRequestScope(hint.RequestBinding); err != nil {
		return MutationResult[SignedReturnResult]{}, err
	}
	if cc.PrincipalID != hint.PrincipalID || cc.IdempotencyKey != hint.IdempotencyKey {
		return MutationResult[SignedReturnResult]{}, d.NewError(d.ErrorCodeForbidden, "request identity/idempotency differs from command context")
	}
	// The signed request identity is stable across transport attempts.
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
			return zero, 0, d.NewError(d.ErrorCodeForbidden, "return key belongs to another operational identity")
		}
		pub, err := key.PublicBytes()
		if err != nil {
			return zero, 0, err
		}
		raw, err := signing.Verify(cmd.Envelope, signing.WorkReturn, key.ID.String(), pub)
		if err != nil {
			return zero, 0, signingError(err)
		}
		var request signing.WorkReturnPayload[SignedReturnMaterial]
		if err = signing.DecodeStrict(raw, &request, 180*1024); err != nil {
			return zero, 0, signingError(err)
		}
		if request.OperationKind != "work_return" {
			return zero, 0, d.NewError(d.ErrorCodeInvalidArgument, "wrong signed operation purpose")
		}
		digest := signing.Digest(raw)
		if prior, e := repo.Acceptance(ctx, scope.NamespaceID, cc.PrincipalID, cc.IdempotencyKey); e == nil {
			if e = s.requireSignedAcceptedCredential(ctx, u, scope, "execution", d.ID(request.ContractID)); e != nil {
				return zero, 0, e
			}
			if prior.Scope != scope || prior.ContractKind != "execution" || prior.RequestDigest != digest || prior.RequestID == nil || *prior.RequestID != d.ID(request.RequestID) {
				return zero, 0, d.NewError(d.ErrorCodeIdempotencyConflict, "confirmed signed intent differs")
			}
			document, e := verifyIssuedFact(ctx, u, prior, signing.AcceptanceReceipt, request.ServerID, cc.PrincipalID)
			if e != nil {
				return zero, 0, e
			}
			var receipt signing.ReceiptPayload
			if e = signing.DecodeStrict(document.Payload, &receipt, 180*1024); e != nil {
				return zero, 0, signingError(e)
			}
			return SignedReturnResult{Receipt: document, durableReplay: true}, d.OutcomeRevision(receipt.OutcomeRevision), nil
		} else if code, _ := d.ErrorCodeOf(e); code != d.ErrorCodeNotFound {
			return zero, 0, e
		}
		identity, policy, err := s.signedAccess(ctx, u, scope, ports.PermissionWorkContractReturn, key.ID)
		if err != nil {
			return zero, 0, err
		}
		server, err := s.requireSignedIssuer(ctx, u, scope.NamespaceID)
		if err != nil {
			return zero, 0, err
		}
		if request.ServerID != server.ID.String() || request.PolicyRevision != signing.Decimal(policy.Version) {
			return zero, 0, d.NewError(d.ErrorCodeVersionConflict, "server trust or live policy changed")
		}
		if _, err = u.Coordination().LockOutcome(ctx, scope); err != nil {
			return zero, 0, err
		}
		if err = requireActiveOutcome(ctx, u, scope); err != nil {
			return zero, 0, err
		}
		contract, err := workRepo.Get(ctx, scope, d.ID(request.ContractID))
		if err != nil {
			return zero, 0, err
		}
		if contract.SignedBinding == nil || contract.SignedBinding.CredentialID != identity.CredentialID || contract.HolderPrincipalID != identity.PrincipalID || contract.WorkItemID != d.ID(request.WorkItemID) {
			return zero, 0, d.NewError(d.ErrorCodeForbidden, "return does not own this credential authority")
		}
		authority := ContractAuthority{ExecutionID: d.ID(request.ExecutionID), FencingToken: d.FencingToken(request.FencingToken), SpecDigest: request.SpecDigest}
		if err = requireIssuedWorkAuthority(ctx, u, contract, authority, request.AuthorityDigest); err != nil {
			return zero, 0, err
		}
		if contract.Version != d.Version(request.ExpectedContractVersion) || contract.LeaseVersion != d.Version(request.ExpectedLeaseVersion) {
			return zero, 0, d.NewError(d.ErrorCodeVersionConflict, "signed contract CAS changed")
		}
		now, err := securityTransactionTime(ctx, u, s.clock)
		if err != nil {
			return zero, 0, err
		}
		if err = contract.Authorize(identity.PrincipalID, authority.ExecutionID, authority.FencingToken, now); err != nil {
			return zero, 0, err
		}
		work, err := u.WorkItems().Get(ctx, scope, contract.WorkItemID)
		if err != nil {
			return zero, 0, err
		}
		if work.Version != d.Version(request.ExpectedWorkItemVersion) || work.CurrentContractID == nil || *work.CurrentContractID != contract.ID || work.Lifecycle != d.WorkItemLifecycleInProgress {
			return zero, 0, d.NewError(d.ErrorCodeVersionConflict, "signed Task CAS/authority changed")
		}
		// Derive obligations from the authenticated immutable specification, rather
		// than execute a separately supplied or persisted unsigned command/spec.
		fact, err := repo.Fact(ctx, scope, *contract.IssuedSpecificationID)
		if err != nil {
			return zero, 0, err
		}
		doc, err := verifyIssuedFact(ctx, u, fact, signing.ContractSpec, server.ID.String(), identity.PrincipalID)
		if err != nil {
			return zero, 0, err
		}
		var spec signing.SpecPayload[d.SignedWorkSpec]
		if err = signing.DecodeStrict(doc.Payload, &spec, 128*1024); err != nil {
			return zero, 0, signingError(err)
		}
		if fact.PayloadDigest != request.SpecDigest || spec.ContractID != contract.ID.String() || spec.WorkItemID != work.ID.String() || spec.ContractKind != "execution" || !reflect.DeepEqual(spec.Spec.WorkSpec(), d.NormalizeContractSpec(contract.Spec)) {
			return zero, 0, d.NewError(d.ErrorCodeContractSpecMismatch, "authenticated specification differs")
		}
		if spec.Spec.AcceptanceFloor != contract.SignedBinding.AcceptanceFloor || spec.Spec.PolicyRevision != contract.SignedBinding.PolicyRevision || !reflect.DeepEqual(spec.Spec.PreviousSubmissionID, contract.SignedBinding.PreviousSubmissionID) || !reflect.DeepEqual(spec.Spec.PreviousReviewCaseID, contract.SignedBinding.PreviousReviewCaseID) || !reflect.DeepEqual(spec.Spec.CorrectionFindings, contract.SignedBinding.CorrectionFindings) {
			return zero, 0, d.NewError(d.ErrorCodeContractSpecMismatch, "signed frozen policy/correction bindings differ")
		}
		contract.Spec = spec.Spec.WorkSpec()
		if !reflect.DeepEqual(contract.Spec.Criteria, d.NormalizeContractSpec(d.WorkContractSpec{Criteria: work.Criteria.Items}).Criteria) {
			return zero, 0, d.NewError(d.ErrorCodeContractSpecMismatch, "live criterion definitions differ from signed obligations")
		}
		grantFact, err := repo.Fact(ctx, scope, *contract.LatestAuthorityID)
		if err != nil {
			return zero, 0, err
		}
		var grant signing.AuthorityPayload
		if err = signing.DecodeStrict(grantFact.Payload, &grant, 128*1024); err != nil {
			return zero, 0, signingError(err)
		}
		allowedKey := false
		for _, id := range grant.AllowedSigningKeyIDs {
			allowedKey = allowedKey || id == key.ID.String()
		}
		if !allowedKey {
			return zero, 0, d.NewError(d.ErrorCodeForbidden, "return signer is outside the issued grant")
		}
		floor, err := s.signedAcceptanceFloor(ctx, u, scope, work.ID, policy, contract.SignedBinding.AcceptanceFloor)
		if err != nil {
			return zero, 0, err
		}
		direct := false
		switch request.CompletionIntent {
		case "auto":
			direct = floor == d.AcceptanceDirect
		case "review":
		case "direct":
			if floor != d.AcceptanceDirect {
				return zero, 0, d.NewError(d.ErrorCodeForbidden, "independent review floor cannot be weakened")
			}
			direct = true
		default:
			return zero, 0, d.NewError(d.ErrorCodeInvalidArgument, "completion intent must be auto/direct/review")
		}
		if direct {
			for _, permission := range []ports.Permission{ports.PermissionWorkCompleteDirect, ports.PermissionAssessmentWrite, ports.PermissionConclusionWrite} {
				if _, _, err = s.signedAccess(ctx, u, scope, permission, key.ID); err != nil {
					return zero, 0, err
				}
			}
		} else if len(request.Material.Assessments) > 0 {
			return zero, 0, d.NewError(d.ErrorCodeForbidden, "review delivery cannot certify its own criteria")
		}
		if len(request.Material.Artifacts)+len(request.Material.Evidence)+len(request.Material.EvidenceLinks) > 0 {
			if _, _, err = s.signedAccess(ctx, u, scope, ports.PermissionRecordsWrite, key.ID); err != nil {
				return zero, 0, err
			}
		}
		material := request.Material.Result.Material()
		if material.ContractID != contract.ID || material.WorkItemID != work.ID || material.SpecDigest != request.SpecDigest {
			return zero, 0, d.NewError(d.ErrorCodeContractSpecMismatch, "material binding differs")
		}
		registered, err := s.registerContractRecords(ctx, u, cc, SyncWorkContractCommand{Scope: scope, Artifacts: request.Material.Artifacts, Evidence: request.Material.Evidence, EvidenceLinks: request.Material.EvidenceLinks}, now)
		if err != nil {
			return zero, 0, err
		}
		if err = resolveSignedMaterial(&material, request.Material, registered); err != nil {
			return zero, 0, err
		}
		if err = validateCorrectionResponses(contract.SignedBinding.CorrectionFindings, request.Material.CorrectionResponses, material, registered.LocalKeys); err != nil {
			return zero, 0, err
		}
		if err = validateSubmittedMaterial(ctx, u, work, material); err != nil {
			return zero, 0, err
		}
		submissionID, err := s.ids.NewID()
		if err != nil {
			return zero, 0, err
		}
		materialDigest, err := d.SignedResultDigest(material)
		if err != nil {
			return zero, 0, err
		}
		submission := d.WorkSubmission{ProtocolVersion: 2, ID: submissionID, Scope: scope, Material: material, Digest: materialDigest, PrincipalID: identity.PrincipalID, Actor: cc.Actor, SubmittedAt: now}
		if request.SupersedesSubmissionID != "" {
			id := d.ID(request.SupersedesSubmissionID)
			submission.SupersedesSubmissionID = &id
		}
		if !reflect.DeepEqual(submission.SupersedesSubmissionID, contract.SignedBinding.PreviousSubmissionID) {
			return zero, 0, d.NewError(d.ErrorCodeSubmissionNotAccepted, "correction must bind the frozen prior submission")
		}
		if err = submission.Validate(); err != nil {
			return zero, 0, err
		}
		cv, lv, wv := contract.Version, contract.LeaseVersion, work.Version
		var review *d.ReviewCase
		disposition := "completed"
		if direct {
			if err = s.recordSignedAssessments(ctx, u, cc, &work, submission, request.Material.Assessments, registered.LocalKeys, now); err != nil {
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
			id, e := s.ids.NewID()
			if e != nil {
				return zero, 0, e
			}
			reason := request.Material.Reason
			if reason == "" {
				reason = material.Summary
			}
			conclusion := conclusionFromContext(id, cc, reason, now)
			if err = contract.CompleteSignedDirect(&work, submission, identity.PrincipalID, authority.ExecutionID, authority.FencingToken, cv, wv, conclusion, now); err != nil {
				return zero, 0, err
			}
		} else {
			if err = contract.Deliver(submission, identity.PrincipalID, authority.ExecutionID, authority.FencingToken, cv, now); err != nil {
				return zero, 0, err
			}
			id, e := s.ids.NewID()
			if e != nil {
				return zero, 0, e
			}
			groups := []string{}
			if contract.SignedBinding.SeparationGroup != "" {
				groups = append(groups, contract.SignedBinding.SeparationGroup)
			}
			review = &d.ReviewCase{ID: id, Scope: scope, WorkItemID: work.ID, WorkContractID: contract.ID, SubmissionID: submission.ID, SubmissionDigest: submission.Digest, IssuedSpecDigest: request.SpecDigest, PolicyRevision: policy.Version, AcceptanceFloor: d.AcceptanceIndependentReview, Round: 1, Version: 1, Status: d.ReviewPending, ExecutionPrincipals: []string{identity.PrincipalID}, ExecutionGroups: groups, CreatedAt: now, UpdatedAt: now}
			// Correction histories are assembled from their immutable prior review case.
			if previous := contract.SignedBinding.PreviousReviewCaseID; previous != nil {
				prior, e := repo.Case(ctx, scope, *previous)
				if e != nil {
					return zero, 0, e
				}
				review.Round = prior.Round + 1
				if review.Round == 0 {
					return zero, 0, d.NewError(d.ErrorCodeInvalidArgument, "review round exhausted")
				}
				review.ExecutionPrincipals = uniqueSignedStrings(append(prior.ExecutionPrincipals, review.ExecutionPrincipals...))
				review.ExecutionGroups = uniqueSignedStrings(append(prior.ExecutionGroups, groups...))
			}
			participants, e := repo.ExecutionParticipants(ctx, scope, work.ID, 101)
			if e != nil {
				return zero, 0, e
			}
			if len(participants) > 100 {
				return zero, 0, d.NewError(d.ErrorCodeGraphLimitExceeded, "execution independence history exceeds bounded participant limit")
			}
			for _, p := range participants {
				review.ExecutionPrincipals = uniqueSignedStrings(append(review.ExecutionPrincipals, p.PrincipalID))
				if p.SeparationGroup != "" {
					review.ExecutionGroups = uniqueSignedStrings(append(review.ExecutionGroups, p.SeparationGroup))
				}
			}
			if err = review.Validate(); err != nil {
				return zero, 0, err
			}
			disposition = "delivered_for_review"
			work.PendingReviewCaseID = &review.ID
			work.LatestReviewCaseID = &review.ID
			work.CorrectionReviewCaseID = nil
			work.CurrentLease = nil
			next, e := work.Version.Next()
			if e != nil {
				return zero, 0, e
			}
			work.Version = next
			work.UpdatedAt = now
		}
		if err = workRepo.InsertSubmission(ctx, submission); err != nil {
			return zero, 0, err
		}
		if err = workRepo.Save(ctx, contract, cv, lv); err != nil {
			return zero, 0, err
		}
		if err = u.WorkItems().Save(ctx, work, wv); err != nil {
			return zero, 0, err
		}
		if review != nil {
			if err = repo.InsertCase(ctx, *review); err != nil {
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
		id, err := s.ids.NewID()
		if err != nil {
			return zero, 0, err
		}
		requestID := d.ID(request.RequestID)
		returnFact := d.SignedFact{ID: id, Scope: scope, ContractID: contract.ID, ContractKind: "execution", Kind: "return", PrincipalID: identity.PrincipalID, KeyID: key.ID, RequestID: &requestID, IdempotencyKey: cc.IdempotencyKey, RequestDigest: digest, PayloadDigest: digest, Payload: raw, Proof: proof, RecordedAt: now}
		if err = repo.InsertFact(ctx, returnFact); err != nil {
			return zero, 0, err
		}
		revision, err := u.Coordination().AdvanceOutcome(ctx, scope)
		if err != nil {
			return zero, 0, err
		}
		localKeys := map[string]string{}
		for k, v := range registered.LocalKeys {
			localKeys[k] = v.String()
		}
		receipt := signing.ReceiptPayload{Binding: signing.Binding{ProtocolVersion: 2, ServerID: server.ID.String(), NamespaceID: scope.NamespaceID.String(), OutcomeID: scope.OutcomeID.String(), PrincipalID: identity.PrincipalID, SignerKeyID: server.IssuerKeyID.String()}, LocalKeys: localKeys, RequestID: request.RequestID, IdempotencyKey: cc.IdempotencyKey, RequestDigest: digest, ContractID: contract.ID.String(), WorkItemID: work.ID.String(), SubmissionID: submission.ID.String(), SubmissionDigest: submission.Digest, OutcomeRevision: signing.Decimal(revision), Accepted: true, Disposition: disposition, ContractStatus: string(contract.Status), WorkItemLifecycle: string(work.Lifecycle), LocalObligationClosed: true, AcceptedAt: now.UTC().Format(time.RFC3339Nano)}
		if review != nil {
			receipt.ReviewCaseID = review.ID.String()
		}
		acceptance, document, err := s.issueSignedFact(ctx, u, server, scope, contract.ID, "execution", "acceptance", identity.PrincipalID, signing.AcceptanceReceipt, receipt, now)
		if err != nil {
			return zero, 0, err
		}
		acceptance.RequestID = &requestID
		acceptance.IdempotencyKey = cc.IdempotencyKey
		acceptance.RequestDigest = digest
		if err = repo.InsertFact(ctx, acceptance); err != nil {
			return zero, 0, err
		}
		registered.Contract = contract
		registered.WorkItem = work
		registered.Submission = &submission
		registered.EvaluatedAt = now
		return SignedReturnResult{Receipt: document, work: &registered, review: review}, revision, nil
	})
}
func uniqueSignedStrings(values []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, v := range values {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}
func resolveSignedMaterial(m *d.WorkResultMaterial, input SignedReturnMaterial, registered WorkContractResult) error {
	artifacts := map[d.ID]d.Artifact{}
	for _, a := range registered.Artifacts {
		artifacts[a.ID] = a
	}
	evidence := map[d.ID]bool{}
	for _, e := range registered.Evidence {
		evidence[e.ID] = true
	}
	for _, key := range input.ArtifactLocalKeys {
		id, ok := registered.LocalKeys[key]
		a, kind := artifacts[id]
		if !ok || !kind {
			return d.NewError(d.ErrorCodeInvalidArgument, "unknown artifact local key")
		}
		m.Artifacts = append(m.Artifacts, d.SubmissionArtifact{ArtifactID: id, SourceVersion: a.SourceVersion, Checksum: a.Checksum})
	}
	for _, key := range input.EvidenceLocalKeys {
		id, ok := registered.LocalKeys[key]
		if !ok || !evidence[id] {
			return d.NewError(d.ErrorCodeInvalidArgument, "unknown evidence local key")
		}
		m.EvidenceIDs = append(m.EvidenceIDs, id)
	}
	for _, ref := range input.CriterionLocalEvidence {
		ids := []d.ID{}
		for _, key := range ref.EvidenceLocalKeys {
			id, ok := registered.LocalKeys[key]
			if !ok || !evidence[id] {
				return d.NewError(d.ErrorCodeInvalidArgument, "unknown criterion evidence local key")
			}
			ids = append(ids, id)
			// An explicit criterion-local reference selects this registered
			// evidence for the same submission; do not require a duplicate
			// selection in the general evidence_local_keys list.
			if !slices.Contains(m.EvidenceIDs, id) {
				m.EvidenceIDs = append(m.EvidenceIDs, id)
			}
		}
		m.CriterionEvidence = append(m.CriterionEvidence, d.SubmissionCriterionEvidence{CriterionID: ref.CriterionID, CriterionRevision: d.CriterionRevision(ref.CriterionRevision), EvidenceIDs: ids})
	}
	return nil
}
func (s *Service) recordSignedAssessments(ctx context.Context, u ports.UnitOfWork, cc d.CommandContext, w *d.WorkItem, submission d.WorkSubmission, inputs []SignedAssessmentInput, localKeys map[string]d.ID, now time.Time, complementary ...[]d.SubmissionCriterionEvidence) error {
	if len(inputs) > 100 {
		return d.NewError(d.ErrorCodeInvalidArgument, "assessment limit is 100")
	}
	seen := map[d.ID]bool{}
	allowed := map[d.ID]bool{}
	for _, id := range submission.Material.EvidenceIDs {
		allowed[id] = true
	}
	criterionEvidence := append([]d.SubmissionCriterionEvidence(nil), submission.Material.CriterionEvidence...)
	for _, list := range complementary {
		for _, ref := range list {
			found := false
			for _, c := range w.Criteria.Items {
				found = found || (c.ID == ref.CriterionID && c.Revision == ref.CriterionRevision && c.Status == d.CriterionStatusActive)
			}
			if !found {
				return d.NewError(d.ErrorCodeSubmissionNotAccepted, "complementary review evidence targets an unplanned/stale criterion")
			}
			for _, id := range ref.EvidenceIDs {
				allowed[id] = true
			}
			criterionEvidence = append(criterionEvidence, ref)
		}
	}
	// An older assessment never proves new returned material.
	w.Criteria.CurrentAssessments = map[d.ID]d.CriterionAssessment{}
	for _, input := range inputs {
		if seen[input.CriterionID] {
			return d.NewError(d.ErrorCodeInvalidArgument, "duplicate criterion assessment")
		}
		seen[input.CriterionID] = true
		ids := append([]d.ID(nil), input.EvidenceIDs...)
		for _, key := range input.EvidenceLocalKeys {
			id, ok := localKeys[key]
			if !ok {
				return d.NewError(d.ErrorCodeInvalidArgument, "unknown assessment local key")
			}
			ids = append(ids, id)
		}
		for _, id := range ids {
			criterionBound := false
			for _, ref := range criterionEvidence {
				if ref.CriterionID == input.CriterionID && ref.CriterionRevision == d.CriterionRevision(input.CriterionRevision) {
					for _, eid := range ref.EvidenceIDs {
						criterionBound = criterionBound || eid == id
					}
				}
			}
			if !criterionBound {
				return d.NewError(d.ErrorCodeSubmissionNotAccepted, "assessment evidence is outside submitted criterion")
			}
			if !allowed[id] {
				return d.NewError(d.ErrorCodeSubmissionNotAccepted, "assessment evidence is outside the target submission")
			}
		}
		if err := validateAssessmentEvidence(ctx, u, w.Scope, ids); err != nil {
			return err
		}
		waiver := input.Result == d.AssessmentResultWaived
		if waiver {
			if _, _, err := s.signedAccess(ctx, u, w.Scope, ports.PermissionAssessmentWaive, d.ID("")); err != nil {
				return err
			}
		}
		id, err := s.ids.NewID()
		if err != nil {
			return err
		}
		sid := submission.ID
		assessment := d.CriterionAssessment{ID: id, CriterionID: input.CriterionID, CriterionRevision: d.CriterionRevision(input.CriterionRevision), Result: input.Result, Rationale: input.Rationale, EvidenceIDs: ids, EvaluatorRef: input.EvaluatorRef, PrincipalID: cc.PrincipalID, Actor: cc.Actor, AssessedAt: now, SubmissionID: &sid, SubmissionDigest: submission.Digest}
		if err = w.Criteria.RecordAssessment(assessment, waiver); err != nil {
			return err
		}
	}
	return nil
}

func signedReturnEvents(s *Service, cc d.CommandContext, meta commandMetadata, result SignedReturnResult, revision d.OutcomeRevision) ([]d.DomainEvent, error) {
	if result.durableReplay {
		return nil, nil
	}
	if result.work == nil {
		return nil, d.NewError(d.ErrorCodeInvalidEvent, "signed return event material missing")
	}
	if result.reviewAuthority != nil {
		return signedReviewReturnEvents(s, cc, meta, result, revision)
	}
	work := result.work
	specs := []compoundEventSpec{{eventType: "work_contract.result_submitted", ref: work.Contract.Ref(), after: versionPtr(work.Contract.Version)}}
	if result.review != nil {
		specs = append(specs, compoundEventSpec{eventType: "work_contract.delivered", ref: work.Contract.Ref(), after: versionPtr(work.Contract.Version)}, compoundEventSpec{eventType: "work_review.opened", ref: work.WorkItem.Ref(), after: versionPtr(work.WorkItem.Version)})
	} else {
		specs = append(specs, compoundEventSpec{eventType: "work_contract.completed", ref: work.Contract.Ref(), after: versionPtr(work.Contract.Version)}, compoundEventSpec{eventType: "work_item.completed", ref: work.WorkItem.Ref(), after: versionPtr(work.WorkItem.Version)}, compoundEventSpec{eventType: "work_item.conclusion_recorded", ref: work.WorkItem.Ref(), after: versionPtr(work.WorkItem.Version)})
	}
	for _, a := range work.Artifacts {
		specs = append(specs, compoundEventSpec{eventType: "artifact.registered", ref: a.Ref(), after: versionPtr(a.Version)})
	}
	for _, e := range work.Evidence {
		specs = append(specs, compoundEventSpec{eventType: "evidence.registered", ref: e.Ref(), after: versionPtr(e.Version)})
	}
	for _, l := range work.EvidenceLinks {
		specs = append(specs, compoundEventSpec{eventType: "evidence.link_created", ref: l.Ref(), after: versionPtr(l.Version)})
	}
	for range work.WorkItem.Criteria.CurrentAssessments {
		if work.Contract.Status == d.ContractCompleted {
			specs = append(specs, compoundEventSpec{eventType: "work_item.assessment_recorded", ref: work.WorkItem.Ref(), after: versionPtr(work.WorkItem.Version)})
		}
	}
	events, err := buildCompoundEvents(s, cc, meta, revision, specs)
	if err != nil {
		return nil, err
	}
	for i := range events {
		payload, err := json.Marshal(map[string]any{"contract_id": work.Contract.ID, "submission_id": work.Submission.ID, "submission_digest": work.Submission.Digest, "status": work.Contract.Status, "review_case": result.review, "request_receipt_digest": signing.Digest(result.Receipt.Payload)})
		if events[i].EventType == "work_item.conclusion_recorded" {
			payload, err = conclusionRecordedPayload(work.WorkItem)
		}
		if err != nil {
			return nil, err
		}
		events[i].Payload = payload
		events[i].RecordedAt = work.EvaluatedAt
		if err = events[i].Validate(); err != nil {
			return nil, err
		}
	}
	return events, nil
}
