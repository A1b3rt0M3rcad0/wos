package memory_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

func TestEventAndIdempotencyRollbackAreAtomic(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	namespaceID := domain.MustParseID("0199e700-0000-7000-8000-000000000001")
	outcomeID := domain.MustParseID("0199e700-0000-7000-8000-000000000010")
	eventID := domain.MustParseID("0199e700-0000-7000-8000-000000000020")
	commandID := domain.MustParseID("0199e700-0000-7000-8000-000000000030")
	scope := domain.Scope{NamespaceID: namespaceID, OutcomeID: outcomeID}
	ref := domain.EntityRef{Scope: scope, Kind: domain.EntityKindOutcome, ID: outcomeID}
	v1 := domain.InitialVersion

	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	identity := domain.IdempotencyIdentity{
		NamespaceID:    namespaceID,
		PrincipalID:    "principal-1",
		CommandName:    "CreateOutcome",
		IdempotencyKey: "rollback-key-0001",
	}
	reservation, err := tx.Idempotency().Reserve(ctx, identity, "fingerprint-a")
	if err != nil {
		t.Fatal(err)
	}
	event := domain.DomainEvent{
		EventID:               eventID,
		EventType:             "outcome.created",
		SchemaVersion:         domain.DomainEventSchemaVersion,
		NamespaceID:           namespaceID,
		OutcomeID:             outcomeID,
		OutcomeRevision:       1,
		EventIndex:            0,
		AggregateRef:          ref,
		AggregateVersionAfter: &v1,
		PrincipalID:           "principal-1",
		Actor:                 domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "local", ID: "human-1"},
		RecordedAt:            time.Date(2026, 10, 2, 2, 0, 0, 0, time.UTC),
		CommandID:             commandID,
		Payload:               json.RawMessage(`{"title":"x"}`),
	}
	if err := tx.Events().Append(ctx, []domain.DomainEvent{event}); err != nil {
		t.Fatal(err)
	}
	if err := tx.Idempotency().Complete(ctx, reservation, domain.StoredCommandResult{
		CommandID:       commandID,
		OutcomeRevision: 1,
		ResponseJSON:    json.RawMessage(`{"ok":true}`),
	}); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	if got := store.SnapshotDomainEvents(scope); len(got) != 0 {
		t.Fatalf("events survived rollback: %d", len(got))
	}
	if got := store.IdempotencyRecordCount(); got != 0 {
		t.Fatalf("idempotency reservation survived rollback: %d", got)
	}
}

func TestIdempotencyReplayAndConflict(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	namespaceID := domain.MustParseID("0199e700-0000-7000-8000-000000000001")
	commandID := domain.MustParseID("0199e700-0000-7000-8000-000000000030")
	identity := domain.IdempotencyIdentity{
		NamespaceID:    namespaceID,
		PrincipalID:    "principal-1",
		CommandName:    "CreateWorkItem",
		IdempotencyKey: "stable-replay-0001",
	}

	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	reservation, err := tx.Idempotency().Reserve(ctx, identity, "fingerprint-a")
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Idempotency().Complete(ctx, reservation, domain.StoredCommandResult{
		CommandID:       commandID,
		OutcomeRevision: 7,
		ResponseJSON:    json.RawMessage(`{"id":"original"}`),
	}); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	replayTx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := replayTx.Idempotency().Reserve(ctx, identity, "fingerprint-a")
	if err != nil {
		t.Fatal(err)
	}
	if !replay.IsReplay() || replay.Replay == nil || replay.Replay.CommandID != commandID {
		t.Fatalf("unexpected replay reservation: %#v", replay)
	}
	if err := replayTx.Rollback(); err != nil {
		t.Fatal(err)
	}

	conflictTx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer conflictTx.Rollback()
	if _, err := conflictTx.Idempotency().Reserve(ctx, identity, "fingerprint-b"); err == nil {
		t.Fatal("Reserve() unexpectedly accepted a different fingerprint")
	}
}
