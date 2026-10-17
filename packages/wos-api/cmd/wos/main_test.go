package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/internal/server"
	wossqlite "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
)

func captureCommand(t *testing.T, args ...string) (int, []byte) {
	t.Helper()
	oldOut, oldErr := os.Stdout, os.Stderr
	out, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	os.Stdout, os.Stderr = writer, writer
	defer func() { os.Stdout, os.Stderr = oldOut, oldErr }()
	// Drain concurrently: Windows anonymous pipes are smaller than root help.
	type captured struct {
		data []byte
		err  error
	}
	result := make(chan captured, 1)
	go func() { data, err := io.ReadAll(out); result <- captured{data, err} }()
	code := run(args)
	writer.Close()
	value := <-result
	if value.err != nil {
		t.Fatal(value.err)
	}
	return code, value.data
}

func TestNativeStorageCommandsDoNotBootstrapServerOrClientConfiguration(t *testing.T) {
	root := t.TempDir()
	absent := filepath.Join(root, "unused-server", "wos.db")
	t.Setenv("WOS_SQLITE_PATH", absent)
	t.Setenv("WOS_AUTH_MODE", "invalid-unused-configuration")
	for _, args := range [][]string{{"storage", "--help"}, {"storage", "migrate-sqlite", "--help"}, {"help", "storage"}} {
		code, output := captureCommand(t, args...)
		if code != 0 || !strings.Contains(string(output), "migrate-sqlite") {
			t.Fatal("storage discovery failed", args, code, string(output))
		}
	}
	if runtime.GOOS != "linux" {
		for _, args := range [][]string{{"storage", "migrate-sqlite", "--backup", "absent"}, {"storage", "tenant-preflight", "--sqlite", "absent", "--tenant", "0199a221-0000-7000-8000-000000000001"}} {
			code, output := captureCommand(t, args...)
			if code == 0 || !strings.Contains(string(output), "supported only on Linux") {
				t.Fatal("unsupported host was not refused before storage access", args, code, string(output))
			}
		}
		if _, err := os.Stat(absent); !os.IsNotExist(err) {
			t.Fatal("unsupported host initialized server storage", err)
		}
		return
	}
	source := filepath.Join(root, "source.db")
	store, err := wossqlite.Open(source, wossqlite.Options{MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite3", source)
	if err != nil {
		t.Fatal(err)
	}
	tenant := "0199a221-0000-7000-8000-000000000001"
	_, err = db.Exec("INSERT INTO tenants(id,name,lifecycle,version,created_at,updated_at) VALUES(?,'native-preflight','active',1,1,1)", tenant)
	if err != nil {
		db.Close()
		t.Fatal(err)
	}
	if err = db.Close(); err != nil {
		t.Fatal(err)
	}
	code, output := captureCommand(t, "storage", "tenant-preflight", "--sqlite", source, "--tenant", tenant)
	if code != 0 {
		t.Fatal(code, string(output))
	}
	var inspection struct {
		Operation string `json:"operation"`
		Ready     bool   `json:"ready_for_transfer"`
		ReadOnly  bool   `json:"read_only"`
		Tenant    string `json:"tenant_id"`
	}
	if err = json.Unmarshal(output, &inspection); err != nil || inspection.Operation != "tenant_transfer_preflight" || inspection.Ready || !inspection.ReadOnly || inspection.Tenant != tenant {
		t.Fatal("native dispatch changed preflight semantics", string(output), err)
	}
	t.Setenv("WOS_MOBILITY_POSTGRES_DSN", "postgres://private-user:private-password@unused.invalid/unused")
	code, output = captureCommand(t, "storage", "migrate-sqlite", "--backup", "unused")
	if code == 0 || !strings.Contains(string(output), "writers-stopped") || strings.Contains(string(output), "private-password") {
		t.Fatal("native migration did not preserve protocol refusal/redaction", code, string(output))
	}
	if _, err = os.Stat(absent); !os.IsNotExist(err) {
		t.Fatal("offline storage operations initialized unrelated server storage", err)
	}
}

func TestUnifiedWOSClientUsesServiceReleaseIdentity(t *testing.T) {
	oldVersion, oldCommit, oldBuilt := server.Version, server.Commit, server.BuiltAt
	defer func() { server.Version, server.Commit, server.BuiltAt = oldVersion, oldCommit, oldBuilt }()
	server.Version, server.Commit, server.BuiltAt = "0.4.0", "0123456789abcdef0123456789abcdef01234567", "2026-10-12T12:00:00Z"
	code, output := captureCommand(t, "--output", "json", "version")
	if code != 0 {
		t.Fatalf("version failed: %s", output)
	}
	var value struct {
		Data struct {
			Version string `json:"version"`
			Commit  string `json:"commit"`
			BuiltAt string `json:"built_at"`
		}
	}
	if err := json.Unmarshal(output, &value); err != nil {
		t.Fatal(err)
	}
	if value.Data.Version != server.Version || value.Data.Commit != server.Commit || value.Data.BuiltAt != server.BuiltAt {
		t.Fatalf("unified version drift: %s", output)
	}
}

func TestUnifiedWOSCreatesAndReadsNativeProject(t *testing.T) {
	root := t.TempDir()
	code, output := captureCommand(t, "project", "init", "--workspace", root, "--server", "http://127.0.0.1:18099", "--server-id", "01a12288-0000-7000-8000-000000000001", "--tenant", "01a12288-0000-7000-8000-000000000002", "--outcome", "01a12288-0000-7000-8000-000000000003", "--output", "json")
	if code != 0 {
		t.Fatalf("project init failed: %s", output)
	}
	if _, err := os.Stat(root + "/.wos/project.yaml"); err != nil {
		t.Fatal(err)
	}
	code, output = captureCommand(t, "work", "checkout", "not-an-id", "--workspace", root, "--profile", "missing", "--output", "json")
	if code == 0 {
		t.Fatal("unconfigured profile gained authority")
	}
	if len(output) == 0 {
		t.Fatal("client error was not surfaced")
	}
}

func TestUnifiedWOSHelpIncludesBundledSkillsAndServiceWithoutConfiguration(t *testing.T) {
	t.Setenv("WOS_AUTH_MODE", "invalid-unused-configuration")
	for _, args := range [][]string{{"--help"}, {"skill", "--help"}, {"db", "--help"}, {"--profile", "missing", "work", "--help"}} {
		code, output := captureCommand(t, args...)
		if code != 0 || !strings.Contains(string(output), "Usage: wos") {
			t.Fatal("primary executable failed local help", args, code, string(output))
		}
		if len(args) == 1 && !strings.Contains(string(output), "wos skill install --agent codex") {
			t.Fatal("root help omitted bundled skill installation")
		}
	}
}

func TestUnifiedSharedConfigValidationDoesNotInitializeStorage(t *testing.T) {
	root := t.TempDir() + "/absent"
	t.Setenv("WOS_SQLITE_PATH", root+"/wos.db")
	t.Setenv("WOS_AUTH_MODE", "local")
	t.Setenv("WOS_LISTEN", "127.0.0.1:8080")
	code, output := captureCommand(t, "config", "validate", "--deployment=shared")
	if code != 0 {
		t.Fatal(code, string(output))
	}
	var report server.DeploymentReport
	if e := json.Unmarshal(output, &report); e != nil || !report.ReadOnly || !report.ConfigValid || report.DatabasePoolLimit != 1 {
		t.Fatal("invalid deployment diagnostic", string(output), e)
	}
	if _, e := os.Stat(root); !os.IsNotExist(e) {
		t.Fatal("read-only validation initialized storage", e)
	}
	code, output = captureCommand(t, "config", "validate")
	if code != 0 || strings.TrimSpace(string(output)) != "configuration valid" {
		t.Fatal("legacy validation changed", code, string(output))
	}
	if code, _ = captureCommand(t, "config", "validate", "--deployment=unknown"); code == 0 {
		t.Fatal("unknown deployment accepted")
	}
}

func TestServerRoleFlagRejectsInvalidOrUnsafeRoleBeforeOpeningStorage(t *testing.T) {
	t.Setenv("WOS_SQLITE_PATH", t.TempDir()+"/absent/wos.db")
	t.Setenv("WOS_PROCESS_ROLE", "complete")
	for _, args := range [][]string{{"server", "--role=unknown"}, {"server", "--role="}, {"server", "--role=worker"}, {"server", "extra"}, {"server", "--unexpected"}} {
		if code, output := captureCommand(t, args...); code == 0 || len(output) == 0 {
			t.Fatal("invalid server role arguments accepted", args, code, string(output))
		}
	}
	if _, err := os.Stat(os.Getenv("WOS_SQLITE_PATH")); !os.IsNotExist(err) {
		t.Fatal("invalid role opened storage", err)
	}
}

func TestNativeTenantCatalogCommandsUseOperatorDispatch(t *testing.T) {
	absent := filepath.Join(t.TempDir(), "unused-server", "wos.db")
	t.Setenv("WOS_SQLITE_PATH", absent)
	t.Setenv("WOS_AUTH_MODE", "invalid-unused-configuration")
	t.Setenv("WOS_POSTGRES_DSN", "postgres://operator:private-native-dispatch@127.0.0.1:1/absent")
	code, output := captureCommand(t, "storage", "tenant", "--help")
	if code != 0 || !strings.Contains(string(output), "import-shared") || !strings.Contains(string(output), "--from-env") {
		t.Fatal("Tenant operator help dispatched incorrectly", code, string(output))
	}
	code, output = captureCommand(t, "storage", "tenant", "show", "--tenant", "invalid")
	if code == 0 || !strings.Contains(string(output), "valid --tenant") || strings.Contains(string(output), "private-native-dispatch") {
		t.Fatal("Tenant validation or redaction failed", code, string(output))
	}
	if _, err := os.Stat(absent); !os.IsNotExist(err) {
		t.Fatal("Tenant command initialized unrelated server storage", err)
	}
}

func TestNativeSingleTenantOperationsRejectHostFleetBeforeOpeningStorage(t *testing.T) {
	absent := filepath.Join(t.TempDir(), "unused-server", "wos.db")
	t.Setenv("WOS_SQLITE_PATH", absent)
	t.Setenv("WOS_AUTH_MODE", "local")
	t.Setenv("WOS_TENANT_HOSTS", `[{"host":"first.example.test","tenant_id":"0199ac10-0000-7000-8000-000000000002"}]`)
	for _, args := range [][]string{{"db", "migrate"}, {"db", "backup", "unused"}, {"mcp", "stdio"}} {
		code, output := captureCommand(t, args...)
		if code != 2 || !strings.Contains(string(output), "Tenant configuration") {
			t.Fatal("fleet operation reached runtime initialization", args, code, string(output))
		}
	}
	if _, err := os.Stat(absent); !os.IsNotExist(err) {
		t.Fatal("unsupported fleet operation initialized storage", err)
	}
}
