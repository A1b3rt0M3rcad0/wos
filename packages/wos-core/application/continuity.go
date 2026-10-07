package application

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const MaxQueryLimit = 100
const MaxSnapshotBytes = 256 << 10

func queryLimit(n int) (int, error) {
	if n == 0 {
		return 25, nil
	}
	if n < 1 || n > MaxQueryLimit {
		return 0, domain.NewError(domain.ErrorCodeInvalidArgument, "limit must be between 1 and 100")
	}
	return n, nil
}

type queryCursor struct {
	Namespace   domain.ID              `json:"ns"`
	Outcome     domain.ID              `json:"outcome,omitempty"`
	Section     string                 `json:"section"`
	Filter      string                 `json:"filter,omitempty"`
	Revision    domain.OutcomeRevision `json:"revision,omitempty"`
	Index       uint32                 `json:"index,omitempty"`
	Key         string                 `json:"key,omitempty"`
	EvaluatedAt time.Time              `json:"evaluated_at,omitempty"`
}

func filterHash(v any) string {
	b, _ := json.Marshal(v)
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
func encodeCursor(c queryCursor) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}
func decodeCursor(raw string, ns, outcome domain.ID, section, filter string) (queryCursor, error) {
	var c queryCursor
	if len(raw) > 4096 {
		return c, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid_cursor")
	}
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || json.Unmarshal(b, &c) != nil || c.Namespace != ns || c.Outcome != outcome || c.Section != section || c.Filter != filter {
		return c, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid_cursor: scope, section or filter mismatch")
	}
	return c, nil
}

type OutcomePage struct {
	Items       []ports.OutcomeIndexEntry `json:"items"`
	NextCursor  string                    `json:"next_cursor,omitempty"`
	Consistency string                    `json:"consistency"`
	Limit       int                       `json:"limit"`
}

func (s *Service) SearchOutcomes(ctx context.Context, ns domain.ID, f ports.OutcomeFilter, limit int, cursor string) (OutcomePage, error) {
	if err := ns.Validate(); err != nil {
		return OutcomePage{}, err
	}
	if err := s.authorizeRead(ctx, ns); err != nil {
		return OutcomePage{}, err
	}
	if err := f.ExternalContext.Validate(); err != nil {
		return OutcomePage{}, err
	}
	if len(f.CreatorPrincipalID) > 256 {
		return OutcomePage{}, domain.NewError(domain.ErrorCodeInvalidArgument, "creator filter too long")
	}
	n, err := queryLimit(limit)
	if err != nil {
		return OutcomePage{}, err
	}
	if f.ExternalProvider != "" || f.ExternalKind != "" || f.ExternalID != "" {
		if err := (domain.ExternalReference{Provider: f.ExternalProvider, Kind: f.ExternalKind, ExternalID: f.ExternalID}).Validate(); err != nil {
			return OutcomePage{}, err
		}
	}
	if f.Lifecycle != "" && !domain.OutcomeLifecycle(f.Lifecycle).Valid() {
		return OutcomePage{}, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid lifecycle filter")
	}
	if len(f.Text) > 512 {
		return OutcomePage{}, domain.NewError(domain.ErrorCodeInvalidArgument, "search text exceeds 512 bytes")
	}
	if f.Priority != "" && !f.Priority.Valid() {
		return OutcomePage{}, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid priority")
	}
	if f.Owner != nil {
		if err := f.Owner.Validate(); err != nil {
			return OutcomePage{}, err
		}
	}
	hash := filterHash(f)
	if cursor != "" {
		c, err := decodeCursor(cursor, ns, "", "outcomes", hash)
		if err != nil {
			return OutcomePage{}, err
		}
		parts := strings.SplitN(c.Key, "/", 2)
		if len(parts) != 2 {
			return OutcomePage{}, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid_cursor")
		}
		f.AfterCreated, err = time.Parse(time.RFC3339Nano, parts[0])
		if err != nil {
			return OutcomePage{}, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid_cursor")
		}
		f.AfterID, err = domain.ParseID(parts[1])
		if err != nil {
			return OutcomePage{}, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid_cursor")
		}
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return OutcomePage{}, err
	}
	defer uow.Rollback()
	repo, ok := uow.Outcomes().(ports.OutcomeDiscoveryRepository)
	if !ok {
		return OutcomePage{}, domain.NewError(domain.ErrorCodeInvalidConfig, "outcome discovery is unavailable")
	}
	f.Limit = n + 1
	items, err := repo.ListOutcomes(ctx, ns, f)
	if err != nil {
		return OutcomePage{}, err
	}
	for i := range items {
		items[i].Title = truncateText(items[i].Title, 512)
		items[i].Description = truncateText(items[i].Description, 512)
	}
	page := OutcomePage{Items: items, Limit: n, Consistency: "live_keyset"}
	if len(items) > n {
		page.Items = items[:n]
		last := page.Items[n-1]
		page.NextCursor = encodeCursor(queryCursor{Namespace: ns, Section: "outcomes", Filter: hash, Key: last.CreatedAt.Format(time.RFC3339Nano) + "/" + last.ID.String()})
	}
	return page, nil
}

