package postgres

import (
	"context"
	"fmt"
	"math"
	"path/filepath"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestPostgresOperationalPolicyRoundTripUint64AndExpectedVersion(t *testing.T) {
	requirePostgres(t)
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "operational-policy.db")
	store := openTestStore(t, path)
	clock := sqliteFixedClock{now: time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)}
	service, err := application.NewService(store, clock, &sqliteSequenceIDs{prefix: "0199d377", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	command := func(n int) domain.CommandContext {
		return sqliteCommandContext(fmt.Sprintf("0199d377-0000-7000-8000-%012x", n), "")
	}
	outcomeResult, err := service.CreateOutcome(ctx, command(1), application.CreateOutcomeCommand{
		NamespaceID: testID("0199d377-0000-7000-8000-000000000099"), Title: "Operational policy",
		DesiredState: "Policy and unsigned completion cycle persist exactly", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	objectiveResult, err := service.CreateObjective(ctx, command(2), application.CreateObjectiveCommand{
		Scope: outcomeResult.Value.Scope(), Title: "Completion mode", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}

	uow, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := uow.Outcomes().Get(ctx, outcomeResult.Value.NamespaceID, outcomeResult.Value.ID)
	if err != nil {
		t.Fatal(err)
	}
	objective, err := uow.Objectives().Get(ctx, objectiveResult.Value.Scope, objectiveResult.Value.ID)
	if err != nil {
		t.Fatal(err)
	}
	invalid := objective
	invalid.CompletionMode = domain.ObjectiveCompletionModeAutomatic
	if err := uow.Objectives().Save(ctx, invalid, objective.Version); securityErrorCode(err) != domain.ErrorCodeInvalidArgument {
		t.Fatalf("automatic mode under compatibility Outcome save error = %v", err)
	}

	outcome.Version++
	outcome.UpdatedAt = outcome.UpdatedAt.Add(time.Second)
	outcome.OperationalProfile = domain.OperationalProfileOperationalV2
	outcome.CompletionPolicy = domain.CompletionPolicyStrictWork
	if err := uow.Outcomes().Save(ctx, outcome, outcome.Version-1); err != nil {
		t.Fatal(err)
	}
	objective.Version++
	objective.UpdatedAt = objective.UpdatedAt.Add(time.Second)
	objective.CompletionMode = domain.ObjectiveCompletionModeAutomatic
	objective.CompletionCycle = math.MaxUint64
	if err := uow.Objectives().Save(ctx, objective, objective.Version-1); err != nil {
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, err := openTestPostgres(t, path, Options{MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = restarted.Close() })
	read, err := restarted.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer read.Rollback()
	gotOutcome, err := read.Outcomes().Get(ctx, outcome.NamespaceID, outcome.ID)
	if err != nil {
		t.Fatal(err)
	}
	gotObjective, err := read.Objectives().Get(ctx, objective.Scope, objective.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotOutcome.OperationalProfile != domain.OperationalProfileOperationalV2 || gotOutcome.CompletionPolicy != domain.CompletionPolicyStrictWork {
		t.Fatalf("Outcome policy after restart = %q/%q", gotOutcome.OperationalProfile, gotOutcome.CompletionPolicy)
	}
	if gotObjective.CompletionMode != domain.ObjectiveCompletionModeAutomatic || gotObjective.CompletionCycle != math.MaxUint64 {
		t.Fatalf("Objective policy after restart = %q/cycle %d", gotObjective.CompletionMode, gotObjective.CompletionCycle)
	}
	if err := read.Rollback(); err != nil {
		t.Fatal(err)
	}

	firstWriter, err := restarted.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	current, err := firstWriter.Objectives().Get(ctx, gotObjective.Scope, gotObjective.ID)
	if err != nil {
		t.Fatal(err)
	}
	firstUpdate := current
	firstUpdate.Version++
	firstUpdate.UpdatedAt = firstUpdate.UpdatedAt.Add(time.Second)
	firstUpdate.Title = "First expected-version writer"
	if err := firstWriter.Objectives().Save(ctx, firstUpdate, current.Version); err != nil {
		t.Fatal(err)
	}
	if err := firstWriter.Commit(); err != nil {
		t.Fatal(err)
	}
	staleWriter, err := restarted.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer staleWriter.Rollback()
	staleUpdate := gotObjective
	staleUpdate.Version++
	staleUpdate.UpdatedAt = staleUpdate.UpdatedAt.Add(2 * time.Second)
	if err := staleWriter.Objectives().Save(ctx, staleUpdate, gotObjective.Version); securityErrorCode(err) != domain.ErrorCodeVersionConflict {
		t.Fatalf("stale expected-version Objective save error = %v", err)
	}

	downgrade := gotOutcome
	downgrade.Version++
	downgrade.UpdatedAt = downgrade.UpdatedAt.Add(time.Second)
	downgrade.OperationalProfile = domain.OperationalProfileCompatibilityV1
	downgrade.CompletionPolicy = domain.CompletionPolicyCompatible
	policyCheck, err := restarted.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer policyCheck.Rollback()
	if err := policyCheck.Outcomes().Save(ctx, downgrade, gotOutcome.Version); securityErrorCode(err) != domain.ErrorCodeInvalidArgument {
		t.Fatalf("Outcome policy downgrade with automatic Objective error = %v", err)
	}
}

func TestPostgresOperationalProfileMigrationDefaultsExistingRows(t *testing.T) {
	requirePostgres(t)
	ctx := context.Background()
	store, _ := openStoreAtSchema33(t)
	t.Cleanup(func() { _ = store.Close() })
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	var migration37 migration
	for _, item := range migrations {
		if item.version > 33 && item.version <= 36 {
			if err := store.applyMigration(ctx, item); err != nil {
				t.Fatalf("apply historical migration %s: %v", item.name, err)
			}
		}
		if item.version == 37 {
			migration37 = item
		}
	}
	if migration37.version != 37 {
		t.Fatal("migration 37 is missing")
	}
	namespaceID := testID("0199d378-0000-7000-8000-000000000001")
	outcomeID := testID("0199d378-0000-7000-8000-000000000002")
	objectiveID := testID("0199d378-0000-7000-8000-000000000003")
	scope := seedLegacyOutcomeObjective(t, store, namespaceID, outcomeID, objectiveID,
		"Legacy outcome", "Existing users keep compatibility semantics", "Legacy objective")
	if err := store.applyMigration(ctx, migration37); err != nil {
		t.Fatal(err)
	}
	if version, err := store.SchemaVersion(ctx); err != nil || version != 37 {
		t.Fatalf("schema version after migration = %d, err=%v; want 37", version, err)
	}
	// Repository reads use the current schema, so first finish the upgrade chain
	// after recording the legacy defaults written by migration 37.
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("apply migrations after the schema-37 defaults: %v", err)
	}
	uow, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	outcome, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		t.Fatal(err)
	}
	objectives, err := uow.Objectives().ListByOutcome(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if outcome.OperationalProfile.Effective() != domain.OperationalProfileCompatibilityV1 {
		t.Fatalf("upgraded legacy Outcome profile = %q", outcome.OperationalProfile)
	}
	if len(objectives) != 1 || objectives[0].CompletionMode.Effective() != domain.ObjectiveCompletionModeManual || objectives[0].EffectiveCompletionCycle() != 1 {
		t.Fatalf("upgraded legacy Objective = %#v", objectives)
	}
}
