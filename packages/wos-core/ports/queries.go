package ports

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"time"
)

type OutcomeFilter struct {
	ExternalProvider string           `json:"external_provider,omitempty"`
	ExternalKind     string           `json:"external_kind,omitempty"`
	ExternalID       string           `json:"external_id,omitempty"`
	Text             string           `json:"text,omitempty"`
	Lifecycle        string           `json:"lifecycle,omitempty"`
	Archived         *bool            `json:"archived,omitempty"`
	Priority         domain.Priority  `json:"priority,omitempty"`
	Owner            *domain.ActorRef `json:"owner,omitempty"`
	AfterCreated     time.Time        `json:"-"`
	AfterID          domain.ID        `json:"-"`
	Limit            int              `json:"-"`
}
type OutcomeDiscoveryRepository interface {
	ListOutcomes(context.Context, domain.ID, OutcomeFilter) ([]OutcomeIndexEntry, error)
}

// OutcomeIndexEntry is discovery metadata, never a hydrated domain aggregate.
type OutcomeIndexEntry struct {
	ID            domain.ID               `json:"id"`
	NamespaceID   domain.ID               `json:"namespace_id"`
	Version       domain.Version          `json:"version"`
	Title         string                  `json:"title"`
	Description   string                  `json:"description,omitempty"`
	Lifecycle     domain.OutcomeLifecycle `json:"lifecycle"`
	Priority      domain.Priority         `json:"priority"`
	ArchivedAt    *time.Time              `json:"archived_at,omitempty"`
	CreatedAt     time.Time               `json:"created_at"`
	UpdatedAt     time.Time               `json:"updated_at"`
	DetailOmitted bool                    `json:"detail_omitted"`
}

func (v OutcomeIndexEntry) Scope() domain.Scope {
	return domain.Scope{NamespaceID: v.NamespaceID, OutcomeID: v.ID}
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
