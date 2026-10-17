package server

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
)

// Each PostgreSQL runtime gets its own schema, including bootstrap credentials
// and receipts. The same configuration is reused on restart within the scenario.
func forEachRuntimeStorage(t *testing.T, scenario func(*testing.T, Config)) {
	t.Helper()
	for _, driver := range []string{StorageDriverSQLite, StorageDriverPostgres} {
		t.Run(driver, func(t *testing.T) {
			cfg := DefaultConfig()
			tenantID, idErr := (uuidV7Generator{}).NewID()
			if idErr != nil {
				t.Fatal(idErr)
			}
			cfg.Auth.LocalTenantID = tenantID.String()
			cfg.Storage.Driver = driver
			cfg.Storage.SQLitePath = filepath.Join(t.TempDir(), "runtime.db")
			if driver == StorageDriverPostgres {
				dsn := os.Getenv("WOS_TEST_POSTGRES_DSN")
				if dsn == "" {
					t.Skip("real PostgreSQL required: CI supplies WOS_TEST_POSTGRES_DSN")
				}
				schema := fmt.Sprintf("wos_runtime_%x", sha256.Sum256([]byte(cfg.Storage.SQLitePath)))[:44]
				db, err := sql.Open("pgx", dsn)
				if err != nil {
					t.Fatal(err)
				}
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				_, err = db.ExecContext(ctx, "CREATE SCHEMA "+schema)
				cancel()
				if err != nil {
					db.Close()
					t.Fatal(err)
				}
				cfg.Storage.PostgresCatalogSchema = schema
				t.Cleanup(func() {
					ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
					defer cancel()
					rows, queryErr := db.QueryContext(ctx, "SELECT schema_name FROM "+schema+".tenant_placements")
					if queryErr == nil {
						var schemas []string
						for rows.Next() {
							var name string
							if err := rows.Scan(&name); err != nil {
								t.Error(err)
								break
							}
							schemas = append(schemas, name)
						}
						rows.Close()
						for _, name := range schemas {
							if _, err := db.ExecContext(ctx, "DROP SCHEMA IF EXISTS "+name+" CASCADE"); err != nil {
								t.Errorf("clean Tenant schema: %v", err)
							}
						}
					}
					if _, err := db.ExecContext(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
						t.Errorf("clean runtime schema: %v", err)
					}
					db.Close()
				})
				if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
					u, err := url.Parse(dsn)
					if err != nil {
						t.Fatal(err)
					}
					q := u.Query()
					q.Set("search_path", schema)
					u.RawQuery = q.Encode()
					cfg.Storage.PostgresDSN = u.String()
				} else {
					cfg.Storage.PostgresDSN = dsn + " search_path=" + schema
				}
			}
			scenario(t, cfg)
		})
	}
}

// Runtime fixtures explicitly perform host provisioning before serving. This is
// intentionally outside OpenRuntime: a production worker/API cannot create a
// Tenant just by naming a schema or presenting a bootstrap credential.
func provisionRuntimeFixture(t *testing.T, cfg *Config) {
	t.Helper()
	if cfg.Storage.Driver != StorageDriverPostgres {
		return
	}
	*cfg = cfg.withTenantBindingDefaults()
	if cfg.Validate() != nil || !cfg.ServesAPI() || !cfg.Storage.MigrateOnStart {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	catalog, err := postgres.OpenTenantCatalog(ctx, cfg.Storage.PostgresDSN, postgres.CatalogOptions{Schema: cfg.Storage.PostgresCatalogSchema, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	defer catalog.Close()
	id := domain.MustParseID(cfg.Storage.TenantID)
	placement, err := catalog.GetTenant(ctx, id)
	if code, _ := domain.ErrorCodeOf(err); code == domain.ErrorCodeNotFound {
		name := cfg.Auth.BootstrapTenantName
		if cfg.Auth.Mode == AuthModeLocal {
			name = cfg.Auth.LocalTenantName
		}
		placement, err = catalog.ProvisionTenant(ctx, postgres.TenantProvision{TenantID: id, Name: name, Reason: "explicit runtime acceptance fixture provisioning"}, func(ctx context.Context, placement postgres.TenantPlacement) error {
			store, openErr := postgres.Open(cfg.Storage.PostgresDSN, placement.InitializerOptions())
			if openErr != nil {
				return openErr
			}
			return store.Close()
		})
	}
	if err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(cfg.Storage.PostgresDSN, "postgres://") || strings.HasPrefix(cfg.Storage.PostgresDSN, "postgresql://") {
		u, err := url.Parse(cfg.Storage.PostgresDSN)
		if err != nil {
			t.Fatal(err)
		}
		q := u.Query()
		q.Set("search_path", placement.Schema)
		u.RawQuery = q.Encode()
		cfg.Storage.PostgresDSN = u.String()
	} else {
		// libpq uses the last occurrence; preserve the supplied credentials verbatim.
		cfg.Storage.PostgresDSN += " search_path=" + placement.Schema
	}
}

func openRuntimeFixture(t *testing.T, cfg *Config) (*Runtime, error) {
	t.Helper()
	provisionRuntimeFixture(t, cfg)
	return OpenRuntime(*cfg)
}
