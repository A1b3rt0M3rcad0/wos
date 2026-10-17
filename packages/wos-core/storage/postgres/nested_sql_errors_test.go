package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/jackc/pgx/v5/pgconn"
)

// PostgreSQL can deliver a serialization failure after QueryContext succeeds
// and after a consumer has read one row. That error must retain the retryable
// storage contract when it emerges from Rows.Err, rather than becoming a raw
// server error in a signed acquisition/return journey.
func TestNestedReadMapsStreamedPostgresSerializationFailure(t *testing.T) {
	requirePostgres(t)
	store, outcome := persistenceHistoryFixture(t, 2)
	ctx := context.Background()
	for _, id := range []string{"good", "zzfault"} {
		actorJSON := `{"kind":"human","provider":"test","id":"` + id + `"}`
		_, err := store.db.ExecContext(ctx, `INSERT INTO entity_actor_links(tenant_id,outcome_id,entity_id,role,actor_kind,actor_provider,actor_id,actor_json) VALUES($1,$2,$3,'owner','human','test',$4,$5)`, outcome.TenantID.String(), outcome.ID.String(), outcome.ID.String(), id, actorJSON)
		if err != nil {
			t.Fatal(err)
		}
	}
	// This isolated schema belongs to this test. Foreign keys keep referencing
	// the renamed real table; the view faults only the read codec under test.
	_, err := store.db.ExecContext(ctx, `ALTER TABLE entity_actor_links RENAME TO nested_error_actor_links;
CREATE INDEX nested_error_actor_order ON nested_error_actor_links(actor_kind,actor_provider,actor_id);
CREATE FUNCTION nested_serialization_failure() RETURNS TEXT LANGUAGE plpgsql VOLATILE AS $$BEGIN RAISE EXCEPTION USING ERRCODE='40001', MESSAGE='injected nested read serialization conflict'; END$$;
CREATE VIEW entity_actor_links AS SELECT tenant_id,outcome_id,entity_id,role,actor_kind,actor_provider,actor_id,CASE WHEN actor_id='zzfault' THEN nested_serialization_failure() ELSE actor_json END AS actor_json FROM nested_error_actor_links ORDER BY actor_kind,actor_provider,actor_id`)
	if err != nil {
		t.Fatal(err)
	}
	// Force the existing ordered primary-key index, so the faulting projection
	// is evaluated after the first valid row rather than during a blocking sort.
	probe, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer probe.Rollback()
	if _, err := probe.ExecContext(ctx, "SET LOCAL enable_seqscan=off; SET LOCAL enable_sort=off"); err != nil {
		t.Fatal(err)
	}
	// Establish that this driver/plan actually reaches the deferred error path.
	rows, err := probe.QueryContext(ctx, `SELECT actor_json FROM entity_actor_links WHERE tenant_id=$1 AND outcome_id=$2 AND entity_id=$3 AND role='owner' ORDER BY actor_kind,actor_provider,actor_id`, outcome.TenantID.String(), outcome.ID.String(), outcome.ID.String())
	if err != nil {
		t.Fatalf("fault did not reach the intended streamed read boundary: %v", err)
	}
	if !rows.Next() {
		t.Fatalf("first valid row missing: %v", rows.Err())
	}
	var first string
	if err := rows.Scan(&first); err != nil {
		t.Fatal(err)
	}
	if rows.Next() {
		t.Fatal("faulting row unexpectedly succeeded")
	}
	var direct *pgconn.PgError
	if !errors.As(rows.Err(), &direct) || direct.Code != "40001" {
		t.Fatalf("expected deferred PostgreSQL serialization error: %v", rows.Err())
	}
	rows.Close()
	if err := probe.Rollback(); err != nil {
		t.Fatal(err)
	}

	u, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer u.Rollback()
	if _, err := u.(*unitOfWork).tx.ExecContext(ctx, "SET LOCAL enable_seqscan=off; SET LOCAL enable_sort=off"); err != nil {
		t.Fatal(err)
	}
	_, err = loadActorLinks(ctx, u.(*unitOfWork).tx, outcome.Ref(), "owner")
	code, ok := domain.ErrorCodeOf(err)
	var original *pgconn.PgError
	if !ok || code != domain.ErrorCodeTransactionConflict || !errors.As(err, &original) || original.Code != "40001" {
		t.Fatalf("streamed serialization error lost retryable code/cause: %v", err)
	}
}
