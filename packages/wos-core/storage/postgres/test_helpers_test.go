package postgres

import (
	"crypto/sha256"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
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

// PostgreSQL contract proves restart; clean pg_dump/restore acceptance is separate.
func reopenIntegrationFixture(t *testing.T, store *Store, path string) *Store {
	t.Helper()
	store.Close()
	return openTestStore(t, path)
}
