package ports

import (
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"time"
)

// QueryObservation describes a completed application query, never its content.
type QueryObservation struct {
	ReturnedItems *int // Optional bounded page count; never query text or content.
	Name          string
	Scope         domain.Scope
	PrincipalID   string
	Actor         domain.ActorRef
	Revision      domain.OutcomeRevision
	Duration      time.Duration
	ErrorCode     domain.ErrorCode
}
type QueryObserver interface{ ObserveQuery(QueryObservation) }
