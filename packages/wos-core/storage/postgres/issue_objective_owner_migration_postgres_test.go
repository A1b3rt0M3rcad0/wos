package postgres

import (
	"context"
	"testing"
)

func TestPostgresIssueObjectiveOwnerMigrationSurvivesCleanRestore(t *testing.T) {
	requirePostgres(t)
	ctx := context.Background()
	store, required, scope, issue, ownerID := openIssueObjectiveSchema35(t)
	if err := store.applyMigration(ctx, required); err != nil {
		t.Fatalf("apply Issue owner migration before restore: %v", err)
	}
	restored := reopenIntegrationFixture(t, store, store.Path())
	var title, persistedOwner string
	if err := restored.db.QueryRowContext(ctx, `SELECT title, objective_id FROM issues WHERE namespace_id = $1 AND outcome_id = $2 AND id = $3`,
		scope.NamespaceID.String(), scope.OutcomeID.String(), issue.ID.String()).Scan(&title, &persistedOwner); err != nil {
		t.Fatal(err)
	}
	if title != issue.Title || persistedOwner != ownerID {
		t.Fatalf("Issue after clean restore: title=%q owner=%q; want title=%q owner=%q", title, persistedOwner, issue.Title, ownerID)
	}
}
