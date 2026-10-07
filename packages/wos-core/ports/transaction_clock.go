package ports

import (
	"context"
	"time"
)

// TransactionClock is supplied by distributed storage. Call it after acquiring
// the Outcome guard so lock wait cannot make a lease decision use stale time.
type TransactionClock interface {
	EvaluationTime(context.Context) (time.Time, error)
}
