package sqlite

import (
	"context"
	"encoding/json"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWorkspaceNotesAndTimelinePreserveDeclarationsReplayCorrectionsAndFrozenPages(t *testing.T) {
	for _, backend := range []string{"memory", "sql"} {
		t.Run(backend, func(t *testing.T) {
			ctx := context.Background()
			now := time.Date(2026, 10, 19, 14, 0, 0, 0, time.UTC)
			path := filepath.Join(t.TempDir(), "history.db")
			var store ports.TransactionManager
			var sqlStore *Store
			if backend == "memory" {
				store = memory.New()
			} else {
				sqlStore = openTestStore(t, path)
				store = sqlStore
			}
			service, err := a.NewService(store, sqliteFixedClock{now: now}, &sqliteSequenceIDs{prefix: "019a3102", next: 100})
			if err != nil {
				t.Fatal(err)
			}
			next := 500
			cc := func(key string) d.CommandContext {
				next++
				if key != "" {
					key = "workspace-" + key
				}
				return sqliteCommandContext(noteTestID(next).String(), key)
			}
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			outcome, err := service.CreateOutcome(ctx, cc(""), a.CreateOutcomeCommand{NamespaceID: noteTestID(1), Title: "Documented collaboration", DesiredState: "Recorded work remains distinct from proof", Priority: d.PriorityNormal})
			must(err)
			scope := outcome.Value.Scope()
			objective, err := service.CreateObjective(ctx, cc(""), a.CreateObjectiveCommand{Scope: scope, Title: "Explain the result", Priority: d.PriorityNormal})
			must(err)
			task, err := service.CreateWorkItem(ctx, cc(""), a.CreateWorkItemCommand{Scope: scope, Title: "External work", ObjectiveID: &objective.Value.ID, Priority: d.PriorityNormal, Lifecycle: d.WorkItemLifecycleBacklog})
			must(err)
			command := a.AppendEntityNoteCommand{Scope: scope, SubjectRef: task.Value.Ref(), Type: d.EntityNoteUpdate, Body: "I finished the Task; this is an authored declaration, not acceptance."}
			originalCC := cc("note-original")
			first, err := service.AppendEntityNote(ctx, originalCC, command)
			must(err)
			replay, err := service.AppendEntityNote(ctx, originalCC, command)
			must(err)
			if !replay.IdempotentReplay || replay.Value.ID != first.Value.ID || replay.OutcomeRevision != first.OutcomeRevision {
				t.Fatal("note replay duplicated its declaration")
			}
			after, err := service.GetWorkItem(ctx, scope, task.Value.ID)
			must(err)
			if after.Value.Lifecycle != d.WorkItemLifecycleBacklog || after.Value.Version != task.Value.Version {
				t.Fatal("declaration changed Task state or version")
			}
			for i := 0; i < 7; i++ {
				command.Body = strings.Repeat("An observation remains documentary. ", 50)
				observed := now.Add(-time.Duration(i+1) * time.Hour)
				command.OccurredAt = &observed
				_, err = service.AppendEntityNote(ctx, cc("note-more-"+noteTestID(i+20).String()), command)
				must(err)
			}
			query := a.WorkspaceTimelineQuery{SubjectRef: &command.SubjectRef, Limit: 3}
			page, err := service.GetWorkspaceTimeline(ctx, scope, query)
			must(err)
			if len(page.Items) != 3 || page.NextCursor == "" || page.ScopeBasis != "current_placement" {
				t.Fatalf("recent bounded history: %#v", page)
			}
			for _, entry := range page.Items {
				if entry.Origin != "entity_note" || entry.Source.Kind != "entity_note" || entry.Source.ID != entry.ID || !entry.PreviewTruncated || len(entry.Description) > a.MaxWorkspaceTimelinePreviewBytes || entry.OccurredAt == nil {
					t.Fatalf("preview/provenance: %#v", entry)
				}
			}
			frontier := page.HistoryThroughRevision
			command.Body = "A new declaration must not enter the traversal already open."
			newest, err := service.AppendEntityNote(ctx, cc("new-after-page"), command)
			must(err)
			seen := map[d.ID]bool{}
			for _, entry := range page.Items {
				seen[entry.ID] = true
			}
			cursor := page.NextCursor
			for cursor != "" {
				query.Cursor = cursor
				continuation, err := service.GetWorkspaceTimeline(ctx, scope, query)
				must(err)
				if continuation.HistoryThroughRevision != frontier || !continuation.HasNewerEntries {
					t.Fatal("timeline did not retain its historical frontier")
				}
				for _, entry := range continuation.Items {
					if entry.ID == newest.Value.ID || seen[entry.ID] {
						t.Fatal("frozen traversal repeated or admitted a new fact")
					}
					seen[entry.ID] = true
				}
				cursor = continuation.NextCursor
			}
			if !seen[first.Value.ID] {
				t.Fatal("same subject history skipped original note")
			}
			command.Body = "Correction: the external checks need another pass."
			command.SupersedesNoteID = &first.Value.ID
			corrected, err := service.AppendEntityNote(ctx, cc("correct-own-note"), command)
			must(err)
			original, err := service.GetEntityNote(ctx, scope, first.Value.ID)
			must(err)
			if original.Body != first.Value.Body || corrected.Value.SupersedesNoteID == nil || *corrected.Value.SupersedesNoteID != first.Value.ID {
				t.Fatal("correction rewrote its original")
			}
			other := cc("correct-other-note")
			other.PrincipalID = "other-principal"
			other.Actor.ID = "other-principal"
			if _, err = service.AppendEntityNote(ctx, other, command); err == nil {
				t.Fatal("another Principal corrected original note")
			}
			before, err := service.GetWorkspaceTimeline(ctx, scope, a.WorkspaceTimelineQuery{Limit: 20, IncludeDescendants: true})
			must(err)
			invalid := command
			invalid.SupersedesNoteID = nil
			invalid.ArtifactRefs = []d.ID{noteTestID(404)}
			if _, err = service.AppendEntityNote(ctx, cc("invalid-reference"), invalid); err == nil {
				t.Fatal("note accepted missing reference")
			}
			afterFailure, err := service.GetWorkspaceTimeline(ctx, scope, a.WorkspaceTimelineQuery{Limit: 20, IncludeDescendants: true})
			must(err)
			if afterFailure.OutcomeRevision != before.OutcomeRevision || len(afterFailure.Items) != len(before.Items) {
				t.Fatal("failed note partially advanced revision/events")
			}
			foreign := scope
			foreign.OutcomeID = noteTestID(405)
			if _, err = service.GetEntityNote(ctx, foreign, first.Value.ID); err == nil {
				t.Fatal("note leaked across scope")
			}
			query.Cursor = page.NextCursor
			otherCtx := a.WithIdentity(ctx, a.Identity{NamespaceID: scope.NamespaceID, PrincipalID: "another-reader"})
			if _, err = service.GetWorkspaceTimeline(otherCtx, scope, query); err == nil {
				t.Fatal("cursor reused by another Principal")
			}
			// Completed/cancelled nonarchived Tasks can receive documentary updates.
			cancelled, err := service.CancelWorkItem(ctx, cc(""), a.CancelWorkItemCommand{Scope: scope, WorkItemID: task.Value.ID, ExpectedVersion: task.Value.Version, Reason: "Stop external work explicitly"})
			must(err)
			command.Body = "A post-work observation does not reopen execution."
			command.SupersedesNoteID = nil
			_, err = service.AppendEntityNote(ctx, cc("terminal-document"), command)
			must(err)
			after, err = service.GetWorkItem(ctx, scope, cancelled.Value.ID)
			must(err)
			if after.Value.Lifecycle != d.WorkItemLifecycleCancelled {
				t.Fatal("note reopened terminal Task")
			}
			current, err := service.GetOutcome(ctx, scope)
			must(err)
			abandoned, err := service.AbandonOutcome(ctx, cc(""), a.AbandonOutcomeCommand{Scope: scope, ExpectedVersion: current.Value.Version, Reason: "Close the planned result explicitly"})
			must(err)
			command.SubjectRef = abandoned.Value.Ref()
			command.Body = "A documentary update remains possible after explicit closure."
			_, err = service.AppendEntityNote(ctx, cc("closed-outcome-note"), command)
			must(err)
			_, err = service.ArchiveOutcome(ctx, cc(""), a.ArchiveOutcomeCommand{Scope: scope, ExpectedVersion: abandoned.Value.Version})
			must(err)

			if _, err = service.AppendEntityNote(ctx, cc("archived-document"), command); err == nil {
				t.Fatal("archived scope accepted documentary note")
			}
			if _, err = service.AppendWorkActivity(ctx, cc("no-authority"), a.AppendWorkActivityCommand{Scope: scope, Type: d.WorkActivityProgress, Description: "A human update cannot impersonate contracted execution"}); err == nil {
				t.Fatal("WorkActivity accepted missing signed authority")
			}
			if sqlStore != nil {
				must(sqlStore.Close())
				sqlStore = openTestStore(t, path)
				service, err = a.NewService(sqlStore, sqliteFixedClock{now: now}, &sqliteSequenceIDs{prefix: "019a3102", next: 900})
				must(err)
				restored, err := service.GetEntityNote(ctx, scope, first.Value.ID)
				must(err)
				if restored.Body != first.Value.Body {
					t.Fatal("restart changed note history")
				}
			}
		})
	}
}

func TestWorkspaceTimelineSameCommitOrderingActivityDedupCriteriaAndDescendantOrigin(t *testing.T) {
	for _, backend := range []string{"memory", "sql"} {
		t.Run(backend, func(t *testing.T) {
			ctx := context.Background()
			now := time.Date(2026, 10, 19, 15, 0, 0, 0, time.UTC)
			var store ports.TransactionManager
			if backend == "memory" {
				store = memory.New()
			} else {
				store = openTestStore(t, filepath.Join(t.TempDir(), "order.db"))
			}
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			scope := d.Scope{NamespaceID: noteTestID(1000), OutcomeID: noteTestID(1001)}
			outcome, err := d.NewOutcome(scope.OutcomeID, scope.NamespaceID, "Timeline", "", "Keep canonical commit order", d.PriorityNormal, now)
			must(err)
			objective, err := d.NewObjective(noteTestID(1002), scope, "Parent", "", d.PriorityNormal, true, now)
			must(err)
			work, err := d.NewWorkItem(noteTestID(1003), scope, "Child", "", d.PriorityNormal, d.WorkItemLifecycleBacklog, now)
			must(err)
			work.ObjectiveID = &objective.ID
			criterion, err := d.NewSuccessCriterion(noteTestID(1004), work.Ref(), "External result", "", true, d.VerificationModeEvidenceReview)
			must(err)
			must(work.AddCriterion(criterion, now))
			u, err := store.Begin(ctx)
			must(err)
			must(u.Outcomes().Insert(ctx, outcome))
			must(u.Objectives().Insert(ctx, objective))
			must(insertWorkItemFixture(ctx, u.Objectives(), u.WorkItems(), &work))
			revision, err := u.Coordination().AdvanceOutcome(ctx, scope)
			must(err)
			activity := d.WorkActivity{ID: noteTestID(1005), Scope: scope, ContractID: noteTestID(1006), WorkItemID: work.ID, Sequence: uint64(revision), Type: d.WorkActivityProgress, Description: "External executor declaration", PrincipalID: "executor", Actor: d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "executor", DisplayName: strings.Repeat("x", 80<<10)}, RecordedAt: now, CriterionRefs: []d.WorkActivityCriterionRef{{CriterionID: criterion.ID, CriterionRevision: 1}}}
			events := []d.DomainEvent{}
			for i := 0; i < 4; i++ {
				payload := json.RawMessage(`{"reason":"Distinct recorded fact"}`)
				typ := "work_item.updated"
				if i == 1 {
					typ = "work_activity.recorded"
					payload, err = json.Marshal(activity)
					must(err)
				}
				events = append(events, d.DomainEvent{EventID: noteTestID(1010 + i), EventType: typ, SchemaVersion: 1, NamespaceID: scope.NamespaceID, OutcomeID: scope.OutcomeID, OutcomeRevision: revision, EventIndex: uint32(i), AggregateRef: work.Ref(), PrincipalID: activity.PrincipalID, Actor: activity.Actor, RecordedAt: now, CommandID: noteTestID(1020), CorrelationID: "one-intention", Payload: payload})
			}
			must(u.Events().Append(ctx, events))
			must(u.Commit())
			service, err := a.NewService(store, sqliteFixedClock{now: now}, &sqliteSequenceIDs{prefix: "019a3103", next: 200})
			must(err)
			workSubject := work.Ref()
			bounded, err := service.GetWorkspaceTimeline(ctx, scope, a.WorkspaceTimelineQuery{SubjectRef: &workSubject, Limit: 20})
			must(err)
			encoded, err := json.Marshal(bounded)
			must(err)
			if len(encoded) > a.MaxSnapshotBytes || len(bounded.Items) >= 4 || bounded.NextCursor == "" {
				t.Fatal("timeline did not enforce its response byte bound")
			}
			ref := objective.Ref()
			q := a.WorkspaceTimelineQuery{SubjectRef: &ref, IncludeDescendants: true, Limit: 1}
			seen := map[d.ID]bool{}
			expectedIndex := 3
			count := 0
			for {
				page, err := service.GetWorkspaceTimeline(ctx, scope, q)
				must(err)
				if len(page.Items) != 1 {
					t.Fatal("bounded event page missing")
				}
				entry := page.Items[0]
				if seen[entry.ID] || int(entry.EventIndex) != expectedIndex || !entry.DescendantOrigin || entry.SubjectRef != work.Ref() {
					t.Fatalf("same revision/origin: %#v", entry)
				}
				if entry.EventType == "work_activity.recorded" && (entry.ID != activity.ID || entry.Origin != "work_activity" || entry.Source.ContractID == nil || *entry.Source.ContractID != activity.ContractID) {
					t.Fatal("activity was not represented once by canonical record")
				}
				seen[entry.ID] = true
				count++
				expectedIndex--
				q.Cursor = page.NextCursor
				if q.Cursor == "" {
					break
				}
			}
			if count != 4 {
				t.Fatalf("same commit facts: %d", count)
			}
			workRef := work.Ref()
			criterionPage, err := service.GetWorkspaceTimeline(ctx, scope, a.WorkspaceTimelineQuery{SubjectRef: &workRef, CriterionID: &criterion.ID, Limit: 20})
			must(err)
			if len(criterionPage.Items) != 1 || criterionPage.Items[0].ID != activity.ID {
				t.Fatalf("criterion activity filter: %#v", criterionPage.Items)
			}
			// A later structural change invalidates descendant membership, rather than
			// pretending the current hierarchy was the historical hierarchy.
			initial, err := service.GetWorkspaceTimeline(ctx, scope, a.WorkspaceTimelineQuery{SubjectRef: &ref, IncludeDescendants: true, Limit: 1})
			must(err)
			u, err = store.Begin(ctx)
			must(err)
			later, err := u.Coordination().AdvanceOutcome(ctx, scope)
			must(err)
			moved := events[0]
			moved.EventID = noteTestID(1040)
			moved.CommandID = noteTestID(1041)
			moved.EventType = "work_item.objective_changed"
			moved.OutcomeRevision = later
			moved.EventIndex = 0
			must(u.Events().Append(ctx, []d.DomainEvent{moved}))
			must(u.Commit())
			q.Cursor = initial.NextCursor
			if _, err = service.GetWorkspaceTimeline(ctx, scope, q); err == nil {
				t.Fatal("descendant cursor silently changed structural membership")
			}
		})
	}
}
