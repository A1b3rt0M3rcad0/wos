package domain

import (
	"testing"
	"time"
)

func TestTriggerPredicatesAreBoundedAndDeterministic(t *testing.T) {
	id := MustParseID("0199d090-0000-7000-8000-000000000001")
	scope := Scope{NamespaceID: id, OutcomeID: id}
	event := DomainEvent{NamespaceID: id, OutcomeID: id, EventType: "work_item.created", AggregateRef: EntityRef{Scope: scope, Kind: EntityKindWorkItem, ID: id}}
	predicate := EventPredicate{All: []EventPredicate{{Field: "aggregate.kind", Op: "eq", Value: "work_item"}, {Field: "event_type", Op: "in", Value: []string{"work_item.created", "work_item.completed"}}}}
	if err := predicate.Validate(); err != nil || !predicate.Matches(event) {
		t.Fatal("valid predicate failed", err)
	}
	invalid := []EventPredicate{{Field: "metadata.secret", Op: "eq", Value: "x"}, {Field: "aggregate.kind", Op: "eval", Value: "x"}, {Field: "event_type", Op: "eq", Value: nil}, {All: []EventPredicate{{All: []EventPredicate{{All: []EventPredicate{{Field: "event_type", Op: "eq", Value: "x"}}}}}}}}
	for _, p := range invalid {
		if p.Validate() == nil {
			t.Fatal("invalid predicate accepted", p)
		}
	}
	trigger := Trigger{ID: id, Scope: scope, Version: 1, Name: "Notify", Enabled: true, EventTypes: []string{"work_item.created"}, Predicate: predicate, SignalType: "product.ready", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if trigger.Validate() != nil || !trigger.Matches(event) {
		t.Fatal("valid trigger failed")
	}
	trigger.Enabled = false
	if trigger.Matches(event) {
		t.Fatal("disabled trigger fired")
	}
}
