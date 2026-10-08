package application

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"reflect"
	"time"
)

type AcquireSignedReviewContractCommand struct {
	Scope                     d.Scope
	ReviewCaseID              d.ID
	ExpectedReviewCaseVersion d.Version
	SignerKeyID               d.ID
	TTLSeconds                int `wos:"optional"`
}
type SignedMaterialReference struct {
	Kind         string `json:"kind"`
	ContractID   d.ID   `json:"contract_id"`
	SubmissionID *d.ID  `json:"submission_id,omitempty"`
	Digest       string `json:"digest"`
}

type SignedReviewSpec struct {
	Omitted                 []SignedMaterialReference `json:"omitted,omitempty"`
	ExecutionPrincipals     []string                  `json:"execution_principals"`
	ExecutionGroups         []string                  `json:"execution_groups"`
	ReviewCaseID            d.ID                      `json:"review_case_id"`
	SubmissionID            d.ID                      `json:"submission_id"`
	SubmissionDigest        string                    `json:"submission_digest"`
	IssuedWorkSpecDigest    string                    `json:"issued_work_spec_digest"`
	Round                   signing.Decimal           `json:"round"`
	WorkItemVersion         signing.Decimal           `json:"work_item_version"`
	IssuedWorkSpecification *signing.Document         `json:"issued_work_specification,omitempty"`
	AcceptanceReceipt       signing.Document          `json:"delivery_acceptance_receipt"`
	Submission              *d.SignedResultMaterial   `json:"submission,omitempty"`
	AcceptanceFloor         d.AcceptanceMode          `json:"acceptance_floor"`
}
type SignedReviewContractResult struct {
	Contract            d.ReviewContract  `json:"review_contract"`
	Case                d.ReviewCase      `json:"review_case"`
	WorkItemVersion     signing.Decimal   `json:"work_item_version"`
	IssuedSpecification *signing.Document `json:"issued_specification,omitempty"`
	IssuedAuthority     signing.Document  `json:"issued_authority"`
	EvaluatedAt         time.Time         `json:"evaluated_at"`
	expired             *d.ReviewContract
	resumed             bool
}

