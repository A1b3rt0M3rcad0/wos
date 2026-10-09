package sqlite

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"strconv"
)

type signedOperationRepository struct{ u *unitOfWork }

func (r signedOperationRepository) FindLegacy(ctx context.Context, namespace d.ID, principal, key string) (d.LegacySignedOperationResult, error) {
	var result d.LegacySignedOperationResult
	rows, e := r.u.tx.QueryContext(ctx, `SELECT command_name,request_hash,command_id,outcome_revision,response_json,created_at FROM idempotency_records WHERE namespace_id=? AND principal_id=? AND idempotency_key=? AND status='completed' AND command_name IN ('AcquireSignedWorkContract','AcquireNextSignedWorkContract','RenewSignedWorkContract','ResumeSignedWorkContract','ReturnSignedWork','AcquireSignedReviewContract','AcquireNextSignedReviewContract','RenewSignedReviewContract','ResumeSignedReviewContract','ReturnSignedReview','InterveneSignedReviewCase','RevokeSignedContract','ReconcileSignedContracts') ORDER BY command_name LIMIT 2`, namespace.String(), principal, key)
	if e != nil {
		return result, e
	}
	defer rows.Close()
	if !rows.Next() {
		if e = rows.Err(); e != nil {
			return result, e
		}
		return result, d.NewError(d.ErrorCodeNotFound, "legacy signed result absent")
	}
	var response, revision string
	var recorded int64
	if e = rows.Scan(&result.CommandName, &result.Fingerprint, &result.Result.CommandID, &revision, &response, &recorded); e != nil {
		return result, e
	}
	if rows.Next() {
		return result, d.NewError(d.ErrorCodeIdempotencyConflict, "legacy intention names multiple signed commands; explicit reconciliation required")
	}
	if e = rows.Err(); e != nil {
		return result, e
	}
	number, e := strconv.ParseUint(revision, 10, 64)
	if e != nil {
		return result, e
	}
	result.Result.OutcomeRevision = d.OutcomeRevision(number)
	result.Result.ResponseJSON = json.RawMessage(response)
	result.RecordedAt = decodeTime(recorded)
	if len(response) > d.MaxSignedOperationResultBytes {
		return result, d.NewError(d.ErrorCodeIdempotencyState, "legacy result exceeds bounded recovery size")
	}
	return result, result.Result.Validate()
}

func (u *unitOfWork) SignedOperations() ports.SignedOperationRepository {
	return signedOperationRepository{u}
}
func (r signedOperationRepository) Get(ctx context.Context, namespace d.ID, principal, key string) (d.SignedOperationResult, error) {
	var result d.SignedOperationResult
	var response, revision string
	var recorded int64
	e := r.u.tx.QueryRowContext(ctx, `SELECT namespace_id,outcome_id,principal_id,credential_id,idempotency_key,command_name,fingerprint,command_id,outcome_revision,response_json,recorded_at FROM signed_operation_results WHERE namespace_id=? AND principal_id=? AND idempotency_key=?`, namespace.String(), principal, key).Scan(&result.Scope.NamespaceID, &result.Scope.OutcomeID, &result.PrincipalID, &result.CredentialID, &result.IdempotencyKey, &result.CommandName, &result.Fingerprint, &result.Result.CommandID, &revision, &response, &recorded)
	if e != nil {
		return result, mapSQLError("read durable signed operation", e)
	}
	number, e := strconv.ParseUint(revision, 10, 64)
	if e != nil || strconv.FormatUint(number, 10) != revision {
		return result, d.NewError(d.ErrorCodeIdempotencyState, "invalid durable result revision")
	}
	result.Result.OutcomeRevision = d.OutcomeRevision(number)
	result.Result.ResponseJSON = json.RawMessage(response)
	result.RecordedAt = decodeTime(recorded)
	return result, result.Validate()
}
func (r signedOperationRepository) Insert(ctx context.Context, result d.SignedOperationResult) error {
	if e := result.Validate(); e != nil {
		return e
	}
	_, e := r.u.tx.ExecContext(ctx, `INSERT INTO signed_operation_results(namespace_id,outcome_id,principal_id,credential_id,idempotency_key,command_name,fingerprint,command_id,outcome_revision,response_json,recorded_at) VALUES(?,?,?,?,?,?,?,?,?,?,?)`, result.Scope.NamespaceID.String(), result.Scope.OutcomeID.String(), result.PrincipalID, result.CredentialID.String(), result.IdempotencyKey, result.CommandName, result.Fingerprint, result.Result.CommandID.String(), strconv.FormatUint(uint64(result.Result.OutcomeRevision), 10), string(result.Result.ResponseJSON), encodeTime(result.RecordedAt))
	return mapSQLError("insert durable signed operation", e)
}

var _ ports.SignedOperationUnitOfWork = (*unitOfWork)(nil)
