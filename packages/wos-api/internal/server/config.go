package server

import (
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/integration"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

const (
	StorageDriverSQLite   = "sqlite"
	StorageDriverPostgres = "postgres"
	AuthModeLocal         = "local"
	AuthModeAPIToken      = "api_token"
	ProcessRoleComplete   = "complete"
	ProcessRoleAPI        = "api"
	ProcessRoleWorker     = "worker"
)

// Config is the initial standalone composition contract. Parsing configuration
// files is intentionally separate from validation so embedded hosts can build
// Config values directly.
type Config struct {
	TenantHosts           []TenantHostConfig
	TenantHostsParseError bool
	Signing               SigningConfig
	Server                ServerConfig
	HTTP                  HTTPConfig
	MCP                   MCPConfig
	Storage               StorageConfig
	Auth                  AuthConfig
	Integration           IntegrationConfig
}

type SigningConfig struct {
	SeedEnv         string
	AcceptanceFloor domain.AcceptanceMode
}

type IntegrationConfig struct {
	MaxBacklogAge          time.Duration
	WorkerConcurrency      int
	MaxInflightPerTenant   int
	MaxInflightPerEndpoint int
	WorkerEnabled          bool
	AllowLoopback          bool
	Endpoints              []ports.WebhookEndpoint
	Secrets                map[string]string
	ParseError             bool
}

type ServerConfig struct {
	Role            string
	Listen          string
	ShutdownTimeout time.Duration
	MaxConnections  int
}

type HTTPConfig struct {
	ParseError           bool
	ReadHeaderTimeout    time.Duration
	ReadTimeout          time.Duration
	WriteTimeout         time.Duration
	IdleTimeout          time.Duration
	MaxHeaderBytes       int
	MaxInflightRequests  int
	MaxInflightPerTenant int
	Enabled              bool
	Prefix               string
	RequestTimeout       time.Duration
}

type MCPConfig struct {
	Enabled   bool
	Path      string
	Stateless bool
}

type StorageConfig struct {
	TenantID                   string
	PostgresSchema             string
	PostgresCatalogSchema      string
	ParseError                 bool
	PostgresMaxOpenConnections int
	Driver                     string
	SQLitePath                 string
	PostgresDSN                string
	MigrateOnStart             bool
}

type AuthConfig struct {
	LocalTenantID                     string
	LocalTenantName                   string
	IndependentReviewer               bool
	BootstrapToken                    string
	BootstrapTenantID                 string
	BootstrapTenantName               string
	Mode                              string
	LocalPrincipalID                  string
	LocalAllowAdministrativeOverrides bool
}

func DefaultConfig() Config {
	return Config{
		Integration: IntegrationConfig{MaxBacklogAge: 5 * time.Minute, WorkerConcurrency: 1},
		Server: ServerConfig{
			Role:            ProcessRoleComplete,
			Listen:          "127.0.0.1:8080",
			ShutdownTimeout: 15 * time.Second,
			MaxConnections:  256,
		},
		HTTP: HTTPConfig{
			Enabled:             true,
			Prefix:              "/api/v1",
			RequestTimeout:      15 * time.Second,
			ReadHeaderTimeout:   5 * time.Second,
			ReadTimeout:         30 * time.Second,
			WriteTimeout:        45 * time.Second,
			IdleTimeout:         60 * time.Second,
			MaxHeaderBytes:      64 << 10,
			MaxInflightRequests: 128,
		},
		MCP: MCPConfig{
			Enabled:   false,
			Path:      "/mcp",
			Stateless: true,
		},
		Storage: StorageConfig{
			PostgresMaxOpenConnections: 16,
			Driver:                     StorageDriverSQLite,
			SQLitePath:                 "./data/wos.db",
			MigrateOnStart:             true,
		},
		Auth: AuthConfig{
			LocalTenantID:                     "0199d000-0000-7000-8000-000000000001",
			LocalTenantName:                   "Local",
			Mode:                              AuthModeLocal,
			LocalPrincipalID:                  "local-user",
			LocalAllowAdministrativeOverrides: false,
		},
	}
}

func (cfg Config) Validate() error {
	cfg = cfg.withTenantBindingDefaults()
	if err := cfg.validateTenancy(); err != nil {
		return err
	}
	switch cfg.ProcessRole() {
	case ProcessRoleComplete:
	case ProcessRoleAPI:
		if cfg.Integration.WorkerEnabled {
			return domain.NewError(domain.ErrorCodeInvalidConfig, "api process role cannot enable a delivery worker")
		}
	case ProcessRoleWorker:
		if cfg.Storage.Driver != StorageDriverPostgres || cfg.Auth.Mode != AuthModeAPIToken {
			return domain.NewError(domain.ErrorCodeInvalidConfig, "worker process role requires PostgreSQL and scoped API authentication")
		}
		if cfg.MCP.Enabled {
			return domain.NewError(domain.ErrorCodeInvalidConfig, "worker process role cannot expose MCP")
		}
	default:
		return domain.NewError(domain.ErrorCodeInvalidConfig, "process role must be complete, api or worker")
	}
	if cfg.ServesAPI() && !cfg.HTTP.Enabled {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "complete and api process roles require HTTP")
	}
	if cfg.Storage.ParseError || cfg.Storage.PostgresMaxOpenConnections < 0 || cfg.Storage.PostgresMaxOpenConnections > 256 {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "PostgreSQL pool limit must be 1..256 when configured")
	}
	if cfg.HTTP.ParseError {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "invalid HTTP/server environment value")
	}
	if cfg.HTTP.ReadHeaderTimeout <= 0 || cfg.HTTP.ReadTimeout < cfg.HTTP.ReadHeaderTimeout || cfg.HTTP.IdleTimeout <= 0 || cfg.HTTP.WriteTimeout <= cfg.HTTP.RequestTimeout || cfg.HTTP.WriteTimeout < cfg.HTTP.ReadTimeout {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "HTTP deadlines require positive header/idle timeouts, read >= header, and write > request and >= read")
	}
	if cfg.HTTP.MaxHeaderBytes < 1024 || cfg.HTTP.MaxHeaderBytes > 1<<20 || cfg.HTTP.MaxInflightRequests < 1 || cfg.HTTP.MaxInflightRequests > 4096 || cfg.Server.MaxConnections < cfg.HTTP.MaxInflightRequests+2 || cfg.Server.MaxConnections > 65536 {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "HTTP admission limits require headers 1 KiB..1 MiB, in-flight 1..4096 and connections in-flight+2..65536")
	}
	if cfg.HTTP.MaxInflightPerTenant < 0 || cfg.HTTP.MaxInflightPerTenant > cfg.HTTP.MaxInflightRequests {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "per-Tenant HTTP admission must be zero or at most the process in-flight limit")
	}

	if cfg.Signing.AcceptanceFloor != "" && !cfg.Signing.AcceptanceFloor.Valid() {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "invalid signed acceptance floor")
	}
	if cfg.Signing.SeedEnv != "" && cfg.Auth.Mode != AuthModeAPIToken {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "signed issuer requires scoped API credentials")
	}
	if cfg.Integration.WorkerConcurrency < 1 || cfg.Integration.WorkerConcurrency > 64 {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "delivery concurrency must be 1..64")
	}
	if cfg.Integration.MaxInflightPerTenant < 0 || cfg.Integration.MaxInflightPerTenant > 64 || cfg.Integration.MaxInflightPerEndpoint < 0 || cfg.Integration.MaxInflightPerEndpoint > 64 {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "delivery in-flight limits must be 0..64; zero disables the limit")
	}
	if cfg.Integration.MaxBacklogAge <= 0 || cfg.Integration.MaxBacklogAge > 7*24*time.Hour {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "delivery backlog age must be positive and at most seven days")
	}
	if cfg.Integration.ParseError {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "invalid integration JSON configuration")
	}
	if cfg.RunsDeliveryWorker() {
		if err := integration.ValidateWorkerPolicy(cfg.Integration.Secrets, ports.DeliveryClaimPolicy{MaxInflightPerTenant: cfg.Integration.MaxInflightPerTenant, MaxInflightPerEndpoint: cfg.Integration.MaxInflightPerEndpoint}); err != nil {
			return err
		}
	}
	seen := map[domain.ID]bool{}
	for _, endpoint := range cfg.Integration.Endpoints {
		if err := endpoint.ID.Validate(); err != nil {
			return err
		}
		if err := endpoint.TenantID.Validate(); err != nil {
			return err
		}
		u, err := url.Parse(endpoint.URL)
		local := err == nil && loopbackHost(u.Host)
		if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || len(endpoint.URL) > 2048 || (u.Scheme != "https" && !(cfg.Integration.AllowLoopback && local && u.Scheme == "http")) || endpoint.SecretRef == "" || endpoint.KeyID == "" || seen[endpoint.ID] {
			return domain.NewError(domain.ErrorCodeInvalidConfig, "invalid or duplicate webhook endpoint")
		}
		seen[endpoint.ID] = true
		if cfg.RunsDeliveryWorker() && len(cfg.Integration.Secrets[endpoint.SecretRef]) < 32 {
			return domain.NewError(domain.ErrorCodeInvalidConfig, "enabled webhook worker requires signing secrets of at least 32 bytes")
		}
	}
	if strings.TrimSpace(cfg.Server.Listen) == "" {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "server.listen is required")
	}
	if _, _, err := net.SplitHostPort(cfg.Server.Listen); err != nil {
		return domain.WrapError(domain.ErrorCodeInvalidConfig, "server.listen must be host:port", err)
	}
	if cfg.Server.ShutdownTimeout <= 0 {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "server.shutdown_timeout must be positive")
	}
	if cfg.HTTP.Enabled && !validAbsolutePath(cfg.HTTP.Prefix) {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "http.prefix must be an absolute path")
	}
	if cfg.HTTP.Enabled && cfg.HTTP.RequestTimeout <= 0 {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "http.request_timeout must be positive")
	}
	if cfg.MCP.Enabled && !validAbsolutePath(cfg.MCP.Path) {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "mcp.path must be an absolute path")
	}

	switch cfg.Storage.Driver {
	case StorageDriverSQLite:
		if strings.TrimSpace(cfg.Storage.SQLitePath) == "" {
			return domain.NewError(domain.ErrorCodeInvalidConfig, "storage.sqlite path is required")
		}
	case StorageDriverPostgres:
		if strings.TrimSpace(cfg.Storage.PostgresDSN) == "" {
			return domain.NewError(domain.ErrorCodeInvalidConfig, "storage.postgres DSN is required")
		}
	default:
		return domain.NewError(domain.ErrorCodeInvalidConfig, "unsupported storage driver")
	}

	switch cfg.Auth.Mode {
	case AuthModeLocal:
		if _, err := domain.ParseID(cfg.Auth.LocalTenantID); err != nil {
			return domain.WrapError(domain.ErrorCodeInvalidConfig, "invalid local tenant", err)
		}
		host, _, _ := net.SplitHostPort(cfg.Server.Listen)
		ip := net.ParseIP(host)
		if host != "localhost" && (ip == nil || !ip.IsLoopback()) {
			return domain.NewError(domain.ErrorCodeInvalidConfig, "local authentication requires a loopback listener")
		}
		if strings.TrimSpace(cfg.Auth.LocalPrincipalID) == "" {
			return domain.NewError(domain.ErrorCodeInvalidConfig, "auth.local principal is required")
		}
	case AuthModeAPIToken:
		if len(cfg.TenantHosts) > 0 {
			break
		}
		if cfg.ProcessRole() == ProcessRoleWorker {
			// A worker consumes an existing installation. It neither provisions a
			// Tenant nor needs possession of its bootstrap credential.
			break
		}
		if len(cfg.Auth.BootstrapToken) < 32 || cfg.Auth.BootstrapTenantName == "" {
			return domain.NewError(domain.ErrorCodeInvalidConfig, "api_token auth requires a bootstrap token and tenant")
		}
		if _, err := domain.ParseID(cfg.Auth.BootstrapTenantID); err != nil {
			return domain.WrapError(domain.ErrorCodeInvalidConfig, "bootstrap tenant ID is invalid", err)
		}
	default:
		return domain.NewError(domain.ErrorCodeInvalidConfig, "unsupported auth mode")
	}

	return nil
}

// ProcessRole treats an omitted embedded Config role as the historical complete
// process. Parsing an explicitly empty environment value is rejected separately.
func (cfg Config) ProcessRole() string {
	if cfg.Server.Role == "" {
		return ProcessRoleComplete
	}
	return cfg.Server.Role
}

func (cfg Config) ServesAPI() bool { return cfg.ProcessRole() != ProcessRoleWorker }

func (cfg Config) RunsDeliveryWorker() bool {
	return cfg.ProcessRole() == ProcessRoleWorker || cfg.ProcessRole() == ProcessRoleComplete && cfg.Integration.WorkerEnabled
}

func validAbsolutePath(value string) bool {
	return strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//")
}
