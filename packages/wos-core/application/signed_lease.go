package application

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"time"
)

type RenewSignedWorkContractCommand struct {
	Scope                d.Scope
	ContractID           d.ID
	SignerKeyID          d.ID
	Authority            ContractAuthority
	AuthorityDigest      string
	ExpectedLeaseVersion d.Version
	TTLSeconds           int `wos:"optional"`
}
type ResumeSignedWorkContractCommand struct {
	Scope                d.Scope
	ContractID           d.ID
	SignerKeyID          d.ID
	Authority            ContractAuthority
	AuthorityDigest      string
	ExpectedLeaseVersion d.Version
}

func factDocument(f d.SignedFact) (signing.Document, error) {
	var proof signing.Proof
	if err := signing.DecodeStrict(f.Proof, &proof, 4096); err != nil {
		return signing.Document{}, signingError(err)
	}
	return signing.Document{Payload: json.RawMessage(f.Payload), Proof: proof}, nil
}
func verifyIssuedFact(ctx context.Context, uow ports.UnitOfWork, f d.SignedFact, purpose signing.PayloadType, serverID, principal string) (signing.Document, error) {
	document, err := factDocument(f)
	if err != nil {
		return document, err
	}
	registry, err := signingRepository(uow)
	if err != nil {
		return document, err
	}
	key, err := registry.Key(ctx, f.Scope.NamespaceID, f.KeyID)
	if err != nil {
		return document, err
	}
	if key.Purpose != "issuer" || key.PrincipalID != "wos-server:"+serverID {
		return document, d.NewError(d.ErrorCodeForbidden, "issued fact signer role differs")
	}
	public, err := key.PublicBytes()
	if err != nil {
		return document, err
	}
	envelope, err := document.Envelope()
	if err != nil {
		return document, signingError(err)
	}
	raw, err := signing.Verify(envelope, purpose, key.ID.String(), public)
	if err != nil {
		return document, signingError(err)
	}
	var binding signing.Binding
	// Purpose-specific schema is decoded by the caller after verification. Read
	// only authenticated bindings here; additional fields remain in the raw bytes.
	if err = json.Unmarshal(raw, &binding); err != nil {
		return document, err
	}
	if binding.ServerID != serverID || binding.NamespaceID != f.Scope.NamespaceID.String() || binding.OutcomeID != f.Scope.OutcomeID.String() || binding.PrincipalID != principal {
		return document, d.NewError(d.ErrorCodeForbidden, "issued fact authenticated scope differs")
	}
	return document, nil
}
func requireIssuedWorkAuthority(ctx context.Context, uow ports.UnitOfWork, c d.WorkContract, authority ContractAuthority, digest string) error {
	if c.SignedBinding == nil || c.LatestAuthorityID == nil || c.IssuedSpecificationID == nil || !d.ValidSignedDigest(digest) {
		return d.NewError(d.ErrorCodeSignedProtocolRequired, "persisted signed authority required")
	}
	repo, _, err := signedRepository(uow)
	if err != nil {
		return err
	}
	fact, err := repo.Fact(ctx, c.Scope, *c.LatestAuthorityID)
	if err != nil {
		return err
	}
	if fact.ContractID != c.ID || fact.Kind != "authority" || fact.PayloadDigest != digest || authority.SpecDigest != c.SignedBinding.SpecificationDigest || authority.ExecutionID != c.ExecutionID || authority.FencingToken != c.FencingToken {
		return d.NewError(d.ErrorCodeContractSpecMismatch, "authority/execution/spec reference differs")
	}
	document, err := verifyIssuedFact(ctx, uow, fact, signing.ContractAuthority, c.SignedBinding.ServerID, c.HolderPrincipalID)
	if err != nil {
		return err
	}
	var payload signing.AuthorityPayload
	if err = signing.DecodeStrict(document.Payload, &payload, signing.MaxPayloadBytes); err != nil {
		return signingError(err)
	}
	if payload.ContractID != c.ID.String() || payload.WorkItemID != c.WorkItemID.String() || payload.ContractKind != "execution" || payload.ExecutionID != c.ExecutionID.String() || payload.FencingToken != signing.Decimal(c.FencingToken) || payload.ContractVersion != signing.Decimal(c.Version) || payload.LeaseVersion != signing.Decimal(c.LeaseVersion) || payload.SpecDigest != c.SignedBinding.SpecificationDigest || payload.ExpiresAt != c.ExpiresAt.UTC().Format(time.RFC3339Nano) {
		return d.NewError(d.ErrorCodeContractSpecMismatch, "issued grant differs from authoritative lease")
	}
	return nil
}
func (s *Service) RenewSignedWorkContract(ctx context.Context, cc d.CommandContext, cmd RenewSignedWorkContractCommand) (MutationResult[WorkContractResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[WorkContractResult]{}, err
	}
	if cc.IdempotencyKey == "" {
		return MutationResult[WorkContractResult]{}, d.NewError(d.ErrorCodeInvalidArgument, "signed renewal requires idempotency intent")
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (WorkContractResult, d.OutcomeRevision, error) {
		return s.changeSignedWorkLease(ctx, uow, cc, cmd, false)
	})
}
func (s *Service) ResumeSignedWorkContract(ctx context.Context, cc d.CommandContext, cmd ResumeSignedWorkContractCommand) (MutationResult[WorkContractResult], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[WorkContractResult]{}, err
	}
	if cc.IdempotencyKey == "" {
		return MutationResult[WorkContractResult]{}, d.NewError(d.ErrorCodeInvalidArgument, "signed takeover requires idempotency intent")
	}
	return transactCommand(ctx, s, cc, cmd, func(uow ports.UnitOfWork) (WorkContractResult, d.OutcomeRevision, error) {
		return s.changeSignedWorkLease(ctx, uow, cc, RenewSignedWorkContractCommand{Scope: cmd.Scope, ContractID: cmd.ContractID, SignerKeyID: cmd.SignerKeyID, Authority: cmd.Authority, AuthorityDigest: cmd.AuthorityDigest, ExpectedLeaseVersion: cmd.ExpectedLeaseVersion}, true)
	})
}
func (s *Service) changeSignedWorkLease(ctx context.Context, uow ports.UnitOfWork, cc d.CommandContext, cmd RenewSignedWorkContractCommand, takeover bool) (WorkContractResult, d.OutcomeRevision, error) {
	var zero WorkContractResult
	if cmd.SignerKeyID.Validate() != nil {
		return zero, 0, d.NewError(d.ErrorCodeInvalidArgument, "agent signer key required")
	}
	identity, policy, err := s.signedAccess(ctx, uow, cmd.Scope, ports.PermissionWorkWrite, cmd.SignerKeyID)
	if err != nil {
		return zero, 0, err
	}
	server, err := s.requireSignedIssuer(ctx, uow, cmd.Scope.NamespaceID)
	if err != nil {
		return zero, 0, err
	}
	if _, err = uow.Coordination().LockOutcome(ctx, cmd.Scope); err != nil {
		return zero, 0, err
	}
	if err = requireActiveOutcome(ctx, uow, cmd.Scope); err != nil {
		return zero, 0, err
	}
	repo, workRepo, err := signedRepository(uow)
	if err != nil {
		return zero, 0, err
	}
	contract, err := workRepo.Get(ctx, cmd.Scope, cmd.ContractID)
	if err != nil {
		return zero, 0, err
	}
	if contract.SignedBinding == nil || contract.SignedBinding.CredentialID != identity.CredentialID || contract.HolderPrincipalID != identity.PrincipalID {
		return zero, 0, d.NewError(d.ErrorCodeForbidden, "contract belongs to another credential/Principal")
	}
	if err = requireIssuedWorkAuthority(ctx, uow, contract, cmd.Authority, cmd.AuthorityDigest); err != nil {
		return zero, 0, err
	}
	now, err := securityTransactionTime(ctx, uow, s.clock)
	if err != nil {
		return zero, 0, err
	}
	work, err := uow.WorkItems().Get(ctx, cmd.Scope, contract.WorkItemID)
	if err != nil {
		return zero, 0, err
	}
	version, lease := contract.Version, contract.LeaseVersion
	if takeover {
		execution, e := s.ids.NewID()
		if e != nil {
			return zero, 0, e
		}
		if err = contract.Resume(identity.PrincipalID, cmd.Authority.ExecutionID, cmd.Authority.FencingToken, cmd.ExpectedLeaseVersion, execution, work.LastFencingToken, now); err != nil {
			return zero, 0, err
		}
		expectedWork := work.Version
		work.LastFencingToken = uint64(contract.FencingToken)
		v, e := work.Version.Next()
		if e != nil {
			return zero, 0, e
		}
		work.Version = v
		work.UpdatedAt = now
		if err = uow.WorkItems().Save(ctx, work, expectedWork); err != nil {
			return zero, 0, err
		}
	} else {
		if err = contract.Renew(identity.PrincipalID, cmd.Authority.ExecutionID, cmd.Authority.FencingToken, cmd.ExpectedLeaseVersion, cmd.TTLSeconds, now); err != nil {
			return zero, 0, err
		}
	}
	allowed := []d.ID{}
	registry, err := signingRepository(uow)
	if err != nil {
		return zero, 0, err
	}
	for _, id := range policy.AllowedSigningKeyIDs {
		key, e := registry.Key(ctx, cmd.Scope.NamespaceID, id)
		if e != nil {
			return zero, 0, e
		}
		if key.Status == "active" && key.Purpose == "agent" && key.PrincipalID == identity.PrincipalID {
			allowed = append(allowed, id)
		}
	}
	floor, err := s.signedAcceptanceFloor(ctx, uow, cmd.Scope, work.ID, policy, contract.SignedBinding.AcceptanceFloor)
	if err != nil {
		return zero, 0, err
	}
	// The acquisition floor stays immutable. A hardened live floor is reflected
	// in each newly issued grant and rechecked again on signed return.
	authorityFact, document, err := s.issueWorkAuthorityWithFloor(ctx, uow, server, contract, policy, allowed, floor, now)
	if err != nil {
		return zero, 0, err
	}
	contract.LatestAuthorityID = &authorityFact.ID
	if err = workRepo.Save(ctx, contract, version, lease); err != nil {
		return zero, 0, err
	}
	if err = repo.InsertFact(ctx, authorityFact); err != nil {
		return zero, 0, err
	}
	revision, err := uow.Coordination().AdvanceOutcome(ctx, cmd.Scope)
	return WorkContractResult{Contract: contract, WorkItem: work, EvaluatedAt: now, IssuedAuthority: &document}, revision, err
}
func (s *Service) issueWorkAuthorityWithFloor(ctx context.Context, uow ports.UnitOfWork, server d.ServerIdentity, c d.WorkContract, policy d.CredentialPolicy, keys []d.ID, floor d.AcceptanceMode, now time.Time) (d.SignedFact, signing.Document, error) {
	copy := *c.SignedBinding
	copy.AcceptanceFloor = floor
	c.SignedBinding = &copy
	return s.issueWorkAuthority(ctx, uow, server, c, policy, keys, now)
}
