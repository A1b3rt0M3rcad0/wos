package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

// ProvisionTenant records an unrouteable placement before initialization. A
// failed or interrupted initializer leaves an explicit provisioning record for
// CAS-controlled retry or deletion; it never exposes partially prepared data.
func (c *TenantCatalog) ProvisionTenant(ctx context.Context, request TenantProvision, initialize func(context.Context, TenantPlacement) error) (TenantPlacement, error) {
	if err := validateTenantMutation(request.TenantID, 1, request.Reason); err != nil {
		return TenantPlacement{}, err
	}
	if strings.TrimSpace(request.Name) == "" || len(request.Name) > 200 || initialize == nil {
		return TenantPlacement{}, domain.NewError(domain.ErrorCodeInvalidArgument, "Tenant provisioning requires a name (at most 200 bytes) and initializer")
	}
	conn, err := c.exclusiveSession(ctx, request.TenantID)
	if err != nil {
		return TenantPlacement{}, err
	}
	defer conn.Close() // The lease pool has no idle sessions; Close releases locks.
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return TenantPlacement{}, mapSQLError("begin Tenant provisioning", err)
	}
	defer tx.Rollback()
	if err := c.lockInitializerData(ctx, tx, request.TenantID); err != nil {
		return TenantPlacement{}, err
	}
	var known bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM tenant_placements WHERE tenant_id=$1)`, request.TenantID.String()).Scan(&known); err != nil {
		return TenantPlacement{}, mapSQLError("check Tenant identity", err)
	}
	if known {
		return TenantPlacement{}, domain.NewError(domain.ErrorCodeAlreadyExists, "Tenant identity is already reserved, including deleted identities")
	}
	now := time.Now().UTC()
	p := TenantPlacement{TenantID: request.TenantID, Schema: tenantDataSchema(request.TenantID), Name: request.Name, State: TenantProvisioning, Version: 1, CreatedAt: now, UpdatedAt: now}
	// Do not adopt a schema that was created independently or by another catalog.
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=$1)`, p.Schema).Scan(&known); err != nil {
		return p, mapSQLError("check Tenant schema ownership", err)
	}
	if known {
		return p, domain.NewError(domain.ErrorCodeAlreadyExists, "Tenant data schema already exists outside this placement")
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO tenant_placements(`+tenantPlacementColumns+`) VALUES($1,$2,$3,$4,$5,$6,$7)`, p.TenantID.String(), p.Schema, p.Name, p.State, p.Version, p.CreatedAt, p.UpdatedAt); err != nil {
		return p, mapSQLError("reserve Tenant placement", err)
	}
	if _, err := tx.ExecContext(ctx, `CREATE SCHEMA `+p.Schema); err != nil {
		return p, mapSQLError("create Tenant data schema", err)
	}
	if err := c.audit(ctx, tx, p, "provisioning", request.Reason); err != nil {
		return p, err
	}
	if err := tx.Commit(); err != nil {
		return p, mapSQLError("commit Tenant reservation", err)
	}
	return c.finishProvision(ctx, conn, p, request.Reason, initialize)
}

func (c *TenantCatalog) RetryProvisionTenant(ctx context.Context, id domain.ID, expected domain.Version, reason string, initialize func(context.Context, TenantPlacement) error) (TenantPlacement, error) {
	if err := validateTenantMutation(id, expected, reason); err != nil {
		return TenantPlacement{}, err
	}
	if initialize == nil {
		return TenantPlacement{}, domain.NewError(domain.ErrorCodeInvalidArgument, "Tenant provisioning requires an initializer")
	}
	conn, err := c.exclusiveSession(ctx, id)
	if err != nil {
		return TenantPlacement{}, err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return TenantPlacement{}, mapSQLError("begin Tenant provisioning retry", err)
	}
	defer tx.Rollback()
	p, err := c.lockMutation(ctx, tx, id, expected)
	if err != nil {
		return p, err
	}
	if p.State != TenantProvisioning {
		return p, domain.NewError(domain.ErrorCodeInvalidTransition, "only incomplete Tenant provisioning can be retried")
	}
	p, err = c.updateState(ctx, tx, p, TenantProvisioning, "provisioning_retry", reason)
	if err != nil {
		return p, err
	}
	if err := tx.Commit(); err != nil {
		return p, mapSQLError("commit Tenant provisioning retry", err)
	}
	return c.finishProvision(ctx, conn, p, reason, initialize)
}

func (c *TenantCatalog) exclusiveSession(ctx context.Context, id domain.ID) (*sql.Conn, error) {
	conn, err := c.leases.Conn(ctx)
	if err != nil {
		return nil, mapSQLError("open Tenant administrative session", err)
	}
	var acquired bool
	if err := conn.QueryRowContext(ctx, `SELECT pg_try_advisory_lock($1)`, tenantLifecycleKey(c.schema, id.String())).Scan(&acquired); err != nil {
		_ = conn.Close()
		return nil, mapSQLError("lock Tenant provisioning", err)
	}
	if !acquired {
		_ = conn.Close()
		return nil, domain.NewError(domain.ErrorCodePreconditionFailed, "another Tenant operation or runtime is still active")
	}
	return conn, nil
}

func (c *TenantCatalog) finishProvision(ctx context.Context, conn *sql.Conn, p TenantPlacement, reason string, initialize func(context.Context, TenantPlacement) error) (TenantPlacement, error) {
	initializerLease := &TenantLease{catalog: c, placement: p, conn: conn, initializing: true}
	p.initializerLease = initializerLease
	defer initializerLease.endInitialization()
	if err := initialize(ctx, p); err != nil {
		return p, err
	}
	initializerLease.endInitialization()
	// A lost administrative session must not be replaced: its lock is the fence
	// against deletion/retry while the initializer is running.
	if err := conn.PingContext(ctx); err != nil {
		return p, mapSQLError("verify Tenant provisioning fence", err)
	}
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return p, mapSQLError("begin Tenant activation", err)
	}
	defer tx.Rollback()
	if err := c.waitForInitializerData(ctx, tx, p.TenantID); err != nil {
		return p, err
	}
	current, err := c.lockMutation(ctx, tx, p.TenantID, p.Version)
	if err != nil {
		return p, err
	}
	if current.State != TenantProvisioning {
		return p, domain.NewError(domain.ErrorCodeInvalidTransition, "Tenant provisioning state changed")
	}
	var exists bool
	if err := tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=$1)`, p.Schema).Scan(&exists); err != nil {
		return p, mapSQLError("verify initialized Tenant schema", err)
	}
	if !exists {
		return p, domain.NewError(domain.ErrorCodePreconditionFailed, "Tenant initializer removed the reserved data schema")
	}
	p, err = c.updateState(ctx, tx, current, TenantActive, "activated", reason)
	if err != nil {
		return p, err
	}
	return p, mapSQLError("commit Tenant activation", tx.Commit())
}

// Store.Close closes client sockets, but PostgreSQL can still be processing
// their termination and releasing session locks. Successful initializer
// completion waits for that server acknowledgment while retaining the parent
// exclusive lifecycle fence. Destructive/admission operations keep their
// nonblocking checks; a leaked initializer session still prevents transition.
func (c *TenantCatalog) waitForInitializerData(ctx context.Context, tx *sql.Tx, id domain.ID) error {
	if _, err := tx.ExecContext(ctx, `SET LOCAL lock_timeout='5s'`); err != nil {
		return mapSQLError("bound Tenant initializer drain", err)
	}
	if _, err := tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, tenantInitializationKey(c.schema, id)); err != nil {
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "55P03" {
			return domain.NewError(domain.ErrorCodePreconditionFailed, "close all Tenant initializer database sessions before changing its lifecycle")
		}
		return mapSQLError("wait for Tenant initializer sessions to close", err)
	}
	return nil
}
