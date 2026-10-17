package postgres

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// PruneExpiredIdempotency removes only the completed, expired short retry cache.
// Row-local SKIP LOCKED claims allow multiple maintenance workers to make bounded
// progress without global leadership. Domain and signed receipts are untouched.
func (s *Store) PruneExpiredIdempotency(ctx context.Context, limit int) (int64, error) {
	if limit < 1 || limit > 10000 {
		return 0, domain.NewError(domain.ErrorCodeInvalidArgument, "idempotency cleanup limit must be 1..10000")
	}
	result, err := s.db.ExecContext(ctx, `WITH expired AS (SELECT tenant_id,principal_id,command_name,idempotency_key FROM idempotency_records WHERE status='completed' AND expires_at<=floor(extract(epoch FROM statement_timestamp())*1000000)::bigint ORDER BY expires_at,tenant_id,principal_id,command_name,idempotency_key LIMIT $1 FOR UPDATE SKIP LOCKED) DELETE FROM idempotency_records cache USING expired WHERE cache.tenant_id=expired.tenant_id AND cache.principal_id=expired.principal_id AND cache.command_name=expired.command_name AND cache.idempotency_key=expired.idempotency_key`, limit)
	if err != nil {
		return 0, mapSQLError("prune expired idempotency cache", err)
	}
	return result.RowsAffected()
}
