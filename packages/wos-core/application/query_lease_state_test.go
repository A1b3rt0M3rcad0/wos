package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/storage/memory"
)

func TestOutcomeStateProjectsExpiredLeaseWithoutMutatingWorkItem(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	claimedAt := time.Date(2026, 10, 2, 22, 0, 0, 0, time.UTC)
	ids := &sequenceIDs{next: 1000}
	service, err := application.NewService(store, fixedClock{now: claimedAt}, ids)
	if err != nil {
		t.Fatal(err)
	}
	cc := commandContext()
	namespaceID := id("0199f230-0000-7000-8000-000000000001")

	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Continuity state",
		DesiredState: "expired leases are visible",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value
	if _, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner:            outcome.Ref(),
		ExpectedVersion:  outcome.Version,
		Title:            "verified",
		Required:         true,
		VerificationMode: domain.VerificationModeAttestation,
	}); err != nil {
		t.Fatal(err)
	}
	outcome.Version++
	activated, err := service.ActivateOutcome(ctx, cc, application.ActivateOutcomeCommand{
		Scope:           outcome.Scope(),
		ExpectedVersion: outcome.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome = activated.Value

	createdWork, err := service.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{
		Scope:     outcome.Scope(),
		Title:     "expiring work",
		Priority:  domain.PriorityNormal,
		Lifecycle: domain.WorkItemLifecycleTodo,
	})
	if err != nil {
		t.Fatal(err)
	}
	work := createdWork.Value
	claimed, err := service.ClaimWorkItem(ctx, cc, application.ClaimWorkItemCommand{
		Scope:           outcome.Scope(),
		WorkItemID:      work.ID,
		ExpectedVersion: work.Version,
		TTL:             domain.MinLeaseTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	work = claimed.Value
	expiresAt := work.CurrentLease.ExpiresAt

	readService, err := application.NewService(
		store,
		fixedClock{now: expiresAt},
		&sequenceIDs{next: 2000},
	)
	if err != nil {
		t.Fatal(err)
	}
	state, err := readService.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if !state.EvaluatedAt.Equal(expiresAt) {
		t.Fatalf("evaluated_at = %v, want %v", state.EvaluatedAt, expiresAt)
	}
	if len(state.WorkItemOperationalStates) != 1 {
		t.Fatalf("operational states = %d, want 1", len(state.WorkItemOperationalStates))
	}
	projected := state.WorkItemOperationalStates[0]
	if projected.Ref.ID != work.ID {
		t.Fatalf("projected work id = %s, want %s", projected.Ref.ID, work.ID)
	}
	if projected.LeaseStatus != domain.LeaseStatusExpired {
		t.Fatalf("lease status = %q, want expired", projected.LeaseStatus)
	}
	if projected.DisplayState != domain.WorkItemDisplayStateAttentionNeeded {
		t.Fatalf("display state = %q, want attention_needed", projected.DisplayState)
	}

	persisted, err := readService.GetWorkItem(ctx, outcome.Scope(), work.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Value.Lifecycle != domain.WorkItemLifecycleInProgress {
		t.Fatalf("projection mutated lifecycle = %q, want in_progress", persisted.Value.Lifecycle)
	}
	if persisted.Value.CurrentLease == nil || persisted.Value.CurrentLease.FencingToken != work.CurrentLease.FencingToken {
		t.Fatalf("projection mutated lease: %#v", persisted.Value.CurrentLease)
	}
}
