package postgres

import (
	"context"
	"database/sql"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"strings"
	"testing"
)

func explainAssessmentHistoryPlan(t *testing.T, store *Store, owner d.EntityRef) {
	t.Helper()
	ctx := context.Background()
	if _, err := store.db.ExecContext(ctx, "VACUUM (ANALYZE) criterion_assessments, success_criteria"); err != nil {
		t.Fatal(err)
	}
	query := `SELECT COUNT(*) FROM criterion_assessments a WHERE a.tenant_id=$1 AND a.outcome_id=$2 AND a.criterion_id IN(SELECT c.id FROM success_criteria c WHERE c.tenant_id=$3 AND c.outcome_id=$4 AND c.owner_id=$5 AND c.owner_kind=$6)`
	args := []any{owner.TenantID.String(), owner.OutcomeID.String(), owner.TenantID.String(), owner.OutcomeID.String(), owner.ID.String(), string(owner.Kind)}
	plan := func(q interface {
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}) string {
		var raw string
		if err := q.QueryRowContext(ctx, "EXPLAIN (ANALYZE, BUFFERS, FORMAT JSON) "+query, args...).Scan(&raw); err != nil {
			t.Fatal(err)
		}
		return raw
	}
	after := plan(store.db)
	if !strings.Contains(after, "criterion_assessment_by_criterion") {
		t.Fatal("new scoped index unused", after)
	}
	var count int
	if err := store.db.QueryRowContext(ctx, query, args...).Scan(&count); err != nil || count != 128 {
		t.Fatal(count, err)
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, "DROP INDEX criterion_assessment_by_criterion"); err != nil {
		t.Fatal(err)
	}
	before := plan(tx)
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	t.Logf("4096 validated assessments / 32 owners; scoped count 128; before index:\n%s\nafter index:\n%s", before, after)
}
