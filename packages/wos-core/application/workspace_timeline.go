package application

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const MaxWorkspaceTimelineLimit = 20
const MaxWorkspaceTimelinePreviewBytes = 512

type WorkspaceTimelineQuery struct {
	SubjectRef         *d.EntityRef `json:"subject_ref,omitempty"`
	CriterionID        *d.ID        `json:"criterion_id,omitempty"`
	IncludeDescendants bool         `json:"include_descendants,omitempty"`
	Limit              int          `json:"-"`
	Cursor             string       `json:"-"`
}
type WorkspaceTimelineSource struct {
	Kind         string `json:"kind"`
	ID           d.ID   `json:"id"`
	ContractID   *d.ID  `json:"contract_id,omitempty"`
	SubmissionID *d.ID  `json:"submission_id,omitempty"`
	ReviewCaseID *d.ID  `json:"review_case_id,omitempty"`
}
type WorkspaceTimelineEntry struct {
	ID               d.ID                         `json:"id"`
	EventID          d.ID                         `json:"event_id"`
	Origin           string                       `json:"origin"`
	Source           WorkspaceTimelineSource      `json:"source"`
	EventType        string                       `json:"event_type"`
	RecordType       string                       `json:"record_type,omitempty"`
	SubjectRef       d.EntityRef                  `json:"subject_ref"`
	CriterionID      *d.ID                        `json:"criterion_id,omitempty"`
	PrincipalID      string                       `json:"principal_id"`
	Actor            d.ActorRef                   `json:"actor_ref"`
	Description      string                       `json:"description,omitempty"`
	PreviewTruncated bool                         `json:"preview_truncated"`
	RecordedAt       time.Time                    `json:"recorded_at"`
	OccurredAt       *time.Time                   `json:"occurred_at,omitempty"`
	OutcomeRevision  d.OutcomeRevision            `json:"outcome_revision,string"`
	EventIndex       uint32                       `json:"event_index"`
	CommandID        d.ID                         `json:"command_id"`
	CorrelationID    string                       `json:"correlation_id,omitempty"`
	CausationID      *d.ID                        `json:"causation_id,omitempty"`
	ArtifactRefs     []d.ID                       `json:"artifact_refs,omitempty"`
	EvidenceRefs     []d.ID                       `json:"evidence_refs,omitempty"`
	CriterionRefs    []d.WorkActivityCriterionRef `json:"criterion_refs,omitempty"`
	SupersedesNoteID *d.ID                        `json:"supersedes_note_id,omitempty"`
	DescendantOrigin bool                         `json:"descendant_origin"`
	ContextOrigin    string                       `json:"context_origin"`
}
type WorkspaceTimelinePage struct {
	Items                  []WorkspaceTimelineEntry `json:"items"`
	NextCursor             string                   `json:"next_cursor,omitempty"`
	OutcomeRevision        d.OutcomeRevision        `json:"outcome_revision,string"`
	HistoryThroughRevision d.OutcomeRevision        `json:"history_through_revision,string"`
	EvaluatedAt            time.Time                `json:"evaluated_at"`
	Limit                  int                      `json:"limit"`
	Order                  string                   `json:"order"`
	ScopeBasis             string                   `json:"scope_basis"`
	HasNewerEntries        bool                     `json:"has_newer_entries"`
}

func (s *Service) GetWorkspaceTimeline(ctx context.Context, scope d.Scope, q WorkspaceTimelineQuery) (WorkspaceTimelinePage, error) {
	frame, handled, err := s.openWorkspaceFrame(ctx, scope)
	if err != nil {
		return WorkspaceTimelinePage{}, err
	}
	if !handled {
		return WorkspaceTimelinePage{}, d.NewError(d.ErrorCodeInvalidConfig, "coherent workspace history is unavailable")
	}
	defer frame.u.Rollback()
	return s.workspaceTimelineInUnitOfWork(ctx, frame.u, scope, q, frame.revision, frame.at)
}

