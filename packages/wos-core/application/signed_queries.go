package application

import (
	"context"
	"encoding/base64"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"time"
)

type SignedStateQuery struct {
	Scope          d.Scope `json:"scope"`
	Resource       string  `json:"resource"`
	ID             d.ID    `json:"id,omitempty"`
	ContractKind   string  `json:"contract_kind,omitempty"`
	Digest         string  `json:"digest,omitempty"`
	IdempotencyKey string  `json:"idempotency_key,omitempty"`
	Limit          int     `json:"limit,omitempty"`
	Cursor         string  `json:"cursor,omitempty"`
}
type SignedContractMetadata struct {
	CredentialID          d.ID                `json:"credential_id"`
	ID                    d.ID                `json:"contract_id"`
	Kind                  string              `json:"contract_kind"`
	WorkItemID            d.ID                `json:"work_item_id"`
	HolderPrincipalID     string              `json:"holder_principal_id"`
	Status                d.ContractStatus    `json:"status"`
	EffectiveStatus       d.ContractStatus    `json:"effective_status"`
	Version               signing.Decimal     `json:"contract_version"`
	LeaseVersion          signing.Decimal     `json:"lease_version"`
	WorkItemVersion       signing.Decimal     `json:"work_item_version"`
	WorkItemLifecycle     d.WorkItemLifecycle `json:"work_item_lifecycle"`
	ExecutionID           d.ID                `json:"execution_id"`
	FencingToken          signing.Decimal     `json:"fencing_token"`
	ExpiresAt             time.Time           `json:"expires_at"`
	SpecDigest            string              `json:"spec_digest"`
	IssuedSpecificationID *d.ID               `json:"issued_specification_id,omitempty"`
	LatestAuthorityID     *d.ID               `json:"latest_authority_id,omitempty"`
	SubmissionID          *d.ID               `json:"submission_id,omitempty"`
	ReviewCaseID          *d.ID               `json:"review_case_id,omitempty"`
	Current               bool                `json:"current"`
	LeaseValid            bool                `json:"lease_valid"`
}

// MaterialPayload is the canonical accepted material encoded as base64. This
// avoids HTML-escape expansion; its digest is linked by the issuer acceptance.
// No query supplies execution authority or claims an evaluation of quality.
type SignedStateResult struct {
	OperationFingerprint string                   `json:"operation_fingerprint,omitempty"`
	OperationCommandID   *d.ID                    `json:"operation_command_id,omitempty"`
	OperationName        string                   `json:"operation_name,omitempty"`
	OperationPayload     string                   `json:"operation_result_payload,omitempty"`
	ProtocolVersion      int                      `json:"protocol_version"`
	NamespaceID          d.ID                     `json:"namespace_id"`
	Scope                *d.Scope                 `json:"scope,omitempty"`
	Resource             string                   `json:"resource"`
	OutcomeRevision      signing.Decimal          `json:"outcome_revision"`
	EvaluatedAt          time.Time                `json:"evaluated_at"`
	Server               *d.ServerIdentity        `json:"server,omitempty"`
	Protocol             *d.NamespaceWorkProtocol `json:"namespace_protocol,omitempty"`
	Contract             *SignedContractMetadata  `json:"contract,omitempty"`
	ReviewCase           *d.ReviewCase            `json:"review_case,omitempty"`
	Cases                []d.ReviewCase           `json:"cases,omitempty"`
	NextCursor           string                   `json:"next_cursor,omitempty"`
	SearchComplete       bool                     `json:"search_complete,omitempty"`
	Scanned              int                      `json:"scanned,omitempty"`
	FactID               *d.ID                    `json:"fact_id,omitempty"`
	PayloadDigest        string                   `json:"payload_digest,omitempty"`
	Envelope             *signing.Envelope        `json:"envelope,omitempty"`
	SignerKey            *d.SigningKey            `json:"signer_key,omitempty"`
	SubmissionID         *d.ID                    `json:"submission_id,omitempty"`
	SubmissionDigest     string                   `json:"submission_digest,omitempty"`
	MaterialPayload      string                   `json:"material_payload,omitempty"`
	AcceptanceFactID     *d.ID                    `json:"acceptance_fact_id,omitempty"`
}

func (r SignedStateResult) MarshalJSON() ([]byte, error) {
	type wire SignedStateResult
	return signedWireJSON(wire(r))
}

