package server

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
)

// Business fixtures share only immutable bytes of a genuinely migrated empty
// database. Tenant binding, credentials and issuer identity are created by each
// runtime. Direct OpenRuntime startup/upgrade tests do not use this helper.
var emptyRuntimeSQLiteFixture = sync.OnceValues(func() ([]byte, error) {
	dir, err := os.MkdirTemp("", "wos-server-sqlite-fixture-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	store, err := sqlite.Open(filepath.Join(dir, "empty.db"), sqlite.Options{MigrateOnOpen: true})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), sqlite.DefaultStartupTimeout)
	defer cancel()
	snapshot := filepath.Join(dir, "snapshot.db")
	backupErr := store.Backup(ctx, snapshot)
	if err := errors.Join(backupErr, store.Close()); err != nil {
		return nil, err
	}
	return os.ReadFile(snapshot)
})

// Serialize publication so two fixture calls targeting the same new path
// cannot observe a partially written copy. Existing files are never replaced.
var runtimeSQLiteFixtureMu sync.Mutex

func seedRuntimeSQLiteFixture(path string) error {
	path = strings.TrimSpace(path)
	if path == ":memory:" {
		return nil
	}
	runtimeSQLiteFixtureMu.Lock()
	defer runtimeSQLiteFixtureMu.Unlock()
	if _, err := os.Lstat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	template, err := emptyRuntimeSQLiteFixture()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return nil
	}
	if err != nil {
		return err
	}
	n, writeErr := file.Write(template)
	if n != len(template) && writeErr == nil {
		writeErr = io.ErrShortWrite
	}
	if err := errors.Join(writeErr, file.Close()); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

func TestSQLiteRuntimeFixtureCopiesAreEmptyIsolatedAndNotReplaced(t *testing.T) {
	first, second := DefaultConfig(), DefaultConfig()
	first.Storage.SQLitePath = filepath.Join(t.TempDir(), "first.db")
	second.Storage.SQLitePath = filepath.Join(t.TempDir(), "second.db")
	provisionRuntimeFixture(t, &first)
	provisionRuntimeFixture(t, &second)
	for _, path := range []string{first.Storage.SQLitePath, second.Storage.SQLitePath} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
			t.Fatal("fixture database is not private", info.Mode())
		}
		db, err := sql.Open("sqlite3", path)
		if err != nil {
			t.Fatal(err)
		}
		defer db.Close()
		for _, table := range []string{"tenants", "credentials", "server_protocol_identity", "signing_keys"} {
			var count int
			if err := db.QueryRow("SELECT count(*) FROM " + table).Scan(&count); err != nil || count != 0 {
				t.Fatal("fixture copied Tenant or identity state", table, count, err)
			}
		}
		var bindings int
		if err := db.QueryRow(`SELECT count(*) FROM sqlite_schema WHERE name IN ('wos_tenant_identity_insert','wos_tenant_identity_update')`).Scan(&bindings); err != nil || bindings != 0 {
			t.Fatal("fixture copied a Tenant binding", bindings, err)
		}
		if err := db.Close(); err != nil {
			t.Fatal(err)
		}
	}
	firstRuntime, err := openRuntimeFixture(t, &first)
	if err != nil {
		t.Fatal(err)
	}
	if err := firstRuntime.Close(); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(first.Storage.SQLitePath)
	if err != nil {
		t.Fatal(err)
	}
	provisionRuntimeFixture(t, &first)
	after, err := os.ReadFile(first.Storage.SQLitePath)
	if err != nil || !bytes.Equal(before, after) {
		t.Fatal("fixture helper replaced existing runtime state", err)
	}
	secondDB, err := sql.Open("sqlite3", second.Storage.SQLitePath)
	if err != nil {
		t.Fatal(err)
	}
	defer secondDB.Close()
	var tenants int
	if err := secondDB.QueryRow(`SELECT count(*) FROM tenants`).Scan(&tenants); err != nil || tenants != 0 {
		t.Fatal("runtime initialization changed another fixture", tenants, err)
	}
}

func TestSQLiteRuntimeFixtureDoesNotInitializeInvalidWorkerOrReadOnlyStartup(t *testing.T) {
	for _, mode := range []string{"invalid", "worker", "migration_disabled"} {
		t.Run(mode, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Storage.SQLitePath = filepath.Join(t.TempDir(), "absent", "runtime.db")
			switch mode {
			case "invalid":
				cfg.HTTP.MaxInflightRequests = 0
			case "worker":
				cfg.Server.Role = ProcessRoleWorker
			case "migration_disabled":
				cfg.Storage.MigrateOnStart = false
			}
			if err := cfg.Validate(); mode == "migration_disabled" && err != nil {
				t.Fatal("fixture exclusion requires an otherwise valid config", err)
			}
			provisionRuntimeFixture(t, &cfg)
			if _, err := os.Lstat(cfg.Storage.SQLitePath); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("fixture helper initialized a non-serving or invalid configuration", err)
			}
		})
	}
}

func TestSQLiteRuntimeFixtureDoesNotCreateLiteralMemoryPath(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Storage.SQLitePath = ":memory:"
	if _, err := os.Lstat(cfg.Storage.SQLitePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("memory-path fixture requires no existing literal file", err)
	}
	provisionRuntimeFixture(t, &cfg)
	if _, err := os.Lstat(cfg.Storage.SQLitePath); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("fixture helper created a literal memory-path file", err)
	}
}
