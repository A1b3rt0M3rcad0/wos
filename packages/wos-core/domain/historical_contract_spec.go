package domain

// Historical contracts authenticated their entity references with the original
// namespace_id field. Reconstruct only these typed references for validation;
// do not rewrite a stored specification, digest, or arbitrary user text.
func historicalContractSpecDigest(spec WorkContractSpec) (string, error) {
	type historicalRef struct {
		NamespaceID ID         `json:"namespace_id"`
		OutcomeID   ID         `json:"outcome_id"`
		Kind        EntityKind `json:"kind"`
		ID          ID         `json:"id"`
	}
	ref := func(value EntityRef) historicalRef {
		return historicalRef{value.TenantID, value.OutcomeID, value.Kind, value.ID}
	}
	type historicalCriterion struct {
		SuccessCriterion
		OwnerRef historicalRef `json:"owner_ref"`
	}
	type historicalExecution struct {
		ExecutionSpec
		ContextRefs []historicalRef `json:"context_refs"`
	}
	type historicalSpec struct {
		WorkContractSpec
		ExecutionSpec historicalExecution   `json:"execution_spec"`
		Criteria      []historicalCriterion `json:"criteria"`
		Dependencies  []historicalRef       `json:"dependencies"`
	}
	spec = NormalizeContractSpec(spec)
	old := historicalSpec{
		WorkContractSpec: spec,
		ExecutionSpec:    historicalExecution{spec.ExecutionSpec, []historicalRef{}},
		Criteria:         []historicalCriterion{}, Dependencies: []historicalRef{},
	}
	for _, value := range spec.ExecutionSpec.ContextRefs {
		old.ExecutionSpec.ContextRefs = append(old.ExecutionSpec.ContextRefs, ref(value))
	}
	for _, value := range spec.Criteria {
		old.Criteria = append(old.Criteria, historicalCriterion{value, ref(value.OwnerRef)})
	}
	for _, value := range spec.Dependencies {
		old.Dependencies = append(old.Dependencies, ref(value))
	}
	return SemanticDigest(old)
}
