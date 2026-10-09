package ports

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// ReferenceFilter is a bounded metadata query, not aggregate hydration. Kinds
// are a fixed public allowlist, never storage table names supplied by callers.
type ReferenceFilter struct {
	Kinds     []domain.EntityKind
	Query     string
	Lifecycle string
	ID        domain.ID
	AfterKind domain.EntityKind
	AfterID   domain.ID
	Limit     int
}
type ReferenceCandidate struct {
	Ref            domain.EntityRef `json:"ref"`
	Title          string           `json:"title"`
	Lifecycle      string           `json:"lifecycle"`
	Version        domain.Version   `json:"version"`
	DisplayContext string           `json:"display_context,omitempty"`
}
type ReferenceSearchRepository interface {
	SearchReferences(context.Context, domain.Scope, ReferenceFilter) ([]ReferenceCandidate, error)
}
