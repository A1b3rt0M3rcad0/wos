package application

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func (s *Service) beginReadSnapshot(ctx context.Context) (ports.UnitOfWork, error) {
	return beginManagerReadSnapshot(ctx, s.tx)
}

func beginManagerReadSnapshot(ctx context.Context, manager ports.TransactionManager) (ports.UnitOfWork, error) {
	if snapshots, ok := manager.(ports.ReadSnapshotManager); ok {
		return snapshots.BeginReadSnapshot(ctx)
	}
	// Preserve the stronger legacy guarded path for third-party adapters.
	return manager.Begin(ctx)
}
