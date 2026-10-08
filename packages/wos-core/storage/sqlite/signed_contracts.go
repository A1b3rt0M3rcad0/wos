package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"strconv"
	"time"
)

type signedContractRepository struct{ u *unitOfWork }

func (u *unitOfWork) SignedContracts() ports.SignedContractRepository {
	return signedContractRepository{u}
}
func signedVersion(v d.Version) string { return strconv.FormatUint(uint64(v), 10) }
func (r signedContractRepository) Case(ctx context.Context, scope d.Scope, id d.ID) (d.ReviewCase, error) {
	c, err := signingRead[d.ReviewCase](ctx, r.u, `SELECT state_json FROM work_review_cases WHERE namespace_id=? AND outcome_id=? AND id=?`, scope.NamespaceID.String(), scope.OutcomeID.String(), id.String())
	if err == nil {
		err = c.Validate()
		if c.Scope != scope || c.ID != id {
			err = d.NewError(d.ErrorCodeInvalidScope, "review case scope differs")
		}
	}
	return c, err
}
func signedRows[T any](rows *sql.Rows, validate func(T) error) ([]T, error) {
	defer rows.Close()
	out := []T{}
	for rows.Next() {
		var raw string
		var v T
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(raw), &v); err != nil {
			return nil, err
		}
		if err := validate(v); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (r signedContractRepository) Cases(ctx context.Context, scope d.Scope, f ports.ReviewFilter) ([]d.ReviewCase, error) {
	if f.Limit < 1 || f.Limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "review case page outside 1..101")
	}
	rows, err := r.u.tx.QueryContext(ctx, `SELECT state_json FROM work_review_cases WHERE namespace_id=? AND outcome_id=? AND (?='' OR work_item_id=?) AND (?='' OR status=?) AND id>? ORDER BY id LIMIT ?`, scope.NamespaceID.String(), scope.OutcomeID.String(), f.WorkItemID.String(), f.WorkItemID.String(), f.Status, f.Status, f.After.String(), f.Limit)
	if err != nil {
		return nil, err
	}
	return signedRows(rows, func(c d.ReviewCase) error {
		if c.Scope != scope {
			return d.NewError(d.ErrorCodeInvalidScope, "review page scope differs")
		}
		return c.Validate()
	})
}
func (r signedContractRepository) OpenCase(ctx context.Context, scope d.Scope, work d.ID) (*d.ReviewCase, error) {
	var id string
	err := r.u.tx.QueryRowContext(ctx, `SELECT id FROM work_review_cases WHERE namespace_id=? AND outcome_id=? AND work_item_id=? AND status IN ('pending','in_review')`, scope.NamespaceID.String(), scope.OutcomeID.String(), work.String()).Scan(&id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	parsed, err := d.ParseID(id)
	if err != nil {
		return nil, err
	}
	c, err := r.Case(ctx, scope, parsed)
	return &c, err
}
func (r signedContractRepository) InsertCase(ctx context.Context, c d.ReviewCase) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if c.Status != d.ReviewPending || c.Version != 1 || c.CurrentContractID != nil {
		return d.NewError(d.ErrorCodeInvalidTransition, "new review must be pending")
	}
	work, err := r.u.SignedWorkContracts().Get(ctx, c.Scope, c.WorkContractID)
	if err != nil {
		return err
	}
	p, err := r.u.SignedWorkContracts().GetSubmission(ctx, c.Scope, c.SubmissionID)
	if err != nil {
		return err
	}
	if work.Status != d.ContractDelivered || work.WorkItemID != c.WorkItemID || work.LatestSubmissionID == nil || *work.LatestSubmissionID != c.SubmissionID || p.Digest != c.SubmissionDigest || p.Material.ContractID != work.ID {
		return d.NewError(d.ErrorCodeInvalidScope, "review immutable delivery target differs")
	}
	raw, err := marshalJSON(c)
	if err != nil {
		return err
	}
	_, err = r.u.tx.ExecContext(ctx, `INSERT INTO work_review_cases(id,namespace_id,outcome_id,work_item_id,work_contract_id,submission_id,status,version,state_json) VALUES(?,?,?,?,?,?,?,?,?)`, c.ID.String(), c.Scope.NamespaceID.String(), c.Scope.OutcomeID.String(), c.WorkItemID.String(), c.WorkContractID.String(), c.SubmissionID.String(), string(c.Status), signedVersion(c.Version), raw)
	return mapSQLError("insert review case", err)
}
func (r signedContractRepository) SaveCase(ctx context.Context, c d.ReviewCase, expected d.Version) error {
	old, err := r.Case(ctx, c.Scope, c.ID)
	if err != nil {
		return err
	}
	if old.Version != expected {
		return d.NewError(d.ErrorCodeVersionConflict, "review case CAS differs")
	}
	if err = d.ValidateReviewCaseUpdate(old, c); err != nil {
		return err
	}
	raw, err := marshalJSON(c)
	if err != nil {
		return err
	}
	result, err := r.u.tx.ExecContext(ctx, `UPDATE work_review_cases SET status=?,version=?,state_json=? WHERE namespace_id=? AND outcome_id=? AND id=? AND version=?`, string(c.Status), signedVersion(c.Version), raw, c.Scope.NamespaceID.String(), c.Scope.OutcomeID.String(), c.ID.String(), signedVersion(expected))
	return signingCAS(result, err)
}
func (r signedContractRepository) ReviewContract(ctx context.Context, scope d.Scope, id d.ID) (d.ReviewContract, error) {
	c, err := signingRead[d.ReviewContract](ctx, r.u, `SELECT state_json FROM work_review_contracts WHERE namespace_id=? AND outcome_id=? AND id=?`, scope.NamespaceID.String(), scope.OutcomeID.String(), id.String())
	if err == nil {
		err = c.Validate()
		if c.Scope != scope || c.ID != id {
			err = d.NewError(d.ErrorCodeInvalidScope, "review contract scope differs")
		}
	}
	return c, err
}
func (r signedContractRepository) ReviewContracts(ctx context.Context, scope d.Scope, f ports.ReviewFilter) ([]d.ReviewContract, error) {
	if f.Limit < 1 || f.Limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "review contract page outside 1..101")
	}
	rows, err := r.u.tx.QueryContext(ctx, `SELECT state_json FROM work_review_contracts WHERE namespace_id=? AND outcome_id=? AND (?='' OR work_item_id=?) AND (?='' OR holder_principal_id=?) AND (?='' OR status=?) AND id>? ORDER BY id LIMIT ?`, scope.NamespaceID.String(), scope.OutcomeID.String(), f.WorkItemID.String(), f.WorkItemID.String(), f.HolderPrincipalID, f.HolderPrincipalID, f.Status, f.Status, f.After.String(), f.Limit)
	if err != nil {
		return nil, err
	}
	return signedRows(rows, func(c d.ReviewContract) error {
		if c.Scope != scope {
			return d.NewError(d.ErrorCodeInvalidScope, "review contract page scope differs")
		}
		return c.Validate()
	})
}
func (r signedContractRepository) InsertReviewContract(ctx context.Context, c d.ReviewContract) error {
	if err := c.Validate(); err != nil {
		return err
	}
	if c.Status != d.ContractActive || c.Version != 1 || c.LeaseVersion != 1 {
		return d.NewError(d.ErrorCodeInvalidTransition, "new review authority must be active")
	}
	target, err := r.Case(ctx, c.Scope, c.CaseID)
	if err != nil {
		return err
	}
	copy := target
	if err = copy.Bind(c, c.AcquiredAt); err != nil {
		return err
	}
	if err = ensurePrincipal(ctx, r.u.tx, c.HolderPrincipalID); err != nil {
		return err
	}
	raw, err := marshalJSON(c)
	if err != nil {
		return err
	}
	_, err = r.u.tx.ExecContext(ctx, `INSERT INTO work_review_contracts(id,namespace_id,outcome_id,case_id,work_item_id,holder_principal_id,credential_id,status,version,lease_version,expires_at,state_json) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, c.ID.String(), c.Scope.NamespaceID.String(), c.Scope.OutcomeID.String(), c.CaseID.String(), c.WorkItemID.String(), c.HolderPrincipalID, c.Binding.CredentialID.String(), string(c.Status), signedVersion(c.Version), signedVersion(c.LeaseVersion), encodeTime(c.ExpiresAt), raw)
	return mapSQLError("insert review authority", err)
}
func (r signedContractRepository) SaveReviewContract(ctx context.Context, c d.ReviewContract, version, lease d.Version) error {
	old, err := r.ReviewContract(ctx, c.Scope, c.ID)
	if err != nil {
		return err
	}
	if old.Version != version || old.LeaseVersion != lease {
		return d.NewError(d.ErrorCodeVersionConflict, "review authority CAS differs")
	}
	if err = d.ValidateReviewContractUpdate(old, c); err != nil {
		return err
	}
	raw, err := marshalJSON(c)
	if err != nil {
		return err
	}
	result, err := r.u.tx.ExecContext(ctx, `UPDATE work_review_contracts SET status=?,version=?,lease_version=?,expires_at=?,state_json=? WHERE namespace_id=? AND outcome_id=? AND id=? AND version=? AND lease_version=?`, string(c.Status), signedVersion(c.Version), signedVersion(c.LeaseVersion), encodeTime(c.ExpiresAt), raw, c.Scope.NamespaceID.String(), c.Scope.OutcomeID.String(), c.ID.String(), signedVersion(version), signedVersion(lease))
	return signingCAS(result, err)
}
func (r signedContractRepository) Fact(ctx context.Context, scope d.Scope, id d.ID) (d.SignedFact, error) {
	f, err := signingRead[d.SignedFact](ctx, r.u, `SELECT state_json FROM signed_protocol_facts WHERE namespace_id=? AND outcome_id=? AND id=?`, scope.NamespaceID.String(), scope.OutcomeID.String(), id.String())
	if err == nil {
		err = f.Validate()
		if f.Scope != scope || f.ID != id {
			err = d.NewError(d.ErrorCodeInvalidScope, "signed fact scope differs")
		}
	}
	return f, err
}
func (r signedContractRepository) Facts(ctx context.Context, scope d.Scope, contract d.ID, kind string, after d.ID, limit int) ([]d.SignedFact, error) {
	if limit < 1 || limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "signed facts page outside 1..101")
	}
	rows, err := r.u.tx.QueryContext(ctx, `SELECT state_json FROM signed_protocol_facts WHERE namespace_id=? AND outcome_id=? AND contract_id=? AND (?='' OR kind=?) AND id>? ORDER BY id LIMIT ?`, scope.NamespaceID.String(), scope.OutcomeID.String(), contract.String(), kind, kind, after.String(), limit)
	if err != nil {
		return nil, err
	}
	return signedRows(rows, func(f d.SignedFact) error {
		if f.Scope != scope || f.ContractID != contract {
			return d.NewError(d.ErrorCodeInvalidScope, "signed facts page scope differs")
		}
		return f.Validate()
	})
}
func (r signedContractRepository) InsertFact(ctx context.Context, f d.SignedFact) error {
	if err := f.Validate(); err != nil {
		return err
	}
	if f.ContractKind == "execution" {
		if _, err := r.u.SignedWorkContracts().Get(ctx, f.Scope, f.ContractID); err != nil {
			return err
		}
	} else {
		if _, err := r.ReviewContract(ctx, f.Scope, f.ContractID); err != nil {
			return err
		}
	}
	key, err := r.u.SigningIdentity().Key(ctx, f.Scope.NamespaceID, f.KeyID)
	if err != nil {
		return err
	}
	if (f.Kind == "return" && (key.Purpose != "agent" || key.PrincipalID != f.PrincipalID)) || (f.Kind != "return" && key.Purpose != "issuer") {
		return d.NewError(d.ErrorCodeForbidden, "signed fact key role differs")
	}
	request := ""
	if f.RequestID != nil {
		request = f.RequestID.String()
	}
	raw, err := marshalJSON(f)
	if err != nil {
		return err
	}
	_, err = r.u.tx.ExecContext(ctx, `INSERT INTO signed_protocol_facts(id,namespace_id,outcome_id,contract_id,contract_kind,kind,principal_id,key_id,idempotency_key,request_id,payload_digest,state_json) VALUES(?,?,?,?,?,?,?,?,?,?,?,?)`, f.ID.String(), f.Scope.NamespaceID.String(), f.Scope.OutcomeID.String(), f.ContractID.String(), f.ContractKind, f.Kind, f.PrincipalID, f.KeyID.String(), f.IdempotencyKey, request, f.PayloadDigest, raw)
	return mapSQLError("insert immutable signed fact", err)
}
func (r signedContractRepository) Acceptance(ctx context.Context, ns d.ID, principal, key string) (d.SignedFact, error) {
	f, err := signingRead[d.SignedFact](ctx, r.u, `SELECT state_json FROM signed_protocol_facts WHERE namespace_id=? AND principal_id=? AND idempotency_key=? AND kind='acceptance'`, ns.String(), principal, key)
	if err == nil {
		err = f.Validate()
		if f.Scope.NamespaceID != ns || f.PrincipalID != principal || f.IdempotencyKey != key || f.Kind != "acceptance" {
			err = d.NewError(d.ErrorCodeInvalidScope, "signed acceptance binding differs")
		}
	}
	return f, err
}
func (r signedContractRepository) Counts(ctx context.Context, ns d.ID, principal string, credential d.ID, now time.Time) (ports.SignedActiveCounts, error) {
	var c ports.SignedActiveCounts
	err := r.u.tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE WHEN holder_principal_id=? THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN credential_id=? THEN 1 ELSE 0 END),0),COUNT(*) FROM signed_work_contracts WHERE namespace_id=? AND status='active' AND expires_at>?`, principal, credential.String(), ns.String(), encodeTime(now)).Scan(&c.PrincipalWork, &c.CredentialWork, &c.NamespaceWork)
	if err != nil {
		return c, err
	}
	err = r.u.tx.QueryRowContext(ctx, `SELECT COALESCE(SUM(CASE WHEN holder_principal_id=? THEN 1 ELSE 0 END),0),COALESCE(SUM(CASE WHEN credential_id=? THEN 1 ELSE 0 END),0),COUNT(*) FROM work_review_contracts WHERE namespace_id=? AND status='active' AND expires_at>?`, principal, credential.String(), ns.String(), encodeTime(now)).Scan(&c.PrincipalReview, &c.CredentialReview, &c.NamespaceReview)
	return c, err
}
