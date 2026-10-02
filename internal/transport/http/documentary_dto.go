package httptransport

import (
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

type registerArtifactRequest struct {
	ArtifactType  string          `json:"artifact_type"`
	Name          string          `json:"name"`
	URI           string          `json:"uri"`
	MediaType     string          `json:"media_type,omitempty"`
	Checksum      string          `json:"checksum,omitempty"`
	SourceVersion string          `json:"source_version,omitempty"`
	ProducerRef   domain.ActorRef `json:"producer_ref"`
	ProducedAt    *time.Time      `json:"produced_at,omitempty"`
}

type registerEvidenceRequest struct {
	EvidenceType  domain.EvidenceType      `json:"evidence_type"`
	Description   string                   `json:"description"`
	SourceRef     domain.ExternalReference `json:"source_ref"`
	ProducerRef   domain.ActorRef          `json:"producer_ref"`
	CapturedAt    time.Time                `json:"captured_at"`
	ArtifactID    *string                  `json:"artifact_id,omitempty"`
	Measurement   *domain.Measurement      `json:"measurement,omitempty"`
	SourceVersion string                   `json:"source_version,omitempty"`
	Checksum      string                   `json:"checksum,omitempty"`
}

type createEvidenceLinkRequest struct {
	EvidenceID  string                  `json:"evidence_id"`
	TargetRef   relationEndpointRequest `json:"target_ref"`
	CriterionID *string                 `json:"criterion_id,omitempty"`
	Stance      domain.EvidenceStance   `json:"stance"`
	Rationale   string                  `json:"rationale"`
}

type proposeDecisionRequest struct {
	Title        string   `json:"title"`
	Proposal     string   `json:"proposal"`
	Rationale    string   `json:"rationale,omitempty"`
	Alternatives []string `json:"alternatives"`
}

type reviseDecisionRequest struct {
	ExpectedVersion *uint64  `json:"expected_version,omitempty"`
	Title           string   `json:"title"`
	Proposal        string   `json:"proposal"`
	Rationale       string   `json:"rationale,omitempty"`
	Alternatives    []string `json:"alternatives"`
}

type acceptDecisionRequest struct {
	ExpectedVersion   *uint64 `json:"expected_version,omitempty"`
	ChosenAlternative string  `json:"chosen_alternative"`
	Rationale         string  `json:"rationale"`
}

type supersedeDecisionRequest struct {
	ExpectedVersion   *uint64  `json:"expected_version,omitempty"`
	Title             string   `json:"title"`
	Proposal          string   `json:"proposal"`
	Alternatives      []string `json:"alternatives"`
	ChosenAlternative string   `json:"chosen_alternative"`
	Rationale         string   `json:"rationale"`
}
