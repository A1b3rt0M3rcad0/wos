package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestSQLiteWave11RoadmapPlanningHistorySurvivesRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "wos.db")
	store, err := Open(path, Options{BusyTimeout: time.Second, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	clock := sqliteFixedClock{now: time.Date(2026, 10, 2, 23, 0, 0, 0, time.UTC)}
	service, err := application.NewService(
		store,
		clock,
		&sqliteSequenceIDs{prefix: "0199f400", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	cc := sqliteCommandContext("0199f401-0000-7000-8000-000000000001", "")
	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  domain.MustParseID("0199f401-0000-7000-8000-000000000010"),
		Title:        "SQLite planning",
		DesiredState: "planning history survives restart",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value
	objectiveResult, err := service.CreateObjective(ctx, sqliteCommandContext(
		"0199f401-0000-7000-8000-000000000002", "",
	), application.CreateObjectiveCommand{
		Scope: outcome.Scope(), Title: "Persisted objective", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	objective := objectiveResult.Value

	roadmapResult, err := service.CreateRoadmap(ctx, sqliteCommandContext(
		"0199f401-0000-7000-8000-000000000003", "",
	), application.CreateRoadmapCommand{
		Scope:     outcome.Scope(),
		PlanScope: domain.RoadmapPlanScope{Kind: domain.RoadmapScopeOutcome, ID: outcome.ID},
		Title:     "Persisted roadmap",
	})
	if err != nil {
		t.Fatal(err)
	}
	roadmap := roadmapResult.Value
	opened, err := service.OpenRoadmapDraft(ctx, sqliteCommandContext(
		"0199f401-0000-7000-8000-000000000004", "",
	), application.OpenRoadmapDraftCommand{
		Scope: outcome.Scope(), RoadmapID: roadmap.ID, ExpectedVersion: roadmap.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	edited, err := service.ReplaceRoadmapDraft(ctx, sqliteCommandContext(
		"0199f401-0000-7000-8000-000000000005", "",
	), application.ReplaceRoadmapDraftCommand{
		Scope: outcome.Scope(), RoadmapID: roadmap.ID,
		ExpectedVersion:      opened.Value.Version,
		ExpectedDraftVersion: opened.Value.Draft.DraftVersion,
		Nodes: []domain.RoadmapNode{
			{
				NodeKey: "objective", NodeType: domain.RoadmapNodeReference,
				TargetRef: func() *domain.EntityRef {
					ref := objective.Ref()
					return &ref
				}(),
				Title: "Persisted objective", Position: 0,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	published, err := service.PublishRoadmapDraft(ctx, sqliteCommandContext(
		"0199f401-0000-7000-8000-000000000006", "",
	), application.PublishRoadmapDraftCommand{
		Scope: outcome.Scope(), RoadmapID: roadmap.ID,
		ExpectedVersion:      edited.Value.Version,
		ExpectedDraftVersion: edited.Value.Draft.DraftVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	activated, err := service.ActivateRoadmapRevision(ctx, sqliteCommandContext(
		"0199f401-0000-7000-8000-000000000007", "",
	), application.ActivateRoadmapRevisionCommand{
		Scope: outcome.Scope(), RoadmapID: roadmap.ID,
		ExpectedVersion: published.Value.Version, RevisionNumber: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if activated.Value.RoadmapID != roadmap.ID {
		t.Fatalf("active roadmap id = %s", activated.Value.RoadmapID)
	}

	reopenedStore := reopenIntegrationFixture(t, store, path)
	defer reopenedStore.Close()
	restarted, err := application.NewService(
		reopenedStore,
		clock,
		&sqliteSequenceIDs{prefix: "0199f402", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	read, err := restarted.GetRoadmap(ctx, outcome.Scope(), roadmap.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(read.Value.Revisions) != 1 {
		t.Fatalf("revision count = %d, want 1", len(read.Value.Revisions))
	}
	if read.Value.Revisions[0].ContentHash == "" ||
		read.Value.Revisions[0].Nodes[0].ReferenceSnapshot == nil ||
		read.Value.Revisions[0].Nodes[0].ReferenceSnapshot.Title != "Persisted objective" {
		t.Fatalf("persisted revision = %#v", read.Value.Revisions[0])
	}
	slot, _, err := restarted.GetActiveRoadmapSlot(
		ctx,
		outcome.Scope(),
		domain.RoadmapPlanScope{Kind: domain.RoadmapScopeOutcome, ID: outcome.ID},
	)
	if err != nil {
		t.Fatal(err)
	}
	if slot == nil || slot.RoadmapID != roadmap.ID || slot.RevisionNumber != 1 {
		t.Fatalf("restarted active slot = %#v", slot)
	}
	history, _, err := restarted.ListRoadmapActivationHistory(
		ctx,
		outcome.Scope(),
		domain.RoadmapPlanScope{Kind: domain.RoadmapScopeOutcome, ID: outcome.ID},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 1 || history[0].Action != domain.RoadmapActivationActivated {
		t.Fatalf("activation history = %#v", history)
	}
}

func TestSQLiteWave11PublishedSnapshotsIgnoreLaterLiveMutations(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "snapshots.db")
	store, err := Open(path, Options{BusyTimeout: time.Second, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	clock := sqliteFixedClock{now: time.Date(2026, 10, 2, 23, 20, 0, 0, time.UTC)}
	service, err := application.NewService(
		store,
		clock,
		&sqliteSequenceIDs{prefix: "0199f410", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	namespaceID := domain.MustParseID("0199f411-0000-7000-8000-000000000001")
	outcomeResult, err := service.CreateOutcome(ctx, sqliteCommandContext(
		"0199f411-0000-7000-8000-000000000101", "",
	), application.CreateOutcomeCommand{
		NamespaceID: namespaceID, Title: "Snapshot history",
		DesiredState: "published labels survive live mutations", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := outcomeResult.Value
	objectiveResult, err := service.CreateObjective(ctx, sqliteCommandContext(
		"0199f411-0000-7000-8000-000000000102", "",
	), application.CreateObjectiveCommand{
		Scope: outcome.Scope(), Title: "Original objective", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	objective := objectiveResult.Value
	criterionResult, err := service.AddCriterion(ctx, sqliteCommandContext(
		"0199f411-0000-7000-8000-000000000103", "",
	), application.AddCriterionCommand{
		Owner: objective.Ref(), ExpectedVersion: objective.Version,
		Title: "Original criterion", Required: true,
		VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	criterion := criterionResult.Value

	roadmapResult, err := service.CreateRoadmap(ctx, sqliteCommandContext(
		"0199f411-0000-7000-8000-000000000104", "",
	), application.CreateRoadmapCommand{
		Scope:     outcome.Scope(),
		PlanScope: domain.RoadmapPlanScope{Kind: domain.RoadmapScopeOutcome, ID: outcome.ID},
		Title:     "Snapshot plan",
	})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := service.OpenRoadmapDraft(ctx, sqliteCommandContext(
		"0199f411-0000-7000-8000-000000000105", "",
	), application.OpenRoadmapDraftCommand{
		Scope: outcome.Scope(), RoadmapID: roadmapResult.Value.ID,
		ExpectedVersion: roadmapResult.Value.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	reason := "baseline publication"
	objectiveRef := objective.Ref()
	edited, err := service.ReplaceRoadmapDraft(ctx, sqliteCommandContext(
		"0199f411-0000-7000-8000-000000000106", "",
	), application.ReplaceRoadmapDraftCommand{
		Scope: outcome.Scope(), RoadmapID: roadmapResult.Value.ID,
		ExpectedVersion:      opened.Value.Version,
		ExpectedDraftVersion: opened.Value.Draft.DraftVersion,
		Reason:               &reason,
		Nodes: []domain.RoadmapNode{
			{
				NodeKey: "objective", NodeType: domain.RoadmapNodeReference,
				TargetRef: &objectiveRef, Title: "Plan reference", Position: 0,
			},
			{
				NodeKey: "milestone", NodeType: domain.RoadmapNodeMilestone,
				Title: "Criterion milestone", Position: 1,
				CriterionRefs: []domain.RoadmapCriterionRef{
					{OwnerRef: objective.Ref(), CriterionID: criterion.ID},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	published, err := service.PublishRoadmapDraft(ctx, sqliteCommandContext(
		"0199f411-0000-7000-8000-000000000107", "",
	), application.PublishRoadmapDraftCommand{
		Scope: outcome.Scope(), RoadmapID: roadmapResult.Value.ID,
		ExpectedVersion:      edited.Value.Version,
		ExpectedDraftVersion: edited.Value.Draft.DraftVersion,
	})
	if err != nil {
		t.Fatal(err)
	}

	currentObjective, err := service.GetObjective(ctx, outcome.Scope(), objective.ID)
	if err != nil {
		t.Fatal(err)
	}
	mutatedTitle := "Mutated objective"
	updatedObjective, err := service.UpdateObjective(ctx, sqliteCommandContext(
		"0199f411-0000-7000-8000-000000000108", "",
	), application.UpdateObjectiveCommand{
		Scope: outcome.Scope(), ObjectiveID: objective.ID,
		ExpectedVersion: currentObjective.Value.Version, Title: &mutatedTitle,
	})
	if err != nil {
		t.Fatal(err)
	}
	mutatedCriterionTitle := "Mutated criterion"
	_, err = service.ReviseCriterion(ctx, sqliteCommandContext(
		"0199f411-0000-7000-8000-000000000109", "",
	), application.ReviseCriterionCommand{
		Owner: objective.Ref(), CriterionID: criterion.ID,
		ExpectedVersion: updatedObjective.Value.Version,
		Title:           mutatedCriterionTitle, Required: true,
		VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}

	reopened := reopenIntegrationFixture(t, store, path)
	defer reopened.Close()
	restarted, err := application.NewService(
		reopened, clock, &sqliteSequenceIDs{prefix: "0199f412", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	revision, err := restarted.GetRoadmapRevision(
		ctx, outcome.Scope(), roadmapResult.Value.ID, 1,
	)
	if err != nil {
		t.Fatal(err)
	}
	if revision.Value.Reason != reason {
		t.Fatalf("revision reason = %q, want %q", revision.Value.Reason, reason)
	}
	nodesByKey := make(map[string]domain.RoadmapNode, len(revision.Value.Nodes))
	for _, node := range revision.Value.Nodes {
		nodesByKey[node.NodeKey] = node
	}
	referenceNode, ok := nodesByKey["objective"]
	if !ok {
		t.Fatalf("objective node missing after restart: %#v", revision.Value.Nodes)
	}
	if referenceNode.ReferenceSnapshot == nil ||
		referenceNode.ReferenceSnapshot.Title != "Original objective" {
		t.Fatalf("reference snapshot = %#v", referenceNode.ReferenceSnapshot)
	}
	milestoneNode, ok := nodesByKey["milestone"]
	if !ok {
		t.Fatalf("milestone node missing after restart: %#v", revision.Value.Nodes)
	}
	if len(milestoneNode.CriterionSnapshots) != 1 ||
		milestoneNode.CriterionSnapshots[0].Title != "Original criterion" ||
		milestoneNode.CriterionSnapshots[0].CriterionRevision != criterion.Revision {
		t.Fatalf("criterion snapshot = %#v", milestoneNode.CriterionSnapshots)
	}
	if published.Value.Revisions[0].ContentHash != revision.Value.ContentHash {
		t.Fatalf("content hash changed across restart: %q != %q", published.Value.Revisions[0].ContentHash, revision.Value.ContentHash)
	}
}