// workspaceTimelineInUnitOfWork uses the caller's authorized coherent frame.
// Current placement defines descendant membership; structure changes invalidate
// that membership cursor. Ordinary new facts never enter its frozen frontier.
func (s *Service) workspaceTimelineInUnitOfWork(ctx context.Context, u ports.UnitOfWork, scope d.Scope, q WorkspaceTimelineQuery, revision d.OutcomeRevision, at time.Time) (WorkspaceTimelinePage, error) {
	var zero WorkspaceTimelinePage
	subject := d.EntityRef{Scope: scope, Kind: d.EntityKindOutcome, ID: scope.OutcomeID}
	if q.SubjectRef != nil {
		subject = *q.SubjectRef
	}
	if subject.Scope != scope {
		return zero, d.NewError(d.ErrorCodeInvalidScope, "timeline subject belongs to another Outcome")
	}
	if err := validateNoteSubject(ctx, u, subject, q.CriterionID); err != nil {
		return zero, err
	}
	if q.IncludeDescendants && subject.Kind != d.EntityKindOutcome && subject.Kind != d.EntityKindObjective {
		return zero, d.NewError(d.ErrorCodeInvalidArgument, "only Outcome and Objective have operational descendants")
	}
	n := q.Limit
	if n == 0 {
		n = MaxWorkspaceTimelineLimit
	}
	if n < 1 || n > MaxWorkspaceTimelineLimit {
		return zero, d.NewError(d.ErrorCodeInvalidArgument, "workspace history limit must be between 1 and 20")
	}
	repo, ok := u.Events().(ports.WorkspaceTimelineRepository)
	if !ok {
		return zero, d.NewError(d.ErrorCodeInvalidConfig, "workspace history repository unavailable")
	}
	identity, _ := IdentityFromContext(ctx)
	fingerprint := filterHash(struct {
		Subject     d.EntityRef
		Criterion   *d.ID
		Descendants bool
		Principal   string
		Credential  string
		Policy      d.Version
	}{subject, q.CriterionID, q.IncludeDescendants, identity.PrincipalID, identity.CredentialID.String(), identity.CredentialPolicyRevision})
	filter := ports.WorkspaceTimelineFilter{SubjectRef: subject, CriterionID: q.CriterionID, IncludeDescendants: q.IncludeDescendants, ThroughRevision: revision, Limit: n + 1}
	if q.Cursor != "" {
		cursor, err := decodeCursor(q.Cursor, scope.NamespaceID, scope.OutcomeID, "workspace_timeline", fingerprint)
		if err != nil {
			return zero, err
		}
		if cursor.CursorVersion != 1 || cursor.Revision == 0 || cursor.Revision > revision || cursor.EvaluatedAt.IsZero() || cursor.EvaluatedAt.After(at) {
			return zero, d.NewError(d.ErrorCodeInvalidArgument, "invalid_cursor: workspace history frontier")
		}
		before, err := strconv.ParseUint(cursor.Key, 10, 64)
		if err != nil || before == 0 || before > uint64(cursor.Revision) {
			return zero, d.NewError(d.ErrorCodeInvalidArgument, "invalid_cursor: workspace history key")
		}
		filter.ThroughRevision = cursor.Revision
		filter.BeforeRevision = d.OutcomeRevision(before)
		filter.BeforeIndex = cursor.Index
		if q.IncludeDescendants {
			changed, err := repo.WorkspaceStructureChanged(ctx, scope, cursor.Revision)
			if err != nil {
				return zero, err
			}
			if changed {
				return zero, d.NewError(d.ErrorCodePreconditionFailed, "timeline_structure_changed: refresh history to use current placement")
			}
		}
		at = cursor.EvaluatedAt
	}
	events, err := repo.ListWorkspaceEvents(ctx, scope, filter)
	if err != nil {
		return zero, err
	}
	page := WorkspaceTimelinePage{Items: []WorkspaceTimelineEntry{}, OutcomeRevision: revision, HistoryThroughRevision: filter.ThroughRevision, EvaluatedAt: at, Limit: n, Order: "newest_first", ScopeBasis: "current_placement", HasNewerEntries: revision > filter.ThroughRevision}
	if revision > filter.ThroughRevision {
		newerFilter := filter
		newerFilter.ThroughRevision = revision
		newerFilter.BeforeRevision = 0
		newerFilter.BeforeIndex = 0
		newerFilter.Limit = 1
		newer, err := repo.ListWorkspaceEvents(ctx, scope, newerFilter)
		if err != nil {
			return zero, err
		}
		page.HasNewerEntries = len(newer) > 0 && newer[0].OutcomeRevision > filter.ThroughRevision
	}
	more := len(events) > n
	if more {
		events = events[:n]
	}
	for _, event := range events {
		entry, err := workspaceTimelineEntry(event, subject)
		if err != nil {
			return zero, err
		}
		page.Items = append(page.Items, entry)
	}
	for {
		if more && len(page.Items) > 0 {
			last := events[len(page.Items)-1]
			page.NextCursor = encodeCursor(queryCursor{CursorVersion: 1, Namespace: scope.NamespaceID, Outcome: scope.OutcomeID, Section: "workspace_timeline", Filter: fingerprint, Revision: filter.ThroughRevision, Index: last.EventIndex, Key: strconv.FormatUint(uint64(last.OutcomeRevision), 10), EvaluatedAt: at})
		}
		raw, err := json.Marshal(page)
		if err != nil {
			return zero, err
		}
		if len(raw) <= MaxSnapshotBytes {
			break
		}
		if len(page.Items) <= 1 {
			return zero, d.NewError(d.ErrorCodeGraphLimitExceeded, "workspace timeline entry exceeds the bounded response; read its canonical record")
		}
		page.Items = page.Items[:len(page.Items)-1]
		more = true
	}

	return page, nil
}

