package domain

import (
	"strings"
	"time"
)

type RoadmapLifecycle string

const (
	RoadmapLifecycleOpen     RoadmapLifecycle = "open"
	RoadmapLifecycleArchived RoadmapLifecycle = "archived"
)

func (l RoadmapLifecycle) Valid() bool {
	return l == RoadmapLifecycleOpen || l == RoadmapLifecycleArchived
}

type RoadmapScopeKind string

const (
	RoadmapScopeOutcome   RoadmapScopeKind = "outcome"
	RoadmapScopeObjective RoadmapScopeKind = "objective"
)

func (k RoadmapScopeKind) Valid() bool {
	return k == RoadmapScopeOutcome || k == RoadmapScopeObjective
}

type RoadmapPlanScope struct {
	Kind RoadmapScopeKind `json:"kind"`
	ID   ID               `json:"id"`
}

func (s RoadmapPlanScope) Validate(boundary Scope) error {
	if !s.Kind.Valid() {
		return NewError(ErrorCodeRoadmap, "roadmap scope kind is invalid")
	}
	if err := s.ID.Validate(); err != nil {
		return WrapError(ErrorCodeRoadmap, "roadmap scope id is invalid", err)
	}
	if s.Kind == RoadmapScopeOutcome && s.ID != boundary.OutcomeID {
		return NewError(ErrorCodeRoadmap, "outcome-scoped roadmap must use the containing Outcome id")
	}
	return nil
}

type RoadmapNodeType string

const (
	RoadmapNodeReference RoadmapNodeType = "reference"
	RoadmapNodePhase     RoadmapNodeType = "phase"
	RoadmapNodeMilestone RoadmapNodeType = "milestone"
)

func (t RoadmapNodeType) Valid() bool {
	switch t {
	case RoadmapNodeReference, RoadmapNodePhase, RoadmapNodeMilestone:
		return true
	default:
		return false
	}
}

type RoadmapCriterionRef struct {
	OwnerRef    EntityRef `json:"owner_ref"`
	CriterionID ID        `json:"criterion_id"`
}

func (r RoadmapCriterionRef) Validate(scope Scope) error {
	if err := r.OwnerRef.Validate(); err != nil {
		return WrapError(ErrorCodeRoadmap, "roadmap criterion owner is invalid", err)
	}
	if r.OwnerRef.Scope != scope {
		return NewError(ErrorCodeRoadmap, "roadmap criterion owner must belong to the same Outcome")
	}
	switch r.OwnerRef.Kind {
	case EntityKindOutcome, EntityKindObjective, EntityKindWorkItem:
	default:
		return NewError(ErrorCodeRoadmap, "roadmap criterion owner kind is invalid")
	}
	if err := r.CriterionID.Validate(); err != nil {
		return WrapError(ErrorCodeRoadmap, "roadmap criterion id is invalid", err)
	}
	return nil
}

type RoadmapReferenceSnapshot struct {
	TargetRef EntityRef `json:"target_ref"`
	Title     string    `json:"title"`
}

func (s RoadmapReferenceSnapshot) Validate(scope Scope) error {
	if err := s.TargetRef.Validate(); err != nil {
		return WrapError(ErrorCodeRoadmap, "roadmap reference snapshot target is invalid", err)
	}
	if s.TargetRef.Scope != scope {
		return NewError(ErrorCodeRoadmap, "roadmap reference snapshot must belong to the same Outcome")
	}
	switch s.TargetRef.Kind {
	case EntityKindObjective, EntityKindWorkItem:
	default:
		return NewError(ErrorCodeRoadmap, "roadmap reference snapshot target kind is invalid")
	}
	if strings.TrimSpace(s.Title) == "" {
		return NewError(ErrorCodeRoadmap, "roadmap reference snapshot title is required")
	}
	return nil
}

