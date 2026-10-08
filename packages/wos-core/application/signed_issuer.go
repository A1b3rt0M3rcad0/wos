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

type signedRuntime struct {
	issuer ports.LocalContractSigner
	floor  d.AcceptanceMode
}

// ConfigureSignedIssuer is operator-only deployment composition, never a remote
// command. Pin public trust persistently before serving signed requests. Signing
// secrets must already be loaded into the local signer outside this transaction.
func (s *Service) ConfigureSignedIssuer(ctx context.Context, issuer ports.LocalContractSigner, floor d.AcceptanceMode) error {
	if issuer == nil || !floor.Valid() {
		return d.NewError(d.ErrorCodeInvalidConfig, "local signer and valid deployment floor required")
	}
	identity := issuer.Identity()
	if err := identity.Validate(); err != nil {
		return err
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return err
	}
	defer uow.Rollback()
	repository, ok := uow.(ports.ServerIdentityUnitOfWork)
	if !ok {
		return d.NewError(d.ErrorCodeInvalidConfig, "persistent server identity unavailable")
	}
	pinned, err := repository.ServerIdentity().Server(ctx)
	if code, _ := d.ErrorCodeOf(err); code == d.ErrorCodeNotFound {
		if err = repository.ServerIdentity().InsertServer(ctx, identity); err != nil {
			return err
		}
	} else if err != nil {
		return err
	} else if !reflect.DeepEqual(pinned, identity) {
		return d.NewError(d.ErrorCodeInvalidConfig, "local signer differs from persistent server identity; explicit recovery required")
	}
	if err = uow.Commit(); err != nil {
		return err
	}
	s.signed = &signedRuntime{issuer: issuer, floor: floor}
	return nil
}
func serverIdentityRepository(uow ports.UnitOfWork) (ports.ServerIdentityRepository, error) {
	repo, ok := uow.(ports.ServerIdentityUnitOfWork)
	if !ok {
		return nil, d.NewError(d.ErrorCodeInvalidConfig, "persistent server identity unavailable")
	}
	return repo.ServerIdentity(), nil
}
func signedRepository(uow ports.UnitOfWork) (ports.SignedContractRepository, ports.WorkContractRepository, error) {
	repo, ok := uow.(ports.SignedContractUnitOfWork)
	if !ok {
		return nil, nil, d.NewError(d.ErrorCodeInvalidConfig, "signed contracts unavailable")
	}
	return repo.SignedContracts(), repo.SignedWorkContracts(), nil
}
func (s *Service) requireSignedIssuer(ctx context.Context, uow ports.UnitOfWork, ns d.ID) (d.ServerIdentity, error) {
	var zero d.ServerIdentity
	if s.signed == nil || s.signed.issuer == nil {
		return zero, d.NewError(d.ErrorCodeInvalidConfig, "local signed issuer not configured")
	}
	repo, err := serverIdentityRepository(uow)
	if err != nil {
		return zero, err
	}
	server, err := repo.Server(ctx)
	if err != nil {
		return zero, err
	}
	if !reflect.DeepEqual(server, s.signed.issuer.Identity()) {
		return zero, d.NewError(d.ErrorCodeInvalidConfig, "signer/persistent server binding differs")
	}
	registry, err := signingRepository(uow)
	if err != nil {
		return zero, err
	}
	key, err := registry.Key(ctx, ns, server.IssuerKeyID)
	if code, _ := d.ErrorCodeOf(err); code == d.ErrorCodeNotFound {
		key = d.SigningKey{ID: server.IssuerKeyID, NamespaceID: ns, PrincipalID: server.PrincipalID(), Purpose: "issuer", Algorithm: "Ed25519", PublicKey: server.PublicKey, Fingerprint: server.Fingerprint, Version: 1, Status: "active", CreatedAt: server.CreatedAt, UpdatedAt: server.CreatedAt}
		if err = registry.SaveKey(ctx, key, 0); err != nil {
			return zero, err
		}
	} else if err != nil {
		return zero, err
	}
	if key.Purpose != "issuer" || key.Status != "active" || key.PrincipalID != server.PrincipalID() || key.PublicKey != server.PublicKey || key.Fingerprint != server.Fingerprint {
		return zero, d.NewError(d.ErrorCodeForbidden, "persistent server signing key inactive/different")
	}
	return server, nil
}
func (s *Service) issueSignedFact(ctx context.Context, uow ports.UnitOfWork, server d.ServerIdentity, scope d.Scope, contract d.ID, contractKind, kind, principal string, purpose signing.PayloadType, payload any, now time.Time) (d.SignedFact, signing.Document, error) {
	var fact d.SignedFact
	var document signing.Document
	raw, err := signing.Canonical(payload)
	if err != nil {
		return fact, document, signingError(err)
	}
	if kind == "specification" && len(raw) > 128*1024 {
		return fact, document, d.NewError(d.ErrorCodeGraphLimitExceeded, "signed specification exceeds 128 KiB")
	}
	proofRaw, err := s.signed.issuer.SignCanonical(string(purpose), raw)
	if err != nil {
		return fact, document, err
	}
	var proof signing.Proof
	if err = signing.DecodeStrict(proofRaw, &proof, 4096); err != nil {
		return fact, document, signingError(err)
	}
	document = signing.Document{Payload: json.RawMessage(raw), Proof: proof}
	envelope, err := document.Envelope()
	if err != nil {
		return fact, document, signingError(err)
	}
	public, err := server.PublicBytes()
	if err != nil {
		return fact, document, err
	}
	if _, err = signing.Verify(envelope, purpose, server.IssuerKeyID.String(), public); err != nil {
		return fact, document, signingError(err)
	}
	id, err := s.ids.NewID()
	if err != nil {
		return fact, document, err
	}
	fact = d.SignedFact{ID: id, Scope: scope, ContractID: contract, ContractKind: contractKind, Kind: kind, PrincipalID: principal, KeyID: server.IssuerKeyID, PayloadDigest: signing.Digest(raw), Payload: raw, Proof: proofRaw, RecordedAt: now.UTC()}
	return fact, document, nil
}