func (s *Service) ReadSignedState(ctx context.Context, q SignedStateQuery) (SignedStateResult, error) {
	var zero SignedStateResult
	if q.Scope.NamespaceID.Validate() != nil {
		return zero, d.NewError(d.ErrorCodeInvalidArgument, "Namespace required")
	}
	if q.Resource != "trust" && q.Scope.Validate() != nil {
		return zero, d.NewError(d.ErrorCodeInvalidArgument, "exact Outcome scope required")
	}
	if q.Digest != "" && !d.ValidSignedDigest(q.Digest) {
		return zero, d.NewError(d.ErrorCodeInvalidArgument, "invalid expected digest")
	}
	identity, ok := IdentityFromContext(ctx)
	if !ok || identity.CredentialDigest == "" || identity.NamespaceID != q.Scope.NamespaceID {
		return zero, d.NewError(d.ErrorCodeForbidden, "authenticated scoped credential required")
	}
	if err := s.authorizeScopedRead(ctx, q.Scope); err != nil {
		return zero, err
	}
	u, err := s.tx.Begin(ctx)
	if err != nil {
		return zero, err
	}
	defer u.Rollback()
	now, err := securityTransactionTime(ctx, u, s.clock)
	if err != nil {
		return zero, err
	}
	snapshot, ok := u.(ports.AccessSnapshotUnitOfWork)
	if !ok {
		return zero, d.NewError(d.ErrorCodeInvalidConfig, "signed query access snapshot unavailable")
	}
	if err = snapshot.AuthorizeAccessSnapshot(ctx, ports.AccessSnapshotRequest{Authorization: ports.AuthorizationRequest{NamespaceID: q.Scope.NamespaceID, OutcomeID: q.Scope.OutcomeID, PrincipalID: identity.PrincipalID, Permission: ports.PermissionStateRead}, CredentialDigest: identity.CredentialDigest, Actor: identity.Actor, Now: now}); err != nil {
		return zero, err
	}
	result := SignedStateResult{ProtocolVersion: 2, NamespaceID: q.Scope.NamespaceID, Resource: q.Resource, EvaluatedAt: now}
	if q.Resource == "trust" {
		serverRepo, ok := u.(ports.ServerIdentityUnitOfWork)
		if !ok {
			return zero, d.NewError(d.ErrorCodeInvalidConfig, "persistent server trust unavailable")
		}
		server, err := serverRepo.ServerIdentity().Server(ctx)
		if err != nil {
			return zero, err
		}
		protocolRepo, err := protocolRepository(u)
		if err != nil {
			return zero, err
		}
		protocol, err := protocolRepo.Lock(ctx, q.Scope.NamespaceID, false)
		if err != nil {
			return zero, err
		}
		result.Server, result.Protocol = &server, &protocol
		return result, nil
	}
	result.Scope = &q.Scope
	coordination, err := u.Coordination().LockOutcome(ctx, q.Scope)
	if err != nil {
		return zero, err
	}
	result.OutcomeRevision = signing.Decimal(coordination.Revision)
	repo, workRepo, err := signedRepository(u)
	if err != nil {
		return zero, err
	}
	if q.Resource != "review_queue" && q.Resource != "cases" && q.Resource != "receipt" && q.Resource != "operation" && q.ID.Validate() != nil {
		return zero, d.NewError(d.ErrorCodeInvalidArgument, "resource identifier required")
	}
	switch q.Resource {
	case "operation":
		if err = d.ValidateIdempotencyKey(q.IdempotencyKey); err != nil {
			return zero, err
		}
		durable, ok := u.(ports.SignedOperationUnitOfWork)
		if !ok {
			return zero, d.NewError(d.ErrorCodeInvalidConfig, "durable signed operation repository required")
		}
		accepted, err := durable.SignedOperations().Get(ctx, q.Scope.NamespaceID, identity.PrincipalID, q.IdempotencyKey)
		if code, _ := d.ErrorCodeOf(err); code == d.ErrorCodeNotFound {
			legacy, e := durable.SignedOperations().FindLegacy(ctx, q.Scope.NamespaceID, identity.PrincipalID, q.IdempotencyKey)
			if e != nil {
				return zero, e
			}
			accepted, err = legacySignedOperationView(ctx, u, q.Scope, identity.PrincipalID, q.IdempotencyKey, legacy)
		}
		if err != nil {
			return zero, err
		}
		registry, err := signingRepository(u)
		if err != nil {
			return zero, err
		}
		credential, err := registry.CredentialByDigest(ctx, identity.CredentialDigest)
		if err != nil {
			return zero, err
		}
		if accepted.Scope != q.Scope || accepted.CredentialID != credential.ID {
			return zero, d.NewError(d.ErrorCodeNotFound, "signed operation unavailable in selected credential/scope")
		}
		result.PayloadDigest = signing.Digest(accepted.Result.ResponseJSON)
		if q.Digest != "" && q.Digest != result.PayloadDigest {
			return zero, d.NewError(d.ErrorCodeContractSpecMismatch, "durable response digest differs")
		}
		result.OperationCommandID = &accepted.Result.CommandID
		result.OperationName = accepted.CommandName
		result.OperationFingerprint = accepted.Fingerprint
		result.OperationPayload = base64.StdEncoding.EncodeToString(accepted.Result.ResponseJSON)
	case "execution", "review", "specification", "authority":
		kind := q.ContractKind
		if q.Resource == "execution" || q.Resource == "review" {
			kind = q.Resource
		}
		state, err := signedContractState(ctx, u, q.Scope, kind, q.ID, now)
		if err != nil {
			return zero, err
		}
		if q.Resource == "execution" || q.Resource == "review" {
			result.Contract = &state
			break
		}
		id := state.IssuedSpecificationID
		if q.Resource == "authority" {
			id = state.LatestAuthorityID
		}
		if id == nil {
			return zero, d.NewError(d.ErrorCodeNotFound, "signed issuance pointer unavailable")
		}
		fact, err := repo.Fact(ctx, q.Scope, *id)
		if err != nil {
			return zero, err
		}
		if fact.ContractID != q.ID || fact.ContractKind != kind || fact.Kind != q.Resource {
			return zero, d.NewError(d.ErrorCodeInvalidScope, "issuance pointer binding differs")
		}
		if err = fillSignedFactView(ctx, u, q, fact, &result); err != nil {
			return zero, err
		}
	case "case", "correction":
		review, err := repo.Case(ctx, q.Scope, q.ID)
		if err != nil {
			return zero, err
		}
		result.ReviewCase = &review
		if q.Resource == "correction" {
			if review.Status != d.ReviewChangesRequested || review.LatestDecisionID == nil {
				return zero, d.NewError(d.ErrorCodePreconditionFailed, "case has no accepted correction decision")
			}
			fact, err := repo.Fact(ctx, q.Scope, *review.LatestDecisionID)
			if err != nil {
				return zero, err
			}
			if fact.ContractKind != "review" || fact.Kind != "return" {
				return zero, d.NewError(d.ErrorCodeInvalidScope, "correction decision pointer differs")
			}
			if err = fillSignedFactView(ctx, u, q, fact, &result); err != nil {
				return zero, err
			}
			serverRepository, ok := u.(ports.ServerIdentityUnitOfWork)
			if !ok {
				return zero, d.NewError(d.ErrorCodeInvalidConfig, "persistent server trust unavailable")
			}
			server, err := serverRepository.ServerIdentity().Server(ctx)
			if err != nil {
				return zero, err
			}
			decision, err := verifiedReviewDecision(ctx, u, fact, server.ID.String())
			if err != nil {
				return zero, err
			}
			if decision.Decision != "changes_requested" || decision.SubmissionID != review.SubmissionID.String() || decision.SubmissionDigest != review.SubmissionDigest {
				return zero, d.NewError(d.ErrorCodeSubmissionNotAccepted, "accepted correction target differs")
			}
			acceptance, err := repo.Acceptance(ctx, q.Scope.NamespaceID, fact.PrincipalID, fact.IdempotencyKey)
			if err != nil {
				return zero, err
			}
			result.AcceptanceFactID = &acceptance.ID
		}
	case "fact":
		fact, err := repo.Fact(ctx, q.Scope, q.ID)
		if err != nil {
			return zero, err
		}
		if err = fillSignedFactView(ctx, u, q, fact, &result); err != nil {
			return zero, err
		}
	case "receipt":
		if err = d.ValidateIdempotencyKey(q.IdempotencyKey); err != nil {
			return zero, err
		}
		fact, err := repo.Acceptance(ctx, q.Scope.NamespaceID, identity.PrincipalID, q.IdempotencyKey)
		if err != nil {
			return zero, err
		}
		if fact.Scope != q.Scope {
			return zero, d.NewError(d.ErrorCodeNotFound, "receipt unavailable in requested scope")
		}
		if err = fillSignedFactView(ctx, u, q, fact, &result); err != nil {
			return zero, err
		}
	case "submission":
		submission, err := workRepo.GetSubmission(ctx, q.Scope, q.ID)
		if err != nil {
			return zero, err
		}
		if submission.ProtocolVersion != 2 || q.Digest != "" && q.Digest != submission.Digest {
			return zero, d.NewError(d.ErrorCodeContractSpecMismatch, "accepted material protocol/digest differs")
		}
		contract, err := workRepo.Get(ctx, q.Scope, submission.Material.ContractID)
		if err != nil {
			return zero, err
		}
		if contract.LatestSubmissionID == nil || *contract.LatestSubmissionID != submission.ID || contract.SignedBinding == nil {
			return zero, d.NewError(d.ErrorCodeSubmissionNotAccepted, "submission is not an accepted signed target")
		}
		acceptances, err := repo.Facts(ctx, q.Scope, contract.ID, "acceptance", "", 2)
		if err != nil {
			return zero, err
		}
		if len(acceptances) != 1 {
			return zero, d.NewError(d.ErrorCodeSubmissionNotAccepted, "unique issuer acceptance required")
		}
		verified := result
		if err = fillSignedFactView(ctx, u, SignedStateQuery{Scope: q.Scope}, acceptances[0], &verified); err != nil {
			return zero, err
		}
		var receipt signing.ReceiptPayload
		public, err := verified.SignerKey.PublicBytes()
		if err != nil {
			return zero, err
		}
		raw, err := signing.Verify(*verified.Envelope, signing.AcceptanceReceipt, verified.SignerKey.ID.String(), public)
		if err != nil {
			return zero, signingError(err)
		}
		if err = signing.DecodeStrict(raw, &receipt, signing.MaxPayloadBytes); err != nil {
			return zero, signingError(err)
		}
		if !receipt.Accepted || !receipt.LocalObligationClosed || receipt.ContractID != contract.ID.String() || receipt.WorkItemID != submission.Material.WorkItemID.String() || receipt.PrincipalID != submission.PrincipalID || receipt.SubmissionID != submission.ID.String() || receipt.SubmissionDigest != submission.Digest {
			return zero, d.NewError(d.ErrorCodeSubmissionNotAccepted, "issuer acceptance does not bind exact material")
		}
		material, err := signing.Canonical(d.NewSignedResultMaterial(submission.Material))
		if err != nil {
			return zero, signingError(err)
		}
		if signing.Digest(material) != submission.Digest {
			return zero, d.NewError(d.ErrorCodeContractSpecMismatch, "accepted material digest differs")
		}
		result.SubmissionID, result.SubmissionDigest, result.MaterialPayload, result.AcceptanceFactID = &submission.ID, submission.Digest, base64.StdEncoding.EncodeToString(material), &acceptances[0].ID
	case "cases", "review_queue":
		limit, err := queryLimit(q.Limit)
		if err != nil {
			return zero, err
		}
		filter := ports.ReviewFilter{Limit: limit + 1, OpenOnly: q.Resource == "review_queue"}
		if q.Cursor != "" {
			cursor, err := decodeCursor(q.Cursor, q.Scope.NamespaceID, q.Scope.OutcomeID, q.Resource, "")
			if err != nil {
				return zero, err
			}
			filter.After, err = d.ParseID(cursor.Key)
			if err != nil {
				return zero, err
			}
		}
		cases, err := repo.Cases(ctx, q.Scope, filter)
		if err != nil {
			return zero, err
		}
		result.SearchComplete = len(cases) <= limit
		if len(cases) > limit {
			cases = cases[:limit]
			last := cases[len(cases)-1].ID
			result.NextCursor = encodeCursor(queryCursor{Namespace: q.Scope.NamespaceID, Outcome: q.Scope.OutcomeID, Section: q.Resource, Key: last.String()})
		}
		result.Scanned = len(cases)
		result.Cases = []d.ReviewCase{}
		for _, review := range cases {
			if q.Resource == "review_queue" && review.CurrentContractID != nil {
				active, err := repo.ReviewContract(ctx, q.Scope, *review.CurrentContractID)
				if err != nil {
					return zero, err
				}
				if active.ValidAt(now) {
					continue
				}
			}
			result.Cases = append(result.Cases, review)
		}
	default:
		return zero, d.NewError(d.ErrorCodeInvalidArgument, "unknown signed resource")
	}
	raw, err := json.Marshal(result)
	if err != nil {
		return zero, err
	}
	maximum := MaxSnapshotBytes
	if q.Resource == "operation" {
		maximum = base64.StdEncoding.EncodedLen(d.MaxSignedOperationResultBytes) + (64 << 10)
	}
	if len(raw) > maximum {
		return zero, d.NewError(d.ErrorCodeGraphLimitExceeded, "signed query byte limit exceeded; use a smaller page")
	}
	return result, nil
}

