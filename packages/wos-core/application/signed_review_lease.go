package application

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"time"
)

type RenewSignedReviewContractCommand struct {
	Scope                d.Scope
	ContractID           d.ID
	SignerKeyID          d.ID
	Authority            ContractAuthority
	AuthorityDigest      string
	ExpectedLeaseVersion d.Version
	TTLSeconds           int `wos:"optional"`
}
type ResumeSignedReviewContractCommand struct {
	Scope                d.Scope
	ContractID           d.ID
	SignerKeyID          d.ID
	Authority            ContractAuthority
	AuthorityDigest      string
	ExpectedLeaseVersion d.Version
}

func requireIssuedReviewAuthority(ctx context.Context, u ports.UnitOfWork, c d.ReviewContract, authority ContractAuthority, digest string) (signing.AuthorityPayload, error) {
	var zero signing.AuthorityPayload
	if c.LatestAuthorityID == nil || c.IssuedSpecificationID == nil || !d.ValidSignedDigest(digest) {
		return zero, d.NewError(d.ErrorCodeSignedProtocolRequired, "persisted review grant/specification required")
	}
	repo, _, err := signedRepository(u)
	if err != nil {
		return zero, err
	}
	f, err := repo.Fact(ctx, c.Scope, *c.LatestAuthorityID)
	if err != nil {
		return zero, err
	}
	if f.ContractKind != "review" || f.Kind != "authority" || f.ContractID != c.ID || f.PayloadDigest != digest || authority.SpecDigest != c.Binding.SpecificationDigest || authority.ExecutionID != c.ExecutionID || authority.FencingToken != c.FencingToken {
		return zero, d.NewError(d.ErrorCodeContractSpecMismatch, "review issued grant differs")
	}
	doc, err := verifyIssuedFact(ctx, u, f, signing.ContractAuthority, c.Binding.ServerID, c.HolderPrincipalID)
	if err != nil {
		return zero, err
	}
	if err = signing.DecodeStrict(doc.Payload, &zero, 128*1024); err != nil {
		return zero, signingError(err)
	}
	if zero.ContractKind != "review" || zero.ContractID != c.ID.String() || zero.WorkItemID != c.WorkItemID.String() || zero.ReviewCaseID != c.CaseID.String() || zero.ExecutionID != c.ExecutionID.String() || zero.FencingToken != signing.Decimal(c.FencingToken) || zero.ContractVersion != signing.Decimal(c.Version) || zero.LeaseVersion != signing.Decimal(c.LeaseVersion) || zero.SpecDigest != c.Binding.SpecificationDigest || zero.ExpiresAt != c.ExpiresAt.UTC().Format(time.RFC3339Nano) {
		return zero, d.NewError(d.ErrorCodeContractSpecMismatch, "authenticated review grant is stale")
	}
	return zero, nil
}
func (s *Service) RenewSignedReviewContract(ctx context.Context, cc d.CommandContext, cmd RenewSignedReviewContractCommand) (MutationResult[SignedReviewContractResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[SignedReviewContractResult]{}, err
	}
	if cc.IdempotencyKey == "" {
		return MutationResult[SignedReviewContractResult]{}, d.NewError(d.ErrorCodeInvalidArgument, "review renewal requires idempotency")
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (SignedReviewContractResult, d.OutcomeRevision, error) {
		return s.changeSignedReviewLease(ctx, u, cc, cmd, false)
	})
}
func (s *Service) ResumeSignedReviewContract(ctx context.Context, cc d.CommandContext, cmd ResumeSignedReviewContractCommand) (MutationResult[SignedReviewContractResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[SignedReviewContractResult]{}, err
	}
	if cc.IdempotencyKey == "" {
		return MutationResult[SignedReviewContractResult]{}, d.NewError(d.ErrorCodeInvalidArgument, "review takeover requires idempotency")
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (SignedReviewContractResult, d.OutcomeRevision, error) {
		return s.changeSignedReviewLease(ctx, u, cc, RenewSignedReviewContractCommand{Scope: cmd.Scope, ContractID: cmd.ContractID, SignerKeyID: cmd.SignerKeyID, Authority: cmd.Authority, AuthorityDigest: cmd.AuthorityDigest, ExpectedLeaseVersion: cmd.ExpectedLeaseVersion}, true)
	})
}
func (s *Service) changeSignedReviewLease(ctx context.Context, u ports.UnitOfWork, cc d.CommandContext, cmd RenewSignedReviewContractCommand, resume bool) (SignedReviewContractResult, d.OutcomeRevision, error) {
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
	contract, err := repo.ReviewContract(ctx, cmd.Scope, cmd.ContractID)
	if err != nil {
		return zero, 0, err
	}
	if contract.Binding.CredentialID != identity.CredentialID || contract.HolderPrincipalID != identity.PrincipalID {
		return zero, 0, d.NewError(d.ErrorCodeForbidden, "review lease belongs to another credential")
	}
	if _, err = requireIssuedReviewAuthority(ctx, u, contract, cmd.Authority, cmd.AuthorityDigest); err != nil {
		return zero, 0, err
	}
	review, err := repo.Case(ctx, cmd.Scope, contract.CaseID)
	if err != nil {
		return zero, 0, err
	}
	if review.CurrentContractID == nil || *review.CurrentContractID != contract.ID || review.Status != d.ReviewInReview {
		return zero, 0, d.NewError(d.ErrorCodeVersionConflict, "review case no longer holds authority")
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
	if err = review.RequireIndependent(identity.PrincipalID, contract.SeparationGroup); err != nil {
		return zero, 0, err
	}
	now, err := securityTransactionTime(ctx, u, s.clock)
	if err != nil {
		return zero, 0, err
	}
	v, l := contract.Version, contract.LeaseVersion
	if resume {
		id, e := s.ids.NewID()
		if e != nil {
			return zero, 0, e
		}
		err = contract.Resume(identity.PrincipalID, cmd.Authority.ExecutionID, cmd.Authority.FencingToken, cmd.ExpectedLeaseVersion, id, now)
	} else {
		err = contract.Renew(identity.PrincipalID, cmd.Authority.ExecutionID, cmd.Authority.FencingToken, cmd.ExpectedLeaseVersion, cmd.TTLSeconds, now)
	}
	if err != nil {
		return zero, 0, err
	}
	allowed, err := liveSignedKeys(ctx, u, policy)
	if err != nil {
		return zero, 0, err
	}
	f, document, err := s.issueReviewAuthority(ctx, u, server, contract, policy, allowed, now)
	if err != nil {
		return zero, 0, err
	}
	contract.LatestAuthorityID = &f.ID
	if err = repo.SaveReviewContract(ctx, contract, v, l); err != nil {
		return zero, 0, err
	}
	if err = repo.InsertFact(ctx, f); err != nil {
		return zero, 0, err
	}
	work, err := u.WorkItems().Get(ctx, cmd.Scope, contract.WorkItemID)
	if err != nil {
		return zero, 0, err
	}
	revision, err := u.Coordination().AdvanceOutcome(ctx, cmd.Scope)
	return SignedReviewContractResult{Contract: contract, Case: review, WorkItemVersion: signing.Decimal(work.Version), IssuedAuthority: document, EvaluatedAt: now, resumed: resume}, revision, err
}
