package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
)

// TenantCatalog is a trusted host administration API. It does not authenticate
// remote requests or grant access to a Tenant's operational state.
type TenantCatalog struct {
	db     *sql.DB
	leases *sql.DB
	schema string
}

type CatalogOptions struct {
	Schema             string
	MaxOpenConnections int
	MaxLeases          int
	MigrateOnOpen      bool
}

type TenantState string

const (
	TenantProvisioning TenantState = "provisioning"
	TenantActive       TenantState = "active"
	TenantSuspended    TenantState = "suspended"
	TenantDeleted      TenantState = "deleted"
)

type TenantPlacement struct {
	TenantID         domain.ID      `json:"tenant_id"`
	Schema           string         `json:"schema"`
	Name             string         `json:"name"`
	State            TenantState    `json:"state"`
	Version          domain.Version `json:"version"`
	CreatedAt        time.Time      `json:"created_at"`
	UpdatedAt        time.Time      `json:"updated_at"`
	initializerLease *TenantLease
}

// InitializerOptions is valid only on the placement passed to a provisioning
// callback. The initializer must use these options and close its Store before
// returning. Copying only Schema/TenantID drops the provisioning data fence.
func (p TenantPlacement) InitializerOptions() Options {
	lease := p.initializerLease
	if lease == nil {
		// An arbitrary placement read from the catalog cannot authorize an
		// initializer or bypass the ordinary active runtime lease contract.
		lease = &TenantLease{closed: true}
	}
	return Options{Schema: p.Schema, TenantID: p.TenantID, TenantName: p.Name, TenantLease: lease, MigrateOnOpen: true}
}

type TenantProvision struct {
	TenantID domain.ID `json:"tenant_id"`
	Name     string    `json:"name"`
	Reason   string    `json:"reason"`
}

var tenantCatalogSchemaPattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

func validTenantCatalogSchema(schema string) bool {
	return tenantCatalogSchemaPattern.MatchString(schema) && schema != "public" && !strings.HasPrefix(schema, "pg_") && !strings.HasPrefix(schema, "wos_t_")
}

func OpenTenantCatalog(ctx context.Context, dsn string, options CatalogOptions) (*TenantCatalog, error) {
	if options.Schema == "" {
		options.Schema = "wos_control"
	}
	if !validTenantCatalogSchema(options.Schema) {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "invalid Tenant catalog schema")
	}
	if options.MaxOpenConnections == 0 {
		options.MaxOpenConnections = 4
	}
	if options.MaxLeases == 0 {
		options.MaxLeases = 32
	}
	if options.MaxOpenConnections < 1 || options.MaxOpenConnections > 64 || options.MaxLeases < 1 || options.MaxLeases > 4096 {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "invalid Tenant catalog connection budget")
	}
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "invalid PostgreSQL configuration")
	}
	// The fixed catalog has no public/user-schema fallback. PostgreSQL implicitly
	// searches pg_catalog first; unqualified application tables remain local.
	cfg.RuntimeParams["search_path"] = options.Schema
	cfg.RuntimeParams["statement_timeout"] = "30000"
	cfg.RuntimeParams["lock_timeout"] = "5000"
	cfg.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	c := &TenantCatalog{db: stdlib.OpenDB(*cfg), leases: stdlib.OpenDB(*cfg), schema: options.Schema}
	c.db.SetMaxOpenConns(options.MaxOpenConnections)
	c.db.SetMaxIdleConns(options.MaxOpenConnections)
	c.db.SetConnMaxLifetime(30 * time.Minute)
	c.leases.SetMaxOpenConns(options.MaxLeases)
	// An idle dedicated lease session must not retain session-level locks.
	c.leases.SetMaxIdleConns(0)
	if err := c.db.PingContext(ctx); err != nil {
		_ = c.Close()
		return nil, fmt.Errorf("connect Tenant catalog: %w", err)
	}
	if options.MigrateOnOpen {
		err = c.Migrate(ctx)
	} else {
		err = c.checkVersion(ctx)
	}
	if err != nil {
		_ = c.Close()
		return nil, err
	}
	return c, nil
}

func (c *TenantCatalog) Close() error {
	if c == nil {
		return nil
	}
	leaseErr := c.leases.Close()
	if err := c.db.Close(); err != nil {
		return err
	}
	return leaseErr
}

