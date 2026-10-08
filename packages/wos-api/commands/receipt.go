package commands

import "encoding/json"

// CommitReceipt identifies committed material even when its full result exceeds
// the transport bound. It does not grant current execution authority.
func CommitReceipt(value any) (map[string]any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	var envelope map[string]json.RawMessage
	if err = json.Unmarshal(raw, &envelope); err != nil {
		return nil, err
	}
	out := map[string]any{"command_id": envelope["command_id"], "outcome_revision": envelope["outcome_revision"], "idempotent_replay": envelope["idempotent_replay"], "result_omitted": true}
	var result map[string]json.RawMessage
	if json.Unmarshal(envelope["value"], &result) != nil {
		return out, nil
	}
	if v, ok := result["acquired"]; ok {
		out["acquired"] = v
		var nested map[string]json.RawMessage
		if json.Unmarshal(result["result"], &nested) == nil {
			result = nested
		}
	}
	var contract map[string]json.RawMessage
	if json.Unmarshal(result["contract"], &contract) == nil {
		for _, key := range []string{"id", "work_item_id", "scope", "version", "lease_version", "spec_digest", "execution_id", "fencing_token", "latest_checkpoint_id", "latest_submission_id"} {
			if v, ok := contract[key]; ok {
				target := key
				if key == "id" {
					target = "contract_id"
				}
				if key == "version" {
					target = "contract_version"
				}
				out[target] = v
			}
		}
	} else if id, ok := result["id"]; ok {
		out["entity_id"] = id
	}
	return out, nil
}
