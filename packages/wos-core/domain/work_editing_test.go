package domain_test

import (
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestWorkDefinitionSettersReplaceClearAndPreserveAuthority(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	scope := d.Scope{NamespaceID: d.MustParseID("0199e300-0000-7000-8000-000000000001"), OutcomeID: d.MustParseID("0199e300-0000-7000-8000-000000000002")}
	w, err := d.NewWorkItem(d.MustParseID("0199e300-0000-7000-8000-000000000003"), scope, "Draft work", "", d.PriorityNormal, d.WorkItemLifecycleBacklog, now)
	if err != nil {
		t.Fatal(err)
	}
	spec := &d.ExecutionSpec{Instructions: []string{"Implement the scoped change"}, Deliverables: []string{"Reviewable patch"}}
	if changed, err := w.SetExecutionSpec(spec, now); err != nil || !changed {
		t.Fatalf("set spec: %v %v", changed, err)
	}
	spec.Instructions[0] = "mutated caller input"
	if w.ExecutionSpec.Instructions[0] != "Implement the scoped change" {
		t.Fatal("setter retained caller-owned mutable slices")
	}
	version := w.Version
	copy := *w.ExecutionSpec
	if changed, err := w.SetExecutionSpec(&copy, now); err != nil || changed || w.Version != version {
		t.Fatal("identical replacement was not a no-op", err)
	}
	if changed, err := w.SetExecutionSpec(nil, now); err != nil || !changed || w.ExecutionSpec != nil {
		t.Fatal("explicit null did not clear specification", err)
	}
	date := now.In(time.FixedZone("caller", -3*60*60)).Add(time.Hour)
	if changed, err := w.SetNotBefore(&date, now); err != nil || !changed || w.NotBefore.Location() != time.UTC {
		t.Fatal("scheduling setter did not normalize UTC", err)
	}
	if _, err := w.SetNotBefore(&time.Time{}, now); err == nil {
		t.Fatal("zero not_before accepted")
	}
	if _, err := w.SetNotBefore(nil, now); err != nil || w.NotBefore != nil {
		t.Fatal("null did not clear scheduling", err)
	}
	parent := d.MustParseID("0199e300-0000-7000-8000-000000000004")
	if _, err := w.SetObjective(&parent, now); err != nil || w.ObjectiveID == nil || *w.ObjectiveID != parent {
		t.Fatal("Objective association failed", err)
	}
	if _, err := w.SetObjective(nil, now); err != nil || w.ObjectiveID != nil {
		t.Fatal("null did not restore direct Outcome membership", err)
	}
	if w.Lifecycle != d.WorkItemLifecycleBacklog || w.CurrentLease != nil || w.CurrentContractID != nil {
		t.Fatal("definition edits granted execution authority")
	}
	if err := w.Activate(now); err != nil {
		t.Fatal(err)
	}
	if err := w.Claim(d.MustParseID("0199e300-0000-7000-8000-000000000005"), "worker", d.ActorRef{Kind: d.ActorKindAgent, Provider: "local", ID: "worker"}, time.Minute, now); err != nil {
		t.Fatal(err)
	}
	before := w.Version
	if _, err := w.SetExecutionSpec(spec, now); err == nil || w.Version != before || w.CurrentLease == nil {
		t.Fatal("active lease definition changed or was revoked")
	}
}

func TestOwnObjectiveHardDependencyIsAnImpossibleReadinessConfiguration(t *testing.T) {
	now := time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)
	scope := d.Scope{NamespaceID: d.MustParseID("0199e300-0000-7000-8000-000000000001"), OutcomeID: d.MustParseID("0199e300-0000-7000-8000-000000000002")}
	o, _ := d.NewOutcome(scope.OutcomeID, scope.NamespaceID, "Outcome", "", "Accepted result", d.PriorityNormal, now)
	o.Lifecycle = d.OutcomeLifecycleActive
	objective, _ := d.NewObjective(d.MustParseID("0199e300-0000-7000-8000-000000000003"), scope, "Condition", "", d.PriorityNormal, false, now)
	w, _ := d.NewWorkItem(d.MustParseID("0199e300-0000-7000-8000-000000000004"), scope, "Work", "", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
	w.ObjectiveID = &objective.ID
	relation, _ := d.NewDependencyRelation(d.MustParseID("0199e300-0000-7000-8000-000000000005"), w.Ref(), objective.Ref(), d.DependencyStrengthHard, now)
	for _, lifecycle := range []d.ObjectiveLifecycle{d.ObjectiveLifecyclePlanned, d.ObjectiveLifecycleInProgress, d.ObjectiveLifecycleAchieved, d.ObjectiveLifecycleCancelled} {
		objective.Lifecycle = lifecycle
		readiness := d.WorkItemReadiness(w, o, &objective, []d.DependencyEvaluation{{Relation: relation, Satisfied: lifecycle == d.ObjectiveLifecycleAchieved}}, now)
		if readiness.Ready || !d.IsOwnObjectiveHardDependency(w, relation) {
			t.Fatalf("own Objective dependency unexpectedly ready in %s", lifecycle)
		}
	}
	relation.Strength = d.DependencyStrengthAdvisory
	if d.IsOwnObjectiveHardDependency(w, relation) {
		t.Fatal("advisory relationship diagnosed as hard restriction")
	}
}