type TimelineQuery struct {
	PrincipalID string    `json:"principal_id,omitempty"`
	CommandID   domain.ID `json:"command_id,omitempty"`
	EntityID    domain.ID `json:"entity_id,omitempty"`
	EventType   string    `json:"event_type,omitempty"`
	Limit       int       `json:"-"`
	Cursor      string    `json:"-"`
}

// TimelineFact is a public history envelope. Internal command DTOs are deliberately not its payload contract.
type TimelineFact struct {
	ID            domain.ID              `json:"event_id"`
	Type          string                 `json:"event_type"`
	SchemaVersion uint32                 `json:"schema_version"`
	Revision      domain.OutcomeRevision `json:"outcome_revision"`
	Index         uint32                 `json:"event_index"`
	Entity        domain.EntityRef       `json:"entity_ref"`
	PrincipalID   string                 `json:"principal_id"`
	Actor         domain.ActorRef        `json:"actor_ref"`
	CommandID     domain.ID              `json:"command_id"`
	RecordedAt    time.Time              `json:"recorded_at"`
}
type TimelinePage struct {
	Items           []TimelineFact         `json:"items"`
	NextCursor      string                 `json:"next_cursor,omitempty"`
	OutcomeRevision domain.OutcomeRevision `json:"outcome_revision"`
	Limit           int                    `json:"limit"`
	Consistency     string                 `json:"consistency"`
}

func (s *Service) GetTimeline(ctx context.Context, scope domain.Scope, q TimelineQuery) (result TimelinePage, queryErr error) {
	defer s.observeQuery(ctx, "timeline", scope, &result.OutcomeRevision, &queryErr, time.Now())
	if err := scope.Validate(); err != nil {
		return TimelinePage{}, err
	}
	if err := s.authorizeRead(ctx, scope.NamespaceID); err != nil {
		return TimelinePage{}, err
	}
	n, err := queryLimit(q.Limit)
	if err != nil {
		return TimelinePage{}, err
	}
	hash := filterHash(q)
	f := ports.EventFilter{Limit: n + 1, PrincipalID: q.PrincipalID, CommandID: q.CommandID, EntityID: q.EntityID, EventType: q.EventType}
	if q.Cursor != "" {
		c, err := decodeCursor(q.Cursor, scope.NamespaceID, scope.OutcomeID, "timeline", hash)
		if err != nil {
			return TimelinePage{}, err
		}
		f.AfterRevision = c.Revision
		f.AfterIndex = c.Index
	}
	uow, err := s.tx.Begin(ctx)
	if err != nil {
		return TimelinePage{}, err
	}
	defer uow.Rollback()
	if err := lockExistingOutcome(ctx, uow, scope); err != nil {
		return TimelinePage{}, err
	}
	repo, ok := uow.Events().(ports.TimelineRepository)
	if !ok {
		return TimelinePage{}, domain.NewError(domain.ErrorCodeInvalidConfig, "timeline is unavailable")
	}
	events, err := repo.ListEvents(ctx, scope, f)
	if err != nil {
		return TimelinePage{}, err
	}
	coord, err := uow.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		return TimelinePage{}, err
	}
	page := TimelinePage{Items: []TimelineFact{}, OutcomeRevision: coord.Revision, Limit: n, Consistency: "append_only_keyset"}
	if len(events) > n {
		events = events[:n]
		last := events[n-1]
		page.NextCursor = encodeCursor(queryCursor{Namespace: scope.NamespaceID, Outcome: scope.OutcomeID, Section: "timeline", Filter: hash, Revision: last.OutcomeRevision, Index: last.EventIndex})
	}
	for _, e := range events {
		page.Items = append(page.Items, TimelineFact{ID: e.EventID, Type: e.EventType, SchemaVersion: 1, Revision: e.OutcomeRevision, Index: e.EventIndex, Entity: e.AggregateRef, PrincipalID: e.PrincipalID, Actor: e.Actor, CommandID: e.CommandID, RecordedAt: e.RecordedAt})
	}
	return page, nil
}

