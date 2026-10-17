package ports

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// ReadSnapshotManager is optional for embedded adapters. All domain reads and
// the revision marker returned by a snapshot belong to one database snapshot.
// Rollback closes the snapshot; it must never be used to commit domain writes.
// Implementations may retain shared security barriers so revocation remains
// ordered against an authorized read, but must not take Outcome write guards.
type ReadSnapshotManager interface {
	BeginReadSnapshot(context.Context) (UnitOfWork, error)
}

// SigningNamespaceVersionReader reads the diagnostic Namespace CAS marker
// without upgrading the shared security barrier to an exclusive mutation lock.
type SigningNamespaceVersionReader interface {
	NamespaceVersion(context.Context, domain.ID) (domain.Version, error)
}
