package domain_test

import (
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func TestDeclarativeOwnershipDoesNotAffectLeasePrincipal(t *testing.T) {
	now := time.Date(2026, 10, 1, 22, 0, 0, 0, time.UTC)
	scope := domain.Scope{
		NamespaceID: domain.MustParseID("0199e300-0000-7000-8000-000000000001"),
		OutcomeID:   domain.MustParseID("0199e300-0000-7000-8000-000000000010"),
	}
	work, err := domain.NewWorkItem(
		domain.MustParseID("0199e300-0000-7000-8000-000000000020"),
		scope,
		"Execute",
		"",
		domain.PriorityNormal,
		domain.WorkItemLifecycleTodo,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}

	assignee := domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "woobe", ID: "agent-a"}
	if err := work.SetAssignees([]domain.ActorRef{assignee}, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	claimID := domain.MustParseID("0199e300-0000-7000-8000-000000000030")
	leaseActor := domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "local", ID: "human-a"}
	if err := work.Claim(claimID, "principal-human-a", leaseActor, domain.DefaultLeaseTTL, now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if work.CurrentLease == nil || work.CurrentLease.PrincipalID != "principal-human-a" {
		t.Fatalf("lease principal = %#v", work.CurrentLease)
	}
	if len(work.AssigneeRefs) != 1 || work.AssigneeRefs[0] != assignee {
		t.Fatalf("claim unexpectedly changed declarative assignees: %#v", work.AssigneeRefs)
	}
}

func TestDuplicateOwnersAreRejected(t *testing.T) {
	now := time.Date(2026, 10, 1, 22, 0, 0, 0, time.UTC)
	outcome, err := domain.NewOutcome(
		domain.MustParseID("0199e300-0000-7000-8000-000000000010"),
		domain.MustParseID("0199e300-0000-7000-8000-000000000001"),
		"Ownership",
		"",
		"State",
		domain.PriorityNormal,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}

	owner := domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "local", ID: "human-a"}
	if err := outcome.SetOwners([]domain.ActorRef{owner, owner}, now.Add(time.Minute)); err == nil {
		t.Fatal("SetOwners() unexpectedly accepted duplicate owner")
	}
}
