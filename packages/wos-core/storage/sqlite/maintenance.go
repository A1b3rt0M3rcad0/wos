package sqlite

import (
	"context"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// PruneExpiredIdempotency removes at most limit completed, expired entries from
// the disposable short retry cache. Durable signed results, receipts, proofs,
// revocations, and unfinished reservations are outside this operation.
func (s *Store) PruneExpiredIdempotency(ctx context.Context, limit int) (int64, error) {
	if limit < 1 || limit > 10000 {
		return 0, domain.NewError(domain.ErrorCodeInvalidArgument, "idempotency cleanup limit must be 1..10000")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, mapSQLError("begin idempotency cleanup", err)
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `DELETE FROM idempotency_records WHERE (namespace_id,principal_id,command_name,idempotency_key) IN (SELECT namespace_id,principal_id,command_name,idempotency_key FROM idempotency_records WHERE status='completed' AND expires_at<=? ORDER BY expires_at,namespace_id,principal_id,command_name,idempotency_key LIMIT ?)`, encodeTime(time.Now().UTC()), limit)
	if err != nil {
		return 0, mapSQLError("prune expired idempotency cache", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, mapSQLError("commit idempotency cleanup", err)
	}
	return count, nil
}
