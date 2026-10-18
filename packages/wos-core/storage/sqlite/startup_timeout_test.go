package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"
)

func TestStartupDeadlineIsIndependentFromWriterContention(t *testing.T) {
	path := filepath.Join(t.TempDir(), "startup.db")
	// An explicit startup deadline still aborts setup rather than extending it.
	if store, err := Open(path, Options{StartupTimeout: time.Nanosecond, MigrateOnOpen: true}); err == nil {
		store.Close()
		t.Fatal("expired startup deadline opened database")
	} else if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("startup did not preserve cancellation: %v", err)
	}
	// The default startup budget accommodates a cold migration under the race
	// detector and remains independent from the one-millisecond writer limit.
	store, err := Open(path, Options{BusyTimeout: time.Millisecond, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	settings, err := store.ConnectionSettings(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if settings.BusyTimeout != time.Millisecond {
		t.Fatal("migration allowance widened writer contention timeout")
	}
}
