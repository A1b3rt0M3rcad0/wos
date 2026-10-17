package ports

import (
	"context"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// SignedQuotaUnitOfWork narrows acquisition coordination independently of the
// shared protocol/security barrier. Tenant ceilings require a shared quota
// lane; otherwise only acquisitions by the same Principal compete. A Principal
// lane also covers all its credentials. Locks live until transaction completion.
type SignedQuotaUnitOfWork interface {
	LockSignedQuota(context.Context, domain.ID, string, string, bool) error
}

// SignedQuotaCountsRepository avoids scanning Tenant-wide authority when
// no Tenant ceiling is enabled. Counts remains the full diagnostic query.
type SignedQuotaCountsRepository interface {
	CountQuota(context.Context, domain.ID, string, domain.ID, time.Time, bool) (SignedActiveCounts, error)
}
