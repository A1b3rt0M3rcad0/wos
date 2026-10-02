package domain

import (
	"strings"
	"time"
)

type EvidenceStance string

const (
	EvidenceStanceSupports    EvidenceStance = "supports"
	EvidenceStanceContradicts EvidenceStance = "contradicts"
	EvidenceStanceContext     EvidenceStance = "context"
)

func (s EvidenceStance) Valid() bool {
	return s == EvidenceStanceSupports || s == EvidenceStanceContradicts || s == EvidenceStanceContext
}

type EvidenceLinkLifecycle string

const (
	EvidenceLinkLifecycleActive    EvidenceLinkLifecycle = "active"
	EvidenceLinkLifecycleRetracted EvidenceLinkLifecycle = "retracted"
)

func (l EvidenceLinkLifecycle) Valid() bool {
	return l == EvidenceLinkLifecycleActive || l == EvidenceLinkLifecycleRetracted
}

type EvidenceLink struct {
	ID          ID                    `json:"id"`
	Scope       Scope                 `json:"scope"`
	Version     Version               `json:"version"`
	EvidenceID  ID                    `json:"evidence_id"`
	TargetRef   EntityRef             `json:"target_ref"`
	CriterionID *ID                   `json:"criterion_id,omitempty"`
	Stance      EvidenceStance        `json:"stance"`
	Rationale   string                `json:"rationale"`
	Lifecycle   EvidenceLinkLifecycle `json:"lifecycle"`
	CreatedAt   time.Time             `json:"created_at"`
	UpdatedAt   time.Time             `json:"updated_at"`
}

func NewEvidenceLink(
	id ID,
	scope Scope,
	evidenceID ID,
	target EntityRef,
	criterionID *ID,
	stance EvidenceStance,
	rationale string,
	now time.Time,
) (EvidenceLink, error) {
	value := EvidenceLink{
		ID:          id,
		Scope:       scope,
		Version:     InitialVersion,
		EvidenceID:  evidenceID,
		TargetRef:   target,
		CriterionID: cloneIDPointerValue(criterionID),
		Stance:      stance,
		Rationale:   strings.TrimSpace(rationale),
		Lifecycle:   EvidenceLinkLifecycleActive,
		CreatedAt:   now.UTC(),
		UpdatedAt:   now.UTC(),
	}
	if err := value.Validate(); err != nil {
		return EvidenceLink{}, err
	}
	return value, nil
}

func (l EvidenceLink) Ref() EntityRef {
	return EntityRef{Scope: l.Scope, Kind: EntityKindEvidenceLink, ID: l.ID}
}

func (l EvidenceLink) Validate() error {
	if err := l.ID.Validate(); err != nil {
		return err
	}
	if err := l.Scope.Validate(); err != nil {
		return err
	}
	if err := l.Version.Validate(); err != nil {
		return err
	}
	if err := l.EvidenceID.Validate(); err != nil {
		return WrapError(ErrorCodeEvidenceLink, "evidence_id is invalid", err)
	}
	if err := l.TargetRef.Validate(); err != nil {
		return WrapError(ErrorCodeEvidenceLink, "target_ref is invalid", err)
	}
	if l.TargetRef.Scope != l.Scope {
		return NewError(ErrorCodeEvidenceLink, "evidence link target must belong to the same Outcome")
	}
	switch l.TargetRef.Kind {
	case EntityKindOutcome, EntityKindObjective, EntityKindWorkItem, EntityKindIssue, EntityKindDecision:
	default:
		return NewError(ErrorCodeEvidenceLink, "evidence link target kind is not supported")
	}
	if l.CriterionID != nil {
		if err := l.CriterionID.Validate(); err != nil {
			return WrapError(ErrorCodeEvidenceLink, "criterion_id is invalid", err)
		}
		switch l.TargetRef.Kind {
		case EntityKindOutcome, EntityKindObjective, EntityKindWorkItem:
		default:
			return NewError(ErrorCodeEvidenceLink, "criterion_id requires outcome, objective or work_item target")
		}
	}
	if !l.Stance.Valid() {
		return NewError(ErrorCodeEvidenceLink, "evidence link stance is invalid")
	}
	if strings.TrimSpace(l.Rationale) == "" {
		return NewError(ErrorCodeEvidenceLink, "evidence link rationale is required")
	}
	if !l.Lifecycle.Valid() {
		return NewError(ErrorCodeEvidenceLink, "evidence link lifecycle is invalid")
	}
	if l.CreatedAt.IsZero() || l.UpdatedAt.IsZero() {
		return NewError(ErrorCodeEvidenceLink, "evidence link timestamps are required")
	}
	return nil
}

func (l *EvidenceLink) Retract(now time.Time) error {
	if l.Lifecycle != EvidenceLinkLifecycleActive {
		return NewError(ErrorCodeInvalidTransition, "only active evidence link can be retracted")
	}
	next, err := nextVersion(l.Version)
	if err != nil {
		return err
	}
	l.Version = next
	l.Lifecycle = EvidenceLinkLifecycleRetracted
	l.UpdatedAt = now.UTC()
	return l.Validate()
}
