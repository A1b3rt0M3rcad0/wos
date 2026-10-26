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
	ctx := applicationFixtureIdentityContext(context.Background(), scope, cc)
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
	created := createObjectiveFixture(t, service, cc, scope, "Test Objective for bounded work", domain.ObjectiveLifecyclePlanned)
	id := created.ID
	return &id
}

// applicationFixtureIdentityContext mirrors the trusted adapter context used
// by application read paths without changing the command's declared actor.
func applicationFixtureIdentityContext(ctx context.Context, scope domain.Scope, cc domain.CommandContext) context.Context {
	return application.WithIdentity(ctx, application.Identity{
		NamespaceID: scope.NamespaceID,
		PrincipalID: cc.PrincipalID,
		Actor:       cc.Actor,
	})
}

// createObjectiveFixture creates an Objective through the existing application
// use case and, when requested, starts it through the current lifecycle gate.
func createObjectiveFixture(
	t testing.TB,
	service *application.Service,
	cc domain.CommandContext,
	scope domain.Scope,
	title string,
	lifecycle domain.ObjectiveLifecycle,
) domain.Objective {
	t.Helper()

	var initialCriteria []application.InitialCriterion
	if lifecycle == domain.ObjectiveLifecycleInProgress {
		initialCriteria = []application.InitialCriterion{{
			Title: "Fixture Objective criterion", Required: true,
			VerificationMode: domain.VerificationModeAttestation,
		}}
	}
	created, err := service.CreateObjective(context.Background(), cc, application.CreateObjectiveCommand{
		Scope: scope, Title: title, Priority: domain.PriorityNormal, InitialCriteria: initialCriteria,
	})
	if err != nil {
		t.Fatal(err)
	}
	objective := created.Value
	if lifecycle == domain.ObjectiveLifecyclePlanned {
		return objective
	}
	if lifecycle != domain.ObjectiveLifecycleInProgress {
		t.Fatalf("unsupported Objective fixture lifecycle %q", lifecycle)
	}
	started, err := service.StartObjective(context.Background(), cc, application.StartObjectiveCommand{
		Scope: scope, ObjectiveID: objective.ID, ExpectedVersion: objective.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	return started.Value
}
