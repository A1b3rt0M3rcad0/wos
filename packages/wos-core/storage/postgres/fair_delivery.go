package postgres

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func fairDeliveryBind(query string) string { return rebind(query) }

const fairDeliveryLock = " FOR UPDATE SKIP LOCKED"

// selectFairDelivery reserves one namespace turn before checking live leases.
// PostgreSQL uses a dedicated SKIP LOCKED scheduler row, independent from the
// domain/security Namespace row. SQLite's immediate writer transaction gives
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
		candidateDue += ` AND (SELECT COUNT(*) FROM (SELECT 1 FROM integration_deliveries live WHERE live.namespace_id=d.namespace_id AND live.endpoint_id=d.endpoint_id AND live.status='leased' AND live.lease_until>? LIMIT ?) endpoint_capacity)<?`
		candidateArgs = append(candidateArgs, encodeTime(now), policy.MaxInflightPerEndpoint, policy.MaxInflightPerEndpoint)
	}
	candidates := `SELECT s.namespace_id FROM delivery_namespace_schedule s WHERE EXISTS(SELECT 1 FROM integration_deliveries d WHERE d.namespace_id=s.namespace_id AND ` + candidateDue + `)`
	if policy.MaxInflightPerNamespace > 0 {
		candidates += ` AND (SELECT COUNT(*) FROM (SELECT 1 FROM integration_deliveries live WHERE live.namespace_id=s.namespace_id AND live.status='leased' AND live.lease_until>? LIMIT ?) namespace_capacity)<?`
		candidateArgs = append(candidateArgs, encodeTime(now), policy.MaxInflightPerNamespace, policy.MaxInflightPerNamespace)
	}
	rows, err := tx.QueryContext(ctx, fairDeliveryBind(candidates+` ORDER BY s.last_turn,s.namespace_id LIMIT 64`), candidateArgs...)
	if err != nil {
		return zero, err
	}
	var namespaces []string
	for rows.Next() {
		var ns string
		if err = rows.Scan(&ns); err != nil {
			rows.Close()
			return zero, err
		}
		namespaces = append(namespaces, ns)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return zero, err
	}
	for _, ns := range namespaces {
		var locked string
		err = tx.QueryRowContext(ctx, fairDeliveryBind(`SELECT namespace_id FROM delivery_namespace_schedule WHERE namespace_id=?`)+fairDeliveryLock, ns).Scan(&locked)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return zero, err
		}
		if policy.MaxInflightPerNamespace > 0 {
			var count int
			err = tx.QueryRowContext(ctx, fairDeliveryBind(`SELECT COUNT(*) FROM (SELECT 1 FROM integration_deliveries WHERE namespace_id=? AND status='leased' AND lease_until>? LIMIT ?) active`), ns, encodeTime(now), policy.MaxInflightPerNamespace).Scan(&count)
			if err != nil {
				return zero, err
			}
			if count >= policy.MaxInflightPerNamespace {
				continue
			}
		}
		query := `SELECT ` + deliveryColumns + ` FROM integration_deliveries d WHERE d.namespace_id=? AND ` + due
		values := append([]any{ns}, args...)
		if policy.MaxInflightPerEndpoint > 0 {
			query += ` AND (SELECT COUNT(*) FROM (SELECT 1 FROM integration_deliveries live WHERE live.namespace_id=d.namespace_id AND live.endpoint_id=d.endpoint_id AND live.status='leased' AND live.lease_until>? LIMIT ?) endpoint_capacity)<?`
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
		// Concurrent namespaces can share a turn; stable Namespace ID breaks ties.
		if _, err = tx.ExecContext(ctx, fairDeliveryBind(`UPDATE delivery_namespace_schedule SET last_turn=(SELECT COALESCE(MAX(last_turn),0)+1 FROM delivery_namespace_schedule) WHERE namespace_id=?`), ns); err != nil {
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
	var now time.Time
	err := tx.QueryRowContext(ctx, "SELECT clock_timestamp()").Scan(&now)
	return now.UTC(), mapSQLError("read delivery clock", err)
}

func lockDeliveryForACK(ctx context.Context, tx *sql.Tx, id string) error {
	var locked string
	return mapSQLError("lock delivery acknowledgement", tx.QueryRowContext(ctx, `SELECT id FROM integration_deliveries WHERE id=$1 FOR UPDATE`, id).Scan(&locked))
}

var _ ports.DurableDeliveryStore = (*Store)(nil)

func (s *Store) FinishDeliveryUsingDatabaseTime(ctx context.Context, v ports.Delivery, result string, success bool, retryDelay time.Duration) error {
	if retryDelay < 0 || retryDelay > 24*time.Hour {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "delivery retry delay must be 0..24h")
	}
	return s.finishDelivery(ctx, v, time.Time{}, result, success, time.Time{}, true, retryDelay)
}
