package commands

import (
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"strconv"
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
	operationalReceiptIdentities(out, result)
	return out, nil
}

// Retain the exact native review/recovery targets without copying result
// material, reasons, policy records or any credential information. Integration
// deliveries lack this typed Task binding and are deliberately not interpreted
// as TaskDelivery records.
func operationalReceiptIdentities(out map[string]any, result map[string]json.RawMessage) {
	var delivery struct {
		ID         domain.ID    `json:"id"`
		Scope      domain.Scope `json:"scope"`
		WorkItemID domain.ID    `json:"work_item_id"`
		Digest     string       `json:"delivery_digest"`
		Round      uint64       `json:"round,string"`
	}
	if json.Unmarshal(result["delivery"], &delivery) == nil && delivery.ID.Validate() == nil && delivery.Scope.Validate() == nil && delivery.WorkItemID.Validate() == nil && domain.ValidSignedDigest(delivery.Digest) && delivery.Round > 0 {
		// Preserve the exact quoted decimal bytes, including values above 2^53.
		var fields map[string]json.RawMessage
		_ = json.Unmarshal(result["delivery"], &fields)
		var round string
		if json.Unmarshal(fields["round"], &round) != nil || round != strconv.FormatUint(delivery.Round, 10) {
			return
		}
		out["delivery_id"], out["delivery_digest"] = delivery.ID, delivery.Digest
		out["delivery_round"] = fields["round"]
		out["work_item_id"], out["scope"] = delivery.WorkItemID, delivery.Scope
		var withdrawal struct {
			ID             domain.ID    `json:"id"`
			Scope          domain.Scope `json:"scope"`
			WorkItemID     domain.ID    `json:"work_item_id"`
			DeliveryID     domain.ID    `json:"delivery_id"`
			DeliveryDigest string       `json:"delivery_digest"`
			Round          uint64       `json:"round,string"`
		}
		if json.Unmarshal(result["withdrawal"], &withdrawal) == nil && withdrawal.ID.Validate() == nil && withdrawal.Scope == delivery.Scope && withdrawal.WorkItemID == delivery.WorkItemID && withdrawal.DeliveryID == delivery.ID && withdrawal.DeliveryDigest == delivery.Digest && withdrawal.Round == delivery.Round {
			out["withdrawal_id"] = withdrawal.ID
		}
	}
	var authority struct {
		ID          domain.ID    `json:"id"`
		Scope       domain.Scope `json:"scope"`
		ObjectiveID domain.ID    `json:"objective_id"`
	}
	var objective struct {
		ID    domain.ID    `json:"id"`
		Scope domain.Scope `json:"scope"`
	}
	if json.Unmarshal(result["authority"], &authority) == nil && authority.ID.Validate() == nil && authority.Scope.Validate() == nil && authority.ObjectiveID.Validate() == nil &&
		json.Unmarshal(result["objective"], &objective) == nil && objective.ID == authority.ObjectiveID && objective.Scope == authority.Scope {
		out["completion_authority_id"] = authority.ID
	}
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
