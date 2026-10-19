package commands

import (
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

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
	workspaceReceiptIdentities(out, result)
	return out, nil
}

// Preserve identities from the bounded workspace compositions without copying
// descriptions, Criteria definitions or signed/documentary payloads. These are
// retained result references, never execution authority or acceptance.
func workspaceReceiptIdentities(out map[string]any, result map[string]json.RawMessage) {
	if raw, ok := result["refs"]; ok {
		var refs map[string]domain.EntityRef
		if json.Unmarshal(raw, &refs) == nil && len(refs) <= 50 {
			valid := true
			for key, ref := range refs {
				valid = valid && len(key) <= 64 && ref.Validate() == nil
			}
			if valid {
				out["created_refs"] = refs
				var items []struct {
					LocalKey string           `json:"local_key"`
					Ref      domain.EntityRef `json:"ref"`
					Version  json.RawMessage  `json:"version"`
					Criteria []struct {
						ID domain.ID `json:"id"`
					} `json:"criteria"`
				}
				if json.Unmarshal(result["items"], &items) == nil && len(items) <= 50 {
					compact := []map[string]any{}
					criterionCount := 0
					for _, item := range items {
						if ref, ok := refs[item.LocalKey]; !ok || ref != item.Ref {
							return
						}
						ids := []domain.ID{}
						criterionCount += len(item.Criteria)
						if criterionCount > 100 {
							return
						}
						for _, criterion := range item.Criteria {
							if criterion.ID.Validate() != nil {
								return
							}
							ids = append(ids, criterion.ID)
						}
						compact = append(compact, map[string]any{"local_key": item.LocalKey, "ref": item.Ref, "version": item.Version, "criterion_ids": ids})
					}
					out["created_items"] = compact
				}
			}
		}
	}
	for _, kind := range []string{"evidence", "artifact", "observation", "decision", "relation"} {
		var record struct {
			ID      domain.ID       `json:"id"`
			Version json.RawMessage `json:"version"`
		}
		if raw, ok := result[kind]; ok && json.Unmarshal(raw, &record) == nil && record.ID.Validate() == nil {
			out[kind+"_id"] = record.ID
			if len(record.Version) != 0 {
				out[kind+"_version"] = record.Version
			}
		}
	}
	var links []struct {
		ID domain.ID `json:"id"`
	}
	if raw, ok := result["links"]; ok && json.Unmarshal(raw, &links) == nil && len(links) <= 100 {
		ids := []domain.ID{}
		for _, link := range links {
			if link.ID.Validate() != nil {
				return
			}
			ids = append(ids, link.ID)
		}
		out["evidence_link_ids"] = ids
	}
}
