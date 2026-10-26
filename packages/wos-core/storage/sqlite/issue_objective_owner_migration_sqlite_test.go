package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

func TestSQLiteIssueObjectiveOwnerMigrationPreservesIndexesTriggersAndReverseReferences(t *testing.T) {
	ctx := context.Background()
	store, required, scope, issue, _ := openIssueObjectiveSchema35(t)
	if _, err := store.db.ExecContext(ctx, `CREATE INDEX migration36_issues_title ON issues(title);
CREATE TABLE migration36_issue_audit(id TEXT NOT NULL, title TEXT NOT NULL);
CREATE TRIGGER migration36_issues_title_audit AFTER UPDATE OF title ON issues
BEGIN INSERT INTO migration36_issue_audit(id,title) VALUES(NEW.id,NEW.title); END;`); err != nil {
		t.Fatal(err)
	}
	if err := store.applyMigration(ctx, required); err != nil {
		t.Fatalf("apply Issue owner migration: %v", err)
	}
	for _, name := range []string{"idx_issues_scope", "idx_issues_scope_objective_lifecycle", "migration36_issues_title"} {
		var count int
		if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`, name).Scan(&count); err != nil || count != 1 {
			t.Fatalf("Issue index %s count=%d err=%v; want 1", name, count, err)
		}
	}
	var triggerCount int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='trigger' AND name='migration36_issues_title_audit'`).Scan(&triggerCount); err != nil || triggerCount != 1 {
		t.Fatalf("Issue trigger count=%d err=%v; want 1", triggerCount, err)
	}
	if _, err := store.db.ExecContext(ctx, `UPDATE issues SET title = ? WHERE id = ?`, "Updated after migration", issue.ID.String()); err != nil {
		t.Fatal(err)
	}
	var auditCount, affectedCount int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM migration36_issue_audit WHERE id = ?`, issue.ID.String()).Scan(&auditCount); err != nil || auditCount != 1 {
		t.Fatalf("preserved trigger audit count=%d err=%v; want 1", auditCount, err)
	}
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM issue_affected_refs WHERE namespace_id = ? AND outcome_id = ? AND issue_id = ?`,
		scope.NamespaceID.String(), scope.OutcomeID.String(), issue.ID.String()).Scan(&affectedCount); err != nil || affectedCount != 1 {
		t.Fatalf("reverse Issue references count=%d err=%v; want 1", affectedCount, err)
	}
	rows, err := store.db.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	if rows.Next() {
		t.Fatal("Issue owner migration left a foreign-key violation")
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLiteBackupRestoreAtSchema35CanUpgradeIssueObjectiveOwner(t *testing.T) {
	ctx := context.Background()
	store, required, scope, issue, _ := openIssueObjectiveSchema35(t)
	backup := filepath.Join(t.TempDir(), "schema35-backup.db")
	restored := filepath.Join(t.TempDir(), "schema35-restored.db")
	if err := store.Backup(ctx, backup); err != nil {
		t.Fatal(err)
	}
	if err := RestoreFile(backup, restored); err != nil {
		t.Fatal(err)
	}
	upgraded, err := Open(restored, Options{})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = upgraded.Close() })
	if err := upgraded.applyMigration(ctx, required); err != nil {
		t.Fatalf("upgrade restored schema-35 database: %v", err)
	}
	var title string
	if err := upgraded.db.QueryRowContext(ctx, `SELECT title FROM issues WHERE namespace_id = ? AND outcome_id = ? AND id = ?`,
		scope.NamespaceID.String(), scope.OutcomeID.String(), issue.ID.String()).Scan(&title); err != nil || title != issue.Title {
		t.Fatalf("restored Issue title=%q err=%v", title, err)
	}
}
