package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/storage/memory"
)

func TestSaveRejectsStructurallyInvalidAggregate(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	namespaceID := domain.MustParseID("0199e600-0000-7000-8000-000000000001")
	outcomeID := domain.MustParseID("0199e600-0000-7000-8000-000000000010")
	now := time.Date(2026, 10, 1, 23, 30, 0, 0, time.UTC)

	outcome, err := domain.NewOutcome(
		outcomeID,
		namespaceID,
		"Validate storage boundary",
		"",
		"State",
		domain.PriorityNormal,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Outcomes().Insert(ctx, outcome); err != nil {
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

	persisted, err := tx.Outcomes().Get(ctx, namespaceID, outcomeID)
	if err != nil {
		t.Fatal(err)
	}
	expected := persisted.Version
	persisted.Version++
	persisted.Lifecycle = domain.OutcomeLifecycle("corrupt")

	if err := tx.Outcomes().Save(ctx, persisted, expected); err == nil {
		t.Fatal("Save() unexpectedly accepted invalid aggregate state")
	}
}
