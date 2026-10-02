package domain

import (
	"strings"
	"time"
)

type BlockerLifecycle string

const (
	BlockerLifecycleActive    BlockerLifecycle = "active"
	BlockerLifecycleResolved  BlockerLifecycle = "resolved"
	BlockerLifecycleCancelled BlockerLifecycle = "cancelled"
)

func (l BlockerLifecycle) Valid() bool {
	return l == BlockerLifecycleActive || l == BlockerLifecycleResolved || l == BlockerLifecycleCancelled
}

type BlockerPropagation string

const (
	BlockerPropagationDirect  BlockerPropagation = "direct"
	BlockerPropagationSubtree BlockerPropagation = "subtree"
)

func (p BlockerPropagation) Valid() bool {
	return p == BlockerPropagationDirect || p == BlockerPropagationSubtree
}

type ExternalCause struct {
	Provider    string `json:"provider"`
	ID          string `json:"id,omitempty"`
	URI         string `json:"uri,omitempty"`
	Description string `json:"description"`
}

func (c ExternalCause) Validate() error {
	if strings.TrimSpace(c.Provider) == "" {
		return NewError(ErrorCodeBlocker, "external cause provider is required")
	}
	if strings.TrimSpace(c.ID) == "" && strings.TrimSpace(c.URI) == "" {
		return NewError(ErrorCodeBlocker, "external cause requires id or uri")
	}
	if strings.TrimSpace(c.Description) == "" {
		return NewError(ErrorCodeBlocker, "external cause description is required")
	}
	return nil
}

