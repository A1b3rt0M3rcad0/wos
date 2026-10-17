package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestSQLiteBusyWriterIsARecoverableTransactionConflict(t *testing.T) {
	path := filepath.Join(t.TempDir(), "writers.db")
	one := openTestStore(t, path)
	two, err := Open(path, Options{BusyTimeout: 25 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	defer two.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	held, err := one.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer held.Rollback()
	_, err = two.Begin(ctx)
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeTransactionConflict {
		t.Fatalf("writer contention was misclassified: %s %v", code, err)
	}
	held.Rollback()
	fresh, err := two.Begin(ctx)
	if err != nil {
		t.Fatal("released writer did not recover", err)
	}
	fresh.Rollback()
}
