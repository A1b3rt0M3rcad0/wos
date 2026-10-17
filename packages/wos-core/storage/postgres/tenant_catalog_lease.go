package postgres

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"sync"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/jackc/pgx/v5"
)

// TenantLease holds the routing/lifecycle fence for a running Tenant. Close the
// Tenant's Store before closing this lease, then close its catalog last.
type TenantLease struct {
	catalog      *TenantCatalog
	placement    TenantPlacement
	conn         *sql.Conn
	mu           sync.Mutex
	closed       bool
	initializing bool
}

func (c *TenantCatalog) AcquireTenantLease(ctx context.Context, id domain.ID) (*TenantLease, error) {
	if err := id.Validate(); err != nil {
		return nil, err
	}
	conn, err := c.leases.Conn(ctx)
	if err != nil {
		return nil, mapSQLError("open Tenant runtime lease", err)
	}
	var acquired bool
	if err := conn.QueryRowContext(ctx, `SELECT pg_try_advisory_lock_shared($1)`, tenantLifecycleKey(c.schema, id.String())).Scan(&acquired); err != nil {
		_ = conn.Close()
		return nil, mapSQLError("lock Tenant runtime placement", err)
	}
	if !acquired {
		_ = conn.Close()
		return nil, domain.NewError(domain.ErrorCodePreconditionFailed, "Tenant lifecycle operation is in progress")
	}
	p, err := scanTenantPlacement(conn.QueryRowContext(ctx, `SELECT `+tenantPlacementColumns+` FROM tenant_placements WHERE tenant_id=$1`, id.String()))
	if err != nil || p.State != TenantActive {
		_ = conn.Close()
		if err != nil {
			return nil, err
		}
		return nil, domain.NewError(domain.ErrorCodePreconditionFailed, "Tenant is not active")
	}
	return &TenantLease{catalog: c, placement: p, conn: conn}, nil
}

func (l *TenantLease) Placement() TenantPlacement { return l.placement }

func (l *TenantLease) CheckActive(ctx context.Context) error {
	if l == nil {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "Tenant runtime lease is missing")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "Tenant runtime lease is closed")
	}
	// Query the held connection, never a replacement pooled connection. A lost
	// connection also lost its PostgreSQL session lock and must fail readiness.
	p, err := scanTenantPlacement(l.conn.QueryRowContext(ctx, `SELECT `+tenantPlacementColumns+` FROM tenant_placements WHERE tenant_id=$1`, l.placement.TenantID.String()))
	if err != nil {
		return err
	}
	if p.State != l.expectedState() || p.Schema != l.placement.Schema || p.Version != l.placement.Version {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "Tenant runtime placement is no longer active")
	}
	return nil
}

// GuardConnection pins each data connection as well as the runtime lease. A
// dropped lease connection therefore cannot expose a live data pool to schema
// deletion. Newly reconnected data sessions revalidate the catalog under the
// same shared lock before executing any application SQL.
func (l *TenantLease) GuardConnection(ctx context.Context, conn *pgx.Conn) (err error) {
	// pgx stdlib's pinned connector does not close a connection when its
	// AfterConnect callback fails. Close explicitly, including any session lock,
	// so rejected/reconnecting clients cannot leak connections or block deletion.
	defer func() {
		if err != nil {
			closeCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = conn.Close(closeCtx)
		}
	}()
	if l == nil {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "Tenant runtime lease is missing")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "Tenant runtime lease is closed")
	}
	var acquired bool
	key := tenantLifecycleKey(l.catalog.schema, l.placement.TenantID.String())
	if l.initializing {
		key = tenantInitializationKey(l.catalog.schema, l.placement.TenantID)
	}
	if err := conn.QueryRow(ctx, `SELECT pg_try_advisory_lock_shared($1)`, key).Scan(&acquired); err != nil {
		return mapSQLError("pin Tenant data connection", err)
	}
	if !acquired {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "Tenant lifecycle operation is in progress")
	}
	var state, schema string
	var version domain.Version
	err = conn.QueryRow(ctx, `SELECT state,schema_name,version FROM `+l.catalog.schema+`.tenant_placements WHERE tenant_id=$1`, l.placement.TenantID.String()).Scan(&state, &schema, &version)
	if err != nil {
		return mapSQLError("validate Tenant data placement", err)
	}
	if state != string(l.expectedState()) || schema != l.placement.Schema || version != l.placement.Version {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "Tenant data placement is no longer active")
	}
	return nil
}

func (l *TenantLease) expectedState() TenantState {
	if l.initializing {
		return l.placement.State
	}
	return TenantActive
}

// Once the initializer callback has returned, retained options must not admit
// new data connections while its existing sessions finish closing.
func (l *TenantLease) endInitialization() {
	l.mu.Lock()
	l.closed = true
	l.mu.Unlock()
}

func (l *TenantLease) Close() error {
	if l == nil {
		return nil
	}
	if l.initializing {
		return domain.NewError(domain.ErrorCodeInvalidTransition, "initializer lease lifetime belongs to the Tenant catalog")
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.closed {
		return nil
	}
	l.closed = true
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var released bool
	err := l.conn.QueryRowContext(ctx, `SELECT pg_advisory_unlock_shared($1)`, tenantLifecycleKey(l.catalog.schema, l.placement.TenantID.String())).Scan(&released)
	if err != nil {
		// Never put a session with an uncertain lock back into a connection pool.
		_ = l.conn.Raw(func(any) error { return driver.ErrBadConn })
	}
	closeErr := l.conn.Close()
	if err != nil {
		return mapSQLError("release Tenant runtime lease", err)
	}
	return closeErr
}
