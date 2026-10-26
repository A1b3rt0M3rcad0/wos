package application

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"strings"
	"time"
)

type AppendEntityNoteCommand struct {
	Scope            d.Scope
	SubjectRef       d.EntityRef
	CriterionID      *d.ID `wos:"optional"`
	Type             d.EntityNoteType
	Body             string     `wos:"required"`
	OccurredAt       *time.Time `wos:"optional"`
	ArtifactRefs     []d.ID     `wos:"optional"`
	EvidenceRefs     []d.ID     `wos:"optional"`
	SupersedesNoteID *d.ID      `wos:"optional"`
}

func (s *Service) AppendEntityNote(ctx context.Context, cc d.CommandContext, cmd AppendEntityNoteCommand) (MutationResult[d.EntityNote], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[d.EntityNote]{}, err
	}
	if cc.IdempotencyKey == "" {
		return MutationResult[d.EntityNote]{}, d.NewError(d.ErrorCodeInvalidArgument, "a documentary note requires an idempotency intent")
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (d.EntityNote, d.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, u, cmd.Scope); err != nil {
			return d.EntityNote{}, 0, err
		}
		at, err := s.transactionTime(ctx, u)
		if err != nil {
			return d.EntityNote{}, 0, err
		}
		revision, err := u.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		if err != nil {
			return d.EntityNote{}, 0, err
		}
		note, err := s.appendEntityNoteInUnitOfWork(ctx, u, cc, cmd, at, revision)
		return note, revision, err
	})
}

// appendEntityNoteInUnitOfWork is also used by atomic observation+Evidence
// registration. Its caller owns the Outcome guard, single revision and events.
func (s *Service) appendEntityNoteInUnitOfWork(ctx context.Context, u ports.UnitOfWork, cc d.CommandContext, cmd AppendEntityNoteCommand, at time.Time, revision d.OutcomeRevision) (d.EntityNote, error) {
	var zero d.EntityNote
	if len(cmd.Body) > d.MaxEntityNoteBodyBytes {
		return zero, d.NewError(d.ErrorCodeInvalidArgument, "note body exceeds 16 KiB")
	}
	repo, err := entityNotesRepository(u)
	if err != nil {
		return zero, err
	}
	outcome, err := u.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
	if err != nil {
		return zero, err
	}
	if outcome.IsArchived() {
		return zero, d.NewError(d.ErrorCodePreconditionFailed, "archived Outcomes cannot receive documentary notes")
	}
	if cmd.SubjectRef.Scope != cmd.Scope {
		return zero, d.NewError(d.ErrorCodeInvalidScope, "note subject must belong to this Outcome")
	}
	if err = validateNoteSubject(ctx, u, cmd.SubjectRef, cmd.CriterionID); err != nil {
		return zero, err
	}
	if cmd.SubjectRef.Kind == d.EntityKindRoadmap {
		planning, ok := u.(ports.PlanningUnitOfWork)
		if !ok {
			return zero, d.NewError(d.ErrorCodeInvalidConfig, "Roadmap storage is unavailable")
		}
		roadmap, err := planning.Roadmaps().Get(ctx, cmd.Scope, cmd.SubjectRef.ID)
		if err != nil {
			return zero, err
		}
		if roadmap.ArchivedAt != nil {
			return zero, d.NewError(d.ErrorCodePreconditionFailed, "archived Roadmaps cannot receive documentary notes")
		}
	}
	id, err := s.ids.NewID()
	if err != nil {
		return zero, err
	}
	note := d.EntityNote{ID: id, Scope: cmd.Scope, SubjectRef: cmd.SubjectRef, CriterionID: cloneIDPointer(cmd.CriterionID), Type: cmd.Type, Body: strings.TrimSpace(cmd.Body), PrincipalID: cc.PrincipalID, Actor: cc.Actor, RecordedAt: at.UTC(), OccurredAt: cmd.OccurredAt, ArtifactRefs: append([]d.ID(nil), cmd.ArtifactRefs...), EvidenceRefs: append([]d.ID(nil), cmd.EvidenceRefs...), SupersedesNoteID: cloneIDPointer(cmd.SupersedesNoteID), OutcomeRevision: revision}
	if err = note.Validate(); err != nil {
		return zero, err
	}
	if err = validateEntityNoteReferences(ctx, u, note); err != nil {
		return zero, err
	}
	if note.SupersedesNoteID != nil {
		previous, err := repo.Get(ctx, cmd.Scope, *note.SupersedesNoteID)
		if err != nil {
			return zero, err
		}
		if err = note.ValidateCorrection(previous); err != nil {
			return zero, err
		}
	}
	if err = repo.Insert(ctx, note); err != nil {
		return zero, err
	}
	return note, nil
}

