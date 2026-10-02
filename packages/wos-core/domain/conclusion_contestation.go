package domain

type ConclusionContestationKind string

const (
	ConclusionContestationAssessmentContradiction ConclusionContestationKind = "assessment_contradiction"
	ConclusionContestationEvidenceRetracted       ConclusionContestationKind = "evidence_retracted"
)

func (k ConclusionContestationKind) Valid() bool {
	switch k {
	case ConclusionContestationAssessmentContradiction, ConclusionContestationEvidenceRetracted:
		return true
	default:
		return false
	}
}

type ConclusionContestation struct {
	OwnerRef     EntityRef                   `json:"owner_ref"`
	Kind         ConclusionContestationKind  `json:"kind"`
	CriterionID  ID                          `json:"criterion_id"`
	AssessmentID ID                          `json:"assessment_id"`
	EvidenceID   *ID                         `json:"evidence_id,omitempty"`
}
