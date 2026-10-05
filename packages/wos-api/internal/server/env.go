package server

import (
	"encoding/json"
	"os"
	"strings"
)

func ConfigFromEnv() Config {
	cfg := DefaultConfig()
	if v := os.Getenv("WOS_LOCAL_NAMESPACE_ID"); v != "" {
		cfg.Auth.LocalNamespaceID = v
	}
	if v := os.Getenv("WOS_LOCAL_NAMESPACE_NAME"); v != "" {
		cfg.Auth.LocalNamespaceName = v
	}
	cfg.Auth.IndependentReviewer = strings.EqualFold(os.Getenv("WOS_INDEPENDENT_REVIEWER"), "true")
	cfg.Integration.WorkerEnabled = strings.EqualFold(os.Getenv("WOS_DELIVERY_WORKER_ENABLED"), "true")
	cfg.Integration.AllowLoopback = strings.EqualFold(os.Getenv("WOS_WEBHOOK_ALLOW_LOOPBACK"), "true")
	if raw := os.Getenv("WOS_WEBHOOK_ENDPOINTS"); raw != "" {
		cfg.Integration.ParseError = json.Unmarshal([]byte(raw), &cfg.Integration.Endpoints) != nil
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