func signedContractState(ctx context.Context, u ports.UnitOfWork, scope d.Scope, kind string, id d.ID, now time.Time) (SignedContractMetadata, error) {
	var state SignedContractMetadata
	repo, workRepo, err := signedRepository(u)
	if err != nil {
		return state, err
	}
	var workID d.ID
	switch kind {
	case "execution":
		c, err := workRepo.Get(ctx, scope, id)
		if err != nil {
			return state, err
		}
		if c.SignedBinding == nil {
			return state, d.NewError(d.ErrorCodeSignedProtocolRequired, "signed execution required")
		}
		workID = c.WorkItemID
		state = SignedContractMetadata{CredentialID: c.SignedBinding.CredentialID, ID: c.ID, Kind: kind, WorkItemID: c.WorkItemID, HolderPrincipalID: c.HolderPrincipalID, Status: c.Status, EffectiveStatus: c.EffectiveStatus(now), Version: signing.Decimal(c.Version), LeaseVersion: signing.Decimal(c.LeaseVersion), ExecutionID: c.ExecutionID, FencingToken: signing.Decimal(c.FencingToken), ExpiresAt: c.ExpiresAt, SpecDigest: c.SignedBinding.SpecificationDigest, IssuedSpecificationID: c.IssuedSpecificationID, LatestAuthorityID: c.LatestAuthorityID, SubmissionID: c.LatestSubmissionID, LeaseValid: c.ValidAt(now)}
	case "review":
		c, err := repo.ReviewContract(ctx, scope, id)
		if err != nil {
			return state, err
		}
		workID = c.WorkItemID
		state = SignedContractMetadata{CredentialID: c.Binding.CredentialID, ID: c.ID, Kind: kind, WorkItemID: c.WorkItemID, HolderPrincipalID: c.HolderPrincipalID, Status: c.Status, EffectiveStatus: c.EffectiveStatus(now), Version: signing.Decimal(c.Version), LeaseVersion: signing.Decimal(c.LeaseVersion), ExecutionID: c.ExecutionID, FencingToken: signing.Decimal(c.FencingToken), ExpiresAt: c.ExpiresAt, SpecDigest: c.Binding.SpecificationDigest, IssuedSpecificationID: c.IssuedSpecificationID, LatestAuthorityID: c.LatestAuthorityID, SubmissionID: &c.SubmissionID, ReviewCaseID: &c.CaseID, LeaseValid: c.ValidAt(now)}
		review, err := repo.Case(ctx, scope, c.CaseID)
		if err != nil {
			return state, err
		}
		state.Current = review.CurrentContractID != nil && *review.CurrentContractID == c.ID && review.Status == d.ReviewInReview
	default:
		return state, d.NewError(d.ErrorCodeInvalidArgument, "contract kind must be execution or review")
	}
	work, err := u.WorkItems().Get(ctx, scope, workID)
	if err != nil {
		return state, err
	}
	state.WorkItemVersion, state.WorkItemLifecycle = signing.Decimal(work.Version), work.Lifecycle
	if kind == "execution" {
		state.Current = work.CurrentContractID != nil && *work.CurrentContractID == id
		state.ReviewCaseID = work.PendingReviewCaseID
	} else {
		state.Current = state.Current && work.PendingReviewCaseID != nil && *work.PendingReviewCaseID == *state.ReviewCaseID
	}
	return state, nil
}

