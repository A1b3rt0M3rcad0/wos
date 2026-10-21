package application_test

import (
	"context"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestDependencyImpactSeparatesBlockersFromHardWaiting(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	outcome := setupActiveOutcome(t, service)
	a := createTodoWork(t, service, outcome.Scope(), "A — blocked prerequisite")
	b := createTodoWork(t, service, outcome.Scope(), "B — depends on A")
	c := createTodoWork(t, service, outcome.Scope(), "C — depends on B")
	addImpactDependency(t, service, outcome.Scope(), b.Ref(), a.Ref(), domain.DependencyStrengthHard)
	addImpactDependency(t, service, outcome.Scope(), c.Ref(), b.Ref(), domain.DependencyStrengthHard)
	if _, err := service.CreateBlocker(ctx, commandContext(), application.CreateBlockerCommand{
		Scope: outcome.Scope(), BlockedRef: a.Ref(), Description: "A has an actual impediment",
	}); err != nil {
		t.Fatal(err)
	}

	before, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := len(store.SnapshotDomainEvents(outcome.Scope()))
	page, err := service.GetDependencyImpact(ctx, outcome.Scope(), application.DependencyImpactQuery{
		Subject: a.Ref(), Direction: application.DependencyImpactDownstream,
	})
	if err != nil {
		t.Fatal(err)
	}
	aImpact := page.Root
	bImpact := findDependencyImpactNode(t, page, b.Ref())
	cImpact := findDependencyImpactNode(t, page, c.Ref())
	if !aImpact.DirectBlocked || aImpact.DirectBlockerCount == nil || *aImpact.DirectBlockerCount != 1 {
		t.Fatalf("A should show its own active Blocker: %+v", aImpact)
	}
	assertImpactEffect(t, aImpact, application.DependencyImpactDirectBlocked, application.DependencyImpactCurrent)
	for name, impact := range map[string]application.DependencyImpactNode{"B": bImpact, "C": cImpact} {
		if impact.DirectBlocked || impact.DirectBlockerCount == nil || *impact.DirectBlockerCount != 0 {
			t.Fatalf("Blocker from A incorrectly propagated as a Blocker to %s: %+v", name, impact)
		}
		if impact.UnsatisfiedHardDependencyCount == nil || *impact.UnsatisfiedHardDependencyCount != 1 {
			t.Fatalf("%s should report its own incomplete hard prerequisite: %+v", name, impact)
		}
		assertImpactEffect(t, impact, application.DependencyImpactHardWaiting, application.DependencyImpactCurrent)
	}
	if len(bImpact.Path) != 1 || bImpact.Path[0].SourceRef != b.Ref() || bImpact.Path[0].TargetRef != a.Ref() {
		t.Fatalf("B path must preserve source-depends-on-target orientation: %+v", bImpact.Path)
	}
	if len(cImpact.Path) != 2 || cImpact.Path[0].SourceRef != b.Ref() || cImpact.Path[1].SourceRef != c.Ref() {
		t.Fatalf("C should have a deterministic two-step dependency path: %+v", cImpact.Path)
	}
	if !page.Coverage.Complete || !bImpact.StateCoverageComplete || !cImpact.StateCoverageComplete {
		t.Fatalf("small complete graph reported partial coverage: %+v", page.Coverage)
	}
	after, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if after.OutcomeRevision != before.OutcomeRevision || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsBefore {
		t.Fatal("dependency impact query mutated Outcome revision or event history")
	}
	bBlocker, err := service.CreateBlocker(ctx, commandContext(), application.CreateBlockerCommand{
		Scope: outcome.Scope(), BlockedRef: b.Ref(), Description: "B has its own independent impediment",
	})
	if err != nil {
		t.Fatal(err)
	}
	withOwnBlocker, err := service.GetDependencyImpact(ctx, outcome.Scope(), application.DependencyImpactQuery{
		Subject: a.Ref(), Direction: application.DependencyImpactDownstream,
	})
	if err != nil {
		t.Fatal(err)
	}
	bWithOwnBlocker := findDependencyImpactNode(t, withOwnBlocker, b.Ref())
	if !bWithOwnBlocker.DirectBlocked || bWithOwnBlocker.DirectBlockerCount == nil || *bWithOwnBlocker.DirectBlockerCount != 1 || len(bWithOwnBlocker.DirectBlockerSample) != 1 || bWithOwnBlocker.DirectBlockerSample[0].ID != bBlocker.Value.ID {
		t.Fatalf("a Blocker that directly targets B must remain visible independently: %+v", bWithOwnBlocker)
	}
	assertImpactEffect(t, bWithOwnBlocker, application.DependencyImpactDirectBlocked, application.DependencyImpactCurrent)
}

func TestDependencyImpactAdvisoryIsInformationalAndOrientationIsStable(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	prerequisite := createTodoWork(t, service, outcome.Scope(), "Prerequisite")
	dependent := createTodoWork(t, service, outcome.Scope(), "Dependent")
	addImpactDependency(t, service, outcome.Scope(), dependent.Ref(), prerequisite.Ref(), domain.DependencyStrengthAdvisory)

	downstream, err := service.GetDependencyImpact(ctx, outcome.Scope(), application.DependencyImpactQuery{
		Subject: prerequisite.Ref(), Direction: application.DependencyImpactDownstream,
	})
	if err != nil {
		t.Fatal(err)
	}
	dependentImpact := findDependencyImpactNode(t, downstream, dependent.Ref())
	if dependentImpact.UnsatisfiedHardDependencyCount == nil || *dependentImpact.UnsatisfiedHardDependencyCount != 0 || dependentImpact.DirectBlocked {
		t.Fatalf("advisory relation incorrectly changes readiness or blocking: %+v", dependentImpact)
	}
	assertImpactEffect(t, dependentImpact, application.DependencyImpactAdvisoryRelated, application.DependencyImpactInformational)
	if len(dependentImpact.Path) != 1 || dependentImpact.Path[0].Strength != domain.DependencyStrengthAdvisory {
		t.Fatalf("advisory path lost its strength: %+v", dependentImpact.Path)
	}

	upstream, err := service.GetDependencyImpact(ctx, outcome.Scope(), application.DependencyImpactQuery{
		Subject: dependent.Ref(), Direction: application.DependencyImpactUpstream,
	})
	if err != nil {
		t.Fatal(err)
	}
	prerequisiteImpact := findDependencyImpactNode(t, upstream, prerequisite.Ref())
	if len(prerequisiteImpact.Path) != 1 || prerequisiteImpact.Path[0].SourceRef != dependent.Ref() || prerequisiteImpact.Path[0].TargetRef != prerequisite.Ref() {
		t.Fatalf("upstream path did not retain source-depends-on-target orientation: %+v", prerequisiteImpact.Path)
	}
}

func TestDependencyImpactIncludesObjectiveDependencies(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	prerequisite, err := service.CreateObjective(ctx, commandContext(), application.CreateObjectiveCommand{
		Scope: outcome.Scope(), Title: "Prerequisite Objective", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	dependent, err := service.CreateObjective(ctx, commandContext(), application.CreateObjectiveCommand{
		Scope: outcome.Scope(), Title: "Dependent Objective", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddDependency(ctx, commandContext(), application.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: dependent.Value.Ref(), TargetRef: prerequisite.Value.Ref(),
		Strength: domain.DependencyStrengthHard, Reason: "the prerequisite must be achieved first",
	}); err != nil {
		t.Fatal(err)
	}
	page, err := service.GetDependencyImpact(ctx, outcome.Scope(), application.DependencyImpactQuery{
		Subject: prerequisite.Value.Ref(), Direction: application.DependencyImpactDownstream,
	})
	if err != nil {
		t.Fatal(err)
	}
	impact := findDependencyImpactNode(t, page, dependent.Value.Ref())
	if impact.UnsatisfiedHardDependencyCount == nil || *impact.UnsatisfiedHardDependencyCount != 1 {
		t.Fatalf("planned Objective should wait on its incomplete hard prerequisite: %+v", impact)
	}
	assertImpactEffect(t, impact, application.DependencyImpactHardWaiting, application.DependencyImpactCurrent)
}

func TestDependencyImpactShowsSatisfiedHardRelationWithoutRetroactiveWaiting(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	prerequisite := createTodoWork(t, service, outcome.Scope(), "Completed prerequisite")
	prerequisite = deferPlanningWork(t, service, prerequisite)
	criterion, err := service.AddCriterion(ctx, commandContext(), application.AddCriterionCommand{
		Owner: prerequisite.Ref(), ExpectedVersion: prerequisite.Version, Title: "Prerequisite verified",
		Required: true, VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	prerequisite = activatePlanningWork(t, service, prerequisite)
	dependent := createTodoWork(t, service, outcome.Scope(), "Dependent")
	addImpactDependency(t, service, outcome.Scope(), dependent.Ref(), prerequisite.Ref(), domain.DependencyStrengthHard)
	objective, err := service.GetObjective(ctx, outcome.Scope(), *prerequisite.ObjectiveID)
	if err != nil {
		t.Fatal(err)
	}
	if objective.Value.Lifecycle == domain.ObjectiveLifecyclePlanned {
		objectiveCriterion, addErr := service.AddCriterion(ctx, commandContext(), application.AddCriterionCommand{
			Owner: objective.Value.Ref(), ExpectedVersion: objective.Value.Version, Title: "Objective verification",
			Required: true, VerificationMode: domain.VerificationModeAttestation,
		})
		if addErr != nil {
			t.Fatal(addErr)
		}
		objective, err = service.GetObjective(ctx, outcome.Scope(), objective.Value.ID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = service.RecordCriterionAssessment(ctx, commandContext(), application.RecordCriterionAssessmentCommand{
			Owner: objective.Value.Ref(), ExpectedVersion: objective.Value.Version, CriterionID: objectiveCriterion.Value.ID,
			CriterionRevision: objectiveCriterion.Value.Revision, Result: domain.AssessmentResultMet, Rationale: "The objective condition was checked",
		}); err != nil {
			t.Fatal(err)
		}
		objective, err = service.GetObjective(ctx, outcome.Scope(), objective.Value.ID)
		if err != nil {
			t.Fatal(err)
		}
		started, startErr := service.StartObjective(ctx, commandContext(), application.StartObjectiveCommand{
			Scope: outcome.Scope(), ObjectiveID: objective.Value.ID, ExpectedVersion: objective.Value.Version,
		})
		if startErr != nil {
			t.Fatal(startErr)
		}
		objective.Value = started.Value
	}
	latest, err := service.GetWorkItem(ctx, outcome.Scope(), prerequisite.ID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.RecordCriterionAssessment(ctx, commandContext(), application.RecordCriterionAssessmentCommand{
		Owner: latest.Value.Ref(), ExpectedVersion: latest.Value.Version, CriterionID: criterion.Value.ID,
		CriterionRevision: criterion.Value.Revision, Result: domain.AssessmentResultMet, Rationale: "Prerequisite output was checked",
	})
	if err != nil {
		t.Fatal(err)
	}
	latest, err = service.GetWorkItem(ctx, outcome.Scope(), prerequisite.ID)
	if err != nil {
		t.Fatal(err)
	}
	prerequisite = latest.Value

	claimed, err := service.ClaimWorkItem(ctx, commandContext(), application.ClaimWorkItemCommand{Scope: outcome.Scope(), WorkItemID: prerequisite.ID, ExpectedVersion: prerequisite.Version})
	if err != nil {
		t.Fatal(err)
	}
	completed, err := service.CompleteWorkItem(ctx, commandContext(), application.CompleteWorkItemCommand{
		Scope: outcome.Scope(), WorkItemID: claimed.Value.ID, ExpectedVersion: claimed.Value.Version,
		ClaimID: claimed.Value.CurrentLease.ClaimID, FencingToken: claimed.Value.CurrentLease.FencingToken,
		ResultSummary: "The prerequisite is complete", Reason: "Verified by the test fixture",
	})
	if err != nil {
		t.Fatal(err)
	}

	page, err := service.GetDependencyImpact(ctx, outcome.Scope(), application.DependencyImpactQuery{
		Subject: prerequisite.Ref(), Direction: application.DependencyImpactDownstream,
	})
	if err != nil {
		t.Fatal(err)
	}
	impact := findDependencyImpactNode(t, page, dependent.Ref())
	if impact.Summary.Lifecycle != string(domain.WorkItemLifecycleTodo) || impact.UnsatisfiedHardDependencyCount == nil || *impact.UnsatisfiedHardDependencyCount != 0 {
		t.Fatalf("completed prerequisite should satisfy the relation without changing the dependent's lifecycle: %+v (completed=%s)", impact, completed.Value.Lifecycle)
	}
	assertNoImpactEffect(t, impact, application.DependencyImpactHardWaiting)
}

func TestDependencyImpactReportsBoundedCoverageAndRevisionBoundPagination(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	outcome := setupActiveOutcome(t, service)
	a := createTodoWork(t, service, outcome.Scope(), "A")
	b := createTodoWork(t, service, outcome.Scope(), "B")
	c := createTodoWork(t, service, outcome.Scope(), "C")
	d := createTodoWork(t, service, outcome.Scope(), "D")
	addImpactDependency(t, service, outcome.Scope(), b.Ref(), a.Ref(), domain.DependencyStrengthHard)
	addImpactDependency(t, service, outcome.Scope(), c.Ref(), b.Ref(), domain.DependencyStrengthHard)
	addImpactDependency(t, service, outcome.Scope(), d.Ref(), c.Ref(), domain.DependencyStrengthHard)

	bounded, err := service.GetDependencyImpact(ctx, outcome.Scope(), application.DependencyImpactQuery{
		Subject: a.Ref(), Direction: application.DependencyImpactDownstream, Depth: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if bounded.Coverage.Complete || bounded.Coverage.Status != "partial" || bounded.Coverage.Reason != "depth_limit" {
		t.Fatalf("truncated dependency depth must be explicit: %+v", bounded.Coverage)
	}
	if len(bounded.Items) != 1 || bounded.Items[0].Summary.Ref != b.Ref() {
		t.Fatalf("depth 1 should include only the direct dependent: %+v", bounded.Items)
	}

	first, err := service.GetDependencyImpact(ctx, outcome.Scope(), application.DependencyImpactQuery{
		Subject: a.Ref(), Direction: application.DependencyImpactDownstream, Limit: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if first.NextCursor == "" {
		t.Fatal("expected a next cursor for paged impacts")
	}
	addImpactDependency(t, service, outcome.Scope(), d.Ref(), b.Ref(), domain.DependencyStrengthAdvisory)
	_, err = service.GetDependencyImpact(ctx, outcome.Scope(), application.DependencyImpactQuery{
		Subject: a.Ref(), Direction: application.DependencyImpactDownstream, Limit: 1, Cursor: first.NextCursor,
	})
	if code, ok := domain.ErrorCodeOf(err); !ok || code != domain.ErrorCodePreconditionFailed {
		t.Fatalf("stale impact cursor error = %v, want snapshot_changed", err)
	}
	if _, err = service.GetDependencyImpact(ctx, outcome.Scope(), application.DependencyImpactQuery{Subject: a.Ref(), Direction: "sideways"}); err == nil {
		t.Fatal("invalid dependency direction unexpectedly succeeded")
	}
}

func addImpactDependency(t *testing.T, service *application.Service, scope domain.Scope, source, target domain.EntityRef, strength domain.DependencyStrength) {
	t.Helper()
	wasTodo := false
	if source.Kind == domain.EntityKindWorkItem {
		current, err := service.GetWorkItem(context.Background(), scope, source.ID)
		if err != nil {
			t.Fatal(err)
		}
		if current.Value.Lifecycle != domain.WorkItemLifecycleBacklog {
			if current.Value.Lifecycle != domain.WorkItemLifecycleTodo {
				t.Fatalf("dependency fixture source must be To do or Backlog, got %s", current.Value.Lifecycle)
			}
			if _, err := service.DeferWorkItem(context.Background(), commandContext(), application.DeferWorkItemCommand{Scope: scope, WorkItemID: source.ID, ExpectedVersion: current.Value.Version}); err != nil {
				t.Fatal(err)
			}
			wasTodo = true
		}
	}
	if _, err := service.AddDependency(context.Background(), commandContext(), application.AddDependencyCommand{
		Scope: scope, SourceRef: source, TargetRef: target, Strength: strength, Reason: "dependency impact fixture",
	}); err != nil {
		t.Fatal(err)
	}
	if wasTodo {
		if current, err := service.GetWorkItem(context.Background(), scope, source.ID); err != nil {
			t.Fatal(err)
		} else if _, err := service.ActivateWorkItem(context.Background(), commandContext(), application.ActivateWorkItemCommand{Scope: scope, WorkItemID: source.ID, ExpectedVersion: current.Value.Version}); err != nil {
			t.Fatal(err)
		}
	}
}

func findDependencyImpactNode(t *testing.T, page application.DependencyImpactPage, ref domain.EntityRef) application.DependencyImpactNode {
	t.Helper()
	for _, node := range page.Items {
		if node.Summary.Ref == ref {
			return node
		}
	}
	t.Fatalf("dependency impact omitted %s", ref.ID)
	return application.DependencyImpactNode{}
}

func assertImpactEffect(t *testing.T, node application.DependencyImpactNode, class, status string) {
	t.Helper()
	for _, effect := range node.Effects {
		if effect.Class == class && effect.Status == status {
			return
		}
	}
	t.Fatalf("effect %s/%s absent from %+v", class, status, node.Effects)
}

func assertNoImpactEffect(t *testing.T, node application.DependencyImpactNode, class string) {
	t.Helper()
	for _, effect := range node.Effects {
		if effect.Class == class {
			t.Fatalf("unexpected effect %s in %+v", class, node.Effects)
		}
	}
}
