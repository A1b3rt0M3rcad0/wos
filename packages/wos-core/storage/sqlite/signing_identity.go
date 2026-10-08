package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"reflect"
	"strings"
	"time"
)

type signingIdentityRepository struct{ u *unitOfWork }

func (u *unitOfWork) SigningIdentity() ports.SigningIdentityRepository {
	return signingIdentityRepository{u}
}
func (r signingIdentityRepository) LockNamespace(ctx context.Context, ns d.ID) (d.Version, error) {
	var v d.Version
	err := r.u.tx.QueryRowContext(ctx, `SELECT version FROM namespaces WHERE id=? AND lifecycle='active'`, ns.String()).Scan(&v)
	return v, mapSQLError("lock signing namespace", err)
}
func signingRead[T any](ctx context.Context, u *unitOfWork, q string, args ...any) (T, error) {
	var v T
	var raw string
	err := u.tx.QueryRowContext(ctx, q, args...).Scan(&raw)
	if err != nil {
		return v, mapSQLError("read signing state", err)
	}
	err = json.Unmarshal([]byte(raw), &v)
	return v, err
}
func (r signingIdentityRepository) Credential(ctx context.Context, ns, id d.ID) (ports.Credential, error) {
	return r.credential(ctx, `SELECT id,namespace_id,principal_id,actor_json,expires_at,revoked,COALESCE(parent_digest,''),digest FROM credentials WHERE namespace_id=? AND id=?`, ns.String(), id.String())
}
func (r signingIdentityRepository) CredentialByDigest(ctx context.Context, digest string) (ports.Credential, error) {
	return r.credential(ctx, `SELECT id,namespace_id,principal_id,actor_json,expires_at,revoked,COALESCE(parent_digest,''),digest FROM credentials WHERE digest=?`, digest)
}
func (r signingIdentityRepository) credential(ctx context.Context, q string, args ...any) (ports.Credential, error) {
	var c ports.Credential
	var id, ns, actor string
	var expiry int64
	err := r.u.tx.QueryRowContext(ctx, q, args...).Scan(&id, &ns, &c.PrincipalID, &actor, &expiry, &c.Revoked, &c.ParentDigest, &c.Digest)
	if err != nil {
		return c, mapSQLError("read signing credential", err)
	}
	c.ID, err = d.ParseID(id)
	if err != nil {
		return c, err
	}
	c.NamespaceID, err = d.ParseID(ns)
	if err != nil {
		return c, err
	}
	c.ExpiresAt = decodeTime(expiry)
	err = json.Unmarshal([]byte(actor), &c.Actor)
	return c, err
}
func (r signingIdentityRepository) Key(ctx context.Context, ns, id d.ID) (d.SigningKey, error) {
	k, err := signingRead[d.SigningKey](ctx, r.u, `SELECT state_json FROM signing_keys WHERE namespace_id=? AND id=?`, ns.String(), id.String())
	if err == nil {
		err = k.Validate()
		if k.NamespaceID != ns || k.ID != id {
			err = d.NewError(d.ErrorCodeInvalidScope, "key scope mismatch")
		}
	}
	return k, err
}
func (r signingIdentityRepository) Keys(ctx context.Context, ns d.ID, principal string, limit int) ([]d.SigningKey, error) {
	if limit < 1 || limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "key list limit outside 1..101")
	}
	rows, err := r.u.tx.QueryContext(ctx, `SELECT state_json FROM signing_keys WHERE namespace_id=? AND principal_id=? ORDER BY id LIMIT ?`, ns.String(), principal, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []d.SigningKey{}
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var k d.SigningKey
		if err = json.Unmarshal([]byte(raw), &k); err != nil {
			return nil, err
		}
		if err = k.Validate(); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}
