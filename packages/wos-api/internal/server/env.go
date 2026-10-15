package server

import (
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"os"
	"strconv"
	"strings"
	"time"
)

func ConfigFromEnv() Config {
	cfg := DefaultConfig()
	for name, target := range map[string]*time.Duration{"WOS_HTTP_READ_HEADER_TIMEOUT": &cfg.HTTP.ReadHeaderTimeout, "WOS_HTTP_READ_TIMEOUT": &cfg.HTTP.ReadTimeout, "WOS_HTTP_WRITE_TIMEOUT": &cfg.HTTP.WriteTimeout, "WOS_HTTP_IDLE_TIMEOUT": &cfg.HTTP.IdleTimeout, "WOS_HTTP_REQUEST_TIMEOUT": &cfg.HTTP.RequestTimeout, "WOS_SHUTDOWN_TIMEOUT": &cfg.Server.ShutdownTimeout} {
		if value, ok := os.LookupEnv(name); ok {
			parsed, e := time.ParseDuration(value)
			if e != nil || parsed <= 0 {
				cfg.HTTP.ParseError = true
			} else {
				*target = parsed
			}
		}
	}
	for name, target := range map[string]*int{"WOS_HTTP_MAX_HEADER_BYTES": &cfg.HTTP.MaxHeaderBytes, "WOS_HTTP_MAX_INFLIGHT_REQUESTS": &cfg.HTTP.MaxInflightRequests, "WOS_MAX_CONNECTIONS": &cfg.Server.MaxConnections} {
		if value, ok := os.LookupEnv(name); ok {
			parsed, e := strconv.Atoi(value)
			if e != nil || parsed <= 0 {
				cfg.HTTP.ParseError = true
			} else {
				*target = parsed
			}
		}
	}

	if v := os.Getenv("WOS_LOCAL_NAMESPACE_ID"); v != "" {
		cfg.Auth.LocalNamespaceID = v
	}
	if v := os.Getenv("WOS_LOCAL_NAMESPACE_NAME"); v != "" {
		cfg.Auth.LocalNamespaceName = v
	}
	cfg.Auth.IndependentReviewer = strings.EqualFold(os.Getenv("WOS_INDEPENDENT_REVIEWER"), "true")
	if v, ok := os.LookupEnv("WOS_DELIVERY_BACKLOG_MAX_AGE"); ok {
		parsed, e := time.ParseDuration(v)
		if e != nil {
			cfg.Integration.ParseError = true
		} else {
			cfg.Integration.MaxBacklogAge = parsed
		}
	}
	cfg.Integration.WorkerEnabled = strings.EqualFold(os.Getenv("WOS_DELIVERY_WORKER_ENABLED"), "true")
	cfg.Integration.AllowLoopback = strings.EqualFold(os.Getenv("WOS_WEBHOOK_ALLOW_LOOPBACK"), "true")
	if raw := os.Getenv("WOS_WEBHOOK_ENDPOINTS"); raw != "" {
		cfg.Integration.ParseError = cfg.Integration.ParseError || json.Unmarshal([]byte(raw), &cfg.Integration.Endpoints) != nil
	}
	if raw := os.Getenv("WOS_WEBHOOK_SECRETS"); raw != "" {
		cfg.Integration.ParseError = cfg.Integration.ParseError || json.Unmarshal([]byte(raw), &cfg.Integration.Secrets) != nil
	}
	if v := os.Getenv("WOS_STORAGE_DRIVER"); v != "" {
		cfg.Storage.Driver = v
	}
	cfg.Storage.PostgresDSN = os.Getenv("WOS_POSTGRES_DSN")
	if v := os.Getenv("WOS_MCP_ENABLED"); v != "" {
		cfg.MCP.Enabled = v == "1" || strings.EqualFold(v, "true")
	}
	if v := os.Getenv("WOS_MCP_PATH"); v != "" {
		cfg.MCP.Path = v
	}
	if v := os.Getenv("WOS_AUTH_MODE"); v != "" {
		cfg.Auth.Mode = v
	}
	cfg.Signing.SeedEnv = os.Getenv("WOS_SERVER_SIGNING_SEED_ENV")
	cfg.Signing.AcceptanceFloor = domain.AcceptanceMode(os.Getenv("WOS_SIGNED_ACCEPTANCE_FLOOR"))
	cfg.Auth.BootstrapToken = os.Getenv("WOS_BOOTSTRAP_TOKEN")
	cfg.Auth.BootstrapNamespaceID = os.Getenv("WOS_BOOTSTRAP_NAMESPACE_ID")
	cfg.Auth.BootstrapNamespaceName = os.Getenv("WOS_BOOTSTRAP_NAMESPACE_NAME")
	if value := strings.TrimSpace(os.Getenv("WOS_LISTEN")); value != "" {
		cfg.Server.Listen = value
	}
	if value := strings.TrimSpace(os.Getenv("WOS_SQLITE_PATH")); value != "" {
		cfg.Storage.SQLitePath = value
	}
	if value := strings.TrimSpace(os.Getenv("WOS_LOCAL_PRINCIPAL_ID")); value != "" {
		cfg.Auth.LocalPrincipalID = value
	}
	if value := strings.TrimSpace(os.Getenv("WOS_LOCAL_ADMIN_OVERRIDES")); value != "" {
		cfg.Auth.LocalAllowAdministrativeOverrides = value == "1" || strings.EqualFold(value, "true")
	}
	return cfg
}
