package memory

import (
	"context"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func TestRoadmapRepositoryPreservesPublishedRevisionImmutability(t *testing.T) {
	ctx := context.Background()
	store := New()
	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	planning, ok := tx.(ports.PlanningUnitOfWork)
	if !ok {
		t.Fatal("memory transaction does not expose planning repositories")
	}

	scope := domain.Scope{
		NamespaceID: domain.MustParseID("0199f200-0000-7000-8000-000000000001"),
		OutcomeID:   domain.MustParseID("0199f200-0000-7000-8000-000000000010"),
	}
	now := time.Date(2026, 10, 2, 22, 0, 0, 0, time.UTC)
	roadmap, err := domain.NewRoadmap(
		domain.MustParseID("0199f200-0000-7000-8000-000000000020"),
		scope,
		domain.RoadmapPlanScope{Kind: domain.RoadmapScopeOutcome, ID: scope.OutcomeID},
		"Main plan",
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := planning.Roadmaps().Insert(ctx, roadmap); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	planning = tx.(ports.PlanningUnitOfWork)
	current, err := planning.Roadmaps().Get(ctx, scope, roadmap.ID)
	if err != nil {
		t.Fatal(err)
	}
	expected := current.Version
	if err := current.OpenDraft(nil, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := planning.Roadmaps().Save(ctx, current, expected); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	planning = tx.(ports.PlanningUnitOfWork)
	loaded, err := planning.Roadmaps().Get(ctx, scope, roadmap.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Draft == nil || loaded.Draft.DraftVersion != 1 {
		t.Fatalf("persisted draft = %#v", loaded.Draft)
	}
}
