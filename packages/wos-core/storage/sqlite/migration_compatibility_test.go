package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestFutureDatabaseSchemaRefusesStartupAndMigrationWithoutRewritingHistory(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "future.db")
	store := openTestStore(t, path)
	latest, e := LatestSchemaVersion()
	if e != nil {
		t.Fatal(e)
	}
	if _, e = store.db.ExecContext(ctx, `INSERT INTO schema_migrations(version,name,checksum,applied_at) VALUES (?, 'future.sql', 'future-sentinel', 1)`, latest+1); e != nil {
		t.Fatal(e)
	}
	if e = store.Migrate(ctx); e == nil {
		t.Fatal("future schema accepted")
	} else if code, _ := d.ErrorCodeOf(e); code != d.ErrorCodeInvalidConfig {
		t.Fatal("future schema rejection lost its configuration error", e)
	}
	migrations, e := loadMigrations()
	if e != nil {
		t.Fatal(e)
	}
	if e = store.applyMigration(ctx, migrations[0]); e == nil {
		t.Fatal("individual migration bypassed compatibility gate")
	}
	var checksum string
	if e = store.db.QueryRowContext(ctx, `SELECT checksum FROM schema_migrations WHERE version=?`, latest+1).Scan(&checksum); e != nil || checksum != "future-sentinel" {
		t.Fatal("future history changed", e)
	}
	store.Close()
	reopened, e := Open(path, Options{MigrateOnOpen: true})
	if e == nil {
		reopened.Close()
		t.Fatal("incompatible startup accepted")
	}
	if code, _ := d.ErrorCodeOf(e); code != d.ErrorCodeInvalidConfig {
		t.Fatal("startup lost rejection", e)
	}
}
