package server

import (
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"os"
	"strconv"
)

// Declarations describe the operator's external controls. They are not a probe
// or certification of ingress, secret storage, backup schedules or replicas.
type DeploymentDeclarations struct {
	TLSAtIngress             bool `json:"tls_at_ingress"`
	PersistentIssuer         bool `json:"persistent_issuer"`
	InstanceCredentials      bool `json:"instance_credentials"`
	ManagedBackups           bool `json:"managed_backups"`
	ExpectedProcesses        int  `json:"expected_processes,omitempty"`
	DatabaseConnectionBudget int  `json:"database_connection_budget,omitempty"`
	ReservedConnections      int  `json:"reserved_connections,omitempty"`
}
type DeploymentAdvisory struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Action   string `json:"action"`
}
type DeploymentReport struct {
	HostedTenants            int                    `json:"hosted_tenants,omitempty"`
	CatalogPoolLimit         int                    `json:"catalog_pool_limit,omitempty"`
	LeaseConnections         int                    `json:"lease_connections,omitempty"`
	ProcessConnectionLimit   int                    `json:"process_connection_limit,omitempty"`
	ProcessRole              string                 `json:"process_role"`
	ConfigValid              bool                   `json:"config_valid"`
	ReadOnly                 bool                   `json:"read_only"`
	Deployment               string                 `json:"deployment"`
	StorageDriver            string                 `json:"storage_driver"`
	WriterModel              string                 `json:"writer_model"`
	DatabasePoolLimit        int                    `json:"database_pool_limit"`
	WorkerEnabled            bool                   `json:"worker_enabled"`
	WorkerConcurrency        int                    `json:"worker_concurrency"`
	ExpectedConnectionDemand int                    `json:"expected_connection_demand,omitempty"`
	Declarations             DeploymentDeclarations `json:"operator_declarations"`
	Advisories               []DeploymentAdvisory   `json:"advisories"`
}

