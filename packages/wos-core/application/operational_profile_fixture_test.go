package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

type operationalProfileFixture struct {
	Context             context.Context
	Command             domain.CommandContext
	Service             *application.Service
	Store               *memory.Store
	Scope               domain.Scope
	Outcome             domain.Outcome
	PlannedObjective    domain.Objective
	InProgressObjective domain.Objective
	Tasks               map[string]domain.WorkItem
	Credentials         map[string]ports.Credential
	Dependencies        []domain.Relation
	Issues              []domain.Issue
	Blockers            []domain.Blocker
}

// newOperationalProfileFixture builds ordinary persisted records through the
// application API. It intentionally characterizes only lifecycle states and
// authority records the current implementation already supports.
func newOperationalProfileFixture(t *testing.T) *operationalProfileFixture {
	t.Helper()

	ctx := context.Background()
	service, store := newService(t)
	cc := commandContext()
	namespaceID := id("0199e200-0000-7000-8000-000000000001")
	createdOutcome, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Operational profile fixture",
		DesiredState: "Current application lifecycle and authority records can be exercised",
		Priority:     domain.PriorityNormal,
		InitialCriteria: []application.InitialCriterion{{
			Title:    "Fixture outcome criterion",
			Required: true, VerificationMode: domain.VerificationModeAttestation,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	activatedOutcome, err := service.ActivateOutcome(ctx, cc, application.ActivateOutcomeCommand{
		Scope: createdOutcome.Value.Scope(), ExpectedVersion: createdOutcome.Value.Version,
	})
	if err != nil {
		t.Fatal(err)
	}

	planned := createObjectiveFixture(t, service, cc, activatedOutcome.Value.Scope(), "Planned fixture Objective", domain.ObjectiveLifecyclePlanned)
	inProgress := createObjectiveFixture(t, service, cc, activatedOutcome.Value.Scope(), "In-progress fixture Objective", domain.ObjectiveLifecycleInProgress)

	fixture := &operationalProfileFixture{
		Context:             ctx,
		Command:             cc,
		Service:             service,
		Store:               store,
		Scope:               activatedOutcome.Value.Scope(),
		Outcome:             activatedOutcome.Value,
		PlannedObjective:    planned,
		InProgressObjective: inProgress,
		Tasks:               make(map[string]domain.WorkItem),
		Credentials:         make(map[string]ports.Credential),
	}

	fixture.Tasks["backlog"] = createOperationalTaskFixture(t, fixture, "Backlog task", planned.ID, domain.WorkItemLifecycleBacklog)
	fixture.Tasks["todo"] = createOperationalTaskFixture(t, fixture, "To-do task", inProgress.ID, domain.WorkItemLifecycleTodo)
	fixture.Tasks["in_progress"] = createOperationalTaskFixture(t, fixture, "In-progress task", inProgress.ID, domain.WorkItemLifecycleInProgress)
	fixture.Tasks["done"] = createOperationalTaskFixture(t, fixture, "Done task", inProgress.ID, domain.WorkItemLifecycleDone)
	fixture.Tasks["cancelled"] = createOperationalTaskFixture(t, fixture, "Cancelled task", planned.ID, domain.WorkItemLifecycleCancelled)

	seedCredentialFixture(t, fixture)
	return fixture
}

func createOperationalTaskFixture(
	t *testing.T,
	fixture *operationalProfileFixture,
	title string,
	objectiveID domain.ID,
	lifecycle domain.WorkItemLifecycle,
) domain.WorkItem {
	t.Helper()

	initialLifecycle := lifecycle
	if lifecycle == domain.WorkItemLifecycleInProgress || lifecycle == domain.WorkItemLifecycleDone || lifecycle == domain.WorkItemLifecycleCancelled {
		initialLifecycle = domain.WorkItemLifecycleTodo
		if lifecycle == domain.WorkItemLifecycleCancelled {
			initialLifecycle = domain.WorkItemLifecycleBacklog
		}
	}
	created, err := fixture.Service.CreateWorkItem(fixture.Context, fixture.Command, application.CreateWorkItemCommand{
		Scope: fixture.Scope, ObjectiveID: &objectiveID, Title: title,
		Priority: domain.PriorityNormal, Lifecycle: initialLifecycle,
	})
	if err != nil {
		t.Fatal(err)
	}
	item := created.Value

	switch lifecycle {
	case domain.WorkItemLifecycleBacklog, domain.WorkItemLifecycleTodo:
		return item
	case domain.WorkItemLifecycleInProgress, domain.WorkItemLifecycleDone:
		claimed, err := fixture.Service.ClaimWorkItem(fixture.Context, fixture.Command, application.ClaimWorkItemCommand{
			Scope: fixture.Scope, WorkItemID: item.ID, ExpectedVersion: item.Version, TTL: domain.DefaultLeaseTTL,
		})
		if err != nil {
			t.Fatal(err)
		}
		item = claimed.Value
		if lifecycle == domain.WorkItemLifecycleInProgress {
			return item
		}
		completed, err := fixture.Service.CompleteWorkItem(fixture.Context, fixture.Command, application.CompleteWorkItemCommand{
			Scope: fixture.Scope, WorkItemID: item.ID, ExpectedVersion: item.Version,
			ClaimID: item.CurrentLease.ClaimID, FencingToken: item.CurrentLease.FencingToken,
			ResultSummary: "Fixture work was completed through the current lease protocol.",
			Reason:        "Characterize the existing done lifecycle.",
		})
		if err != nil {
			t.Fatal(err)
		}
		return completed.Value
	case domain.WorkItemLifecycleCancelled:
		cancelled, err := fixture.Service.CancelWorkItem(fixture.Context, fixture.Command, application.CancelWorkItemCommand{
			Scope: fixture.Scope, WorkItemID: item.ID, ExpectedVersion: item.Version,
			Reason: "Characterize the existing cancelled lifecycle.",
		})
		if err != nil {
			t.Fatal(err)
		}
		return cancelled.Value
	default:
		t.Fatalf("unsupported Task fixture lifecycle %q", lifecycle)
		return domain.WorkItem{}
	}
}

// seedCredentialFixture adds test identities to the same in-memory Namespace.
// Digests are opaque memory-store lookup labels, not bearer secrets. No
// signing identity is created, so these records cannot stand in for signed
// protocol credentials or proofs.
func seedCredentialFixture(t *testing.T, fixture *operationalProfileFixture) {
	t.Helper()

	now := time.Date(2026, time.October, 1, 21, 30, 0, 0, time.UTC)
	expiresAt := now.Add(366 * 24 * time.Hour)
	human := domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "fixture", ID: "operator"}
	agent := domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "fixture", ID: "worker"}
	reviewer := domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "fixture", ID: "reviewer"}
	primary := ports.Credential{
		NamespaceID: fixture.Scope.NamespaceID, ID: id("0199e280-0000-7000-8000-000000000001"),
		PrincipalID: "principal-operator", Actor: human, Digest: "test-credential-digest/operator-human", ExpiresAt: expiresAt,
	}
	grant := ports.NamespaceGrant{NamespaceID: fixture.Scope.NamespaceID, PrincipalID: primary.PrincipalID, Permissions: []ports.Permission{ports.PermissionStateRead}}
	prepared, err := application.PrepareCredentialIssuance(application.CredentialIssuanceSelection{
		NamespaceID: fixture.Scope.NamespaceID, CredentialID: primary.ID, PrincipalID: primary.PrincipalID,
		Actor: primary.Actor, ExpiresAt: primary.ExpiresAt, Name: "Fixture operator credential",
		CreatedByPrincipalID: primary.PrincipalID, CreatedByActor: primary.Actor,
	}, grant, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := fixture.Store.BootstrapSecurity(fixture.Context,
		ports.Namespace{ID: fixture.Scope.NamespaceID, Name: "fixture"},
		grant,
		primary,
		prepared.Policy,
		prepared.Metadata,
	); err != nil {
		t.Fatal(err)
	}
	worker := ports.Credential{
		NamespaceID: fixture.Scope.NamespaceID, ID: id("0199e280-0000-7000-8000-000000000002"),
		PrincipalID: primary.PrincipalID, Actor: agent, Digest: "test-credential-digest/operator-agent", ExpiresAt: expiresAt,
	}
	if err := fixture.Store.PutCredential(fixture.Context, worker); err != nil {
		t.Fatal(err)
	}
	otherPrincipal := ports.Credential{
		NamespaceID: fixture.Scope.NamespaceID, ID: id("0199e280-0000-7000-8000-000000000003"),
		PrincipalID: "principal-reviewer", Actor: reviewer, Digest: "test-credential-digest/reviewer", ExpiresAt: expiresAt,
	}
	if err := fixture.Store.PutGrant(fixture.Context, ports.NamespaceGrant{
		NamespaceID: fixture.Scope.NamespaceID, PrincipalID: otherPrincipal.PrincipalID,
		Permissions: []ports.Permission{ports.PermissionStateRead},
	}); err != nil {
		t.Fatal(err)
	}
	if err := fixture.Store.PutCredential(fixture.Context, otherPrincipal); err != nil {
		t.Fatal(err)
	}
	fixture.Credentials["operator_human"] = primary
	fixture.Credentials["operator_agent"] = worker
	fixture.Credentials["reviewer"] = otherPrincipal
}

