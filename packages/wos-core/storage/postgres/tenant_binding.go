package postgres

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func (s *Store) BoundTenantID() domain.ID {
	if s == nil {
		return ""
	}
	return s.tenantID
}

var _ ports.TenantBinding = (*Store)(nil)

// Runtime startup verifies the initializer's binding without creating tables,
// constraints or metadata. Delivery-only workers never initialize a Tenant.
func (s *Store) verifyTenantBinding(ctx context.Context, id domain.ID) error {
	var count, mismatched int
	if err := s.db.QueryRowContext(ctx, `SELECT count(*),count(*) FILTER (WHERE id<>$1) FROM tenants`, id.String()).Scan(&count, &mismatched); err != nil {
		return mapSQLError("verify Tenant schema binding", err)
	}
	if count != 1 || mismatched != 0 {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "Tenant schema has not been initialized for this identity")
	}
	var constraint string
	if err := s.db.QueryRowContext(ctx, `SELECT pg_get_constraintdef(c.oid) FROM pg_constraint c JOIN pg_class t ON t.oid=c.conrelid JOIN pg_namespace n ON n.oid=t.relnamespace WHERE n.nspname=$1 AND t.relname='tenants' AND c.conname='wos_tenant_identity_binding' AND c.convalidated`, s.path).Scan(&constraint); err != nil {
		return mapSQLError("verify Tenant identity constraint", err)
	}
	if constraint != "CHECK ((id = '"+id.String()+"'::text))" {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "schema is not permanently bound to this Tenant identity")
	}
	var singleton bool
	if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_index i JOIN pg_class t ON t.oid=i.indrelid JOIN pg_class idx ON idx.oid=i.indexrelid JOIN pg_namespace n ON n.oid=t.relnamespace WHERE n.nspname=$1 AND t.relname='tenants' AND idx.relname='wos_single_tenant' AND i.indisunique AND i.indisvalid AND i.indpred IS NULL AND i.indnkeyatts=1 AND pg_get_expr(i.indexprs,i.indrelid)='1')`, s.path).Scan(&singleton); err != nil {
		return mapSQLError("verify singleton Tenant constraint", err)
	}
	if !singleton {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "schema lacks its singleton Tenant constraint")
	}
	return nil
}

// bindTenant is a trusted composition operation, run before exposing any Store.
// Persistent constraints protect every SQL path, including administrative and
// integration methods that do not enter an application UnitOfWork.
func (s *Store) bindTenant(ctx context.Context, id domain.ID, name string) error {
	if strings.TrimSpace(name) == "" {
		name = id.String()
	}
	if len(name) > 200 {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "Tenant name exceeds 200 bytes")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return mapSQLError("begin Tenant schema binding", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, tenantLifecycleKey(s.path, "schema-binding")); err != nil {
		return mapSQLError("lock Tenant schema binding", err)
	}
	var count, mismatched int
	if err := tx.QueryRowContext(ctx, `SELECT count(*),count(*) FILTER (WHERE id<>$1) FROM tenants`, id.String()).Scan(&count, &mismatched); err != nil {
		return mapSQLError("inspect Tenant schema binding", err)
	}
	if count > 1 || mismatched != 0 {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "schema contains a different Tenant or multiple legacy Tenants; an explicit data migration is required")
	}
	var constraint string
	err = tx.QueryRowContext(ctx, `SELECT pg_get_constraintdef(c.oid) FROM pg_constraint c JOIN pg_class t ON t.oid=c.conrelid JOIN pg_namespace n ON n.oid=t.relnamespace WHERE n.nspname=$1 AND t.relname='tenants' AND c.conname='wos_tenant_identity_binding'`, s.path).Scan(&constraint)
	if err != nil && err != sql.ErrNoRows {
		return mapSQLError("inspect Tenant identity constraint", err)
	}
	if err == nil && constraint != "CHECK ((id = '"+id.String()+"'::text))" {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "schema is permanently bound to another Tenant identity")
	}
	if err == sql.ErrNoRows {
		// UUID validation happened before Open; no user identifier or arbitrary
		// expression is interpolated into this persistent database constraint.
		if _, err := tx.ExecContext(ctx, `ALTER TABLE tenants ADD CONSTRAINT wos_tenant_identity_binding CHECK (id='`+id.String()+`')`); err != nil {
			return mapSQLError("bind Tenant identity", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS wos_single_tenant ON tenants ((1))`); err != nil {
		return mapSQLError("enforce one Tenant per schema", err)
	}
	if count == 0 {
		now := time.Now().UTC().UnixMicro()
		if _, err := tx.ExecContext(ctx, `INSERT INTO tenants(id,name,lifecycle,version,created_at,updated_at) VALUES($1,$2,'active',1,$3,$3)`, id.String(), name, now); err != nil {
			return mapSQLError("initialize bound Tenant", err)
		}
	}
	return mapSQLError("commit Tenant schema binding", tx.Commit())
}
