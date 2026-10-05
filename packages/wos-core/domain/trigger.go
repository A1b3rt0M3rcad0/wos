package domain

import (
	"fmt"
	"reflect"
	"strings"
	"time"
)

type EventPredicate struct {
	Field string           `json:"field,omitempty"`
	Op    string           `json:"op,omitempty"`
	Value any              `json:"value,omitempty"`
	All   []EventPredicate `json:"all,omitempty"`
	Any   []EventPredicate `json:"any,omitempty"`
}

var predicateFields = map[string]bool{"event_type": true, "namespace_id": true, "outcome_id": true, "aggregate.kind": true, "aggregate.id": true, "principal_id": true, "actor.kind": true, "actor.provider": true, "actor.id": true}

func (p EventPredicate) Validate() error { count := 0; return p.validate(1, &count) }
func (p EventPredicate) validate(depth int, count *int) error {
	if depth > 3 {
		return NewError(ErrorCodeInvalidArgument, "predicate depth exceeds 3")
	}
	*count++
	if *count > 20 {
		return NewError(ErrorCodeInvalidArgument, "predicate exceeds 20 clauses")
	}
	if p.Value == nil && p.Field == "" && p.Op == "" && len(p.All) == 0 && len(p.Any) == 0 {
		return nil
	}
	if len(p.All) > 0 || len(p.Any) > 0 {
		if p.Field != "" || p.Op != "" || p.Value != nil || (len(p.All) > 0 && len(p.Any) > 0) {
			return NewError(ErrorCodeInvalidArgument, "predicate must use exactly one composition")
		}
		children := p.All
		if len(p.Any) > 0 {
			children = p.Any
		}
		for _, child := range children {
			if err := child.validate(depth+1, count); err != nil {
				return err
			}
		}
		return nil
	}
	if !predicateFields[p.Field] {
		return NewError(ErrorCodeInvalidArgument, "predicate field is not public")
	}
	switch p.Op {
	case "eq", "neq":
		if _, ok := p.Value.(string); !ok {
			return NewError(ErrorCodeInvalidArgument, "equality predicates require a string")
		}
	case "exists":
		if _, ok := p.Value.(bool); !ok {
			return NewError(ErrorCodeInvalidArgument, "exists predicate requires a boolean")
		}
	case "in":
		v := reflect.ValueOf(p.Value)
		if !v.IsValid() || v.Kind() != reflect.Slice || v.Len() > 20 {
			return NewError(ErrorCodeInvalidArgument, "in predicate requires at most 20 strings")
		}
		for i := 0; i < v.Len(); i++ {
			if _, ok := v.Index(i).Interface().(string); !ok {
				return NewError(ErrorCodeInvalidArgument, "in predicate requires strings")
			}
		}
	default:
		return NewError(ErrorCodeInvalidArgument, "unknown predicate operator")
	}
	return nil
}
func (p EventPredicate) Matches(e DomainEvent) bool {
	if len(p.All) > 0 {
		for _, c := range p.All {
			if !c.Matches(e) {
				return false
			}
		}
		return true
	}
	if len(p.Any) > 0 {
		for _, c := range p.Any {
			if c.Matches(e) {
				return true
			}
		}
		return false
	}
	if p.Field == "" {
		return true
	}
	values := map[string]string{"event_type": e.EventType, "namespace_id": e.NamespaceID.String(), "outcome_id": e.OutcomeID.String(), "aggregate.kind": e.AggregateRef.Kind.String(), "aggregate.id": e.AggregateRef.ID.String(), "principal_id": e.PrincipalID, "actor.kind": string(e.Actor.Kind), "actor.provider": e.Actor.Provider, "actor.id": e.Actor.ID}
	value, exists := values[p.Field]
	switch p.Op {
	case "exists":
		return exists == p.Value
	case "eq":
		return exists && value == p.Value
	case "neq":
		return exists && value != p.Value
	case "in":
		v := reflect.ValueOf(p.Value)
		for i := 0; i < v.Len(); i++ {
			if exists && v.Index(i).Interface() == value {
				return true
			}
		}
	}
	return false
}

type Trigger struct {
	ID                ID             `json:"id"`
	Scope             Scope          `json:"scope"`
	Version           Version        `json:"version"`
	Name              string         `json:"name"`
	Enabled           bool           `json:"enabled"`
	EventTypes        []string       `json:"event_types"`
	Predicate         EventPredicate `json:"predicate"`
	TargetEndpointIDs []ID           `json:"target_endpoint_ids"`
	SignalType        string         `json:"signal_type"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

func (t Trigger) Ref() EntityRef { return EntityRef{Scope: t.Scope, Kind: EntityKindTrigger, ID: t.ID} }
func (t Trigger) Validate() error {
	if err := t.Ref().Validate(); err != nil {
		return err
	}
	if err := t.Version.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(t.Name) == "" || len(t.Name) > 512 || len(t.EventTypes) == 0 || len(t.EventTypes) > 32 || len(t.TargetEndpointIDs) > 5 || !validEventType(t.SignalType) {
		return NewError(ErrorCodeInvalidArgument, "invalid trigger name, event types, targets or signal type")
	}
	seen := map[string]bool{}
	for _, typ := range t.EventTypes {
		if !validEventType(typ) || seen[typ] {
			return NewError(ErrorCodeInvalidArgument, "event types must be unique public fact names")
		}
		seen[typ] = true
	}
	seen = map[string]bool{}
	for _, id := range t.TargetEndpointIDs {
		if err := id.Validate(); err != nil {
			return err
		}
		if seen[id.String()] {
			return NewError(ErrorCodeInvalidArgument, "duplicate endpoint")
		}
		seen[id.String()] = true
	}
	if t.CreatedAt.IsZero() || t.UpdatedAt.IsZero() {
		return NewError(ErrorCodeInvalidArgument, "trigger timestamps required")
	}
	return t.Predicate.Validate()
}
func (t Trigger) Matches(e DomainEvent) bool {
	if !t.Enabled || t.Scope != e.Scope() {
		return false
	}
	for _, typ := range t.EventTypes {
		if typ == e.EventType {
			return t.Predicate.Matches(e)
		}
	}
	return false
}
func (t *Trigger) Replace(name string, eventTypes []string, predicate EventPredicate, targets []ID, signal string, now time.Time) error {
	candidate := *t
	candidate.Name = name
	candidate.EventTypes = append([]string{}, eventTypes...)
	candidate.Predicate = predicate
	candidate.TargetEndpointIDs = append([]ID{}, targets...)
	candidate.SignalType = signal
	candidate.Version++
	candidate.UpdatedAt = now.UTC()
	if err := candidate.Validate(); err != nil {
		return err
	}
	*t = candidate
	return nil
}
func (t *Trigger) SetEnabled(enabled bool, now time.Time) error {
	if t.Enabled == enabled {
		return nil
	}
	t.Enabled = enabled
	t.Version++
	t.UpdatedAt = now.UTC()
	return t.Validate()
}
func (t Trigger) String() string { return fmt.Sprintf("trigger %s version %d", t.ID, t.Version) }