type RoadmapNode struct {
	NodeKey           string                    `json:"node_key"`
	NodeType          RoadmapNodeType           `json:"node_type"`
	ParentNodeKey     string                    `json:"parent_node_key,omitempty"`
	TargetRef         *EntityRef                `json:"target_ref,omitempty"`
	Title             string                    `json:"title"`
	Position          int                       `json:"position"`
	CriterionRefs     []RoadmapCriterionRef      `json:"criterion_refs,omitempty"`
	PlannedStart      *time.Time                 `json:"planned_start,omitempty"`
	PlannedEnd        *time.Time                 `json:"planned_end,omitempty"`
	ReferenceSnapshot *RoadmapReferenceSnapshot `json:"published_reference_snapshot,omitempty"`
}

func (n RoadmapNode) Validate(scope Scope, published bool) error {
	if strings.TrimSpace(n.NodeKey) == "" {
		return NewError(ErrorCodeRoadmap, "roadmap node_key is required")
	}
	if !n.NodeType.Valid() {
		return NewError(ErrorCodeRoadmap, "roadmap node type is invalid")
	}
	if strings.TrimSpace(n.Title) == "" {
		return NewError(ErrorCodeRoadmap, "roadmap node title is required")
	}
	if n.Position < 0 {
		return NewError(ErrorCodeRoadmap, "roadmap node position cannot be negative")
	}
	if n.PlannedStart != nil && n.PlannedStart.IsZero() {
		return NewError(ErrorCodeRoadmap, "roadmap planned_start is invalid")
	}
	if n.PlannedEnd != nil && n.PlannedEnd.IsZero() {
		return NewError(ErrorCodeRoadmap, "roadmap planned_end is invalid")
	}
	if n.PlannedStart != nil && n.PlannedEnd != nil && n.PlannedEnd.Before(*n.PlannedStart) {
		return NewError(ErrorCodeRoadmap, "roadmap planned_end cannot precede planned_start")
	}

	switch n.NodeType {
	case RoadmapNodeReference:
		if n.TargetRef == nil {
			return NewError(ErrorCodeRoadmap, "reference node requires target_ref")
		}
		if err := n.TargetRef.Validate(); err != nil {
			return WrapError(ErrorCodeRoadmap, "roadmap target_ref is invalid", err)
		}
		if n.TargetRef.Scope != scope {
			return NewError(ErrorCodeRoadmap, "roadmap reference target must belong to the same Outcome")
		}
		switch n.TargetRef.Kind {
		case EntityKindObjective, EntityKindWorkItem:
		default:
			return NewError(ErrorCodeRoadmap, "reference node may target only Objective or WorkItem")
		}
		if len(n.CriterionRefs) != 0 {
			return NewError(ErrorCodeRoadmap, "reference node cannot contain criterion_refs")
		}
		if published {
			if n.ReferenceSnapshot == nil {
				return NewError(ErrorCodeRoadmap, "published reference node requires reference snapshot")
			}
			if err := n.ReferenceSnapshot.Validate(scope); err != nil {
				return err
			}
			if n.ReferenceSnapshot.TargetRef != *n.TargetRef {
				return NewError(ErrorCodeRoadmap, "reference snapshot target must match target_ref")
			}
		} else if n.ReferenceSnapshot != nil {
			return NewError(ErrorCodeRoadmap, "draft reference node cannot contain published snapshot")
		}
	case RoadmapNodePhase:
		if n.TargetRef != nil || n.ReferenceSnapshot != nil || len(n.CriterionRefs) != 0 {
			return NewError(ErrorCodeRoadmap, "phase node cannot target operational state or criteria")
		}
	case RoadmapNodeMilestone:
		if n.TargetRef != nil || n.ReferenceSnapshot != nil {
			return NewError(ErrorCodeRoadmap, "milestone node cannot contain target_ref")
		}
		seen := make(map[RoadmapCriterionRef]struct{}, len(n.CriterionRefs))
		for _, ref := range n.CriterionRefs {
			if err := ref.Validate(scope); err != nil {
				return err
			}
			if _, exists := seen[ref]; exists {
				return NewError(ErrorCodeRoadmap, "milestone criterion_refs cannot contain duplicates")
			}
			seen[ref] = struct{}{}
		}
	}
	return nil
}