type EntitySummary struct {
	Ref           domain.EntityRef `json:"ref"`
	Version       domain.Version   `json:"version,omitempty"`
	Title         string           `json:"title,omitempty"`
	Lifecycle     string           `json:"lifecycle,omitempty"`
	Priority      domain.Priority  `json:"priority,omitempty"`
	DetailOmitted bool             `json:"detail_omitted"`
}

func summary(ref domain.EntityRef, v domain.Version, title, lifecycle string, priority domain.Priority) EntitySummary {
	if len(title) > 512 {
		title = title[:512]
		for !utf8.ValidString(title) {
			title = title[:len(title)-1]
		}
	}
	return EntitySummary{Ref: ref, Version: v, Title: title, Lifecycle: lifecycle, Priority: priority, DetailOmitted: true}
}

type ActivePlan struct {
	Slot           domain.RoadmapActiveSlot `json:"slot"`
	Revision       domain.RoadmapRevision   `json:"published_revision"`
	LiveReferences []PlanLiveReference      `json:"live_references"`
}
type ProgressMetric struct {
	Numerator   int      `json:"numerator"`
	Denominator int      `json:"denominator"`
	Value       *float64 `json:"value"`
	Reason      string   `json:"reason,omitempty"`
}

func metric(n, d int) ProgressMetric {
	m := ProgressMetric{Numerator: n, Denominator: d}
	if d == 0 {
		m.Reason = "no_applicable_items"
	} else {
		v := float64(n) / float64(d)
		m.Value = &v
	}
	return m
}

type ContinuitySnapshot struct {
	SchemaVersion   int                          `json:"snapshot_schema_version"`
	OutcomeRevision domain.OutcomeRevision       `json:"outcome_revision"`
	EvaluatedAt     time.Time                    `json:"evaluated_at"`
	Consistency     string                       `json:"consistency"`
	Outcome         EntitySummary                `json:"outcome"`
	Sections        map[string][]json.RawMessage `json:"sections"`
	Counts          map[string]int               `json:"counts"`
	Omitted         map[string]int               `json:"omitted"`
	SectionCursors  map[string]string            `json:"section_cursors"`
	Truncated       bool                         `json:"truncated"`
	Limit           int                          `json:"section_limit"`
	Progress        map[string]ProgressMetric    `json:"progress"`
}
type SectionPage struct {
	Items           []json.RawMessage      `json:"items"`
	NextCursor      string                 `json:"next_cursor,omitempty"`
	OutcomeRevision domain.OutcomeRevision `json:"outcome_revision"`
	EvaluatedAt     time.Time              `json:"evaluated_at"`
	Count           int                    `json:"count"`
	Omitted         int                    `json:"omitted"`
}

