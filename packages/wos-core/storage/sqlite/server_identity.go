package sqlite

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sort"
)

type serverIdentityRepository struct{ u *unitOfWork }

func (u *unitOfWork) ServerIdentity() ports.ServerIdentityRepository {
	return serverIdentityRepository{u}
}
func (r serverIdentityRepository) Server(ctx context.Context) (d.ServerIdentity, error) {
	s, err := signingRead[d.ServerIdentity](ctx, r.u, `SELECT state_json FROM server_protocol_identity WHERE singleton=1`)
	if err == nil {
		err = s.Validate()
	}
	return s, err
}
func (r serverIdentityRepository) InsertServer(ctx context.Context, s d.ServerIdentity) error {
	if err := s.Validate(); err != nil {
		return err
	}
	raw, err := marshalJSON(s)
	if err != nil {
		return err
	}
	_, err = r.u.tx.ExecContext(ctx, `INSERT INTO server_protocol_identity(singleton,state_json) VALUES(1,?)`, raw)
	return mapSQLError("insert immutable server identity", err)
}

func (r serverIdentityRepository) LockServerIssuer(ctx context.Context) (d.ServerIdentity, error) {
	value, e := signingRead[d.ServerIdentity](ctx, r.u, `SELECT state_json FROM server_protocol_identity WHERE singleton=1 /* issuer recovery */`)
	if e == nil {
		e = value.Validate()
	}
	return value, e
}
func (r serverIdentityRepository) IssuerRecovery(ctx context.Context, id d.ID) (d.IssuerRecovery, error) {
	value, e := signingRead[d.IssuerRecovery](ctx, r.u, `SELECT state_json FROM server_issuer_recoveries WHERE replacement_key_id=?`, id)
	if e == nil {
		e = value.Validate()
	}
	return value, e
}
func (r serverIdentityRepository) ReplaceServerIssuer(ctx context.Context, record d.IssuerRecovery) error {
	if e := record.Validate(); e != nil {
		return e
	}
	var observed string
	if e := r.u.tx.QueryRowContext(ctx, `SELECT state_json FROM server_protocol_identity WHERE singleton=1`).Scan(&observed); e != nil {
		return mapSQLError("read issuer CAS", e)
	}
	var current d.ServerIdentity
	if e := unmarshalJSON(observed, &current); e != nil {
		return e
	}
	if current != record.Previous {
		return d.NewError(d.ErrorCodeVersionConflict, "persistent issuer changed")
	}
	for _, identity := range []d.ServerIdentity{record.Previous, record.Replacement} {
		raw, e := marshalJSON(identity)
		if e != nil {
			return e
		}
		_, e = r.u.tx.ExecContext(ctx, `INSERT OR IGNORE INTO server_issuer_public_keys(key_id,server_id,fingerprint,state_json) VALUES(?,?,?,?)`, identity.IssuerKeyID, identity.ID, identity.Fingerprint, raw)
		if e != nil {
			return mapSQLError("retain public issuer", e)
		}
		saved, e := signingRead[d.ServerIdentity](ctx, r.u, `SELECT state_json FROM server_issuer_public_keys WHERE key_id=?`, identity.IssuerKeyID)
		if e != nil {
			return e
		}
		if saved != identity {
			return d.NewError(d.ErrorCodeAlreadyExists, "issuer identity already used")
		}
	}
	raw, e := marshalJSON(record)
	if e != nil {
		return e
	}
	if _, e = r.u.tx.ExecContext(ctx, `INSERT INTO server_issuer_recoveries(replacement_key_id,intent_digest,state_json) VALUES(?,?,?)`, record.Replacement.IssuerKeyID, record.IntentDigest, raw); e != nil {
		return mapSQLError("record issuer recovery", e)
	}
	replacement, e := marshalJSON(record.Replacement)
	if e != nil {
		return e
	}
	result, e := r.u.tx.ExecContext(ctx, `UPDATE server_protocol_identity SET state_json=? WHERE singleton=1 AND state_json=?`, replacement, observed)
	if e != nil {
		return mapSQLError("replace issuer", e)
	}
	count, e := result.RowsAffected()
	if e != nil {
		return e
	}
	if count != 1 {
		return d.NewError(d.ErrorCodeVersionConflict, "persistent issuer changed")
	}
	return nil
}
func (r serverIdentityRepository) IssuerHistory(ctx context.Context, after d.ID, limit int) ([]d.ServerIdentity, error) {
	if limit < 1 || limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "issuer history limit must be 1..101")
	}
	rows, e := r.u.tx.QueryContext(ctx, `SELECT state_json FROM server_issuer_public_keys WHERE key_id>? ORDER BY key_id LIMIT ?`, after.String(), limit)
	if e != nil {
		return nil, mapSQLError("read public issuer history", e)
	}
	defer rows.Close()
	result := []d.ServerIdentity{}
	for rows.Next() {
		var raw string
		if e = rows.Scan(&raw); e != nil {
			return nil, e
		}
		var identity d.ServerIdentity
		if e = unmarshalJSON(raw, &identity); e != nil {
			return nil, e
		}
		if e = identity.Validate(); e != nil {
			return nil, e
		}
		result = append(result, identity)
	}
	if e = rows.Err(); e != nil {
		return nil, e
	}
	rows.Close()
	current, e := r.Server(ctx)
	if e != nil {
		return nil, e
	}
	seen := false
	for _, identity := range result {
		if identity.IssuerKeyID == current.IssuerKeyID {
			seen = true
		}
	}
	if !seen && current.IssuerKeyID > after {
		result = append(result, current)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].IssuerKeyID < result[j].IssuerKeyID })
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}
