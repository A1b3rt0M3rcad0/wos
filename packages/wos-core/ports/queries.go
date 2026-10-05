package ports

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"time"
)

type OutcomeFilter struct {
	Text         string           `json:"text,omitempty"`
	Lifecycle    string           `json:"lifecycle,omitempty"`
	Archived     *bool            `json:"archived,omitempty"`
	Priority     domain.Priority  `json:"priority,omitempty"`
	Owner        *domain.ActorRef `json:"owner,omitempty"`
	AfterCreated time.Time        `json:"-"`
	AfterID      domain.ID        `json:"-"`
	Limit        int              `json:"-"`
}
type OutcomeDiscoveryRepository interface {
	ListOutcomes(context.Context, domain.ID, OutcomeFilter) ([]domain.Outcome, error)
}
type EventFilter struct {
	AfterRevision domain.OutcomeRevision
	AfterIndex    uint32
	Limit         int
	PrincipalID   string
	CommandID     domain.ID
	EntityID      domain.ID
	EventType     string
}
type TimelineRepository interface {
	ListEvents(context.Context, domain.Scope, EventFilter) ([]domain.DomainEvent, error)
}
