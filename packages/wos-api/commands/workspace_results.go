package commands

import (
	"reflect"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// WorkspaceCommandResults supplies the additive workspace command response
// contracts to HTTP documentation and MCP without maintaining a second DTO.
func WorkspaceCommandResults() map[string]map[string]any {
	resultTypes := map[string]reflect.Type{
		"append_entity_note":           reflect.TypeFor[application.MutationResult[domain.EntityNote]](),
		"create_plan_items":            reflect.TypeFor[application.MutationResult[application.CreatePlanItemsResult]](),
		"set_work_item_execution_spec": reflect.TypeFor[application.MutationResult[domain.WorkItem]](),
		"set_work_item_not_before":     reflect.TypeFor[application.MutationResult[domain.WorkItem]](),
		"set_work_item_objective":      reflect.TypeFor[application.MutationResult[domain.WorkItem]](),
		"set_objective_parent":         reflect.TypeFor[application.MutationResult[domain.Objective]](),
		"register_evidence_with_links": reflect.TypeFor[application.MutationResult[application.RegisterEvidenceWithLinksResult]](),
		"register_artifact_in_context": reflect.TypeFor[application.MutationResult[application.RegisterArtifactInContextResult]](),
		"propose_decision_in_context":  reflect.TypeFor[application.MutationResult[application.ProposeDecisionInContextResult]](),
	}
	results := make(map[string]map[string]any, len(resultTypes))
	for name, resultType := range resultTypes {
		results[name] = map[string]any{"type": "object", "anyOf": []any{Schema(resultType), map[string]any{
			"type": "object", "required": []string{"command_id", "outcome_revision", "idempotent_replay", "result_omitted"},
			"properties": workspaceReceiptSchema(),
		}}}
	}
	return results
}

func workspaceReceiptSchema() map[string]any {
	properties := map[string]any{
		"command_id": Schema(reflect.TypeFor[domain.ID]()), "outcome_revision": map[string]any{"type": "integer"},
		"idempotent_replay": map[string]any{"type": "boolean"}, "result_omitted": map[string]any{"const": true},
		"created_refs": map[string]any{"type": "object", "maxProperties": 50, "additionalProperties": Schema(reflect.TypeFor[domain.EntityRef]())},
		"created_items": map[string]any{"type": "array", "maxItems": 50, "items": map[string]any{"type": "object", "properties": map[string]any{
			"local_key": map[string]any{"type": "string"}, "ref": Schema(reflect.TypeFor[domain.EntityRef]()),
			"version":       map[string]any{"type": "string", "pattern": "^[1-9][0-9]{0,19}$"},
			"criterion_ids": map[string]any{"type": "array", "maxItems": 100, "items": Schema(reflect.TypeFor[domain.ID]())},
		}}},
		"evidence_link_ids": map[string]any{"type": "array", "maxItems": 100, "items": Schema(reflect.TypeFor[domain.ID]())},
	}
	for _, kind := range []string{"entity", "evidence", "artifact", "observation", "decision", "relation"} {
		properties[kind+"_id"] = Schema(reflect.TypeFor[domain.ID]())
	}
	return properties
}
