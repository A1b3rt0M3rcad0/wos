package ports

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type EntityNoteFilter struct {
	SubjectRef      *d.EntityRef
	CriterionID     *d.ID
	BeforeRevision  d.OutcomeRevision
	BeforeID        d.ID
	ThroughRevision d.OutcomeRevision
	Limit           int
}

// EntityNoteRepository offers no Save or Delete: corrections are new records.
type EntityNoteRepository interface {
	Insert(context.Context, d.EntityNote) error
	Get(context.Context, d.Scope, d.ID) (d.EntityNote, error)
	List(context.Context, d.Scope, EntityNoteFilter) ([]d.EntityNote, error)
}
type EntityNoteUnitOfWork interface{ EntityNotes() EntityNoteRepository }
