package sqlite

import (
	"context"
	"fmt"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// Replay must succeed without consulting or reserving the short cache. The same
// fixture executes on Memory, SQLite and the generated real PostgreSQL suite.
type signedNoCacheManager struct{ base ports.TransactionManager }

func (m signedNoCacheManager) Begin(ctx context.Context) (ports.UnitOfWork, error) {
	u, e := m.base.Begin(ctx)
	if e != nil {
		return nil, e
	}
	return signedNoCacheUnit{UnitOfWork: u, SignedOperationUnitOfWork: u.(ports.SignedOperationUnitOfWork), WorkProtocolUnitOfWork: u.(ports.WorkProtocolUnitOfWork), SigningIdentityUnitOfWork: u.(ports.SigningIdentityUnitOfWork), AccessSnapshotUnitOfWork: u.(ports.AccessSnapshotUnitOfWork)}, nil
}

type signedNoCacheUnit struct {
	ports.UnitOfWork
	ports.SignedOperationUnitOfWork
	ports.WorkProtocolUnitOfWork
	ports.SigningIdentityUnitOfWork
	ports.AccessSnapshotUnitOfWork
}

func (signedNoCacheUnit) Idempotency() ports.IdempotencyStore { return unavailableSignedCache{} }

type unavailableSignedCache struct{}

func (unavailableSignedCache) Reserve(context.Context, d.IdempotencyIdentity, string) (d.IdempotencyReservation, error) {
	return d.IdempotencyReservation{}, fmt.Errorf("short cache must not participate in durable replay")
}
func (unavailableSignedCache) Complete(context.Context, d.IdempotencyReservation, d.StoredCommandResult) error {
	return fmt.Errorf("short cache must not participate in durable replay")
}
