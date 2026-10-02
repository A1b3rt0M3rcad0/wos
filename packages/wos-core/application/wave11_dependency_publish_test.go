package application_test

import (
	"context"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestWave11PublicationAppliesDependencyChangesAtomically(t *testing.T) {
	ctx := context.Background()
	service, store := newWave09Service(t)
	cc := commandContext()

	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  domain.MustParseID("0199f320-0000-7000-8000-000000000001"),
		Title:        "Atomic planning publication",
		DesiredState: "plan publication and dependency mutations commit together",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value

	firstResult, err := service.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{
		Scope: outcome.Scope(), Title: "Prerequisite", Priority: domain.PriorityNormal,
		Lifecycle: domain.WorkItemLifecycleTodo,
	})
	if err != nil {
		t.Fatal(err)
	}
	first := firstResult.Value
	secondResult, err := service.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{
		Scope: outcome.Scope(), Title: "Dependent", Priority: domain.PriorityNormal,
		Lifecycle: domain.WorkItemLifecycleTodo,
	})
	if err != nil {
		t.Fatal(err)
	}
	second := secondResult.Value

	roadmapResult, err := service.CreateRoadmap(ctx, cc, application.CreateRoadmapCommand{
		Scope: outcome.Scope(),
		PlanScope: domain.RoadmapPlanScope{
			Kind: domain.RoadmapScopeOutcome,
			ID:   outcome.ID,
		},
		Title: "Atomic plan",
	})
	if err != nil {
		t.Fatal(err)
	}
	roadmap := roadmapResult.Value
	opened, err := service.OpenRoadmapDraft(ctx, cc, application.OpenRoadmapDraftCommand{
		Scope: outcome.Scope(), RoadmapID: roadmap.ID, ExpectedVersion: roadmap.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	edited, err := service.ReplaceRoadmapDraft(ctx, cc, application.ReplaceRoadmapDraftCommand{
		Scope:                outcome.Scope(),
		RoadmapID:            roadmap.ID,
		ExpectedVersion:      opened.Value.Version,
		ExpectedDraftVersion: opened.Value.Draft.DraftVersion,
		Nodes: []domain.RoadmapNode{
			{
				NodeKey: "prerequisite", NodeType: domain.RoadmapNodeReference,
				TargetRef: refPtr(first.Ref()), Title: first.Title, Position: 0,
			},
			{
				NodeKey: "dependent", NodeType: domain.RoadmapNodeReference,
				TargetRef: refPtr(second.Ref()), Title: second.Title, Position: 1,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.PublishRoadmapDraft(ctx, cc, application.PublishRoadmapDraftCommand{
		Scope:                outcome.Scope(),
		RoadmapID:            roadmap.ID,
		ExpectedVersion:      edited.Value.Version,
		ExpectedDraftVersion: edited.Value.Draft.DraftVersion,
		DependencyChanges: []application.RoadmapDependencyChange{
			{
				Action:    application.RoadmapDependencyChangeAdd,
				SourceRef: second.Ref(), TargetRef: first.Ref(),
				Strength: domain.DependencyStrengthHard,
			},
			{
				Action:          application.RoadmapDependencyChangeRemove,
				RelationID:      domain.MustParseID("0199f320-0000-7000-8000-000000000099"),
				ExpectedVersion: domain.InitialVersion,
				Reason:          "force rollback",
			},
		},
	})
	if err == nil {
		t.Fatal("invalid dependency batch unexpectedly published")
	}

	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	relations, err := tx.Relations().ListByOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	persistedRoadmap, err := tx.(interface {
		Roadmaps() interface {
			Get(context.Context, domain.Scope, domain.ID) (domain.Roadmap, error)
		}
	}).Roadmaps().Get(ctx, outcome.Scope(), roadmap.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if len(relations) != 0 {
		t.Fatalf("failed publication leaked %d dependency relations", len(relations))
	}
	if persistedRoadmap.Version != edited.Value.Version ||
		persistedRoadmap.Draft == nil ||
		len(persistedRoadmap.Revisions) != 0 {
		t.Fatalf("failed publication mutated Roadmap = %#v", persistedRoadmap)
	}

	published, err := service.PublishRoadmapDraft(ctx, cc, application.PublishRoadmapDraftCommand{
		Scope:                outcome.Scope(),
		RoadmapID:            roadmap.ID,
		ExpectedVersion:      edited.Value.Version,
		ExpectedDraftVersion: edited.Value.Draft.DraftVersion,
		DependencyChanges: []application.RoadmapDependencyChange{
			{
				Action:    application.RoadmapDependencyChangeAdd,
				SourceRef: second.Ref(), TargetRef: first.Ref(),
				Strength: domain.DependencyStrengthHard,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(published.Value.Revisions) != 1 ||
		len(published.Value.Revisions[0].DependencySnapshots) != 1 {
		t.Fatalf("published dependency snapshot = %#v", published.Value.Revisions)
	}

	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	relations, err = tx.Relations().ListByOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if len(relations) != 1 ||
		relations[0].Lifecycle != domain.RelationLifecycleActive ||
		relations[0].SourceRef != second.Ref() ||
		relations[0].TargetRef != first.Ref() {
		t.Fatalf("committed dependency relations = %#v", relations)
	}
}
