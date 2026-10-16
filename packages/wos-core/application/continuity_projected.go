package application

import (
	"context"
	"encoding/hex"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"strings"
	"time"
)

var continuitySectionsOrdered = []string{"objectives", "work_items", "ready_work", "in_progress_work", "blocked_work", "scheduled_work", "attention_needed_work", "waiting_dependencies_work", "waiting_scope_work", "backlog_work", "done_work", "cancelled_work", "external_references", "external_context", "relations", "blocking_states", "conclusion_contestations", "issues", "blockers", "artifacts", "evidence", "decisions", "evidence_links", "active_roadmaps", "active_plan_references"}

type continuityReadFrame struct {
	u          ports.UnitOfWork
	repo       ports.ContinuityReadRepository
	projection ports.ContinuityProjectionRepository
	revision   d.OutcomeRevision
	at         time.Time
}

func (s *Service) openContinuityFrame(ctx context.Context, scope d.Scope) (*continuityReadFrame, bool, error) {
	if e := scope.Validate(); e != nil {
		return nil, true, e
	}
	if e := s.authorizeScopedRead(ctx, scope); e != nil {
		return nil, true, e
	}
	u, e := s.tx.Begin(ctx)
	if e != nil {
		return nil, true, e
	}
	keep := false
	defer func() {
		if !keep {
			u.Rollback()
		}
	}()
	if e = s.authorizeScopedReadInUnitOfWork(ctx, u, scope); e != nil {
		return nil, true, e
	}
	reads, ok := u.(ports.ContinuityReadUnitOfWork)
	if !ok {
		return nil, false, nil
	}
	repo := reads.ContinuityReads()
	projection, ok := repo.(ports.ContinuityProjectionRepository)
	if !ok {
		return nil, false, nil
	}
	if _, e = repo.Outcome(ctx, scope); e != nil {
		return nil, true, e
	}
	coord, e := u.Coordination().LockOutcome(ctx, scope)
	if e != nil {
		return nil, true, e
	}
	now, e := s.transactionTime(ctx, u)
	if e != nil {
		return nil, true, e
	}
	keep = true
	return &continuityReadFrame{u: u, repo: repo, projection: projection, revision: coord.Revision, at: now}, true, nil
}
func projectionState(data ports.ContinuityProjectionInputs, revision d.OutcomeRevision, at time.Time) (OutcomeState, error) {
	state, e := projectOutcomeReadState(OutcomeState{Outcome: data.Outcome, Objectives: data.Objectives, WorkItems: data.WorkItems, Relations: data.Relations, Blockers: data.Blockers, Evidence: data.Evidence, OutcomeRevision: revision, EvaluatedAt: at})
	if e != nil {
		return state, e
	}
	state.ActiveRoadmaps = []ActivePlan{}
	for _, p := range data.ActivePlans {
		state.ActiveRoadmaps = append(state.ActiveRoadmaps, ActivePlan{Slot: p.Slot, Revision: p.Revision, LiveReferences: projectPlanReferences(p.Slot, p.Revision, state.Objectives, state.WorkItems, state.WorkItemOperationalStates)})
	}
	return state, nil
}
func (s *Service) projectedContinuity(ctx context.Context, scope d.Scope, n int) (ContinuitySnapshot, bool, error) {
	var zero ContinuitySnapshot
	frame, handled, e := s.openContinuityFrame(ctx, scope)
	if !handled || e != nil {
		return zero, handled, e
	}
	defer frame.u.Rollback()
	data, e := frame.projection.Projection(ctx, scope, ports.ContinuityProjectionOptions{Operational: true, Conclusions: true, Progress: true, Plans: true})
	if e != nil {
		return zero, true, e
	}
	state, e := projectionState(data, frame.revision, frame.at)
	if e != nil {
		return zero, true, e
	}
	cached := map[string]ports.ContinuityReadPage{}
	for _, section := range continuitySectionsOrdered {
		var page ports.ContinuityReadPage
		if ports.DirectContinuitySection(section) {
			page, e = frame.repo.Page(ctx, scope, section, "", n+1)
		} else {
			page, e = derivedReadPage(state, section, "", n+1)
		}
		if e != nil {
			return zero, true, e
		}
		cached[section] = page
	}
	progress := map[string]ProgressMetric{"required_criteria_met": metric(data.MetCriteria, data.RequiredCriteria), "required_criteria_waived": metric(data.WaivedCriteria, data.RequiredCriteria)}
	wd, wn, od, on, rd, rn := 0, 0, 0, 0, 0, 0
	for _, v := range state.WorkItems {
		if v.Lifecycle != d.WorkItemLifecycleCancelled {
			wd++
			if v.Lifecycle == d.WorkItemLifecycleDone {
				wn++
			}
		}
	}
	for _, v := range state.Objectives {
		if v.Lifecycle != d.ObjectiveLifecycleCancelled {
			od++
			if v.Lifecycle == d.ObjectiveLifecycleAchieved {
				on++
			}
		}
		if v.RequiredForOutcome {
			rd++
			if v.Lifecycle == d.ObjectiveLifecycleAchieved {
				rn++
			}
		}
	}
	progress["work_completion"] = metric(wn, wd)
	progress["objective_completion"] = metric(on, od)
	progress["required_objectives_completion"] = metric(rn, rd)
	for {
		snapshot := ContinuitySnapshot{SchemaVersion: 1, OutcomeRevision: frame.revision, EvaluatedAt: frame.at, Consistency: "transactional", Outcome: summary(state.Outcome.Ref(), state.Outcome.Version, state.Outcome.Title, string(state.Outcome.Lifecycle), state.Outcome.Priority), Sections: map[string][]json.RawMessage{}, Counts: map[string]int{}, Omitted: map[string]int{}, SectionCursors: map[string]string{}, Limit: n, Progress: progress}
		tooLarge := false
		for _, section := range continuitySectionsOrdered {
			page, e := projectedSectionPage(ctx, frame, scope, section, cached[section], n)
			if e != nil {
				if code, _ := d.ErrorCodeOf(e); code == d.ErrorCodeGraphLimitExceeded && n > 1 {
					tooLarge = true
					break
				}
				return zero, true, e
			}
			snapshot.Sections[section] = page.Items
			snapshot.Counts[section] = page.Count
			snapshot.Omitted[section] = page.Omitted
			if page.NextCursor != "" {
				snapshot.SectionCursors[section] = page.NextCursor
				snapshot.Truncated = true
			}
		}
		if !tooLarge {
			raw, e := json.Marshal(snapshot)
			if e != nil {
				return zero, true, e
			}
			if len(raw) <= MaxSnapshotBytes {
				return snapshot, true, nil
			}
		}
		if n == 1 {
			return zero, true, d.NewError(d.ErrorCodeGraphLimitExceeded, "snapshot exceeds byte limit; use section queries")
		}
		n = max(1, n/2)
	}
}
func derivedOptions(section string) (ports.ContinuityProjectionOptions, bool) {
	switch section {
	case "ready_work", "in_progress_work", "blocked_work", "scheduled_work", "attention_needed_work", "waiting_dependencies_work", "waiting_scope_work", "backlog_work", "done_work", "cancelled_work", "blocking_states":
		return ports.ContinuityProjectionOptions{Operational: true}, true
	case "conclusion_contestations":
		return ports.ContinuityProjectionOptions{Conclusions: true}, true
	case "active_roadmaps", "active_plan_references":
		return ports.ContinuityProjectionOptions{Plans: true}, true
	case "external_context":
		return ports.ContinuityProjectionOptions{}, true
	}
	return ports.ContinuityProjectionOptions{}, false
}
func (s *Service) projectedContinuitySection(ctx context.Context, scope d.Scope, section string, n int, rawCursor string) (SectionPage, bool, error) {
	var zero SectionPage
	options, known := derivedOptions(section)
	if !known {
		return zero, false, nil
	}
	var cursor queryCursor
	if rawCursor != "" {
		var e error
		cursor, e = decodeCursor(rawCursor, scope.NamespaceID, scope.OutcomeID, section, "")
		if e != nil {
			return zero, true, e
		}
		if cursor.CursorVersion != 0 && cursor.CursorVersion != 2 || cursor.EvaluatedAt.IsZero() {
			return zero, true, d.NewError(d.ErrorCodeInvalidArgument, "invalid_cursor: version or evaluation time")
		}
	}
	frame, handled, e := s.openContinuityFrame(ctx, scope)
	if !handled || e != nil {
		return zero, handled, e
	}
	defer frame.u.Rollback()
	if rawCursor != "" {
		if cursor.Revision != frame.revision {
			return zero, true, d.NewError(d.ErrorCodePreconditionFailed, "snapshot_changed: reload continuity snapshot")
		}
		if cursor.EvaluatedAt.After(frame.at) {
			return zero, true, d.NewError(d.ErrorCodeInvalidArgument, "invalid_cursor: evaluation time")
		}
		frame.at = cursor.EvaluatedAt
	}
	data, e := frame.projection.Projection(ctx, scope, options)
	if e != nil {
		return zero, true, e
	}
	state, e := projectionState(data, frame.revision, frame.at)
	if e != nil {
		return zero, true, e
	}
	after := cursor.Key
	if rawCursor != "" && (cursor.CursorVersion == 0 || strings.HasPrefix(after, "h:")) {
		digest := after
		if cursor.CursorVersion == 2 {
			digest = strings.TrimPrefix(after, "h:")
		}
		value, e := hex.DecodeString(digest)
		if e != nil || len(value) != 32 {
			return zero, true, d.NewError(d.ErrorCodeInvalidArgument, "invalid_cursor: section digest")
		}
		found := false
		e = eachDerivedItem(state, section, func(key string, value any) error {
			if found {
				return nil
			}
			raw, e := json.Marshal(value)
			if e != nil {
				return e
			}
			if section == "blocking_states" {
				raw, e = enrichBlockingItems(ctx, frame, scope, []json.RawMessage{raw})
				if e != nil {
					return e
				}
			}
			if filterHash(json.RawMessage(raw)) == digest {
				after = key
				found = true
			}
			return nil
		})
		if e != nil {
			return zero, true, e
		}
		if !found {
			return zero, true, d.NewError(d.ErrorCodeInvalidArgument, "invalid_cursor: section key is missing")
		}
	}
	page, e := derivedReadPage(state, section, after, n+1)
	if e != nil {
		return zero, true, e
	}
	result, e := projectedSectionPage(ctx, frame, scope, section, page, n)
	return result, true, e
}
func derivedReadPage(state OutcomeState, section, after string, limit int) (ports.ContinuityReadPage, error) {
	b, e := ports.NewContinuityPageBuilder(after, limit)
	if e != nil {
		return ports.ContinuityReadPage{}, e
	}
	if e = eachDerivedItem(state, section, b.Add); e != nil {
		return ports.ContinuityReadPage{}, e
	}
	if !b.HasAfter() {
		return ports.ContinuityReadPage{}, d.NewError(d.ErrorCodeInvalidArgument, "invalid_cursor: section key is missing")
	}
	return b.Page(), nil
}
func eachDerivedItem(state OutcomeState, section string, visit func(string, any) error) error {
	if _, known := derivedOptions(section); !known {
		return d.NewError(d.ErrorCodeInvalidArgument, "unknown continuity section")
	}
	switch section {
	case "blocking_states":
		for _, v := range state.BlockingStates {
			if e := visit(string(v.Ref.Kind)+"/"+v.Ref.ID.String(), v); e != nil {
				return e
			}
		}
	case "conclusion_contestations":
		for _, v := range state.ConclusionContestations {
			raw, e := json.Marshal(v)
			if e != nil {
				return e
			}
			if e = visit(string(raw), v); e != nil {
				return e
			}
		}
	case "external_context":
		raw, e := json.Marshal(state.Outcome.ExternalContext)
		if e != nil {
			return e
		}
		return visit(string(raw), state.Outcome.ExternalContext)
	case "active_roadmaps":
		for _, p := range state.ActiveRoadmaps {
			v := struct {
				Slot          d.RoadmapActiveSlot `json:"slot"`
				Hash          string              `json:"content_hash"`
				NodeCount     int                 `json:"node_count"`
				DetailOmitted bool                `json:"detail_omitted"`
			}{p.Slot, p.Revision.ContentHash, len(p.Revision.Nodes), true}
			if e := visit(string(p.Slot.PlanScope.Kind)+"/"+p.Slot.PlanScope.ID.String(), v); e != nil {
				return e
			}
		}
	case "active_plan_references":
		for _, p := range state.ActiveRoadmaps {
			for _, v := range p.LiveReferences {
				node, e := json.Marshal(v.NodeKey)
				if e != nil {
					return e
				}
				if e = visit(v.RoadmapID.String()+"/"+string(node), v); e != nil {
					return e
				}
			}
		}
	default:
		group := strings.TrimSuffix(section, "_work")
		for i, v := range state.WorkItems {
			if i >= len(state.WorkItemOperationalStates) || state.WorkItemOperationalStates[i].Ref != v.Ref() {
				return d.NewError(d.ErrorCodeInvalidConfig, "operational projection header order mismatch")
			}
			st := state.WorkItemOperationalStates[i]
			if string(st.DisplayState) != group {
				continue
			}
			item := struct {
				Work  EntitySummary              `json:"work_item"`
				State d.WorkItemOperationalState `json:"state"`
			}{summary(v.Ref(), v.Version, v.Title, string(v.Lifecycle), v.Priority), st}
			if e := visit(v.ID.String(), item); e != nil {
				return e
			}
		}
	}
	return nil
}
func projectedSectionPage(ctx context.Context, frame *continuityReadFrame, scope d.Scope, section string, page ports.ContinuityReadPage, n int) (SectionPage, error) {
	result := SectionPage{Items: []json.RawMessage{}, Count: page.Count, Omitted: page.Remaining + max(0, len(page.Items)-n), OutcomeRevision: frame.revision, EvaluatedAt: frame.at}
	for _, item := range page.Items[:min(n, len(page.Items))] {
		result.Items = append(result.Items, item.Value)
	}
	if section == "blocking_states" {
		var e error
		result.Items, e = populateBlockingItems(ctx, frame, scope, result.Items)
		if e != nil {
			return SectionPage{}, e
		}
	}
	if len(page.Items) > n {
		last := page.Items[n-1]
		key := last.Key
		switch section {
		case "external_references", "active_plan_references", "conclusion_contestations":
			key = "h:" + filterHash(last.Value)
		}
		result.NextCursor = encodeCursor(queryCursor{CursorVersion: 2, Namespace: scope.NamespaceID, Outcome: scope.OutcomeID, Section: section, Revision: frame.revision, Key: key, EvaluatedAt: frame.at})
	}
	raw, e := json.Marshal(result)
	if e != nil {
		return SectionPage{}, e
	}
	if len(raw) > MaxSnapshotBytes {
		return SectionPage{}, d.NewError(d.ErrorCodeGraphLimitExceeded, "section exceeds byte limit; reduce limit")
	}
	return result, nil
}
func enrichBlockingItems(ctx context.Context, frame *continuityReadFrame, scope d.Scope, values []json.RawMessage) (json.RawMessage, error) {
	result, e := populateBlockingItems(ctx, frame, scope, values)
	if e != nil {
		return nil, e
	}
	if len(result) != 1 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "invalid blocking proof item count")
	}
	return result[0], nil
}
func populateBlockingItems(ctx context.Context, frame *continuityReadFrame, scope d.Scope, values []json.RawMessage) ([]json.RawMessage, error) {
	states := make([]BlockingState, len(values))
	ids := []d.ID{}
	seen := map[d.ID]bool{}
	for i, raw := range values {
		if e := json.Unmarshal(raw, &states[i]); e != nil {
			return nil, e
		}
		for _, applied := range states[i].ActiveBlockers {
			if !seen[applied.Blocker.ID] {
				ids = append(ids, applied.Blocker.ID)
				seen[applied.Blocker.ID] = true
			}
		}
	}
	full, e := frame.projection.BlockerDetails(ctx, scope, ids, MaxSnapshotBytes)
	if e != nil {
		return nil, e
	}
	byID := map[d.ID]d.Blocker{}
	for _, v := range full {
		byID[v.ID] = v
	}
	result := []json.RawMessage{}
	for _, state := range states {
		for i, v := range state.ActiveBlockers {
			detail, ok := byID[v.Blocker.ID]
			if !ok {
				return nil, d.NewError(d.ErrorCodeNotFound, "projected blocker missing")
			}
			state.ActiveBlockers[i].Blocker = detail
		}
		raw, e := json.Marshal(state)
		if e != nil {
			return nil, e
		}
		result = append(result, raw)
	}
	return result, nil
}
