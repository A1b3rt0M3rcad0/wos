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
	WithdrawnAt      *time.Time        `json:"withdrawn_at,omitempty"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

func NewArtifact(id ID, scope Scope, artifactType, name, uri, mediaType, checksum, sourceVersion string, producer ActorRef, producedAt *time.Time, now time.Time) (Artifact, error) {
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
		ProducedAt:    cloneTime(producedAt),
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
	if !a.Lifecycle.Valid() {
		return NewError(ErrorCodeArtifact, "artifact lifecycle is invalid")
	}
	if a.RegisteredAt.IsZero() || a.UpdatedAt.IsZero() {
		return NewError(ErrorCodeArtifact, "artifact timestamps are required")
	}
	if a.ProducedAt != nil && a.ProducedAt.IsZero() {
		return NewError(ErrorCodeArtifact, "artifact produced_at is invalid")
	}
	switch a.Lifecycle {
	case ArtifactLifecycleRegistered:
		if strings.TrimSpace(a.WithdrawalReason) != "" || a.WithdrawnAt != nil {
			return NewError(ErrorCodeArtifact, "registered artifact cannot contain withdrawal fields")
		}
	case ArtifactLifecycleWithdrawn:
		if strings.TrimSpace(a.WithdrawalReason) == "" || a.WithdrawnAt == nil || a.WithdrawnAt.IsZero() {
			return NewError(ErrorCodeArtifact, "withdrawn artifact requires reason and withdrawn_at")
		}
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
	at := now.UTC()
	a.Version = next
	a.Lifecycle = ArtifactLifecycleWithdrawn
	a.WithdrawalReason = reason
	a.WithdrawnAt = &at
	a.UpdatedAt = at
	return a.Validate()
}

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
	case EvidenceTypeMeasurement, EvidenceTypeTestResult, EvidenceTypeInspection, EvidenceTypeAttestation, EvidenceTypeSource, EvidenceTypeExternalEvaluation:
		return true
	default:
		return false
	}
}

type SourceReference struct {
	Provider    string `json:"provider"`
	ID          string `json:"id,omitempty"`
	URI         string `json:"uri,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
}

func (r SourceReference) Validate() error {
	if strings.TrimSpace(r.Provider) == "" {
		return NewError(ErrorCodeEvidence, "source_ref provider is required")
	}
	if strings.TrimSpace(r.ID) == "" && strings.TrimSpace(r.URI) == "" {
		return NewError(ErrorCodeEvidence, "source_ref requires id or uri")
	}
	return nil
}

type Measurement struct {
	Value      string `json:"value"`
	Unit       string `json:"unit,omitempty"`
	Method     string `json:"method,omitempty"`
	Conditions string `json:"conditions,omitempty"`
}

func (m Measurement) Validate() error {
	if strings.TrimSpace(m.Value) == "" {
		return NewError(ErrorCodeEvidence, "measurement value is required")
	}
	return nil
}

type EvidenceLifecycle string

const (
	EvidenceLifecycleRegistered EvidenceLifecycle = "registered"
	EvidenceLifecycleRetracted  EvidenceLifecycle = "retracted"
)

func (l EvidenceLifecycle) Valid() bool {
	return l == EvidenceLifecycleRegistered || l == EvidenceLifecycleRetracted
}

