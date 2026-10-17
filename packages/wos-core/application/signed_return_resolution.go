package application

import (
	"context"
	"encoding/json"
	"slices"
	"strings"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

type ResolveSignedReturnIntentCommand struct {
	Scope            d.Scope
	OriginalEnvelope signing.Envelope
	Envelope         signing.Envelope
}
type SignedReturnResolutionResult struct {
	Decision          string            `json:"decision"`
	AcceptanceReceipt *signing.Document `json:"acceptance_receipt,omitempty"`
	ResolutionReceipt *signing.Document `json:"resolution_receipt,omitempty"`
}

func resolutionRepository(u ports.UnitOfWork) (ports.SignedReturnResolutionRepository, error) {
	r, ok := u.(ports.SignedReturnResolutionUnitOfWork)
	if !ok {
		return nil, d.NewError(d.ErrorCodeInvalidConfig, "durable terminal return resolution repository required")
	}
	return r.SignedReturnResolutions(), nil
}
func originalReturnBinding(env signing.Envelope) (b signing.RequestBinding, command, kind string, err error) {
	switch env.PayloadType {
	case signing.WorkReturn:
		var p signing.WorkReturnPayload[SignedReturnMaterial]
		err = signedRequestHint(env, &p)
		b = p.RequestBinding
		command, kind = "ReturnSignedWork", "execution"
	case signing.ReviewReturn:
		var p signing.ReviewReturnPayload[SignedReviewMaterial]
		err = signedRequestHint(env, &p)
		b = p.RequestBinding
		command, kind = "ReturnSignedReview", "review"
	default:
		err = d.NewError(d.ErrorCodeInvalidArgument, "original envelope must be an execution or review return")
	}
	if err == nil {
		_, err = signedRequestScope(b)
	}
	if err == nil && (kind == "execution" && b.OperationKind != "work_return" || kind == "review" && b.OperationKind != "review_return") {
		err = d.NewError(d.ErrorCodeInvalidArgument, "original return purpose differs")
	}
	return
}

// Called after signature verification, under the shared signed security barrier,
// before any acceptance effect. Serializable isolation and Outcome coordination
// keep acceptance exclusive with closure. Closed intentions survive cache expiry.
func (s *Service) rejectClosedReturn(ctx context.Context, u ports.UnitOfWork, b signing.RequestBinding, command, kind string, digest string) error {
	scope, _ := signedRequestScope(b)
	if e := s.requireSignedAcceptedCredential(ctx, u, scope, kind, d.ID(b.ContractID)); e != nil {
		return e
	}
	identity, _, e := s.signedAccess(ctx, u, scope, commandPermission(command), "")
	if e != nil {
		return e
	}
	r, e := resolutionRepository(u)
	if e != nil {
		return e
	}
	closed, e := r.GetByOriginalIntent(ctx, scope.TenantID, b.PrincipalID, identity.CredentialID, command, b.IdempotencyKey)
	if code, _ := d.ErrorCodeOf(e); code == d.ErrorCodeNotFound {
		return nil
	}
	if e != nil {
		return e
	}
	if closed.Scope != scope || closed.ContractID != d.ID(b.ContractID) || closed.OriginalRequestID != d.ID(b.RequestID) || closed.OriginalRequestDigest != digest {
		return d.NewError(d.ErrorCodeIdempotencyConflict, "terminal original intention differs")
	}
	return d.NewError(d.ErrorCodeSignedIntentClosed, "the original signed return is durably closed without acceptance")
}

func (s *Service) ResolveSignedReturnIntent(ctx context.Context, cc d.CommandContext, cmd ResolveSignedReturnIntentCommand) (MutationResult[SignedReturnResolutionResult], error) {
	var hint signing.ReturnResolutionPayload
	if e := signedRequestHint(cmd.Envelope, &hint); e != nil {
		return MutationResult[SignedReturnResolutionResult]{}, e
	}
	if cmd.Scope.Validate() != nil || hint.ProtocolVersion != 3 || hint.OperationKind != "resolve_return_intent" || hint.TenantID != cmd.Scope.TenantID.String() || hint.OutcomeID != cmd.Scope.OutcomeID.String() || hint.PrincipalID != cc.PrincipalID || hint.IdempotencyKey != cc.IdempotencyKey || d.ID(hint.RequestID).Validate() != nil || d.ID(hint.SignerKeyID).Validate() != nil || d.ValidateIdempotencyKey(hint.IdempotencyKey) != nil || strings.TrimSpace(hint.Reason) == "" || len(hint.Reason) > 4096 {
		return MutationResult[SignedReturnResolutionResult]{}, d.NewError(d.ErrorCodeInvalidArgument, "invalid signed resolution identity or reason")
	}
	cc.CommandID = d.ID(hint.RequestID)
	if e := cc.Validate(); e != nil {
		return MutationResult[SignedReturnResolutionResult]{}, e
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (SignedReturnResolutionResult, d.OutcomeRevision, error) {
		var zero SignedReturnResolutionResult
		b, command, kind, e := originalReturnBinding(cmd.OriginalEnvelope)
		if e != nil {
			return zero, 0, e
		}
		if b.TenantID != hint.TenantID || b.OutcomeID != hint.OutcomeID || b.ServerID != hint.ServerID || b.PrincipalID != hint.PrincipalID {
			return zero, 0, d.NewError(d.ErrorCodeForbidden, "original destination identity differs")
		}
		permission := commandPermission(command)
		identity, policy, e := s.signedAccess(ctx, u, cmd.Scope, permission, d.ID(hint.SignerKeyID))
		if e != nil {
			return zero, 0, e
		}
		if e = s.requireSignedAcceptedCredential(ctx, u, cmd.Scope, kind, d.ID(b.ContractID)); e != nil {
			return zero, 0, e
		}
		// Both signatures must still belong to the original live credential's
		// permitted signer. Retired/replaced keys require explicit recovery.
		if _, _, e = s.signedAccess(ctx, u, cmd.Scope, permission, d.ID(b.SignerKeyID)); e != nil {
			return zero, 0, e
		}
		registry, e := signingRepository(u)
		if e != nil {
			return zero, 0, e
		}
		key, e := registry.Key(ctx, cmd.Scope.TenantID, d.ID(hint.SignerKeyID))
		if e != nil {
			return zero, 0, e
		}
		pub, e := key.PublicBytes()
		if e != nil {
			return zero, 0, e
		}
		raw, e := signing.Verify(cmd.Envelope, signing.ReturnResolution, key.ID.String(), pub)
		if e != nil {
			return zero, 0, signingError(e)
		}
		var request signing.ReturnResolutionPayload
		if e = signing.DecodeStrict(raw, &request, 32<<10); e != nil {
			return zero, 0, signingError(e)
		}
		originalKey, e := registry.Key(ctx, cmd.Scope.TenantID, d.ID(b.SignerKeyID))
		if e != nil {
			return zero, 0, e
		}
		originalPub, e := originalKey.PublicBytes()
		if e != nil {
			return zero, 0, e
		}
		originalRaw, e := signing.Verify(cmd.OriginalEnvelope, cmd.OriginalEnvelope.PayloadType, originalKey.ID.String(), originalPub)
		if e != nil {
			return zero, 0, signingError(e)
		}
		envelopeBytes, e := signing.Canonical(cmd.OriginalEnvelope)
		if e != nil {
			return zero, 0, signingError(e)
		}
		if b.TenantID != request.TenantID || b.OutcomeID != request.OutcomeID || b.ServerID != request.ServerID || b.PrincipalID != request.PrincipalID || request.CredentialID != identity.CredentialID.String() || request.OriginalCommandName != command || request.ContractID != b.ContractID || request.OriginalRequestID != b.RequestID || request.OriginalIdempotencyKey != b.IdempotencyKey || request.OriginalRequestDigest != signing.Digest(originalRaw) || request.OriginalEnvelopeDigest != signing.Digest(envelopeBytes) || request.IdempotencyKey == b.IdempotencyKey || request.RequestID == b.RequestID {
			return zero, 0, d.NewError(d.ErrorCodeForbidden, "resolution does not bind the exact original credential, envelope and scope")
		}
		serverRepo, e := serverIdentityRepository(u)
		if e != nil {
			return zero, 0, e
		}
		server, e := serverRepo.Server(ctx)
		if e != nil {
			return zero, 0, e
		}
		if request.ServerID != server.ID.String() {
			return zero, 0, d.NewError(d.ErrorCodeForbidden, "resolution destination issuer differs")
		}
		coord, e := u.Coordination().LockOutcome(ctx, cmd.Scope)
		if e != nil {
			return zero, 0, e
		}
		repo, workRepo, e := signedRepository(u)
		if e != nil {
			return zero, 0, e
		}
		if prior, err := repo.Acceptance(ctx, cmd.Scope.TenantID, cc.PrincipalID, b.IdempotencyKey); err == nil {
			if prior.Scope != cmd.Scope || prior.ContractKind != kind || prior.ContractID != d.ID(b.ContractID) || prior.RequestID == nil || prior.RequestID.String() != b.RequestID || prior.RequestDigest != request.OriginalRequestDigest {
				return zero, 0, d.NewError(d.ErrorCodeIdempotencyConflict, "original accepted intention differs")
			}
			doc, err := verifyIssuedFact(ctx, u, prior, signing.AcceptanceReceipt, b.ServerID, cc.PrincipalID)
			if err != nil {
				return zero, 0, err
			}
			return SignedReturnResolutionResult{Decision: "accepted", AcceptanceReceipt: &doc}, coord.Revision, nil
		} else if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodeNotFound {
			return zero, 0, err
		}
		resolutions, e := resolutionRepository(u)
		if e != nil {
			return zero, 0, e
		}
		if prior, err := resolutions.GetByOriginalIntent(ctx, cmd.Scope.TenantID, cc.PrincipalID, identity.CredentialID, command, b.IdempotencyKey); err == nil {
			if prior.ResolutionRequestID != cc.CommandID || prior.ResolutionIdempotencyKey != cc.IdempotencyKey || prior.ResolutionDigest != signing.Digest(raw) {
				return zero, 0, d.NewError(d.ErrorCodeIdempotencyConflict, "another terminal resolution already exists")
			}
			doc, err := verifyResolutionReceipt(ctx, u, prior, server.ID.String())
			if err != nil {
				return zero, 0, err
			}
			return SignedReturnResolutionResult{Decision: prior.Decision, ResolutionReceipt: &doc}, prior.OutcomeRevision, nil
		} else if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodeNotFound {
			return zero, 0, err
		}
		if request.PolicyRevision != signing.Decimal(policy.Version) {
			return zero, 0, d.NewError(d.ErrorCodeVersionConflict, "live resolution policy changed")
		}
		now, e := securityTransactionTime(ctx, u, s.clock)
		if e != nil {
			return zero, 0, e
		}
		if e = requireActiveOutcome(ctx, u, cmd.Scope); e != nil {
			return zero, 0, e
		}
		work, e := u.WorkItems().Get(ctx, cmd.Scope, d.ID(b.WorkItemID))
		if e != nil {
			return zero, 0, e
		}
		if work.Version != d.Version(b.ExpectedWorkItemVersion) {
			return zero, 0, d.NewError(d.ErrorCodeVersionConflict, "original Task CAS changed; authorized recovery required")
		}
		authority := ContractAuthority{ExecutionID: d.ID(b.ExecutionID), FencingToken: d.FencingToken(b.FencingToken), SpecDigest: b.SpecDigest}
		var grant signing.AuthorityPayload
		if kind == "execution" {
			c, err := workRepo.Get(ctx, cmd.Scope, d.ID(b.ContractID))
			if err != nil {
				return zero, 0, err
			}
			if c.HolderPrincipalID != cc.PrincipalID || c.WorkItemID != work.ID || c.SignedBinding == nil || c.SignedBinding.CredentialID != identity.CredentialID {
				return zero, 0, d.NewError(d.ErrorCodeForbidden, "original execution owner differs")
			}
			if e = requireIssuedWorkAuthority(ctx, u, c, authority, b.AuthorityDigest); e != nil {
				return zero, 0, e
			}
			if c.Version != d.Version(b.ExpectedContractVersion) || c.LeaseVersion != d.Version(b.ExpectedLeaseVersion) || work.CurrentContractID == nil || *work.CurrentContractID != c.ID {
				return zero, 0, d.NewError(d.ErrorCodeVersionConflict, "original execution CAS changed")
			}
			if e = c.Authorize(cc.PrincipalID, authority.ExecutionID, authority.FencingToken, now); e != nil {
				return zero, 0, e
			}
			fact, err := repo.Fact(ctx, cmd.Scope, *c.LatestAuthorityID)
			if err != nil {
				return zero, 0, err
			}
			if e = signing.DecodeStrict(fact.Payload, &grant, 128<<10); e != nil {
				return zero, 0, signingError(e)
			}
		} else {
			c, err := repo.ReviewContract(ctx, cmd.Scope, d.ID(b.ContractID))
			if err != nil {
				return zero, 0, err
			}
			if c.HolderPrincipalID != cc.PrincipalID || c.WorkItemID != work.ID || c.Binding.CredentialID != identity.CredentialID {
				return zero, 0, d.NewError(d.ErrorCodeForbidden, "original review owner differs")
			}
			grant, e = requireIssuedReviewAuthority(ctx, u, c, authority, b.AuthorityDigest)
			if e != nil {
				return zero, 0, e
			}
			if c.Version != d.Version(b.ExpectedContractVersion) || c.LeaseVersion != d.Version(b.ExpectedLeaseVersion) {
				return zero, 0, d.NewError(d.ErrorCodeVersionConflict, "original review CAS changed")
			}
			if e = c.Authorize(cc.PrincipalID, authority.ExecutionID, authority.FencingToken, now); e != nil {
				return zero, 0, e
			}
			var original signing.ReviewReturnPayload[SignedReviewMaterial]
			if e = signing.DecodeStrict(originalRaw, &original, 180<<10); e != nil {
				return zero, 0, signingError(e)
			}
			if original.ReviewCaseID != c.CaseID.String() || original.SubmissionID != c.SubmissionID.String() || original.SubmissionDigest != c.SubmissionDigest {
				return zero, 0, d.NewError(d.ErrorCodeForbidden, "original review target differs")
			}
			caseState, err := repo.Case(ctx, cmd.Scope, c.CaseID)
			if err != nil {
				return zero, 0, err
			}
			if caseState.Version != d.Version(original.ExpectedReviewCaseVersion) || caseState.CurrentContractID == nil || *caseState.CurrentContractID != c.ID {
				return zero, 0, d.NewError(d.ErrorCodeVersionConflict, "original review case CAS changed")
			}
		}
		if !slices.Contains(grant.AllowedSigningKeyIDs, b.SignerKeyID) || !slices.Contains(grant.AllowedSigningKeyIDs, request.SignerKeyID) {
			return zero, 0, d.NewError(d.ErrorCodeForbidden, "resolution signer outside original grant")
		}
		server, e = s.requireSignedIssuer(ctx, u, cmd.Scope.TenantID)
		if e != nil {
			return zero, 0, e
		}
		payload := signing.ReturnResolutionReceiptPayload{Binding: signing.Binding{ProtocolVersion: 3, ServerID: server.ID.String(), TenantID: cmd.Scope.TenantID.String(), OutcomeID: cmd.Scope.OutcomeID.String(), PrincipalID: cc.PrincipalID, SignerKeyID: server.IssuerKeyID.String()}, CredentialID: identity.CredentialID.String(), RequestID: cc.CommandID.String(), IdempotencyKey: cc.IdempotencyKey, RequestDigest: signing.Digest(raw), OriginalCommandName: command, OriginalRequestID: b.RequestID, OriginalIdempotencyKey: b.IdempotencyKey, OriginalRequestDigest: request.OriginalRequestDigest, OriginalEnvelopeDigest: request.OriginalEnvelopeDigest, ContractID: b.ContractID, Decision: "closed_without_acceptance", Reason: request.Reason, OutcomeRevision: signing.Decimal(coord.Revision), ResolvedAt: now.UTC().Format(time.RFC3339Nano)}
		_, doc, e := s.issueSignedFact(ctx, u, server, cmd.Scope, d.ID(b.ContractID), kind, "resolution", cc.PrincipalID, signing.ReturnResolutionReceipt, payload, now)
		if e != nil {
			return zero, 0, e
		}
		receipt, e := json.Marshal(doc)
		if e != nil {
			return zero, 0, e
		}
		record := d.SignedReturnIntentResolution{Scope: cmd.Scope, PrincipalID: cc.PrincipalID, CredentialID: identity.CredentialID, OriginalCommandName: command, OriginalIdempotencyKey: b.IdempotencyKey, OriginalRequestID: d.ID(b.RequestID), OriginalRequestDigest: request.OriginalRequestDigest, OriginalEnvelopeDigest: request.OriginalEnvelopeDigest, ContractID: d.ID(b.ContractID), ResolutionRequestID: cc.CommandID, ResolutionIdempotencyKey: cc.IdempotencyKey, ResolutionDigest: signing.Digest(raw), IssuerKeyID: server.IssuerKeyID, Decision: payload.Decision, Reason: payload.Reason, OutcomeRevision: coord.Revision, Receipt: receipt, RecordedAt: now}
		if e = resolutions.InsertTerminal(ctx, record); e != nil {
			// A negative decision intentionally leaves the Outcome revision
			// unchanged. A concurrent Serializable snapshot can therefore miss
			// the winner after waiting for its Outcome row lock. The immutable
			// original-intent uniqueness check remains authoritative: roll back
			// this transaction and report the same conflict as a visible winner.
			if code, _ := d.ErrorCodeOf(e); code == d.ErrorCodeAlreadyExists {
				return zero, 0, d.NewError(d.ErrorCodeIdempotencyConflict, "another terminal resolution already exists")
			}
			return zero, 0, e
		}
		return SignedReturnResolutionResult{Decision: payload.Decision, ResolutionReceipt: &doc}, coord.Revision, nil
	})
}

func verifyResolutionReceipt(ctx context.Context, u ports.UnitOfWork, r d.SignedReturnIntentResolution, serverID string) (signing.Document, error) {
	var doc signing.Document
	if e := r.Validate(); e != nil {
		return doc, e
	}
	if e := signing.DecodeStrict(r.Receipt, &doc, 32<<10); e != nil {
		return doc, signingError(e)
	}
	registry, e := signingRepository(u)
	if e != nil {
		return doc, e
	}
	key, e := registry.Key(ctx, r.Scope.TenantID, r.IssuerKeyID)
	if e != nil {
		return doc, e
	}
	if key.Purpose != "issuer" || key.PrincipalID != "wos-server:"+serverID {
		return doc, d.NewError(d.ErrorCodeForbidden, "resolution issuer differs")
	}
	pub, e := key.PublicBytes()
	if e != nil {
		return doc, e
	}
	env, e := doc.Envelope()
	if e != nil {
		return doc, signingError(e)
	}
	raw, e := signing.VerifyHistorical(env, signing.ReturnResolutionReceipt, key.ID.String(), pub)
	if e != nil {
		return doc, signingError(e)
	}
	var p signing.ReturnResolutionReceiptPayload
	if e = signing.DecodeHistoricalPayload(raw, &p, 32<<10); e != nil {
		return doc, signingError(e)
	}
	if (p.ProtocolVersion != 2 && p.ProtocolVersion != 3) || p.ServerID != serverID || p.TenantID != r.Scope.TenantID.String() || p.OutcomeID != r.Scope.OutcomeID.String() || p.PrincipalID != r.PrincipalID || p.CredentialID != r.CredentialID.String() || p.ContractID != r.ContractID.String() || p.OriginalCommandName != r.OriginalCommandName || p.OriginalRequestID != r.OriginalRequestID.String() || p.OriginalIdempotencyKey != r.OriginalIdempotencyKey || p.OriginalRequestDigest != r.OriginalRequestDigest || p.OriginalEnvelopeDigest != r.OriginalEnvelopeDigest || p.RequestID != r.ResolutionRequestID.String() || p.IdempotencyKey != r.ResolutionIdempotencyKey || p.RequestDigest != r.ResolutionDigest || p.Decision != r.Decision || p.Reason != r.Reason || p.OutcomeRevision != signing.Decimal(r.OutcomeRevision) || p.ResolvedAt != r.RecordedAt.UTC().Format(time.RFC3339Nano) {
		return doc, d.NewError(d.ErrorCodeIdempotencyState, "resolution receipt metadata differs")
	}
	return doc, nil
}
