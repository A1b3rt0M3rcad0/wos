package domain

// New signed wire DTOs quote exact counters without changing v1 JSON or
// fingerprints. Domain aggregates remain the shared authoritative state.
type SignedCriterion struct {
	ID               ID                `json:"id"`
	OwnerRef         EntityRef         `json:"owner_ref"`
	Title            string            `json:"title"`
	Description      string            `json:"description,omitempty"`
	Required         bool              `json:"required"`
	Revision         CriterionRevision `json:"criterion_revision,string"`
	VerificationMode VerificationMode  `json:"verification_mode"`
	Status           CriterionStatus   `json:"status"`
}

func (c SignedCriterion) Criterion() SuccessCriterion {
	return SuccessCriterion{ID: c.ID, OwnerRef: c.OwnerRef, Title: c.Title, Description: c.Description, Required: c.Required, Revision: c.Revision, VerificationMode: c.VerificationMode, Status: c.Status}
}

type SignedWorkSpec struct {
	OutcomeIntent        string            `json:"outcome_intent,omitempty"`
	ObjectiveIntent      string            `json:"objective_intent,omitempty"`
	Title                string            `json:"title"`
	Description          string            `json:"description"`
	ObjectiveID          *ID               `json:"objective_id"`
	ExecutionSpec        ExecutionSpec     `json:"execution_spec"`
	Criteria             []SignedCriterion `json:"criteria"`
	Dependencies         []EntityRef       `json:"dependencies"`
	AcceptanceFloor      AcceptanceMode    `json:"acceptance_floor"`
	PolicyRevision       Version           `json:"policy_revision,string"`
	PreviousSubmissionID *ID               `json:"previous_submission_id,omitempty"`
	PreviousReviewCaseID *ID               `json:"previous_review_case_id,omitempty"`
}

func NewSignedWorkSpec(spec WorkContractSpec, binding SignedContractBinding) SignedWorkSpec {
	spec = NormalizeContractSpec(spec)
	out := SignedWorkSpec{OutcomeIntent: spec.OutcomeIntent, ObjectiveIntent: spec.ObjectiveIntent, Title: spec.Title, Description: spec.Description, ObjectiveID: spec.ObjectiveID, ExecutionSpec: spec.ExecutionSpec, Criteria: []SignedCriterion{}, Dependencies: spec.Dependencies, AcceptanceFloor: binding.AcceptanceFloor, PolicyRevision: binding.PolicyRevision, PreviousSubmissionID: binding.PreviousSubmissionID, PreviousReviewCaseID: binding.PreviousReviewCaseID}
	for _, c := range spec.Criteria {
		out.Criteria = append(out.Criteria, SignedCriterion{ID: c.ID, OwnerRef: c.OwnerRef, Title: c.Title, Description: c.Description, Required: c.Required, Revision: c.Revision, VerificationMode: c.VerificationMode, Status: c.Status})
	}
	return out
}
func (s SignedWorkSpec) WorkSpec() WorkContractSpec {
	out := WorkContractSpec{OutcomeIntent: s.OutcomeIntent, ObjectiveIntent: s.ObjectiveIntent, Title: s.Title, Description: s.Description, ObjectiveID: s.ObjectiveID, ExecutionSpec: s.ExecutionSpec, Criteria: []SuccessCriterion{}, Dependencies: s.Dependencies}
	for _, c := range s.Criteria {
		out.Criteria = append(out.Criteria, c.Criterion())
	}
	return NormalizeContractSpec(out)
}

type SignedCriterionEvidence struct {
	CriterionID       ID                `json:"criterion_id"`
	CriterionRevision CriterionRevision `json:"criterion_revision,string"`
	EvidenceIDs       []ID              `json:"evidence_ids"`
}
type SignedResultMaterial struct {
	ContractID        ID                        `json:"contract_id"`
	WorkItemID        ID                        `json:"work_item_id"`
	SpecDigest        string                    `json:"spec_digest"`
	Summary           string                    `json:"summary"`
	Artifacts         []SubmissionArtifact      `json:"artifacts"`
	EvidenceIDs       []ID                      `json:"evidence_ids"`
	CriterionEvidence []SignedCriterionEvidence `json:"criterion_evidence"`
}

func NewSignedResultMaterial(m WorkResultMaterial) SignedResultMaterial {
	m = NormalizeResultMaterial(m)
	out := SignedResultMaterial{ContractID: m.ContractID, WorkItemID: m.WorkItemID, SpecDigest: m.SpecDigest, Summary: m.Summary, Artifacts: m.Artifacts, EvidenceIDs: m.EvidenceIDs, CriterionEvidence: []SignedCriterionEvidence{}}
	for _, c := range m.CriterionEvidence {
		out.CriterionEvidence = append(out.CriterionEvidence, SignedCriterionEvidence{CriterionID: c.CriterionID, CriterionRevision: c.CriterionRevision, EvidenceIDs: c.EvidenceIDs})
	}
	return out
}
func (s SignedResultMaterial) Material() WorkResultMaterial {
	out := WorkResultMaterial{ContractID: s.ContractID, WorkItemID: s.WorkItemID, SpecDigest: s.SpecDigest, Summary: s.Summary, Artifacts: s.Artifacts, EvidenceIDs: s.EvidenceIDs, CriterionEvidence: []SubmissionCriterionEvidence{}}
	for _, c := range s.CriterionEvidence {
		out.CriterionEvidence = append(out.CriterionEvidence, SubmissionCriterionEvidence{CriterionID: c.CriterionID, CriterionRevision: c.CriterionRevision, EvidenceIDs: c.EvidenceIDs})
	}
	return NormalizeResultMaterial(out)
}
func SignedResultDigest(m WorkResultMaterial) (string, error) {
	return SemanticDigest(NewSignedResultMaterial(m))
}