func (s *Service) AcquireSignedReviewContract(ctx context.Context, cc d.CommandContext, cmd AcquireSignedReviewContractCommand) (MutationResult[SignedReviewContractResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[SignedReviewContractResult]{}, err
	}
	if cc.IdempotencyKey == "" {
		return MutationResult[SignedReviewContractResult]{}, d.NewError(d.ErrorCodeInvalidArgument, "review acquisition requires idempotency")
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (SignedReviewContractResult, d.OutcomeRevision, error) {
		var zero SignedReviewContractResult
		identity, policy, err := s.signedAccess(ctx, u, cmd.Scope, ports.PermissionWorkReviewAcquire, cmd.SignerKeyID)
		if err != nil {
			return zero, 0, err
		}
		server, err := s.requireSignedIssuer(ctx, u, cmd.Scope.NamespaceID)
		if err != nil {
			return zero, 0, err
		}
		if _, err = u.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
			return zero, 0, err
		}
		if err = requireActiveOutcome(ctx, u, cmd.Scope); err != nil {
			return zero, 0, err
		}
		repo, _, err := signedRepository(u)
		if err != nil {
			return zero, 0, err
		}
		review, err := repo.Case(ctx, cmd.Scope, cmd.ReviewCaseID)
		if err != nil {
			return zero, 0, err
		}
		if review.Version != cmd.ExpectedReviewCaseVersion {
			return zero, 0, d.NewError(d.ErrorCodeVersionConflict, "review case CAS changed")
		}
		registry, err := signingRepository(u)
		if err != nil {
			return zero, 0, err
		}
		group, err := registry.PrincipalGroup(ctx, cmd.Scope.NamespaceID, identity.PrincipalID)
		if err != nil {
			return zero, 0, err
		}
		if err = s.requireSignedReviewIndependence(ctx, u, review, identity.PrincipalID, group); err != nil {
			return zero, 0, err
		}
		now, err := securityTransactionTime(ctx, u, s.clock)
		if err != nil {
			return zero, 0, err
		}
		counts, err := repo.Counts(ctx, cmd.Scope.NamespaceID, identity.PrincipalID, identity.CredentialID, now)
		if err != nil {
			return zero, 0, err
		}
		if counts.PrincipalReview >= 1 || counts.CredentialReview >= policy.MaxActiveReviewContracts {
			return zero, 0, d.NewError(d.ErrorCodePreconditionFailed, "review quota reached")
		}
		if np, e := registry.AcceptancePolicy(ctx, cmd.Scope.NamespaceID, nil, nil); e == nil {
			if counts.NamespaceReview >= np.MaxActiveReviewContracts {
				return zero, 0, d.NewError(d.ErrorCodePreconditionFailed, "Namespace review quota reached")
			}
		} else if code, _ := d.ErrorCodeOf(e); code != d.ErrorCodeNotFound {
			return zero, 0, e
		}
		var expired *d.ReviewContract
		if review.CurrentContractID != nil {
			old, e := repo.ReviewContract(ctx, cmd.Scope, *review.CurrentContractID)
			if e != nil {
				return zero, 0, e
			}
			if old.ValidAt(now) {
				return zero, 0, d.NewError(d.ErrorCodeWorkAlreadyClaimed, "review authority remains active")
			}
			v, l, rv := old.Version, old.LeaseVersion, review.Version
			if e = old.Close(d.ContractExpired, "system:expiry", "review lease expired", v, now); e != nil {
				return zero, 0, e
			}
			if e = repo.SaveReviewContract(ctx, old, v, l); e != nil {
				return zero, 0, e
			}
			if e = review.Release(old, now); e != nil {
				return zero, 0, e
			}
			if e = repo.SaveCase(ctx, review, rv); e != nil {
				return zero, 0, e
			}
			expired = &old
		}
		target, work, err := s.authenticatedReviewTarget(ctx, u, review, server.ID.String())
		if err != nil {
			return zero, 0, err
		}
		id, err := s.ids.NewID()
		if err != nil {
			return zero, 0, err
		}
		execution, err := s.ids.NewID()
		if err != nil {
			return zero, 0, err
		}
		allowed, err := liveSignedKeys(ctx, u, policy)
		if err != nil {
			return zero, 0, err
		}
		binding := d.SignedContractBinding{ProtocolVersion: 2, ServerID: server.ID.String(), CredentialID: identity.CredentialID, SignerKeyID: cmd.SignerKeyID, AcceptanceFloor: d.AcceptanceIndependentReview, PolicyRevision: policy.Version, AllowedSigningKeyIDs: allowed, SeparationGroup: group}
		protocol, err := protocolRepository(u)
		if err != nil {
			return zero, 0, err
		}
		state, err := protocol.Lock(ctx, cmd.Scope.NamespaceID, true)
		if err != nil {
			return zero, 0, err
		}
		contract, err := d.NewReviewContract(id, execution, review, identity.PrincipalID, identity.Actor, group, binding, state.LeasePolicy, cmd.TTLSeconds, now)
		if err != nil {
			return zero, 0, err
		}
		payload := signing.SpecPayload[SignedReviewSpec]{Binding: signing.Binding{ProtocolVersion: 2, ServerID: server.ID.String(), NamespaceID: cmd.Scope.NamespaceID.String(), OutcomeID: cmd.Scope.OutcomeID.String(), PrincipalID: identity.PrincipalID, SignerKeyID: server.IssuerKeyID.String()}, ContractKind: "review", ContractID: id.String(), WorkItemID: work.ID.String(), Spec: target}
		payload.Spec, err = compactSignedReviewSpec(payload)
		if err != nil {
			return zero, 0, err
		}
		specFact, specDoc, err := s.issueSignedFact(ctx, u, server, cmd.Scope, id, "review", "specification", identity.PrincipalID, signing.ContractSpec, payload, now)
		if err != nil {
			return zero, 0, err
		}
		contract.Binding.SpecificationDigest = specFact.PayloadDigest
		contract.IssuedSpecificationID = &specFact.ID
		grantFact, grantDoc, err := s.issueReviewAuthority(ctx, u, server, contract, policy, allowed, now)
		if err != nil {
			return zero, 0, err
		}
		contract.LatestAuthorityID = &grantFact.ID
		if err = repo.InsertReviewContract(ctx, contract); err != nil {
			return zero, 0, err
		}
		rv := review.Version
		if err = review.Bind(contract, now); err != nil {
			return zero, 0, err
		}
		if err = repo.SaveCase(ctx, review, rv); err != nil {
			return zero, 0, err
		}
		if err = repo.InsertFact(ctx, specFact); err != nil {
			return zero, 0, err
		}
		if err = repo.InsertFact(ctx, grantFact); err != nil {
			return zero, 0, err
		}
		revision, err := u.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return SignedReviewContractResult{Contract: contract, Case: review, WorkItemVersion: signing.Decimal(work.Version), IssuedSpecification: &specDoc, IssuedAuthority: grantDoc, EvaluatedAt: now, expired: expired}, revision, err
	})
}
func liveSignedKeys(ctx context.Context, u ports.UnitOfWork, p d.CredentialPolicy) ([]d.ID, error) {
	registry, err := signingRepository(u)
	if err != nil {
		return nil, err
	}
	out := []d.ID{}
	for _, id := range p.AllowedSigningKeyIDs {
		k, e := registry.Key(ctx, p.NamespaceID, id)
		if e != nil {
			return nil, e
		}
		if k.Purpose == "agent" && k.Status == "active" && k.PrincipalID == p.PrincipalID {
			out = append(out, id)
		}
	}
	return out, nil
}
func (s *Service) requireSignedReviewIndependence(ctx context.Context, u ports.UnitOfWork, r d.ReviewCase, principal, group string) error {
	if err := r.RequireIndependent(principal, group); err != nil {
		return err
	}
	registry, err := signingRepository(u)
	if err != nil {
		return err
	}
	if group != "" {
		for _, executor := range r.ExecutionPrincipals {
			current, e := registry.PrincipalGroup(ctx, r.Scope.NamespaceID, executor)
			if e != nil {
				return e
			}
			if current == group {
				return d.NewError(d.ErrorCodeForbidden, "reviewer shares current execution separation group")
			}
		}
	}
	return s.requireIndependentReview(ctx, u, r.Scope, principal)
}
func (s *Service) authenticatedReviewTarget(ctx context.Context, u ports.UnitOfWork, r d.ReviewCase, serverID string) (SignedReviewSpec, d.WorkItem, error) {
	var zero SignedReviewSpec
	var work d.WorkItem
	repo, workRepo, err := signedRepository(u)
	if err != nil {
		return zero, work, err
	}
	source, err := workRepo.Get(ctx, r.Scope, r.WorkContractID)
	if err != nil {
		return zero, work, err
	}
	if source.Status != d.ContractDelivered || source.LatestSubmissionID == nil || *source.LatestSubmissionID != r.SubmissionID || source.SignedBinding == nil || source.SignedBinding.ServerID != serverID || source.IssuedSpecificationID == nil {
		return zero, work, d.NewError(d.ErrorCodeSubmissionNotAccepted, "review has no signed immutable delivery")
	}
	submission, err := workRepo.GetSubmission(ctx, r.Scope, r.SubmissionID)
	if err != nil {
		return zero, work, err
	}
	if submission.ProtocolVersion != 2 || submission.Digest != r.SubmissionDigest || submission.Material.SpecDigest != r.IssuedSpecDigest {
		return zero, work, d.NewError(d.ErrorCodeSubmissionNotAccepted, "review submission digest differs")
	}
	fact, err := repo.Fact(ctx, r.Scope, *source.IssuedSpecificationID)
	if err != nil {
		return zero, work, err
	}
	doc, err := verifyIssuedFact(ctx, u, fact, signing.ContractSpec, serverID, source.HolderPrincipalID)
	if err != nil {
		return zero, work, err
	}
	var payload signing.SpecPayload[d.SignedWorkSpec]
	if err = signing.DecodeStrict(doc.Payload, &payload, 128*1024); err != nil {
		return zero, work, signingError(err)
	}
	if fact.PayloadDigest != r.IssuedSpecDigest || payload.ContractID != source.ID.String() || payload.WorkItemID != r.WorkItemID.String() || payload.ContractKind != "execution" {
		return zero, work, d.NewError(d.ErrorCodeContractSpecMismatch, "review target specification differs")
	}
	acceptances, err := repo.Facts(ctx, r.Scope, source.ID, "acceptance", d.ID(""), 2)
	if err != nil {
		return zero, work, err
	}
	if len(acceptances) != 1 {
		return zero, work, d.NewError(d.ErrorCodeSubmissionNotAccepted, "delivery must have one immutable acceptance")
	}
	receiptDoc, err := verifyIssuedFact(ctx, u, acceptances[0], signing.AcceptanceReceipt, serverID, source.HolderPrincipalID)
	if err != nil {
		return zero, work, err
	}
	var receipt signing.ReceiptPayload
	if err = signing.DecodeStrict(receiptDoc.Payload, &receipt, 180*1024); err != nil {
		return zero, work, signingError(err)
	}
	if !receipt.Accepted || !receipt.LocalObligationClosed || receipt.Disposition != "delivered_for_review" || receipt.ContractID != source.ID.String() || receipt.ReviewCaseID != r.ID.String() || receipt.SubmissionID != submission.ID.String() || receipt.SubmissionDigest != submission.Digest || receipt.WorkItemID != r.WorkItemID.String() {
		return zero, work, d.NewError(d.ErrorCodeSubmissionNotAccepted, "signed delivery receipt differs from review target")
	}
	work, err = u.WorkItems().Get(ctx, r.Scope, r.WorkItemID)
	if err != nil {
		return zero, work, err
	}
	if work.PendingReviewCaseID == nil || *work.PendingReviewCaseID != r.ID || work.Lifecycle != d.WorkItemLifecycleInProgress || work.CurrentContractID == nil || *work.CurrentContractID != source.ID || !reflect.DeepEqual(payload.Spec.WorkSpec().Criteria, d.NormalizeContractSpec(d.WorkContractSpec{Criteria: work.Criteria.Items}).Criteria) {
		return zero, work, d.NewError(d.ErrorCodeContractSpecMismatch, "pending review obligations changed")
	}
	if err = validateSubmittedMaterial(ctx, u, work, submission.Material); err != nil {
		return zero, work, err
	}
	material := d.NewSignedResultMaterial(submission.Material)
	return SignedReviewSpec{ExecutionPrincipals: r.ExecutionPrincipals, ExecutionGroups: r.ExecutionGroups, ReviewCaseID: r.ID, SubmissionID: submission.ID, SubmissionDigest: submission.Digest, IssuedWorkSpecDigest: r.IssuedSpecDigest, Round: signing.Decimal(r.Round), WorkItemVersion: signing.Decimal(work.Version), IssuedWorkSpecification: &doc, AcceptanceReceipt: receiptDoc, Submission: &material, AcceptanceFloor: d.AcceptanceIndependentReview}, work, nil
}
func (s *Service) issueReviewAuthority(ctx context.Context, u ports.UnitOfWork, server d.ServerIdentity, c d.ReviewContract, p d.CredentialPolicy, keys []d.ID, now time.Time) (d.SignedFact, signing.Document, error) {
	allowed := []string{}
	for _, id := range keys {
		allowed = append(allowed, id.String())
	}
	payload := signing.AuthorityPayload{Binding: signing.Binding{ProtocolVersion: 2, ServerID: server.ID.String(), NamespaceID: c.Scope.NamespaceID.String(), OutcomeID: c.Scope.OutcomeID.String(), PrincipalID: c.HolderPrincipalID, SignerKeyID: server.IssuerKeyID.String()}, ContractKind: "review", ContractID: c.ID.String(), WorkItemID: c.WorkItemID.String(), ReviewCaseID: c.CaseID.String(), ExecutionID: c.ExecutionID.String(), FencingToken: signing.Decimal(c.FencingToken), ContractVersion: signing.Decimal(c.Version), LeaseVersion: signing.Decimal(c.LeaseVersion), SpecDigest: c.Binding.SpecificationDigest, IssuedAt: now.UTC().Format(time.RFC3339Nano), ExpiresAt: c.ExpiresAt.UTC().Format(time.RFC3339Nano), AcceptanceMode: string(d.AcceptanceIndependentReview), PolicyRevision: signing.Decimal(p.Version), AllowedSigningKeyIDs: allowed}
	return s.issueSignedFact(ctx, u, server, c.Scope, c.ID, "review", "authority", c.HolderPrincipalID, signing.ContractAuthority, payload, now)
}
func signedReviewEvents(s *Service, cc d.CommandContext, meta commandMetadata, r SignedReviewContractResult, revision d.OutcomeRevision) ([]d.DomainEvent, error) {
	typ := "work_review.acquired"
	if meta.Name == "RenewSignedReviewContract" {
		typ = "work_review.renewed"
	}
	if r.resumed {
		typ = "work_review.execution_resumed"
	}
	specs := []compoundEventSpec{}
	ref := d.EntityRef{Scope: r.Contract.Scope, Kind: d.EntityKindWorkItem, ID: r.Contract.WorkItemID}
	if r.expired != nil {
		specs = append(specs, compoundEventSpec{eventType: "work_review.expired", ref: ref, after: versionPtr(d.Version(r.WorkItemVersion))})
	}
	specs = append(specs, compoundEventSpec{eventType: typ, ref: ref, after: versionPtr(d.Version(r.WorkItemVersion))})
	events, err := buildCompoundEvents(s, cc, meta, revision, specs)
	if err != nil {
		return nil, err
	}
	for i := range events {
		c := r.Contract
		if events[i].EventType == "work_review.expired" && r.expired != nil {
			c = *r.expired
		}
		raw, e := json.Marshal(map[string]any{"review_case_id": r.Case.ID, "review_contract_id": c.ID, "submission_id": c.SubmissionID, "submission_digest": c.SubmissionDigest, "spec_digest": c.Binding.SpecificationDigest, "execution_id": c.ExecutionID, "fencing_token": c.FencingToken, "contract_version": signing.Decimal(c.Version), "lease_version": signing.Decimal(c.LeaseVersion), "expires_at": c.ExpiresAt})
		if e != nil {
			return nil, e
		}
		events[i].Payload = raw
		events[i].RecordedAt = r.EvaluatedAt
		if e = events[i].Validate(); e != nil {
			return nil, e
		}
	}
	return events, nil
}

