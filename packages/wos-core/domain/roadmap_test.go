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

func TestRoadmapDraftLifecycleAndImmutablePublication(t *testing.T) {
	scope := roadmapTestScope()
	now := time.Date(2026, 10, 2, 22, 30, 0, 0, time.UTC)
	roadmap, err := NewRoadmap(
		MustParseID("0199f100-0000-7000-8000-000000000050"),
		scope,
		RoadmapPlanScope{Kind: RoadmapScopeOutcome, ID: scope.OutcomeID},
		"Main plan",
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := roadmap.OpenDraft(nil, now); err != nil {
		t.Fatal(err)
	}
	target := EntityRef{
		Scope: scope,
		Kind:  EntityKindWorkItem,
		ID:    MustParseID("0199f100-0000-7000-8000-000000000060"),
	}
	nodes := []RoadmapNode{
		{
			NodeKey:   "work",
			NodeType:  RoadmapNodeReference,
			TargetRef: &target,
			Title:     "Ship work",
			Position:  0,
		},
	}
	if err := roadmap.ReplaceDraft(1, nodes, nil, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	publishedNodes := cloneRoadmapNodes(nodes)
	publishedNodes[0].ReferenceSnapshot = &RoadmapReferenceSnapshot{
		TargetRef: target,
		Title:     "Ship work",
	}
	revision, err := roadmap.PublishDraft(
		2,
		"sha256:revision-one",
		publishedNodes,
		nil,
		ActorRef{Kind: ActorKindService, Provider: "test", ID: "planner"},
		now.Add(2*time.Minute),
	)
	if err != nil {
		t.Fatal(err)
	}
	if revision.RevisionNumber != 1 || len(roadmap.Revisions) != 1 || roadmap.Draft != nil {
		t.Fatalf("published roadmap = %#v", roadmap)
	}

	originalTitle := roadmap.Revisions[0].Nodes[0].Title
	publishedNodes[0].Title = "mutated caller value"
	if roadmap.Revisions[0].Nodes[0].Title != originalTitle {
		t.Fatal("published revision aliases caller-owned node memory")
	}

	base := uint64(1)
	if err := roadmap.OpenDraft(&base, now.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if roadmap.Draft.Nodes[0].ReferenceSnapshot != nil {
		t.Fatal("draft cloned from revision retained immutable publication snapshot")
	}
	if err := roadmap.DiscardDraft(1, now.Add(4*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := roadmap.PublishDraft(
		2,
		"sha256:invalid",
		publishedNodes,
		nil,
		ActorRef{Kind: ActorKindService, Provider: "test", ID: "planner"},
		now.Add(5*time.Minute),
	); err == nil {
		t.Fatal("discarded draft unexpectedly published")
	}
}

func TestRoadmapParentMustBePhase(t *testing.T) {
	scope := roadmapTestScope()
	now := time.Date(2026, 10, 2, 22, 0, 0, 0, time.UTC)
	draft := RoadmapDraft{
		DraftVersion: 1,
		Lifecycle:    RoadmapDraftOpen,
		CreatedAt:    now,
		UpdatedAt:    now,
		Nodes: []RoadmapNode{
			{NodeKey: "milestone", NodeType: RoadmapNodeMilestone, Title: "M", Position: 0},
			{NodeKey: "child", NodeType: RoadmapNodePhase, ParentNodeKey: "milestone", Title: "Child", Position: 1},
		},
	}
	if err := draft.Validate(scope); err == nil {
		t.Fatal("non-phase parent unexpectedly accepted")
	}
}
