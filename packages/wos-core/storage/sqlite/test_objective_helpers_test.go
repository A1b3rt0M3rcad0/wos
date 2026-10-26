package sqlite

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

var sqliteFixtureObjectiveSequence atomic.Uint64

func sqliteObjectiveForWorkFixture(t testing.TB, service *application.Service, scope domain.Scope) *domain.ID {
	t.Helper()
	ctx := context.Background()
	state, err := service.GetOutcomeState(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	for _, objective := range state.Objectives {
		if objective.Lifecycle != domain.ObjectiveLifecycleAchieved && objective.Lifecycle != domain.ObjectiveLifecycleCancelled {
			id := objective.ID
			return &id
		}
	}
	sequence := sqliteFixtureObjectiveSequence.Add(1)
	commandID := fmt.Sprintf("0199feaa-0000-7000-8000-%012x", sequence)
	created, err := service.CreateObjective(ctx, sqliteCommandContext(commandID, ""), application.CreateObjectiveCommand{
		Scope: scope, Title: "Test Objective for bounded work", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	id := created.Value.ID
	return &id
}
