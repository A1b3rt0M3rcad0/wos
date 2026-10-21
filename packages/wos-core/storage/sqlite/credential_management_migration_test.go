package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

func TestCredentialManagementMigrationCreatesAtomicMetadataSchema(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "credential-metadata.db"))
	version, err := store.SchemaVersion(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if version != 40 {
		t.Fatalf("schema version = %d, want 40", version)
	}

	rows, err := store.db.QueryContext(context.Background(), `PRAGMA table_info(credential_metadata)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	columns := map[string]bool{}
	for rows.Next() {
		var cid, notnull, primaryKey int
		var name, kind string
		var defaultValue any
		if err = rows.Scan(&cid, &name, &kind, &notnull, &defaultValue, &primaryKey); err != nil {
			t.Fatal(err)
		}
		columns[name] = true
	}
	if err = rows.Err(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"namespace_id", "credential_id", "name", "description", "created_at", "created_by", "created_by_actor_json", "role_template_id", "role_template_version", "policy_revision", "updated_at", "metadata_revision", "last_used_at", "replaced_by_credential_id", "replaces_credential_id", "rotation_overlap_until"} {
		if !columns[name] {
			t.Errorf("credential_metadata missing %q", name)
		}
	}

	foreignKeys, err := store.db.QueryContext(context.Background(), `PRAGMA foreign_key_list(credential_metadata)`)
	if err != nil {
		t.Fatal(err)
	}
	defer foreignKeys.Close()
	parents := map[string]bool{}
	for foreignKeys.Next() {
		var id, sequence int
		var table, from, to string
		var onUpdate, onDelete, match string
		if err = foreignKeys.Scan(&id, &sequence, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			t.Fatal(err)
		}
		parents[table] = true
	}
	if err = foreignKeys.Err(); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"credentials", "credential_policies", "principals"} {
		if !parents[table] {
			t.Errorf("credential_metadata missing foreign-key parent %q", table)
		}
	}
	var indexCount int
	if err = store.db.QueryRowContext(context.Background(), `SELECT count(*) FROM sqlite_master WHERE type='index' AND name='credential_metadata_namespace_last_used'`).Scan(&indexCount); err != nil || indexCount != 1 {
		t.Fatalf("last-used coalescing index count=%d err=%v", indexCount, err)
	}
}
