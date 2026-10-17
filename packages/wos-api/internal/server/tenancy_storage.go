package server

import (
	"context"
	"errors"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
)

type tenantStorageBinding struct {
	Store      *postgres.Store
	Catalog    *postgres.TenantCatalog
	Lease      *postgres.TenantLease
	OwnCatalog bool
}

func (binding *tenantStorageBinding) Close() error {
	if binding == nil {
		return nil
	}
	var errs []error
	if binding.Store != nil {
		errs = append(errs, binding.Store.Close())
	}
	if binding.Lease != nil {
		errs = append(errs, binding.Lease.Close())
	}
	if binding.OwnCatalog && binding.Catalog != nil {
		errs = append(errs, binding.Catalog.Close())
	}
	return errors.Join(errs...)
}

// Runtime and offline operators resolve the same placement and hold its lifecycle
// lease. Storage is always closed before releasing the lease or its catalog.
func openTenantStorage(ctx context.Context, config Config, migrate bool, shared *postgres.TenantCatalog) (binding *tenantStorageBinding, err error) {
	config = config.withTenantBindingDefaults()
	if len(config.TenantHosts) != 0 || config.Storage.TenantID == "" {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "this operation requires one explicit Tenant binding")
	}
	opened := &tenantStorageBinding{Catalog: shared}
	defer func() {
		if binding == nil {
			_ = opened.Close()
		}
	}()
	if opened.Catalog == nil {
		opened.Catalog, err = postgres.OpenTenantCatalog(ctx, config.Storage.PostgresDSN, postgres.CatalogOptions{
			Schema: config.Storage.PostgresCatalogSchema, MaxOpenConnections: 2, MaxLeases: 1, MigrateOnOpen: migrate,
		})
		if err != nil {
			return nil, err
		}
		opened.OwnCatalog = true
	}
	tenantID, err := domain.ParseID(config.Storage.TenantID)
	if err != nil {
		return nil, err
	}
	opened.Lease, err = opened.Catalog.AcquireTenantLease(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	placement := opened.Lease.Placement()
	if config.Storage.PostgresSchema != "" && config.Storage.PostgresSchema != placement.Schema {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "configured PostgreSQL schema does not match Tenant placement")
	}
	opened.Store, err = postgres.Open(config.Storage.PostgresDSN, postgres.Options{
		MigrateOnOpen: false, MaxOpenConnections: config.Storage.PostgresMaxOpenConnections,
		Schema: placement.Schema, TenantID: tenantID, TenantLease: opened.Lease,
	})
	if err != nil {
		return nil, err
	}
	return opened, nil
}