func DeploymentDeclarationsFromEnv() (DeploymentDeclarations, error) {
	var v DeploymentDeclarations
	for name, target := range map[string]*bool{"WOS_DEPLOYMENT_TLS_AT_INGRESS": &v.TLSAtIngress, "WOS_DEPLOYMENT_PERSISTENT_ISSUER": &v.PersistentIssuer, "WOS_DEPLOYMENT_INSTANCE_CREDENTIALS": &v.InstanceCredentials, "WOS_DEPLOYMENT_MANAGED_BACKUPS": &v.ManagedBackups} {
		if raw, ok := os.LookupEnv(name); ok {
			parsed, e := strconv.ParseBool(raw)
			if e != nil {
				return v, d.NewError(d.ErrorCodeInvalidConfig, "invalid deployment declaration: "+name)
			}
			*target = parsed
		}
	}
	for name, target := range map[string]*int{"WOS_DEPLOYMENT_EXPECTED_PROCESSES": &v.ExpectedProcesses, "WOS_DEPLOYMENT_DATABASE_CONNECTION_BUDGET": &v.DatabaseConnectionBudget, "WOS_DEPLOYMENT_RESERVED_CONNECTIONS": &v.ReservedConnections} {
		if raw, ok := os.LookupEnv(name); ok {
			parsed, err := strconv.Atoi(raw)
			if err != nil || parsed < 0 || parsed > 100000 || parsed == 0 && name != "WOS_DEPLOYMENT_RESERVED_CONNECTIONS" {
				return v, d.NewError(d.ErrorCodeInvalidConfig, "invalid deployment declaration: "+name)
			}
			*target = parsed
		}
	}
	return v, nil
}
func (cfg Config) DeploymentDiagnostics(profile string, decl DeploymentDeclarations) (DeploymentReport, error) {
	r := DeploymentReport{ProcessRole: cfg.ProcessRole(), ReadOnly: true, Deployment: profile, StorageDriver: cfg.Storage.Driver, DatabasePoolLimit: 1, WriterModel: "single SQLite writer per instance", WorkerEnabled: cfg.RunsDeliveryWorker(), WorkerConcurrency: cfg.Integration.WorkerConcurrency, Declarations: decl, Advisories: []DeploymentAdvisory{}}
	if profile != "local" && profile != "shared" {
		return r, d.NewError(d.ErrorCodeInvalidArgument, "deployment must be local or shared")
	}
	if e := cfg.Validate(); e != nil {
		return r, e
	}
	if decl.ExpectedProcesses < 0 || decl.ExpectedProcesses > 100000 || decl.DatabaseConnectionBudget < 0 || decl.DatabaseConnectionBudget > 100000 || decl.ReservedConnections < 0 || decl.ReservedConnections > 100000 || (decl.ExpectedProcesses == 0) != (decl.DatabaseConnectionBudget == 0) || decl.DatabaseConnectionBudget == 0 && decl.ReservedConnections != 0 {
		return r, d.NewError(d.ErrorCodeInvalidConfig, "connection budget declarations require a process count and database budget together")
	}
	r.ConfigValid = true
	if cfg.Storage.Driver == StorageDriverPostgres {
		r.WriterModel = "PostgreSQL Serializable domain transactions with Outcome and Tenant guards"
		r.DatabasePoolLimit = cfg.Storage.PostgresMaxOpenConnections
		if r.DatabasePoolLimit == 0 {
			r.DatabasePoolLimit = 16
		}
		r.HostedTenants = len(cfg.TenantHosts)
		r.CatalogPoolLimit = 4
		if r.HostedTenants == 0 {
			r.HostedTenants = 1
			r.CatalogPoolLimit = 2
		}
		r.LeaseConnections = r.HostedTenants
		r.ProcessConnectionLimit = r.HostedTenants*r.DatabasePoolLimit + r.LeaseConnections + r.CatalogPoolLimit
	}
	add := func(code, severity, message, action string) {
		r.Advisories = append(r.Advisories, DeploymentAdvisory{code, severity, message, action})
	}
	if cfg.Storage.Driver == StorageDriverPostgres && decl.ExpectedProcesses > 0 {
		// This is an operator-declared upper bound, not an observation of a
		// fleet. Mixed pools should use their highest pool for this estimate.
		r.ExpectedConnectionDemand = decl.ExpectedProcesses*r.ProcessConnectionLimit + decl.ReservedConnections
		if r.ExpectedConnectionDemand > decl.DatabaseConnectionBudget {
			add("database_connection_budget", "warning", "Declared Tenant data pools, lifecycle leases, catalog pools and reserved connections exceed the database connection budget.", "Lower pools or replica counts, or increase the measured database budget. Count APIs, workers, maintenance and other clients together; use the largest pool for a conservative mixed-process estimate.")
		}
	}
	if profile == "local" {
		return r, nil
	}
	if cfg.Storage.Driver == StorageDriverSQLite {
		add("sqlite_single_writer", "warning", "SQLite serializes writes; it does not provide multiple concurrent writers or a shared-database replica topology.", "Use PostgreSQL for multiple replicas or measured writer contention; keep SQLite for a small single instance.")
	}
	if cfg.Auth.Mode != AuthModeAPIToken {
		add("scoped_authentication", "warning", "Shared deployment needs scoped instance credentials.", "Set WOS_AUTH_MODE=api_token and enroll/revoke each agent credential explicitly.")
	}
	if !decl.InstanceCredentials || cfg.Auth.BootstrapToken == "wos-local-development-bootstrap-token-change-for-shared-instances" {
		add("instance_credentials", "warning", "Dedicated instance credentials are not declared, or the public local example token is configured.", "Provision a unique bootstrap secret externally, enroll scoped credentials and declare WOS_DEPLOYMENT_INSTANCE_CREDENTIALS=true only after verification.")
	}
	if !decl.TLSAtIngress {
		add("ingress_tls", "warning", "TLS termination at the ingress is not declared.", "Terminate TLS at an ingress, verify the public certificate and block direct external access to the HTTP service.")
	}
	issuerConfigured := cfg.Signing.SeedEnv != ""
	if len(cfg.TenantHosts) > 0 {
		issuerConfigured = true
		for _, entry := range cfg.TenantHosts {
			if entry.SigningSeedEnv == "" {
				issuerConfigured = false
			}
		}
	}
	if !issuerConfigured {
		add("issuer_unconfigured", "info", "At least one Tenant runtime has no signed issuer seed source configured.", "For signed work, provision one persistent seed per server identity, protect it separately and verify the approved public pin before activating the Tenant protocol.")
	} else if !decl.PersistentIssuer {
		add("issuer_persistence", "warning", "The issuer seed source is configured but persistence/recovery is not declared.", "Persist the same protected issuer across restarts/replicas and keep a separate tested recovery copy; never generate a replacement silently.")
	}
	if !decl.ManagedBackups {
		add("external_backups", "warning", "An external backup and restore-validation process is not declared.", "Install the opt-in tools/operations procedure outside Core, monitor backup age and validate a restore into a new destination.")
	}
	if !cfg.RunsDeliveryWorker() {
		add("delivery_worker_optional", "info", "The optional webhook worker is disabled in this process.", "If integration signals are emitted, run a supervised worker and monitor scoped delivery-health; API readiness alone does not prove delivery.")
	}
	if cfg.ProcessRole() == ProcessRoleWorker {
		add("worker_operations_listener", "info", "The worker exposes process probes and authorized metrics only; readiness refers to this worker pool and database schema.", "Keep the listener private. Configure the same endpoint allowlist and secret references as the installation; monitor durable queue health through a WOS API process.")
	}
	if cfg.Integration.WorkerConcurrency >= r.DatabasePoolLimit && cfg.Storage.Driver == StorageDriverPostgres {
		add("pool_headroom", "warning", "Webhook concurrency can consume all configured database connections during claims and acknowledgments.", "Measure pool wait with API traffic and reserve connection headroom; default worker concurrency is 1 and the reference is 4.")
	}
	add("external_controls_declared", "info", "External controls are operator declarations; this read-only command does not connect, initialize storage, generate issuer identity or certify an SLA.", "Verify ingress, secret permissions, clocks, database capacity, endpoints and actual backup/restore evidence in the deployment environment.")
	return r, nil
}
