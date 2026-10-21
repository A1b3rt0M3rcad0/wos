package memory_test

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

func TestOperationalPolicyMemoryRoundTripAndCrossAggregateGuards(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	namespaceID := domain.MustParseID("0199e300-0000-7000-8000-000000000001")
	outcomeID := domain.MustParseID("0199e300-0000-7000-8000-000000000010")
	objectiveID := domain.MustParseID("0199e300-0000-7000-8000-000000000020")
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)

	outcome, err := domain.NewOutcome(outcomeID, namespaceID, "Operational profile", "", "Persist the configuration", domain.PriorityNormal, now)
	if err != nil {
		t.Fatal(err)
	}
	objective, err := domain.NewObjective(objectiveID, outcome.Scope(), "Completion mode", "", domain.PriorityNormal, false, now)
	if err != nil {
		t.Fatal(err)
	}

	write, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := write.Outcomes().Insert(ctx, outcome); err != nil {
		t.Fatal(err)
	}
	objective.CompletionMode = domain.ObjectiveCompletionModeAutomatic
	if err := write.Objectives().Insert(ctx, objective); err == nil {
		t.Fatal("automatic Objective was inserted into compatibility Outcome")
	}
	if _, err := write.Objectives().Get(ctx, outcome.Scope(), objectiveID); err == nil {
		t.Fatal("rejected automatic Objective unexpectedly persisted")
	}

	outcome.OperationalProfile = domain.OperationalProfileOperationalV2
	outcome.CompletionPolicy = domain.CompletionPolicyStrictWork
	priorOutcomeVersion := outcome.Version
	outcome.Version++
	if err := write.Outcomes().Save(ctx, outcome, priorOutcomeVersion); err != nil {
		t.Fatal(err)
	}
	objective.CompletionCycle = math.MaxUint64
	if err := write.Objectives().Insert(ctx, objective); err != nil {
		t.Fatalf("valid operational_v2/strict_work Objective was rejected: %v", err)
	}
	if err := write.Commit(); err != nil {
		t.Fatal(err)
	}

	read, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer read.Rollback()
	storedOutcome, err := read.Outcomes().Get(ctx, namespaceID, outcomeID)
	if err != nil {
		t.Fatal(err)
	}
	storedObjective, err := read.Objectives().Get(ctx, outcome.Scope(), objectiveID)
	if err != nil {
		t.Fatal(err)
	}
	if storedOutcome.OperationalProfile != domain.OperationalProfileOperationalV2 || storedOutcome.CompletionPolicy != domain.CompletionPolicyStrictWork {
		t.Fatalf("Outcome policy round trip = %q/%q", storedOutcome.OperationalProfile, storedOutcome.CompletionPolicy)
	}
	if storedObjective.CompletionMode != domain.ObjectiveCompletionModeAutomatic || storedObjective.CompletionCycle != math.MaxUint64 {
		t.Fatalf("Objective policy round trip = %q/cycle %d", storedObjective.CompletionMode, storedObjective.CompletionCycle)
	}

	incompatible := storedOutcome
	incompatible.OperationalProfile = domain.OperationalProfileCompatibilityV1
	incompatible.CompletionPolicy = domain.CompletionPolicyCompatible
	incompatible.Version++
	if err := read.Outcomes().Save(ctx, incompatible, storedOutcome.Version); err == nil {
		t.Fatal("Outcome policy update invalidating an existing automatic Objective unexpectedly succeeded")
	}

	updated := storedObjective
	updated.CompletionMode = domain.ObjectiveCompletionModeManual
	updated.Version++
	if err := read.Objectives().Save(ctx, updated, storedObjective.Version); err != nil {
		t.Fatal(err)
	}
	if err := read.Commit(); err != nil {
		t.Fatal(err)
	}

	final, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer final.Rollback()
	finalObjective, err := final.Objectives().Get(ctx, outcome.Scope(), objectiveID)
	if err != nil {
		t.Fatal(err)
	}
	if finalObjective.CompletionMode != domain.ObjectiveCompletionModeManual || finalObjective.CompletionCycle != math.MaxUint64 {
		t.Fatalf("expected-version Objective save round trip = %q/cycle %d", finalObjective.CompletionMode, finalObjective.CompletionCycle)
	}
}
