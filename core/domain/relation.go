package domain

import (
	"fmt"
	"strings"
	"time"
)

type RelationType string

const (
	RelationTypeDependsOn   RelationType = "depends_on"
	RelationTypeRelatesTo   RelationType = "relates_to"
	RelationTypeProduces    RelationType = "produces"
	RelationTypeDerivedFrom RelationType = "derived_from"
)

func (t RelationType) Valid() bool {
	switch t {
	case RelationTypeDependsOn, RelationTypeRelatesTo, RelationTypeProduces, RelationTypeDerivedFrom:
		return true
	default:
		return false
	}
}

type DependencyStrength string

const (
	DependencyStrengthHard     DependencyStrength = "hard"
	DependencyStrengthAdvisory DependencyStrength = "advisory"
)

func (s DependencyStrength) Valid() bool {
	return s == DependencyStrengthHard || s == DependencyStrengthAdvisory
}

type DependencySatisfaction string

const DependencySatisfactionTargetCompleted DependencySatisfaction = "target_completed"

func (s DependencySatisfaction) Valid() bool {
	return s == DependencySatisfactionTargetCompleted
}

type RelationLifecycle string

const (
	RelationLifecycleActive  RelationLifecycle = "active"
	RelationLifecycleRemoved RelationLifecycle = "removed"
)

func (l RelationLifecycle) Valid() bool {
	return l == RelationLifecycleActive || l == RelationLifecycleRemoved
}

