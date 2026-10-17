package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func fairDeliveryBind(query string) string { return query }

const fairDeliveryLock = ""

// selectFairDelivery reserves one tenant turn before checking live leases.
// PostgreSQL uses a dedicated SKIP LOCKED scheduler row, independent from the
// domain/security Tenant row. SQLite's immediate writer transaction gives
// the same lease/count atomicity. No external call holds this short guard.
func selectFairDelivery(ctx context.Context, tx *sql.Tx, now time.Time, policy ports.DeliveryClaimPolicy) (ports.Delivery, error) {
	var zero ports.Delivery
	if policy.SecretRefs != nil && len(policy.SecretRefs) == 0 {
		return zero, sql.ErrNoRows
	}
	due := `((d.status='pending' AND d.next_attempt<=?) OR (d.status='leased' AND d.lease_until<=?))`
	args := []any{encodeTime(now), encodeTime(now)}
	if policy.SecretRefs != nil {
		due += ` AND d.secret_ref IN (` + strings.TrimSuffix(strings.Repeat("?,", len(policy.SecretRefs)), ",") + `)`
		for _, ref := range policy.SecretRefs {
			args = append(args, ref)
		}
	}
	candidateDue := due
	candidateArgs := append([]any{}, args...)
	if policy.MaxInflightPerEndpoint > 0 {
		candidateDue += ` AND (SELECT COUNT(*) FROM (SELECT 1 FROM integration_deliveries live WHERE live.tenant_id=d.tenant_id AND live.endpoint_id=d.endpoint_id AND live.status='leased' AND live.lease_until>? LIMIT ?) endpoint_capacity)<?`
		candidateArgs = append(candidateArgs, encodeTime(now), policy.MaxInflightPerEndpoint, policy.MaxInflightPerEndpoint)
	}
	candidates := `SELECT s.tenant_id FROM delivery_tenant_schedule s WHERE EXISTS(SELECT 1 FROM integration_deliveries d WHERE d.tenant_id=s.tenant_id AND ` + candidateDue + `)`
	if policy.MaxInflightPerTenant > 0 {
		candidates += ` AND (SELECT COUNT(*) FROM (SELECT 1 FROM integration_deliveries live WHERE live.tenant_id=s.tenant_id AND live.status='leased' AND live.lease_until>? LIMIT ?) tenant_capacity)<?`
		candidateArgs = append(candidateArgs, encodeTime(now), policy.MaxInflightPerTenant, policy.MaxInflightPerTenant)
	}
	rows, err := tx.QueryContext(ctx, fairDeliveryBind(candidates+` ORDER BY s.last_turn,s.tenant_id LIMIT 64`), candidateArgs...)
	if err != nil {
		return zero, err
	}
	var tenants []string
	for rows.Next() {
		var ns string
		if err = rows.Scan(&ns); err != nil {
			rows.Close()
			return zero, err
		}
		tenants = append(tenants, ns)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return zero, err
	}
	for _, ns := range tenants {
		var locked string
		err = tx.QueryRowContext(ctx, fairDeliveryBind(`SELECT tenant_id FROM delivery_tenant_schedule WHERE tenant_id=?`)+fairDeliveryLock, ns).Scan(&locked)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return zero, err
		}
		if policy.MaxInflightPerTenant > 0 {
			var count int
			err = tx.QueryRowContext(ctx, fairDeliveryBind(`SELECT COUNT(*) FROM (SELECT 1 FROM integration_deliveries WHERE tenant_id=? AND status='leased' AND lease_until>? LIMIT ?) active`), ns, encodeTime(now), policy.MaxInflightPerTenant).Scan(&count)
			if err != nil {
				return zero, err
			}
			if count >= policy.MaxInflightPerTenant {
				continue
			}
		}
		query := `SELECT ` + deliveryColumns + ` FROM integration_deliveries d WHERE d.tenant_id=? AND ` + due
		values := append([]any{ns}, args...)
		if policy.MaxInflightPerEndpoint > 0 {
			query += ` AND (SELECT COUNT(*) FROM (SELECT 1 FROM integration_deliveries live WHERE live.tenant_id=d.tenant_id AND live.endpoint_id=d.endpoint_id AND live.status='leased' AND live.lease_until>? LIMIT ?) endpoint_capacity)<?`
			values = append(values, encodeTime(now), policy.MaxInflightPerEndpoint, policy.MaxInflightPerEndpoint)
		}
		v, e := scanDelivery(tx.QueryRowContext(ctx, fairDeliveryBind(query+` ORDER BY next_attempt,id LIMIT 1`)+fairDeliveryLock, values...))
		if errors.Is(e, sql.ErrNoRows) {
			continue
		}
		if e != nil {
			return zero, e
		}
		// A monotonically increasing turn also works with deterministic clocks.
		// Concurrent tenants can share a turn; stable Tenant ID breaks ties.
		if _, err = tx.ExecContext(ctx, fairDeliveryBind(`UPDATE delivery_tenant_schedule SET last_turn=(SELECT COALESCE(MAX(last_turn),0)+1 FROM delivery_tenant_schedule) WHERE tenant_id=?`), ns); err != nil {
			return zero, err
		}
		return v, nil
	}
	return zero, sql.ErrNoRows
}

var _ ports.FairDeliveryStore = (*Store)(nil)

// deliveryDatabaseTime is evaluated in the short queue transaction, after it
// begins. PostgreSQL workers never decide fleet lease authority from host time.
func deliveryDatabaseTime(ctx context.Context, tx *sql.Tx) (time.Time, error) {
	return time.Now().UTC(), nil // embedded single-host clock
}

func lockDeliveryForACK(context.Context, *sql.Tx, string) error { return nil }
