package domain

import (
	"fmt"
	"strings"
	"time"
)

type CriterionRevision uint64

const InitialCriterionRevision CriterionRevision = 1

type VerificationMode string

const (
	VerificationModeAttestation        VerificationMode = "attestation"
	VerificationModeEvidenceReview     VerificationMode = "evidence_review"
	VerificationModeExternalEvaluation VerificationMode = "external_evaluation"
)

func (m VerificationMode) Valid() bool {
	switch m {
	case VerificationModeAttestation, VerificationModeEvidenceReview, VerificationModeExternalEvaluation:
		return true
	default:
		return false
	}
}

type CriterionDefinitionRevision struct {
	CriterionID      ID                `json:"criterion_id"`
	OwnerRef         EntityRef         `json:"owner_ref"`
	Revision         CriterionRevision `json:"criterion_revision"`
	Title            string            `json:"title"`
	Description      string            `json:"description,omitempty"`
	Required         bool              `json:"required"`
	VerificationMode VerificationMode  `json:"verification_mode"`
}

func (r CriterionDefinitionRevision) Validate() error {
	if err := r.CriterionID.Validate(); err != nil {
		return WrapError(ErrorCodeCriterion, "criterion revision id is invalid", err)
	}
	if err := r.OwnerRef.Validate(); err != nil {
		return WrapError(ErrorCodeCriterion, "criterion revision owner is invalid", err)
	}
	if r.Revision < InitialCriterionRevision {
		return NewError(ErrorCodeCriterion, "criterion revision must be at least 1")
	}
	if strings.TrimSpace(r.Title) == "" {
		return NewError(ErrorCodeCriterion, "criterion revision title is required")
	}
	if !r.VerificationMode.Valid() {
		return NewError(ErrorCodeCriterion, "criterion revision verification mode is invalid")
	}
	return nil
}

type CriterionStatus string

const (
	CriterionStatusActive  CriterionStatus = "active"
	CriterionStatusRetired CriterionStatus = "retired"
)

type AssessmentResult string

const (
	AssessmentResultMet          AssessmentResult = "met"
	AssessmentResultNotMet       AssessmentResult = "not_met"
	AssessmentResultInconclusive AssessmentResult = "inconclusive"
	AssessmentResultWaived       AssessmentResult = "waived"
)

func (r AssessmentResult) Valid() bool {
	switch r {
	case AssessmentResultMet, AssessmentResultNotMet, AssessmentResultInconclusive, AssessmentResultWaived:
		return true
	default:
		return false
	}
}

type SuccessCriterion struct {
	ID               ID                `json:"id"`
	OwnerRef         EntityRef         `json:"owner_ref"`
	Title            string            `json:"title"`
	Description      string            `json:"description,omitempty"`
	Required         bool              `json:"required"`
	Revision         CriterionRevision `json:"criterion_revision"`
	VerificationMode VerificationMode  `json:"verification_mode"`
	Status           CriterionStatus   `json:"status"`
}

func NewSuccessCriterion(id ID, owner EntityRef, title, description string, required bool, mode VerificationMode) (SuccessCriterion, error) {
	c := SuccessCriterion{
		ID:               id,
		OwnerRef:         owner,
		Title:            strings.TrimSpace(title),
		Description:      strings.TrimSpace(description),
		Required:         required,
		Revision:         InitialCriterionRevision,
		VerificationMode: mode,
		Status:           CriterionStatusActive,
	}
	if err := c.Validate(); err != nil {
		return SuccessCriterion{}, err
	}
	return c, nil
}

func (c SuccessCriterion) DefinitionRevision() CriterionDefinitionRevision {
	return CriterionDefinitionRevision{
		CriterionID:      c.ID,
		OwnerRef:         c.OwnerRef,
		Revision:         c.Revision,
		Title:            c.Title,
		Description:      c.Description,
		Required:         c.Required,
		VerificationMode: c.VerificationMode,
	}
}

