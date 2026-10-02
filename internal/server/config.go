package server

import (
	"net"
	"strings"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
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
	Server  ServerConfig
	HTTP    HTTPConfig
	MCP     MCPConfig
	Storage StorageConfig
	Auth    AuthConfig
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
	Mode             string
	LocalPrincipalID string
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
			Mode:             AuthModeLocal,
			LocalPrincipalID: "local-user",
		},
	}
}

func (cfg Config) Validate() error {
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
		if strings.TrimSpace(cfg.Auth.LocalPrincipalID) == "" {
			return domain.NewError(domain.ErrorCodeInvalidConfig, "auth.local principal is required")
		}
	case AuthModeAPIToken:
		// Token material is loaded by an authentication adapter in a later wave.
	default:
		return domain.NewError(domain.ErrorCodeInvalidConfig, "unsupported auth mode")
	}

	return nil
}

func validAbsolutePath(value string) bool {
	return strings.HasPrefix(value, "/") && !strings.HasPrefix(value, "//")
}
