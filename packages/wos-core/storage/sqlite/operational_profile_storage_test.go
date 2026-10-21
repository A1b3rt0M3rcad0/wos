package sqlite

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

func newOperationalProfileFixture(t *testing.T) (*Store, *application.Service, domain.Outcome, domain.Objective) {
	t.Helper()
	return newOperationalProfileFixtureAt(t, filepath.Join(t.TempDir(), "operational-profile.db"))
}

func newOperationalProfileFixtureAt(t *testing.T, path string) (*Store, *application.Service, domain.Outcome, domain.Objective) {
	t.Helper()
	ctx := context.Background()
	store := openTestStore(t, path)
	service, err := application.NewService(store,
		sqliteFixedClock{now: time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)},
		&sqliteSequenceIDs{prefix: "0199d377", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	cc := func(n int) domain.CommandContext {
		return sqliteCommandContext(fmt.Sprintf("0199d377-0000-7000-8000-%012x", n), "")
	}
	outcome, err := service.CreateOutcome(ctx, cc(1), application.CreateOutcomeCommand{
		NamespaceID: testID("0199d377-0000-7000-8000-000000000099"), Title: "Operational policy storage",
		DesiredState: "Policy fields persist without narrowing unsigned cycles", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	objective, err := service.CreateObjective(ctx, cc(2), application.CreateObjectiveCommand{
		Scope: outcome.Value.Scope(), Title: "Storage objective", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	return store, service, outcome.Value, objective.Value
}

func TestSQLiteOperationalPolicyUpgradeDefaultsPopulatedRows(t *testing.T) {
	ctx := context.Background()
	store, scope := openStoreAtSchema36WithLegacyRows(t)
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	var migration37 migration
	for _, item := range migrations {
		if item.version == 37 {
			migration37 = item
			break
		}
	}
	if migration37.version == 0 {
		t.Fatal("migration 37 is missing")
	}
	if err := store.applyMigration(ctx, migration37); err != nil {
		t.Fatal(err)
	}
	var profile, mode, cycle string
	if err := store.db.QueryRowContext(ctx, `SELECT operational_profile FROM outcomes WHERE namespace_id=? AND id=?`, scope.NamespaceID.String(), scope.OutcomeID.String()).Scan(&profile); err != nil {
		t.Fatal(err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT completion_mode,completion_cycle FROM objectives WHERE namespace_id=? AND outcome_id=? LIMIT 1`, scope.NamespaceID.String(), scope.OutcomeID.String()).Scan(&mode, &cycle); err != nil {
		t.Fatal(err)
	}
	if profile != string(domain.OperationalProfileCompatibilityV1) || mode != string(domain.ObjectiveCompletionModeManual) || cycle != "1" {
		t.Fatalf("legacy defaults = profile %q, mode %q, cycle %q", profile, mode, cycle)
	}
	uow, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	outcome, err := uow.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil || outcome.OperationalProfile.Effective() != domain.OperationalProfileCompatibilityV1 {
		t.Fatalf("legacy Outcome read = %q, err=%v", outcome.OperationalProfile, err)
	}
	objectives, err := uow.Objectives().ListByOutcome(ctx, scope)
	if err != nil || len(objectives) == 0 || objectives[0].CompletionMode.Effective() != domain.ObjectiveCompletionModeManual || objectives[0].EffectiveCompletionCycle() != 1 {
		t.Fatalf("legacy Objective list = %#v, err=%v", objectives, err)
	}
}

func openStoreAtSchema36WithLegacyRows(t *testing.T) (*Store, domain.Scope) {
	t.Helper()
	ctx := context.Background()
	store, _ := openStoreAtSchema33(t)
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range migrations {
		if item.version > 36 {
			break
		}
		if item.version > 33 {
			if err := store.applyMigration(ctx, item); err != nil {
				t.Fatalf("apply migration %d: %v", item.version, err)
			}
		}
	}
	ns := testID("0199d378-0000-7000-8000-000000000001")
	outcomeID := testID("0199d378-0000-7000-8000-000000000002")
	objectiveID := testID("0199d378-0000-7000-8000-000000000003")
	scope := domain.Scope{NamespaceID: ns, OutcomeID: outcomeID}
	now := encodeTime(time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC))
	queries := []struct {
		query string
		args  []any
	}{
		{`INSERT INTO namespaces(id,name,lifecycle,version,created_at,updated_at) VALUES(?,?,'active',1,?,?)`, []any{ns.String(), "legacy-operational-profile", now, now}},
		{`INSERT INTO outcome_coordination(namespace_id,outcome_id,state_revision) VALUES(?,?,0)`, []any{ns.String(), outcomeID.String()}},
		{`INSERT INTO entity_refs(id,namespace_id,outcome_id,kind) VALUES(?,?,?,'outcome')`, []any{outcomeID.String(), ns.String(), outcomeID.String()}},
		{`INSERT INTO outcomes(id,namespace_id,outcome_id,version,created_at,updated_at,title,description,priority,desired_state,lifecycle) VALUES(?,?,?,1,?,?,?,?,'normal',?,'draft')`, []any{outcomeID.String(), ns.String(), outcomeID.String(), now, now, "Legacy outcome", "Retain compatibility semantics after upgrade", "Verified behavior remains stable"}},
		{`INSERT INTO entity_refs(id,namespace_id,outcome_id,kind) VALUES(?,?,?,'objective')`, []any{objectiveID.String(), ns.String(), outcomeID.String()}},
		{`INSERT INTO objectives(id,namespace_id,outcome_id,version,created_at,updated_at,title,description,priority,lifecycle,required_for_outcome) VALUES(?,?,?,1,?,?,?,?,'normal','planned',0)`, []any{objectiveID.String(), ns.String(), outcomeID.String(), now, now, "Legacy objective", "Uses manual completion by default"}},
	}
	for _, item := range queries {
		if _, err := store.db.ExecContext(ctx, item.query, item.args...); err != nil {
			t.Fatalf("seed schema 36 fixture: %v", err)
		}
	}
	return store, scope
}

func TestSQLiteOperationalPolicyRoundTripUint64AndExpectedVersion(t *testing.T) {
	ctx := context.Background()
	store, _, outcome, objective := newOperationalProfileFixture(t)
	uow, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	storedOutcome, err := uow.Outcomes().Get(ctx, outcome.NamespaceID, outcome.ID)
	if err != nil {
		t.Fatal(err)
	}
	if storedOutcome.OperationalProfile.Effective() != domain.OperationalProfileCompatibilityV1 || storedOutcome.CompletionPolicy.Effective() != domain.CompletionPolicyCompatible {
		t.Fatalf("fresh Outcome defaults = %q/%q", storedOutcome.OperationalProfile, storedOutcome.CompletionPolicy)
	}
	freshObjective, err := uow.Objectives().Get(ctx, objective.Scope, objective.ID)
	if err != nil {
		t.Fatal(err)
	}
	if freshObjective.CompletionMode.Effective() != domain.ObjectiveCompletionModeManual || freshObjective.EffectiveCompletionCycle() != 1 {
		t.Fatalf("fresh Objective defaults = %q/%d", freshObjective.CompletionMode, freshObjective.CompletionCycle)
	}
	storedOutcome.Version++
	storedOutcome.UpdatedAt = storedOutcome.UpdatedAt.Add(time.Second)
	storedOutcome.OperationalProfile = domain.OperationalProfileOperationalV2
	storedOutcome.CompletionPolicy = domain.CompletionPolicyStrictWork
	if err := uow.Outcomes().Save(ctx, storedOutcome, storedOutcome.Version-1); err != nil {
		t.Fatal(err)
	}
	storedObjective, err := uow.Objectives().Get(ctx, objective.Scope, objective.ID)
	if err != nil {
		t.Fatal(err)
	}
	storedObjective.Version++
	storedObjective.UpdatedAt = storedObjective.UpdatedAt.Add(time.Second)
	storedObjective.CompletionMode = domain.ObjectiveCompletionModeAutomatic
	storedObjective.CompletionCycle = math.MaxUint64
	if err := uow.Objectives().Save(ctx, storedObjective, storedObjective.Version-1); err != nil {
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}

	read, err := store.Begin(ctx)
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
		t.Fatalf("Outcome policies = %q/%q", gotOutcome.OperationalProfile, gotOutcome.CompletionPolicy)
	}
	if gotObjective.CompletionMode != domain.ObjectiveCompletionModeAutomatic || gotObjective.CompletionCycle != math.MaxUint64 {
		t.Fatalf("Objective policies = %q/%d", gotObjective.CompletionMode, gotObjective.CompletionCycle)
	}

	stale := gotObjective
	stale.Version++
	stale.UpdatedAt = stale.UpdatedAt.Add(time.Second)
	stale.Title = "First writer wins"
	if err := read.Objectives().Save(ctx, stale, gotObjective.Version); err != nil {
		t.Fatal(err)
	}
	if err := read.Commit(); err != nil {
		t.Fatal(err)
	}

	conflict, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conflict.Rollback()
	stale.Title = "Stale writer"
	if err := conflict.Objectives().Save(ctx, stale, gotObjective.Version); errorCode(err) != domain.ErrorCodeVersionConflict {
		t.Fatalf("stale expected-version save error = %v", err)
	}
}

func TestSQLiteOperationalPolicyPersistsAcrossStoreRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "operational-policy-restart.db")
	store, _, outcome, objective := newOperationalProfileFixtureAt(t, path)
	uow, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	storedOutcome, err := uow.Outcomes().Get(ctx, outcome.NamespaceID, outcome.ID)
	if err != nil {
		t.Fatal(err)
	}
	storedOutcome.Version++
	storedOutcome.UpdatedAt = storedOutcome.UpdatedAt.Add(time.Second)
	storedOutcome.OperationalProfile = domain.OperationalProfileOperationalV2
	storedOutcome.CompletionPolicy = domain.CompletionPolicyStrictWork
	if err := uow.Outcomes().Save(ctx, storedOutcome, storedOutcome.Version-1); err != nil {
		t.Fatal(err)
	}
	storedObjective, err := uow.Objectives().Get(ctx, objective.Scope, objective.ID)
	if err != nil {
		t.Fatal(err)
	}
	storedObjective.Version++
	storedObjective.UpdatedAt = storedObjective.UpdatedAt.Add(time.Second)
	storedObjective.CompletionMode = domain.ObjectiveCompletionModeAutomatic
	storedObjective.CompletionCycle = math.MaxUint64
	if err := uow.Objectives().Save(ctx, storedObjective, storedObjective.Version-1); err != nil {
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened := openTestStore(t, path)
	read, err := reopened.Begin(ctx)
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
		t.Fatalf("reopened Outcome policies = %q/%q", gotOutcome.OperationalProfile, gotOutcome.CompletionPolicy)
	}
	if gotObjective.CompletionMode != domain.ObjectiveCompletionModeAutomatic || gotObjective.CompletionCycle != math.MaxUint64 {
		t.Fatalf("reopened Objective policies = %q/%d", gotObjective.CompletionMode, gotObjective.CompletionCycle)
	}
}

func TestSQLiteOperationalPolicyRejectsInvalidEnumsAndCrossAggregateModes(t *testing.T) {
	ctx := context.Background()
	store, _, outcome, objective := newOperationalProfileFixture(t)

	if _, err := store.db.ExecContext(ctx, `UPDATE outcomes SET operational_profile='unknown' WHERE namespace_id=? AND id=?`, outcome.NamespaceID.String(), outcome.ID.String()); err == nil {
		t.Fatal("Outcome operational_profile CHECK accepted an unknown value")
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE objectives SET completion_mode='unknown' WHERE namespace_id=? AND outcome_id=? AND id=?`, objective.Scope.NamespaceID.String(), objective.Scope.OutcomeID.String(), objective.ID.String()); err == nil {
		t.Fatal("Objective completion_mode CHECK accepted an unknown value")
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE objectives SET completion_cycle='18446744073709551616' WHERE namespace_id=? AND outcome_id=? AND id=?`, objective.Scope.NamespaceID.String(), objective.Scope.OutcomeID.String(), objective.ID.String()); err == nil {
		t.Fatal("Objective completion_cycle CHECK accepted a value above uint64")
	}

	invalid := objective
	invalid.CompletionMode = domain.ObjectiveCompletionModeAutomatic
	uow, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := uow.Objectives().Save(ctx, invalid, objective.Version-1); errorCode(err) != domain.ErrorCodeInvalidArgument {
		t.Fatalf("automatic mode under compatibility Outcome save error = %v", err)
	}
	_ = uow.Rollback()

	setUp, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	owner, err := setUp.Outcomes().Get(ctx, outcome.NamespaceID, outcome.ID)
	if err != nil {
		t.Fatal(err)
	}
	owner.Version++
	owner.UpdatedAt = owner.UpdatedAt.Add(time.Second)
	owner.OperationalProfile = domain.OperationalProfileOperationalV2
	owner.CompletionPolicy = domain.CompletionPolicyStrictWork
	if err := setUp.Outcomes().Save(ctx, owner, owner.Version-1); err != nil {
		t.Fatal(err)
	}
	if err := setUp.Commit(); err != nil {
		t.Fatal(err)
	}

	setAutomatic, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	current, err := setAutomatic.Objectives().Get(ctx, objective.Scope, objective.ID)
	if err != nil {
		t.Fatal(err)
	}
	current.Version++
	current.UpdatedAt = current.UpdatedAt.Add(time.Second)
	current.CompletionMode = domain.ObjectiveCompletionModeAutomatic
	if err := setAutomatic.Objectives().Save(ctx, current, current.Version-1); err != nil {
		t.Fatal(err)
	}
	if err := setAutomatic.Commit(); err != nil {
		t.Fatal(err)
	}

	downgrade, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	badOutcome, err := downgrade.Outcomes().Get(ctx, outcome.NamespaceID, outcome.ID)
	if err != nil {
		t.Fatal(err)
	}
	badOutcome.Version++
	badOutcome.UpdatedAt = badOutcome.UpdatedAt.Add(time.Second)
	badOutcome.OperationalProfile = domain.OperationalProfileCompatibilityV1
	if err := downgrade.Outcomes().Save(ctx, badOutcome, badOutcome.Version-1); errorCode(err) != domain.ErrorCodeInvalidArgument {
		t.Fatalf("Outcome downgrade with automatic Objective error = %v", err)
	}
	_ = downgrade.Rollback()

	// Simulate an out-of-band bad write to make sure aggregate and list reads
	// fail closed even when a relational policy constraint was bypassed.
	if _, err := store.db.ExecContext(ctx, `UPDATE outcomes SET operational_profile='compatibility_v1' WHERE namespace_id=? AND id=?`, outcome.NamespaceID.String(), outcome.ID.String()); err != nil {
		t.Fatal(err)
	}
	read, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer read.Rollback()
	if _, err := read.Objectives().Get(ctx, objective.Scope, objective.ID); errorCode(err) != domain.ErrorCodeInvalidArgument {
		t.Fatalf("Objective get with invalid owning policy error = %v", err)
	}
	if _, err := read.Objectives().ListByOutcome(ctx, objective.Scope); errorCode(err) != domain.ErrorCodeInvalidArgument {
		t.Fatalf("Objective list with invalid owning policy error = %v", err)
	}
}

func errorCode(err error) domain.ErrorCode {
	if err == nil {
		return ""
	}
	code, _ := domain.ErrorCodeOf(err)
	return code
}