type Evidence struct {
	ID               ID                `json:"id"`
	Scope            Scope             `json:"scope"`
	Version          Version           `json:"version"`
	EvidenceType     EvidenceType      `json:"evidence_type"`
	Description      string            `json:"description"`
	SourceRef        SourceReference   `json:"source_ref"`
	ProducerRef      ActorRef          `json:"producer_ref"`
	CapturedAt       time.Time         `json:"captured_at"`
	RegisteredAt     time.Time         `json:"registered_at"`
	ArtifactID       *ID               `json:"artifact_id,omitempty"`
	Measurement      *Measurement      `json:"measurement,omitempty"`
	SourceVersion    string            `json:"source_version,omitempty"`
	Checksum         string            `json:"checksum,omitempty"`
	Lifecycle        EvidenceLifecycle `json:"lifecycle"`
	RetractionReason string            `json:"retraction_reason,omitempty"`
	RetractedAt      *time.Time        `json:"retracted_at,omitempty"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

func NewEvidence(id ID, scope Scope, evidenceType EvidenceType, description string, source SourceReference, producer ActorRef, capturedAt time.Time, artifactID *ID, measurement *Measurement, sourceVersion, checksum string, now time.Time) (Evidence, error) {
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
		ArtifactID:    cloneID(artifactID),
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
		return err
	}
	if err := e.ProducerRef.Validate(); err != nil {
		return WrapError(ErrorCodeEvidence, "evidence producer_ref is invalid", err)
	}
	if e.CapturedAt.IsZero() || e.RegisteredAt.IsZero() || e.UpdatedAt.IsZero() {
		return NewError(ErrorCodeEvidence, "evidence timestamps are required")
	}
	if e.ArtifactID != nil {
		if err := e.ArtifactID.Validate(); err != nil {
			return WrapError(ErrorCodeEvidence, "artifact_id is invalid", err)
		}
	}
	if e.Measurement != nil {
		if err := e.Measurement.Validate(); err != nil {
			return err
		}
	}
	if e.EvidenceType == EvidenceTypeMeasurement && e.Measurement == nil {
		return NewError(ErrorCodeEvidence, "measurement evidence requires measurement details")
	}
	if e.EvidenceType != EvidenceTypeMeasurement && e.Measurement != nil {
		return NewError(ErrorCodeEvidence, "measurement details require evidence_type measurement")
	}
	if !e.Lifecycle.Valid() {
		return NewError(ErrorCodeEvidence, "evidence lifecycle is invalid")
	}
	switch e.Lifecycle {
	case EvidenceLifecycleRegistered:
		if strings.TrimSpace(e.RetractionReason) != "" || e.RetractedAt != nil {
			return NewError(ErrorCodeEvidence, "registered evidence cannot contain retraction fields")
		}
	case EvidenceLifecycleRetracted:
		if strings.TrimSpace(e.RetractionReason) == "" || e.RetractedAt == nil || e.RetractedAt.IsZero() {
			return NewError(ErrorCodeEvidence, "retracted evidence requires reason and retracted_at")
		}
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
	at := now.UTC()
	e.Version = next
	e.Lifecycle = EvidenceLifecycleRetracted
	e.RetractionReason = reason
	e.RetractedAt = &at
	e.UpdatedAt = at
	return e.Validate()
}

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
	ID               ID                    `json:"id"`
	Scope            Scope                 `json:"scope"`
	Version          Version               `json:"version"`
	EvidenceID       ID                    `json:"evidence_id"`
	TargetRef        EntityRef             `json:"target_ref"`
	CriterionID      *ID                   `json:"criterion_id,omitempty"`
	Stance           EvidenceStance        `json:"stance"`
	Rationale        string                `json:"rationale"`
	Lifecycle        EvidenceLinkLifecycle `json:"lifecycle"`
	RetractionReason string                `json:"retraction_reason,omitempty"`
	RetractedAt      *time.Time            `json:"retracted_at,omitempty"`
	CreatedAt        time.Time             `json:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at"`
}

