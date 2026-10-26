package sqlite

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func openStoreAtSchema41(t *testing.T) (*Store, string) {
	t.Helper()
	store, path := openStoreAtSchema33(t)
	migrations, err := loadMigrations()
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range migrations {
		if item.version > 41 {
			break
		}
		if item.version <= 33 {
			continue
		}
		if err := store.applyMigration(context.Background(), item); err != nil {
			t.Fatalf("apply migration %s: %v", item.name, err)
		}
	}
	if version, err := store.SchemaVersion(context.Background()); err != nil || version != 41 {
		t.Fatalf("fixture schema version=%d err=%v; want 41", version, err)
	}
	return store, path
}

func seedHistoricalLeaseAtSchema41(t *testing.T, store *Store) (domain.Scope, domain.ID) {
	t.Helper()
	ctx := context.Background()
	namespaceID := testID("0199fa42-0000-7000-8000-000000000001")
	outcomeID := testID("0199fa42-0000-7000-8000-000000000002")
	objectiveID := testID("0199fa42-0000-7000-8000-000000000003")
	workID := testID("0199fa42-0000-7000-8000-000000000004")
	principalID := testID("0199fa42-0000-7000-8000-000000000005")
	scope := seedLegacyOutcomeObjective(t, store, namespaceID, outcomeID, objectiveID,
		"Lease binding migration", "Historical leases remain unattributed", "Legacy owner")
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	actor := sqliteCommandContext("0199fa42-0000-7000-8000-000000000006", "").Actor
	actorJSON, err := marshalJSON(actor)
	if err != nil {
		t.Fatal(err)
	}
	tx, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `INSERT INTO principals(id,display_name,kind,lifecycle,created_at) VALUES(?, 'Historical lease holder', 'human', 'active', ?)`, principalID.String(), encodeTime(now)); err != nil {
		t.Fatal(err)
	}
	if err := insertEntityRef(ctx, tx, domain.EntityRef{Scope: scope, Kind: domain.EntityKindWorkItem, ID: workID}); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.ExecContext(ctx, `INSERT INTO work_items(
		id,namespace_id,outcome_id,version,created_at,updated_at,title,description,priority,objective_id,lifecycle,
		last_fencing_token,lease_claim_id,lease_principal_id,lease_actor_json,lease_acquired_at,lease_expires_at
	) VALUES(?,?,?,1,?,?,?,?, 'normal', ?, 'in_progress', 7, ?, ?, ?, ?, ?)`,
		workID.String(), namespaceID.String(), outcomeID.String(), encodeTime(now), encodeTime(now),
		"Historic unbound lease", "No credential identity is present in this row.", objectiveID.String(),
		testID("0199fa42-0000-7000-8000-000000000007").String(), principalID.String(), actorJSON,
		encodeTime(now), encodeTime(now.Add(30*time.Second)),
	); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return scope, workID
}

func getWorkItemForLeaseTest(t *testing.T, store *Store, scope domain.Scope, workID domain.ID) domain.WorkItem {
	t.Helper()
	tx, err := store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	item, err := tx.WorkItems().Get(context.Background(), scope, workID)
	if err != nil {
		t.Fatal(err)
	}
	return item
}

func assertUnboundHistoricalLease(t *testing.T, item domain.WorkItem) {
	t.Helper()
	if item.CurrentLease == nil || item.CurrentLease.CredentialBindingVersion != 0 || !item.CurrentLease.HolderCredentialID.IsZero() || item.CurrentLease.SessionCredentialID != nil {
		t.Fatalf("historical lease was attributed during migration: %#v", item.CurrentLease)
	}
}

