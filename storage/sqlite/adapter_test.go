package sqlite

import (
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func testID(value string) domain.ID {
	return domain.MustParseID(value)
}

func openTestStore(t *testing.T, path string) *Store {
	t.Helper()
	store, err := Open(path, Options{
		BusyTimeout:   500 * time.Millisecond,
		MigrateOnOpen: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
	return store
}

func TestSQLiteConnectionProfileAndMigrations(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "wos.db"))

	settings, err := store.ConnectionSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if !settings.ForeignKeys {
		t.Fatal("foreign_keys must be enabled")
	}
	if settings.JournalMode != "wal" {
		t.Fatalf("journal mode = %q, want wal", settings.JournalMode)
	}
	if settings.BusyTimeout != 500*time.Millisecond {
		t.Fatalf("busy timeout = %s, want 500ms", settings.BusyTimeout)
	}

	version, err := store.SchemaVersion(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if version != 1 {
		t.Fatalf("schema version = %d, want 1", version)
	}

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("idempotent migration failed: %v", err)
	}
}

func TestSQLiteExpectedVersionRejectsStaleMutation(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "wos.db")
	store := openTestStore(t, path)
	service, err := application.NewService(
		store,
		sqliteFixedClock{now: time.Date(2026, 10, 2, 4, 0, 0, 0, time.UTC)},
		&sqliteSequenceIDs{prefix: "0199e940", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}

	cc := sqliteCommandContext("0199e941-0000-7000-8000-000000000001", "")
	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  testID("0199e941-0000-7000-8000-000000000010"),
		Title:        "Optimistic lock",
		DesiredState: "Only one stale write can succeed",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}

	ownerA := domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "local", ID: "owner-a"}
	ownerB := domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "local", ID: "owner-b"}
	updated, err := service.SetOutcomeOwners(
		ctx,
		sqliteCommandContext("0199e941-0000-7000-8000-000000000002", ""),
		application.SetOutcomeOwnersCommand{
			Scope:           created.Value.Scope(),
			ExpectedVersion: created.Value.Version,
			OwnerRefs:       []domain.ActorRef{ownerA},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Value.Version != created.Value.Version+1 {
		t.Fatalf("updated version = %d, want %d", updated.Value.Version, created.Value.Version+1)
	}

	_, err = service.SetOutcomeOwners(
		ctx,
		sqliteCommandContext("0199e941-0000-7000-8000-000000000003", ""),
		application.SetOutcomeOwnersCommand{
			Scope:           created.Value.Scope(),
			ExpectedVersion: created.Value.Version,
			OwnerRefs:       []domain.ActorRef{ownerB},
		},
	)
	if err == nil {
		t.Fatal("stale expected_version unexpectedly succeeded")
	}
	code, ok := domain.ErrorCodeOf(err)
	if !ok || code != domain.ErrorCodeVersionConflict {
		t.Fatalf("error code = %q, want version_conflict: %v", code, err)
	}
}

func TestSQLiteCompoundRollbackDiscardsStateEventsAndIdempotency(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "wos.db"))

	namespaceID := testID("0199e942-0000-7000-8000-000000000001")
	outcomeID := testID("0199e942-0000-7000-8000-000000000010")
	commandID := testID("0199e942-0000-7000-8000-000000000020")
	eventID := testID("0199e942-0000-7000-8000-000000000030")
	now := time.Date(2026, 10, 2, 4, 10, 0, 0, time.UTC)
	actor := domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "local", ID: "rollback-human"}

	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	identity := domain.IdempotencyIdentity{
		NamespaceID:    namespaceID,
		PrincipalID:    "rollback-human",
		CommandName:    "CreateOutcome",
		IdempotencyKey: "sqlite-rollback-key-0001",
	}
	reservation, err := tx.Idempotency().Reserve(ctx, identity, "fingerprint-rollback")
	if err != nil {
		t.Fatal(err)
	}
	coord, err := tx.Coordination().LockOutcome(ctx, domain.Scope{
		NamespaceID: namespaceID,
		OutcomeID:   outcomeID,
	})
	if err != nil {
		t.Fatal(err)
	}
	if coord.Revision != 0 {
		t.Fatalf("initial revision = %d, want 0", coord.Revision)
	}
	outcome, err := domain.NewOutcome(
		outcomeID,
		namespaceID,
		"Rollback",
		"",
		"Nothing survives rollback",
		domain.PriorityNormal,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Outcomes().Insert(ctx, outcome); err != nil {
		t.Fatal(err)
	}
	revision, err := tx.Coordination().AdvanceOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	after := outcome.Version
	event := domain.DomainEvent{
		EventID:               eventID,
		EventType:             "outcome.created",
		SchemaVersion:         domain.DomainEventSchemaVersion,
		NamespaceID:           namespaceID,
		OutcomeID:             outcomeID,
		OutcomeRevision:       revision,
		EventIndex:            0,
		AggregateRef:          outcome.Ref(),
		AggregateVersionAfter: &after,
		PrincipalID:           "rollback-human",
		Actor:                 actor,
		RecordedAt:            now,
		CommandID:             commandID,
		Payload:               json.RawMessage(`{"title":"Rollback"}`),
	}
	if err := tx.Events().Append(ctx, []domain.DomainEvent{event}); err != nil {
		t.Fatal(err)
	}
	if err := tx.Idempotency().Complete(ctx, reservation, domain.StoredCommandResult{
		CommandID:       commandID,
		OutcomeRevision: revision,
		ResponseJSON:    json.RawMessage(`{"ok":true}`),
	}); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	for _, table := range []string{"outcomes", "domain_events", "idempotency_records"} {
		var count int
		if err := store.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("%s count after rollback = %d, want 0", table, count)
		}
	}
}

