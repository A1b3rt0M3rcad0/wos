package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"testing"
	"time"
)

// Only the test harness invokes external programs. Core production code remains
// passive. The CI service uses the same client/server major version and owns this
// isolated test database; no production credentials enter process arguments.
func restoreIntegrationFixture(t *testing.T, source *Store) *Store {
	t.Helper()
	container := os.Getenv("WOS_TEST_POSTGRES_CONTAINER")
	dsn, err := url.Parse(os.Getenv("WOS_TEST_POSTGRES_DSN"))
	if err != nil || (dsn.Scheme != "postgres" && dsn.Scheme != "postgresql") || dsn.User == nil {
		t.Fatal("clean restore test requires PostgreSQL URL DSN")
	}
	user := dsn.User.Username()
	targetDatabase := fmt.Sprintf("wos_restore_%x", sha256.Sum256([]byte(t.TempDir())))[:44]
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()
	var sourceDatabase string
	if err := source.db.QueryRowContext(ctx, "SELECT current_database()").Scan(&sourceDatabase); err != nil {
		t.Fatal(err)
	}
	// Catalog scans in a long-lived test database may choose parallel workers
	// exceeding Docker's small /dev/shm. Bound only this test client session;
	// do not change production or server settings.
	dump := exec.CommandContext(ctx, "docker", "exec", "-e", "PGOPTIONS=-c max_parallel_workers_per_gather=0", container, "pg_dump", "-U", user, "-d", sourceDatabase, "--schema="+source.Path(), "--format=custom", "--no-owner", "--no-acl")
	var dumpErrors bytes.Buffer
	dump.Stderr = &dumpErrors
	backup, err := dump.Output()
	if err != nil {
		t.Fatalf("pg_dump: %v: %s", err, dumpErrors.String())
	}
	if _, err := source.db.ExecContext(ctx, "CREATE DATABASE "+targetDatabase); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		admin, err := Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), Options{})
		if err != nil {
			t.Errorf("restore cleanup: %v", err)
			return
		}
		defer admin.Close()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if _, err := admin.db.ExecContext(ctx, "DROP DATABASE "+targetDatabase+" WITH (FORCE)"); err != nil {
			t.Errorf("drop restored database: %v", err)
		}
	})
	restore := exec.CommandContext(ctx, "docker", "exec", "-i", container, "pg_restore", "-U", user, "-d", targetDatabase, "--single-transaction", "--no-owner", "--no-acl")
	restore.Stdin = bytes.NewReader(backup)
	if output, err := restore.CombinedOutput(); err != nil {
		t.Fatalf("pg_restore: %v: %s", err, output)
	}
	dsn.Path = "/" + targetDatabase
	schema := source.Path()
	if err := source.Close(); err != nil {
		t.Fatal(err)
	}
	restored, err := Open(dsn.String(), Options{Schema: schema, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { restored.Close() })
	t.Logf("restored %d backup bytes into an empty PostgreSQL database; continuing the shared integration contract", len(backup))
	return restored
}
