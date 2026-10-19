package ports

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// WorkspaceTimelineFilter bounds the event spine rather than reading a whole
// aggregate or history. BeforeRevision/Index are an exclusive descending key.
type WorkspaceTimelineFilter struct {
	SubjectRef         d.EntityRef
	CriterionID        *d.ID
	IncludeDescendants bool
	ThroughRevision    d.OutcomeRevision
	BeforeRevision     d.OutcomeRevision
	BeforeIndex        uint32
	Limit              int
}
type WorkspaceTimelineRepository interface {
	ListWorkspaceEvents(context.Context, d.Scope, WorkspaceTimelineFilter) ([]d.DomainEvent, error)
	WorkspaceStructureChanged(context.Context, d.Scope, d.OutcomeRevision) (bool, error)
}