func TestSQLiteCrossScopeForeignKeyIsRejected(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "wos.db"))
	service, err := application.NewService(
		store,
		sqliteFixedClock{now: time.Date(2026, 10, 2, 4, 20, 0, 0, time.UTC)},
		&sqliteSequenceIDs{prefix: "0199e943", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	namespaceID := testID("0199e943-0000-7000-8000-000000000001")
	a, err := service.CreateOutcome(ctx, sqliteCommandContext(
		"0199e943-0000-7000-8000-000000000101", "",
	), application.CreateOutcomeCommand{
		NamespaceID: namespaceID, Title: "A", DesiredState: "A", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	b, err := service.CreateOutcome(ctx, sqliteCommandContext(
		"0199e943-0000-7000-8000-000000000102", "",
	), application.CreateOutcomeCommand{
		NamespaceID: namespaceID, Title: "B", DesiredState: "B", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	objective, err := service.CreateObjective(ctx, sqliteCommandContext(
		"0199e943-0000-7000-8000-000000000103", "",
	), application.CreateObjectiveCommand{
		Scope: a.Value.Scope(), Title: "A objective", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}

	workID := testID("0199e943-0000-7000-8000-000000000200")
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
INSERT INTO entity_refs (id, namespace_id, outcome_id, kind)
VALUES (?, ?, ?, 'work_item')`,
		workID.String(), namespaceID.String(), b.Value.ID.String(),
	); err != nil {
		t.Fatal(err)
	}
	_, err = tx.ExecContext(ctx, `
INSERT INTO work_items (
    id, namespace_id, outcome_id, kind, version, created_at, updated_at,
    title, description, priority, objective_id, lifecycle, result_summary
) VALUES (?, ?, ?, 'work_item', 1, ?, ?, 'cross scope', '', 'normal', ?, 'todo', '')`,
		workID.String(),
		namespaceID.String(),
		b.Value.ID.String(),
		encodeTime(time.Date(2026, 10, 2, 4, 20, 0, 0, time.UTC)),
		encodeTime(time.Date(2026, 10, 2, 4, 20, 0, 0, time.UTC)),
		objective.Value.ID.String(),
	)
	if err == nil {
		t.Fatal("cross-scope objective reference unexpectedly passed foreign-key validation")
	}
}

func TestSQLiteBackupAndRestore(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	source := filepath.Join(dir, "source.db")
	backup := filepath.Join(dir, "backup.db")
	restored := filepath.Join(dir, "restored.db")

	store := openTestStore(t, source)
	service, err := application.NewService(
		store,
		sqliteFixedClock{now: time.Date(2026, 10, 2, 4, 30, 0, 0, time.UTC)},
		&sqliteSequenceIDs{prefix: "0199e944", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	namespaceID := testID("0199e944-0000-7000-8000-000000000001")
	created, err := service.CreateOutcome(ctx, sqliteCommandContext(
		"0199e944-0000-7000-8000-000000000101", "",
	), application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Backup outcome",
		DesiredState: "Survive backup and restore",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Backup(ctx, backup); err != nil {
		t.Fatal(err)
	}
	if err := RestoreFile(backup, restored); err != nil {
		t.Fatal(err)
	}

	copyStore, err := Open(restored, Options{BusyTimeout: 500 * time.Millisecond, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	defer copyStore.Close()
	uow, err := copyStore.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := uow.Outcomes().Get(ctx, namespaceID, created.Value.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Title != "Backup outcome" {
		t.Fatalf("restored title = %q", persisted.Title)
	}
	if err := uow.Rollback(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLiteWriterContentionHonorsContext(t *testing.T) {
	path := filepath.Join(t.TempDir(), "wos.db")
	first := openTestStore(t, path)
	second, err := Open(path, Options{BusyTimeout: 2 * time.Second, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()

	ctx := context.Background()
	firstTx, err := first.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer firstTx.Rollback()

	blockedCtx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
	defer cancel()
	start := time.Now()
	secondTx, err := second.Begin(blockedCtx)
	if err == nil {
		_ = secondTx.Rollback()
		t.Fatal("second writer unexpectedly acquired an immediate transaction")
	}
	if time.Since(start) > time.Second {
		t.Fatalf("writer contention ignored context deadline: %s", time.Since(start))
	}
}

func TestSQLiteCancelledTransactionStopsRepositoryWork(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "wos.db"))
	ctx, cancel := context.WithCancel(context.Background())
	uow, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	cancel()

	_, err = uow.Coordination().LockOutcome(ctx, domain.Scope{
		NamespaceID: testID("0199e945-0000-7000-8000-000000000001"),
		OutcomeID:   testID("0199e945-0000-7000-8000-000000000010"),
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("LockOutcome error = %v, want context.Canceled", err)
	}
	if err := uow.Rollback(); err != nil {
		t.Fatal(err)
	}
}

func TestSQLiteClaimByNewPrincipalCreatesLeasePrincipalBeforeSave(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "wos.db"))
	service, err := application.NewService(
		store,
		sqliteFixedClock{now: time.Date(2026, 10, 2, 4, 40, 0, 0, time.UTC)},
		&sqliteSequenceIDs{prefix: "0199e946", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	namespaceID := testID("0199e946-0000-7000-8000-000000000001")
	created, err := service.CreateOutcome(ctx, sqliteCommandContext(
		"0199e946-0000-7000-8000-000000000101", "",
	), application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Principal FK",
		DesiredState: "A new principal can claim persisted work",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	work, err := service.CreateWorkItem(ctx, sqliteCommandContext(
		"0199e946-0000-7000-8000-000000000102", "",
	), application.CreateWorkItemCommand{
		Scope:     created.Value.Scope(),
		Title:     "Claim me",
		Priority:  domain.PriorityNormal,
		Lifecycle: domain.WorkItemLifecycleTodo,
	})
	if err != nil {
		t.Fatal(err)
	}

	claimContext := sqliteCommandContext(
		"0199e946-0000-7000-8000-000000000103", "",
	)
	claimContext.PrincipalID = "sqlite-new-principal"
	claimContext.Actor.ID = "sqlite-new-principal"

	claimed, err := service.ClaimWorkItem(ctx, claimContext, application.ClaimWorkItemCommand{
		Scope:           created.Value.Scope(),
		WorkItemID:      work.Value.ID,
		ExpectedVersion: work.Value.Version,
		TTL:             domain.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if claimed.Value.CurrentLease == nil {
		t.Fatal("claim did not persist a lease")
	}
	if claimed.Value.CurrentLease.PrincipalID != "sqlite-new-principal" {
		t.Fatalf("lease principal = %q", claimed.Value.CurrentLease.PrincipalID)
	}

	var count int
	if err := store.db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM principals WHERE id = ?",
		"sqlite-new-principal",
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("new lease principal count = %d, want 1", count)
	}
}
