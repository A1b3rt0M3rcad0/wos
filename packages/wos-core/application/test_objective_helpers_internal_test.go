package application

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

var fixtureObjectiveSequence atomic.Uint64

func objectiveForWorkFixture(t testing.TB, service *Service, scope d.Scope) *d.ID {
	t.Helper()
	sequence := fixtureObjectiveSequence.Add(1)
	commandID := d.MustParseID(fmt.Sprintf("0199ffff-0000-7000-8000-%012x", sequence))
	command := d.CommandContext{CommandID: commandID, PrincipalID: "test-fixture", Actor: d.ActorRef{Kind: d.ActorKindHuman, Provider: "test", ID: "fixture"}}
	ctx := WithIdentity(context.Background(), Identity{NamespaceID: scope.NamespaceID, PrincipalID: command.PrincipalID, Actor: command.Actor})
	state, err := service.GetOutcomeState(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	for _, objective := range state.Objectives {
		if objective.Lifecycle != d.ObjectiveLifecycleAchieved && objective.Lifecycle != d.ObjectiveLifecycleCancelled {
			id := objective.ID
			return &id
		}
	}
	created, err := service.CreateObjective(ctx, command, CreateObjectiveCommand{
		Scope: scope, Title: "Test Objective for bounded work", Priority: d.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	id := created.Value.ID
	return &id
}
