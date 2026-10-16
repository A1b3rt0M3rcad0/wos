// Package testsupport drives the opt-in operator tools from real storage
// journeys. It is imported only by tests; Core never invokes these programs.
package testsupport

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// Restore uses real clients from PATH. The caller provisions a new destination
// and retains private issuer/agent keys separately, just as an operator must.
func Restore(t *testing.T, db *sql.DB, backend, source, destination, schema string, stopWriters func() error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	var raw string
	err := db.QueryRowContext(ctx, "SELECT state_json FROM server_protocol_identity WHERE singleton=1").Scan(&raw)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		t.Fatal(err)
	}
	var approved struct {
		ServerID    string `json:"server_id"`
		Fingerprint string `json:"fingerprint"`
	}
	if raw != "" {
		if err = json.Unmarshal([]byte(raw), &approved); err != nil {
			t.Fatal(err)
		}
	}
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err = os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		parent := filepath.Dir(root)
		if parent == root {
			t.Fatal("repository root unavailable")
		}
		root = parent
	}
	commit := os.Getenv("WOS_OPERATIONS_SOURCE_COMMIT")
	if commit == "" {
		cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
		cmd.Dir = root
		output, err := cmd.Output()
		if err != nil {
			t.Fatal(err)
		}
		commit = strings.TrimSpace(string(output))
	}
	backup := filepath.Join(t.TempDir(), "private-snapshot")
	run := func(arguments ...string) map[string]any {
		t.Helper()
		cmd := exec.CommandContext(ctx, "python3", append([]string{filepath.Join(root, "tools", "operations", "backup_restore.py")}, arguments...)...)
		cmd.Env = append(os.Environ(), "WOS_OPERATIONS_SOURCE_DSN="+source, "WOS_OPERATIONS_TARGET_DSN="+destination)
		output, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("operator procedure failed: %v: %s", err, output)
		}
		var result map[string]any
		if err = json.Unmarshal(output, &result); err != nil {
			t.Fatal(err)
		}
		return result
	}
	arguments := []string{"backup", "--backend", backend, "--output", backup, "--source-commit", commit, "--product-version", "test-fixture"}
	if backend == "sqlite" {
		arguments = append(arguments, "--sqlite", source)
	} else {
		arguments = append(arguments, "--dsn-env", "WOS_OPERATIONS_SOURCE_DSN", "--schema", schema)
	}
	created := run(arguments...)
	if err = stopWriters(); err != nil {
		t.Fatal(err)
	}
	arguments = []string{"restore", "--backup", backup, "--manifest-sha256", created["manifest_sha256"].(string), "--writers-stopped"}
	if backend == "sqlite" {
		arguments = append(arguments, "--sqlite", destination)
	} else {
		arguments = append(arguments, "--dsn-env", "WOS_OPERATIONS_TARGET_DSN")
	}
	if approved.ServerID != "" {
		arguments = append(arguments, "--expected-server-id", approved.ServerID, "--issuer-fingerprint", approved.Fingerprint)
	}
	restored := run(arguments...)
	if restored["verified"] != true || restored["validation_read_only"] != true {
		t.Fatalf("restore not verified: %v", restored)
	}
	t.Logf("operator tool restored %s into a new destination; approved public pin and every logical table digest verified; continuing actual authority journey", backend)
}
