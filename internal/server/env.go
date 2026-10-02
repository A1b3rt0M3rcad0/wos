package server

import (
	"os"
	"strings"
)

func ConfigFromEnv() Config {
	cfg := DefaultConfig()
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
