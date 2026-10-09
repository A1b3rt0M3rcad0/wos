package application

import (
	"bytes"
	"encoding/json"
)

// Exact v2 transport metadata is isolated from the unchanged v1 aggregate
// encoder. Issued payload bytes/proofs and stored response_json remain opaque.
func signedWireJSON(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var tree any
	if err = decoder.Decode(&tree); err != nil {
		return nil, err
	}
	quoteSignedCounters(tree)
	return json.Marshal(tree)
}
func quoteSignedCounters(value any) {
	switch node := value.(type) {
	case map[string]any:
		for key, child := range node {
			if key == "payload" || key == "response_json" || key == "external_context" {
				continue
			}
			// The Namespace CAS version shares a name with the literal signed
			// protocol marker. Quote only the mutable Namespace projection.
			if key == "protocol_version" && node["phase"] != nil && node["writer_epoch"] != nil {
				if number, ok := child.(json.Number); ok {
					node[key] = string(number)
					continue
				}
			}
			switch key {
			case "last_fencing_token", "fencing_token", "round", "version", "revision", "lease_version", "contract_version", "policy_revision", "criterion_revision", "outcome_revision", "work_item_version", "work_item_version_at_acquire", "outcome_revision_at_acquire":
				if number, ok := child.(json.Number); ok {
					node[key] = string(number)
					continue
				}
			}
			quoteSignedCounters(child)
		}
	case []any:
		for _, child := range node {
			quoteSignedCounters(child)
		}
	}
}
func (r WorkContractResult) MarshalJSON() ([]byte, error) {
	type legacy WorkContractResult
	if r.Contract.SignedBinding == nil {
		return json.Marshal(legacy(r))
	}
	return signedWireJSON(legacy(r))
}
func (r SignedReviewContractResult) MarshalJSON() ([]byte, error) {
	type wire SignedReviewContractResult
	return signedWireJSON(wire(r))
}
func (r SignedInterventionResult) MarshalJSON() ([]byte, error) {
	type wire SignedInterventionResult
	return signedWireJSON(wire(r))
}
func (r MutationResult[T]) MarshalJSON() ([]byte, error) {
	type legacy MutationResult[T]
	signed := false
	switch value := any(r.Value).(type) {
	case WorkContractResult:
		signed = value.Contract.SignedBinding != nil
	case SignedReviewContractResult, SignedReturnResult, SignedInterventionResult:
		signed = true
	}
	if signed {
		return signedWireJSON(legacy(r))
	}
	return json.Marshal(legacy(r))
}
