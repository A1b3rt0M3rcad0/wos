package memory

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type signedOperationRepository struct{ tx *transaction }

func (r signedOperationRepository) FindLegacy(ctx context.Context, namespace d.ID, principal, key string) (d.LegacySignedOperationResult, error) {
	var result d.LegacySignedOperationResult
	if e := ctx.Err(); e != nil {
		return result, e
	}
	if e := r.tx.ensureOpen(); e != nil {
		return result, e
	}
	found := false
	for _, cached := range r.tx.idempotency {
		if !cached.Completed || cached.Identity.NamespaceID != namespace || cached.Identity.PrincipalID != principal || cached.Identity.IdempotencyKey != key {
			continue
		}
		name := cached.Identity.CommandName
		switch name {
		case "AcquireSignedWorkContract", "AcquireNextSignedWorkContract", "RenewSignedWorkContract", "ResumeSignedWorkContract", "ReturnSignedWork", "AcquireSignedReviewContract", "RenewSignedReviewContract", "ResumeSignedReviewContract", "ReturnSignedReview", "InterveneSignedReviewCase", "RevokeSignedContract", "ReconcileSignedContracts":
		default:
			continue
		}
		if found {
			return result, d.NewError(d.ErrorCodeIdempotencyConflict, "legacy intention names multiple signed commands")
		}
		found = true
		stored := cached.Result
		stored.ResponseJSON = append(json.RawMessage(nil), stored.ResponseJSON...)
		result = d.LegacySignedOperationResult{CommandName: name, Fingerprint: cached.Fingerprint, Result: stored}
	}
	if !found {
		return result, d.NewError(d.ErrorCodeNotFound, "legacy signed result absent")
	}
	if len(result.Result.ResponseJSON) > d.MaxSignedOperationResultBytes {
		return result, d.NewError(d.ErrorCodeIdempotencyState, "legacy result exceeds bounded recovery size")
	}
	return result, result.Result.Validate()
}

func (tx *transaction) SignedOperations() ports.SignedOperationRepository {
	return signedOperationRepository{tx}
}
func signedOperationKey(ns d.ID, principal, key string) string {
	raw, _ := json.Marshal([]string{ns.String(), principal, key})
	return string(raw)
}
func cloneSignedOperation(result d.SignedOperationResult) d.SignedOperationResult {
	result.Result.ResponseJSON = append(json.RawMessage(nil), result.Result.ResponseJSON...)
	return result
}
func (r signedOperationRepository) Get(ctx context.Context, namespace d.ID, principal, key string) (d.SignedOperationResult, error) {
	if e := ctx.Err(); e != nil {
		return d.SignedOperationResult{}, e
	}
	if e := r.tx.ensureOpen(); e != nil {
		return d.SignedOperationResult{}, e
	}
	result, ok := r.tx.signedReview.Operations[signedOperationKey(namespace, principal, key)]
	if !ok {
		return result, d.NewError(d.ErrorCodeNotFound, "durable signed operation absent")
	}
	return cloneSignedOperation(result), result.Validate()
}
func (r signedOperationRepository) Insert(ctx context.Context, result d.SignedOperationResult) error {
	if e := ctx.Err(); e != nil {
		return e
	}
	if e := r.tx.ensureOpen(); e != nil {
		return e
	}
	if e := result.Validate(); e != nil {
		return e
	}
	key := signedOperationKey(result.Scope.NamespaceID, result.PrincipalID, result.IdempotencyKey)
	if _, exists := r.tx.signedReview.Operations[key]; exists {
		return d.NewError(d.ErrorCodeIdempotencyConflict, "durable signed operation is immutable")
	}
	r.tx.signedReview.Operations[key] = cloneSignedOperation(result)
	return nil
}

var _ ports.SignedOperationUnitOfWork = (*transaction)(nil)
