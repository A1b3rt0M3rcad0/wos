package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

var _ ports.TenantBinding = (*Store)(nil)

func (s *Store) BoundTenantID() domain.ID {
	if s == nil {
		return ""
	}
	return s.tenantID
}

const tenantInsertTriggerPrefix = "CREATE TRIGGER wos_tenant_identity_insert BEFORE INSERT ON tenants WHEN NEW.id IS NOT '"
const tenantTriggerSuffix = "' BEGIN SELECT RAISE(ABORT,'Tenant differs from SQLite file binding'); END"

func (s *Store) configureTenantBinding(ctx context.Context, id domain.ID, name string) error {
	if id.IsZero() {
		var definition string
		err := s.db.QueryRowContext(ctx, `SELECT sql FROM sqlite_schema WHERE type='trigger' AND name='wos_tenant_identity_insert'`).Scan(&definition)
		if err == sql.ErrNoRows {
			var partial int
			if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_schema WHERE type='trigger' AND name='wos_tenant_identity_update'`).Scan(&partial); err != nil {
				return mapSQLError("inspect SQLite Tenant binding", err)
			}
			if partial != 0 {
				return domain.NewError(domain.ErrorCodePreconditionFailed, "SQLite Tenant binding is incomplete")
			}
			return nil
		}
		if err != nil {
			return mapSQLError("inspect SQLite Tenant binding", err)
		}
		if !strings.HasPrefix(definition, tenantInsertTriggerPrefix) || !strings.HasSuffix(definition, tenantTriggerSuffix) {
			return domain.NewError(domain.ErrorCodePreconditionFailed, "SQLite Tenant binding definition differs")
		}
		persisted := strings.TrimSuffix(strings.TrimPrefix(definition, tenantInsertTriggerPrefix), tenantTriggerSuffix)
		id, err = domain.ParseID(persisted)
		if err != nil {
			return domain.WrapError(domain.ErrorCodePreconditionFailed, "invalid persisted SQLite Tenant binding", err)
		}
	}
	if err := s.bindTenant(ctx, id, name); err != nil {
		return err
	}
	s.tenantID = id
	return nil
}

// bindTenant runs before exposing the Store. SQLite's immediate transaction
// serializes competing bind attempts; durable constraints protect all repository
// paths, including direct administrative methods outside an application UoW.
func (s *Store) bindTenant(ctx context.Context, id domain.ID, name string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		name = id.String()
	}
	if len(name) > 200 {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "Tenant name exceeds 200 bytes")
	}
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return mapSQLError("begin SQLite Tenant binding", err)
	}
	defer tx.Rollback()
	// Binding can create constraints and seed Tenant identity. Inspect the
	// schema under the same writer lock before making either change, including
	// when the caller deliberately disabled migrations for a runtime replica.
	latest, err := LatestSchemaVersion()
	if err != nil {
		return domain.WrapError(domain.ErrorCodeInvalidConfig, "read supported SQLite schema version", err)
	}
	var version int64
	if err := tx.QueryRowContext(ctx, `SELECT COALESCE(MAX(version),0) FROM schema_migrations`).Scan(&version); err != nil {
		return domain.WrapError(domain.ErrorCodeInvalidConfig, "read SQLite schema before Tenant binding", err)
	}
	if version != latest {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "SQLite Tenant binding requires the current supported schema; migrate with a compatible release before starting this runtime")
	}
	var count, mismatched int
	if err := tx.QueryRowContext(ctx, `SELECT count(*),count(*) FILTER (WHERE id<>?) FROM tenants`, id.String()).Scan(&count, &mismatched); err != nil {
		return mapSQLError("inspect SQLite Tenant contents", err)
	}
	if count > 1 || mismatched != 0 {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "SQLite file contains a different Tenant or multiple legacy Tenants; an explicit data migration is required")
	}
	// Only a previously validated UUID enters these DDL literals. The permanent
	// guards survive removal of the Tenant row and add no application data table.
	triggers := []struct{ name, definition string }{
		{"wos_tenant_identity_insert", tenantInsertTriggerPrefix + id.String() + tenantTriggerSuffix},
		{"wos_tenant_identity_update", "CREATE TRIGGER wos_tenant_identity_update BEFORE UPDATE OF id ON tenants WHEN NEW.id IS NOT '" + id.String() + tenantTriggerSuffix},
	}
	for _, trigger := range triggers {
		var definition string
		err := tx.QueryRowContext(ctx, `SELECT sql FROM sqlite_schema WHERE type='trigger' AND name=?`, trigger.name).Scan(&definition)
		if err != nil && err != sql.ErrNoRows {
			return mapSQLError("inspect SQLite Tenant identity guard", err)
		}
		if err == nil && definition != trigger.definition {
			return domain.NewError(domain.ErrorCodePreconditionFailed, "SQLite file is permanently bound to another Tenant identity or has an invalid binding definition")
		}
		if err == sql.ErrNoRows {
			if _, err := tx.ExecContext(ctx, trigger.definition); err != nil {
				return mapSQLError("enforce SQLite Tenant identity", err)
			}
		}
	}
	if _, err := tx.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS wos_single_tenant ON tenants ((1))`); err != nil {
		return mapSQLError("enforce one Tenant per SQLite file", err)
	}
	if count == 0 {
		now := time.Now().UTC().UnixMicro()
		if _, err := tx.ExecContext(ctx, `INSERT INTO tenants(id,name,lifecycle,version,created_at,updated_at) VALUES(?,?,'active',1,?,?)`, id.String(), name, now, now); err != nil {
			return mapSQLError("initialize bound SQLite Tenant", err)
		}
	}
	return mapSQLError("commit SQLite Tenant binding", tx.Commit())
}
