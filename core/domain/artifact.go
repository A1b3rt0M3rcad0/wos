package domain

import (
	"strings"
	"time"
)

type ArtifactLifecycle string

const (
	ArtifactLifecycleRegistered ArtifactLifecycle = "registered"
	ArtifactLifecycleWithdrawn  ArtifactLifecycle = "withdrawn"
)

func (l ArtifactLifecycle) Valid() bool {
	return l == ArtifactLifecycleRegistered || l == ArtifactLifecycleWithdrawn
}

// Artifact is an immutable description of a material deliverable. WOS stores
// the reference and metadata only; URI content is never fetched by the domain.
type Artifact struct {
	ID               ID                `json:"id"`
	Scope            Scope             `json:"scope"`
	Version          Version           `json:"version"`
	ArtifactType     string            `json:"artifact_type"`
	Name             string            `json:"name"`
	URI              string            `json:"uri"`
	MediaType        string            `json:"media_type,omitempty"`
	Checksum         string            `json:"checksum,omitempty"`
	SourceVersion    string            `json:"source_version,omitempty"`
	ProducerRef      ActorRef          `json:"producer_ref"`
	ProducedAt       *time.Time        `json:"produced_at,omitempty"`
	RegisteredAt     time.Time         `json:"registered_at"`
	Lifecycle        ArtifactLifecycle `json:"lifecycle"`
	WithdrawalReason string            `json:"withdrawal_reason,omitempty"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

func NewArtifact(
	id ID,
	scope Scope,
	artifactType, name, uri, mediaType, checksum, sourceVersion string,
	producer ActorRef,
	producedAt *time.Time,
	now time.Time,
) (Artifact, error) {
	value := Artifact{
		ID:            id,
		Scope:         scope,
		Version:       InitialVersion,
		ArtifactType:  strings.TrimSpace(artifactType),
		Name:          strings.TrimSpace(name),
		URI:           strings.TrimSpace(uri),
		MediaType:     strings.TrimSpace(mediaType),
		Checksum:      strings.TrimSpace(checksum),
		SourceVersion: strings.TrimSpace(sourceVersion),
		ProducerRef:   producer,
		ProducedAt:    cloneTimePointer(producedAt),
		RegisteredAt:  now.UTC(),
		Lifecycle:     ArtifactLifecycleRegistered,
		UpdatedAt:     now.UTC(),
	}
	if err := value.Validate(); err != nil {
		return Artifact{}, err
	}
	return value, nil
}

func (a Artifact) Ref() EntityRef {
	return EntityRef{Scope: a.Scope, Kind: EntityKindArtifact, ID: a.ID}
}

func (a Artifact) Validate() error {
	if err := a.ID.Validate(); err != nil {
		return err
	}
	if err := a.Scope.Validate(); err != nil {
		return err
	}
	if err := a.Version.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(a.ArtifactType) == "" {
		return NewError(ErrorCodeArtifact, "artifact_type is required")
	}
	if strings.TrimSpace(a.Name) == "" {
		return NewError(ErrorCodeArtifact, "artifact name is required")
	}
	if strings.TrimSpace(a.URI) == "" {
		return NewError(ErrorCodeArtifact, "artifact uri is required")
	}
	if err := a.ProducerRef.Validate(); err != nil {
		return WrapError(ErrorCodeArtifact, "artifact producer_ref is invalid", err)
	}
	if a.ProducedAt != nil && a.ProducedAt.IsZero() {
		return NewError(ErrorCodeArtifact, "artifact produced_at cannot be zero")
	}
	if a.RegisteredAt.IsZero() || a.UpdatedAt.IsZero() {
		return NewError(ErrorCodeArtifact, "artifact timestamps are required")
	}
	if !a.Lifecycle.Valid() {
		return NewError(ErrorCodeArtifact, "artifact lifecycle is invalid")
	}
	if a.Lifecycle == ArtifactLifecycleRegistered && strings.TrimSpace(a.WithdrawalReason) != "" {
		return NewError(ErrorCodeArtifact, "registered artifact cannot have withdrawal_reason")
	}
	if a.Lifecycle == ArtifactLifecycleWithdrawn && strings.TrimSpace(a.WithdrawalReason) == "" {
		return NewError(ErrorCodeArtifact, "withdrawn artifact requires withdrawal_reason")
	}
	return nil
}

func (a *Artifact) Withdraw(reason string, now time.Time) error {
	if a.Lifecycle != ArtifactLifecycleRegistered {
		return NewError(ErrorCodeInvalidTransition, "only registered artifact can be withdrawn")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return NewError(ErrorCodeArtifact, "artifact withdrawal reason is required")
	}
	next, err := nextVersion(a.Version)
	if err != nil {
		return err
	}
	a.Version = next
	a.Lifecycle = ArtifactLifecycleWithdrawn
	a.WithdrawalReason = reason
	a.UpdatedAt = now.UTC()
	return a.Validate()
}

func cloneTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	cloned := value.UTC()
	return &cloned
}