// Large immutable material is referenced explicitly, never silently truncated.
// Fetched specification/accepted material must match these authenticated digests.
func compactSignedReviewSpec(payload signing.SpecPayload[SignedReviewSpec]) (SignedReviewSpec, error) {
	spec := payload.Spec
	size := func() (int, error) { payload.Spec = spec; raw, err := json.Marshal(payload); return len(raw), err }
	n, err := size()
	if err != nil {
		return spec, err
	}
	if n > 128*1024 && spec.Submission != nil {
		contract := spec.Submission.ContractID
		submission := spec.SubmissionID
		spec.Omitted = append(spec.Omitted, SignedMaterialReference{Kind: "submission", ContractID: contract, SubmissionID: &submission, Digest: spec.SubmissionDigest})
		spec.Submission = nil
		n, err = size()
		if err != nil {
			return spec, err
		}
	}
	if n > 128*1024 && spec.IssuedWorkSpecification != nil {
		var binding struct {
			ContractID d.ID `json:"contract_id"`
		}
		if err = json.Unmarshal(spec.IssuedWorkSpecification.Payload, &binding); err != nil {
			return spec, err
		}
		spec.Omitted = append(spec.Omitted, SignedMaterialReference{Kind: "execution_specification", ContractID: binding.ContractID, Digest: spec.IssuedWorkSpecDigest})
		spec.IssuedWorkSpecification = nil
	}
	return spec, nil
}