func (c SuccessCriterion) Validate() error {
	if err := c.ID.Validate(); err != nil {
		return WrapError(ErrorCodeCriterion, "criterion id is invalid", err)
	}
	if err := c.OwnerRef.Validate(); err != nil {
		return WrapError(ErrorCodeCriterion, "criterion owner is invalid", err)
	}
	if strings.TrimSpace(c.Title) == "" {
		return NewError(ErrorCodeCriterion, "criterion title is required")
	}
	if c.Revision < InitialCriterionRevision {
		return NewError(ErrorCodeCriterion, "criterion revision must be at least 1")
	}
	if !c.VerificationMode.Valid() {
		return NewError(ErrorCodeCriterion, "criterion verification mode is invalid")
	}
	if c.Status != CriterionStatusActive && c.Status != CriterionStatusRetired {
		return NewError(ErrorCodeCriterion, "criterion status is invalid")
	}
	return nil
}

type EvaluatorRef struct {
	Provider string `json:"provider"`
	ID       string `json:"id"`
	Version  string `json:"version"`
}

func (r EvaluatorRef) Validate() error {
	if strings.TrimSpace(r.Provider) == "" {
		return NewError(ErrorCodeAssessment, "evaluator_ref provider is required")
	}
	if strings.TrimSpace(r.ID) == "" {
		return NewError(ErrorCodeAssessment, "evaluator_ref id is required")
	}
	if strings.TrimSpace(r.Version) == "" {
		return NewError(ErrorCodeAssessment, "evaluator_ref version is required")
	}
	return nil
}

type CriterionAssessment struct {
	ID                     ID                `json:"id"`
	CriterionID            ID                `json:"criterion_id"`
	CriterionRevision      CriterionRevision `json:"criterion_revision"`
	Result                 AssessmentResult  `json:"result"`
	Rationale              string            `json:"rationale"`
	EvidenceIDs            []ID              `json:"evidence_ids,omitempty"`
	EvaluatorRef           *EvaluatorRef     `json:"evaluator_ref,omitempty"`
	PrincipalID            string            `json:"principal_id"`
	Actor                  ActorRef          `json:"actor_ref"`
	AssessedAt             time.Time         `json:"assessed_at"`
	SupersedesAssessmentID *ID               `json:"supersedes_assessment_id,omitempty"`
}

func (a CriterionAssessment) Validate() error {
	if err := a.ID.Validate(); err != nil {
		return WrapError(ErrorCodeAssessment, "assessment id is invalid", err)
	}
	if err := a.CriterionID.Validate(); err != nil {
		return WrapError(ErrorCodeAssessment, "criterion id is invalid", err)
	}
	if a.CriterionRevision < InitialCriterionRevision {
		return NewError(ErrorCodeAssessment, "criterion revision is invalid")
	}
	if !a.Result.Valid() {
		return NewError(ErrorCodeAssessment, "assessment result is invalid")
	}
	if strings.TrimSpace(a.Rationale) == "" {
		return NewError(ErrorCodeAssessment, "assessment rationale is required")
	}
	seenEvidence := make(map[ID]struct{}, len(a.EvidenceIDs))
	for _, evidenceID := range a.EvidenceIDs {
		if err := evidenceID.Validate(); err != nil {
			return WrapError(ErrorCodeAssessment, "assessment evidence id is invalid", err)
		}
		if _, exists := seenEvidence[evidenceID]; exists {
			return NewError(ErrorCodeAssessment, "assessment evidence_ids cannot contain duplicates")
		}
		seenEvidence[evidenceID] = struct{}{}
	}
	if a.EvaluatorRef != nil {
		if err := a.EvaluatorRef.Validate(); err != nil {
			return err
		}
	}
	if strings.TrimSpace(a.PrincipalID) == "" {
		return NewError(ErrorCodeAssessment, "assessment principal_id is required")
	}
	if err := a.Actor.Validate(); err != nil {
		return WrapError(ErrorCodeAssessment, "assessment actor is invalid", err)
	}
	if a.AssessedAt.IsZero() {
		return NewError(ErrorCodeAssessment, "assessment time is required")
	}
	return nil
}

type CriterionAssessmentRef struct {
	AssessmentID      ID                `json:"assessment_id"`
	CriterionID       ID                `json:"criterion_id"`
	CriterionRevision CriterionRevision `json:"criterion_revision"`
	Result            AssessmentResult  `json:"result"`
}

