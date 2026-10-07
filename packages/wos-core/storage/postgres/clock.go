package postgres

import (
	"context"
	"time"
)

func (u *unitOfWork) EvaluationTime(ctx context.Context) (time.Time, error) {
	if err := u.ensureOpen(); err != nil {
		return time.Time{}, err
	}
	var now time.Time
	// CURRENT_TIMESTAMP is transaction-start time; clock_timestamp observes the
	// current instant, including time spent waiting for the Outcome row guard.
	if err := u.tx.QueryRowContext(ctx, "SELECT clock_timestamp()").Scan(&now); err != nil {
		return time.Time{}, mapSQLError("read authoritative evaluation time", err)
	}
	return now.UTC(), nil
}
