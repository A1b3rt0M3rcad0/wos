package server

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

func TestWorkerConfigurationValidatesActualSecretEligibilityBeforeStorage(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Integration.WorkerEnabled = true
	cfg.Storage.SQLitePath = t.TempDir() + "/absent/wos.db"
	cfg.Integration.Secrets = map[string]string{}
	for i := 0; i < 256; i++ {
		cfg.Integration.Secrets[fmt.Sprintf("reference-%03d", i)] = strings.Repeat("s", 32)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatal("supported worker eligibility rejected", err)
	}
	cfg.Integration.Secrets["overflow"] = strings.Repeat("s", 32)
	if runtime, err := OpenRuntime(cfg); err == nil {
		runtime.Close()
		t.Fatal("worker configuration failed only after the runtime started")
	}
	if _, err := os.Stat(cfg.Storage.SQLitePath); !os.IsNotExist(err) {
		t.Fatal("invalid worker policy initialized storage", err)
	}
	cfg.Server.Role = ProcessRoleAPI
	cfg.Integration.WorkerEnabled = false
	if err := cfg.Validate(); err != nil {
		t.Fatal("API-only process unnecessarily constrained unused delivery secrets", err)
	}
	cfg = DefaultConfig()
	cfg.Integration.WorkerEnabled = true
	cfg.Integration.Secrets = map[string]string{strings.Repeat("opaque-reference/", 80): strings.Repeat("s", 32)}
	if err := cfg.Validate(); err != nil {
		t.Fatal("existing opaque reference contract gained an arbitrary byte bound", err)
	}
}