type CriterionSet struct {
	Items              []SuccessCriterion         `json:"items,omitempty"`
	Assessments        []CriterionAssessment      `json:"assessments,omitempty"`
	CurrentAssessments map[ID]CriterionAssessment `json:"current_assessments,omitempty"`
}

func NewCriterionSet() CriterionSet {
	return CriterionSet{CurrentAssessments: make(map[ID]CriterionAssessment)}
}

func (s *CriterionSet) ensureMap() {
	if s.CurrentAssessments == nil {
		s.CurrentAssessments = make(map[ID]CriterionAssessment)
	}
}

func (s *CriterionSet) Add(c SuccessCriterion) error {
	if err := c.Validate(); err != nil {
		return err
	}
	for _, existing := range s.Items {
		if existing.ID == c.ID {
			return NewError(ErrorCodeAlreadyExists, "criterion already exists")
		}
	}
	s.Items = append(s.Items, c)
	s.ensureMap()
	return nil
}

func (s *CriterionSet) Find(id ID) (*SuccessCriterion, error) {
	for i := range s.Items {
		if s.Items[i].ID == id {
			return &s.Items[i], nil
		}
	}
	return nil, NewError(ErrorCodeNotFound, "criterion not found")
}

func (s *CriterionSet) Revise(id ID, title, description string, required bool, mode VerificationMode) error {
	c, err := s.Find(id)
	if err != nil {
		return err
	}
	if c.Status != CriterionStatusActive {
		return NewError(ErrorCodeCriterion, "retired criterion cannot be revised")
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return NewError(ErrorCodeCriterion, "criterion title is required")
	}
	if !mode.Valid() {
		return NewError(ErrorCodeCriterion, "criterion verification mode is invalid")
	}
	if c.Revision == ^CriterionRevision(0) {
		return NewError(ErrorCodeCriterion, "criterion revision overflow")
	}
	c.Title = title
	c.Description = strings.TrimSpace(description)
	c.Required = required
	c.VerificationMode = mode
	c.Revision++
	s.ensureMap()
	delete(s.CurrentAssessments, id)
	return nil
}

func (s *CriterionSet) Retire(id ID) error {
	c, err := s.Find(id)
	if err != nil {
		return err
	}
	c.Status = CriterionStatusRetired
	s.ensureMap()
	delete(s.CurrentAssessments, id)
	return nil
}

func (s *CriterionSet) RecordAssessment(a CriterionAssessment, waiverAuthorized bool) error {
	if err := a.Validate(); err != nil {
		return err
	}
	c, err := s.Find(a.CriterionID)
	if err != nil {
		return err
	}
	if c.Status != CriterionStatusActive {
		return NewError(ErrorCodeAssessment, "criterion is not active")
	}
	if a.CriterionRevision != c.Revision {
		return NewError(ErrorCodeAssessment, "assessment targets a stale criterion revision")
	}
	if a.Result == AssessmentResultWaived && !waiverAuthorized {
		return NewError(ErrorCodeForbidden, "waived assessment requires assessment:waive authorization")
	}
	switch c.VerificationMode {
	case VerificationModeAttestation:
		if a.EvaluatorRef != nil {
			return NewError(ErrorCodeAssessment, "attestation assessment must not include evaluator_ref")
		}
	case VerificationModeEvidenceReview:
		if len(a.EvidenceIDs) == 0 {
			return NewError(ErrorCodeAssessment, "evidence_review assessment requires evidence_ids")
		}
		if a.EvaluatorRef != nil {
			return NewError(ErrorCodeAssessment, "evidence_review assessment must not include evaluator_ref")
		}
	case VerificationModeExternalEvaluation:
		if a.EvaluatorRef == nil {
			return NewError(ErrorCodeAssessment, "external_evaluation assessment requires evaluator_ref")
		}
	default:
		return NewError(ErrorCodeAssessment, "criterion verification mode is invalid")
	}
	s.ensureMap()
	if current, ok := s.CurrentAssessments[a.CriterionID]; ok {
		previousID := current.ID
		a.SupersedesAssessmentID = &previousID
	}
	s.Assessments = append(s.Assessments, a)
	s.CurrentAssessments[a.CriterionID] = a
	return nil
}

