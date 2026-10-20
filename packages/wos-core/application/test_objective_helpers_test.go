package application_test

import (
	"context"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func objectiveForWorkFixture(t testing.TB, service *application.Service, scope domain.Scope) *domain.ID {
	t.Helper()
	cc := commandContext()
	ctx := application.WithIdentity(context.Background(), application.Identity{
		NamespaceID: scope.NamespaceID, PrincipalID: cc.PrincipalID, Actor: cc.Actor,
	})
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
	created, err := service.CreateObjective(ctx, cc, application.CreateObjectiveCommand{
		Scope: scope, Title: "Test Objective for bounded work", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	id := created.Value.ID
	return &id
}