func (c *TenantCatalog) Migrate(ctx context.Context) error {
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return mapSQLError("begin Tenant catalog migration", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, tenantLifecycleKey(c.schema, "catalog-migration")); err != nil {
		return mapSQLError("lock Tenant catalog migration", err)
	}
	if _, err := tx.ExecContext(ctx, `CREATE SCHEMA IF NOT EXISTS `+c.schema); err != nil {
		return mapSQLError("create Tenant catalog schema", err)
	}
	if _, err := tx.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS tenant_catalog_version (
		singleton INTEGER PRIMARY KEY CHECK (singleton=1), version BIGINT NOT NULL
	)`); err != nil {
		return mapSQLError("create Tenant catalog version", err)
	}
	var version int64
	err = tx.QueryRowContext(ctx, `SELECT version FROM tenant_catalog_version WHERE singleton=1`).Scan(&version)
	if err == nil && version != 1 {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "unsupported Tenant catalog schema version")
	}
	if err != nil && err != sql.ErrNoRows {
		return mapSQLError("read Tenant catalog version", err)
	}
	if _, err := tx.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS tenant_placements (
	tenant_id TEXT PRIMARY KEY, schema_name TEXT NOT NULL UNIQUE, name TEXT NOT NULL,
	state TEXT NOT NULL CHECK (state IN ('provisioning','active','suspended','deleted')),
	version BIGINT NOT NULL CHECK (version>0), created_at TIMESTAMPTZ NOT NULL, updated_at TIMESTAMPTZ NOT NULL
);
CREATE TABLE IF NOT EXISTS tenant_placement_audit (
	sequence BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
	tenant_id TEXT NOT NULL REFERENCES tenant_placements(tenant_id),
	version BIGINT NOT NULL, operation TEXT NOT NULL, state TEXT NOT NULL,
	reason TEXT NOT NULL CHECK (length(reason)>0), recorded_at TIMESTAMPTZ NOT NULL,
	UNIQUE(tenant_id,version)
);
INSERT INTO tenant_catalog_version(singleton,version) VALUES(1,1) ON CONFLICT(singleton) DO NOTHING;
`); err != nil {
		return mapSQLError("initialize Tenant catalog", err)
	}
	return mapSQLError("commit Tenant catalog migration", tx.Commit())
}

func (c *TenantCatalog) checkVersion(ctx context.Context) error {
	var version int64
	if err := c.db.QueryRowContext(ctx, `SELECT version FROM tenant_catalog_version WHERE singleton=1`).Scan(&version); err != nil {
		return mapSQLError("read Tenant catalog version", err)
	}
	if version != 1 {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "unsupported Tenant catalog schema version")
	}
	return nil
}

func tenantLifecycleKey(schema, id string) int64 {
	digest := sha256.Sum256([]byte("wos:tenant-lifecycle:" + schema + ":" + id))
	return int64(binary.BigEndian.Uint64(digest[:8]))
}

func tenantInitializationKey(schema string, id domain.ID) int64 {
	return tenantLifecycleKey(schema, id.String()+":initialization-data")
}

func tenantDataSchema(id domain.ID) string {
	return "wos_t_" + strings.ReplaceAll(id.String(), "-", "")
}

func validateTenantMutation(id domain.ID, expected domain.Version, reason string) error {
	if err := id.Validate(); err != nil {
		return err
	}
	if expected < 1 || expected >= math.MaxInt64 || strings.TrimSpace(reason) == "" || len(reason) > 4096 {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "Tenant operation requires a valid expected version and nonempty reason (at most 4096 bytes)")
	}
	return nil
}

type tenantPlacementScanner interface{ Scan(...any) error }

func scanTenantPlacement(row tenantPlacementScanner) (TenantPlacement, error) {
	var p TenantPlacement
	var id string
	if err := row.Scan(&id, &p.Schema, &p.Name, &p.State, &p.Version, &p.CreatedAt, &p.UpdatedAt); err != nil {
		return p, mapSQLError("read Tenant placement", err)
	}
	var err error
	p.TenantID, err = domain.ParseID(id)
	if err != nil || p.Schema != tenantDataSchema(p.TenantID) {
		return TenantPlacement{}, domain.NewError(domain.ErrorCodeInvalidConfig, "invalid persisted Tenant placement")
	}
	return p, nil
}

const tenantPlacementColumns = `tenant_id,schema_name,name,state,version,created_at,updated_at`

