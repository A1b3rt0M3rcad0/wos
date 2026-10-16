package woscli

import (
	"fmt"
	"strings"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

// Stable shape checks run before network/signing. They never predict current
// policy, authorization, CAS, lease validity or server acceptance.
func validateReturnDraftV2(profile ProfileV2, file ContractFileV2, view ContractViewV2) error {
	var assessments []a.SignedAssessmentInput
	if file.Execution != nil {
		assessments = file.Execution.Material.Assessments
		if strings.TrimSpace(file.Execution.Material.Result.Summary) == "" {
			return fmt.Errorf("result summary is required before signing")
		}
	} else {
		assessments = file.Review.Material.Assessments
		if strings.TrimSpace(file.Review.Material.Reason) == "" {
			return fmt.Errorf("review reason is required before signing")
		}
		if file.Review.Decision == "changes_requested" && len(file.Review.Material.Findings) == 0 {
			return fmt.Errorf("changes_requested needs a finding for an existing obligation")
		}
		if view.ReviewSpecification != nil && view.ReviewSpecification.Spec.IssuedWorkSpecification != nil {
			doc := *view.ReviewSpecification.Spec.IssuedWorkSpecification
			raw, e := verifyProfileDocumentV2(profile, doc, signing.ContractSpec)
			if e != nil {
				return e
			}
			var work signing.SpecPayload[d.SignedWorkSpec]
			if e = signing.DecodeStrict(raw, &work, 128<<10); e != nil {
				return e
			}
			if work.ProtocolVersion != 2 || work.ServerID != view.Authority.ServerID || work.NamespaceID != view.Authority.NamespaceID || work.OutcomeID != view.Authority.OutcomeID || work.WorkItemID != view.Authority.WorkItemID || signing.Digest(raw) != view.ReviewSpecification.Spec.IssuedWorkSpecDigest {
				return fmt.Errorf("reviewed specification binding differs")
			}
			for _, finding := range file.Review.Material.Findings {
				if e = finding.ValidateAgainst(work.Spec.WorkSpec()); e != nil {
					return e
				}
			}
		}
	}
	if len(assessments) > 100 {
		return fmt.Errorf("assessment limit is 100")
	}
	seen := map[d.ID]bool{}
	for _, assessment := range assessments {
		if assessment.CriterionID.Validate() != nil || assessment.CriterionRevision == 0 || !assessment.Result.Valid() || strings.TrimSpace(assessment.Rationale) == "" || len(assessment.Rationale) > 16384 || seen[assessment.CriterionID] {
			return fmt.Errorf("assessment needs a unique criterion, revision, valid result and rationale")
		}
		seen[assessment.CriterionID] = true
		for _, id := range assessment.EvidenceIDs {
			if e := id.Validate(); e != nil {
				return e
			}
		}
	}
	return nil
}
