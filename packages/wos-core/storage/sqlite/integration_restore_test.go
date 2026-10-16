package sqlite

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/tools/operations/testsupport"
	"os"
	"path/filepath"
	"testing"
)

// SQLite additionally proves a clean-file restore rather than only restart.
func reopenIntegrationFixture(t *testing.T, store *Store, path string) *Store {
	t.Helper()
	backup := filepath.Join(t.TempDir(), "snapshot.db")
	restored := filepath.Join(t.TempDir(), "restored.db")
	if os.Getenv("WOS_TEST_OPERATIONS_TOOL") == "1" {
		testsupport.Restore(t, store.db, "sqlite", store.Path(), restored, "main", store.Close)
		return openTestStore(t, restored)
	}
	if err := store.Backup(context.Background(), backup); err != nil {
		t.Fatal(err)
	}
	if err := RestoreFile(backup, restored); err != nil {
		t.Fatal(err)
	}
	store.Close()
	return openTestStore(t, restored)
}
