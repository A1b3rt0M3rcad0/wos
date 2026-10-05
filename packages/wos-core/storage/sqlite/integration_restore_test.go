package sqlite

import (
	"context"
	"path/filepath"
	"testing"
)

// SQLite additionally proves a clean-file restore rather than only restart.
func reopenIntegrationFixture(t *testing.T, store *Store, path string) *Store {
	t.Helper()
	backup := filepath.Join(t.TempDir(), "snapshot.db")
	restored := filepath.Join(t.TempDir(), "restored.db")
	if err := store.Backup(context.Background(), backup); err != nil {
		t.Fatal(err)
	}
	if err := RestoreFile(backup, restored); err != nil {
		t.Fatal(err)
	}
	store.Close()
	return openTestStore(t, restored)
}
