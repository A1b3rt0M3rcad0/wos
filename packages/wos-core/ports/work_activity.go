package ports

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type WorkActivityFilter struct {
	WorkItemID      domain.ID
	ContractID      domain.ID
	Descending      bool
	AfterSequence   uint64
	ThroughSequence uint64
	Limit           int
}

// WorkActivityRepository is append-only. Reads use indexed Outcome ordering;
// ThroughSequence freezes a traversal without loading the entire history.
type WorkActivityRepository interface {
	Insert(context.Context, domain.WorkActivity) error
	List(context.Context, domain.Scope, WorkActivityFilter) ([]domain.WorkActivity, error)
}
type WorkActivityUnitOfWork interface{ WorkActivities() WorkActivityRepository }
