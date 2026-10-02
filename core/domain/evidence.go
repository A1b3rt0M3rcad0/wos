package domain

import (
	"bytes"
	"encoding/json"
	"strings"
	"time"
)

type EvidenceType string

const (
	EvidenceTypeMeasurement        EvidenceType = "measurement"
	EvidenceTypeTestResult         EvidenceType = "test_result"
	EvidenceTypeInspection         EvidenceType = "inspection"
	EvidenceTypeAttestation        EvidenceType = "attestation"
	EvidenceTypeSource             EvidenceType = "source"
	EvidenceTypeExternalEvaluation EvidenceType = "external_evaluation"
)

func (t EvidenceType) Valid() bool {
	switch t {
	case EvidenceTypeMeasurement,
		EvidenceTypeTestResult,
		EvidenceTypeInspection,
		EvidenceTypeAttestation,
		EvidenceTypeSource,
		EvidenceTypeExternalEvaluation:
		return true
	default:
		return false
	}
}

type EvidenceLifecycle string

const (
	EvidenceLifecycleRegistered EvidenceLifecycle = "registered"
	EvidenceLifecycleRetracted  EvidenceLifecycle = "retracted"
)

func (l EvidenceLifecycle) Valid() bool {
	return l == EvidenceLifecycleRegistered || l == EvidenceLifecycleRetracted
}

type Measurement struct {
	Value      json.RawMessage `json:"value"`
	Unit       string          `json:"unit,omitempty"`
	Method     string          `json:"method,omitempty"`
	Conditions json.RawMessage `json:"conditions,omitempty"`
}

func (m Measurement) Validate() error {
	if len(m.Value) == 0 || !json.Valid(m.Value) || bytes.Equal(bytes.TrimSpace(m.Value), []byte("null")) {
		return NewError(ErrorCodeEvidence, "measurement value must be valid non-null JSON")
	}
	if len(m.Conditions) > 0 && !json.Valid(m.Conditions) {
		return NewError(ErrorCodeEvidence, "measurement conditions must be valid JSON")
	}
	return nil
}

type Evidence struct {
	ID               ID                `json:"id"`
	Scope            Scope             `json:"scope"`
	Version          Version           `json:"version"`
	EvidenceType     EvidenceType      `json:"evidence_type"`
	Description      string            `json:"description"`
	SourceRef        ExternalReference `json:"source_ref"`
	ProducerRef      ActorRef          `json:"producer_ref"`
	CapturedAt       time.Time         `json:"captured_at"`
	RegisteredAt     time.Time         `json:"registered_at"`
	ArtifactID       *ID               `json:"artifact_id,omitempty"`
	Measurement      *Measurement      `json:"measurement,omitempty"`
	SourceVersion    string            `json:"source_version,omitempty"`
	Checksum         string            `json:"checksum,omitempty"`
	Lifecycle        EvidenceLifecycle `json:"lifecycle"`
	RetractionReason string            `json:"retraction_reason,omitempty"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

func NewEvidence(
	id ID,
	scope Scope,
	evidenceType EvidenceType,
	description string,
	source ExternalReference,
	producer ActorRef,
	capturedAt time.Time,
	artifactID *ID,
	measurement *Measurement,
	sourceVersion, checksum string,
	now time.Time,
) (Evidence, error) {
	value := Evidence{
		ID:            id,
		Scope:         scope,
		Version:       InitialVersion,
		EvidenceType:  evidenceType,
		Description:   strings.TrimSpace(description),
		SourceRef:     source,
		ProducerRef:   producer,
		CapturedAt:    capturedAt.UTC(),
		RegisteredAt:  now.UTC(),
		ArtifactID:    cloneIDPointerValue(artifactID),
		Measurement:   cloneMeasurement(measurement),
		SourceVersion: strings.TrimSpace(sourceVersion),
		Checksum:      strings.TrimSpace(checksum),
		Lifecycle:     EvidenceLifecycleRegistered,
		UpdatedAt:     now.UTC(),
	}
	if err := value.Validate(); err != nil {
		return Evidence{}, err
	}
	return value, nil
}

func (e Evidence) Ref() EntityRef {
	return EntityRef{Scope: e.Scope, Kind: EntityKindEvidence, ID: e.ID}
}

func (e Evidence) Validate() error {
	if err := e.ID.Validate(); err != nil {
		return err
	}
	if err := e.Scope.Validate(); err != nil {
		return err
	}
	if err := e.Version.Validate(); err != nil {
		return err
	}
	if !e.EvidenceType.Valid() {
		return NewError(ErrorCodeEvidence, "evidence_type is invalid")
	}
	if strings.TrimSpace(e.Description) == "" {
		return NewError(ErrorCodeEvidence, "evidence description is required")
	}
	if err := e.SourceRef.Validate(); err != nil {
		return WrapError(ErrorCodeEvidence, "evidence source_ref is invalid", err)
	}
	if err := e.ProducerRef.Validate(); err != nil {
		return WrapError(ErrorCodeEvidence, "evidence producer_ref is invalid", err)
	}
	if e.CapturedAt.IsZero() || e.RegisteredAt.IsZero() || e.UpdatedAt.IsZero() {
		return NewError(ErrorCodeEvidence, "evidence timestamps are required")
	}
	if e.ArtifactID != nil {
		if err := e.ArtifactID.Validate(); err != nil {
			return WrapError(ErrorCodeEvidence, "evidence artifact_id is invalid", err)
		}
	}
	if e.Measurement != nil {
		if err := e.Measurement.Validate(); err != nil {
			return err
		}
	}
	if e.EvidenceType == EvidenceTypeMeasurement && e.Measurement == nil {
		return NewError(ErrorCodeEvidence, "measurement evidence requires measurement")
	}
	if e.EvidenceType != EvidenceTypeMeasurement && e.Measurement != nil {
		return NewError(ErrorCodeEvidence, "only measurement evidence may contain measurement")
	}
	if !e.Lifecycle.Valid() {
		return NewError(ErrorCodeEvidence, "evidence lifecycle is invalid")
	}
	if e.Lifecycle == EvidenceLifecycleRegistered && strings.TrimSpace(e.RetractionReason) != "" {
		return NewError(ErrorCodeEvidence, "registered evidence cannot have retraction_reason")
	}
	if e.Lifecycle == EvidenceLifecycleRetracted && strings.TrimSpace(e.RetractionReason) == "" {
		return NewError(ErrorCodeEvidence, "retracted evidence requires retraction_reason")
	}
	return nil
}

func (e *Evidence) Retract(reason string, now time.Time) error {
	if e.Lifecycle != EvidenceLifecycleRegistered {
		return NewError(ErrorCodeInvalidTransition, "only registered evidence can be retracted")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return NewError(ErrorCodeEvidence, "evidence retraction reason is required")
	}
	next, err := nextVersion(e.Version)
	if err != nil {
		return err
	}
	e.Version = next
	e.Lifecycle = EvidenceLifecycleRetracted
	e.RetractionReason = reason
	e.UpdatedAt = now.UTC()
	return e.Validate()
}

func cloneIDPointerValue(value *ID) *ID {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func cloneMeasurement(value *Measurement) *Measurement {
	if value == nil {
		return nil
	}
	cloned := *value
	cloned.Value = append(json.RawMessage(nil), value.Value...)
	cloned.Conditions = append(json.RawMessage(nil), value.Conditions...)
	return &cloned
}
