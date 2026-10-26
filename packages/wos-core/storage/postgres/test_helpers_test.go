package postgres

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func requirePostgres(t *testing.T) {
	t.Helper()
	if os.Getenv("WOS_TEST_POSTGRES_DSN") == "" {
		t.Skip("real PostgreSQL required: set WOS_TEST_POSTGRES_DSN; CI always supplies it")
	}
}

func testID(s string) domain.ID { return domain.MustParseID(s) }

func openTestPostgres(tb testing.TB, path string, opts Options) (*Store, error) {
	tb.Helper()
	schema := fmt.Sprintf("wos_test_%x", sha256.Sum256([]byte(path)))[:42]
	opts.Schema = schema

	var store *Store
	// Every test store owns a deterministic schema. Drop it even when opening or
	// migrating fails, so PostgreSQL suites do not accumulate catalogs that slow
	// down pg_dump-based restore tests. Register before Open to cover partial opens.
	tb.Cleanup(func() {
		if err := store.Close(); err != nil {
			tb.Errorf("close PostgreSQL test store: %v", err)
		}
		admin, err := Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), Options{})
		if err != nil {
			tb.Errorf("open PostgreSQL cleanup connection: %v", err)
			return
		}
		defer admin.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if _, err := admin.db.ExecContext(ctx, `DROP SCHEMA IF EXISTS "`+schema+`" CASCADE`); err != nil {
			tb.Errorf("drop PostgreSQL test schema %s: %v", schema, err)
		}
	})

	store, err := Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), opts)
	return store, err
}

func openTestStore(t *testing.T, path string) *Store {
	t.Helper()
	requirePostgres(t)
	s, err := openTestPostgres(t, path, Options{MigrateOnOpen: true, BusyTimeout: 500 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	return s
}

// CI additionally restores the integration fixture into an empty database using
// pg_dump/pg_restore from the exact PostgreSQL server image.
func reopenIntegrationFixture(t *testing.T, store *Store, path string) *Store {
	t.Helper()
	if os.Getenv("WOS_TEST_POSTGRES_CONTAINER") != "" {
		return restoreIntegrationFixture(t, store)
	}
	if os.Getenv("CI") == "true" {
		t.Fatal("PostgreSQL CI must supply WOS_TEST_POSTGRES_CONTAINER for clean restore")
	}
	store.Close()
	return openTestStore(t, path)
}
