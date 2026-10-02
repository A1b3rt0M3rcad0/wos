package domain_test

import (
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func TestWorkItemOperationalStateMarksExpiredLeaseAttentionNeeded(t *testing.T) {
	now := time.Date(2026, 10, 2, 19, 0, 0, 0, time.UTC)
	scope := domain.Scope{
		NamespaceID: domain.MustParseID("0199f220-0000-7000-8000-000000000001"),
		OutcomeID:   domain.MustParseID("0199f220-0000-7000-8000-000000000002"),
	}
	work, err := domain.NewWorkItem(
		domain.MustParseID("0199f220-0000-7000-8000-000000000003"),
		scope,
		"project lease state",
		"",
		domain.PriorityNormal,
		domain.WorkItemLifecycleTodo,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	claimID := domain.MustParseID("0199f220-0000-7000-8000-000000000004")
	actor := domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "worker"}
	if err := work.Claim(claimID, "principal-a", actor, domain.MinLeaseTTL, now); err != nil {
		t.Fatal(err)
	}

	outcome := domain.Outcome{Lifecycle: domain.OutcomeLifecycleActive}
	active := domain.ProjectWorkItemOperationalState(work, outcome, nil, nil, false, now.Add(10*time.Second))
	if active.LeaseStatus != domain.LeaseStatusActive || active.DisplayState != domain.WorkItemDisplayStateInProgress {
		t.Fatalf("active projection = %#v", active)
	}

	expiredAt := work.CurrentLease.ExpiresAt
	expired := domain.ProjectWorkItemOperationalState(work, outcome, nil, nil, false, expiredAt)
	if expired.LeaseStatus != domain.LeaseStatusExpired {
		t.Fatalf("lease status = %q, want expired", expired.LeaseStatus)
	}
	if expired.DisplayState != domain.WorkItemDisplayStateAttentionNeeded {
		t.Fatalf("display state = %q, want attention_needed", expired.DisplayState)
	}
	if !containsReadinessReason(expired.ReadinessReasons, domain.ReadinessReasonLeaseExpired) {
		t.Fatalf("lease_expired reason missing: %#v", expired.ReadinessReasons)
	}
}

func TestWorkItemOperationalStatePreservesBlockedPrecedence(t *testing.T) {
	now := time.Date(2026, 10, 2, 20, 0, 0, 0, time.UTC)
	scope := domain.Scope{
		NamespaceID: domain.MustParseID("0199f220-0000-7000-8000-000000000011"),
		OutcomeID:   domain.MustParseID("0199f220-0000-7000-8000-000000000012"),
	}
	work, err := domain.NewWorkItem(
		domain.MustParseID("0199f220-0000-7000-8000-000000000013"),
		scope,
		"blocked work",
		"",
		domain.PriorityNormal,
		domain.WorkItemLifecycleTodo,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := work.Claim(
		domain.MustParseID("0199f220-0000-7000-8000-000000000014"),
		"principal-a",
		domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "worker"},
		domain.MinLeaseTTL,
		now,
	); err != nil {
		t.Fatal(err)
	}

	outcome := domain.Outcome{Lifecycle: domain.OutcomeLifecycleActive}
	projected := domain.ProjectWorkItemOperationalState(work, outcome, nil, nil, true, work.CurrentLease.ExpiresAt)
	if projected.DisplayState != domain.WorkItemDisplayStateBlocked {
		t.Fatalf("display state = %q, want blocked", projected.DisplayState)
	}
	if !containsReadinessReason(projected.ReadinessReasons, domain.ReadinessReasonBlocked) ||
		!containsReadinessReason(projected.ReadinessReasons, domain.ReadinessReasonLeaseExpired) {
		t.Fatalf("projection lost simultaneous reasons: %#v", projected.ReadinessReasons)
	}
}

func TestTodoOperationalStateUsesDeterministicPrecedence(t *testing.T) {
	now := time.Date(2026, 10, 2, 21, 0, 0, 0, time.UTC)
	scope := domain.Scope{
		NamespaceID: domain.MustParseID("0199f220-0000-7000-8000-000000000021"),
		OutcomeID:   domain.MustParseID("0199f220-0000-7000-8000-000000000022"),
	}
	work, err := domain.NewWorkItem(
		domain.MustParseID("0199f220-0000-7000-8000-000000000023"),
		scope,
		"waiting work",
		"",
		domain.PriorityNormal,
		domain.WorkItemLifecycleTodo,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	notBefore := now.Add(time.Hour)
	work.NotBefore = &notBefore
	outcome := domain.Outcome{Lifecycle: domain.OutcomeLifecycleActive}
	dependencies := []domain.DependencyEvaluation{{
		Relation: domain.Relation{
			Lifecycle:    domain.RelationLifecycleActive,
			RelationType: domain.RelationTypeDependsOn,
			Strength:     domain.DependencyStrengthHard,
		},
		Satisfied: false,
	}}

	projected := domain.ProjectWorkItemOperationalState(work, outcome, nil, dependencies, false, now)
	if projected.DisplayState != domain.WorkItemDisplayStateWaitingDependencies {
		t.Fatalf("display state = %q, want waiting_dependencies", projected.DisplayState)
	}
	if !containsReadinessReason(projected.ReadinessReasons, domain.ReadinessReasonHardDependencyUnsatisfied) ||
		!containsReadinessReason(projected.ReadinessReasons, domain.ReadinessReasonNotBefore) {
		t.Fatalf("projection should preserve all readiness reasons: %#v", projected.ReadinessReasons)
	}
}

func containsReadinessReason(values []domain.ReadinessReason, target domain.ReadinessReason) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