func (r signingIdentityRepository) SaveKey(ctx context.Context, k d.SigningKey, expected d.Version) error {
	if err := k.Validate(); err != nil {
		return err
	}
	if k.Version != expected+1 {
		return d.NewError(d.ErrorCodeVersionConflict, "key version must advance once")
	}
	if expected > 0 {
		old, err := r.Key(ctx, k.NamespaceID, k.ID)
		if err != nil {
			return err
		}
		if old.Version != expected {
			return d.NewError(d.ErrorCodeVersionConflict, "key version changed")
		}
		o, n := old, k
		o.Status = n.Status
		o.Version = n.Version
		o.UpdatedAt = n.UpdatedAt
		o.Reason = n.Reason
		if !reflect.DeepEqual(o, n) || k.Status == "active" || old.Status == "revoked" {
			return d.NewError(d.ErrorCodeInvalidTransition, "public key identity/history is immutable")
		}
	}
	raw, err := marshalJSON(k)
	if err != nil {
		return err
	}
	if expected == 0 {
		_, err = r.u.tx.ExecContext(ctx, `INSERT INTO signing_keys(namespace_id,id,principal_id,purpose,fingerprint,status,version,state_json) VALUES(?,?,?,?,?,?,?,?)`, k.NamespaceID.String(), k.ID.String(), k.PrincipalID, k.Purpose, k.Fingerprint, k.Status, k.Version, raw)
		return mapSQLError("create signing key", err)
	}
	result, err := r.u.tx.ExecContext(ctx, `UPDATE signing_keys SET status=?,version=?,state_json=? WHERE namespace_id=? AND id=? AND version=?`, k.Status, k.Version, raw, k.NamespaceID.String(), k.ID.String(), expected)
	return signingCAS(result, err)
}
func (r signingIdentityRepository) Enrollment(ctx context.Context, ns, id d.ID) (d.SigningEnrollment, error) {
	e, err := signingRead[d.SigningEnrollment](ctx, r.u, `SELECT state_json FROM signing_key_enrollments WHERE namespace_id=? AND id=?`, ns.String(), id.String())
	if err == nil {
		err = e.Validate()
	}
	return e, err
}
func (r signingIdentityRepository) SaveEnrollment(ctx context.Context, e d.SigningEnrollment, expected d.Version) error {
	if err := e.Validate(); err != nil {
		return err
	}
	if e.Version != expected+1 {
		return d.NewError(d.ErrorCodeVersionConflict, "enrollment version changed")
	}
	if expected > 0 {
		old, err := r.Enrollment(ctx, e.NamespaceID, e.ID)
		if err != nil {
			return err
		}
		o, n := old, e
		o.Version = n.Version
		o.Consumed = n.Consumed
		o.AcceptedDigest = n.AcceptedDigest
		if old.Version != expected || old.Consumed || !e.Consumed || e.AcceptedDigest == "" || !reflect.DeepEqual(o, n) {
			return d.NewError(d.ErrorCodeInvalidTransition, "enrollment challenge is immutable/one-use")
		}
	}
	raw, err := marshalJSON(e)
	if err != nil {
		return err
	}
	if expected == 0 {
		_, err = r.u.tx.ExecContext(ctx, `INSERT INTO signing_key_enrollments(namespace_id,id,principal_id,credential_id,version,expires_at,consumed,state_json) VALUES(?,?,?,?,?,?,?,?)`, e.NamespaceID.String(), e.ID.String(), e.PrincipalID, e.CredentialID.String(), e.Version, encodeTime(e.ExpiresAt), boolInt(e.Consumed), raw)
		return mapSQLError("create signing enrollment", err)
	}
	result, err := r.u.tx.ExecContext(ctx, `UPDATE signing_key_enrollments SET version=?,consumed=?,state_json=? WHERE namespace_id=? AND id=? AND version=? AND consumed=0`, e.Version, boolInt(e.Consumed), raw, e.NamespaceID.String(), e.ID.String(), expected)
	return signingCAS(result, err)
}
func (r signingIdentityRepository) CredentialPolicy(ctx context.Context, ns, id d.ID) (d.CredentialPolicy, error) {
	p, err := signingRead[d.CredentialPolicy](ctx, r.u, `SELECT state_json FROM credential_policies WHERE namespace_id=? AND credential_id=?`, ns.String(), id.String())
	if err == nil {
		err = p.Validate()
	}
	return p, err
}
func (r signingIdentityRepository) SaveCredentialPolicy(ctx context.Context, p d.CredentialPolicy, expected d.Version) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if p.Version != expected+1 {
		return d.NewError(d.ErrorCodeVersionConflict, "policy version changed")
	}
	c, err := r.Credential(ctx, p.NamespaceID, p.CredentialID)
	if err != nil {
		return err
	}
	if c.PrincipalID != p.PrincipalID {
		return d.NewError(d.ErrorCodeInvalidScope, "policy Principal differs from credential")
	}
	raw, err := marshalJSON(p)
	if err != nil {
		return err
	}
	if expected == 0 {
		_, err = r.u.tx.ExecContext(ctx, `INSERT INTO credential_policies(namespace_id,credential_id,principal_id,version,state_json) VALUES(?,?,?,?,?)`, p.NamespaceID.String(), p.CredentialID.String(), p.PrincipalID, p.Version, raw)
		return mapSQLError("create credential policy", err)
	}
	result, err := r.u.tx.ExecContext(ctx, `UPDATE credential_policies SET version=?,state_json=? WHERE namespace_id=? AND credential_id=? AND principal_id=? AND version=?`, p.Version, raw, p.NamespaceID.String(), p.CredentialID.String(), p.PrincipalID, expected)
	return signingCAS(result, err)
}
func policyKeys(outcome, work *d.ID) (string, string) {
	o, w := "", ""
	if outcome != nil {
		o = outcome.String()
	}
	if work != nil {
		w = work.String()
	}
	return o, w
}
func (r signingIdentityRepository) AcceptancePolicy(ctx context.Context, ns d.ID, outcome, work *d.ID) (d.WorkAcceptancePolicy, error) {
	o, w := policyKeys(outcome, work)
	p, err := signingRead[d.WorkAcceptancePolicy](ctx, r.u, `SELECT state_json FROM work_acceptance_policies WHERE namespace_id=? AND outcome_key=? AND work_key=?`, ns.String(), o, w)
	if err == nil {
		err = p.Validate()
	}
	return p, err
}
func (r signingIdentityRepository) SaveAcceptancePolicy(ctx context.Context, p d.WorkAcceptancePolicy, expected d.Version) error {
	if err := p.Validate(); err != nil {
		return err
	}
	if p.Version != expected+1 {
		return d.NewError(d.ErrorCodeVersionConflict, "acceptance policy version changed")
	}
	if p.OutcomeID != nil {
		if _, err := r.u.Outcomes().Get(ctx, p.NamespaceID, *p.OutcomeID); err != nil {
			return err
		}
		if p.WorkItemID != nil {
			if _, err := r.u.WorkItems().Get(ctx, d.Scope{NamespaceID: p.NamespaceID, OutcomeID: *p.OutcomeID}, *p.WorkItemID); err != nil {
				return err
			}
		}
	}
	o, w := policyKeys(p.OutcomeID, p.WorkItemID)
	raw, err := marshalJSON(p)
	if err != nil {
		return err
	}
	if expected == 0 {
		_, err = r.u.tx.ExecContext(ctx, `INSERT INTO work_acceptance_policies(namespace_id,outcome_key,work_key,version,state_json) VALUES(?,?,?,?,?)`, p.NamespaceID.String(), o, w, p.Version, raw)
		return mapSQLError("create acceptance policy", err)
	}
	result, err := r.u.tx.ExecContext(ctx, `UPDATE work_acceptance_policies SET version=?,state_json=? WHERE namespace_id=? AND outcome_key=? AND work_key=? AND version=?`, p.Version, raw, p.NamespaceID.String(), o, w, expected)
	return signingCAS(result, err)
}
func (r signingIdentityRepository) PrincipalGroup(ctx context.Context, ns d.ID, principal string) (string, error) {
	var group string
	err := r.u.tx.QueryRowContext(ctx, `SELECT separation_group FROM principal_review_groups WHERE namespace_id=? AND principal_id=?`, ns.String(), principal).Scan(&group)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return group, err
}
func (r signingIdentityRepository) SetPrincipalGroup(ctx context.Context, ns d.ID, principal, group string) error {
	if len(group) > 256 || strings.TrimSpace(principal) == "" {
		return d.NewError(d.ErrorCodeInvalidArgument, "invalid Principal separation group")
	}
	var granted int
	if err := r.u.tx.QueryRowContext(ctx, `SELECT 1 FROM namespace_grants WHERE namespace_id=? AND principal_id=?`, ns.String(), principal).Scan(&granted); err != nil {
		return mapSQLError("review group Principal scope", err)
	}
	_, err := r.u.tx.ExecContext(ctx, `INSERT INTO principal_review_groups(namespace_id,principal_id,separation_group) VALUES(?,?,?) ON CONFLICT(namespace_id,principal_id) DO UPDATE SET separation_group=excluded.separation_group`, ns.String(), principal, group)
	return mapSQLError("set Principal review group", err)
}
func (r signingIdentityRepository) Receipt(ctx context.Context, ns d.ID, principal, key string) (string, json.RawMessage, error) {
	var fingerprint, payload string
	err := r.u.tx.QueryRowContext(ctx, `SELECT fingerprint,result_json FROM signing_security_results WHERE namespace_id=? AND principal_id=? AND idempotency_key=?`, ns.String(), principal, key).Scan(&fingerprint, &payload)
	return fingerprint, json.RawMessage(payload), mapSQLError("read signing security receipt", err)
}
func (r signingIdentityRepository) SaveReceipt(ctx context.Context, ns d.ID, principal, key, fingerprint string, payload json.RawMessage) error {
	_, err := r.u.tx.ExecContext(ctx, `INSERT INTO signing_security_results(namespace_id,principal_id,idempotency_key,fingerprint,result_json) VALUES(?,?,?,?,?)`, ns.String(), principal, key, fingerprint, string(payload))
	return mapSQLError("save signing security receipt", err)
}
func (r signingIdentityRepository) Audit(ctx context.Context, ns d.ID, principal string, actor d.ActorRef, op, target string, now time.Time) (d.Version, error) {
	version, err := r.LockNamespace(ctx, ns)
	if err != nil {
		return 0, err
	}
	next, err := version.Next()
	if err != nil {
		return 0, err
	}
	result, err := r.u.tx.ExecContext(ctx, `UPDATE namespaces SET version=?,updated_at=? WHERE id=? AND version=?`, next, encodeTime(now), ns.String(), version)
	if err = signingCAS(result, err); err != nil {
		return 0, err
	}
	a, err := marshalJSON(actor)
	if err != nil {
		return 0, err
	}
	_, err = r.u.tx.ExecContext(ctx, `INSERT INTO security_audit(namespace_id,namespace_version,principal_id,actor_json,operation,target,recorded_at) VALUES(?,?,?,?,?,?,?)`, ns.String(), next, principal, a, op, target, encodeTime(now))
	return next, mapSQLError("record signing security audit", err)
}
func signingCAS(result sql.Result, err error) error {
	if err != nil {
		return mapSQLError("write signing state", err)
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return d.NewError(d.ErrorCodeVersionConflict, "signing state version changed")
	}
	return nil
}

var _ ports.SigningIdentityUnitOfWork = (*unitOfWork)(nil)

func (s *Store) CredentialPolicy(ctx context.Context, ns, id d.ID) (d.CredentialPolicy, error) {
	var p d.CredentialPolicy
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT state_json FROM credential_policies WHERE namespace_id=? AND credential_id=?`, ns.String(), id.String()).Scan(&raw)
	if err != nil {
		return p, mapSQLError("read credential policy", err)
	}
	err = json.Unmarshal([]byte(raw), &p)
	if err == nil {
		err = p.Validate()
	}
	return p, err
}