func NewEvidenceLink(id ID, scope Scope, evidenceID ID, target EntityRef, criterionID *ID, stance EvidenceStance, rationale string, now time.Time) (EvidenceLink, error) {
	value := EvidenceLink{
		ID:          id,
		Scope:       scope,
		Version:     InitialVersion,
		EvidenceID:  evidenceID,
		TargetRef:   target,
		CriterionID: cloneID(criterionID),
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
		return NewError(ErrorCodeEvidenceLink, "target_ref must belong to the same Outcome")
	}
	switch l.TargetRef.Kind {
	case EntityKindOutcome, EntityKindObjective, EntityKindWorkItem, EntityKindIssue, EntityKindDecision:
	default:
		return NewError(ErrorCodeEvidenceLink, "unsupported evidence target kind")
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
		return NewError(ErrorCodeEvidenceLink, "evidence stance is invalid")
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
	switch l.Lifecycle {
	case EvidenceLinkLifecycleActive:
		if strings.TrimSpace(l.RetractionReason) != "" || l.RetractedAt != nil {
			return NewError(ErrorCodeEvidenceLink, "active evidence link cannot contain retraction fields")
		}
	case EvidenceLinkLifecycleRetracted:
		if strings.TrimSpace(l.RetractionReason) == "" || l.RetractedAt == nil || l.RetractedAt.IsZero() {
			return NewError(ErrorCodeEvidenceLink, "retracted evidence link requires reason and retracted_at")
		}
	}
	return nil
}

func (l *EvidenceLink) Retract(reason string, now time.Time) error {
	if l.Lifecycle != EvidenceLinkLifecycleActive {
		return NewError(ErrorCodeInvalidTransition, "only active evidence link can be retracted")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return NewError(ErrorCodeEvidenceLink, "evidence link retraction reason is required")
	}
	next, err := nextVersion(l.Version)
	if err != nil {
		return err
	}
	at := now.UTC()
	l.Version = next
	l.Lifecycle = EvidenceLinkLifecycleRetracted
	l.RetractionReason = reason
	l.RetractedAt = &at
	l.UpdatedAt = at
	return l.Validate()
}

type DecisionLifecycle string

const (
	DecisionLifecycleProposed   DecisionLifecycle = "proposed"
	DecisionLifecycleAccepted   DecisionLifecycle = "accepted"
	DecisionLifecycleRejected   DecisionLifecycle = "rejected"
	DecisionLifecycleSuperseded DecisionLifecycle = "superseded"
)

func (l DecisionLifecycle) Valid() bool {
	switch l {
	case DecisionLifecycleProposed, DecisionLifecycleAccepted, DecisionLifecycleRejected, DecisionLifecycleSuperseded:
		return true
	default:
		return false
	}
}

type Decision struct {
	ID                   ID                `json:"id"`
	Scope                Scope             `json:"scope"`
	Version              Version           `json:"version"`
	Title                string            `json:"title"`
	Proposal             string            `json:"proposal"`
	ChosenAlternative    string            `json:"chosen_alternative,omitempty"`
	Rationale            string            `json:"rationale,omitempty"`
	Alternatives         []string          `json:"alternatives,omitempty"`
	Lifecycle            DecisionLifecycle `json:"lifecycle"`
	ProposedBy           ActorRef          `json:"proposed_by"`
	DecidedBy            *ActorRef         `json:"decided_by,omitempty"`
	DecidedAt            *time.Time        `json:"decided_at,omitempty"`
	RejectionReason      string            `json:"rejection_reason,omitempty"`
	SupersedesDecisionID *ID               `json:"supersedes_decision_id,omitempty"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
}

func NewDecision(id ID, scope Scope, title, proposal string, alternatives []string, chosenAlternative, rationale string, proposedBy ActorRef, now time.Time) (Decision, error) {
	value := Decision{
		ID:                id,
		Scope:             scope,
		Version:           InitialVersion,
		Title:             strings.TrimSpace(title),
		Proposal:          strings.TrimSpace(proposal),
		ChosenAlternative: strings.TrimSpace(chosenAlternative),
		Rationale:         strings.TrimSpace(rationale),
		Alternatives:      normalizeAlternatives(alternatives),
		Lifecycle:         DecisionLifecycleProposed,
		ProposedBy:        proposedBy,
		CreatedAt:         now.UTC(),
		UpdatedAt:         now.UTC(),
	}
	if err := value.Validate(); err != nil {
		return Decision{}, err
	}
	return value, nil
}

func NewSupersedingDecision(id ID, scope Scope, predecessor ID, title, proposal string, alternatives []string, chosenAlternative, rationale string, actor ActorRef, now time.Time) (Decision, error) {
	value, err := NewDecision(id, scope, title, proposal, alternatives, chosenAlternative, rationale, actor, now)
	if err != nil {
		return Decision{}, err
	}
	if err := predecessor.Validate(); err != nil {
		return Decision{}, WrapError(ErrorCodeDecision, "superseded decision id is invalid", err)
	}
	value.SupersedesDecisionID = &predecessor
	value.Lifecycle = DecisionLifecycleAccepted
	decidedAt := now.UTC()
	decidedBy := actor
	value.DecidedBy = &decidedBy
	value.DecidedAt = &decidedAt
	if strings.TrimSpace(value.Rationale) == "" {
		return Decision{}, NewError(ErrorCodeDecision, "accepted decision requires rationale")
	}
	if err := value.Validate(); err != nil {
		return Decision{}, err
	}
	return value, nil
}

func (d Decision) Ref() EntityRef {
	return EntityRef{Scope: d.Scope, Kind: EntityKindDecision, ID: d.ID}
}

func (d Decision) Validate() error {
	if err := d.ID.Validate(); err != nil {
		return err
	}
	if err := d.Scope.Validate(); err != nil {
		return err
	}
	if err := d.Version.Validate(); err != nil {
		return err
	}
	if strings.TrimSpace(d.Title) == "" {
		return NewError(ErrorCodeDecision, "decision title is required")
	}
	if strings.TrimSpace(d.Proposal) == "" {
		return NewError(ErrorCodeDecision, "decision proposal is required")
	}
	if err := d.ProposedBy.Validate(); err != nil {
		return WrapError(ErrorCodeDecision, "decision proposed_by is invalid", err)
	}
	if !d.Lifecycle.Valid() {
		return NewError(ErrorCodeDecision, "decision lifecycle is invalid")
	}
	if d.CreatedAt.IsZero() || d.UpdatedAt.IsZero() {
		return NewError(ErrorCodeDecision, "decision timestamps are required")
	}
	seen := make(map[string]struct{}, len(d.Alternatives))
	for _, item := range d.Alternatives {
		item = strings.TrimSpace(item)
		if item == "" {
			return NewError(ErrorCodeDecision, "decision alternatives cannot contain empty values")
		}
		if _, ok := seen[item]; ok {
			return NewError(ErrorCodeDecision, "decision alternatives cannot contain duplicates")
		}
		seen[item] = struct{}{}
	}
	if d.ChosenAlternative != "" {
		if _, ok := seen[d.ChosenAlternative]; !ok && len(d.Alternatives) > 0 {
			return NewError(ErrorCodeDecision, "chosen_alternative must exist in alternatives")
		}
	}
	if d.SupersedesDecisionID != nil {
		if err := d.SupersedesDecisionID.Validate(); err != nil {
			return WrapError(ErrorCodeDecision, "supersedes_decision_id is invalid", err)
		}
		if *d.SupersedesDecisionID == d.ID {
			return NewError(ErrorCodeDecision, "decision cannot supersede itself")
		}
	}
	switch d.Lifecycle {
	case DecisionLifecycleProposed:
		if d.DecidedBy != nil || d.DecidedAt != nil || strings.TrimSpace(d.RejectionReason) != "" || d.SupersedesDecisionID != nil {
			return NewError(ErrorCodeDecision, "proposed decision cannot contain terminal decision fields")
		}
	case DecisionLifecycleAccepted:
		if d.DecidedBy == nil || d.DecidedAt == nil || d.DecidedAt.IsZero() {
			return NewError(ErrorCodeDecision, "accepted decision requires decided_by and decided_at")
		}
		if err := d.DecidedBy.Validate(); err != nil {
			return WrapError(ErrorCodeDecision, "decision decided_by is invalid", err)
		}
		if strings.TrimSpace(d.Rationale) == "" {
			return NewError(ErrorCodeDecision, "accepted decision requires rationale")
		}
		if strings.TrimSpace(d.RejectionReason) != "" {
			return NewError(ErrorCodeDecision, "accepted decision cannot contain rejection_reason")
		}
	case DecisionLifecycleRejected:
		if d.DecidedBy == nil || d.DecidedAt == nil || d.DecidedAt.IsZero() {
			return NewError(ErrorCodeDecision, "rejected decision requires decided_by and decided_at")
		}
		if err := d.DecidedBy.Validate(); err != nil {
			return WrapError(ErrorCodeDecision, "decision decided_by is invalid", err)
		}
		if strings.TrimSpace(d.RejectionReason) == "" {
			return NewError(ErrorCodeDecision, "rejected decision requires rejection_reason")
		}
		if d.SupersedesDecisionID != nil {
			return NewError(ErrorCodeDecision, "rejected decision cannot supersede another decision")
		}
	case DecisionLifecycleSuperseded:
		if d.DecidedBy == nil || d.DecidedAt == nil || d.DecidedAt.IsZero() {
			return NewError(ErrorCodeDecision, "superseded decision must preserve original acceptance")
		}
		if strings.TrimSpace(d.Rationale) == "" {
			return NewError(ErrorCodeDecision, "superseded decision must preserve acceptance rationale")
		}
	}
	return nil
}

func (d *Decision) Update(title, proposal, chosenAlternative, rationale *string, alternatives *[]string, now time.Time) (bool, error) {
	if d.Lifecycle != DecisionLifecycleProposed {
		return false, NewError(ErrorCodeInvalidTransition, "only proposed decision can be edited")
	}
	nextTitle, nextProposal := d.Title, d.Proposal
	nextChosen, nextRationale := d.ChosenAlternative, d.Rationale
	nextAlternatives := append([]string(nil), d.Alternatives...)
	if title != nil {
		nextTitle = strings.TrimSpace(*title)
	}
	if proposal != nil {
		nextProposal = strings.TrimSpace(*proposal)
	}
	if chosenAlternative != nil {
		nextChosen = strings.TrimSpace(*chosenAlternative)
	}
	if rationale != nil {
		nextRationale = strings.TrimSpace(*rationale)
	}
	if alternatives != nil {
		nextAlternatives = normalizeAlternatives(*alternatives)
	}
	if nextTitle == d.Title && nextProposal == d.Proposal && nextChosen == d.ChosenAlternative && nextRationale == d.Rationale && stringSlicesEqual(nextAlternatives, d.Alternatives) {
		return false, nil
	}
	original := *d
	d.Title, d.Proposal = nextTitle, nextProposal
	d.ChosenAlternative, d.Rationale = nextChosen, nextRationale
	d.Alternatives = nextAlternatives
	next, err := nextVersion(d.Version)
	if err != nil {
		*d = original
		return false, err
	}
	d.Version = next
	d.UpdatedAt = now.UTC()
	if err := d.Validate(); err != nil {
		*d = original
		return false, err
	}
	return true, nil
}

func (d *Decision) Accept(actor ActorRef, now time.Time) error {
	if d.Lifecycle != DecisionLifecycleProposed {
		return NewError(ErrorCodeInvalidTransition, "only proposed decision can be accepted")
	}
	if strings.TrimSpace(d.Rationale) == "" {
		return NewError(ErrorCodeDecision, "accepted decision requires rationale")
	}
	if err := actor.Validate(); err != nil {
		return err
	}
	next, err := nextVersion(d.Version)
	if err != nil {
		return err
	}
	at := now.UTC()
	by := actor
	d.Version = next
	d.Lifecycle = DecisionLifecycleAccepted
	d.DecidedBy = &by
	d.DecidedAt = &at
	d.UpdatedAt = at
	return d.Validate()
}

func (d *Decision) Reject(reason string, actor ActorRef, now time.Time) error {
	if d.Lifecycle != DecisionLifecycleProposed {
		return NewError(ErrorCodeInvalidTransition, "only proposed decision can be rejected")
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return NewError(ErrorCodeDecision, "decision rejection reason is required")
	}
	if err := actor.Validate(); err != nil {
		return err
	}
	next, err := nextVersion(d.Version)
	if err != nil {
		return err
	}
	at := now.UTC()
	by := actor
	d.Version = next
	d.Lifecycle = DecisionLifecycleRejected
	d.DecidedBy = &by
	d.DecidedAt = &at
	d.RejectionReason = reason
	d.UpdatedAt = at
	return d.Validate()
}

func (d *Decision) MarkSuperseded(now time.Time) error {
	if d.Lifecycle != DecisionLifecycleAccepted {
		return NewError(ErrorCodeInvalidTransition, "only accepted decision can be superseded")
	}
	next, err := nextVersion(d.Version)
	if err != nil {
		return err
	}
	d.Version = next
	d.Lifecycle = DecisionLifecycleSuperseded
	d.UpdatedAt = now.UTC()
	return d.Validate()
}

func normalizeAlternatives(values []string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, strings.TrimSpace(value))
	}
	return result
}

func stringSlicesEqual(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func cloneID(value *ID) *ID {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func cloneTime(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	copy := value.UTC()
	return &copy
}

func cloneMeasurement(value *Measurement) *Measurement {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
