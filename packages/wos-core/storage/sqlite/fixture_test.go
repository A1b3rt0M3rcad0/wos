package sqlite

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

// Business contracts need independent empty databases, not a new execution of
// every historical migration for each scenario. Build this immutable template
// through the real migration and backup paths once per test process. Migration,
// startup and binding tests continue to call Open directly on their own files.
var migratedTestTemplate = sync.OnceValues(func() ([]byte, error) {
	directory, err := os.MkdirTemp("", "wos-sqlite-test-template-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(directory)
	store, err := Open(filepath.Join(directory, "source.db"), Options{MigrateOnOpen: true})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), DefaultStartupTimeout)
	defer cancel()
	snapshot := filepath.Join(directory, "snapshot.db")
	backupErr := store.Backup(ctx, snapshot)
	closeErr := store.Close()
	if backupErr != nil {
		return nil, backupErr
	}
	if closeErr != nil {
		return nil, closeErr
	}
	return os.ReadFile(snapshot)
})

// Serialize creation so two fixture callers cannot open the same file while its
// initial snapshot is being copied. No database connections or state are shared.
var testFixtureCreation sync.Mutex

func prepareCurrentTestStore(t *testing.T, path string) {
	t.Helper()
	if err := seedMigratedTestStore(path); err != nil {
		t.Fatal(err)
	}
}

func seedMigratedTestStore(path string) error {
	if strings.TrimSpace(path) == ":memory:" {
		return nil
	}
	testFixtureCreation.Lock()
	defer testFixtureCreation.Unlock()
	// Check before loading the template: historical fixtures, restored backups
	// and live replica paths must pass unchanged to production Open.
	if _, err := os.Lstat(path); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	snapshot, err := migratedTestTemplate()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return nil
	}
	if err != nil {
		return err
	}
	complete := false
	defer func() {
		_ = file.Close()
		if !complete {
			_ = os.Remove(path)
		}
	}()
	written, err := file.Write(snapshot)
	if err != nil {
		return err
	}
	if written != len(snapshot) {
		return io.ErrShortWrite
	}
	if err := file.Close(); err != nil {
		return err
	}
	complete = true
	return nil
}

func TestSQLiteFixtureCopiesAreIsolatedAndReopensPreserveState(t *testing.T) {
	firstPath := filepath.Join(t.TempDir(), "first.db")
	first := openTestStore(t, firstPath)
	if _, err := first.db.Exec(`INSERT INTO tenants(id,name,lifecycle,version,created_at,updated_at) VALUES('0199ff31-0000-7000-8000-000000000001','Only the first fixture','active',1,1,1)`); err != nil {
		t.Fatal(err)
	}
	secondPath := filepath.Join(t.TempDir(), "nested", "second.db")
	second := openTestStore(t, secondPath)
	var count int
	if err := second.db.QueryRow(`SELECT count(*) FROM tenants`).Scan(&count); err != nil || count != 0 {
		t.Fatal("independent fixture inherited another database's state", count, err)
	}
	if err := first.Close(); err != nil {
		t.Fatal(err)
	}
	reopened := openTestStore(t, firstPath)
	if err := reopened.db.QueryRow(`SELECT count(*) FROM tenants`).Scan(&count); err != nil || count != 1 {
		t.Fatal("fixture reopen replaced persisted state", count, err)
	}
	info, err := os.Stat(secondPath)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
		t.Fatalf("fixture permissions = %v, want 0600", info.Mode().Perm())
	}
}

func TestSQLiteFixturePreservesExistingBytes(t *testing.T) {
	path := filepath.Join(t.TempDir(), "existing.db")
	original := []byte("Existing fixtures must reach Open unchanged, even if they are invalid databases.")
	if err := os.WriteFile(path, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := seedMigratedTestStore(path); err != nil {
		t.Fatal(err)
	}
	actual, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(actual, original) {
		t.Fatal("fixture preparation changed an existing file", err)
	}
}

func TestSQLiteFixtureDoesNotMaterializeMemoryName(t *testing.T) {
	t.Chdir(t.TempDir())
	if err := seedMigratedTestStore(":memory:"); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(".")
	if err != nil || len(entries) != 0 {
		t.Fatal("fixture preparation created a file for the memory name", entries, err)
	}
}
