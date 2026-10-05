package ports

import (
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"time"
)

// Observations carry safe metadata only; documentary payloads and credentials are excluded.
type CommandObservation struct {
	Name            string
	NamespaceID     domain.ID
	OutcomeID       domain.ID
	PrincipalID     string
	Actor           domain.ActorRef
	CommandID       domain.ID
	CorrelationID   string
	Revision        domain.OutcomeRevision
	Duration        time.Duration
	TransactionWait time.Duration
	CommitDuration  time.Duration
	ErrorCode       domain.ErrorCode
	Replay          bool
}
type CommandObserver interface{ ObserveCommand(CommandObservation) }