func entityNotesRepository(u ports.UnitOfWork) (ports.EntityNoteRepository, error) {
	notes, ok := u.(ports.EntityNoteUnitOfWork)
	if !ok {
		return nil, d.NewError(d.ErrorCodeInvalidConfig, "transaction adapter does not support documentary notes")
	}
	return notes.EntityNotes(), nil
}

func validateNoteSubject(ctx context.Context, u ports.UnitOfWork, ref d.EntityRef, criterionID *d.ID) error {
	if err := ref.Validate(); err != nil {
		return err
	}
	if !referenceKindAllowed(ref.Kind) {
		return d.NewError(d.ErrorCodeInvalidArgument, "unsupported documentary note subject")
	}
	if criterionID != nil {
		_, _, _, decisions, err := documentaryRepositories(u)
		if err != nil {
			return err
		}
		return validateEvidenceLinkTarget(ctx, u, decisions, ref, criterionID)
	}
	repo, ok := u.Outcomes().(ports.ReferenceSearchRepository)
	if !ok {
		return d.NewError(d.ErrorCodeInvalidConfig, "scoped subject resolution is unavailable")
	}
	records, err := repo.SearchReferences(ctx, ref.Scope, ports.ReferenceFilter{Kinds: []d.EntityKind{ref.Kind}, ID: ref.ID, Limit: 1})
	if err != nil {
		return err
	}
	if len(records) != 1 || records[0].Ref != ref {
		return d.NewError(d.ErrorCodeNotFound, "documentary note subject not found")
	}
	return nil
}

func validateEntityNoteReferences(ctx context.Context, u ports.UnitOfWork, n d.EntityNote) error {
	artifacts, evidence, _, _, err := documentaryRepositories(u)
	if err != nil {
		return err
	}
	// Existing withdrawn/retracted records remain legitimate historical references;
	// referencing them is documentary and never turns them into current proof.
	for _, id := range n.ArtifactRefs {
		if _, err = artifacts.Get(ctx, n.Scope, id); err != nil {
			return err
		}
	}
	for _, id := range n.EvidenceRefs {
		if _, err = evidence.Get(ctx, n.Scope, id); err != nil {
			return err
		}
	}
	return nil
}

func validateEntityNoteSource(ctx context.Context, u ports.UnitOfWork, scope d.Scope, source d.SourceReference) (*d.EntityNote, error) {
	if source.Provider != d.EntityNoteSourceProvider {
		return nil, nil
	}
	id, err := d.ParseID(source.ID)
	if err != nil {
		return nil, d.NewError(d.ErrorCodeEvidence, "internal observation source requires a note identity")
	}
	repo, err := entityNotesRepository(u)
	if err != nil {
		return nil, err
	}
	note, err := repo.Get(ctx, scope, id)
	if err != nil {
		return nil, err
	}
	if note.Type != d.EntityNoteObservation {
		return nil, d.NewError(d.ErrorCodeEvidence, "internal observation source must identify a recorded observation")
	}
	return &note, nil
}

func (s *Service) GetEntityNote(ctx context.Context, scope d.Scope, id d.ID) (d.EntityNote, error) {
	if err := scope.Validate(); err != nil {
		return d.EntityNote{}, err
	}
	if err := id.Validate(); err != nil {
		return d.EntityNote{}, err
	}
	if err := s.authorizeScopedRead(ctx, scope); err != nil {
		return d.EntityNote{}, err
	}
	u, err := s.tx.Begin(ctx)
	if err != nil {
		return d.EntityNote{}, err
	}
	defer u.Rollback()
	if err = s.authorizeScopedReadInUnitOfWork(ctx, u, scope); err != nil {
		return d.EntityNote{}, err
	}
	if err = lockExistingOutcome(ctx, u, scope); err != nil {
		return d.EntityNote{}, err
	}
	repo, err := entityNotesRepository(u)
	if err != nil {
		return d.EntityNote{}, err
	}
	return repo.Get(ctx, scope, id)
}

// entityNoteEvents is an event factory, not another transaction. Compound
// documentary commands reuse the same note payload with their own event index.
func entityNoteEvents(s *Service, cc d.CommandContext, meta commandMetadata, note d.EntityNote, revision d.OutcomeRevision) ([]d.DomainEvent, error) {
	events, err := buildCompoundEvents(s, cc, meta, revision, []compoundEventSpec{{eventType: "entity_note.recorded", ref: note.SubjectRef}})
	if err != nil {
		return nil, err
	}
	events[0].Payload, err = json.Marshal(note)
	if err != nil {
		return nil, err
	}
	events[0].RecordedAt = note.RecordedAt
	return events, events[0].Validate()
}
