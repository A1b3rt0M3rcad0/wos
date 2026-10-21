package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSQLiteWorkItemObjectiveMigrationPreservesExplicitIndexesTriggersAndForeignKeys(t *testing.T) {
	ctx := context.Background()
	store, _, required := openWorkItemObjectiveSchema34(t)
	_, _, workID := createWorkItemObjectiveMigrationFixture(t, store)
	if _, err := store.db.ExecContext(ctx, `CREATE INDEX migration35_work_items_title ON work_items(title);
CREATE TABLE migration35_work_item_audit(id TEXT NOT NULL, title TEXT NOT NULL);
CREATE TRIGGER migration35_work_items_title_audit AFTER UPDATE OF title ON work_items
BEGIN INSERT INTO migration35_work_item_audit(id,title) VALUES(NEW.id,NEW.title); END;`); err != nil {
		t.Fatal(err)
	}
	if err := store.applyMigration(ctx, required); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"idx_work_scope",
		"idx_work_objective",
		"work_contract_candidate_order",
		"work_items_pending_review",
	} {
		var count int
		if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`, name).Scan(&count); err != nil || count != 1 {
			t.Fatalf("work_items index %s was not preserved: count=%d err=%v", name, count, err)
		}
	}
	var indexCount, triggerCount int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='migration35_work_items_title'`).Scan(&indexCount); err != nil || indexCount != 1 {
		t.Fatalf("explicit index count=%d err=%v", indexCount, err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='trigger' AND name='migration35_work_items_title_audit'`).Scan(&triggerCount); err != nil || triggerCount != 1 {
		t.Fatalf("explicit trigger count=%d err=%v", triggerCount, err)
	}
	if _, err := store.db.ExecContext(ctx, "UPDATE work_items SET title = 'Updated after migration' WHERE id = ?", workID.String()); err != nil {
		t.Fatal(err)
	}
	var auditCount int
	if err := store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM migration35_work_item_audit WHERE id = ?", workID.String()).Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("preserved trigger audit count=%d err=%v", auditCount, err)
	}
	rows, err := store.db.QueryContext(ctx, "PRAGMA foreign_key_check")
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("migration left a foreign key violation")
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLiteBackupRestoreAtSchema34CanUpgradeWorkItemObjective(t *testing.T) {
	ctx := context.Background()
	store, _, _ := openWorkItemObjectiveSchema34(t)
	_, _, workID := createWorkItemObjectiveMigrationFixture(t, store)
	backup := filepath.Join(t.TempDir(), "schema34-backup.db")
	restored := filepath.Join(t.TempDir(), "schema34-restored.db")
	if err := store.Backup(ctx, backup); err != nil {
		t.Fatal(err)
	}
	if err := RestoreFile(backup, restored); err != nil {
		t.Fatal(err)
	}
	upgraded, err := Open(restored, Options{MigrateOnOpen: true})
	if err != nil {
		t.Fatalf("upgrade restored schema-34 database: %v", err)
	}
	t.Cleanup(func() { _ = upgraded.Close() })
	version, err := upgraded.SchemaVersion(ctx)
	if err != nil || version != 42 {
		t.Fatalf("restored schema version = %d, err=%v; want 42", version, err)
	}
	var title string
	if err := upgraded.db.QueryRowContext(ctx, "SELECT title FROM work_items WHERE id = ?", workID.String()).Scan(&title); err != nil || title != "Preserved Task" {
		t.Fatalf("restored Task title=%q err=%v", title, err)
	}
}
