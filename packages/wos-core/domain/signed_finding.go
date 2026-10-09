package domain

import (
	"fmt"
	"strings"
)

// Findings address an existing obligation; they never add criteria or scope.
type SignedFinding struct {
	ID             ID     `json:"id"`
	CriterionID    *ID    `json:"criterion_id,omitempty"`
	RequirementRef string `json:"requirement_ref,omitempty"`
	Description    string `json:"description"`
}

func (f SignedFinding) Validate() error {
	if f.ID.Validate() != nil || strings.TrimSpace(f.Description) == "" || len(f.Description) > 16384 || (f.CriterionID == nil) == (f.RequirementRef == "") {
		return NewError(ErrorCodeInvalidArgument, "finding needs identity, description and exactly one existing obligation")
	}
	return nil
}

func (f SignedFinding) ValidateAgainst(spec WorkContractSpec) error {
	if err := f.Validate(); err != nil {
		return err
	}
	if f.CriterionID != nil {
		for _, c := range spec.Criteria {
			if c.ID == *f.CriterionID && c.Status == CriterionStatusActive {
				return nil
			}
		}
		return NewError(ErrorCodeCriterion, "finding cannot add an unplanned criterion")
	}
	refs := map[string]bool{"task.title": true, "task.description": true, "outcome_intent": true, "objective_intent": true}
	for name, list := range map[string][]string{"instructions": spec.ExecutionSpec.Instructions, "constraints": spec.ExecutionSpec.Constraints, "deliverables": spec.ExecutionSpec.Deliverables} {
		for i := range list {
			refs[fmt.Sprintf("execution_spec.%s[%d]", name, i)] = true
		}
	}
	if !refs[f.RequirementRef] {
		return NewError(ErrorCodeInvalidArgument, "finding must reference an existing contracted obligation")
	}
	return nil
}
