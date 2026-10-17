package tenants

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
	_ "github.com/jackc/pgx/v5/stdlib"
)

func TestTenantCommandsValidateBeforeConnectingAndRedactDSN(t *testing.T) {
	t.Setenv("WOS_POSTGRES_DSN", "postgres://operator:never-print-this-password@127.0.0.1:1/absent?connect_timeout=1")
	tests := []struct {
		args     []string
		success  bool
		contains string
	}{
		{[]string{"--help"}, true, "retry-provision"},
		{[]string{"show", "--tenant", "invalid"}, false, "valid --tenant"},
		{[]string{"list", "--limit", "0"}, false, "limit"},
		{[]string{"suspend", "--if-absent"}, false, "only for provision"},
		{[]string{"provision", "--from-env", "--file", "absent.json"}, false, "exclusive"},
		{[]string{"import-shared", "--file", "absent.json"}, false, "writers-stopped"},
		{[]string{"list"}, false, "cannot open Tenant catalog"},
	}
	for _, tt := range tests {
		var out, errOut bytes.Buffer
		code := Run(context.Background(), tt.args, &out, &errOut)
		all := out.String() + errOut.String()
		if (code == 0) != tt.success || !strings.Contains(all, tt.contains) || strings.Contains(all, "never-print-this-password") {
			t.Fatal(tt.args, code, all)
		}
	}
	for _, raw := range []string{`{"tenant_id":"invalid","reason":"test"}`, `{"tenant_id":"0199ac10-0000-7000-8000-000000000002","reason":"first","reason":"second"}`, `{"tenant_id":"0199ac10-0000-7000-8000-000000000002","reason":"test","extra":true}`, `{"tenant_id":"0199ac10-0000-7000-8000-000000000002","reason":"test"} {}`} {
		if _, err := decodeIntent(strings.NewReader(raw)); err == nil {
			t.Fatal("accepted invalid intent", raw)
		}
	}
}

