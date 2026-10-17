package server

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestSharedDeploymentDiagnosticsAreSpecificReadOnlyAdvisories(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Storage.SQLitePath = t.TempDir() + "/never-created/wos.db"
	report, e := cfg.DeploymentDiagnostics("shared", DeploymentDeclarations{})
	if e != nil {
		t.Fatal(e)
	}
	codes := map[string]bool{}
	for _, a := range report.Advisories {
		codes[a.Code] = true
		if a.Action == "" {
			t.Fatal("unactionable advisory", a)
		}
	}
	if !report.ConfigValid || !report.ReadOnly || report.DatabasePoolLimit != 1 || !strings.Contains(report.WriterModel, "single SQLite writer") || !codes["sqlite_single_writer"] || !codes["ingress_tls"] || !codes["scoped_authentication"] || !codes["external_backups"] {
		t.Fatal("shared diagnostics missed actual limits", report)
	}
	// An unreachable DSN still validates locally: diagnostics must never connect.
	cfg.Storage.Driver = StorageDriverPostgres
	cfg.Storage.PostgresDSN = "postgres://private-user:never-print-me@invalid.test:1/wos"
	cfg.Storage.PostgresMaxOpenConnections = 12
	cfg.Auth.Mode = AuthModeAPIToken
	cfg.Auth.BootstrapToken = "also-never-print-me-private-bootstrap-secret"
	cfg.Auth.BootstrapTenantID = cfg.Auth.LocalTenantID
	cfg.Auth.BootstrapTenantName = "Instance"
	cfg.Signing.SeedEnv = "EXTERNALLY_PROVISIONED_SEED"
	cfg.Integration.WorkerEnabled = true
	cfg.Integration.WorkerConcurrency = 4
	report, e = cfg.DeploymentDiagnostics("shared", DeploymentDeclarations{TLSAtIngress: true, PersistentIssuer: true, InstanceCredentials: true, ManagedBackups: true})
	if e != nil {
		t.Fatal(e)
	}
	for _, a := range report.Advisories {
		if a.Severity == "warning" {
			t.Fatal("declared reference was falsely blocked", a)
		}
	}
	raw, _ := json.Marshal(report)
	if strings.Contains(string(raw), "never-print-me") || strings.Contains(string(raw), "private-user") || report.DatabasePoolLimit != 12 {
		t.Fatal("diagnostic leaked a credential or lost pool limit", string(raw))
	}
	if _, e = cfg.DeploymentDiagnostics("unrecognized", DeploymentDeclarations{}); e == nil {
		t.Fatal("unknown deployment silently accepted")
	}
}
func TestDeploymentPoolEnvironmentAndDeclarationsAreValidated(t *testing.T) {
	for _, value := range []string{"0", "-1", "257", "not-a-number"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv("WOS_POSTGRES_MAX_OPEN_CONNECTIONS", value)
			if ConfigFromEnv().Validate() == nil {
				t.Fatal("invalid pool setting ignored")
			}
		})
	}
	t.Setenv("WOS_POSTGRES_MAX_OPEN_CONNECTIONS", "8")
	if c := ConfigFromEnv(); c.Storage.PostgresMaxOpenConnections != 8 || c.Validate() != nil {
		t.Fatal("valid pool setting rejected")
	}
	t.Setenv("WOS_DEPLOYMENT_TLS_AT_INGRESS", "not-a-bool")
	if _, e := DeploymentDeclarationsFromEnv(); e == nil {
		t.Fatal("invalid declaration ignored")
	}
}

func TestDeclaredAggregateConnectionBudget(t *testing.T) {
	cfg := workerTestConfig(DefaultConfig())
	cfg.Storage.PostgresMaxOpenConnections = 8
	decl := DeploymentDeclarations{ExpectedProcesses: 4, DatabaseConnectionBudget: 50, ReservedConnections: 20}
	report, err := cfg.DeploymentDiagnostics("shared", decl)
	if err != nil || report.ExpectedConnectionDemand != 64 || report.ProcessRole != ProcessRoleWorker || !report.WorkerEnabled {
		t.Fatal("lost aggregate worker pool budget", report, err)
	}
	found := false
	for _, advisory := range report.Advisories {
		if advisory.Code == "database_connection_budget" && advisory.Severity == "warning" {
			found = true
		}
	}
	if !found {
		t.Fatal("oversubscribed database pools not reported")
	}
	decl.DatabaseConnectionBudget = 80
	report, err = cfg.DeploymentDiagnostics("shared", decl)
	if err != nil {
		t.Fatal(err)
	}
	for _, advisory := range report.Advisories {
		if advisory.Code == "database_connection_budget" {
			t.Fatal("valid declared budget rejected")
		}
	}
	decl.ExpectedProcesses = 0
	if _, err = cfg.DeploymentDiagnostics("shared", decl); err == nil {
		t.Fatal("incomplete budget declarations accepted")
	}
	t.Setenv("WOS_DEPLOYMENT_EXPECTED_PROCESSES", "not-a-count")
	if _, err = DeploymentDeclarationsFromEnv(); err == nil {
		t.Fatal("invalid process count ignored")
	}
}