func (c *TenantCatalog) GetTenant(ctx context.Context, id domain.ID) (TenantPlacement, error) {
	if err := id.Validate(); err != nil {
		return TenantPlacement{}, err
	}
	return scanTenantPlacement(c.db.QueryRowContext(ctx, `SELECT `+tenantPlacementColumns+` FROM tenant_placements WHERE tenant_id=$1`, id.String()))
}

// ListTenants is bounded and includes tombstones so callers cannot confuse a
// permanently deleted identity with an identity available for provisioning.
func (c *TenantCatalog) ListTenants(ctx context.Context, afterID domain.ID, limit int) ([]TenantPlacement, error) {
	if !afterID.IsZero() {
		if err := afterID.Validate(); err != nil {
			return nil, err
		}
	}
	if limit < 1 || limit > 1000 {
		return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "Tenant page limit must be between 1 and 1000")
	}
	rows, err := c.db.QueryContext(ctx, `SELECT `+tenantPlacementColumns+` FROM tenant_placements WHERE tenant_id>$1 ORDER BY tenant_id LIMIT $2`, afterID.String(), limit)
	if err != nil {
		return nil, mapSQLError("list Tenant placements", err)
	}
	defer rows.Close()
	out := []TenantPlacement{}
	for rows.Next() {
		p, err := scanTenantPlacement(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, mapSQLError("list Tenant placements", rows.Err())
}

func (c *TenantCatalog) audit(ctx context.Context, tx *sql.Tx, p TenantPlacement, operation, reason string) error {
	_, err := tx.ExecContext(ctx, `INSERT INTO tenant_placement_audit(tenant_id,version,operation,state,reason,recorded_at) VALUES($1,$2,$3,$4,$5,$6)`, p.TenantID.String(), p.Version, operation, p.State, reason, p.UpdatedAt)
	return mapSQLError("record Tenant operation", err)
}

func (c *TenantCatalog) lockMutation(ctx context.Context, tx *sql.Tx, id domain.ID, expected domain.Version) (TenantPlacement, error) {
	var acquired bool
	if err := tx.QueryRowContext(ctx, `SELECT pg_try_advisory_xact_lock($1)`, tenantLifecycleKey(c.schema, id.String())).Scan(&acquired); err != nil {
		return TenantPlacement{}, mapSQLError("lock Tenant lifecycle", err)
	}
	if !acquired {
		return TenantPlacement{}, domain.NewError(domain.ErrorCodePreconditionFailed, "stop and drain all Tenant runtimes before changing its lifecycle")
	}
	if err := c.lockInitializerData(ctx, tx, id); err != nil {
		return TenantPlacement{}, err
	}
	p, err := scanTenantPlacement(tx.QueryRowContext(ctx, `SELECT `+tenantPlacementColumns+` FROM tenant_placements WHERE tenant_id=$1 FOR UPDATE`, id.String()))
	if err != nil {
		return p, err
	}
	if p.Version != expected {
		return p, domain.NewError(domain.ErrorCodeVersionConflict, "Tenant placement version changed")
	}
	return p, nil
}

func (c *TenantCatalog) lockInitializerData(ctx context.Context, tx *sql.Tx, id domain.ID) error {
	var acquired bool
	if err := tx.QueryRowContext(ctx, `SELECT pg_try_advisory_xact_lock($1)`, tenantInitializationKey(c.schema, id)).Scan(&acquired); err != nil {
		return mapSQLError("lock Tenant initialization data", err)
	}
	if !acquired {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "close all Tenant initializer database sessions before changing its lifecycle")
	}
	return nil
}

func (c *TenantCatalog) updateState(ctx context.Context, tx *sql.Tx, p TenantPlacement, state TenantState, operation, reason string) (TenantPlacement, error) {
	p.State, p.Version, p.UpdatedAt = state, p.Version+1, time.Now().UTC()
	if _, err := tx.ExecContext(ctx, `UPDATE tenant_placements SET state=$1,version=$2,updated_at=$3 WHERE tenant_id=$4`, p.State, p.Version, p.UpdatedAt, p.TenantID.String()); err != nil {
		return TenantPlacement{}, mapSQLError("change Tenant lifecycle", err)
	}
	return p, c.audit(ctx, tx, p, operation, reason)
}

