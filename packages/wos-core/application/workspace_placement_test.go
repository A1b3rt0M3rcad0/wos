package application

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
)

func TestPlacementPreviewRequiresExplicitNullableParent(t *testing.T) {
	for _, raw := range []string{`{}`, `{"wrong":null}`, `{"parent_id":null,"wrong":1}`, `null`} {
		var query PlacementPreviewQuery
		if err := json.Unmarshal([]byte(raw), &query); err == nil {
			t.Fatalf("accepted absent or confused destination: %s", raw)
		}
	}
	var query PlacementPreviewQuery
	if err := json.Unmarshal([]byte(`{"parent_id":null}`), &query); err != nil || query.ParentID != nil {
		t.Fatal("explicit Outcome destination did not decode", err)
	}
}

func TestWorkspacePlacementPreviewSharesGuardedBlockingAndReadinessFacts(t *testing.T) {
	for _, backend := range []string{"memory", "sqlite", "postgres"} {
		t.Run(backend, func(t *testing.T) {
			ctx := context.Background()
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			var store ports.TransactionManager
			switch backend {
			case "memory":
				store = memory.New()
			case "sqlite":
				sql, err := sqlite.Open(filepath.Join(t.TempDir(), "placement.db"), sqlite.Options{MigrateOnOpen: true})
				must(err)
				t.Cleanup(func() { sql.Close() })
				store = sql
			case "postgres":
				dsn := os.Getenv("WOS_TEST_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("real PostgreSQL required; CI supplies WOS_TEST_POSTGRES_DSN")
				}
				schema := fmt.Sprintf("workspace_placement_%x", sha256.Sum256([]byte(t.TempDir())))[:55]
				sql, err := postgres.Open(dsn, postgres.Options{Schema: schema, MigrateOnOpen: true})
				must(err)
				t.Cleanup(func() { sql.Close() })
				store = sql
			}
			now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
			clock := &workspaceBoundaryClock{now: now}
			ids := &workspaceSnapshotIDs{}
			id := func() d.ID { value, err := ids.NewID(); must(err); return value }
			service, err := NewService(store, clock, ids)
			must(err)
			outcome, err := d.NewOutcome(id(), id(), "Real structural impact", "", "Reviewed placement", d.PriorityNormal, now)
			must(err)
			outcome.Lifecycle = d.OutcomeLifecycleActive
			scope := outcome.Scope()
			objective := func(title string, parent *d.ID) d.Objective {
				value, err := d.NewObjective(id(), scope, title, "", d.PriorityNormal, false, now)
				must(err)
				value.ParentObjectiveID = parent
				return value
			}
			first := objective("Old parent", nil)
			second := objective("New parent", nil)
			clear := objective("Unblocked parent", nil)
			branch := objective("Planned branch", &first.ID)
			child := objective("Nested branch", &branch.ID)
			frozen := objective("Execution branch", nil)
			work := func(title string, parent *d.ID) d.WorkItem {
				value, err := d.NewWorkItem(id(), scope, title, "Preserved authored work", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
				must(err)
				value.ObjectiveID = parent
				return value
			}
			task := work("Move with a direct impediment", &branch.ID)
			ready := work("Move out of inherited blocking", &child.ID)
			executing := work("Do not move executing descendants", &frozen.ID)
			must(executing.Claim(id(), "executor", d.ActorRef{Kind: d.ActorKindHuman, Provider: "fixture", ID: "executor"}, 30*time.Second, now.Add(-time.Minute)))
			blocker := func(ref d.EntityRef, title string, propagation d.BlockerPropagation) d.Blocker {
				value, err := d.NewBlocker(id(), ref, nil, &d.ExternalCause{Provider: "fixture", ID: title, Description: title}, title, propagation, now)
				must(err)
				return value
			}
			oldInherited := blocker(first.Ref(), "Old branch authorization", d.BlockerPropagationSubtree)
			newInherited := blocker(second.Ref(), "New branch authorization", d.BlockerPropagationSubtree)
			direct := blocker(task.Ref(), "Direct Task requirement", d.BlockerPropagationDirect)
			parentDirect := blocker(first.Ref(), "Parent-only requirement", d.BlockerPropagationDirect)
			u, err := store.Begin(ctx)
			must(err)
			_, err = u.Coordination().LockOutcome(ctx, scope)
			must(err)
			must(u.Outcomes().Insert(ctx, outcome))
			for _, value := range []d.Objective{first, second, clear, branch, child, frozen} {
				must(u.Objectives().Insert(ctx, value))
			}
			for _, value := range []d.WorkItem{task, ready, executing} {
				must(u.WorkItems().Insert(ctx, value))
			}
			for _, value := range []d.Blocker{oldInherited, newInherited, direct, parentDirect} {
				must(u.(ports.IssueBlockerUnitOfWork).Blockers().Insert(ctx, value))
			}
			_, err = u.Coordination().AdvanceOutcome(ctx, scope)
			must(err)
			must(u.Commit())
			preview := func(ref d.EntityRef, parent *d.ID) EntityWorkspaceView {
				value, err := service.GetEntityWorkspaceView(ctx, scope, WorkspaceViewQuery{Subject: WorkspaceSubject{Kind: "entity", Ref: ref}, PlacementPreview: &PlacementPreviewQuery{ParentID: parent}})
				must(err)
				if value.PlacementPreview == nil {
					t.Fatal("missing explicit placement preview")
				}
				return value
			}
			plain, err := service.GetEntityWorkspaceView(ctx, scope, WorkspaceViewQuery{Subject: WorkspaceSubject{Kind: "entity", Ref: task.Ref()}})
			must(err)
			if plain.PlacementPreview != nil || plain.Coverage["placement_preview"].Status != "not_loaded" {
				t.Fatal("ordinary inspector loaded structural preview")
			}
			moved := preview(task.Ref(), &second.ID)
			impact := moved.PlacementPreview
			if impact.Availability != "blocked" || len(impact.ReasonCodes) != 1 || impact.ReasonCodes[0] != "work_item_planning_frozen" || !impact.Changed || impact.OldParent.Ref != branch.Ref() || impact.NewParent.Ref != second.Ref() || impact.AffectedTasks != 1 || impact.AffectedObjectives != 0 {
				t.Fatalf("incorrect Task impact: %+v", impact)
			}
			assertBlockers := func(value WorkspacePlacementBlocking, want ...d.ID) {
				t.Helper()
				found := map[d.ID]WorkspacePlacementBlocker{}
				for _, item := range value.Blockers {
					found[item.Ref.ID] = item
				}
				if len(found) != len(want) || value.IsBlocked != (len(want) > 0) {
					t.Fatalf("wrong applied Blockers: %+v; expected %v", value, want)
				}
				for _, id := range want {
					if _, ok := found[id]; !ok {
						t.Fatalf("missing Blocker %s in %+v", id, value)
					}
				}
				if item, ok := found[direct.ID]; ok && item.Inherited {
					t.Fatal("direct Task blocker was marked inherited")
				}
				for _, id := range []d.ID{oldInherited.ID, newInherited.ID} {
					if item, ok := found[id]; ok && (!item.Inherited || item.InheritedFrom == nil) {
						t.Fatal("inherited Blocker lost its origin")
					}
				}
			}
			assertBlockers(impact.BlockingChanges[0].Before, oldInherited.ID, direct.ID)
			assertBlockers(impact.BlockingChanges[0].After, newInherited.ID, direct.ID)
			if impact.ReadinessBefore == nil || impact.ReadinessAfter == nil || impact.ReadinessAfter.DisplayState != d.WorkItemDisplayStateBlocked || !impact.ReadinessAfter.EvaluatedAt.Equal(moved.EvaluatedAt) {
				t.Fatal("Task operational projection lost the frame")
			}
			unblocked := preview(ready.Ref(), &clear.ID).PlacementPreview
			if unblocked.ReadinessBefore.DisplayState != d.WorkItemDisplayStateBlocked || unblocked.ReadinessAfter.DisplayState != d.WorkItemDisplayStateReady {
				t.Fatalf("readiness did not reflect inherited blocking: %+v", unblocked)
			}
			assertBlockers(unblocked.BlockingChanges[0].After)
			branchMove := preview(branch.Ref(), &second.ID).PlacementPreview
			if branchMove.Availability != "available" || branchMove.AffectedObjectives != 2 || branchMove.AffectedTasks != 2 || len(branchMove.BlockingChanges) != 4 || branchMove.Coverage.Status != "complete" {
				t.Fatalf("wrong planned subtree impact: %+v", branchMove)
			}
			for _, changed := range branchMove.BlockingChanges {
				if changed.Title == "" {
					t.Fatal("affected child has no authorized name")
				}
				if changed.Ref == task.Ref() {
					assertBlockers(changed.Before, oldInherited.ID, direct.ID)
					assertBlockers(changed.After, newInherited.ID, direct.ID)
				} else {
					assertBlockers(changed.Before, oldInherited.ID)
					assertBlockers(changed.After, newInherited.ID)
				}
			}
			cycle := preview(branch.Ref(), &child.ID).PlacementPreview
			if cycle.Availability != "blocked" || cycle.Error == nil || cycle.ReasonCodes[0] != "objective_hierarchy_cycle" {
				t.Fatalf("cycle was not explained: %+v", cycle)
			}
			if cycle.Coverage.Status != "unknown" || len(cycle.BlockingChanges) != 0 {
				t.Fatal("invalid cyclic ancestry was presented as complete operational impact")
			}
			missingID := id()
			missing := preview(task.Ref(), &missingID).PlacementPreview
			if missing.Availability != "blocked" || missing.Error == nil || missing.Error.Code != d.ErrorCodeNotFound || missing.Coverage.Status != "unknown" || missing.ReadinessAfter != nil {
				t.Fatalf("missing destination was presented as valid readiness: %+v", missing)
			}
			inExecution := preview(frozen.Ref(), &second.ID).PlacementPreview
			if inExecution.Availability != "blocked" || inExecution.Error == nil || inExecution.ReasonCodes[0] != "work_item_planning_frozen" {
				t.Fatalf("executing subtree was not guarded: %+v", inExecution)
			}
			unchanged, err := service.GetEntityWorkspaceView(ctx, scope, WorkspaceViewQuery{Subject: WorkspaceSubject{Kind: "entity", Ref: task.Ref()}})
			must(err)
			if unchanged.OutcomeRevision != plain.OutcomeRevision || unchanged.EntityVersion != task.Version || unchanged.Placement.Parent.Ref != branch.Ref() || len(unchanged.Activity.Items) != len(plain.Activity.Items) {
				t.Fatal("read-only preview mutated placement, version, revision or timeline")
			}
			// Large branches and many active Blockers retain exact totals while
			// explicitly limiting the documentary examples in the response.
			u, err = store.Begin(ctx)
			must(err)
			_, err = u.Coordination().LockOutcome(ctx, scope)
			must(err)
			for index := 0; index < 24; index++ {
				must(u.WorkItems().Insert(ctx, work(fmt.Sprintf("Additional planned Task %02d", index), &child.ID)))
			}
			for index := 0; index < 6; index++ {
				must(u.(ports.IssueBlockerUnitOfWork).Blockers().Insert(ctx, blocker(task.Ref(), fmt.Sprintf("Additional direct condition %02d", index), d.BlockerPropagationDirect)))
			}
			_, err = u.Coordination().AdvanceOutcome(ctx, scope)
			must(err)
			must(u.Commit())
			bounded := preview(branch.Ref(), &second.ID).PlacementPreview
			if bounded.AffectedObjectives != 2 || bounded.AffectedTasks != 26 || len(bounded.BlockingChanges) != 20 || bounded.Coverage.Status != "partial" || *bounded.Coverage.TotalCount != 28 {
				t.Fatalf("large branch hid its actual extent: %+v", bounded)
			}
			foundBounded := false
			for _, changed := range bounded.BlockingChanges {
				if changed.Ref == task.Ref() {
					foundBounded = true
					if !changed.After.IsBlocked || len(changed.After.Blockers) != 5 || changed.After.Coverage.Status != "partial" || *changed.After.Coverage.TotalCount != 8 {
						t.Fatalf("Blocker sample was presented as the whole state: %+v", changed.After)
					}
				}
			}
			if !foundBounded {
				t.Fatal("stable sample omitted its earliest affected Task")
			}
			// A preview is not an authority token and cannot be silently reused
			// after another mutation advances the Outcome's coordination state.
			cc := d.CommandContext{CommandID: id(), PrincipalID: "owner", Actor: d.ActorRef{Kind: d.ActorKindHuman, Provider: "fixture", ID: "owner"}}
			_, err = service.CreateWorkItem(ctx, cc, CreateWorkItemCommand{Scope: scope, Title: "Concurrent authored work", Priority: d.PriorityNormal})
			must(err)
			cc.CommandID = id()
			_, err = service.SetWorkItemObjective(ctx, cc, SetWorkItemObjectiveCommand{Scope: scope, WorkItemID: task.ID, ExpectedVersion: task.Version, ExpectedOutcomeRevision: &moved.OutcomeRevision, ObjectiveID: &second.ID})
			var conflict *d.Error
			if !errors.As(err, &conflict) || conflict.Details == nil || conflict.Details.ReasonCode != "structure_preview_changed" {
				t.Fatalf("stale impact review was accepted: %v", err)
			}
		})
	}
}
