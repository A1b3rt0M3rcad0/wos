package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func openWorkItemObjectiveSchema34(t *testing.T) (*Store, string, migration) {
	t.Helper()
	store, path := openStoreAtSchema33(t)
	ctx := context.Background()
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	var contract migration
	for _, item := range migrations {
		if item.version == 34 {
			contract = item
			break
		}
	}
	if contract.version != 34 {
		t.Fatal("schema 34 migration not found")
	}
	if err := store.applyMigration(ctx, contract); err != nil {
		t.Fatalf("apply schema 34: %v", err)
	}
	version, err := store.SchemaVersion(ctx)
	if err != nil || version != 34 {
		t.Fatalf("schema version = %d, err=%v; want 34", version, err)
	}
	var required migration
	for _, item := range migrations {
		if item.version == 35 {
			required = item
			break
		}
	}
	if required.version != 35 {
		t.Fatal("schema 35 migration not found")
	}
	return store, path, required
}

func createWorkItemObjectiveMigrationFixture(t *testing.T, store *Store) (domain.Scope, domain.ID, domain.ID) {
	t.Helper()
	ctx := context.Background()
	namespaceID := testID("0199f237-0000-7000-8000-000000000001")
	outcomeID := testID("0199f237-0000-7000-8000-000000000002")
	objectiveID := testID("0199f237-0000-7000-8000-000000000003")
	workID := testID("0199f237-0000-7000-8000-000000000004")
	scope := seedLegacyOutcomeObjective(t, store, namespaceID, outcomeID, objectiveID,
		"Migration contract", "Every Task remains explicitly owned", "Task owner")
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if err := insertEntityRef(ctx, tx, domain.EntityRef{Scope: scope, Kind: domain.EntityKindWorkItem, ID: workID}); err != nil {
		t.Fatal(err)
	}
	now := encodeTime(time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC))
	if _, err := tx.ExecContext(ctx, `INSERT INTO work_items(id,namespace_id,outcome_id,version,created_at,updated_at,title,description,priority,objective_id,lifecycle) VALUES(?,?,?,1,?,?,?,'Migration fixture task','high',?,'backlog')`,
		workID.String(), namespaceID.String(), outcomeID.String(), now, now, "Preserved Task", objectiveID.String()); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return scope, objectiveID, workID
}

func addWorkContractForeignKeyFixtures(t *testing.T, store *Store, scope domain.Scope, workID domain.ID) {
	t.Helper()
	ctx := context.Background()
	principalID := testID("0199f238-0000-7000-8000-000000000001")
	unsignedContractID := testID("0199f238-0000-7000-8000-000000000002")
	signedContractID := testID("0199f238-0000-7000-8000-000000000003")
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO principals(id,display_name,kind,lifecycle,created_at)
VALUES(?, 'Migration fixture', 'human', 'active', 1)`, principalID.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO work_contracts(
id,namespace_id,outcome_id,work_item_id,holder_principal_id,status,version,lease_version,
fencing_token,expires_at,state_json,spec_json,spec_digest
) VALUES(?,?,?,?,?,'completed',1,1,'1',1,'{}','{}','migration-spec')`,
		unsignedContractID.String(), scope.NamespaceID.String(), scope.OutcomeID.String(), workID.String(), principalID.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO signed_work_contracts(
id,namespace_id,outcome_id,work_item_id,credential_id,holder_principal_id,status,version,
lease_version,fencing_token,expires_at,state_json,spec_json,spec_digest
) VALUES(?,?,?,?,?,?,'completed','1','1','1',1,'{}','{}','migration-signed-spec')`,
		signedContractID.String(), scope.NamespaceID.String(), scope.OutcomeID.String(), workID.String(), "migration-credential", principalID.String()); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
}

func workItemObjectiveContractMigration(t *testing.T, store *Store, required migration) {
	t.Helper()
	ctx := context.Background()
	scope, objectiveID, workID := createWorkItemObjectiveMigrationFixture(t, store)
	addWorkContractForeignKeyFixtures(t, store, scope, workID)
	if err := store.applyMigration(ctx, required); err != nil {
		t.Fatalf("apply Objective contract migration: %v", err)
	}
	version, err := store.SchemaVersion(ctx)
	if err != nil || version != 35 {
		t.Fatalf("schema version = %d, err=%v; want 35", version, err)
	}
	var title, persistedObjective string
	if err := store.db.QueryRowContext(ctx, `SELECT title, objective_id FROM work_items WHERE id = ?`, workID.String()).Scan(&title, &persistedObjective); err != nil {
		t.Fatal(err)
	}
	if title != "Preserved Task" || persistedObjective != objectiveID.String() {
		t.Fatalf("Task changed across contract migration: title=%q objective=%q", title, persistedObjective)
	}
	for _, table := range []string{"work_contracts", "signed_work_contracts"} {
		var count int
		query := fmt.Sprintf(`SELECT COUNT(*) FROM %s WHERE work_item_id = ?`, table)
		if err := store.db.QueryRowContext(ctx, query, workID.String()).Scan(&count); err != nil || count != 1 {
			t.Fatalf("%s no longer references preserved Task: count=%d err=%v", table, count, err)
		}
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE work_items SET objective_id = NULL WHERE id = ?`, workID.String()); err == nil {
		t.Fatal("schema 35 accepted a Task without Objective")
	}
}

