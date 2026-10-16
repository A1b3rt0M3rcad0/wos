package server

import (
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"os"
	"strconv"
)

// Declarations describe the operator's external controls. They are not a probe
// or certification of ingress, secret storage, backup schedules or replicas.
type DeploymentDeclarations struct {
	TLSAtIngress        bool `json:"tls_at_ingress"`
	PersistentIssuer    bool `json:"persistent_issuer"`
	InstanceCredentials bool `json:"instance_credentials"`
	ManagedBackups      bool `json:"managed_backups"`
}
type DeploymentAdvisory struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Action   string `json:"action"`
}
type DeploymentReport struct {
	ConfigValid       bool                   `json:"config_valid"`
	ReadOnly          bool                   `json:"read_only"`
	Deployment        string                 `json:"deployment"`
	StorageDriver     string                 `json:"storage_driver"`
	WriterModel       string                 `json:"writer_model"`
	DatabasePoolLimit int                    `json:"database_pool_limit"`
	WorkerEnabled     bool                   `json:"worker_enabled"`
	WorkerConcurrency int                    `json:"worker_concurrency"`
	Declarations      DeploymentDeclarations `json:"operator_declarations"`
	Advisories        []DeploymentAdvisory   `json:"advisories"`
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
	return v, nil
}
func (cfg Config) DeploymentDiagnostics(profile string, decl DeploymentDeclarations) (DeploymentReport, error) {
	r := DeploymentReport{ReadOnly: true, Deployment: profile, StorageDriver: cfg.Storage.Driver, DatabasePoolLimit: 1, WriterModel: "single SQLite writer per instance", WorkerEnabled: cfg.Integration.WorkerEnabled, WorkerConcurrency: cfg.Integration.WorkerConcurrency, Declarations: decl, Advisories: []DeploymentAdvisory{}}
	if profile != "local" && profile != "shared" {
		return r, d.NewError(d.ErrorCodeInvalidArgument, "deployment must be local or shared")
	}
	if e := cfg.Validate(); e != nil {
		return r, e
	}
	r.ConfigValid = true
	if cfg.Storage.Driver == StorageDriverPostgres {
		r.WriterModel = "PostgreSQL Serializable domain transactions with Outcome and Namespace guards"
		r.DatabasePoolLimit = cfg.Storage.PostgresMaxOpenConnections
		if r.DatabasePoolLimit == 0 {
			r.DatabasePoolLimit = 16
		}
	}
	add := func(code, severity, message, action string) {
		r.Advisories = append(r.Advisories, DeploymentAdvisory{code, severity, message, action})
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
	if cfg.Signing.SeedEnv == "" {
		add("issuer_unconfigured", "info", "No signed issuer seed source is configured.", "For signed work, provision one persistent seed per server identity, protect it separately and verify the approved public pin before activating the Namespace protocol.")
	} else if !decl.PersistentIssuer {
		add("issuer_persistence", "warning", "The issuer seed source is configured but persistence/recovery is not declared.", "Persist the same protected issuer across restarts/replicas and keep a separate tested recovery copy; never generate a replacement silently.")
	}
	if !decl.ManagedBackups {
		add("external_backups", "warning", "An external backup and restore-validation process is not declared.", "Install the opt-in tools/operations procedure outside Core, monitor backup age and validate a restore into a new destination.")
	}
	if !cfg.Integration.WorkerEnabled {
		add("delivery_worker_optional", "info", "The optional webhook worker is disabled in this process.", "If integration signals are emitted, run a supervised worker and monitor scoped delivery-health; API readiness alone does not prove delivery.")
	}
	if cfg.Integration.WorkerConcurrency >= r.DatabasePoolLimit && cfg.Storage.Driver == StorageDriverPostgres {
		add("pool_headroom", "warning", "Webhook concurrency can consume all configured database connections during claims and acknowledgments.", "Measure pool wait with API traffic and reserve connection headroom; default worker concurrency is 1 and the reference is 4.")
	}
	add("external_controls_declared", "info", "External controls are operator declarations; this read-only command does not connect, initialize storage, generate issuer identity or certify an SLA.", "Verify ingress, secret permissions, clocks, database capacity, endpoints and actual backup/restore evidence in the deployment environment.")
	return r, nil
}
