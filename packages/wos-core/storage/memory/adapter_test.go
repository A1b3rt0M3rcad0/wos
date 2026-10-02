package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/storage/memory"
)

func TestRollbackDiscardsChanges(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	ns := domain.MustParseID("0199e100-0000-7000-8000-000000000001")
	outID := domain.MustParseID("0199e100-0000-7000-8000-000000000010")
	outcome, err := domain.NewOutcome(outID, ns, "Rollback", "", "No state after rollback", domain.PriorityNormal, time.Now().UTC())
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
	if _, err := tx.Coordination().AdvanceOutcome(ctx, outcome.Scope()); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	readTx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer readTx.Rollback()
	if _, err := readTx.Outcomes().Get(ctx, ns, outID); err == nil {
		t.Fatal("rolled back outcome unexpectedly persisted")
	}
	coord, err := readTx.Coordination().LockOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if coord.Revision != 0 {
		t.Fatalf("revision = %d, want 0 after rollback", coord.Revision)
	}
}

func TestCommitPersistsAndVersionConflictIsDetected(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	ns := domain.MustParseID("0199e100-0000-7000-8000-000000000001")
	outID := domain.MustParseID("0199e100-0000-7000-8000-000000000010")
	now := time.Now().UTC()
	outcome, err := domain.NewOutcome(outID, ns, "Persist", "", "State", domain.PriorityNormal, now)
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
	if _, err := tx.Coordination().AdvanceOutcome(ctx, outcome.Scope()); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	tx2, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	current, err := tx2.Outcomes().Get(ctx, ns, outID)
	if err != nil {
		t.Fatal(err)
	}
	criterionID := domain.MustParseID("0199e100-0000-7000-8000-000000000040")
	criterion, err := domain.NewSuccessCriterion(criterionID, current.Ref(), "Verified", "", true, domain.VerificationModeAttestation)
	if err != nil {
		t.Fatal(err)
	}
	expected := current.Version
	if err := current.AddCriterion(criterion, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := tx2.Outcomes().Save(ctx, current, expected-1); err == nil {
		t.Fatal("Save() unexpectedly accepted stale expected_version")
	}
	if err := tx2.Rollback(); err != nil {
		t.Fatal(err)
	}

	tx3, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx3.Rollback()
	coord, err := tx3.Coordination().LockOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if coord.Revision != 1 {
		t.Fatalf("revision = %d, want 1", coord.Revision)
	}
}
