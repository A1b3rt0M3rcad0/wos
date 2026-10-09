package application

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func signedCommand(name string) bool {
	switch name {
	case "AcquireSignedWorkContract", "AcquireNextSignedWorkContract", "RenewSignedWorkContract", "ResumeSignedWorkContract", "ReturnSignedWork", "AcquireSignedReviewContract", "AcquireNextSignedReviewContract", "RenewSignedReviewContract", "ResumeSignedReviewContract", "ReturnSignedReview", "InterveneSignedReviewCase", "RevokeSignedContract", "ReconcileSignedContracts":
		return true
	}
	return false
}
func signedCutoverCommand(command any) bool {
	cmd, ok := command.(SetNamespaceWorkProtocolCommand)
	return ok && (cmd.Phase == d.WorkProtocolSignedDraining || cmd.Phase == d.WorkProtocolSigned)
}
func (s *Service) signedAccess(ctx context.Context, uow ports.UnitOfWork, scope d.Scope, permission ports.Permission, keyID d.ID) (Identity, d.CredentialPolicy, error) {
	var policy d.CredentialPolicy
	identity, ok := IdentityFromContext(ctx)
	if !ok || identity.NamespaceID != scope.NamespaceID || identity.CredentialDigest == "" {
		return identity, policy, d.NewError(d.ErrorCodeForbidden, "authenticated signed identity required")
	}
	protocol, err := protocolRepository(uow)
	if err != nil {
		return identity, policy, err
	}
	p, err := protocol.Lock(ctx, scope.NamespaceID, true)
	if err != nil {
		return identity, policy, err
	}
	if p.Phase != d.WorkProtocolSigned {
		return identity, policy, d.NewError(d.ErrorCodeSignedProtocolRequired, "Namespace has not activated signed-v2 protocol")
	}
	now, err := securityTransactionTime(ctx, uow, s.clock)
	if err != nil {
		return identity, policy, err
	}
	access, ok := uow.(ports.AccessSnapshotUnitOfWork)
	if !ok {
		return identity, policy, d.NewError(d.ErrorCodeInvalidConfig, "transactional signed access unavailable")
	}
	if err = access.AuthorizeAccessSnapshot(ctx, ports.AccessSnapshotRequest{Authorization: ports.AuthorizationRequest{NamespaceID: scope.NamespaceID, OutcomeID: scope.OutcomeID, PrincipalID: identity.PrincipalID, Permission: permission}, CredentialDigest: identity.CredentialDigest, Actor: identity.Actor, Now: now}); err != nil {
		return identity, policy, err
	}
	registry, err := signingRepository(uow)
	if err != nil {
		return identity, policy, err
	}
	credential, err := registry.CredentialByDigest(ctx, identity.CredentialDigest)
	if err != nil {
		return identity, policy, err
	}
	if credential.ParentDigest != "" {
		return identity, policy, d.NewError(d.ErrorCodeForbidden, "browser sessions cannot exercise signed authority")
	}
	policy, err = registry.CredentialPolicy(ctx, scope.NamespaceID, credential.ID)
	if err != nil || policy.PrincipalID != identity.PrincipalID || !policy.Permits(string(permission), scope.OutcomeID) {
		return identity, policy, d.NewError(d.ErrorCodeForbidden, "explicit signed credential policy required")
	}
	identity.CredentialID = credential.ID
	identity.CredentialPolicyRevision = policy.Version
	if !keyID.IsZero() {
		key, e := registry.Key(ctx, scope.NamespaceID, keyID)
		if e != nil {
			return identity, policy, e
		}
		if key.Purpose != "agent" || key.Status != "active" || key.PrincipalID != identity.PrincipalID || !policy.PermitsKey(keyID) {
			return identity, policy, d.NewError(d.ErrorCodeForbidden, "agent signing key inactive/different/outside credential policy")
		}
	}
	return identity, policy, nil
}
func (s *Service) signedAcceptanceFloor(ctx context.Context, uow ports.UnitOfWork, scope d.Scope, work d.ID, credential d.CredentialPolicy, frozen d.AcceptanceMode) (d.AcceptanceMode, error) {
	if s.signed == nil {
		return "", d.NewError(d.ErrorCodeInvalidConfig, "signed deployment not configured")
	}
	floor := d.StrongerAcceptance(s.signed.floor, credential.AcceptanceFloor, frozen)
	if s.independentReviewer {
		floor = d.AcceptanceIndependentReview
	}
	registry, err := signingRepository(uow)
	if err != nil {
		return "", err
	}
	for _, target := range []struct{ o, w *d.ID }{{nil, nil}, {&scope.OutcomeID, nil}, {&scope.OutcomeID, &work}} {
		p, e := registry.AcceptancePolicy(ctx, scope.NamespaceID, target.o, target.w)
		if e != nil {
			if code, _ := d.ErrorCodeOf(e); code == d.ErrorCodeNotFound {
				continue
			}
			return "", e
		}
		floor = d.StrongerAcceptance(floor, p.AcceptanceFloor)
	}
	return floor, nil
}
