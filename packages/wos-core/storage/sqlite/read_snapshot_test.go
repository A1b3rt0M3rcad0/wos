package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// A held snapshot must neither block an ordinary writer nor mix that writer's
// new aggregate with the old revision. This exercises the actual pooled store.
func TestReadSnapshotCoexistsWithWriterAndKeepsRevision(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	store := openTestStore(t, filepath.Join(t.TempDir(), "coherent-read.db"))
	service, err := a.NewService(store, sqliteFixedClock{time.Now().UTC()}, &sqliteSequenceIDs{prefix: "0199f381", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	cc := sqliteCommandContext("0199f381-0000-7000-8000-000000000101", "")
	created, err := service.CreateOutcome(ctx, cc, a.CreateOutcomeCommand{TenantID: testID("0199f381-0000-7000-8000-000000000102"), Title: "Before", DesiredState: "Stable snapshot", Priority: d.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	scope := created.Value.Scope()
	reader, err := store.BeginReadSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Rollback()
	before, err := reader.Outcomes().Get(ctx, scope.TenantID, scope.OutcomeID)
	if err != nil {
		t.Fatal(err)
	}
	revision, err := reader.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	title := "After"
	updated, err := service.UpdateOutcome(ctx, cc, a.UpdateOutcomeCommand{Scope: scope, ExpectedVersion: before.Version, Title: &title})
	if err != nil {
		t.Fatalf("reader prevented independent writer: %v", err)
	}
	still, err := reader.Outcomes().Get(ctx, scope.TenantID, scope.OutcomeID)
	if err != nil {
		t.Fatal(err)
	}
	stillRevision, err := reader.Coordination().LockOutcome(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if still.Title != before.Title || still.Version != before.Version || stillRevision.Revision != revision.Revision {
		t.Fatal("snapshot mixed revisions", before, still, revision, stillRevision)
	}
	if err := reader.Commit(); err == nil {
		t.Fatal("read snapshot accepted commit")
	}
	fresh, err := service.GetOutcome(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if fresh.Value.Title != title || fresh.OutcomeRevision != updated.OutcomeRevision || fresh.OutcomeRevision <= revision.Revision {
		t.Fatal("fresh reader did not observe committed state", fresh)
	}
}

func TestReadSnapshotMissingScopeDoesNotCreateCoordination(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "missing-read.db"))
	u, err := store.BeginReadSnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	scope := d.Scope{TenantID: testID("0199f382-0000-7000-8000-000000000001"), OutcomeID: testID("0199f382-0000-7000-8000-000000000002")}
	_, err = u.Coordination().LockOutcome(ctx, scope)
	if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodeNotFound {
		t.Fatal("missing scope was not rejected", err)
	}
	u.Rollback()
	var count int
	if err := store.db.QueryRowContext(ctx, `SELECT count(*) FROM tenants`).Scan(&count); err != nil || count != 0 {
		t.Fatal("read created Tenant state", count, err)
	}
}
