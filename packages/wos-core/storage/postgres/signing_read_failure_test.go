package postgres

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/jackc/pgx/v5/pgconn"
)

// Inject a real PostgreSQL read failure at the group lookup, rather than
// exercising mapSQLError alone. The view/function exist only in this test's
// isolated schema; production data and migrations remain untouched.
func TestPrincipalGroupReadPreservesDatabaseSerializationFailure(t *testing.T) {
	requirePostgres(t)
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "principal-group-read-failure.db"))
	ns := testID("0199f391-0000-7000-8000-000000000001")
	u, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	group, err := u.(ports.SigningIdentityUnitOfWork).SigningIdentity().PrincipalGroup(ctx, ns, "executor")
	u.Rollback()
	if err != nil || group != "" {
		t.Fatal("absent optional group changed semantics", group, err)
	}
	_, err = store.db.ExecContext(ctx, `
ALTER TABLE principal_review_groups RENAME TO principal_review_groups_fixture;
CREATE FUNCTION wos_principal_group_fault() RETURNS TEXT LANGUAGE plpgsql VOLATILE AS $fault$
BEGIN
 RAISE EXCEPTION 'injected Principal-group serialization conflict' USING ERRCODE='40001';
END
$fault$;
CREATE VIEW principal_review_groups AS SELECT
 '0199f391-0000-7000-8000-000000000001'::TEXT AS tenant_id,
 'executor'::TEXT AS principal_id,
 wos_principal_group_fault() AS separation_group;
`)
	if err != nil {
		t.Fatal(err)
	}
	u, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer u.Rollback()
	_, err = u.(ports.SigningIdentityUnitOfWork).SigningIdentity().PrincipalGroup(ctx, ns, "executor")
	if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodeTransactionConflict {
		t.Fatal("SQL serialization failure escaped its storage boundary", err)
	}
	var cause *pgconn.PgError
	if !errors.As(err, &cause) || cause.Code != "40001" {
		t.Fatal("typed conflict lost its database cause", err)
	}
}
