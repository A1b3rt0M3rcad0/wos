package sqlite

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
	if _, err := store.db.ExecContext(ctx, "ANALYZE"); err != nil {
		t.Fatal(err)
	}
	query := `SELECT COUNT(*) FROM criterion_assessments a WHERE a.namespace_id=? AND a.outcome_id=? AND a.criterion_id IN(SELECT c.id FROM success_criteria c WHERE c.namespace_id=? AND c.outcome_id=? AND c.owner_id=? AND c.owner_kind=?)`
	args := []any{owner.NamespaceID.String(), owner.OutcomeID.String(), owner.NamespaceID.String(), owner.OutcomeID.String(), owner.ID.String(), string(owner.Kind)}
	plan := func(q interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	}) string {
		rows, err := q.QueryContext(ctx, "EXPLAIN QUERY PLAN "+query, args...)
		if err != nil {
			t.Fatal(err)
		}
		defer rows.Close()
		var lines []string
		for rows.Next() {
			var id, parent, unused int
			var detail string
			if err = rows.Scan(&id, &parent, &unused, &detail); err != nil {
				t.Fatal(err)
			}
			lines = append(lines, detail)
		}
		if err = rows.Err(); err != nil {
			t.Fatal(err)
		}
		return strings.Join(lines, "\n")
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
