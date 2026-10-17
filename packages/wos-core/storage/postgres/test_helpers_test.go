package postgres

import (
	"context"
	"crypto/sha256"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/jackc/pgx/v5"
	"os"
	"testing"
	"time"
)

func requirePostgres(t *testing.T) {
	t.Helper()
	if os.Getenv("WOS_TEST_POSTGRES_DSN") == "" {
		t.Skip("real PostgreSQL required: set WOS_TEST_POSTGRES_DSN; CI always supplies it")
	}
}
func testID(s string) domain.ID { return domain.MustParseID(s) }
func openTestPostgres(path string, opts Options) (*Store, error) {
	opts.Schema = fmt.Sprintf("wos_test_%x", sha256.Sum256([]byte(path)))[:42]
	// Migration-disabled adapter startup never creates schemas. Tests that build
	// an old migration chain provision their empty fixture explicitly instead.
	if !opts.MigrateOnOpen {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		conn, err := pgx.Connect(ctx, os.Getenv("WOS_TEST_POSTGRES_DSN"))
		if err != nil {
			return nil, err
		}
		defer conn.Close(ctx)
		if _, err = conn.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS `+opts.Schema); err != nil {
			return nil, err
		}
	}
	return Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), opts)
}
func openTestStore(t *testing.T, path string) *Store {
	t.Helper()
	requirePostgres(t)
	s, err := openTestPostgres(path, Options{MigrateOnOpen: true, BusyTimeout: 500 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
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
	// Store.Path is already a PostgreSQL schema. Reusing the path-to-schema
	// fixture helper would hash it again and reopen an unrelated empty schema.
	schema := store.Path()
	store.Close()
	reopened, err := Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), Options{Schema: schema, MigrateOnOpen: true, BusyTimeout: 500 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { reopened.Close() })
	return reopened
}