type Relation struct {
	ID            ID                     `json:"id"`
	Scope         Scope                  `json:"scope"`
	Version       Version                `json:"version"`
	SourceRef     EntityRef              `json:"source_ref"`
	RelationType  RelationType           `json:"relation_type"`
	TargetRef     EntityRef              `json:"target_ref"`
	Strength      DependencyStrength     `json:"strength,omitempty"`
	Satisfaction  DependencySatisfaction `json:"satisfaction,omitempty"`
	Lifecycle     RelationLifecycle      `json:"lifecycle"`
	RemovalReason string                 `json:"removal_reason,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
}

func NewDependencyRelation(id ID, source, target EntityRef, strength DependencyStrength, now time.Time) (Relation, error) {
	r := Relation{
		ID:           id,
		Scope:        source.Scope,
		Version:      InitialVersion,
		SourceRef:    source,
		RelationType: RelationTypeDependsOn,
		TargetRef:    target,
		Strength:     strength,
		Satisfaction: DependencySatisfactionTargetCompleted,
		Lifecycle:    RelationLifecycleActive,
		CreatedAt:    now.UTC(),
		UpdatedAt:    now.UTC(),
	}
	if err := r.Validate(); err != nil {
		return Relation{}, err
	}
	return r, nil
}

func (r Relation) Ref() EntityRef {
	return EntityRef{Scope: r.Scope, Kind: EntityKindRelation, ID: r.ID}
}

func (r Relation) Validate() error {
	if err := r.ID.Validate(); err != nil {
		return err
	}
	if err := r.Scope.Validate(); err != nil {
		return err
	}
	if err := r.Version.Validate(); err != nil {
		return err
	}
	if err := r.SourceRef.Validate(); err != nil {
		return WrapError(ErrorCodeInvalidRelation, "source_ref is invalid", err)
	}
	if err := r.TargetRef.Validate(); err != nil {
		return WrapError(ErrorCodeInvalidRelation, "target_ref is invalid", err)
	}
	if r.SourceRef.Scope != r.Scope || r.TargetRef.Scope != r.Scope {
		return NewError(ErrorCodeInvalidRelation, "relation endpoints must belong to the relation Outcome")
	}
	if !r.RelationType.Valid() {
		return NewError(ErrorCodeInvalidRelation, "relation_type is invalid")
	}
	if !r.Lifecycle.Valid() {
		return NewError(ErrorCodeInvalidRelation, "relation lifecycle is invalid")
	}
	if r.SourceRef.Kind == r.TargetRef.Kind && r.SourceRef.ID == r.TargetRef.ID {
		return NewError(ErrorCodeInvalidRelation, "relation cannot target its own source entity")
	}
	if r.RelationType == RelationTypeDependsOn {
		if !dependencyEndpointKind(r.SourceRef.Kind) || !dependencyEndpointKind(r.TargetRef.Kind) {
			return NewError(ErrorCodeInvalidRelation, "depends_on endpoints must be objective or work_item")
		}
		if !r.Strength.Valid() {
			return NewError(ErrorCodeInvalidRelation, "dependency strength must be hard or advisory")
		}
		if !r.Satisfaction.Valid() {
			return NewError(ErrorCodeInvalidRelation, "dependency satisfaction is invalid")
		}
	}
	if r.Lifecycle == RelationLifecycleRemoved && strings.TrimSpace(r.RemovalReason) == "" {
		return NewError(ErrorCodeInvalidRelation, "removed relation requires a removal reason")
	}
	if r.CreatedAt.IsZero() || r.UpdatedAt.IsZero() {
		return NewError(ErrorCodeInvalidRelation, "relation timestamps are required")
	}
	return nil
}

func (r *Relation) Remove(reason string, now time.Time) error {
	if r.Lifecycle != RelationLifecycleActive {
		return NewError(ErrorCodeInvalidTransition, "only active relation can be removed")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return NewError(ErrorCodeInvalidArgument, "relation removal reason is required")
	}
	next, err := nextVersion(r.Version)
	if err != nil {
		return err
	}
	r.Version = next
	r.Lifecycle = RelationLifecycleRemoved
	r.RemovalReason = reason
	r.UpdatedAt = now.UTC()
	return r.Validate()
}

func dependencyEndpointKind(kind EntityKind) bool {
	return kind == EntityKindObjective || kind == EntityKindWorkItem
}

const DefaultDependencyGraphLimit = 10000

func ValidateDependencyAcyclic(existing []Relation, candidate Relation, maxEdges int) error {
	if candidate.RelationType != RelationTypeDependsOn || candidate.Lifecycle != RelationLifecycleActive {
		return NewError(ErrorCodeInvalidRelation, "candidate must be an active depends_on relation")
	}
	if err := candidate.Validate(); err != nil {
		return err
	}
	if maxEdges <= 0 {
		maxEdges = DefaultDependencyGraphLimit
	}

	type node struct {
		kind EntityKind
		id   ID
	}
	key := func(ref EntityRef) node { return node{kind: ref.Kind, id: ref.ID} }

	adj := make(map[node][]node)
	edgeCount := 0
	for _, relation := range existing {
		if relation.Lifecycle != RelationLifecycleActive || relation.RelationType != RelationTypeDependsOn {
			continue
		}
		if relation.Scope != candidate.Scope {
			continue
		}
		edgeCount++
		if edgeCount >= maxEdges {
			return NewError(ErrorCodeGraphLimitExceeded, "dependency graph validation limit exceeded")
		}
		adj[key(relation.SourceRef)] = append(adj[key(relation.SourceRef)], key(relation.TargetRef))
	}
	adj[key(candidate.SourceRef)] = append(adj[key(candidate.SourceRef)], key(candidate.TargetRef))

	start := key(candidate.TargetRef)
	goal := key(candidate.SourceRef)
	visited := make(map[node]bool)
	parent := make(map[node]node)
	stack := []node{start}
	visited[start] = true
	steps := 0

	for len(stack) > 0 {
		steps++
		if steps > maxEdges {
			return NewError(ErrorCodeGraphLimitExceeded, "dependency graph validation limit exceeded")
		}
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		if current == goal {
			path := []node{current}
			for current != start {
				current = parent[current]
				path = append(path, current)
			}
			parts := make([]string, 0, len(path)+1)
			for i := len(path) - 1; i >= 0; i-- {
				parts = append(parts, fmt.Sprintf("%s:%s", path[i].kind, path[i].id))
			}
			parts = append(parts, fmt.Sprintf("%s:%s", candidate.TargetRef.Kind, candidate.TargetRef.ID))
			return NewError(ErrorCodeDependencyCycle, "dependency would create cycle: "+strings.Join(parts, " -> "))
		}
		for _, next := range adj[current] {
			if visited[next] {
				continue
			}
			visited[next] = true
			parent[next] = current
			stack = append(stack, next)
		}
	}
	return nil
}