func workspaceTimelineEntry(event d.DomainEvent, selected d.EntityRef) (WorkspaceTimelineEntry, error) {
	entry := WorkspaceTimelineEntry{ID: event.EventID, EventID: event.EventID, Origin: "domain_event", Source: WorkspaceTimelineSource{Kind: "domain_event", ID: event.EventID}, EventType: event.EventType, SubjectRef: event.AggregateRef, PrincipalID: event.PrincipalID, Actor: event.Actor, RecordedAt: event.RecordedAt, OutcomeRevision: event.OutcomeRevision, EventIndex: event.EventIndex, CommandID: event.CommandID, CorrelationID: event.CorrelationID, CausationID: cloneIDPointer(event.CausationID)}
	description := ""
	switch event.EventType {
	case "entity_note.recorded":
		var note d.EntityNote
		if err := json.Unmarshal(event.Payload, &note); err != nil {
			return entry, err
		}
		if err := note.Validate(); err != nil {
			return entry, err
		}
		if note.Scope != event.Scope() || note.SubjectRef != event.AggregateRef || note.OutcomeRevision != event.OutcomeRevision || note.PrincipalID != event.PrincipalID || note.Actor != event.Actor {
			return entry, d.NewError(d.ErrorCodeInvalidEvent, "note history provenance differs from its event")
		}
		entry.ID = note.ID
		entry.Origin = "entity_note"
		entry.Source = WorkspaceTimelineSource{Kind: "entity_note", ID: note.ID}
		entry.RecordType = string(note.Type)
		entry.CriterionID = cloneIDPointer(note.CriterionID)
		entry.OccurredAt = note.OccurredAt
		entry.ArtifactRefs = note.ArtifactRefs
		entry.EvidenceRefs = note.EvidenceRefs
		entry.SupersedesNoteID = cloneIDPointer(note.SupersedesNoteID)
		description = note.Body
	case "work_activity.recorded":
		var activity d.WorkActivity
		if err := json.Unmarshal(event.Payload, &activity); err != nil {
			return entry, err
		}
		if err := activity.Validate(); err != nil {
			return entry, err
		}
		if activity.Scope != event.Scope() || activity.WorkItemID != event.AggregateRef.ID || activity.Sequence != uint64(event.OutcomeRevision) || activity.PrincipalID != event.PrincipalID || activity.Actor != event.Actor {
			return entry, d.NewError(d.ErrorCodeInvalidEvent, "contracted activity provenance differs from its event")
		}
		entry.ID = activity.ID
		entry.Origin = "work_activity"
		entry.Source = WorkspaceTimelineSource{Kind: "work_activity", ID: activity.ID, ContractID: cloneIDPointer(&activity.ContractID)}
		entry.RecordType = string(activity.Type)
		entry.ArtifactRefs = activity.ArtifactRefs
		entry.EvidenceRefs = activity.EvidenceRefs
		entry.CriterionRefs = activity.CriterionRefs
		if len(activity.CriterionRefs) == 1 {
			entry.CriterionID = cloneIDPointer(&activity.CriterionRefs[0].CriterionID)
		}
		description = activity.Description
	default:
		// Only explicitly named documentary fields are previewed. Never expose raw
		// command/authority payloads, private signing material or arbitrary metadata.
		var fields map[string]json.RawMessage
		if json.Unmarshal(event.Payload, &fields) == nil {
			for _, name := range []string{"description", "Description", "result_summary", "ResultSummary", "reason", "Reason", "rationale", "Rationale", "resolution_summary", "ResolutionSummary", "title", "Title"} {
				if value, ok := fields[name]; ok && json.Unmarshal(value, &description) == nil && description != "" {
					break
				}
			}
			for _, name := range []string{"criterion_id", "CriterionID"} {
				if value, ok := fields[name]; ok {
					var id d.ID
					if json.Unmarshal(value, &id) == nil && id.Validate() == nil {
						entry.CriterionID = &id
						break
					}
				}
			}
			for _, field := range []struct {
				names  []string
				target **d.ID
			}{{[]string{"contract_id", "ContractID"}, &entry.Source.ContractID}, {[]string{"submission_id", "SubmissionID"}, &entry.Source.SubmissionID}, {[]string{"review_case_id", "ReviewCaseID"}, &entry.Source.ReviewCaseID}} {
				for _, name := range field.names {
					if value, ok := fields[name]; ok {
						var id d.ID
						if json.Unmarshal(value, &id) == nil && id.Validate() == nil {
							*field.target = &id
							break
						}
					}
				}
			}
		}
	}
	entry.Description, entry.PreviewTruncated = workspaceTimelinePreview(description)
	entry.ContextOrigin = "self"
	if entry.SubjectRef != selected {
		entry.ContextOrigin = "related_record"
		if (selected.Kind == d.EntityKindOutcome || selected.Kind == d.EntityKindObjective) && (entry.SubjectRef.Kind == d.EntityKindObjective || entry.SubjectRef.Kind == d.EntityKindWorkItem) {
			entry.DescendantOrigin = true
			entry.ContextOrigin = "descendant"
		}
	}
	return entry, nil
}
func workspaceTimelinePreview(text string) (string, bool) {
	text = strings.TrimSpace(text)
	if len(text) <= MaxWorkspaceTimelinePreviewBytes {
		return text, false
	}
	end := MaxWorkspaceTimelinePreviewBytes
	for !utf8.ValidString(text[:end]) {
		end--
	}
	return text[:end], true
}