// ChangeTenantState requires all replicas/workers to be stopped and their
// database sessions closed. It never terminates an in-flight request or lease.
func (c *TenantCatalog) ChangeTenantState(ctx context.Context, id domain.ID, expected domain.Version, state TenantState, reason string) (TenantPlacement, error) {
	if err := validateTenantMutation(id, expected, reason); err != nil {
		return TenantPlacement{}, err
	}
	if state != TenantActive && state != TenantSuspended {
		return TenantPlacement{}, domain.NewError(domain.ErrorCodeInvalidArgument, "only active and suspended lifecycle transitions are allowed")
	}
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return TenantPlacement{}, mapSQLError("begin Tenant lifecycle change", err)
	}
	defer tx.Rollback()
	p, err := c.lockMutation(ctx, tx, id, expected)
	if err != nil {
		return p, err
	}
	if p.State != TenantActive && p.State != TenantSuspended || p.State == state {
		return p, domain.NewError(domain.ErrorCodeInvalidTransition, "Tenant is not in the required previous lifecycle state")
	}
	p, err = c.updateState(ctx, tx, p, state, string(state), reason)
	if err != nil {
		return p, err
	}
	return p, mapSQLError("commit Tenant lifecycle change", tx.Commit())
}

// DeleteTenant removes the complete database schema and retains its identity,
// placement and audit tombstone. External artifacts and retained backups are
// operator-owned; this method does not claim to erase those copies.
func (c *TenantCatalog) DeleteTenant(ctx context.Context, id domain.ID, expected domain.Version, reason string) (TenantPlacement, error) {
	if err := validateTenantMutation(id, expected, reason); err != nil {
		return TenantPlacement{}, err
	}
	tx, err := c.db.BeginTx(ctx, nil)
	if err != nil {
		return TenantPlacement{}, mapSQLError("begin Tenant deletion", err)
	}
	defer tx.Rollback()
	p, err := c.lockMutation(ctx, tx, id, expected)
	if err != nil {
		return p, err
	}
	if p.State != TenantSuspended && p.State != TenantProvisioning {
		return p, domain.NewError(domain.ErrorCodeInvalidTransition, "only suspended or interrupted provisioning Tenants can be deleted")
	}
	// CASCADE must not remove objects belonging to another customer's schema.
	// WOS creates no cross-schema domain references; fail closed if a host has
	// introduced such a dependency outside the supported storage contract.
	var external bool
	err = tx.QueryRowContext(ctx, `SELECT EXISTS (
		SELECT 1 FROM pg_depend d
		CROSS JOIN LATERAL pg_identify_object(d.refclassid,d.refobjid,d.refobjsubid) source
		CROSS JOIN LATERAL pg_identify_object(d.classid,d.objid,d.objsubid) dependent
		LEFT JOIN pg_rewrite rw ON d.classid='pg_rewrite'::regclass AND rw.oid=d.objid
		LEFT JOIN pg_trigger tr ON d.classid='pg_trigger'::regclass AND tr.oid=d.objid
		LEFT JOIN pg_attrdef ad ON d.classid='pg_attrdef'::regclass AND ad.oid=d.objid
		LEFT JOIN pg_policy pol ON d.classid='pg_policy'::regclass AND pol.oid=d.objid
		LEFT JOIN pg_class owner ON owner.oid=COALESCE(rw.ev_class,tr.tgrelid,ad.adrelid,pol.polrelid)
		LEFT JOIN pg_namespace owner_schema ON owner_schema.oid=owner.relnamespace
		WHERE source.schema=$1 AND COALESCE(dependent.schema,owner_schema.nspname) IS NOT NULL
		AND COALESCE(dependent.schema,owner_schema.nspname)<>$1
		AND COALESCE(dependent.schema,owner_schema.nspname) NOT IN ('pg_catalog','pg_toast')
	)`, p.Schema).Scan(&external)
	if err != nil {
		return p, mapSQLError("verify Tenant deletion boundary", err)
	}
	if external {
		return p, domain.NewError(domain.ErrorCodePreconditionFailed, "external schema dependencies prevent Tenant deletion")
	}
	if _, err := tx.ExecContext(ctx, `DROP SCHEMA `+p.Schema+` CASCADE`); err != nil {
		return p, mapSQLError("delete Tenant schema", err)
	}
	p, err = c.updateState(ctx, tx, p, TenantDeleted, "deleted", reason)
	if err != nil {
		return p, err
	}
	return p, mapSQLError("commit Tenant deletion", tx.Commit())
}
