package application

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// validateRequiredAssessmentProof checks the documentary references used by a
// new conclusion. Historical assessments and conclusions remain immutable when
// an observation is subsequently retracted.
func validateRequiredAssessmentProof(ctx context.Context, u ports.UnitOfWork, owner domain.EntityRef, criteria domain.CriterionSet) error {
	if !criteria.HasRequiredActive() {
		return nil
	}
	if _, err := criteria.RequiredSatisfied(); err != nil {
		return err
	}
	validated := map[domain.ID]bool{}
	for _, criterion := range criteria.Items {
		if !criterion.Required || criterion.Status != domain.CriterionStatusActive {
			continue
		}
		assessment := criteria.CurrentAssessments[criterion.ID]
		for _, id := range assessment.EvidenceIDs {
			if validated[id] {
				continue
			}
			if err := validateAssessmentEvidence(ctx, u, owner.Scope, []domain.ID{id}); err != nil {
				code, ok := domain.ErrorCodeOf(err)
				if !ok || code != domain.ErrorCodeNotFound && code != domain.ErrorCodePreconditionFailed {
					return err
				}
				reason := "current_proof_inapplicable"
				if code == domain.ErrorCodePreconditionFailed {
					reason = "evidence_retracted"
				}
				subject := owner
				criterionID := criterion.ID
				evidence := domain.EntityRef{Scope: owner.Scope, Kind: domain.EntityKindEvidence, ID: id}
				return domain.NewDetailedError(domain.ErrorCodePreconditionFailed, "current criterion proof is unavailable or retracted; record a current assessment before concluding", domain.ErrorDetails{ReasonCode: reason, SubjectRef: &subject, FieldPaths: []string{"criteria." + criterion.ID.String()}, RelatedRefs: []domain.EntityRef{evidence}, Remedies: []domain.ErrorRemedy{{ActionID: "record_criterion_assessment", SubjectRef: &subject, CriterionID: &criterionID}}})
			}
			validated[id] = true
		}
	}
	return nil
}