type RoadmapAfterLink struct {
	NodeKey      string `json:"node_key"`
	AfterNodeKey string `json:"after_node_key"`
}

func (l RoadmapAfterLink) Validate() error {
	if strings.TrimSpace(l.NodeKey) == "" || strings.TrimSpace(l.AfterNodeKey) == "" {
		return NewError(ErrorCodeRoadmap, "roadmap after link requires both node keys")
	}
	if l.NodeKey == l.AfterNodeKey {
		return NewError(ErrorCodeRoadmap, "roadmap node cannot be after itself")
	}
	return nil
}

type RoadmapDraftLifecycle string

const (
	RoadmapDraftOpen      RoadmapDraftLifecycle = "open"
	RoadmapDraftDiscarded RoadmapDraftLifecycle = "discarded"
)

func (l RoadmapDraftLifecycle) Valid() bool {
	return l == RoadmapDraftOpen || l == RoadmapDraftDiscarded
}

type RoadmapDraft struct {
	DraftVersion       uint64                `json:"draft_version"`
	BaseRevisionNumber *uint64               `json:"base_revision_number,omitempty"`
	Lifecycle          RoadmapDraftLifecycle `json:"lifecycle"`
	Nodes              []RoadmapNode         `json:"nodes,omitempty"`
	AfterLinks         []RoadmapAfterLink    `json:"after_links,omitempty"`
	CreatedAt          time.Time             `json:"created_at"`
	UpdatedAt          time.Time             `json:"updated_at"`
	DiscardedAt        *time.Time            `json:"discarded_at,omitempty"`
}

func (d RoadmapDraft) Validate(scope Scope) error {
	if d.DraftVersion == 0 {
		return NewError(ErrorCodeRoadmap, "roadmap draft_version must be at least 1")
	}
	if !d.Lifecycle.Valid() {
		return NewError(ErrorCodeRoadmap, "roadmap draft lifecycle is invalid")
	}
	if d.BaseRevisionNumber != nil && *d.BaseRevisionNumber == 0 {
		return NewError(ErrorCodeRoadmap, "base revision number must be at least 1")
	}
	if d.CreatedAt.IsZero() || d.UpdatedAt.IsZero() {
		return NewError(ErrorCodeRoadmap, "roadmap draft timestamps are required")
	}
	switch d.Lifecycle {
	case RoadmapDraftOpen:
		if d.DiscardedAt != nil {
			return NewError(ErrorCodeRoadmap, "open roadmap draft cannot have discarded_at")
		}
	case RoadmapDraftDiscarded:
		if d.DiscardedAt == nil || d.DiscardedAt.IsZero() {
			return NewError(ErrorCodeRoadmap, "discarded roadmap draft requires discarded_at")
		}
	}
	return validateRoadmapPlanContent(scope, d.Nodes, d.AfterLinks, false)
}

type RoadmapDependencySnapshot struct {
	DependentRef  EntityRef `json:"dependent_ref"`
	PrerequisiteRef EntityRef `json:"prerequisite_ref"`
}

func (s RoadmapDependencySnapshot) Validate(scope Scope) error {
	if err := s.DependentRef.Validate(); err != nil {
		return WrapError(ErrorCodeRoadmap, "dependency snapshot dependent_ref is invalid", err)
	}
	if err := s.PrerequisiteRef.Validate(); err != nil {
		return WrapError(ErrorCodeRoadmap, "dependency snapshot prerequisite_ref is invalid", err)
	}
	if s.DependentRef.Scope != scope || s.PrerequisiteRef.Scope != scope {
		return NewError(ErrorCodeRoadmap, "dependency snapshot refs must belong to the same Outcome")
	}
	if s.DependentRef == s.PrerequisiteRef {
		return NewError(ErrorCodeRoadmap, "dependency snapshot cannot self-reference")
	}
	return nil
}

