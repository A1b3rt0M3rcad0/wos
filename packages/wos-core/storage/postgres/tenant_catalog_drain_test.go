package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestTenantInitializerCompletionWaitsForServerLockRelease(t *testing.T) {
	for _, operation := range []string{"provision", "migrate"} {
		t.Run(operation, func(t *testing.T) {
			c := newTestTenantCatalog(t)
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			id := catalogTestID(t, "initializer close acknowledgment")
			var before TenantPlacement
			if operation == "migrate" {
				before = provisionCatalogTestTenant(t, c, "Drain before migration receipt")
				id = before.TenantID
				var err error
				before, err = c.ChangeTenantState(ctx, id, before.Version, TenantSuspended, "offline migration fixture")
				if err != nil {
					t.Fatal(err)
				}
			}
			// Model PostgreSQL still retaining an initializer session lock after
			// the client has closed its Store. Control the server acknowledgment
			// explicitly rather than relying on a timing-sensitive socket race.
			closingSession, err := c.db.Conn(ctx)
			if err != nil {
				t.Fatal(err)
			}
			defer closingSession.Close()
			type initialized struct {
				placement TenantPlacement
				pid       int
			}
			ready := make(chan initialized, 1)
			type outcome struct {
				placement TenantPlacement
				err       error
			}
			result := make(chan outcome, 1)
			done := make(chan struct{})
			unlock := func() {
				closeCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
				defer stop()
				_, _ = closingSession.ExecContext(closeCtx, `SELECT pg_advisory_unlock_shared($1)`, tenantInitializationKey(c.schema, id))
			}
			defer func() {
				unlock()
				cancel()
				select {
				case <-done:
				case <-time.After(5 * time.Second):
					t.Error("initializer operation did not stop after cancellation")
				}
			}()
			initialize := func(ctx context.Context, p TenantPlacement) error {
				if err := initializeCatalogTestTenant(ctx, p); err != nil {
					return err
				}
				var pid int
				if err := p.initializerLease.conn.QueryRowContext(ctx, `SELECT pg_backend_pid()`).Scan(&pid); err != nil {
					return err
				}
				if _, err := closingSession.ExecContext(ctx, `SELECT pg_advisory_lock_shared($1)`, tenantInitializationKey(c.schema, id)); err != nil {
					return err
				}
				ready <- initialized{placement: p, pid: pid}
				return nil
			}
			go func() {
				defer close(done)
				var p TenantPlacement
				var err error
				if operation == "provision" {
					p, err = c.ProvisionTenant(ctx, TenantProvision{TenantID: id, Name: "Drain before activation", Reason: "server acknowledgment fixture"}, initialize)
				} else {
					p, err = c.MigrateTenant(ctx, id, before.Version, "server acknowledgment fixture", initialize)
				}
				result <- outcome{placement: p, err: err}
			}()
			var initializedState initialized
			select {
			case initializedState = <-ready:
			case early := <-result:
				t.Fatal("initializer did not reach closing session fixture", early.err)
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
			ticker := time.NewTicker(10 * time.Millisecond)
			defer ticker.Stop()
			for {
				var waiting bool
				if err := c.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM pg_locks WHERE pid=$1 AND locktype='advisory' AND NOT granted)`, initializedState.pid).Scan(&waiting); err != nil {
					t.Fatal(err)
				}
				if waiting {
					break
				}
				select {
				case early := <-result:
					t.Fatal("completion did not wait for PostgreSQL to release the initializer lock", early.err)
				case <-ctx.Done():
					t.Fatal(ctx.Err())
				case <-ticker.C:
				}
			}
			current, err := c.GetTenant(ctx, id)
			if err != nil || current.State != initializedState.placement.State || current.Version != initializedState.placement.Version {
				t.Fatal("lifecycle changed before initializer sessions drained", current, err)
			}
			_, err = c.DeleteTenant(ctx, id, current.Version, "must retain parent lifecycle fence while draining")
			requireTenantError(t, err, domain.ErrorCodePreconditionFailed)
			if s, err := Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), initializedState.placement.InitializerOptions()); err == nil {
				_ = s.Close()
				t.Fatal("returned initializer capability admitted another data connection")
			}
			unlock()
			select {
			case completed := <-result:
				want := TenantActive
				if operation == "migrate" {
					want = TenantSuspended
				}
				if completed.err != nil || completed.placement.State != want || completed.placement.Version != current.Version+1 {
					t.Fatal("initializer completion failed after server released the lock", completed.placement, completed.err)
				}
			case <-ctx.Done():
				t.Fatal(ctx.Err())
			}
		})
	}
}