func TestPostgresTenantCommandLifecycleIsVersionedDrainedAndSchemaScoped(t *testing.T) {
	dsn := os.Getenv("WOS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("WOS_TEST_POSTGRES_DSN not configured")
	}
	t.Setenv("WOS_TENANT_CLI_TEST_DSN", dsn)
	suffix := time.Now().UnixNano()
	schema := fmt.Sprintf("wos_cli_%x", suffix)
	id := domain.MustParseID(fmt.Sprintf("0199ac10-1000-7000-8000-%012x", uint64(suffix)&0xffffffffffff))
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	t.Cleanup(func() {
		_, _ = db.Exec(`DROP SCHEMA IF EXISTS "` + schema + `" CASCADE`)
		_, _ = db.Exec(`DROP SCHEMA IF EXISTS "wos_t_` + strings.ReplaceAll(id.String(), "-", "") + `" CASCADE`)
	})
	t.Setenv("WOS_TENANT_ID", id.String())
	t.Setenv("WOS_BOOTSTRAP_TENANT_NAME", "CLI lifecycle")
	t.Setenv("WOS_POSTGRES_CATALOG_SCHEMA", schema)
	requestFile := filepath.Join(t.TempDir(), "intent.json")
	invoke := func(action string, intent *Intent, extra ...string) (int, []byte) {
		t.Helper()
		args := []string{action, "--dsn-env", "WOS_TENANT_CLI_TEST_DSN"}
		if intent != nil {
			raw, _ := json.Marshal(intent)
			if err := os.WriteFile(requestFile, raw, 0600); err != nil {
				t.Fatal(err)
			}
			args = append(args, "--file", requestFile)
		}
		args = append(args, extra...)
		var out, stderr bytes.Buffer
		code := Run(context.Background(), args, &out, &stderr)
		if code != 0 {
			return code, stderr.Bytes()
		}
		return code, out.Bytes()
	}
	placement := func(raw []byte) postgres.TenantPlacement {
		t.Helper()
		var response struct {
			Data postgres.TenantPlacement `json:"data"`
		}
		if err := json.Unmarshal(raw, &response); err != nil {
			t.Fatal(err, string(raw))
		}
		return response.Data
	}
	code, raw := invoke("provision", nil, "--from-env", "--if-absent")
	if code != 0 {
		t.Fatal(string(raw))
	}
	p := placement(raw)
	if p.State != postgres.TenantActive || p.TenantID != id || p.Version == 0 {
		t.Fatal(p)
	}
	code, raw = invoke("provision", nil, "--from-env", "--if-absent")
	if code != 0 || placement(raw).Version != p.Version {
		t.Fatal("ensure mutated existing placement", code, string(raw))
	}
	code, raw = invoke("provision", &Intent{TenantID: id, Name: "different name", Reason: "same ID different purpose"}, "--if-absent")
	if code == 0 {
		t.Fatal("ensure silently renamed Tenant")
	}
	stdinRaw, _ := json.Marshal(Intent{TenantID: id, Name: "CLI lifecycle", Reason: "stdin ensure"})
	t.Setenv("WOS_POSTGRES_CATALOG_SCHEMA", schema+"_wrong")
	var stdinOut, stdinErr bytes.Buffer
	code = RunWithInput(context.Background(), []string{"provision", "--dsn-env", "WOS_TENANT_CLI_TEST_DSN", "--catalog-schema", schema, "--file", "-", "--if-absent"}, bytes.NewReader(stdinRaw), &stdinOut, &stdinErr)
	if code != 0 || placement(stdinOut.Bytes()).Version != p.Version {
		t.Fatal("stdin provision failed", stdinErr.String())
	}
	t.Setenv("WOS_POSTGRES_CATALOG_SCHEMA", schema)
	catalog, err := postgres.OpenTenantCatalog(context.Background(), dsn, postgres.CatalogOptions{Schema: schema})
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	lease, err := catalog.AcquireTenantLease(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	code, raw = invoke("suspend", &Intent{TenantID: id, ExpectedVersion: p.Version, Reason: "drain"}, "--writers-stopped")
	if code == 0 {
		lease.Close()
		t.Fatal("suspended live runtime", string(raw))
	}
	lease.Close()
	code, raw = invoke("delete", &Intent{TenantID: id, ExpectedVersion: p.Version, Reason: "active delete"}, "--writers-stopped")
	if code == 0 {
		t.Fatal("deleted active Tenant", string(raw))
	}
	code, raw = invoke("migrate", &Intent{TenantID: id, ExpectedVersion: p.Version, Reason: "unsafe active upgrade"}, "--writers-stopped")
	if code == 0 {
		t.Fatal("migrated active Tenant")
	}
	code, raw = invoke("suspend", &Intent{TenantID: id, ExpectedVersion: p.Version, Reason: "planned pause"}, "--writers-stopped")
	if code != 0 {
		t.Fatal(string(raw))
	}
	p = placement(raw)
	if p.State != postgres.TenantSuspended {
		t.Fatal(p)
	}
	code, raw = invoke("provision", nil, "--from-env", "--if-absent")
	if code == 0 {
		t.Fatal("ensure reactivated suspended Tenant", string(raw))
	}
	beforeMigration := p.Version
	code, raw = invoke("migrate", &Intent{TenantID: id, ExpectedVersion: p.Version, Reason: "approved schema upgrade"}, "--writers-stopped")
	if code != 0 {
		t.Fatal(string(raw))
	}
	p = placement(raw)
	if p.State != postgres.TenantSuspended || p.Version != beforeMigration+1 {
		t.Fatal("migration changed suspension or lost version", p)
	}
	code, raw = invoke("resume", &Intent{TenantID: id, ExpectedVersion: p.Version - 1, Reason: "stale"}, "--writers-stopped")
	if code == 0 {
		t.Fatal("stale version accepted")
	}
	code, raw = invoke("resume", &Intent{TenantID: id, ExpectedVersion: p.Version, Reason: "resume"}, "--writers-stopped")
	if code != 0 {
		t.Fatal(string(raw))
	}
	p = placement(raw)
	code, raw = invoke("suspend", &Intent{TenantID: id, ExpectedVersion: p.Version, Reason: "retire"}, "--writers-stopped")
	if code != 0 {
		t.Fatal(string(raw))
	}
	p = placement(raw)
	code, raw = invoke("delete", &Intent{TenantID: id, ExpectedVersion: p.Version, Reason: "retention completed"}, "--writers-stopped")
	if code != 0 {
		t.Fatal(string(raw))
	}
	p = placement(raw)
	if p.State != postgres.TenantDeleted {
		t.Fatal(p)
	}
	var exists bool
	if err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=$1)`, p.Schema).Scan(&exists); err != nil || exists {
		t.Fatal("Tenant schema remains", err)
	}
	code, raw = invoke("show", nil, "--tenant", id.String())
	if code != 0 || placement(raw).State != postgres.TenantDeleted {
		t.Fatal("deletion tombstone lost", string(raw))
	}
	code, raw = invoke("provision", &Intent{TenantID: id, Name: "reuse", Reason: "test"})
	if code == 0 {
		t.Fatal("deleted Tenant identity was reused")
	}
}

func TestPostgresImportCommandChecksSourceBeforeMigrationAndPreservesTenantIdentity(t *testing.T) {
	dsn := os.Getenv("WOS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("WOS_TEST_POSTGRES_DSN not configured")
	}
	t.Setenv("WOS_TENANT_IMPORT_CLI_DSN", dsn)
	suffix := time.Now().UnixNano()
	control := fmt.Sprintf("wos_import_cli_%x", suffix)
	source := fmt.Sprintf("wos_source_cli_%x", suffix)
	absent := source + "_absent"
	id := domain.MustParseID(fmt.Sprintf("0199ac10-2000-7000-8000-%012x", uint64(suffix)&0xffffffffffff))
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	t.Cleanup(func() {
		for _, schema := range []string{control, source, absent, "wos_t_" + strings.ReplaceAll(id.String(), "-", "")} {
			_, _ = db.Exec(`DROP SCHEMA IF EXISTS "` + schema + `" CASCADE`)
		}
	})
	invoke := func(from string) (int, []byte) {
		t.Helper()
		raw, _ := json.Marshal(map[string]string{"source_schema": from, "reason": "offline CLI import test"})
		var out, stderr bytes.Buffer
		code := RunWithInput(context.Background(), []string{"import-shared", "--file", "-", "--writers-stopped", "--dsn-env", "WOS_TENANT_IMPORT_CLI_DSN", "--catalog-schema", control}, bytes.NewReader(raw), &out, &stderr)
		if code != 0 {
			return code, stderr.Bytes()
		}
		return code, out.Bytes()
	}
	code, raw := invoke(absent)
	if code == 0 || !strings.Contains(string(raw), "no migration was attempted") {
		t.Fatal(code, string(raw))
	}
	var exists bool
	if err = db.QueryRow(`SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=$1)`, absent).Scan(&exists); err != nil || exists {
		t.Fatal("invalid source was initialized", err)
	}
	store, err := postgres.Open(dsn, postgres.Options{Schema: source, MigrateOnOpen: true, MaxOpenConnections: 2})
	if err != nil {
		t.Fatal(err)
	}
	if err = store.PutTenant(context.Background(), ports.Tenant{ID: id, Name: "Preserved Tenant"}); err != nil {
		store.Close()
		t.Fatal(err)
	}
	store.Close()
	code, raw = invoke(source)
	if code != 0 {
		t.Fatal(string(raw))
	}
	var result struct {
		Data postgres.TenantPlacementImportResult `json:"data"`
	}
	if err = json.Unmarshal(raw, &result); err != nil || !result.Data.SourceWriteFenced || len(result.Data.Placements) != 1 || result.Data.Placements[0].TenantID != id || result.Data.Placements[0].State != postgres.TenantSuspended {
		t.Fatal(string(raw), err)
	}
	code, raw = invoke(source)
	if code == 0 || !strings.Contains(string(raw), "no migration was attempted") {
		t.Fatal("already fenced source was migrated again", code, string(raw))
	}
}