func TestLeaseCredentialBindingUpgradeRestartAndRestore(t *testing.T) {
	ctx := context.Background()
	legacy, _ := openStoreAtSchema41(t)
	scope, workID := seedHistoricalLeaseAtSchema41(t, legacy)
	backup := filepath.Join(t.TempDir(), "schema41-backup.db")
	restored := filepath.Join(t.TempDir(), "schema41-restored.db")
	if err := legacy.Backup(ctx, backup); err != nil {
		t.Fatal(err)
	}
	if err := RestoreFile(backup, restored); err != nil {
		t.Fatal(err)
	}
	upgraded, err := Open(restored, Options{MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = upgraded.Close() })
	if version, err := upgraded.SchemaVersion(ctx); err != nil || version != latestMigrationVersion(t) {
		t.Fatalf("upgraded schema version=%d err=%v; want %d", version, err, latestMigrationVersion(t))
	}
	legacyItem := getWorkItemForLeaseTest(t, upgraded, scope, workID)
	assertUnboundHistoricalLease(t, legacyItem)
	if legacyItem.CurrentLease.PrincipalID != testID("0199fa42-0000-7000-8000-000000000005").String() {
		t.Fatalf("legacy principal changed during upgrade: %q", legacyItem.CurrentLease.PrincipalID)
	}

	holderID := testID("0199fa42-0000-7000-8000-000000000008")
	sessionID := testID("0199fa42-0000-7000-8000-000000000009")
	actor := sqliteCommandContext("0199fa42-0000-7000-8000-000000000010", "").Actor
	item := legacyItem
	now := item.CurrentLease.ExpiresAt.Add(time.Second)
	if err := item.ReclaimForCredential(testID("0199fa42-0000-7000-8000-000000000011"), item.CurrentLease.PrincipalID, holderID, &sessionID, actor, 30*time.Second, now); err != nil {
		t.Fatal(err)
	}
	tx, err := upgraded.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.WorkItems().Save(ctx, item, legacyItem.Version); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := upgraded.Close(); err != nil {
		t.Fatal(err)
	}
	restarted, err := Open(restored, Options{MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = restarted.Close() })
	roundtrip := getWorkItemForLeaseTest(t, restarted, scope, workID)
	if roundtrip.CurrentLease.CredentialBindingVersion != domain.WorkLeaseCredentialBindingV1 || roundtrip.CurrentLease.HolderCredentialID != holderID || roundtrip.CurrentLease.SessionCredentialID == nil || *roundtrip.CurrentLease.SessionCredentialID != sessionID {
		t.Fatalf("bound lease was not preserved across restart: %#v", roundtrip.CurrentLease)
	}

	boundBackup := filepath.Join(t.TempDir(), "schema42-backup.db")
	boundRestore := filepath.Join(t.TempDir(), "schema42-restored.db")
	if err := restarted.Backup(ctx, boundBackup); err != nil {
		t.Fatal(err)
	}
	if err := RestoreFile(boundBackup, boundRestore); err != nil {
		t.Fatal(err)
	}
	restoredStore, err := Open(boundRestore, Options{MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = restoredStore.Close() })
	restoredItem := getWorkItemForLeaseTest(t, restoredStore, scope, workID)
	if restoredItem.CurrentLease.CredentialBindingVersion != domain.WorkLeaseCredentialBindingV1 || restoredItem.CurrentLease.HolderCredentialID != holderID || restoredItem.CurrentLease.SessionCredentialID == nil || *restoredItem.CurrentLease.SessionCredentialID != sessionID {
		t.Fatalf("bound lease was not preserved across backup and restore: %#v", restoredItem.CurrentLease)
	}

	var version int64
	var rawHolder, rawSession sql.NullString
	if err := restoredStore.db.QueryRowContext(ctx, `SELECT lease_credential_binding_version,lease_holder_credential_id,lease_session_credential_id FROM work_items WHERE id=?`, workID.String()).Scan(&version, &rawHolder, &rawSession); err != nil {
		t.Fatal(err)
	}
	if version != 1 || !rawHolder.Valid || rawHolder.String != holderID.String() || !rawSession.Valid || rawSession.String != sessionID.String() {
		t.Fatalf("persisted binding tuple = (%d,%v,%v)", version, rawHolder, rawSession)
	}

	resetAt := restoredItem.CurrentLease.ExpiresAt.Add(time.Second)
	resetTx, err := restoredStore.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	protocolRepo := resetTx.(ports.WorkProtocolUnitOfWork).WorkProtocol()
	if count, err := protocolRepo.ValidLegacyLeases(ctx, scope.NamespaceID, resetAt); err != nil || count != 0 {
		resetTx.Rollback()
		t.Fatalf("expired bound lease count=%d err=%v; want 0", count, err)
	}
	if err := protocolRepo.EnableContracts(ctx, scope.NamespaceID, resetAt); err != nil {
		resetTx.Rollback()
		t.Fatal(err)
	}
	if err := resetTx.Commit(); err != nil {
		t.Fatal(err)
	}
	cleared := getWorkItemForLeaseTest(t, restoredStore, scope, workID)
	if !cleared.ContractsEnabled || cleared.CurrentLease != nil || cleared.Lifecycle != domain.WorkItemLifecycleTodo || cleared.Version != restoredItem.Version+1 {
		t.Fatalf("protocol reset did not clear lease: contracts=%v lease=%#v", cleared.ContractsEnabled, cleared.CurrentLease)
	}
	if err := restoredStore.db.QueryRowContext(ctx, `SELECT lease_credential_binding_version,lease_holder_credential_id,lease_session_credential_id FROM work_items WHERE id=?`, workID.String()).Scan(&version, &rawHolder, &rawSession); err != nil {
		t.Fatal(err)
	}
	if version != 0 || rawHolder.Valid || rawSession.Valid {
		t.Fatalf("protocol reset retained binding tuple = (%d,%v,%v)", version, rawHolder, rawSession)
	}
}
