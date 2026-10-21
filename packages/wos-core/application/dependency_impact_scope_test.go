package application_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
)

type dependencyImpactDenyRead struct{}

func (dependencyImpactDenyRead) Authorize(_ context.Context, request ports.AuthorizationRequest) error {
	if request.Permission == ports.PermissionStateRead {
		return domain.NewError(domain.ErrorCodeForbidden, "Outcome read is not authorized")
	}
	return nil
}

func TestDependencyImpactSQLiteReadsStayInsideAuthorizedOutcomeScope(t *testing.T) {
	ctx := context.Background()
	store, err := sqlite.Open(filepath.Join(t.TempDir(), "dependency-impact-scope.db"), sqlite.Options{MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ids := &sequenceIDs{}
	clock := fixedClock{now: time.Date(2026, 10, 21, 19, 0, 0, 0, time.UTC)}
	service, err := application.NewService(store, clock, ids)
	if err != nil {
		t.Fatal(err)
	}
	first := setupActiveOutcome(t, service)
	secondCreated, err := service.CreateOutcome(ctx, commandContext(), application.CreateOutcomeCommand{
		NamespaceID: first.Scope().NamespaceID, Title: "Sibling Outcome", DesiredState: "A separate graph in the same namespace", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.AddCriterion(ctx, commandContext(), application.AddCriterionCommand{
		Owner: secondCreated.Value.Ref(), ExpectedVersion: secondCreated.Value.Version, Title: "Sibling condition is met", Required: true, VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	secondOutcome, err := service.ActivateOutcome(ctx, commandContext(), application.ActivateOutcomeCommand{
		Scope: secondCreated.Value.Scope(), ExpectedVersion: secondCreated.Value.Version + 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	second := secondOutcome.Value

	firstTarget := createTodoWork(t, service, first.Scope(), "First Outcome prerequisite")
	firstSource := createTodoWork(t, service, first.Scope(), "First Outcome dependent")
	firstOtherSource := createTodoWork(t, service, first.Scope(), "Second dependent in the first Outcome")
	firstOtherTarget := createTodoWork(t, service, first.Scope(), "Second dependency target in the first Outcome")
	secondTarget := createTodoWork(t, service, second.Scope(), "Sibling Outcome prerequisite")
	secondSource := createTodoWork(t, service, second.Scope(), "Sibling Outcome dependent")
	addImpactDependency(t, service, first.Scope(), firstSource.Ref(), firstTarget.Ref(), domain.DependencyStrengthHard)
	addImpactDependency(t, service, first.Scope(), firstOtherSource.Ref(), firstTarget.Ref(), domain.DependencyStrengthHard)
	addImpactDependency(t, service, second.Scope(), secondSource.Ref(), secondTarget.Ref(), domain.DependencyStrengthHard)

	page, err := service.GetDependencyImpact(ctx, first.Scope(), application.DependencyImpactQuery{
		Subject: firstTarget.Ref(), Direction: application.DependencyImpactDownstream, Limit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 1 || page.Items[0].Summary.Ref != firstSource.Ref() {
		t.Fatalf("SQLite dependency graph crossed Outcome scope: %#v", page.Items)
	}
	if page.NextCursor == "" {
		t.Fatal("expected the SQLite impact page to include a revision-pinned continuation cursor")
	}
	impact := page.Items[0]
	if !impact.DependencyCoverageComplete || impact.DependencyCount == nil || *impact.DependencyCount != 1 || impact.SatisfiedHardDependencyCount == nil || *impact.SatisfiedHardDependencyCount != 0 {
		t.Fatalf("complete direct relation facts missing: %+v", impact)
	}
	if len(impact.Dependencies) != 1 || impact.Dependencies[0].RelationID == "" || impact.Dependencies[0].TargetRef != firstTarget.Ref() || impact.Dependencies[0].Satisfied || impact.Dependencies[0].Gate != application.DependencyImpactGateTaskAcquisition {
		t.Fatalf("direct relation satisfaction/gate facts are incorrect: %+v", impact.Dependencies)
	}
	assertImpactEffectGate(t, impact, application.DependencyImpactHardWaiting, application.DependencyImpactGateTaskAcquisition)
	continuation, continuationErr := service.GetDependencyImpact(ctx, first.Scope(), application.DependencyImpactQuery{Subject: firstTarget.Ref(), Direction: application.DependencyImpactDownstream, Limit: 1, Cursor: page.NextCursor})
	if continuationErr != nil {
		t.Fatalf("current SQLite cursor continuation failed: %v", continuationErr)
	}
	if len(continuation.Items) != 1 || continuation.Items[0].Summary.Ref != firstOtherSource.Ref() {
		t.Fatalf("SQLite cursor returned an unexpected scoped dependent: %+v", continuation.Items)
	}
	addImpactDependency(t, service, first.Scope(), firstOtherSource.Ref(), firstOtherTarget.Ref(), domain.DependencyStrengthAdvisory)
	_, err = service.GetDependencyImpact(ctx, first.Scope(), application.DependencyImpactQuery{Subject: firstTarget.Ref(), Direction: application.DependencyImpactDownstream, Limit: 1, Cursor: page.NextCursor})
	if code, ok := domain.ErrorCodeOf(err); !ok || code != domain.ErrorCodePreconditionFailed {
		t.Fatalf("stale SQLite impact cursor error = %v, want precondition_failed", err)
	}

	if _, err := service.GetDependencyImpact(ctx, first.Scope(), application.DependencyImpactQuery{Subject: secondTarget.Ref()}); err == nil {
		t.Fatal("query with a sibling-Outcome subject succeeded under the first Outcome scope")
	}

	authorized, err := application.NewAuthorizedService(store, clock, ids, dependencyImpactDenyRead{})
	if err != nil {
		t.Fatal(err)
	}
	identity := application.Identity{NamespaceID: first.Scope().NamespaceID, PrincipalID: "reader-unauthorized"}
	_, err = authorized.GetDependencyImpact(application.WithIdentity(ctx, identity), first.Scope(), application.DependencyImpactQuery{Subject: firstTarget.Ref()})
	if code, ok := domain.ErrorCodeOf(err); !ok || code != domain.ErrorCodeForbidden {
		t.Fatalf("unauthorized Outcome impact read error = %v, want forbidden", err)
	}
}

func TestDependencyImpactAchievedObjectiveSatisfiesAndReleasesDependentStartGate(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	prerequisiteCreated, err := service.CreateObjective(ctx, commandContext(), application.CreateObjectiveCommand{
		Scope: outcome.Scope(), Title: "Prerequisite Objective", Priority: domain.PriorityNormal,
		InitialCriteria: []application.InitialCriterion{{Title: "Prerequisite is verified", Required: true, VerificationMode: domain.VerificationModeAttestation}},
	})
	if err != nil {
		t.Fatal(err)
	}
	dependentCreated, err := service.CreateObjective(ctx, commandContext(), application.CreateObjectiveCommand{
		Scope: outcome.Scope(), Title: "Dependent Objective", Priority: domain.PriorityNormal,
		InitialCriteria: []application.InitialCriterion{{Title: "Dependent condition is ready", Required: true, VerificationMode: domain.VerificationModeAttestation}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddDependency(ctx, commandContext(), application.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: dependentCreated.Value.Ref(), TargetRef: prerequisiteCreated.Value.Ref(), Strength: domain.DependencyStrengthHard,
	}); err != nil {
		t.Fatal(err)
	}
	criterion := prerequisiteCreated.Value.Criteria.Items[0]
	if _, err := service.RecordCriterionAssessment(ctx, commandContext(), application.RecordCriterionAssessmentCommand{
		Owner: prerequisiteCreated.Value.Ref(), ExpectedVersion: prerequisiteCreated.Value.Version, CriterionID: criterion.ID,
		CriterionRevision: criterion.Revision, Result: domain.AssessmentResultMet, Rationale: "The prerequisite condition was verified",
	}); err != nil {
		t.Fatal(err)
	}
	current, err := service.GetObjective(ctx, outcome.Scope(), prerequisiteCreated.Value.ID)
	if err != nil {
		t.Fatal(err)
	}
	started, err := service.StartObjective(ctx, commandContext(), application.StartObjectiveCommand{Scope: outcome.Scope(), ObjectiveID: current.Value.ID, ExpectedVersion: current.Value.Version})
	if err != nil {
		t.Fatal(err)
	}
	achieved, err := service.AchieveObjective(ctx, commandContext(), application.AchieveObjectiveCommand{
		Scope: outcome.Scope(), ObjectiveID: started.Value.ID, ExpectedVersion: started.Value.Version, Reason: "Accept the verified prerequisite",
	})
	if err != nil {
		t.Fatal(err)
	}

	page, err := service.GetDependencyImpact(ctx, outcome.Scope(), application.DependencyImpactQuery{Subject: achieved.Value.Ref(), Direction: application.DependencyImpactDownstream})
	if err != nil {
		t.Fatal(err)
	}
	impact := findDependencyImpactNode(t, page, dependentCreated.Value.Ref())
	if impact.SatisfiedHardDependencyCount == nil || *impact.SatisfiedHardDependencyCount != 1 || impact.UnsatisfiedHardDependencyCount == nil || *impact.UnsatisfiedHardDependencyCount != 0 || len(impact.Dependencies) != 1 || !impact.Dependencies[0].Satisfied || impact.Dependencies[0].Gate != application.DependencyImpactGateObjectiveStart {
		t.Fatalf("Achieved prerequisite did not satisfy its dependent Objective gate: %+v", impact)
	}
	assertNoImpactEffect(t, impact, application.DependencyImpactHardWaiting)
	if _, err := service.StartObjective(ctx, commandContext(), application.StartObjectiveCommand{
		Scope: outcome.Scope(), ObjectiveID: dependentCreated.Value.ID, ExpectedVersion: dependentCreated.Value.Version,
	}); err != nil {
		t.Fatalf("dependent Objective should start after prerequisite achievement: %v", err)
	}
}

func assertImpactEffectGate(t *testing.T, node application.DependencyImpactNode, class, gate string) {
	t.Helper()
	for _, effect := range node.Effects {
		if effect.Class == class && effect.Gate == gate {
			return
		}
	}
	t.Fatalf("effect %s with gate %s absent from %+v", class, gate, node.Effects)
}
