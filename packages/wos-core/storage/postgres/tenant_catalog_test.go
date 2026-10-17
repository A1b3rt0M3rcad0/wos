package postgres

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func newTestTenantCatalog(t *testing.T) *TenantCatalog {
	t.Helper()
	requirePostgres(t)
	digest := sha256.Sum256([]byte(t.TempDir()))
	schema := fmt.Sprintf("wos_catalog_test_%x", digest[:10])
	c, err := OpenTenantCatalog(context.Background(), os.Getenv("WOS_TEST_POSTGRES_DSN"), CatalogOptions{Schema: schema, MaxOpenConnections: 2, MaxLeases: 8, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		rows, err := c.ListTenants(ctx, "", 1000)
		if err == nil {
			for _, p := range rows {
				if _, err := c.db.ExecContext(ctx, `DROP SCHEMA IF EXISTS `+p.Schema+` CASCADE`); err != nil {
					t.Error("clean Tenant fixture", err)
				}
			}
		}
		if _, err := c.db.ExecContext(ctx, `DROP SCHEMA `+c.schema+` CASCADE`); err != nil {
			t.Error("clean catalog fixture", err)
		}
		if err := c.Close(); err != nil {
			t.Error(err)
		}
	})
	return c
}

func catalogTestID(t *testing.T, label string) domain.ID {
	t.Helper()
	digest := sha256.Sum256([]byte(t.TempDir() + ":" + label))
	raw := fmt.Sprintf("%x", digest[:16])
	return domain.MustParseID(raw[:8] + "-" + raw[8:12] + "-7" + raw[13:16] + "-8" + raw[17:20] + "-" + raw[20:])
}

func initializeCatalogTestTenant(ctx context.Context, p TenantPlacement) error {
	options := p.InitializerOptions()
	options.MaxOpenConnections = 2
	s, err := Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), options)
	if err != nil {
		return err
	}
	return s.Close()
}

func provisionCatalogTestTenant(t *testing.T, c *TenantCatalog, name string) TenantPlacement {
	t.Helper()
	p, err := c.ProvisionTenant(context.Background(), TenantProvision{TenantID: catalogTestID(t, name), Name: name, Reason: "create isolated acceptance fixture"}, initializeCatalogTestTenant)
	if err != nil {
		t.Fatal(err)
	}
	if p.State != TenantActive || p.Version != 2 {
		t.Fatal("provisioning did not activate exactly once", p)
	}
	return p
}

func requireTenantError(t *testing.T, err error, code domain.ErrorCode) {
	t.Helper()
	if got, _ := domain.ErrorCodeOf(err); got != code {
		t.Fatalf("expected %s; got %v", code, err)
	}
}

