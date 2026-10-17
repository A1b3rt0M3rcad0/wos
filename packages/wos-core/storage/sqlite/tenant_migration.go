package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// Refuse an unsettled published database before applying intermediate
// migrations. The previous release must still be able to open it and settle
// its authority. Migration 33 repeats the check in its own DDL transaction.
func (s *Store) preflightTenantUpgrade(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return mapSQLError("begin tenant upgrade preflight", err)
	}
	defer tx.Rollback()
	var current int64
	if err = tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return mapSQLError("read tenant upgrade source version", err)
	}
	// Migration 21 introduced all signed authority tables. Older databases
	// receive their final preflight when migration 33 is reached.
	if current >= 21 && current < 33 {
		return preflightTenantIdentityCutover(ctx, tx)
	}
	return nil
}

// This hook runs before migration 33 renames any existing table or column.
// Protocol-v2 authority must be settled using a compatible old client/server;
// changing an in-flight signed intent would invalidate its exact binding.
func preflightTenantIdentityCutover(ctx context.Context, tx *sql.Tx) error {
	var unfinished int
	err := tx.QueryRowContext(ctx, `SELECT
 (SELECT COUNT(*) FROM work_contracts WHERE status='active') +
 (SELECT COUNT(*) FROM work_items WHERE lease_claim_id IS NOT NULL AND lease_expires_at>?) +
 (SELECT COUNT(*) FROM signed_work_contracts WHERE status='active') +
 (SELECT COUNT(*) FROM work_review_contracts WHERE status='active') +
 (SELECT COUNT(*) FROM work_review_cases WHERE status IN ('pending','in_review'))`, time.Now().UTC().UnixMicro()).Scan(&unfinished)
	if err != nil {
		return mapSQLError("inspect tenant protocol cutover", err)
	}
	if unfinished != 0 {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "settle or revoke active legacy contracts, live leases and open reviews with the previous release before migrating to Tenant protocol v3")
	}
	return nil
}

// Only mutable current configuration is rewritten. Immutable receipts,
// signatures, event payloads, operation results and contract specifications
// retain their original bytes and are decoded through historical readers.
func migrateTenantProtocolMetadata(ctx context.Context, tx *sql.Tx) error {
	rows, err := tx.QueryContext(ctx, `SELECT tenant_id,state_json FROM tenant_work_protocol ORDER BY tenant_id`)
	if err != nil {
		return mapSQLError("read tenant protocol cutover", err)
	}
	type update struct{ id, raw string }
	var updates []update
	for rows.Next() {
		var v update
		if err = rows.Scan(&v.id, &v.raw); err != nil {
			rows.Close()
			return err
		}
		updates = append(updates, v)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, v := range updates {
		var p domain.TenantWorkProtocol
		if err = unmarshalJSON(v.raw, &p); err != nil {
			return err
		}
		if p.TenantID.String() != v.id {
			return domain.NewError(domain.ErrorCodePreconditionFailed, "stored protocol Tenant identity does not match its row")
		}
		switch string(p.Phase) {
		case "signed_contracts_v2":
			p.Phase, p.WriterEpoch = domain.WorkProtocolSigned, 3
			p.Version++
		case "draining_to_signed_v2":
			p.Phase = domain.WorkProtocolSignedDraining
			p.Version++
		}
		if err = p.Validate(); err != nil {
			return err
		}
		raw, err := marshalJSON(p)
		if err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE tenant_work_protocol SET version=?,phase=?,writer_epoch=?,state_json=? WHERE tenant_id=?`, uint64(p.Version), string(p.Phase), p.WriterEpoch, raw, v.id); err != nil {
			return mapSQLError("migrate tenant protocol metadata", err)
		}
	}
	rows, err = tx.QueryContext(ctx, `SELECT tenant_id,principal_id,permissions_json FROM tenant_grants`)
	if err != nil {
		return err
	}
	type grant struct{ tenant, principal, raw string }
	var grants []grant
	for rows.Next() {
		var g grant
		if err = rows.Scan(&g.tenant, &g.principal, &g.raw); err != nil {
			rows.Close()
			return err
		}
		grants = append(grants, g)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, g := range grants {
		var ps []string
		if err = json.Unmarshal([]byte(g.raw), &ps); err != nil {
			return err
		}
		changed := false
		for i := range ps {
			if ps[i] == "namespace:admin" {
				ps[i], changed = "tenant:admin", true
			}
		}
		if changed {
			raw, err := json.Marshal(ps)
			if err != nil {
				return err
			}
			if _, err = tx.ExecContext(ctx, `UPDATE tenant_grants SET permissions_json=? WHERE tenant_id=? AND principal_id=?`, string(raw), g.tenant, g.principal); err != nil {
				return mapSQLError("migrate tenant administrator permission", err)
			}
		}
	}
	return migrateTenantCredentialPolicies(ctx, tx)
}

func migrateTenantCredentialPolicies(ctx context.Context, tx *sql.Tx) error {
	// Drain each bounded page before writing on the same connection. Only live
	// policy configuration changes; historical authority/policy snapshots retain
	// their original permission spelling and signed digests.
	afterTenant, afterCredential := "", ""
	for {
		rows, err := tx.QueryContext(ctx, `SELECT tenant_id,credential_id,state_json FROM credential_policies WHERE (tenant_id,credential_id)>(?,?) ORDER BY tenant_id,credential_id LIMIT 256`, afterTenant, afterCredential)
		if err != nil {
			return mapSQLError("read tenant credential policy migration", err)
		}
		type entry struct{ tenant, credential, raw string }
		var page []entry
		for rows.Next() {
			var p entry
			if err = rows.Scan(&p.tenant, &p.credential, &p.raw); err != nil {
				rows.Close()
				return err
			}
			page = append(page, p)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return err
		}
		if len(page) == 0 {
			return nil
		}
		for _, row := range page {
			var policy domain.CredentialPolicy
			if err = unmarshalJSON(row.raw, &policy); err != nil {
				return err
			}
			if policy.TenantID.String() != row.tenant || policy.CredentialID.String() != row.credential {
				return domain.NewError(domain.ErrorCodePreconditionFailed, "stored credential policy identity differs from its row")
			}
			changed := false
			for i, permission := range policy.PermittedOperations {
				if permission == "namespace:admin" {
					policy.PermittedOperations[i], changed = "tenant:admin", true
				}
			}
			if changed {
				policy.Version++
				if err = policy.Validate(); err != nil {
					return err
				}
				raw, err := marshalJSON(policy)
				if err != nil {
					return err
				}
				if _, err = tx.ExecContext(ctx, `UPDATE credential_policies SET version=?,state_json=? WHERE tenant_id=? AND credential_id=?`, uint64(policy.Version), raw, row.tenant, row.credential); err != nil {
					return mapSQLError("migrate tenant credential policy", err)
				}
			}
			afterTenant, afterCredential = row.tenant, row.credential
		}
	}
}
