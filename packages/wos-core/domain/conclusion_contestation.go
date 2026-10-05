package domain

type ConclusionContestationKind string

const (
	ConclusionContestationObligationsChanged      ConclusionContestationKind = "obligations_changed"
	ConclusionContestationRequiredObjective       ConclusionContestationKind = "required_objective_not_achieved"
	ConclusionContestationAssessmentContradiction ConclusionContestationKind = "assessment_contradiction"
	ConclusionContestationEvidenceRetracted       ConclusionContestationKind = "evidence_retracted"
)

func (k ConclusionContestationKind) Valid() bool {
	switch k {
	case ConclusionContestationAssessmentContradiction, ConclusionContestationEvidenceRetracted, ConclusionContestationObligationsChanged, ConclusionContestationRequiredObjective:
		return true
	default:
		return false
	}
}

type ConclusionContestation struct {
	ObjectiveID  *ID                        `json:"objective_id,omitempty"`
	OwnerRef     EntityRef                  `json:"owner_ref"`
	Kind         ConclusionContestationKind `json:"kind"`
	CriterionID  ID                         `json:"criterion_id,omitempty"`
	AssessmentID ID                         `json:"assessment_id,omitempty"`
	EvidenceID   *ID                        `json:"evidence_id,omitempty"`
}
