package domain

import (
	"testing"
	"time"
)

func roadmapTestScope() Scope {
	return Scope{
		NamespaceID: MustParseID("0199f100-0000-7000-8000-000000000001"),
		OutcomeID:   MustParseID("0199f100-0000-7000-8000-000000000010"),
	}
}

func TestRoadmapDraftRejectsDuplicateReferenceAndCycles(t *testing.T) {
	scope := roadmapTestScope()
	objective := EntityRef{
		Scope: scope,
		Kind:  EntityKindObjective,
		ID:    MustParseID("0199f100-0000-7000-8000-000000000020"),
	}
	now := time.Date(2026, 10, 2, 22, 0, 0, 0, time.UTC)

	duplicate := RoadmapDraft{
		DraftVersion: 1,
		Lifecycle:    RoadmapDraftOpen,
		CreatedAt:    now,
		UpdatedAt:    now,
		Nodes: []RoadmapNode{
			{NodeKey: "a", NodeType: RoadmapNodeReference, TargetRef: &objective, Title: "Objective", Position: 0},
			{NodeKey: "b", NodeType: RoadmapNodeReference, TargetRef: &objective, Title: "Objective again", Position: 1},
		},
	}
	if err := duplicate.Validate(scope); err == nil {
		t.Fatal("duplicate reference target unexpectedly validated")
	}

	parentCycle := duplicate
	parentCycle.Nodes = []RoadmapNode{
		{NodeKey: "a", NodeType: RoadmapNodePhase, ParentNodeKey: "b", Title: "A", Position: 0},
		{NodeKey: "b", NodeType: RoadmapNodePhase, ParentNodeKey: "a", Title: "B", Position: 1},
	}
	if err := parentCycle.Validate(scope); err == nil {
		t.Fatal("parent cycle unexpectedly validated")
	}

	afterCycle := duplicate
	afterCycle.Nodes = []RoadmapNode{
		{NodeKey: "a", NodeType: RoadmapNodePhase, Title: "A", Position: 0},
		{NodeKey: "b", NodeType: RoadmapNodePhase, Title: "B", Position: 1},
	}
	afterCycle.AfterLinks = []RoadmapAfterLink{
		{NodeKey: "a", AfterNodeKey: "b"},
		{NodeKey: "b", AfterNodeKey: "a"},
	}
	if err := afterCycle.Validate(scope); err == nil {
		t.Fatal("after cycle unexpectedly validated")
	}
}

func TestPublishedRoadmapReferenceRequiresHistoricalSnapshot(t *testing.T) {
	scope := roadmapTestScope()
	target := EntityRef{
		Scope: scope,
		Kind:  EntityKindWorkItem,
		ID:    MustParseID("0199f100-0000-7000-8000-000000000030"),
	}
	revision := RoadmapRevision{
		RevisionNumber: 1,
		ContentHash:    "sha256:test",
		PublishedBy:    ActorRef{Kind: ActorKindService, Provider: "test", ID: "planner"},
		PublishedAt:    time.Date(2026, 10, 2, 22, 0, 0, 0, time.UTC),
		Nodes: []RoadmapNode{
			{
				NodeKey:   "work",
				NodeType:  RoadmapNodeReference,
				TargetRef: &target,
				Title:     "Deliver implementation",
				Position:  0,
			},
		},
	}
	if err := revision.Validate(scope); err == nil {
		t.Fatal("published reference without snapshot unexpectedly validated")
	}
	revision.Nodes[0].ReferenceSnapshot = &RoadmapReferenceSnapshot{
		TargetRef: target,
		Title:     "Deliver implementation",
	}
	if err := revision.Validate(scope); err != nil {
		t.Fatal(err)
	}
}

func TestRoadmapScopeIsOutcomeBounded(t *testing.T) {
	scope := roadmapTestScope()
	roadmap, err := NewRoadmap(
		MustParseID("0199f100-0000-7000-8000-000000000040"),
		scope,
		RoadmapPlanScope{Kind: RoadmapScopeOutcome, ID: scope.OutcomeID},
		"Main plan",
		time.Date(2026, 10, 2, 22, 0, 0, 0, time.UTC),
	)
	if err != nil {
		t.Fatal(err)
	}
	if roadmap.Ref().Kind != EntityKindRoadmap {
		t.Fatalf("roadmap ref kind = %q", roadmap.Ref().Kind)
	}

	_, err = NewRoadmap(
		MustParseID("0199f100-0000-7000-8000-000000000041"),
		scope,
		RoadmapPlanScope{
			Kind: RoadmapScopeOutcome,
			ID:   MustParseID("0199f100-0000-7000-8000-000000000099"),
		},
		"Invalid main plan",
		time.Date(2026, 10, 2, 22, 0, 0, 0, time.UTC),
	)
	if err == nil {
		t.Fatal("roadmap with foreign Outcome scope unexpectedly created")
	}
}
