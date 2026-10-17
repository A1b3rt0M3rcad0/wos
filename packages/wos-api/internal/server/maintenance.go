package server

import (
	"context"
	"log/slog"
	"time"
)

type expiredIdempotencyPruner interface {
	PruneExpiredIdempotency(context.Context, int) (int64, error)
}

// Each pass is bounded and coordinated by the adapter, so multiple workers can
// maintain an installation without a process-local or global leader. This only
// removes disposable completed cache entries; durable signed receipts are not a
// maintenance target.
func (r *Runtime) runMaintenance(ctx context.Context, interval time.Duration) {
	pruner, ok := r.store.(expiredIdempotencyPruner)
	if !ok {
		return
	}
	timer := time.NewTicker(interval)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-timer.C:
			runMaintenancePass(ctx, pruner)
		}
	}
}

func runMaintenancePass(ctx context.Context, pruner expiredIdempotencyPruner) {
	passCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// Keep every lock/delete batch small while allowing a busy installation to
	// drain more than 256 expired records per minute. Both time and batch count
	// bound the pass, and partial batches yield immediately.
	for batch := 0; batch < 4096 && passCtx.Err() == nil; batch++ {
		removed, err := pruner.PruneExpiredIdempotency(passCtx, 256)
		if err != nil {
			if ctx.Err() == nil && passCtx.Err() == nil {
				// Never log a database error that can contain connection details.
				slog.Warn("idempotency_maintenance_failed", "failure_class", "storage_error")
			}
			return
		}
		if removed < 256 {
			return
		}
	}
}