func TestTenantCatalogLifecyclePinsEveryDataConnection(t *testing.T) {
	c := newTestTenantCatalog(t)
	ctx := context.Background()
	p := provisionCatalogTestTenant(t, c, "Lifecycle")
	lease, err := c.AcquireTenantLease(ctx, p.TenantID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lease.Close() })
	s, err := Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), Options{Schema: p.Schema, TenantID: p.TenantID, TenantLease: lease, MaxOpenConnections: 2})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	if s.BoundTenantID() != p.TenantID {
		t.Fatal("store lost its explicit Tenant binding")
	}
	_, err = c.ChangeTenantState(ctx, p.TenantID, p.Version, TenantSuspended, "must drain")
	requireTenantError(t, err, domain.ErrorCodePreconditionFailed)
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	// The catalog session alone is insufficient: every live data session must
	// independently fence deletion, even if a caller closes the lease early.
	_, err = c.ChangeTenantState(ctx, p.TenantID, p.Version, TenantSuspended, "still has data connections")
	requireTenantError(t, err, domain.ErrorCodePreconditionFailed)
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	p, err = c.ChangeTenantState(ctx, p.TenantID, p.Version, TenantSuspended, "all replicas drained")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.AcquireTenantLease(ctx, p.TenantID)
	requireTenantError(t, err, domain.ErrorCodePreconditionFailed)
	_, err = c.ChangeTenantState(ctx, p.TenantID, p.Version-1, TenantActive, "stale operator")
	requireTenantError(t, err, domain.ErrorCodeVersionConflict)
	p, err = c.ChangeTenantState(ctx, p.TenantID, p.Version, TenantActive, "resume")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.DeleteTenant(ctx, p.TenantID, p.Version, "active cannot be deleted")
	requireTenantError(t, err, domain.ErrorCodeInvalidTransition)
	p, err = c.ChangeTenantState(ctx, p.TenantID, p.Version, TenantSuspended, "retire")
	if err != nil {
		t.Fatal(err)
	}
	p, err = c.DeleteTenant(ctx, p.TenantID, p.Version, "retention complete")
	if err != nil {
		t.Fatal(err)
	}
	if p.State != TenantDeleted || p.Version != 6 {
		t.Fatal("incorrect lifecycle result", p)
	}
	var exists bool
	if err := c.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=$1)`, p.Schema).Scan(&exists); err != nil || exists {
		t.Fatal("deleted Tenant schema still exists", exists, err)
	}
	var audits int
	if err := c.db.QueryRowContext(ctx, `SELECT count(*) FROM tenant_placement_audit WHERE tenant_id=$1`, p.TenantID.String()).Scan(&audits); err != nil || audits != 6 {
		t.Fatal("incomplete durable lifecycle audit", audits, err)
	}
	_, err = c.ProvisionTenant(ctx, TenantProvision{TenantID: p.TenantID, Name: "reused identity", Reason: "must reject"}, initializeCatalogTestTenant)
	requireTenantError(t, err, domain.ErrorCodeAlreadyExists)
	all, err := c.ListTenants(ctx, "", 10)
	if err != nil || len(all) != 1 || all[0].State != TenantDeleted {
		t.Fatal("deleted identity did not remain in catalog", all, err)
	}
}

func TestTenantProvisionFailureIsUnrouteableAndRetryUsesCAS(t *testing.T) {
	c := newTestTenantCatalog(t)
	ctx := context.Background()
	id := catalogTestID(t, "interruption")
	interrupted := errors.New("initializer interrupted")
	p, err := c.ProvisionTenant(ctx, TenantProvision{TenantID: id, Name: "Retry", Reason: "initial attempt"}, func(ctx context.Context, p TenantPlacement) error {
		if err := initializeCatalogTestTenant(ctx, p); err != nil {
			return err
		}
		return interrupted
	})
	if !errors.Is(err, interrupted) || p.State != TenantProvisioning {
		t.Fatal("failed initializer became active", p, err)
	}
	_, err = c.AcquireTenantLease(ctx, id)
	requireTenantError(t, err, domain.ErrorCodePreconditionFailed)
	_, err = c.RetryProvisionTenant(ctx, id, p.Version+1, "stale retry", initializeCatalogTestTenant)
	requireTenantError(t, err, domain.ErrorCodeVersionConflict)
	p, err = c.RetryProvisionTenant(ctx, id, p.Version, "resume interrupted initialization", initializeCatalogTestTenant)
	if err != nil || p.State != TenantActive || p.Version != 3 {
		t.Fatal("retry did not preserve/activate initialized schema", p, err)
	}
	if _, err := c.AcquireTenantLease(ctx, catalogTestID(t, "missing")); err == nil {
		t.Fatal("missing Tenant was routed")
	}
}

func TestTenantSchemaRejectsDifferentIdentityAndHasNoPublicFallback(t *testing.T) {
	c := newTestTenantCatalog(t)
	ctx := context.Background()
	p := provisionCatalogTestTenant(t, c, "Bound")
	otherID := catalogTestID(t, "wrong identity")
	_, err := Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), Options{Schema: p.Schema, TenantID: otherID})
	requireTenantError(t, err, domain.ErrorCodePreconditionFailed)
	s, err := Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), Options{Schema: p.Schema, TenantID: p.TenantID})
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.PutTenant(ctx, ports.Tenant{ID: otherID, Name: "wrong"}); err == nil {
		t.Fatal("persistent schema constraint accepted a second Tenant")
	}
	// Even deleting the singleton row cannot make this schema change ownership.
	if _, err := s.db.ExecContext(ctx, `DELETE FROM tenants WHERE id=$1`, p.TenantID.String()); err != nil {
		t.Fatal(err)
	}
	_, err = Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), Options{Schema: p.Schema, TenantID: otherID})
	requireTenantError(t, err, domain.ErrorCodePreconditionFailed)
	var path string
	if err := s.db.QueryRowContext(ctx, `SHOW search_path`).Scan(&path); err != nil || path != p.Schema || strings.Contains(path, "public") {
		t.Fatal("Tenant pool has fallback schema", path, err)
	}
}

func TestTenantDeletionRefusesDependenciesInAnotherSchema(t *testing.T) {
	c := newTestTenantCatalog(t)
	ctx := context.Background()
	a := provisionCatalogTestTenant(t, c, "Owner")
	b := provisionCatalogTestTenant(t, c, "Neighbor")
	if _, err := c.db.ExecContext(ctx, `CREATE TABLE `+a.Schema+`.deletion_probe(id INTEGER PRIMARY KEY); CREATE TABLE `+b.Schema+`.outside_reference(owner_id INTEGER REFERENCES `+a.Schema+`.deletion_probe(id));`); err != nil {
		t.Fatal(err)
	}
	a, err := c.ChangeTenantState(ctx, a.TenantID, a.Version, TenantSuspended, "drained")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.DeleteTenant(ctx, a.TenantID, a.Version, "must preserve neighboring dependencies")
	requireTenantError(t, err, domain.ErrorCodePreconditionFailed)
	var remains bool
	if err := c.db.QueryRowContext(ctx, `SELECT to_regclass($1) IS NOT NULL`, b.Schema+".outside_reference").Scan(&remains); err != nil || !remains {
		t.Fatal("deletion changed another Tenant", remains, err)
	}
	if _, err := c.db.ExecContext(ctx, `DROP TABLE `+b.Schema+`.outside_reference`); err != nil {
		t.Fatal(err)
	}
	if _, err := c.db.ExecContext(ctx, `CREATE VIEW `+b.Schema+`.outside_view AS SELECT id FROM `+a.Schema+`.deletion_probe`); err != nil {
		t.Fatal(err)
	}
	_, err = c.DeleteTenant(ctx, a.TenantID, a.Version, "cross-schema view must survive")
	requireTenantError(t, err, domain.ErrorCodePreconditionFailed)
	if _, err := c.db.ExecContext(ctx, `DROP VIEW `+b.Schema+`.outside_view`); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DeleteTenant(ctx, a.TenantID, a.Version, "external reference removed"); err != nil {
		t.Fatal(err)
	}
}

func TestTenantCatalogRejectsUnsafeNamesAndUnboundedPools(t *testing.T) {
	for _, options := range []CatalogOptions{{Schema: "public"}, {Schema: "pg_catalog"}, {Schema: "wos_t_reserved"}, {Schema: "x;DROP SCHEMA public"}, {Schema: "wos_control", MaxOpenConnections: -1}, {Schema: "wos_control", MaxLeases: 4097}} {
		_, err := OpenTenantCatalog(context.Background(), "this DSN must never be parsed", options)
		requireTenantError(t, err, domain.ErrorCodeInvalidConfig)
	}
}

func TestTenantLeaseLossFailsReadinessAndReconnectRevalidatesPlacement(t *testing.T) {
	c := newTestTenantCatalog(t)
	ctx := context.Background()
	p := provisionCatalogTestTenant(t, c, "Lost session")
	lease, err := c.AcquireTenantLease(ctx, p.TenantID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = lease.Close() })
	s, err := Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), Options{Schema: p.Schema, TenantID: p.TenantID, TenantLease: lease, MaxOpenConnections: 1})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	var leasePID, dataPID int
	if err := lease.conn.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&leasePID); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&dataPID); err != nil {
		t.Fatal(err)
	}
	if _, err := c.db.ExecContext(ctx, `SELECT pg_terminate_backend($1)`, leasePID); err != nil {
		t.Fatal(err)
	}
	if err := lease.CheckActive(ctx); err == nil {
		t.Fatal("a replacement connection hid loss of the held runtime lock")
	}
	_, err = c.ChangeTenantState(ctx, p.TenantID, p.Version, TenantSuspended, "data pool still connected")
	requireTenantError(t, err, domain.ErrorCodePreconditionFailed)
	if _, err := c.db.ExecContext(ctx, `SELECT pg_terminate_backend($1,5000)`, dataPID); err != nil {
		t.Fatal(err)
	}
	p, err = c.ChangeTenantState(ctx, p.TenantID, p.Version, TenantSuspended, "all database sessions lost")
	if err != nil {
		t.Fatal(err)
	}
	// database/sql may reconnect after a bad connection, but the new pgx
	// session must run the catalog guard before any operational SQL.
	var count int
	for i := 0; i < 3; i++ {
		if err := s.db.QueryRowContext(ctx, `SELECT count(*) FROM tenants`).Scan(&count); err == nil {
			t.Fatal("data pool reconnected after Tenant suspension")
		}
	}
	if err := s.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := c.DeleteTenant(ctx, p.TenantID, p.Version, "failed connection did not leak shared lock"); err != nil {
		t.Fatal(err)
	}
}

func TestProvisioningConnectionLossCannotRecreateDeletedSchema(t *testing.T) {
	for _, openBeforeLoss := range []bool{false, true} {
		t.Run(fmt.Sprintf("initializer_already_connected_%t", openBeforeLoss), func(t *testing.T) {
			c := newTestTenantCatalog(t)
			ctx := context.Background()
			id := catalogTestID(t, "provisioning fence loss")
			_, err := c.ProvisionTenant(ctx, TenantProvision{TenantID: id, Name: "Interrupted", Reason: "test interrupted initializer"}, func(ctx context.Context, p TenantPlacement) error {
				var s *Store
				if openBeforeLoss {
					var err error
					s, err = Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), p.InitializerOptions())
					if err != nil {
						return err
					}
					defer s.Close()
				}
				var pid int
				if err := p.initializerLease.conn.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
					return err
				}
				if _, err := c.db.ExecContext(ctx, `SELECT pg_terminate_backend($1,5000)`, pid); err != nil {
					return err
				}
				if s != nil {
					_, err := c.DeleteTenant(ctx, p.TenantID, p.Version, "live initializer data is still protected")
					requireTenantError(t, err, domain.ErrorCodePreconditionFailed)
					if err := s.Close(); err != nil {
						return err
					}
				}
				if _, err := c.DeleteTenant(ctx, p.TenantID, p.Version, "delete after initializer sessions drained"); err != nil {
					return err
				}
				if recreated, err := Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), p.InitializerOptions()); err == nil {
					_ = recreated.Close()
					t.Fatal("delayed initializer recreated a tombstoned Tenant schema")
				}
				return errors.New("initializer intentionally interrupted")
			})
			if err == nil {
				t.Fatal("interrupted provisioning unexpectedly succeeded")
			}
			p, err := c.GetTenant(ctx, id)
			if err != nil || p.State != TenantDeleted {
				t.Fatal("Tenant deletion tombstone was lost", p, err)
			}
			var exists bool
			if err := c.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_namespace WHERE nspname=$1)`, p.Schema).Scan(&exists); err != nil || exists {
				t.Fatal("delayed initializer left an orphan schema after deletion", exists, err)
			}
		})
	}
}

