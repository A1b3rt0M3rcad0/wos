package sqlite

import (
	"context"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func seedLegacyOutcomeObjective(t *testing.T, store *Store, namespaceID, outcomeID, objectiveID domain.ID, outcomeTitle, outcomeDesired, objectiveTitle string) domain.Scope {
	t.Helper()
	ctx := context.Background()
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	now := encodeTime(time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC))
	if _, err := tx.ExecContext(ctx, `INSERT INTO namespaces(id,name,lifecycle,version,created_at,updated_at) VALUES(?,?,'active',1,?,?)`, namespaceID.String(), "migration-"+outcomeID.String(), now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO outcome_coordination(namespace_id,outcome_id,state_revision) VALUES(?,?,0)`, namespaceID.String(), outcomeID.String()); err != nil {
		t.Fatal(err)
	}
	if err := insertEntityRef(ctx, tx, domain.EntityRef{Scope: domain.Scope{NamespaceID: namespaceID, OutcomeID: outcomeID}, Kind: domain.EntityKindOutcome, ID: outcomeID}); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO outcomes(id,namespace_id,outcome_id,version,created_at,updated_at,title,description,priority,desired_state,lifecycle) VALUES(?,?,?,1,?,?,?,'', 'normal',?,'draft')`, outcomeID.String(), namespaceID.String(), outcomeID.String(), now, now, outcomeTitle, outcomeDesired); err != nil {
		t.Fatal(err)
	}
	scope := domain.Scope{NamespaceID: namespaceID, OutcomeID: outcomeID}
	if err := insertEntityRef(ctx, tx, domain.EntityRef{Scope: scope, Kind: domain.EntityKindObjective, ID: objectiveID}); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO objectives(id,namespace_id,outcome_id,version,created_at,updated_at,title,description,priority,lifecycle,required_for_outcome) VALUES(?,?,?,1,?,?,?,'','normal','planned',0)`, objectiveID.String(), namespaceID.String(), outcomeID.String(), now, now, objectiveTitle); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return scope
}
