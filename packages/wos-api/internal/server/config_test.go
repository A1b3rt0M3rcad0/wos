package server

import "testing"

func TestDefaultConfigIsValid(t *testing.T) {
	if err := DefaultConfig().Validate(); err != nil {
		t.Fatalf("DefaultConfig().Validate() error = %v", err)
	}
}

func TestConfigRejectsUnsupportedStorage(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Storage.Driver = "unknown"
	if err := cfg.Validate(); err == nil {
		t.Fatal("Config.Validate() unexpectedly accepted unsupported storage driver")
	}
}

func TestNewServerValidatesConfiguration(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Server.Listen = "invalid"
	if _, err := New(cfg); err == nil {
		t.Fatal("New() unexpectedly accepted invalid listen address")
	}
}
