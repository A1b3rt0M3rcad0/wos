package server

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Each PostgreSQL runtime gets its own schema, including bootstrap credentials
// and receipts. The same configuration is reused on restart within the scenario.
func forEachRuntimeStorage(t *testing.T, scenario func(*testing.T, Config)) {
	t.Helper()
	for _, driver := range []string{StorageDriverSQLite, StorageDriverPostgres} {
		t.Run(driver, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.Storage.Driver = driver
			cfg.Storage.SQLitePath = filepath.Join(t.TempDir(), "runtime.db")
			if driver == StorageDriverPostgres {
				dsn := os.Getenv("WOS_TEST_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("real PostgreSQL required: CI supplies WOS_TEST_POSTGRES_DSN")
				}
				schema := fmt.Sprintf("wos_runtime_%x", sha256.Sum256([]byte(cfg.Storage.SQLitePath)))[:44]
				db, err := sql.Open("pgx", dsn)
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				_, err = db.ExecContext(ctx, "CREATE SCHEMA "+schema)
				cancel()
				if err != nil {
					db.Close()
					t.Fatal(err)
				}
				t.Cleanup(func() {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					if _, err := db.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
						t.Errorf("clean runtime schema: %v", err)
					}
					db.Close()
				})
				if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
					u, err := url.Parse(dsn)
					if err != nil {
						t.Fatal(err)
					}
					q := u.Query()
					q.Set("search_path", schema)
					u.RawQuery = q.Encode()
					cfg.Storage.PostgresDSN = u.String()
				} else {
					cfg.Storage.PostgresDSN = dsn + " search_path=" + schema
				}
			}
			scenario(t, cfg)
		})
	}
}