func TestRejectedTenantConnectionClosesSocketAndSessionLock(t *testing.T) {
	c := newTestTenantCatalog(t)
	ctx := context.Background()
	p := provisionCatalogTestTenant(t, c, "Rejected connection")
	lease, err := c.AcquireTenantLease(ctx, p.TenantID)
	if err != nil {
		t.Fatal(err)
	}
	// Force the catalog inconsistency only for this test. The SQL constructor
	// must still clean up its connection if validation fails after locking it.
	if _, err := c.db.ExecContext(ctx, `UPDATE tenant_placements SET version=version+1 WHERE tenant_id=$1`, p.TenantID.String()); err != nil {
		t.Fatal(err)
	}
	dsn, err := url.Parse(os.Getenv("WOS_TEST_POSTGRES_DSN"))
	if err != nil {
		t.Fatal(err)
	}
	query := dsn.Query()
	applicationName := "rejected_tenant_" + strings.ReplaceAll(p.TenantID.String(), "-", "")
	query.Set("application_name", applicationName)
	dsn.RawQuery = query.Encode()
	for i := 0; i < 3; i++ {
		if s, err := Open(dsn.String(), Options{Schema: p.Schema, TenantID: p.TenantID, TenantLease: lease}); err == nil {
			_ = s.Close()
			t.Fatal("changed placement was accepted by a new data connection")
		}
	}
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	var sessions int
	if err := c.db.QueryRowContext(ctx, `SELECT count(*) FROM pg_stat_activity WHERE application_name=$1`, applicationName).Scan(&sessions); err != nil || sessions != 0 {
		t.Fatal("rejected AfterConnect callback leaked database sessions", sessions, err)
	}
	if _, err := c.ChangeTenantState(ctx, p.TenantID, p.Version+1, TenantSuspended, "failed callbacks released their session locks"); err != nil {
		t.Fatal(err)
	}
}