type RoadmapRevision struct {
	RevisionNumber      uint64                      `json:"revision_number"`
	ContentHash         string                      `json:"content_hash"`
	Nodes               []RoadmapNode               `json:"nodes,omitempty"`
	AfterLinks          []RoadmapAfterLink           `json:"after_links,omitempty"`
	DependencySnapshots []RoadmapDependencySnapshot `json:"dependency_snapshots,omitempty"`
	PublishedBy         ActorRef                    `json:"published_by"`
	PublishedAt         time.Time                   `json:"published_at"`
}

func (r RoadmapRevision) Validate(scope Scope) error {
	if r.RevisionNumber == 0 {
		return NewError(ErrorCodeRoadmap, "roadmap revision_number must be at least 1")
	}
	if strings.TrimSpace(r.ContentHash) == "" {
		return NewError(ErrorCodeRoadmap, "published roadmap revision requires content_hash")
	}
	if err := r.PublishedBy.Validate(); err != nil {
		return WrapError(ErrorCodeRoadmap, "roadmap revision published_by is invalid", err)
	}
	if r.PublishedAt.IsZero() {
		return NewError(ErrorCodeRoadmap, "roadmap revision published_at is required")
	}
	if err := validateRoadmapPlanContent(scope, r.Nodes, r.AfterLinks, true); err != nil {
		return err
	}
	seen := make(map[RoadmapDependencySnapshot]struct{}, len(r.DependencySnapshots))
	for _, snapshot := range r.DependencySnapshots {
		if err := snapshot.Validate(scope); err != nil {
			return err
		}
		if _, exists := seen[snapshot]; exists {
			return NewError(ErrorCodeRoadmap, "roadmap dependency snapshots cannot contain duplicates")
		}
		seen[snapshot] = struct{}{}
	}
	return nil
}

type Roadmap struct {
	ID                 ID               `json:"id"`
	Scope              Scope            `json:"scope"`
	Version            Version          `json:"version"`
	PlanScope          RoadmapPlanScope `json:"plan_scope"`
	Title              string           `json:"title"`
	Lifecycle          RoadmapLifecycle `json:"lifecycle"`
	Draft              *RoadmapDraft    `json:"draft,omitempty"`
	Revisions          []RoadmapRevision `json:"revisions,omitempty"`
	CreatedAt          time.Time        `json:"created_at"`
	UpdatedAt          time.Time        `json:"updated_at"`
	ArchivedAt         *time.Time       `json:"archived_at,omitempty"`
}

func NewRoadmap(id ID, scope Scope, planScope RoadmapPlanScope, title string, now time.Time) (Roadmap, error) {
	value := Roadmap{
		ID:        id,
		Scope:     scope,
		Version:   InitialVersion,
		PlanScope: planScope,
		Title:     strings.TrimSpace(title),
		Lifecycle: RoadmapLifecycleOpen,
		CreatedAt: now.UTC(),
		UpdatedAt: now.UTC(),
	}
	if err := value.Validate(); err != nil {
		return Roadmap{}, err
	}
	return value, nil
}

func (r Roadmap) Ref() EntityRef {
	return EntityRef{Scope: r.Scope, Kind: EntityKindRoadmap, ID: r.ID}
}

func (r Roadmap) Validate() error {
	if err := r.ID.Validate(); err != nil {
		return err
	}
	if err := r.Scope.Validate(); err != nil {
		return err
	}
	if err := r.Version.Validate(); err != nil {
		return err
	}
	if err := r.PlanScope.Validate(r.Scope); err != nil {
		return err
	}
	if strings.TrimSpace(r.Title) == "" {
		return NewError(ErrorCodeRoadmap, "roadmap title is required")
	}
	if !r.Lifecycle.Valid() {
		return NewError(ErrorCodeRoadmap, "roadmap lifecycle is invalid")
	}
	if r.CreatedAt.IsZero() || r.UpdatedAt.IsZero() {
		return NewError(ErrorCodeRoadmap, "roadmap timestamps are required")
	}
	switch r.Lifecycle {
	case RoadmapLifecycleOpen:
		if r.ArchivedAt != nil {
			return NewError(ErrorCodeRoadmap, "open roadmap cannot contain archived_at")
		}
	case RoadmapLifecycleArchived:
		if r.ArivedAt == nil {
			return NewError(ErrorCodeRoadmap, "archived roadmap requires archived_at")
		}
	}
	if r.Draft != nil {
		if err := r.Draft.Validate(r.Scope); err != nil {
			return err
		}
	}
	var expected uint64 = 1
	for _, revision := range r.Revisions {
		if revision.RevisionNumber != expected {
			return NewError(ErrorCodeRoadmap, "roadmap revision history must be contiguous")
		}
		if err := revision.Validate(r.Scope); err != nil {
			return err
		}
		expected++
	}
	return nil
}

