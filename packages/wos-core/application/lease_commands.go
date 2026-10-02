package application

import (
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// RenewWorkItemLeaseCommand extends a live lease owned by the authenticated
// principal. Claim identity and fencing token remain unchanged.
type RenewWorkItemLeaseCommand struct {
	Scope           domain.Scope
	WorkItemID      domain.ID
	ExpectedVersion domain.Version
	ClaimID         domain.ID
	FencingToken    uint64
	TTL             time.Duration
}

// ReclaimWorkItemCommand acquires an in-progress WorkItem whose current lease
// has expired. A successful reclaim receives a new claim ID and fencing token.
type ReclaimWorkItemCommand struct {
	Scope           domain.Scope
	WorkItemID      domain.ID
	ExpectedVersion domain.Version
	TTL             time.Duration
}
