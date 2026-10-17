package memory

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type signedReturnResolutionRepository struct{ tx *transaction }

func (tx *transaction) SignedReturnResolutions() ports.SignedReturnResolutionRepository {
	return signedReturnResolutionRepository{tx}
}
func resolutionKey(ns d.ID, principal string, credential d.ID, command, key string) string {
	raw, _ := json.Marshal([]string{ns.String(), principal, credential.String(), command, key})
	return string(raw)
}
func (r signedReturnResolutionRepository) GetByOriginalIntent(ctx context.Context, ns d.ID, principal string, credential d.ID, command, key string) (d.SignedReturnIntentResolution, error) {
	var result d.SignedReturnIntentResolution
	if e := ctx.Err(); e != nil {
		return result, e
	}
	if e := r.tx.ensureOpen(); e != nil {
		return result, e
	}
	result, ok := r.tx.signedReview.Resolutions[resolutionKey(ns, principal, credential, command, key)]
	if !ok {
		return result, d.NewError(d.ErrorCodeNotFound, "terminal return resolution absent")
	}
	return deepCopy(result), result.Validate()
}
func (r signedReturnResolutionRepository) InsertTerminal(ctx context.Context, result d.SignedReturnIntentResolution) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	if e := r.tx.ensureOpen(); e != nil {
		return e
	}
	if e := result.Validate(); e != nil {
		return e
	}
	if result.OriginalCommandName == "ReturnSignedWork" {
		if _, e := r.tx.SignedWorkContracts().Get(ctx, result.Scope, result.ContractID); e != nil {
			return e
		}
	} else {
		if _, e := r.tx.SignedContracts().ReviewContract(ctx, result.Scope, result.ContractID); e != nil {
			return e
		}
	}
	if _, e := r.tx.SigningIdentity().Key(ctx, result.Scope.TenantID, result.IssuerKeyID); e != nil {
		return e
	}
	k := resolutionKey(result.Scope.TenantID, result.PrincipalID, result.CredentialID, result.OriginalCommandName, result.OriginalIdempotencyKey)
	for key, old := range r.tx.signedReview.Resolutions {
		if key == k || old.Scope.TenantID == result.Scope.TenantID && old.PrincipalID == result.PrincipalID && old.OriginalRequestID == result.OriginalRequestID {
			return d.NewError(d.ErrorCodeAlreadyExists, "terminal return resolution is immutable")
		}
	}
	r.tx.signedReview.Resolutions[k] = deepCopy(result)
	return nil
}

var _ ports.SignedReturnResolutionUnitOfWork = (*transaction)(nil)