func rawItems[T any](values []T) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(values))
	for _, v := range values {
		b, _ := json.Marshal(v)
		out = append(out, b)
	}
	return out
}
func (s *Service) continuitySections(state OutcomeState) map[string][]json.RawMessage {
	sections := map[string][]json.RawMessage{}
	objectives := []EntitySummary{}
	for _, v := range state.Objectives {
		objectives = append(objectives, summary(v.Ref(), v.Version, v.Title, string(v.Lifecycle), v.Priority))
	}
	sections["objectives"] = rawItems(objectives)
	work := []EntitySummary{}
	byID := map[domain.ID]domain.WorkItemOperationalState{}
	for _, v := range state.WorkItemOperationalStates {
		byID[v.Ref.ID] = v
	}
	groups := map[string][]any{}
	for _, v := range state.WorkItems {
		su := summary(v.Ref(), v.Version, v.Title, string(v.Lifecycle), v.Priority)
		work = append(work, su)
		st := byID[v.ID]
		groups[string(st.DisplayState)] = append(groups[string(st.DisplayState)], struct {
			Work  EntitySummary                   `json:"work_item"`
			State domain.WorkItemOperationalState `json:"state"`
		}{su, st})
	}
	sections["work_items"] = rawItems(work)
	for _, key := range []string{"ready", "in_progress", "blocked", "scheduled", "attention_needed", "waiting_dependencies", "waiting_scope", "backlog", "done", "cancelled"} {
		sections[key+"_work"] = rawItems(groups[key])
	}
	sections["external_references"] = rawItems(state.ExternalReferences)
	sections["external_context"] = rawItems([]domain.ExternalContext{state.Outcome.ExternalContext})
	sections["relations"] = rawItems(state.Relations)
	sections["blocking_states"] = rawItems(state.BlockingStates)
	sections["conclusion_contestations"] = rawItems(state.ConclusionContestations)
	issues := []EntitySummary{}
	for _, v := range state.Issues {
		issues = append(issues, summary(v.Ref(), v.Version, v.Title, string(v.Lifecycle), ""))
	}
	sections["issues"] = rawItems(issues)
	blockers := []EntitySummary{}
	for _, v := range state.Blockers {
		blockers = append(blockers, summary(v.Ref(), v.Version, v.Description, string(v.Lifecycle), ""))
	}
	sections["blockers"] = rawItems(blockers)
	artifacts := []EntitySummary{}
	for _, v := range state.Artifacts {
		artifacts = append(artifacts, summary(v.Ref(), v.Version, v.Name, string(v.Lifecycle), ""))
	}
	sections["artifacts"] = rawItems(artifacts)
	evidence := []EntitySummary{}
	for _, v := range state.Evidence {
		evidence = append(evidence, summary(v.Ref(), v.Version, v.Description, string(v.Lifecycle), ""))
	}
	sections["evidence"] = rawItems(evidence)
	decisions := []EntitySummary{}
	for _, v := range state.Decisions {
		decisions = append(decisions, summary(v.Ref(), v.Version, v.Title, string(v.Lifecycle), ""))
	}
	sections["decisions"] = rawItems(decisions)
	sections["evidence_links"] = rawItems(state.EvidenceLinks)
	plans := []any{}
	for _, plan := range state.ActiveRoadmaps {
		plans = append(plans, struct {
			Slot          domain.RoadmapActiveSlot `json:"slot"`
			Hash          string                   `json:"content_hash"`
			NodeCount     int                      `json:"node_count"`
			DetailOmitted bool                     `json:"detail_omitted"`
		}{plan.Slot, plan.Revision.ContentHash, len(plan.Revision.Nodes), true})
	}
	sections["active_roadmaps"] = rawItems(plans)
	livePlans := []PlanLiveReference{}
	for _, plan := range state.ActiveRoadmaps {
		livePlans = append(livePlans, plan.LiveReferences...)
	}
	sections["active_plan_references"] = rawItems(livePlans)
	for _, values := range sections {
		sort.Slice(values, func(i, j int) bool { return string(values[i]) < string(values[j]) })
	}
	return sections
}
func sectionPage(scope domain.Scope, revision domain.OutcomeRevision, at time.Time, section string, values []json.RawMessage, n int, cursor string) (SectionPage, error) {
	after := ""
	if cursor != "" {
		c, err := decodeCursor(cursor, scope.NamespaceID, scope.OutcomeID, section, "")
		if err != nil {
			return SectionPage{}, err
		}
		if c.Revision != revision {
			return SectionPage{}, domain.NewError(domain.ErrorCodePreconditionFailed, "snapshot_changed: reload continuity snapshot")
		}
		after = c.Key
	}
	// The key is a deterministic content hash; order uses stable serialized content, not an offset.
	start := 0
	if after != "" {
		found := false
		for i, v := range values {
			if filterHash(json.RawMessage(v)) == after {
				start = i + 1
				found = true
				break
			}
		}
		if !found {
			return SectionPage{}, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid_cursor: section key is missing")
		}
	}
	end := min(start+n, len(values))
	page := SectionPage{Items: values[start:end], OutcomeRevision: revision, EvaluatedAt: at, Count: len(values), Omitted: len(values) - end}
	if end < len(values) {
		page.NextCursor = encodeCursor(queryCursor{Namespace: scope.NamespaceID, Outcome: scope.OutcomeID, Section: section, Revision: revision, Key: filterHash(json.RawMessage(values[end-1])), EvaluatedAt: at})
	}
	return page, nil
}
func (s *Service) GetContinuity(ctx context.Context, scope domain.Scope, limit int) (result ContinuitySnapshot, queryErr error) {
	defer s.observeQuery(ctx, "continuity", scope, &result.OutcomeRevision, &queryErr, time.Now())
	n, err := queryLimit(limit)
	if err != nil {
		return ContinuitySnapshot{}, err
	}
	state, err := s.GetOutcomeState(ctx, scope)
	if err != nil {
		return ContinuitySnapshot{}, err
	}
	all := s.continuitySections(state)
	for {
		snapshot := ContinuitySnapshot{SchemaVersion: 1, OutcomeRevision: state.OutcomeRevision, EvaluatedAt: state.EvaluatedAt, Consistency: "transactional", Outcome: summary(state.Outcome.Ref(), state.Outcome.Version, state.Outcome.Title, string(state.Outcome.Lifecycle), state.Outcome.Priority), Sections: map[string][]json.RawMessage{}, Counts: map[string]int{}, Omitted: map[string]int{}, SectionCursors: map[string]string{}, Limit: n, Progress: map[string]ProgressMetric{}}
		for section, values := range all {
			page, _ := sectionPage(scope, state.OutcomeRevision, state.EvaluatedAt, section, values, n, "")
			snapshot.Sections[section] = page.Items
			snapshot.Counts[section] = page.Count
			snapshot.Omitted[section] = page.Omitted
			if page.NextCursor != "" {
				snapshot.SectionCursors[section] = page.NextCursor
				snapshot.Truncated = true
			}
		}
		wd, wn, od, on, rd, rn := 0, 0, 0, 0, 0, 0
		for _, v := range state.WorkItems {
			if v.Lifecycle != domain.WorkItemLifecycleCancelled {
				wd++
				if v.Lifecycle == domain.WorkItemLifecycleDone {
					wn++
				}
			}
		}
		for _, v := range state.Objectives {
			if v.Lifecycle != domain.ObjectiveLifecycleCancelled {
				od++
				if v.Lifecycle == domain.ObjectiveLifecycleAchieved {
					on++
				}
			}
			if v.RequiredForOutcome {
				rd++
				if v.Lifecycle == domain.ObjectiveLifecycleAchieved {
					rn++
				}
			}
		}
		for name, value := range proofProgress(state) {
			snapshot.Progress[name] = value
		}
		snapshot.Progress["work_completion"] = metric(wn, wd)
		snapshot.Progress["objective_completion"] = metric(on, od)
		snapshot.Progress["required_objectives_completion"] = metric(rn, rd)
		b, err := json.Marshal(snapshot)
		if err != nil {
			return ContinuitySnapshot{}, err
		}
		if len(b) <= MaxSnapshotBytes {
			return snapshot, nil
		}
		if n == 1 {
			return ContinuitySnapshot{}, domain.NewError(domain.ErrorCodeGraphLimitExceeded, "snapshot exceeds byte limit; use section queries")
		}
		n = max(1, n/2)
	}
}
func (s *Service) GetContinuitySection(ctx context.Context, scope domain.Scope, section string, limit int, cursor string) (result SectionPage, queryErr error) {
	defer s.observeQuery(ctx, "continuity_section", scope, &result.OutcomeRevision, &queryErr, time.Now())
	n, err := queryLimit(limit)
	if err != nil {
		return SectionPage{}, err
	}
	if cursor != "" {
		c, err := decodeCursor(cursor, scope.NamespaceID, scope.OutcomeID, section, "")
		if err != nil {
			return SectionPage{}, err
		}
		if c.EvaluatedAt.IsZero() || c.EvaluatedAt.After(s.clock.Now()) {
			return SectionPage{}, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid_cursor: evaluation time")
		}
		ctx = context.WithValue(ctx, evaluationKey{}, c.EvaluatedAt)
	}
	state, err := s.GetOutcomeState(ctx, scope)
	if err != nil {
		return SectionPage{}, err
	}
	values, ok := s.continuitySections(state)[section]
	if !ok {
		return SectionPage{}, domain.NewError(domain.ErrorCodeInvalidArgument, "unknown continuity section")
	}
	page, err := sectionPage(scope, state.OutcomeRevision, state.EvaluatedAt, section, values, n, cursor)
	if err != nil {
		return SectionPage{}, err
	}
	b, err := json.Marshal(page)
	if err != nil {
		return SectionPage{}, err
	}
	if len(b) > MaxSnapshotBytes {
		return SectionPage{}, domain.NewError(domain.ErrorCodeGraphLimitExceeded, "section exceeds byte limit; reduce limit")
	}
	return page, nil
}