// addSupportedRelationships opts into genuine current relations for tests that
// need dependency/Issue/Blocker state. They are created through application
// use cases and do not change Task or Objective lifecycle fields.
func (fixture *operationalProfileFixture) addSupportedRelationships(t *testing.T) {
	t.Helper()
	if len(fixture.Dependencies)+len(fixture.Issues)+len(fixture.Blockers) != 0 {
		t.Fatal("supported relationships already added to this fixture")
	}

	source := createOperationalTaskFixture(t, fixture, "Dependency source", fixture.PlannedObjective.ID, domain.WorkItemLifecycleTodo)
	target := createOperationalTaskFixture(t, fixture, "Dependency target", fixture.PlannedObjective.ID, domain.WorkItemLifecycleTodo)
	deferredSource, err := fixture.Service.DeferWorkItem(fixture.Context, fixture.Command, application.DeferWorkItemCommand{
		Scope: fixture.Scope, WorkItemID: source.ID, ExpectedVersion: source.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	source = deferredSource.Value
	relation, err := fixture.Service.AddDependency(fixture.Context, fixture.Command, application.AddDependencyCommand{
		Scope: fixture.Scope, SourceRef: source.Ref(), TargetRef: target.Ref(),
		Strength: domain.DependencyStrengthHard, Reason: "Fixture dependency for readiness characterization.",
	})
	if err != nil {
		t.Fatal(err)
	}
	fixture.Dependencies = append(fixture.Dependencies, relation.Value)

	issue, err := fixture.Service.CreateIssue(fixture.Context, fixture.Command, application.CreateIssueCommand{
		Scope: fixture.Scope, ObjectiveID: fixture.PlannedObjective.ID,
		Title: "Fixture issue", Description: "An owned issue created through the current application service.",
		Severity: domain.IssueSeverityMinor, AffectedRefs: []domain.EntityRef{target.Ref()},
	})
	if err != nil {
		t.Fatal(err)
	}
	fixture.Issues = append(fixture.Issues, issue.Value)

	blocker, err := fixture.Service.CreateBlocker(fixture.Context, fixture.Command, application.CreateBlockerCommand{
		Scope: fixture.Scope, BlockedRef: target.Ref(), CauseRef: refPointer(issue.Value.Ref()),
		Description: "Fixture blocker tied to the explicit issue.", Propagation: domain.BlockerPropagationDirect,
	})
	if err != nil {
		t.Fatal(err)
	}
	fixture.Blockers = append(fixture.Blockers, blocker.Value)

	subtreeBlocker, err := fixture.Service.CreateBlocker(fixture.Context, fixture.Command, application.CreateBlockerCommand{
		Scope: fixture.Scope, BlockedRef: fixture.PlannedObjective.Ref(), CauseRef: refPointer(issue.Value.Ref()),
		Description: "Fixture subtree blocker on the Objective and its descendants.",
		Propagation: domain.BlockerPropagationSubtree,
	})
	if err != nil {
		t.Fatal(err)
	}
	fixture.Blockers = append(fixture.Blockers, subtreeBlocker.Value)
}

func refPointer(ref domain.EntityRef) *domain.EntityRef { return &ref }

func TestOperationalProfileFixtureCapturesCurrentApplicationBaseline(t *testing.T) {
	fixture := newOperationalProfileFixture(t)
	if fixture.Outcome.Lifecycle != domain.OutcomeLifecycleActive {
		t.Fatalf("Outcome lifecycle = %q, want active", fixture.Outcome.Lifecycle)
	}
	if fixture.PlannedObjective.Lifecycle != domain.ObjectiveLifecyclePlanned || fixture.InProgressObjective.Lifecycle != domain.ObjectiveLifecycleInProgress {
		t.Fatalf("Objective fixture lifecycles = %q and %q", fixture.PlannedObjective.Lifecycle, fixture.InProgressObjective.Lifecycle)
	}
	wantTaskLifecycles := map[string]domain.WorkItemLifecycle{
		"backlog":     domain.WorkItemLifecycleBacklog,
		"todo":        domain.WorkItemLifecycleTodo,
		"in_progress": domain.WorkItemLifecycleInProgress,
		"done":        domain.WorkItemLifecycleDone,
		"cancelled":   domain.WorkItemLifecycleCancelled,
	}
	for name, want := range wantTaskLifecycles {
		if got := fixture.Tasks[name].Lifecycle; got != want {
			t.Errorf("Task %q lifecycle = %q, want %q", name, got, want)
		}
	}
	if lease := fixture.Tasks["in_progress"].CurrentLease; lease == nil || lease.PrincipalID != fixture.Command.PrincipalID {
		t.Fatal("in-progress Task must retain the lease created by the application claim use case")
	}
	if fixture.Tasks["done"].CurrentLease != nil {
		t.Fatal("done Task must not retain an active lease")
	}

	operatorHuman := fixture.Credentials["operator_human"]
	operatorAgent := fixture.Credentials["operator_agent"]
	reviewer := fixture.Credentials["reviewer"]
	if operatorHuman.PrincipalID != operatorAgent.PrincipalID || operatorHuman.ID == operatorAgent.ID {
		t.Fatal("fixture must model distinct credentials belonging to the same Principal")
	}
	if reviewer.PrincipalID == operatorHuman.PrincipalID {
		t.Fatal("reviewer fixture must use a separate Principal")
	}
	for label, credential := range fixture.Credentials {
		stored, err := fixture.Store.GetCredential(fixture.Context, credential.Digest)
		if err != nil {
			t.Errorf("credential %q was not stored: %v", label, err)
			continue
		}
		if stored.ID != credential.ID || stored.PrincipalID != credential.PrincipalID {
			t.Errorf("credential %q changed during test bootstrap: got %#v", label, stored)
		}
	}
}

func TestOperationalProfileFixtureAddsOnlySupportedRelationships(t *testing.T) {
	fixture := newOperationalProfileFixture(t)
	fixture.addSupportedRelationships(t)
	if len(fixture.Dependencies) != 1 || fixture.Dependencies[0].RelationType != domain.RelationTypeDependsOn {
		t.Fatalf("dependency fixture = %#v", fixture.Dependencies)
	}
	if len(fixture.Issues) != 1 || fixture.Issues[0].ObjectiveID == nil || *fixture.Issues[0].ObjectiveID != fixture.PlannedObjective.ID {
		t.Fatalf("Issue fixture is not owned by the planned Objective: %#v", fixture.Issues)
	}
	if len(fixture.Blockers) != 2 || fixture.Blockers[0].CauseRef == nil || fixture.Blockers[0].CauseRef.Kind != domain.EntityKindIssue {
		t.Fatalf("Blocker fixtures do not reference the explicit Issue: %#v", fixture.Blockers)
	}
	if fixture.Blockers[0].Propagation != domain.BlockerPropagationDirect {
		t.Fatalf("Task Blocker propagation = %q, want direct", fixture.Blockers[0].Propagation)
	}
	if fixture.Blockers[1].BlockedRef.Kind != domain.EntityKindObjective || fixture.Blockers[1].Propagation != domain.BlockerPropagationSubtree {
		t.Fatalf("Objective subtree Blocker fixture = %#v", fixture.Blockers[1])
	}
}
