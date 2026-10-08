package server

import (
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
)

// Config is the initial standalone composition contract. Parsing configuration
// files is intentionally separate from validation so embedded hosts can build
// Config values directly.
type Config struct {
	Signing     SigningConfig
	Server      ServerConfig
	HTTP        HTTPConfig
	MCP         MCPConfig
	Storage     StorageConfig
	Auth        AuthConfig
	Integration IntegrationConfig
}

type SigningConfig struct {
	SeedEnv         string
	AcceptanceFloor domain.AcceptanceMode
}

type IntegrationConfig struct {
	WorkerEnabled bool
	AllowLoopback bool
	Endpoints     []ports.WebhookEndpoint
	Secrets       map[string]string
	ParseError    bool
}

type ServerConfig struct {
	Listen          string
	ShutdownTimeout time.Duration
}

type HTTPConfig struct {
	Enabled        bool
	Prefix         string
	RequestTimeout time.Duration
}

type MCPConfig struct {
	Enabled   bool
	Path      string
	Stateless bool
}

type StorageConfig struct {
	Driver         string
	SQLitePath     string
	PostgresDSN    string
	MigrateOnStart bool
}

type AuthConfig struct {
	LocalNamespaceID                  string
	LocalNamespaceName                string
	IndependentReviewer               bool
	BootstrapToken                    string
	BootstrapNamespaceID              string
	BootstrapNamespaceName            string
	Mode                              string
	LocalPrincipalID                  string
	LocalAllowAdministrativeOverrides bool
}

func DefaultConfig() Config {
	return Config{
		Server: ServerConfig{
			Listen:          "127.0.0.1:8080",
			ShutdownTimeout: 15 * time.Second,
		},
		HTTP: HTTPConfig{
			Enabled:        true,
			Prefix:         "/api/v1",
			RequestTimeout: 15 * time.Second,
		},
		MCP: MCPConfig{
			Enabled:   false,
			Path:      "/mcp",
			Stateless: true,
		},
		Storage: StorageConfig{
			Driver:         StorageDriverSQLite,
			SQLitePath:     "./data/wos.db",
			MigrateOnStart: true,
		},
		Auth: AuthConfig{
			LocalNamespaceID:                  "0199d000-0000-7000-8000-000000000001",
			LocalNamespaceName:                "Local",
			Mode:                              AuthModeLocal,
			LocalPrincipalID:                  "local-user",
			LocalAllowAdministrativeOverrides: false,
		},
	}
}

func (cfg Config) Validate() error {
	if cfg.Signing.AcceptanceFloor != "" && !cfg.Signing.AcceptanceFloor.Valid() {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "invalid signed acceptance floor")
	}
	if cfg.Signing.SeedEnv != "" && cfg.Auth.Mode != AuthModeAPIToken {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "signed issuer requires scoped API credentials")
	}
	if cfg.Integration.ParseError {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "invalid integration JSON configuration")
	}
	seen := map[domain.ID]bool{}
	for _, endpoint := range cfg.Integration.Endpoints {
		if err := endpoint.ID.Validate(); err != nil {
			return err
		}
		if err := endpoint.NamespaceID.Validate(); err != nil {
			return err
		}
		u, err := url.Parse(endpoint.URL)
		local := err == nil && loopbackHost(u.Host)
		if err != nil || u.Host == "" || u.User != nil || u.Fragment != "" || len(endpoint.URL) > 2048 || (u.Scheme != "https" && !(cfg.Integration.AllowLoopback && local && u.Scheme == "http")) || endpoint.SecretRef == "" || endpoint.KeyID == "" || seen[endpoint.ID] {
			return domain.NewError(domain.ErrorCodeInvalidConfig, "invalid or duplicate webhook endpoint")
		}
		seen[endpoint.ID] = true
		if cfg.Integration.WorkerEnabled && len(cfg.Integration.Secrets[endpoint.SecretRef]) < 32 {
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
		if _, err := domain.ParseID(cfg.Auth.LocalNamespaceID); err != nil {
			return domain.WrapError(domain.ErrorCodeInvalidConfig, "invalid local namespace", err)
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
		if len(cfg.Auth.BootstrapToken) < 32 || cfg.Auth.BootstrapNamespaceName == "" {
			return domain.NewError(domain.ErrorCodeInvalidConfig, "api_token auth requires a bootstrap token and namespace")
		}
		if _, err := domain.ParseID(cfg.Auth.BootstrapNamespaceID); err != nil {
			return domain.WrapError(domain.ErrorCodeInvalidConfig, "bootstrap namespace ID is invalid", err)
		}
	default:
		return domain.NewError(domain.ErrorCodeInvalidConfig, "unsupported auth mode")
	}

	return nil
}

func validAbsolutePath(value string) bool {
	return strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//")
}