type evaluationKey struct{}

func evaluationTime(ctx context.Context, now time.Time) time.Time {
	if at, ok := ctx.Value(evaluationKey{}).(time.Time); ok {
		return at.UTC()
	}
	return now.UTC()
}

func truncateText(s string, limit int) string {
	if len(s) <= limit {
		return s
	}
	s = s[:limit]
	for !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}

// Proof metrics remain distinct from work completion and lifecycle certification.
func proofProgress(state OutcomeState) map[string]ProgressMetric {
	evidence := map[domain.ID]bool{}
	for _, v := range state.Evidence {
		evidence[v.ID] = v.Lifecycle == domain.EvidenceLifecycleRegistered
	}
	required, met, waived := 0, 0, 0
	sets := []domain.CriterionSet{state.Outcome.Criteria}
	for _, v := range state.Objectives {
		if v.Lifecycle != domain.ObjectiveLifecycleCancelled {
			sets = append(sets, v.Criteria)
		}
	}
	for _, v := range state.WorkItems {
		if v.Lifecycle != domain.WorkItemLifecycleCancelled {
			sets = append(sets, v.Criteria)
		}
	}
	for _, set := range sets {
		for _, criterion := range set.Items {
			if !criterion.Required || criterion.Status != domain.CriterionStatusActive {
				continue
			}
			required++
			assessment, ok := set.CurrentAssessments[criterion.ID]
			if !ok || assessment.CriterionRevision != criterion.Revision {
				continue
			}
			if assessment.Result == domain.AssessmentResultWaived {
				waived++
				continue
			}
			if assessment.Result != domain.AssessmentResultMet {
				continue
			}
			usable := true
			for _, id := range assessment.EvidenceIDs {
				if !evidence[id] {
					usable = false
					break
				}
			}
			if usable {
				met++
			}
		}
	}
	return map[string]ProgressMetric{"required_criteria_met": metric(met, required), "required_criteria_waived": metric(waived, required)}
}
