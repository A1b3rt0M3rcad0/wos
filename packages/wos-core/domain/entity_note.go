package domain

import (
	"encoding/json"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	MaxEntityNoteBodyBytes  = 16 << 10
	MaxEntityNoteReferences = 50
	// EntityNoteSourceProvider identifies an application-verified local observation.
	// A provider name alone is never proof that the referenced note exists.
	EntityNoteSourceProvider = "wos.entity-note"
)

type EntityNoteType string

const (
	EntityNoteUpdate      EntityNoteType = "update"
	EntityNoteDiscovery   EntityNoteType = "discovery"
	EntityNoteObservation EntityNoteType = "observation"
)

func (t EntityNoteType) Valid() bool {
	return t == EntityNoteUpdate || t == EntityNoteDiscovery || t == EntityNoteObservation
}

// EntityNote is an immutable documentary declaration. It grants no execution
// authority and does not change lifecycle, criteria, assessments or conclusions.
// Corrections append another note; the original declaration remains unchanged.
type EntityNote struct {
	ID               ID              `json:"id"`
	Scope            Scope           `json:"scope"`
	SubjectRef       EntityRef       `json:"subject_ref"`
	CriterionID      *ID             `json:"criterion_id,omitempty"`
	Type             EntityNoteType  `json:"type"`
	Body             string          `json:"body"`
	PrincipalID      string          `json:"principal_id"`
	Actor            ActorRef        `json:"actor_ref"`
	RecordedAt       time.Time       `json:"recorded_at"`
	OccurredAt       *time.Time      `json:"occurred_at,omitempty"`
	ArtifactRefs     []ID            `json:"artifact_refs,omitempty"`
	EvidenceRefs     []ID            `json:"evidence_refs,omitempty"`
	SupersedesNoteID *ID             `json:"supersedes_note_id,omitempty"`
	OutcomeRevision  OutcomeRevision `json:"outcome_revision,string"`
}

func (n EntityNote) Validate() error {
	if err := n.ID.Validate(); err != nil {
		return err
	}
	if err := n.Scope.Validate(); err != nil {
		return err
	}
	if err := n.SubjectRef.Validate(); err != nil {
		return err
	}
	if n.SubjectRef.Scope != n.Scope {
		return NewError(ErrorCodeInvalidScope, "note subject must belong to its Outcome")
	}
	if n.SubjectRef.Kind == EntityKindOutcome && n.SubjectRef.ID != n.Scope.OutcomeID {
		return NewError(ErrorCodeInvalidScope, "note Outcome subject must match its scope")
	}
	if n.SubjectRef.Kind == EntityKindTrigger {
		return NewError(ErrorCodeInvalidArgument, "integration configuration cannot own a workspace note")
	}
	if n.CriterionID != nil {
		if err := n.CriterionID.Validate(); err != nil {
			return err
		}
		if n.SubjectRef.Kind != EntityKindOutcome && n.SubjectRef.Kind != EntityKindObjective && n.SubjectRef.Kind != EntityKindWorkItem {
			return NewError(ErrorCodeInvalidArgument, "only Outcome, Objective and Task can own criterion notes")
		}
	}
	if !n.Type.Valid() || strings.TrimSpace(n.Body) == "" || len(n.Body) > MaxEntityNoteBodyBytes || !utf8.ValidString(n.Body) || strings.ContainsRune(n.Body, 0) {
		return NewError(ErrorCodeInvalidArgument, "note requires a documentary type and valid text up to 16 KiB")
	}
	if strings.TrimSpace(n.PrincipalID) == "" || len(n.PrincipalID) > 256 {
		return NewError(ErrorCodeInvalidArgument, "note requires its authenticated Principal")
	}
	if err := n.Actor.Validate(); err != nil {
		return err
	}
	if n.RecordedAt.IsZero() || n.OutcomeRevision == 0 {
		return NewError(ErrorCodeInvalidArgument, "note requires server recording time and commit revision")
	}
	if n.OccurredAt != nil && n.OccurredAt.IsZero() {
		return NewError(ErrorCodeInvalidArgument, "note observation time cannot be zero")
	}
	if n.SupersedesNoteID != nil {
		if err := n.SupersedesNoteID.Validate(); err != nil {
			return err
		}
		if *n.SupersedesNoteID == n.ID {
			return NewError(ErrorCodeInvalidArgument, "a note cannot correct itself")
		}
	}
	for _, refs := range [][]ID{n.ArtifactRefs, n.EvidenceRefs} {
		if len(refs) > MaxEntityNoteReferences {
			return NewError(ErrorCodeInvalidArgument, "note references exceed the limit of 50 per kind")
		}
		seen := map[ID]bool{}
		for _, id := range refs {
			if err := id.Validate(); err != nil {
				return err
			}
			if seen[id] {
				return NewError(ErrorCodeInvalidArgument, "note references must be unique")
			}
			seen[id] = true
		}
	}
	raw, err := json.Marshal(n)
	if err != nil {
		return err
	}
	if len(raw) > 128<<10 {
		return NewError(ErrorCodeInvalidArgument, "complete documentary note exceeds 128 KiB")
	}
	return nil
}

// ValidateCorrection checks provenance without editing either note. Author means
// authenticated Principal, not a client-declared ActorRef or a write permission.
func (n EntityNote) ValidateCorrection(previous EntityNote) error {
	sameCriterion := n.CriterionID == nil && previous.CriterionID == nil || n.CriterionID != nil && previous.CriterionID != nil && *n.CriterionID == *previous.CriterionID
	if n.SupersedesNoteID == nil || *n.SupersedesNoteID != previous.ID || n.Scope != previous.Scope || n.SubjectRef != previous.SubjectRef || !sameCriterion {
		return NewError(ErrorCodeInvalidScope, "a correction must retain the original note subject and criterion")
	}
	if n.PrincipalID != previous.PrincipalID {
		return NewError(ErrorCodeForbidden, "only the original Principal can correct a note; append a separate response instead")
	}
	if n.OutcomeRevision < previous.OutcomeRevision {
		return NewError(ErrorCodeInvalidArgument, "a correction cannot precede its original note")
	}
	return nil
}