func TestWorkItemObjectiveContractMigrationPreservesTasksAndContractForeignKeys(t *testing.T) {
	store, _, required := openWorkItemObjectiveSchema34(t)
	workItemObjectiveContractMigration(t, store, required)
}

func TestWorkItemObjectiveContractMigrationRejectsOrphanAndCanRetry(t *testing.T) {
	ctx := context.Background()
	store, _, required := openWorkItemObjectiveSchema34(t)
	if required.version != 35 {
		t.Fatalf("migration fixture version = %d; want 35", required.version)
	}
	scope, objectiveID, workID := createWorkItemObjectiveMigrationFixture(t, store)
	if _, err := store.db.ExecContext(ctx, `UPDATE work_items SET objective_id = NULL WHERE id = ?`, workID.String()); err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(ctx); err == nil || !strings.Contains(err.Error(), "found 1") {
		t.Fatalf("migration with an orphan Task error = %v, want explicit zero-orphan gate", err)
	}
	version, err := store.SchemaVersion(ctx)
	if err != nil || version != 34 {
		t.Fatalf("failed migration version = %d, err=%v; want unchanged schema 34", version, err)
	}
	var title string
	var owner sql.NullString
	if err := store.db.QueryRowContext(ctx, `SELECT title, objective_id FROM work_items WHERE id = ?`, workID.String()).Scan(&title, &owner); err != nil {
		t.Fatal(err)
	}
	if title != "Preserved Task" || owner.Valid {
		t.Fatalf("failed migration changed orphaned row: title=%q objective=%+v", title, owner)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE work_items SET objective_id = ? WHERE id = ?`, objectiveID.String(), workID.String()); err != nil {
		t.Fatal(err)
	}
	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("retry after explicit owner repair failed: %v", err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT title FROM work_items WHERE id = ?`, workID.String()).Scan(&title); err != nil || title != "Preserved Task" {
		t.Fatalf("repaired Task was not preserved: title=%q err=%v scope=%+v", title, err, scope)
	}
	version, err = store.SchemaVersion(ctx)
	if err != nil || version != 42 {
		t.Fatalf("retry schema version = %d, err=%v; want 42", version, err)
	}
}

func TestFreshSchemaRequiresWorkItemObjective(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "fresh-objective-contract.db"), Options{MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	_, _, workID := createWorkItemObjectiveMigrationFixture(t, store)
	if _, err := store.db.ExecContext(ctx, `UPDATE work_items SET objective_id = NULL WHERE id = ?`, workID.String()); err == nil {
		t.Fatal("fresh schema accepted a Task without Objective")
	}
	version, err := store.SchemaVersion(ctx)
	if err != nil || version != 42 {
		t.Fatalf("fresh schema version = %d, err=%v; want 42", version, err)
	}
}
