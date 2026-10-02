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
	return cfg
}
