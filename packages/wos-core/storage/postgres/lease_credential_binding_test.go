package postgres

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestWorkLeaseCredentialBindingSurvivesPostgresRestart(t *testing.T) {
	requirePostgres(t)
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "lease-binding")
	store := openTestStore(t, path)
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	item := contractStorageFixture(t, store, now)
	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := tx.WorkItems().Get(ctx, item.Scope, item.ID)
	if err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	loaded.ContractsEnabled = false
	holderID := testID("0199fa43-0000-7000-8000-000000000001")
	sessionID := testID("0199fa43-0000-7000-8000-000000000002")
	actor := sqliteCommandContext("0199fa43-0000-7000-8000-000000000003", "").Actor
	if err := loaded.ClaimForCredential(testID("0199fa43-0000-7000-8000-000000000004"), "lease-binding-principal", holderID, &sessionID, actor, 30*time.Second, now); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.WorkItems().Save(ctx, loaded, item.Version); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	store = reopenIntegrationFixture(t, store, path)
	read, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer read.Rollback()
	got, err := read.WorkItems().Get(ctx, item.Scope, item.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.CurrentLease == nil || got.CurrentLease.CredentialBindingVersion != domain.WorkLeaseCredentialBindingV1 || got.CurrentLease.HolderCredentialID != holderID || got.CurrentLease.SessionCredentialID == nil || *got.CurrentLease.SessionCredentialID != sessionID {
		t.Fatalf("bound lease was not preserved across PostgreSQL restart: %#v", got.CurrentLease)
	}
}