type Blocker struct {
	ID                ID                 `json:"id"`
	Scope             Scope              `json:"scope"`
	Version           Version            `json:"version"`
	BlockedRef        EntityRef          `json:"blocked_ref"`
	CauseRef          *EntityRef         `json:"cause_ref,omitempty"`
	ExternalCause     *ExternalCause     `json:"external_cause,omitempty"`
	Description       string             `json:"description,omitempty"`
	Lifecycle         BlockerLifecycle   `json:"lifecycle"`
	Propagation       BlockerPropagation `json:"propagation"`
	ResolvedAt        *time.Time         `json:"resolved_at,omitempty"`
	ResolutionSummary string             `json:"resolution_summary,omitempty"`
	CreatedAt         time.Time          `json:"created_at"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

func NewBlocker(
	id ID,
	blockedRef EntityRef,
	causeRef *EntityRef,
	externalCause *ExternalCause,
	description string,
	propagation BlockerPropagation,
	now time.Time,
) (Blocker, error) {
	if propagation == "" {
		switch blockedRef.Kind {
		case EntityKindOutcome, EntityKindObjective:
			propagation = BlockerPropagationSubtree
		case EntityKindWorkItem:
			propagation = BlockerPropagationDirect
		}
	}
	value := Blocker{
		ID:            id,
		Scope:         blockedRef.Scope,
		Version:       InitialVersion,
		BlockedRef:    blockedRef,
		CauseRef:      cloneEntityRef(causeRef),
		ExternalCause: cloneExternalCause(externalCause),
		Description:   strings.TrimSpace(description),
		Lifecycle:     BlockerLifecycleActive,
		Propagation:   propagation,
		CreatedAt:     now.UTC(),
		UpdatedAt:     now.UTC(),
	}
	if err := value.Validate(); err != nil {
		return Blocker{}, err
	}
	return value, nil
}

func (b Blocker) Ref() EntityRef {
	return EntityRef{Scope: b.Scope, Kind: EntityKindBlocker, ID: b.ID}
}

func (b Blocker) Validate() error {
	if err := b.ID.Validate(); err != nil {
		return err
	}
	if err := b.Scope.Validate(); err != nil {
		return err
	}
	if err := b.Version.Validate(); err != nil {
		return err
	}
	if err := b.BlockedRef.Validate(); err != nil {
		return WrapError(ErrorCodeBlocker, "blocked_ref is invalid", err)
	}
	if b.BlockedRef.Scope != b.Scope {
		return NewError(ErrorCodeBlocker, "blocked_ref must belong to blocker Outcome")
	}
	if !blockerTargetKind(b.BlockedRef.Kind) {
		return NewError(ErrorCodeBlocker, "blocker target must be outcome, objective or work_item")
	}
	if !b.Lifecycle.Valid() {
		return NewError(ErrorCodeBlocker, "blocker lifecycle is invalid")
	}
	if !b.Propagation.Valid() {
		return NewError(ErrorCodeBlocker, "blocker propagation is invalid")
	}
	if b.BlockedRef.Kind == EntityKindWorkItem && b.Propagation != BlockerPropagationDirect {
		return NewError(ErrorCodeBlocker, "work_item blocker propagation must be direct")
	}
	if b.BlockedRef.Kind == EntityKindOutcome && b.Propagation != BlockerPropagationSubtree {
		return NewError(ErrorCodeBlocker, "outcome blocker propagation must be subtree")
	}
	if b.CauseRef != nil {
		if err := b.CauseRef.Validate(); err != nil {
			return WrapError(ErrorCodeBlocker, "cause_ref is invalid", err)
		}
		if b.CauseRef.Scope != b.Scope {
			return NewError(ErrorCodeBlocker, "cause_ref must belong to blocker Outcome")
		}
		if !blockerCauseKind(b.CauseRef.Kind) {
			return NewError(ErrorCodeBlocker, "blocker cause must be issue, work_item, objective or decision")
		}
	}
	if b.ExternalCause != nil {
		if err := b.ExternalCause.Validate(); err != nil {
			return err
		}
	}
	if b.CauseRef != nil && b.ExternalCause != nil {
		return NewError(ErrorCodeBlocker, "blocker cannot have both cause_ref and external_cause")
	}
	if b.CauseRef == nil && b.ExternalCause == nil && strings.TrimSpace(b.Description) == "" {
		return NewError(ErrorCodeBlocker, "blocker requires a cause or concrete description")
	}
	switch b.Lifecycle {
	case BlockerLifecycleActive:
		if b.ResolvedAt != nil || strings.TrimSpace(b.ResolutionSummary) != "" {
			return NewError(ErrorCodeBlocker, "active blocker cannot contain resolution fields")
		}
	case BlockerLifecycleResolved, BlockerLifecycleCancelled:
		if b.ResolvedAt == nil || b.ResolvedAt.IsZero() {
			return NewError(ErrorCodeBlocker, "closed blocker requires resolved_at")
		}
		if strings.TrimSpace(b.ResolutionSummary) == "" {
			return NewError(ErrorCodeBlocker, "closed blocker requires resolution_summary")
		}
	}
	if b.CreatedAt.IsZero() || b.UpdatedAt.IsZero() {
		return NewError(ErrorCodeBlocker, "blocker timestamps are required")
	}
	return nil
}

func (b *Blocker) Resolve(summary string, now time.Time) error {
	if b.Lifecycle != BlockerLifecycleActive {
		return NewError(ErrorCodeInvalidTransition, "only active blocker can be resolved")
	}
	return b.close(BlockerLifecycleResolved, summary, now)
}

func (b *Blocker) Cancel(summary string, now time.Time) error {
	if b.Lifecycle != BlockerLifecycleActive {
		return NewError(ErrorCodeInvalidTransition, "only active blocker can be cancelled")
	}
	return b.close(BlockerLifecycleCancelled, summary, now)
}

func (b *Blocker) close(lifecycle BlockerLifecycle, summary string, now time.Time) error {
	summary = strings.TrimSpace(summary)
	if summary == "" {
		return NewError(ErrorCodeBlocker, "blocker resolution summary is required")
	}
	next, err := nextVersion(b.Version)
	if err != nil {
		return err
	}
	resolvedAt := now.UTC()
	b.Version = next
	b.Lifecycle = lifecycle
	b.ResolvedAt = &resolvedAt
	b.ResolutionSummary = summary
	b.UpdatedAt = resolvedAt
	return b.Validate()
}

func blockerTargetKind(kind EntityKind) bool {
	return kind == EntityKindOutcome || kind == EntityKindObjective || kind == EntityKindWorkItem
}

func blockerCauseKind(kind EntityKind) bool {
	switch kind {
	case EntityKindIssue, EntityKindWorkItem, EntityKindObjective, EntityKindDecision:
		return true
	default:
		return false
	}
}

func cloneEntityRef(value *EntityRef) *EntityRef {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneExternalCause(value *ExternalCause) *ExternalCause {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}