func (s *CriterionSet) AssessAttestation(a CriterionAssessment) error {
	c, err := s.Find(a.CriterionID)
	if err != nil {
		return err
	}
	if c.VerificationMode != VerificationModeAttestation {
		return NewError(ErrorCodeAssessment, "criterion does not accept attestation")
	}
	if a.Result == AssessmentResultWaived {
		return NewError(ErrorCodeAssessment, "waiver authorization is not part of the attestation compatibility path")
	}
	return s.RecordAssessment(a, false)
}

func (s CriterionSet) HasRequiredActive() bool {
	for _, c := range s.Items {
		if c.Status == CriterionStatusActive && c.Required {
			return true
		}
	}
	return false
}

func (s CriterionSet) RequiredSatisfied() ([]CriterionAssessmentRef, error) {
	refs := make([]CriterionAssessmentRef, 0)
	for _, c := range s.Items {
		if c.Status != CriterionStatusActive || !c.Required {
			continue
		}
		a, ok := s.CurrentAssessments[c.ID]
		if !ok {
			return nil, NewError(ErrorCodePreconditionFailed, fmt.Sprintf("required criterion %s has no current assessment", c.ID))
		}
		if a.CriterionRevision != c.Revision {
			return nil, NewError(ErrorCodePreconditionFailed, fmt.Sprintf("required criterion %s has stale assessment", c.ID))
		}
		if a.Result != AssessmentResultMet {
			return nil, NewError(ErrorCodePreconditionFailed, fmt.Sprintf("required criterion %s is not met", c.ID))
		}
		refs = append(refs, CriterionAssessmentRef{
			AssessmentID:      a.ID,
			CriterionID:       c.ID,
			CriterionRevision: c.Revision,
			Result:            a.Result,
		})
	}
	if len(refs) == 0 {
		return nil, NewError(ErrorCodePreconditionFailed, "at least one required active criterion is required")
	}
	return refs, nil
}

func (s CriterionSet) ValidateForOwner(owner EntityRef) error {
	criteria := make(map[ID]SuccessCriterion, len(s.Items))
	for _, criterion := range s.Items {
		if err := criterion.Validate(); err != nil {
			return err
		}
		if criterion.OwnerRef != owner {
			return NewError(ErrorCodeCriterion, "criterion owner does not match aggregate")
		}
		if _, exists := criteria[criterion.ID]; exists {
			return NewError(ErrorCodeCriterion, "duplicate criterion id")
		}
		criteria[criterion.ID] = criterion
	}

	assessmentIDs := make(map[ID]struct{}, len(s.Assessments))
	for _, assessment := range s.Assessments {
		if err := assessment.Validate(); err != nil {
			return err
		}
		criterion, exists := criteria[assessment.CriterionID]
		if !exists {
			return NewError(ErrorCodeAssessment, "assessment references unknown criterion")
		}
		if assessment.CriterionRevision > criterion.Revision {
			return NewError(ErrorCodeAssessment, "assessment references future criterion revision")
		}
		if _, exists := assessmentIDs[assessment.ID]; exists {
			return NewError(ErrorCodeAssessment, "duplicate assessment id")
		}
		assessmentIDs[assessment.ID] = struct{}{}
	}

	for criterionID, current := range s.CurrentAssessments {
		criterion, exists := criteria[criterionID]
		if !exists {
			return NewError(ErrorCodeAssessment, "current assessment references unknown criterion")
		}
		if criterion.Status != CriterionStatusActive {
			return NewError(ErrorCodeAssessment, "retired criterion cannot have current assessment")
		}
		if current.CriterionID != criterionID || current.CriterionRevision != criterion.Revision {
			return NewError(ErrorCodeAssessment, "current assessment does not match current criterion revision")
		}
		if _, exists := assessmentIDs[current.ID]; !exists {
			return NewError(ErrorCodeAssessment, "current assessment is missing from immutable assessment history")
		}
	}

	return nil
}
