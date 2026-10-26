package memory

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sort"
)

type entityNoteRepository struct{ tx *transaction }

func (tx *transaction) EntityNotes() ports.EntityNoteRepository { return entityNoteRepository{tx} }
func cloneEntityNotes(src map[d.ID]d.EntityNote) map[d.ID]d.EntityNote {
	out := make(map[d.ID]d.EntityNote, len(src))
	for id, n := range src {
		out[id] = contractCopy(n)
	}
	return out
}
func (r entityNoteRepository) Insert(ctx context.Context, n d.EntityNote) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := n.Validate(); err != nil {
		return err
	}
	if _, exists := r.tx.entityNotes[n.ID]; exists {
		return d.NewError(d.ErrorCodeAlreadyExists, "documentary notes are immutable")
	}
	records, err := r.tx.Outcomes().(ports.ReferenceSearchRepository).SearchReferences(ctx, n.Scope, ports.ReferenceFilter{Kinds: []d.EntityKind{n.SubjectRef.Kind}, ID: n.SubjectRef.ID, Limit: 1})
	if err != nil {
		return err
	}
	if len(records) != 1 || records[0].Ref != n.SubjectRef {
		return d.NewError(d.ErrorCodeNotFound, "note subject does not resolve in its scope")
	}
	if n.CriterionID != nil {
		var criteria d.CriterionSet
		switch n.SubjectRef.Kind {
		case d.EntityKindOutcome:
			owner, err := r.tx.Outcomes().Get(ctx, n.Scope.NamespaceID, n.Scope.OutcomeID)
			if err != nil {
				return err
			}
			criteria = owner.Criteria
		case d.EntityKindObjective:
			owner, err := r.tx.Objectives().Get(ctx, n.Scope, n.SubjectRef.ID)
			if err != nil {
				return err
			}
			criteria = owner.Criteria
		case d.EntityKindWorkItem:
			owner, err := r.tx.WorkItems().Get(ctx, n.Scope, n.SubjectRef.ID)
			if err != nil {
				return err
			}
			criteria = owner.Criteria
		}
		if _, err := criteria.Find(*n.CriterionID); err != nil {
			return err
		}
	}
	if n.SupersedesNoteID != nil {
		previous, err := r.Get(ctx, n.Scope, *n.SupersedesNoteID)
		if err != nil {
			return err
		}
		if err = n.ValidateCorrection(previous); err != nil {
			return err
		}
	}
	for _, id := range n.ArtifactRefs {
		if _, err := r.tx.Artifacts().Get(ctx, n.Scope, id); err != nil {
			return err
		}
	}
	for _, id := range n.EvidenceRefs {
		if _, err := r.tx.Evidence().Get(ctx, n.Scope, id); err != nil {
			return err
		}
	}
	r.tx.entityNotes[n.ID] = contractCopy(n)
	return nil
}
func (r entityNoteRepository) Get(ctx context.Context, scope d.Scope, id d.ID) (d.EntityNote, error) {
	if err := ctx.Err(); err != nil {
		return d.EntityNote{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return d.EntityNote{}, err
	}
	if err := scope.Validate(); err != nil {
		return d.EntityNote{}, err
	}
	if err := id.Validate(); err != nil {
		return d.EntityNote{}, err
	}
	n, ok := r.tx.entityNotes[id]
	if !ok || n.Scope != scope {
		return d.EntityNote{}, d.NewError(d.ErrorCodeNotFound, "documentary note not found")
	}
	return contractCopy(n), nil
}
func (r entityNoteRepository) List(ctx context.Context, scope d.Scope, f ports.EntityNoteFilter) ([]d.EntityNote, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if f.Limit < 1 || f.Limit > 101 || f.ThroughRevision == 0 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "invalid documentary note page bounds")
	}
	out := []d.EntityNote{}
	for _, n := range r.tx.entityNotes {
		if n.Scope != scope || n.OutcomeRevision > f.ThroughRevision || f.SubjectRef != nil && n.SubjectRef != *f.SubjectRef || f.CriterionID != nil && (n.CriterionID == nil || *n.CriterionID != *f.CriterionID) {
			continue
		}
		if f.BeforeRevision != 0 && (n.OutcomeRevision > f.BeforeRevision || n.OutcomeRevision == f.BeforeRevision && n.ID >= f.BeforeID) {
			continue
		}
		out = append(out, contractCopy(n))
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].OutcomeRevision == out[j].OutcomeRevision {
			return out[i].ID > out[j].ID
		}
		return out[i].OutcomeRevision > out[j].OutcomeRevision
	})
	return out[:min(f.Limit, len(out))], nil
}
