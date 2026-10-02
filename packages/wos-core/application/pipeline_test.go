package application_test

import (
	"context"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func wave3Context(commandID, key string) domain.CommandContext {
	return domain.CommandContext{
		PrincipalID: "human-wave3",
		Actor: domain.ActorRef{
			Kind:     domain.ActorKindHuman,
			Provider: "local",
			ID:       "human-wave3",
		},
		CommandID:      id(commandID),
		IdempotencyKey: key,
		CorrelationID:  "wave3-test",
	}
}

func TestCreateWorkItemReplayReturnsOriginalResultWithoutNewEvent(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	namespaceID := id("0199e200-0000-7000-8000-000000000001")

	createdOutcome, err := service.CreateOutcome(ctx, wave3Context(
		"0199e710-0000-7000-8000-000000000001",
		"",
	), application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Replay outcome",
		DesiredState: "State",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := createdOutcome.Value

	cmd := application.CreateWorkItemCommand{
		Scope:     outcome.Scope(),
		Title:     "Idempotent work",
		Priority:  domain.PriorityNormal,
		Lifecycle: domain.WorkItemLifecycleTodo,
	}
	key := "create-work-replay-0001"
	first, err := service.CreateWorkItem(ctx, wave3Context(
		"0199e710-0000-7000-8000-000000000002",
		key,
	), cmd)
	if err != nil {
		t.Fatal(err)
	}
	before := store.SnapshotDomainEvents(outcome.Scope())

	replay, err := service.CreateWorkItem(ctx, wave3Context(
		"0199e710-0000-7000-8000-000000000003",
		key,
	), cmd)
	if err != nil {
		t.Fatal(err)
	}
	after := store.SnapshotDomainEvents(outcome.Scope())

	if !replay.IdempotentReplay {
		t.Fatal("replay result is not marked idempotent_replay")
	}
	if replay.Value.ID != first.Value.ID {
		t.Fatalf("replay id = %s, want original %s", replay.Value.ID, first.Value.ID)
	}
	if replay.CommandID != first.CommandID {
		t.Fatalf("replay command_id = %s, want original %s", replay.CommandID, first.CommandID)
	}
	if replay.OutcomeRevision != first.OutcomeRevision {
		t.Fatalf("replay revision = %d, want %d", replay.OutcomeRevision, first.OutcomeRevision)
	}
	if len(after) != len(before) {
		t.Fatalf("replay emitted duplicate events: before=%d after=%d", len(before), len(after))
	}
}

func TestSameIdempotencyKeyWithDifferentPayloadConflicts(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	namespaceID := id("0199e200-0000-7000-8000-000000000001")
	outcomeResult, err := service.CreateOutcome(ctx, wave3Context(
		"0199e711-0000-7000-8000-000000000001",
		"",
	), application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Conflict outcome",
		DesiredState: "State",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	scope := outcomeResult.Value.Scope()
	key := "create-work-conflict-01"

	firstCmd := application.CreateWorkItemCommand{
		Scope: scope, Title: "First", Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleTodo,
	}
	if _, err := service.CreateWorkItem(ctx, wave3Context(
		"0199e711-0000-7000-8000-000000000002",
		key,
	), firstCmd); err != nil {
		t.Fatal(err)
	}
	before := store.SnapshotDomainEvents(scope)

	secondCmd := firstCmd
	secondCmd.Title = "Different payload"
	_, err = service.CreateWorkItem(ctx, wave3Context(
		"0199e711-0000-7000-8000-000000000003",
		key,
	), secondCmd)
	if err == nil {
		t.Fatal("CreateWorkItem() unexpectedly accepted a different payload for the same idempotency key")
	}
	code, ok := domain.ErrorCodeOf(err)
	if !ok || code != domain.ErrorCodeIdempotencyConflict {
		t.Fatalf("error code = %q, want idempotency_conflict: %v", code, err)
	}
	after := store.SnapshotDomainEvents(scope)
	if len(after) != len(before) {
		t.Fatalf("conflicting replay mutated event log: before=%d after=%d", len(before), len(after))
	}
}

func TestExplicitOwnerNoOpDoesNotAdvanceRevisionOrEmitEvent(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	namespaceID := id("0199e200-0000-7000-8000-000000000001")
	outcomeResult, err := service.CreateOutcome(ctx, wave3Context(
		"0199e712-0000-7000-8000-000000000001",
		"",
	), application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "No-op outcome",
		DesiredState: "State",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := outcomeResult.Value
	owners := []domain.ActorRef{{Kind: domain.ActorKindHuman, Provider: "local", ID: "owner-noop"}}

	changed, err := service.SetOutcomeOwners(ctx, wave3Context(
		"0199e712-0000-7000-8000-000000000002",
		"",
	), application.SetOutcomeOwnersCommand{
		Scope: outcome.Scope(), ExpectedVersion: outcome.Version, OwnerRefs: owners,
	})
	if err != nil {
		t.Fatal(err)
	}
	beforeEvents := store.SnapshotDomainEvents(outcome.Scope())

	noOp, err := service.SetOutcomeOwners(ctx, wave3Context(
		"0199e712-0000-7000-8000-000000000003",
		"",
	), application.SetOutcomeOwnersCommand{
		Scope: outcome.Scope(), ExpectedVersion: changed.Value.Version, OwnerRefs: owners,
	})
	if err != nil {
		t.Fatal(err)
	}
	afterEvents := store.SnapshotDomainEvents(outcome.Scope())

	if noOp.Value.Version != changed.Value.Version {
		t.Fatalf("no-op version = %d, want %d", noOp.Value.Version, changed.Value.Version)
	}
	if noOp.OutcomeRevision != changed.OutcomeRevision {
		t.Fatalf("no-op revision = %d, want %d", noOp.OutcomeRevision, changed.OutcomeRevision)
	}
	if len(afterEvents) != len(beforeEvents) {
		t.Fatalf("no-op emitted an event: before=%d after=%d", len(beforeEvents), len(afterEvents))
	}
}

func TestIdempotencyKeyValidation(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	namespaceID := id("0199e200-0000-7000-8000-000000000001")

	_, err := service.CreateOutcome(ctx, wave3Context(
		"0199e713-0000-7000-8000-000000000001",
		"short",
	), application.CreateOutcomeCommand{
		NamespaceID: namespaceID, Title: "Invalid key", DesiredState: "State", Priority: domain.PriorityNormal,
	})
	if err == nil {
		t.Fatal("CreateOutcome() unexpectedly accepted a short idempotency key")
	}
	code, ok := domain.ErrorCodeOf(err)
	if !ok || code != domain.ErrorCodeInvalidIdempotencyKey {
		t.Fatalf("error code = %q, want invalid_idempotency_key: %v", code, err)
	}
}
