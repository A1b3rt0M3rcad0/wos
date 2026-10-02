package domain

import (
	"testing"
	"time"
)

func TestDependencyRelationValidationAndCycle(t *testing.T) {
	scope := Scope{
		NamespaceID: MustParseID("0199f000-0000-7000-8000-000000000001"),
		OutcomeID:   MustParseID("0199f000-0000-7000-8000-000000000002"),
	}
	ref := func(kind EntityKind, suffix string) EntityRef {
		return EntityRef{Scope: scope, Kind: kind, ID: MustParseID("0199f000-0000-7000-8000-" + suffix)}
	}
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	a := ref(EntityKindWorkItem, "000000000011")
	b := ref(EntityKindObjective, "000000000012")
	c := ref(EntityKindWorkItem, "000000000013")

	ab, err := NewDependencyRelation(MustParseID("0199f000-0000-7000-8000-000000000021"), a, b, DependencyStrengthHard, now)
	if err != nil {
		t.Fatal(err)
	}
	bc, err := NewDependencyRelation(MustParseID("0199f000-0000-7000-8000-000000000022"), b, c, DependencyStrengthAdvisory, now)
	if err != nil {
		t.Fatal(err)
	}
	ca, err := NewDependencyRelation(MustParseID("0199f000-0000-7000-8000-000000000023"), c, a, DependencyStrengthHard, now)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateDependencyAcyclic([]Relation{ab, bc}, ca, 100); err == nil {
		t.Fatal("expected dependency cycle")
	} else if code, ok := ErrorCodeOf(err); !ok || code != ErrorCodeDependencyCycle {
		t.Fatalf("cycle code = %q, want %q", code, ErrorCodeDependencyCycle)
	}
}

func TestReadinessUsesHardDependenciesAndNotBefore(t *testing.T) {
	scope := Scope{
		NamespaceID: MustParseID("0199f001-0000-7000-8000-000000000001"),
		OutcomeID:   MustParseID("0199f001-0000-7000-8000-000000000002"),
	}
	now := time.Date(2026, 10, 2, 12, 0, 0, 0, time.UTC)
	outcome, err := NewOutcome(scope.OutcomeID, scope.NamespaceID, "Outcome", "", "Done", PriorityNormal, now)
	if err != nil {
		t.Fatal(err)
	}
	outcome.Lifecycle = OutcomeLifecycleActive
	item, err := NewWorkItem(MustParseID("0199f001-0000-7000-8000-000000000003"), scope, "Work", "", PriorityNormal, WorkItemLifecycleTodo, now)
	if err != nil {
		t.Fatal(err)
	}
	future := now.Add(time.Minute)
	item.NotBefore = &future

	target := EntityRef{Scope: scope, Kind: EntityKindObjective, ID: MustParseID("0199f001-0000-7000-8000-000000000004")}
	dep, err := NewDependencyRelation(MustParseID("0199f001-0000-7000-8000-000000000005"), item.Ref(), target, DependencyStrengthHard, now)
	if err != nil {
		t.Fatal(err)
	}

	result := WorkItemReadiness(item, outcome, nil, []DependencyEvaluation{{Relation: dep, Satisfied: false}}, now)
	if result.Ready {
		t.Fatal("work item should not be ready")
	}
	if len(result.Reasons) != 2 {
		t.Fatalf("reasons = %v, want hard dependency + not_before", result.Reasons)
	}

	at := future
	result = WorkItemReadiness(item, outcome, nil, []DependencyEvaluation{{Relation: dep, Satisfied: true}}, at)
	if !result.Ready {
		t.Fatalf("work item should be ready at not_before, reasons=%v", result.Reasons)
	}
}
