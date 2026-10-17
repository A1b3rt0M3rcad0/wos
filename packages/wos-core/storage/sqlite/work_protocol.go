package sqlite

import (
	"context"
	"database/sql"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"time"
)

type workProtocolRepository struct{ uow *unitOfWork }

func (u *unitOfWork) WorkProtocol() ports.WorkProtocolRepository { return workProtocolRepository{u} }
func (r workProtocolRepository) Lock(ctx context.Context, ns d.ID, exclusive bool) (d.TenantWorkProtocol, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return d.TenantWorkProtocol{}, err
	}
	var err error
	if exclusive {
		var version int64
		err = r.uow.tx.QueryRowContext(ctx, `SELECT version FROM tenants WHERE id=?`, ns.String()).Scan(&version)
	} else {
		var lifecycle string
		err = r.uow.tx.QueryRowContext(ctx, `SELECT lifecycle FROM tenants WHERE id=?`, ns.String()).Scan(&lifecycle)
	}
	if err != nil && err != sql.ErrNoRows {
		return d.TenantWorkProtocol{}, mapSQLError("tenant protocol guard", err)
	}
	p := d.DefaultWorkProtocol(ns)
	var raw string
	err = r.uow.tx.QueryRowContext(ctx, `SELECT state_json FROM tenant_work_protocol WHERE tenant_id=?`, ns.String()).Scan(&raw)
	if err == sql.ErrNoRows {
		return p, nil
	}
	if err != nil {
		return p, mapSQLError("work protocol", err)
	}
	if err = unmarshalJSON(raw, &p); err != nil {
		return p, err
	}
	return p, p.Validate()
}
func (r workProtocolRepository) Scopes(ctx context.Context, ns d.ID) ([]d.Scope, error) {
	rows, err := r.uow.tx.QueryContext(ctx, `SELECT id FROM outcomes WHERE tenant_id=? ORDER BY id`, ns.String())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []d.Scope{}
	for rows.Next() {
		var raw string
		if err = rows.Scan(&raw); err != nil {
			return nil, err
		}
		id, err := d.ParseID(raw)
		if err != nil {
			return nil, err
		}
		out = append(out, d.Scope{TenantID: ns, OutcomeID: id})
	}
	return out, rows.Err()
}
func (r workProtocolRepository) ValidLegacyLeases(ctx context.Context, ns d.ID, now time.Time) (int, error) {
	var count int
	err := r.uow.tx.QueryRowContext(ctx, `SELECT count(*) FROM work_items WHERE tenant_id=? AND lease_claim_id IS NOT NULL AND lease_expires_at>?`, ns.String(), encodeTime(now)).Scan(&count)
	return count, err
}
func (r workProtocolRepository) ValidUnsignedContracts(ctx context.Context, ns d.ID, now time.Time) (int, error) {
	var count int
	err := r.uow.tx.QueryRowContext(ctx, `SELECT count(*) FROM work_contracts WHERE tenant_id=? AND status='active' AND expires_at>?`, ns.String(), encodeTime(now)).Scan(&count)
	return count, mapSQLError("count valid unsigned authority", err)
}
func (r workProtocolRepository) EnableContracts(ctx context.Context, ns d.ID, now time.Time) error {
	count, err := r.ValidLegacyLeases(ctx, ns, now)
	if err != nil {
		return err
	}
	if count > 0 {
		return d.NewError(d.ErrorCodePreconditionFailed, "valid legacy leases remain")
	}
	_, err = r.uow.tx.ExecContext(ctx, `UPDATE work_items SET contracts_enabled=1,lease_claim_id=NULL,lease_principal_id=NULL,lease_actor_json=NULL,lease_acquired_at=NULL,lease_expires_at=NULL,fencing_token_decimal=COALESCE(fencing_token_decimal,CAST(last_fencing_token AS TEXT)) WHERE tenant_id=?`, ns.String())
	return mapSQLError("enable work contracts", err)
}
func (r workProtocolRepository) Save(ctx context.Context, p d.TenantWorkProtocol, expected d.Version) error {
	if err := p.Validate(); err != nil {
		return err
	}
	old, err := r.Lock(ctx, p.TenantID, true)
	if err != nil {
		return err
	}
	if old.Version != expected || p.Version != expected+1 {
		return d.NewError(d.ErrorCodeVersionConflict, "protocol version conflict")
	}
	raw, err := marshalJSON(p)
	if err != nil {
		return err
	}
	_, err = r.uow.tx.ExecContext(ctx, `INSERT INTO tenant_work_protocol(tenant_id,version,phase,writer_epoch,state_json) VALUES(?,?,?,?,?) ON CONFLICT(tenant_id) DO UPDATE SET version=excluded.version,phase=excluded.phase,writer_epoch=excluded.writer_epoch,state_json=excluded.state_json`, p.TenantID.String(), int64(p.Version), string(p.Phase), p.WriterEpoch, raw)
	return mapSQLError("save work protocol", err)
}
