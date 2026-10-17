package postgres

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// MigrateTenant runs a trusted, guarded initializer against a suspended Tenant.
// It never activates the Tenant. A failed migration remains suspended, retaining
// the normal migration ledger so the same compiled chain can be retried.
func (c *TenantCatalog) MigrateTenant(ctx context.Context, id domain.ID, expected domain.Version, reason string, initialize func(context.Context, TenantPlacement) error) (TenantPlacement, error) {
	if err := validateTenantMutation(id, expected, reason); err != nil {
		return TenantPlacement{}, err
	}
	if initialize == nil {
		return TenantPlacement{}, domain.NewError(domain.ErrorCodeInvalidArgument, "Tenant migration requires an initializer")
	}
	conn, err := c.exclusiveSession(ctx, id)
	if err != nil {
		return TenantPlacement{}, err
	}
	defer conn.Close()
	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return TenantPlacement{}, mapSQLError("begin Tenant migration", err)
	}
	defer tx.Rollback()
	p, err := c.lockMutation(ctx, tx, id, expected)
	if err != nil {
		return p, err
	}
	if p.State != TenantSuspended {
		return p, domain.NewError(domain.ErrorCodeInvalidTransition, "suspend the Tenant before migrating its schema")
	}
	if err := tx.Commit(); err != nil {
		return p, mapSQLError("start suspended Tenant migration", err)
	}
	initializerLease := &TenantLease{catalog: c, placement: p, conn: conn, initializing: true}
	p.initializerLease = initializerLease
	defer func() {
		initializerLease.mu.Lock()
		initializerLease.closed = true
		initializerLease.mu.Unlock()
	}()
	if err := initialize(ctx, p); err != nil {
		return p, err
	}
	if err := conn.PingContext(ctx); err != nil {
		return p, mapSQLError("verify Tenant migration fence", err)
	}
	tx, err = conn.BeginTx(ctx, nil)
	if err != nil {
		return p, mapSQLError("begin Tenant migration receipt", err)
	}
	defer tx.Rollback()
	current, err := c.lockMutation(ctx, tx, id, expected)
	if err != nil {
		return p, err
	}
	if current.State != TenantSuspended {
		return p, domain.NewError(domain.ErrorCodeInvalidTransition, "Tenant state changed during migration")
	}
	p, err = c.updateState(ctx, tx, current, TenantSuspended, "schema_migrated", reason)
	if err != nil {
		return p, err
	}
	return p, mapSQLError("commit Tenant migration receipt", tx.Commit())
}