func TestFiveTenantsShareDatabaseWithIndependentCredentialsAndPools(t *testing.T) {
	c := newTestTenantCatalog(t)
	ctx := context.Background()
	credentialID := domain.MustParseID("019a0251-0000-7000-8000-000000000001")
	actor := domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "tenant-test", ID: "same-human"}
	const digest = "same-credential-digest-in-five-separate-schemas"
	for i := 0; i < 5; i++ {
		p := provisionCatalogTestTenant(t, c, fmt.Sprintf("Account %d", i))
		lease, err := c.AcquireTenantLease(ctx, p.TenantID)
		if err != nil {
			t.Fatal(err)
		}
		s, err := Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), Options{Schema: p.Schema, TenantID: p.TenantID, TenantLease: lease, MaxOpenConnections: 2})
		if err != nil {
			_ = lease.Close()
			t.Fatal(err)
		}
		// Hold all five pools at once. Duplicate credential/entity identifiers
		// across them must remain independent and resolve to their bound Tenant.
		t.Cleanup(func() { _ = s.Close(); _ = lease.Close() })
		credential := ports.Credential{ID: credentialID, TenantID: p.TenantID, PrincipalID: "same-human", Actor: actor, Digest: digest, ExpiresAt: time.Now().Add(time.Hour)}
		if err := s.PutCredential(ctx, credential); err != nil {
			t.Fatal(err)
		}
		resolved, err := s.GetCredential(ctx, digest)
		if err != nil || resolved.TenantID != p.TenantID || resolved.ID != credentialID {
			t.Fatal("credential escaped its Tenant schema", resolved, err)
		}
		if i == 0 {
			if err := s.RevokeCredential(ctx, credentialID); err != nil {
				t.Fatal(err)
			}
		} else if resolved.Revoked {
			t.Fatal("revoking a matching credential in another schema affected this Tenant")
		}
		if s.db.Stats().MaxOpenConnections != 2 {
			t.Fatal("Tenant pool exceeded configured connection budget")
		}
	}
	placements, err := c.ListTenants(ctx, "", 3)
	if err != nil || len(placements) != 3 {
		t.Fatal("catalog did not honor bounded first page", placements, err)
	}
	rest, err := c.ListTenants(ctx, placements[2].TenantID, 3)
	if err != nil || len(rest) != 2 {
		t.Fatal("catalog continuation skipped or duplicated Tenants", rest, err)
	}
}

