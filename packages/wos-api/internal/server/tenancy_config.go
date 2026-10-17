package server

import (
	"crypto/sha256"
	"encoding/json"
	"io"
	"net"
	"os"
	"regexp"
	"strings"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

const maxHostedTenants = 32

// TenantHostConfig selects a provisioned Tenant by an explicit host mapping.
// Secret values are read from named environment variables and are never accepted
// from a request. Every child keeps an independent, fixed-schema Store.
type TenantHostConfig struct {
	Host              string `json:"host"`
	TenantID          string `json:"tenant_id"`
	BootstrapTokenEnv string `json:"bootstrap_token_env,omitempty"`
	SigningSeedEnv    string `json:"signing_seed_env,omitempty"`
	PrincipalID       string `json:"principal_id,omitempty"`
}

func parseTenantHosts(raw string) ([]TenantHostConfig, bool) {
	if len(raw) == 0 || len(raw) > 64<<10 {
		return nil, true
	}
	var hosts []TenantHostConfig
	decoder := json.NewDecoder(strings.NewReader(raw))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&hosts) != nil || len(hosts) == 0 {
		return nil, true
	}
	var trailing any
	if decoder.Decode(&trailing) != io.EOF {
		return nil, true
	}
	return hosts, false
}

var environmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var tenantHostname = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9.-]*[a-z0-9])?$`)
var tenantSchemaName = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

func canonicalTenantHost(raw string) (string, bool) {
	if raw == "" || strings.TrimSpace(raw) != raw {
		return "", false
	}
	host := raw
	if strings.Contains(raw, ":") {
		var err error
		host, _, err = net.SplitHostPort(raw)
		if err != nil {
			return "", false
		}
	}
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	if len(host) > 253 || !tenantHostname.MatchString(host) || strings.Contains(host, "..") {
		return "", false
	}
	for _, label := range strings.Split(host, ".") {
		if len(label) > 63 || strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return "", false
		}
	}
	return host, true
}

func (cfg Config) validateTenancy() error {
	invalid := func(message string) error { return domain.NewError(domain.ErrorCodeInvalidConfig, message) }
	if cfg.TenantHostsParseError || len(cfg.TenantHosts) > maxHostedTenants {
		return invalid("tenant hosts must be a strict JSON array containing 1..32 configured hosts")
	}
	if schema := cfg.Storage.PostgresCatalogSchema; schema != "" && (!tenantSchemaName.MatchString(schema) || schema == "public" || strings.HasPrefix(schema, "pg_") || strings.HasPrefix(schema, "wos_t_")) {
		return invalid("invalid PostgreSQL tenant catalog schema")
	}
	if cfg.Storage.PostgresSchema != "" && (!tenantSchemaName.MatchString(cfg.Storage.PostgresSchema) || cfg.Storage.TenantID == "") {
		return invalid("PostgreSQL schema assertion requires a valid bound Tenant")
	}
	if cfg.Storage.Driver == StorageDriverPostgres && cfg.Storage.TenantID == "" && len(cfg.TenantHosts) == 0 {
		return invalid("PostgreSQL runtime requires a provisioned Tenant ID")
	}
	if cfg.Storage.TenantID != "" {
		if (cfg.Storage.Driver != StorageDriverPostgres && cfg.Storage.Driver != StorageDriverSQLite) || len(cfg.TenantHosts) > 0 {
			return invalid("a bound Tenant requires supported storage and cannot also configure tenant hosts")
		}
		if _, err := domain.ParseID(cfg.Storage.TenantID); err != nil {
			return invalid("bound Tenant ID is invalid")
		}
		if cfg.Auth.Mode == AuthModeLocal && cfg.Auth.LocalTenantID != cfg.Storage.TenantID || cfg.Auth.Mode == AuthModeAPIToken && cfg.ServesAPI() && cfg.Auth.BootstrapTenantID != cfg.Storage.TenantID {
			return invalid("authentication Tenant must equal the bound storage Tenant")
		}
		for _, endpoint := range cfg.Integration.Endpoints {
			if endpoint.TenantID.String() != cfg.Storage.TenantID {
				return invalid("webhook endpoints must belong to the bound Tenant")
			}
		}
	}
	if len(cfg.TenantHosts) == 0 {
		return nil
	}
	if cfg.Storage.Driver != StorageDriverPostgres || cfg.Auth.Mode != AuthModeAPIToken || cfg.Storage.PostgresSchema != "" {
		return invalid("hosted Tenants require PostgreSQL, API credentials and catalog-resolved schemas")
	}
	hosts, tenants := map[string]bool{}, map[string]bool{}
	credentials := map[[32]byte]bool{}
	for _, entry := range cfg.TenantHosts {
		host, valid := canonicalTenantHost(entry.Host)
		if !valid || strings.Contains(entry.Host, ":") || hosts[host] || tenants[entry.TenantID] {
			return invalid("tenant host mappings require unique exact hostnames and Tenant IDs")
		}
		if _, err := domain.ParseID(entry.TenantID); err != nil {
			return invalid("hosted Tenant ID is invalid")
		}
		if entry.BootstrapTokenEnv != "" && !environmentName.MatchString(entry.BootstrapTokenEnv) || entry.SigningSeedEnv != "" && !environmentName.MatchString(entry.SigningSeedEnv) {
			return invalid("tenant secret references must name environment variables")
		}
		if cfg.ServesAPI() && (entry.BootstrapTokenEnv == "" || len(os.Getenv(entry.BootstrapTokenEnv)) < 32) {
			return invalid("each hosted API Tenant requires its own bootstrap credential reference")
		}
		if cfg.ServesAPI() {
			digest := sha256.Sum256([]byte(os.Getenv(entry.BootstrapTokenEnv)))
			if credentials[digest] {
				return invalid("hosted Tenants must not share a bootstrap credential")
			}
			credentials[digest] = true
		}
		hosts[host], tenants[entry.TenantID] = true, true
	}
	return nil
}

func (cfg Config) hostedTenantConfig(entry TenantHostConfig) Config {
	child := cfg
	child.TenantHosts = nil
	child.TenantHostsParseError = false
	child.Storage.TenantID = entry.TenantID
	child.Auth.BootstrapTenantID = entry.TenantID
	child.Auth.BootstrapTenantName = "Tenant"
	child.Auth.BootstrapToken = os.Getenv(entry.BootstrapTokenEnv)
	child.Signing.SeedEnv = entry.SigningSeedEnv
	if entry.PrincipalID != "" {
		child.Auth.LocalPrincipalID = entry.PrincipalID
	}
	child.Integration.Endpoints = nil
	for _, endpoint := range cfg.Integration.Endpoints {
		if endpoint.TenantID.String() == entry.TenantID {
			child.Integration.Endpoints = append(child.Integration.Endpoints, endpoint)
		}
	}
	return child
}

// A simple installation uses its bootstrap Tenant as the explicit
// storage binding. Workers without bootstrap configuration must name a Tenant.
func (cfg Config) withTenantBindingDefaults() Config {
	if len(cfg.TenantHosts) > 0 {
		return cfg
	}
	if cfg.Storage.TenantID == "" {
		if cfg.Auth.Mode == AuthModeLocal {
			cfg.Storage.TenantID = cfg.Auth.LocalTenantID
		} else {
			cfg.Storage.TenantID = cfg.Auth.BootstrapTenantID
		}
	}
	if cfg.Storage.TenantID != "" && cfg.Auth.Mode == AuthModeAPIToken {
		if cfg.Auth.BootstrapTenantID == "" {
			cfg.Auth.BootstrapTenantID = cfg.Storage.TenantID
		}
		if cfg.Auth.BootstrapTenantName == "" {
			cfg.Auth.BootstrapTenantName = "Tenant"
		}
	}
	return cfg
}
