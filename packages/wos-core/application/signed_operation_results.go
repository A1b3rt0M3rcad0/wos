package application

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

// Only a previously committed trusted cache result is eligible for promotion.
// The credential is recovered from the authoritative historical contract, never
// from a hint in the caller's new request or its authenticated replacement CID.
func legacySignedCacheCredential(ctx context.Context, u ports.UnitOfWork, scope d.Scope, value any) (d.ID, error) {
	switch result := value.(type) {
	case WorkContractResult:
		if result.Contract.Scope != scope || result.Contract.SignedBinding == nil {
			return "", d.NewError(d.ErrorCodeIdempotencyState, "cached signed execution binding absent")
		}
		return result.Contract.SignedBinding.CredentialID, nil
	case SignedReviewContractResult:
		if result.Contract.Scope != scope {
			return "", d.NewError(d.ErrorCodeIdempotencyState, "cached signed review scope differs")
		}
		return result.Contract.Binding.CredentialID, nil
	case SignedReturnResult:
		var receipt signing.ReceiptPayload
		if e := json.Unmarshal(result.Receipt.Payload, &receipt); e != nil {
			return "", e
		}
		if receipt.NamespaceID != scope.NamespaceID.String() || receipt.OutcomeID != scope.OutcomeID.String() {
			return "", d.NewError(d.ErrorCodeIdempotencyState, "cached acceptance scope differs")
		}
		contracts, ok := u.(ports.SignedContractUnitOfWork)
		if !ok {
			return "", d.NewError(d.ErrorCodeInvalidConfig, "signed contract repository required")
		}
		contract, e := contracts.SignedWorkContracts().Get(ctx, scope, d.ID(receipt.ContractID))
		if e == nil {
			if contract.SignedBinding == nil {
				return "", d.NewError(d.ErrorCodeIdempotencyState, "cached acceptance references an unsigned contract")
			}
			return contract.SignedBinding.CredentialID, nil
		}
		if code, _ := d.ErrorCodeOf(e); code != d.ErrorCodeNotFound {
			return "", e
		}
		review, e := contracts.SignedContracts().ReviewContract(ctx, scope, d.ID(receipt.ContractID))
		if e != nil {
			return "", e
		}
		return review.Binding.CredentialID, nil
	default:
		return "", d.NewError(d.ErrorCodeSignedProtocolRequired, "legacy signed administrative cache requires explicit reconciliation through scoped state")
	}
}

func legacySignedOperationView(ctx context.Context, u ports.UnitOfWork, scope d.Scope, principal, key string, legacy d.LegacySignedOperationResult) (d.SignedOperationResult, error) {
	var value any
	switch legacy.CommandName {
	case "AcquireSignedWorkContract", "AcquireNextSignedWorkContract", "RenewSignedWorkContract", "ResumeSignedWorkContract":
		var result MutationResult[WorkContractResult]
		if e := json.Unmarshal(legacy.Result.ResponseJSON, &result); e != nil {
			return d.SignedOperationResult{}, e
		}
		value = result.Value
	case "AcquireSignedReviewContract", "RenewSignedReviewContract", "ResumeSignedReviewContract":
		var result MutationResult[SignedReviewContractResult]
		if e := json.Unmarshal(legacy.Result.ResponseJSON, &result); e != nil {
			return d.SignedOperationResult{}, e
		}
		value = result.Value
	case "ReturnSignedWork", "ReturnSignedReview":
		var result MutationResult[SignedReturnResult]
		if e := json.Unmarshal(legacy.Result.ResponseJSON, &result); e != nil {
			return d.SignedOperationResult{}, e
		}
		value = result.Value
	default:
		return d.SignedOperationResult{}, d.NewError(d.ErrorCodeSignedProtocolRequired, "legacy signed administration requires explicit scoped reconciliation")
	}
	cid, e := legacySignedCacheCredential(ctx, u, scope, value)
	if e != nil {
		return d.SignedOperationResult{}, e
	}
	return d.SignedOperationResult{Scope: scope, PrincipalID: principal, CredentialID: cid, CommandName: legacy.CommandName, IdempotencyKey: key, Fingerprint: legacy.Fingerprint, Result: legacy.Result, RecordedAt: legacy.RecordedAt}, nil
}