func TestTenantAdmissionAndSuspensionHaveOneWinner(t *testing.T) {
	c := newTestTenantCatalog(t)
	ctx := context.Background()
	p := provisionCatalogTestTenant(t, c, "Concurrent lifecycle")
	for i := 0; i < 12; i++ {
		start := make(chan struct{})
		var wg sync.WaitGroup
		var lease *TenantLease
		var acquireErr, suspendErr error
		var suspended TenantPlacement
		wg.Add(2)
		go func() {
			defer wg.Done()
			<-start
			lease, acquireErr = c.AcquireTenantLease(ctx, p.TenantID)
		}()
		go func() {
			defer wg.Done()
			<-start
			suspended, suspendErr = c.ChangeTenantState(ctx, p.TenantID, p.Version, TenantSuspended, "concurrent suspension")
		}()
		close(start)
		wg.Wait()
		if acquireErr == nil && suspendErr == nil {
			_ = lease.Close()
			t.Fatal("runtime admitted while suspension succeeded")
		}
		if acquireErr != nil {
			requireTenantError(t, acquireErr, domain.ErrorCodePreconditionFailed)
		} else if err := lease.Close(); err != nil {
			t.Fatal(err)
		}
		if suspendErr != nil {
			requireTenantError(t, suspendErr, domain.ErrorCodePreconditionFailed)
			var err error
			suspended, err = c.ChangeTenantState(ctx, p.TenantID, p.Version, TenantSuspended, "admitted runtime drained")
			if err != nil {
				t.Fatal(err)
			}
		}
		var err error
		p, err = c.ChangeTenantState(ctx, p.TenantID, suspended.Version, TenantActive, "next concurrency iteration")
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestTenantMigrationRequiresSuspensionAndPreservesIt(t *testing.T) {
	c := newTestTenantCatalog(t)
	ctx := context.Background()
	p := provisionCatalogTestTenant(t, c, "Upgrade")
	_, err := c.MigrateTenant(ctx, p.TenantID, p.Version, "online migration must fail", initializeCatalogTestTenant)
	requireTenantError(t, err, domain.ErrorCodeInvalidTransition)
	lease, err := c.AcquireTenantLease(ctx, p.TenantID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), Options{Schema: p.Schema, TenantID: p.TenantID, TenantLease: lease, MigrateOnOpen: true})
	requireTenantError(t, err, domain.ErrorCodeInvalidConfig)
	if err := lease.Close(); err != nil {
		t.Fatal(err)
	}
	p, err = c.ChangeTenantState(ctx, p.TenantID, p.Version, TenantSuspended, "stop before upgrade")
	if err != nil {
		t.Fatal(err)
	}
	_, err = c.MigrateTenant(ctx, p.TenantID, p.Version-1, "stale schema upgrade", initializeCatalogTestTenant)
	requireTenantError(t, err, domain.ErrorCodeVersionConflict)
	failed := errors.New("migration interrupted")
	_, err = c.MigrateTenant(ctx, p.TenantID, p.Version, "failed schema upgrade", func(context.Context, TenantPlacement) error { return failed })
	if !errors.Is(err, failed) {
		t.Fatal(err)
	}
	still, err := c.GetTenant(ctx, p.TenantID)
	if err != nil || still.State != TenantSuspended || still.Version != p.Version {
		t.Fatal("failed migration changed lifecycle", still, err)
	}
	var retainedOptions Options
	p, err = c.MigrateTenant(ctx, p.TenantID, p.Version, "retry supported migration chain", func(ctx context.Context, placement TenantPlacement) error {
		retainedOptions = placement.InitializerOptions()
		return initializeCatalogTestTenant(ctx, placement)
	})
	if err != nil || p.State != TenantSuspended || p.Version != still.Version+1 {
		t.Fatal("migration did not stay suspended", p, err)
	}
	if s, err := Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), retainedOptions); err == nil {
		_ = s.Close()
		t.Fatal("initializer capability remained usable after callback returned")
	}
	_, err = c.AcquireTenantLease(ctx, p.TenantID)
	requireTenantError(t, err, domain.ErrorCodePreconditionFailed)
	if _, err := c.ChangeTenantState(ctx, p.TenantID, p.Version, TenantActive, "explicit post-upgrade resume"); err != nil {
		t.Fatal(err)
	}
}

