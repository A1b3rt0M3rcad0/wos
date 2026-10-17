package sqlite

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type signedReturnResolutionRepository struct{ u *unitOfWork }

func (u *unitOfWork) SignedReturnResolutions() ports.SignedReturnResolutionRepository {
	return signedReturnResolutionRepository{u}
}
func (r signedReturnResolutionRepository) GetByOriginalIntent(ctx context.Context, ns d.ID, principal string, credential d.ID, command, key string) (d.SignedReturnIntentResolution, error) {
	var result d.SignedReturnIntentResolution
	var raw string
	e := r.u.tx.QueryRowContext(ctx, `SELECT state_json FROM signed_return_intent_resolutions WHERE tenant_id=? AND principal_id=? AND original_credential_id=? AND original_command_name=? AND original_idempotency_key=?`, ns.String(), principal, credential.String(), command, key).Scan(&raw)
	if e != nil {
		return result, mapSQLError("read terminal return resolution", e)
	}
	if e = unmarshalJSON(raw, &result); e != nil {
		return result, d.WrapError(d.ErrorCodeIdempotencyState, "invalid persisted terminal resolution", e)
	}
	if result.Scope.TenantID != ns || result.PrincipalID != principal || result.CredentialID != credential || result.OriginalCommandName != command || result.OriginalIdempotencyKey != key {
		return result, d.NewError(d.ErrorCodeIdempotencyState, "persisted terminal resolution identity differs")
	}
	return result, result.Validate()
}
func (r signedReturnResolutionRepository) InsertTerminal(ctx context.Context, result d.SignedReturnIntentResolution) error {
	if e := result.Validate(); e != nil {
		return e
	}
	raw, e := json.Marshal(result)
	if e != nil {
		return e
	}
	var execution, review *d.ID
	if result.OriginalCommandName == "ReturnSignedWork" {
		execution = &result.ContractID
	} else {
		review = &result.ContractID
	}
	_, e = r.u.tx.ExecContext(ctx, `INSERT INTO signed_return_intent_resolutions(tenant_id,outcome_id,principal_id,original_credential_id,original_command_name,original_idempotency_key,original_request_id,issuer_key_id,execution_contract_id,review_contract_id,state_json) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, result.Scope.TenantID.String(), result.Scope.OutcomeID.String(), result.PrincipalID, result.CredentialID.String(), result.OriginalCommandName, result.OriginalIdempotencyKey, result.OriginalRequestID.String(), result.IssuerKeyID.String(), nullableID(execution), nullableID(review), string(raw))
	return mapSQLError("insert immutable terminal resolution", e)
}

var _ ports.SignedReturnResolutionUnitOfWork = (*unitOfWork)(nil)
