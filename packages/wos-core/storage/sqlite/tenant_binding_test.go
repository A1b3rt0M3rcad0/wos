package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func TestSQLiteTenantBindingPersistsAcrossEmbeddedReopen(t *testing.T) {
	ctx := context.Background()
	id := domain.MustParseID("0199f490-0000-7000-8000-000000000001")
	other := domain.MustParseID("0199f490-0000-7000-8000-000000000002")
	path := filepath.Join(t.TempDir(), "tenant.db")
	store, err := Open(path, Options{StartupTimeout: 2 * time.Minute, MigrateOnOpen: true, TenantID: id, TenantName: "Bound account"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = store.Close() }()
	if store.BoundTenantID() != id {
		t.Fatal("bound identity was not exposed")
	}
	if err = store.PutTenant(ctx, ports.Tenant{ID: other, Name: "Another account"}); err == nil {
		t.Fatal("bound SQLite accepted another Tenant")
	}
	if _, err = store.db.ExecContext(ctx, `UPDATE tenants SET id=?`, other.String()); err == nil {
		t.Fatal("bound SQLite allowed Tenant identity replacement")
	}
	if _, err = store.db.ExecContext(ctx, `INSERT INTO outcome_coordination(tenant_id,outcome_id,state_revision) VALUES(?,?,0)`, other.String(), other.String()); err == nil {
		t.Fatal("bound SQLite allowed a different Tenant's state")
	}
	if _, err = store.db.ExecContext(ctx, `DELETE FROM tenants`); err != nil {
		t.Fatal("could not remove empty Tenant row", err)
	}
	if err = store.PutTenant(ctx, ports.Tenant{ID: other, Name: "Another account"}); err == nil {
		t.Fatal("removing Tenant data erased its permanent file binding")
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	// Omitting an option cannot turn an already bound file back into a shared
	// embedded store or bypass the application's provisioning guard.
	store, err = Open(path, Options{StartupTimeout: 2 * time.Minute, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	if store.BoundTenantID() != id {
		t.Fatal("embedded reopen forgot persistent Tenant binding")
	}
	if err = store.PutTenant(ctx, ports.Tenant{ID: other, Name: "Another account"}); err == nil {
		t.Fatal("embedded reopen bypassed persistent Tenant binding")
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	wrong, err := Open(path, Options{StartupTimeout: 2 * time.Minute, MigrateOnOpen: true, TenantID: other})
	if wrong != nil {
		_ = wrong.Close()
	}
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodePreconditionFailed {
		t.Fatalf("different Tenant did not fail closed: %v", err)
	}
}

func TestSQLiteTenantBindingRefusesExistingSharedFileWithoutChangingIt(t *testing.T) {
	ctx := context.Background()
	id := domain.MustParseID("0199f490-0000-7000-8000-000000000011")
	other := domain.MustParseID("0199f490-0000-7000-8000-000000000012")
	path := filepath.Join(t.TempDir(), "shared.db")
	store, err := Open(path, Options{StartupTimeout: 2 * time.Minute, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	for index, tenant := range []domain.ID{id, other} {
		name := []string{"First account", "Second account"}[index]
		if err = store.PutTenant(ctx, ports.Tenant{ID: tenant, Name: name}); err != nil {
			t.Fatal(err)
		}
	}
	if !store.BoundTenantID().IsZero() {
		t.Fatal("unbound embedded file unexpectedly bound a Tenant")
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	bound, err := Open(path, Options{StartupTimeout: 2 * time.Minute, MigrateOnOpen: true, TenantID: id})
	if bound != nil {
		_ = bound.Close()
	}
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodePreconditionFailed {
		t.Fatalf("shared file should require explicit migration: %v", err)
	}
	store, err = Open(path, Options{StartupTimeout: 2 * time.Minute, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var count, bindings int
	if err = store.db.QueryRowContext(ctx, `SELECT count(*) FROM tenants`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if err = store.db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_schema WHERE name IN ('wos_tenant_identity_insert','wos_tenant_identity_update','wos_single_tenant')`).Scan(&bindings); err != nil {
		t.Fatal(err)
	}
	if count != 2 || bindings != 0 || !store.BoundTenantID().IsZero() {
		t.Fatalf("failed binding changed shared data: tenants=%d bindings=%d", count, bindings)
	}
}

func TestSQLiteTenantBindingRejectsFutureSchemaBeforeWriting(t *testing.T) {
	ctx := context.Background()
	id := domain.MustParseID("0199f490-0000-7000-8000-000000000031")
	path := filepath.Join(t.TempDir(), "future.db")
	store, err := Open(path, Options{StartupTimeout: 2 * time.Minute, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	version, err := store.SchemaVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = store.db.ExecContext(ctx, `INSERT INTO schema_migrations(version,name,checksum,applied_at) VALUES(?,'future','future',0)`, version+1); err != nil {
		t.Fatal(err)
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	bound, err := Open(path, Options{MigrateOnOpen: false, TenantID: id})
	if bound != nil {
		_ = bound.Close()
	}
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeInvalidConfig {
		t.Fatalf("unsupported schema was not rejected: %v", err)
	}
	// Unbound embedded inspection remains available without migration. A
	// rejected runtime must not leave a Tenant row or any binding DDL behind.
	store, err = Open(path, Options{MigrateOnOpen: false})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	var tenants, guards int
	if err = store.db.QueryRowContext(ctx, `SELECT count(*) FROM tenants`).Scan(&tenants); err != nil {
		t.Fatal(err)
	}
	if err = store.db.QueryRowContext(ctx, `SELECT count(*) FROM sqlite_schema WHERE name IN ('wos_tenant_identity_insert','wos_tenant_identity_update','wos_single_tenant')`).Scan(&guards); err != nil {
		t.Fatal(err)
	}
	if tenants != 0 || guards != 0 {
		t.Fatalf("unsupported schema changed before runtime rejection: tenants=%d guards=%d", tenants, guards)
	}
}