func fillSignedFactView(ctx context.Context, u ports.UnitOfWork, q SignedStateQuery, f d.SignedFact, result *SignedStateResult) error {
	if f.Scope != q.Scope || q.Digest != "" && f.PayloadDigest != q.Digest {
		return d.NewError(d.ErrorCodeContractSpecMismatch, "signed resource scope/digest differs")
	}
	registry, err := signingRepository(u)
	if err != nil {
		return err
	}
	key, err := registry.Key(ctx, q.Scope.NamespaceID, f.KeyID)
	if err != nil {
		return err
	}
	serverRepo, ok := u.(ports.ServerIdentityUnitOfWork)
	if !ok {
		return d.NewError(d.ErrorCodeInvalidConfig, "server trust unavailable")
	}
	server, err := serverRepo.ServerIdentity().Server(ctx)
	if err != nil {
		return err
	}
	purpose := signing.ContractSpec
	var schema any
	switch f.Kind {
	case "specification":
		if f.ContractKind == "execution" {
			schema = new(signing.SpecPayload[d.SignedWorkSpec])
		} else {
			schema = new(signing.SpecPayload[SignedReviewSpec])
		}
	case "authority":
		purpose, schema = signing.ContractAuthority, new(signing.AuthorityPayload)
	case "acceptance":
		purpose, schema = signing.AcceptanceReceipt, new(signing.ReceiptPayload)
	case "return":
		if f.ContractKind == "execution" {
			purpose, schema = signing.WorkReturn, new(signing.WorkReturnPayload[SignedReturnMaterial])
		} else {
			purpose, schema = signing.ReviewReturn, new(signing.ReviewReturnPayload[SignedReviewMaterial])
		}
	default:
		return d.NewError(d.ErrorCodeInvalidArgument, "unknown signed fact kind")
	}
	if f.Kind == "return" {
		if key.Purpose != "agent" || key.PrincipalID != f.PrincipalID {
			return d.NewError(d.ErrorCodeForbidden, "return signer role differs")
		}
	} else if key.Purpose != "issuer" || key.PrincipalID != server.PrincipalID() {
		return d.NewError(d.ErrorCodeForbidden, "issuer signer role differs")
	}
	public, err := key.PublicBytes()
	if err != nil {
		return err
	}
	document, err := factDocument(f)
	if err != nil {
		return err
	}
	envelope, err := document.Envelope()
	if err != nil {
		return signingError(err)
	}
	raw, err := signing.Verify(envelope, purpose, key.ID.String(), public)
	if err != nil {
		return signingError(err)
	}
	if signing.Digest(raw) != f.PayloadDigest {
		return d.NewError(d.ErrorCodeContractSpecMismatch, "stored authenticated bytes differ")
	}
	if err = signing.DecodeStrict(raw, schema, signing.MaxPayloadBytes); err != nil {
		return signingError(err)
	}
	var target struct {
		signing.Binding
		ContractID     string `json:"contract_id"`
		ReviewCaseID   string `json:"review_case_id"`
		RequestID      string `json:"request_id"`
		RequestDigest  string `json:"request_digest"`
		IdempotencyKey string `json:"idempotency_key"`
	}
	if err = json.Unmarshal(raw, &target); err != nil {
		return err
	}
	if target.ServerID != server.ID.String() || target.NamespaceID != q.Scope.NamespaceID.String() || target.OutcomeID != q.Scope.OutcomeID.String() || target.PrincipalID != f.PrincipalID || target.ContractID != f.ContractID.String() {
		return d.NewError(d.ErrorCodeInvalidScope, "authenticated resource bindings differ")
	}
	if f.RequestID != nil && (target.RequestID != f.RequestID.String() || target.IdempotencyKey != f.IdempotencyKey) {
		return d.NewError(d.ErrorCodeContractSpecMismatch, "authenticated request differs from receipt record")
	}
	if f.Kind == "return" && f.RequestDigest != f.PayloadDigest || f.Kind == "acceptance" && target.RequestDigest != f.RequestDigest {
		return d.NewError(d.ErrorCodeContractSpecMismatch, "authenticated request digest differs")
	}
	if q.Resource == "correction" && target.ReviewCaseID != q.ID.String() {
		return d.NewError(d.ErrorCodeInvalidScope, "correction targets another case")
	}
	result.FactID, result.PayloadDigest, result.Envelope, result.SignerKey = &f.ID, f.PayloadDigest, &envelope, &key
	return nil
}
