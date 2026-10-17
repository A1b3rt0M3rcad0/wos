package server

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
)

func TestRuntimeRefusesFutureSchemaEvenWithMigrationsDisabled(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Storage.SQLitePath = filepath.Join(t.TempDir(), "future.db")
	store, e := sqlite.Open(cfg.Storage.SQLitePath, sqlite.Options{MigrateOnOpen: true})
	if e != nil {
		t.Fatal(e)
	}
	latest, e := store.SchemaVersion(context.Background())
	if e != nil {
		t.Fatal(e)
	}
	store.Close()
	db, e := sql.Open("sqlite3", cfg.Storage.SQLitePath)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = db.Exec(`INSERT INTO schema_migrations(version,name,checksum,applied_at) VALUES (?, 'future.sql', 'future-sentinel', 1)`, latest+1); e != nil {
		t.Fatal(e)
	}
	db.Close()
	for _, migrate := range []bool{true, false} {
		cfg.Storage.MigrateOnStart = migrate
		runtime, e := OpenRuntime(cfg)
		if e == nil {
			runtime.Close()
			t.Fatal("future schema allowed serving", migrate)
		}
		if code, _ := d.ErrorCodeOf(e); code != d.ErrorCodeInvalidConfig {
			t.Fatal("unexpected rejection", e)
		}
		inspect, err := sql.Open("sqlite3", cfg.Storage.SQLitePath)
		if err != nil {
			t.Fatal(err)
		}
		var tenants, guards int
		err = inspect.QueryRow(`SELECT count(*) FROM tenants`).Scan(&tenants)
		if err == nil {
			err = inspect.QueryRow(`SELECT count(*) FROM sqlite_schema WHERE name IN ('wos_tenant_identity_insert','wos_tenant_identity_update','wos_single_tenant')`).Scan(&guards)
		}
		inspect.Close()
		if err != nil || tenants != 0 || guards != 0 {
			t.Fatal("unsupported schema was modified before startup refusal", migrate, tenants, guards, err)
		}
	}
}