func TestRuntimeBindingDoesNotInitializeMissingConstraints(t *testing.T) {
	c := newTestTenantCatalog(t)
	ctx := context.Background()
	p := provisionCatalogTestTenant(t, c, "Verify only")
	if _, err := c.db.ExecContext(ctx, `DROP INDEX `+p.Schema+`.wos_single_tenant`); err != nil {
		t.Fatal(err)
	}
	lease, err := c.AcquireTenantLease(ctx, p.TenantID)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	_, err = Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), Options{Schema: p.Schema, TenantID: p.TenantID, TenantLease: lease})
	requireTenantError(t, err, domain.ErrorCodePreconditionFailed)
	var exists bool
	if err := c.db.QueryRowContext(ctx, `SELECT to_regclass($1) IS NOT NULL`, p.Schema+".wos_single_tenant").Scan(&exists); err != nil || exists {
		t.Fatal("live runtime created a missing binding constraint", exists, err)
	}
}

func TestRuntimeValidatesMigrationLedgerWithoutChangingIt(t *testing.T) {
	c := newTestTenantCatalog(t)
	ctx := context.Background()
	p := provisionCatalogTestTenant(t, c, "Verify migration inventory")
	lease, err := c.AcquireTenantLease(ctx, p.TenantID)
	if err != nil {
		t.Fatal(err)
	}
	defer lease.Close()
	var version int64
	var name, checksum string
	if err := c.db.QueryRowContext(ctx, `SELECT version,name,checksum FROM `+p.Schema+`.schema_migrations ORDER BY version DESC LIMIT 1`).Scan(&version, &name, &checksum); err != nil {
		t.Fatal(err)
	}
	for _, field := range []string{"name", "checksum"} {
		t.Run(field, func(t *testing.T) {
			if _, err := c.db.ExecContext(ctx, `UPDATE `+p.Schema+`.schema_migrations SET `+field+`=$1 WHERE version=$2`, "tampered", version); err != nil {
				t.Fatal(err)
			}
			s, err := Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), Options{Schema: p.Schema, TenantID: p.TenantID, TenantLease: lease})
			if s != nil {
				_ = s.Close()
			}
			requireTenantError(t, err, domain.ErrorCodePreconditionFailed)
			var retained string
			if err := c.db.QueryRowContext(ctx, `SELECT `+field+` FROM `+p.Schema+`.schema_migrations WHERE version=$1`, version).Scan(&retained); err != nil || retained != "tampered" {
				t.Fatal("runtime changed migration ledger", retained, err)
			}
			if _, err := c.db.ExecContext(ctx, `UPDATE `+p.Schema+`.schema_migrations SET name=$1,checksum=$2 WHERE version=$3`, name, checksum, version); err != nil {
				t.Fatal(err)
			}
		})
	}
	s, err := Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), Options{Schema: p.Schema, TenantID: p.TenantID, TenantLease: lease})
	if err != nil {
		t.Fatal("runtime rejected restored compiled migration inventory", err)
	}
	_ = s.Close()
}
