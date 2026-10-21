package application

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"time"
)

// WorkspaceObservationSource resolves documentary provenance, never truth or
// acceptance. Client source labels and URLs are not authenticated authorship.
// A full observation remains available through the authorized note resource.
type WorkspaceObservationSource struct {
	Status           string       `json:"status"`
	NoteID           *d.ID        `json:"note_id,omitempty"`
	SubjectRef       *d.EntityRef `json:"subject_ref,omitempty"`
	CriterionID      *d.ID        `json:"criterion_id,omitempty"`
	PrincipalID      string       `json:"principal_id,omitempty"`
	Actor            *d.ActorRef  `json:"actor_ref,omitempty"`
	RecordedAt       *time.Time   `json:"recorded_at,omitempty"`
	OccurredAt       *time.Time   `json:"occurred_at,omitempty"`
	Preview          string       `json:"preview,omitempty"`
	PreviewTruncated bool         `json:"preview_truncated"`
	SupersedesNoteID *d.ID        `json:"supersedes_note_id,omitempty"`
	ReasonCode       string       `json:"reason_code,omitempty"`
}

// workspaceObservationSource runs inside the caller's authorized coherent
// frame. Non-internal sources have no observation facet. Older unresolved
// references remain unavailable; no author, date or observation is invented.
func workspaceObservationSource(ctx context.Context, u ports.UnitOfWork, scope d.Scope, source d.SourceReference) (*WorkspaceObservationSource, error) {
	if source.Provider != d.EntityNoteSourceProvider {
		return nil, nil
	}
	note, err := validateEntityNoteSource(ctx, u, scope, source)
	if err != nil {
		code, _ := d.ErrorCodeOf(err)
		reason := "observation_source_invalid"
		switch code {
		case d.ErrorCodeNotFound:
			reason = "observation_source_not_found"
		case d.ErrorCodeEvidence, d.ErrorCodeInvalidID, d.ErrorCodeInvalidArgument:
		case d.ErrorCodeInvalidConfig:
			reason = "observation_source_unavailable"
		default:
			return nil, err
		}
		return &WorkspaceObservationSource{Status: "unavailable", ReasonCode: reason}, nil
	}
	if note == nil {
		return &WorkspaceObservationSource{Status: "unavailable", ReasonCode: "observation_source_unavailable"}, nil
	}
	subject, actor, recorded := note.SubjectRef, note.Actor, note.RecordedAt.UTC()
	preview, truncated := workspaceTimelinePreview(note.Body)
	result := &WorkspaceObservationSource{Status: "resolved", NoteID: cloneIDPointer(&note.ID), SubjectRef: &subject, CriterionID: cloneIDPointer(note.CriterionID), PrincipalID: note.PrincipalID, Actor: &actor, RecordedAt: &recorded, Preview: preview, PreviewTruncated: truncated, SupersedesNoteID: cloneIDPointer(note.SupersedesNoteID)}
	if note.OccurredAt != nil {
		at := note.OccurredAt.UTC()
		result.OccurredAt = &at
	}
	return result, nil
}