func validateRoadmapPlanContent(scope Scope, nodes []RoadmapNode, afterLinks []RoadmapAfterLink, published bool) error {
	byKey := make(map[string]RoadmapNode, len(nodes))
	targets := make(map[EntityRef]string)
	for _, node := range nodes {
		if err := node.Validate(scope, published); err != nil {
			return err
		}
		key := strings.TrimSpace(node.NodeKey)
		if _, exists := byKey[key]; exists {
			return NewError(ErrorCodeRoadmap, "roadmap node_key must be unique within a revision")
		}
		byKey[key] = node
		if node.TargetRef != nil {
			if previous, exists := targets[*node.TargetRef]; exists {
				return NewError(ErrorCodeRoadmap, "operational entity appears more than once in roadmap content: "+previous)
			}
			targets[*node.TargetRef] = key
		}
	}
	for key, node := range byKey {
		if node.ParentNodeKey == "" {
			continue
		}
		if _, exists := byKey[node.ParentNodeKey]; !exists {
			return NewError(ErrorCodeRoadmap, "roadmap parent_node_key references unknown node")
		}
		if node.ParentNodeKey == key {
			return NewError(ErrorCodeRoadmap, "roadmap node cannot parent itself")
		}
	}
	if hasRoadmapCycle(byKey, func(node RoadmapNode) []string {
		if node.ParentNodeKey == "" {
			return nil
		}
		return []string{node.ParentNodeKey}
	}) {
		return NewError(ErrorCodeRoadmap, "roadmap parent graph contains cycle")
	}

	afterAdj := make(map[string][]string, len(nodes))
	seenLinks := make(map[RoadmapAfterLink]struct{}, len(afterLinks))
	for _, link := range afterLinks {
		if err := link.Validate(); err != nil {
			return err
		}
		if _, exists := byKey[link.NodeKey]; !exists {
			return NewError(ErrorCodeRoadmap, "roadmap after link node_key references unknown node")
		}
		if _, exists := byKey[link.AfterNodeKey]; !exists {
			return NewError(ErrorCodeRoadmap, "roadmap after link after_node_key references unknown node")
		}
		if _, exists := seenLinks[link]; exists {
			return NewError(ErrorCodeRoadmap, "roadmap after links cannot contain duplicates")
		}
		seenLinks[link] = struct{}{}
		afterAdj[link.NodeKey] = append(afterAdj[link.NodeKey], link.AfterNodeKey)
	}
	if hasRoadmapCycle(byKey, func(node RoadmapNode) []string {
		return afterAdj[node.NodeKey]
	}) {
		return NewError(ErrorCodeRoadmap, "roadmap after graph contains cycle")
	}
	return nil
}

func hasRoadmapCycle(nodes map[string]RoadmapNode, edges func(RoadmapNode) []string) bool {
	const (
		unseen = iota
		visiting
		visited
	)
	state := make(map[string]int, len(nodes))
	var visit func(string) bool
	visit = func(key string) bool {
		switch state[key] {
		case visiting:
			return true
		case visited:
			return false
		}
		state[key] = visiting
		for _, next := range edges(nodes[key]) {
			if visit(next) {
				return true
			}
		}
		state[key] = visited
		return false
	}
	for key := range nodes {
		if state[key] == unseen && visit(key) {
			return true
		}
	}
	return false
}
