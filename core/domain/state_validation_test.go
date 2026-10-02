package domain_test

import (
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func TestAggregateValidationRejectsInvalidManualStates(t *testing.T) {
	now := time.Date(2026, 10, 1, 23, 0, 0, 0, time.UTC)
	outcome, err := domain.NewOutcome(
		domain.MustParseID("0199e500-0000-7000-8000-000000000010"),
		domain.MustParseID("0199e500-0000-7000-8000-000000000001"),
		"Validate",
		"",
		"State",
		domain.PriorityNormal,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}

	invalidOutcome := outcome
	invalidOutcome.Lifecycle = domain.OutcomeLifecycle("unknown")
	if err := invalidOutcome.Validate(); err == nil {
		t.Fatal("Outcome.Validate() unexpectedly accepted invalid lifecycle")
	}

	terminalWithoutConclusion := outcome
	terminalWithoutConclusion.Lifecycle = domain.OutcomeLifecycleFailed
	if err := terminalWithoutConclusion.Validate(); err == nil {
		t.Fatal("Outcome.Validate() unexpectedly accepted terminal state without conclusion")
	}

	scope := outcome.Scope()
	work, err := domain.NewWorkItem(
		domain.MustParseID("0199e500-0000-7000-8000-000000000020"),
		scope,
		"Work",
		"",
		domain.PriorityNormal,
		domain.WorkItemLifecycleTodo,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	work.Lifecycle = domain.WorkItemLifecycleInProgress
	if err := work.Validate(); err == nil {
		t.Fatal("WorkItem.Validate() unexpectedly accepted in_progress without lease")
	}
}

func TestWorkLeaseValidation(t *testing.T) {
	now := time.Date(2026, 10, 1, 23, 0, 0, 0, time.UTC)
	lease := domain.WorkLease{
		ClaimID:      domain.MustParseID("0199e500-0000-7000-8000-000000000030"),
		PrincipalID:  "principal-1",
		Actor:        actor(),
		FencingToken: 1,
		AcquiredAt:   now,
		ExpiresAt:    now.Add(domain.DefaultLeaseTTL),
	}
	if err := lease.Validate(); err != nil {
		t.Fatalf("WorkLease.Validate() error = %v", err)
	}

	lease.ExpiresAt = lease.AcquiredAt
	if err := lease.Validate(); err == nil {
		t.Fatal("WorkLease.Validate() unexpectedly accepted non-positive lifetime")
	}
}
