package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"strconv"
)

type entityNoteRepository struct{ uow *unitOfWork }

func (u *unitOfWork) EntityNotes() ports.EntityNoteRepository { return entityNoteRepository{u} }
func (r entityNoteRepository) Insert(ctx context.Context, n d.EntityNote) error {
	if err := r.uow.ensureOpen(); err != nil {
		return err
	}
	if err := n.Validate(); err != nil {
		return err
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
	raw, err := marshalJSON(n)
	if err != nil {
		return err
	}
	q := `INSERT INTO entity_notes(id,namespace_id,outcome_id,subject_kind,subject_id,criterion_id,principal_id,outcome_revision,supersedes_note_id,payload_json) VALUES(?,?,?,?,?,?,?,?,?,?)`
	_, err = r.uow.tx.ExecContext(ctx, q, n.ID.String(), n.Scope.NamespaceID.String(), n.Scope.OutcomeID.String(), n.SubjectRef.Kind.String(), n.SubjectRef.ID.String(), nullableID(n.CriterionID), n.PrincipalID, activitySequenceKey(uint64(n.OutcomeRevision)), nullableID(n.SupersedesNoteID), raw)
	if err != nil {
		return mapSQLError("insert documentary note", err)
	}
	for _, refs := range []struct {
		kind string
		ids  []d.ID
	}{{"artifact", n.ArtifactRefs}, {"evidence", n.EvidenceRefs}} {
		for _, id := range refs.ids {
			q := `INSERT INTO entity_note_references(namespace_id,outcome_id,note_id,reference_kind,reference_id) VALUES(?,?,?,?,?)`
			if _, err = r.uow.tx.ExecContext(ctx, q, n.Scope.NamespaceID.String(), n.Scope.OutcomeID.String(), n.ID.String(), refs.kind, id.String()); err != nil {
				return mapSQLError("insert note reference", err)
			}
		}
	}
	return nil
}
func (r entityNoteRepository) Get(ctx context.Context, scope d.Scope, id d.ID) (d.EntityNote, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return d.EntityNote{}, err
	}
	if err := scope.Validate(); err != nil {
		return d.EntityNote{}, err
	}
	if err := id.Validate(); err != nil {
		return d.EntityNote{}, err
	}
	q := `SELECT id,subject_kind,subject_id,criterion_id,principal_id,outcome_revision,supersedes_note_id,payload_json FROM entity_notes WHERE namespace_id=? AND outcome_id=? AND id=?`

	n, err := scanEntityNote(r.uow.tx.QueryRowContext(ctx, q, scope.NamespaceID.String(), scope.OutcomeID.String(), id.String()), scope)
	if err != nil {
		return d.EntityNote{}, mapSQLError("get documentary note", err)
	}
	if n.ID != id {
		return d.EntityNote{}, d.NewError(d.ErrorCodeInvalidScope, "persisted note identity differs")
	}
	return n, nil
}
func (r entityNoteRepository) List(ctx context.Context, scope d.Scope, f ports.EntityNoteFilter) ([]d.EntityNote, error) {
	if err := r.uow.ensureOpen(); err != nil {
		return nil, err
	}
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if f.Limit < 1 || f.Limit > 101 || f.ThroughRevision == 0 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "invalid documentary note page bounds")
	}
	q := `SELECT id,subject_kind,subject_id,criterion_id,principal_id,outcome_revision,supersedes_note_id,payload_json FROM entity_notes WHERE namespace_id=? AND outcome_id=? AND outcome_revision<=?`
	args := []any{scope.NamespaceID.String(), scope.OutcomeID.String(), activitySequenceKey(uint64(f.ThroughRevision))}
	if f.SubjectRef != nil {
		q += ` AND subject_kind=? AND subject_id=?`
		args = append(args, f.SubjectRef.Kind.String(), f.SubjectRef.ID.String())
	}
	if f.CriterionID != nil {
		q += ` AND criterion_id=?`
		args = append(args, f.CriterionID.String())
	}
	if f.BeforeRevision != 0 {
		q += ` AND (outcome_revision<? OR (outcome_revision=? AND id<?))`
		key := activitySequenceKey(uint64(f.BeforeRevision))
		args = append(args, key, key, f.BeforeID.String())
	}
	q += ` ORDER BY outcome_revision DESC,id DESC LIMIT ?`
	args = append(args, f.Limit)
	rows, err := r.uow.tx.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, mapSQLError("list documentary notes", err)
	}
	defer rows.Close()
	out := []d.EntityNote{}
	for rows.Next() {
		n, err := scanEntityNote(rows, scope)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func scanEntityNote(row rowScanner, scope d.Scope) (d.EntityNote, error) {
	var id, kind, subject, principal, revision, raw string
	var criterion, supersedes sql.NullString
	if err := row.Scan(&id, &kind, &subject, &criterion, &principal, &revision, &supersedes, &raw); err != nil {
		return d.EntityNote{}, err
	}
	var note d.EntityNote
	if err := json.Unmarshal([]byte(raw), &note); err != nil {
		return note, err
	}
	parsed, err := strconv.ParseUint(revision, 10, 64)
	if err != nil {
		return note, err
	}
	sameOptional := func(column sql.NullString, value *d.ID) bool {
		return !column.Valid && value == nil || column.Valid && value != nil && column.String == value.String()
	}
	if note.ID.String() != id || note.Scope != scope || note.SubjectRef.Kind.String() != kind || note.SubjectRef.ID.String() != subject || note.PrincipalID != principal || uint64(note.OutcomeRevision) != parsed || !sameOptional(criterion, note.CriterionID) || !sameOptional(supersedes, note.SupersedesNoteID) {
		return d.EntityNote{}, d.NewError(d.ErrorCodeInvalidScope, "persisted note binding differs")
	}
	return note, note.Validate()
}
