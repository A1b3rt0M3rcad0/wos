package commands

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// These refinements belong only to additive operational contracts. In
// particular, inherited Criterion/WorkItem counters and signed material keep
// their historical JSON representation and canonical bytes.
func operationalFieldSchema(owner reflect.Type, name string, schema map[string]any) map[string]any {
	switch owner {
	case reflect.TypeFor[application.SubmitTaskForReviewCommand]():
		switch name {
		case "expected_work_item_version", "expected_contract_version":
			schema["pattern"] = operationalUint64Pattern(true)
		case "fencing_token":
			schema["pattern"] = operationalUint64Pattern(false)
		case "legacy_material":
			schema["description"] = "Inline material for the exact current legacy lease. Delivery ends executor authority and awaits review; it does not accept the Task."
		case "submission_material":
			schema["description"] = "Exact contracts_v1 WorkResultMaterial, retaining its existing JSON encoding. The full command is bounded to 256 KiB; scoped material, current Criteria and immutable source revisions are checked by the Application."
		}
		switch name {
		case "claim_id", "legacy_material", "contract_id", "authority", "submission_material", "supersedes_submission_id":
			return map[string]any{"anyOf": []any{schema, map[string]any{"type": "null"}}}
		}
	case reflect.TypeFor[application.AcceptTaskDeliveryCommand](), reflect.TypeFor[application.RequestTaskDeliveryChangesCommand](), reflect.TypeFor[application.WithdrawTaskDeliveryCommand]():
		switch name {
		case "expected_work_item_version", "expected_delivery_round":
			schema["pattern"] = operationalUint64Pattern(true)
		case "expected_delivery_digest":
			schema["pattern"] = "^sha256:[0-9a-f]{64}$"
		case "reason":
			operationalTextBounds(schema, 4096)
		case "findings":
			schema["minItems"], schema["maxItems"] = 1, 100
			operationalTextBounds(schema["items"].(map[string]any), 4096)
		case "assessment_refs":
			schema["maxItems"] = 100
		}
	case reflect.TypeFor[application.TaskDeliveryResult]():
		if name == "expected_work_item_version" || name == "expected_contract_version" {
			schema["pattern"] = operationalUint64Pattern(true)
		}
	case reflect.TypeFor[domain.TaskDeliveryMaterial]():
		switch name {
		case "summary":
			operationalTextBounds(schema, 16384)
		case "artifacts":
			schema["maxItems"] = 100
			artifact := schema["items"].(map[string]any)
			artifact["required"] = []string{"artifact_id"}
			artifact["anyOf"] = []any{
				map[string]any{"required": []string{"source_version"}, "properties": map[string]any{"source_version": map[string]any{"type": "string", "minLength": 1}}},
				map[string]any{"required": []string{"checksum"}, "properties": map[string]any{"checksum": map[string]any{"type": "string", "minLength": 1}}},
			}
		case "evidence_ids":
			schema["maxItems"], schema["uniqueItems"] = 500, true
		case "criterion_evidence":
			schema["maxItems"] = 100
			properties := schema["items"].(map[string]any)["properties"].(map[string]any)
			evidence := properties["evidence_ids"].(map[string]any)
			evidence["minItems"], evidence["maxItems"], evidence["uniqueItems"] = 1, 500, true
			appendSchemaRequired(schema["items"].(map[string]any), "evidence_ids")
		}
	case reflect.TypeFor[domain.TaskDelivery]():
		switch name {
		case "round":
			schema["pattern"] = operationalUint64Pattern(true)
		case "delivery_digest", "definition_digest":
			schema["pattern"] = "^sha256:[0-9a-f]{64}$"
		case "criterion_snapshot":
			schema["maxItems"] = 100
		}
	case reflect.TypeFor[domain.TaskDeliveryAuthority]():
		if name == "kind" {
			schema["enum"] = []string{string(domain.TaskDeliveryAuthorityLegacyLease), string(domain.TaskDeliveryAuthorityContractsV1)}
		}
	case reflect.TypeFor[domain.TaskDeliveryDecision]():
		if name == "result" {
			schema["enum"] = []string{string(domain.TaskDeliveryDecisionAccepted), string(domain.TaskDeliveryDecisionChangesRequested)}
		}
	case reflect.TypeFor[application.SetObjectiveCompletionModeCommand]():
		switch name {
		case "mode":
			schema["enum"] = []string{string(domain.ObjectiveCompletionModeManual), string(domain.ObjectiveCompletionModeAutomatic)}
		case "expected_version":
			schema["pattern"] = operationalUint64Pattern(true)
		case "expected_outcome_revision":
			schema["pattern"] = operationalUint64Pattern(false)
		case "reason":
			schema["minLength"], schema["maxLength"] = 1, 1024
		}
	case reflect.TypeFor[application.ConfigureOutcomeOperationalProfileCommand]():
		if name == "reason" {
			schema["minLength"], schema["maxLength"] = 1, 1024
		}
	case reflect.TypeFor[application.ReportIssueWithBlockersCommand]():
		switch name {
		case "observed_outcome_revision":
			schema["pattern"] = operationalUint64Pattern(false)
		case "blockers":
			schema["minItems"], schema["maxItems"] = 1, 64
		}
	case reflect.TypeFor[application.ApplyDependencyChangesCommand](), reflect.TypeFor[application.DependencyChangesPreviewQuery]():
		if name == "changes" || name == "expected_sources" {
			schema["minItems"], schema["maxItems"] = 1, 128
		}
	}
	return schema
}

func operationalObjectSchema(owner reflect.Type, schema map[string]any) {
	switch owner {
	case reflect.TypeFor[application.SubmitTaskForReviewCommand]():
		schema["oneOf"] = []any{
			map[string]any{
				"required": []string{"claim_id", "fencing_token", "legacy_material"},
				"properties": map[string]any{
					"claim_id": map[string]any{"type": "string"}, "fencing_token": map[string]any{"pattern": operationalUint64Pattern(true)}, "legacy_material": map[string]any{"type": "object"},
					"contract_id": map[string]any{"type": "null"}, "authority": map[string]any{"type": "null"}, "expected_contract_version": map[string]any{"type": "null"}, "submission_material": map[string]any{"type": "null"}, "supersedes_submission_id": map[string]any{"type": "null"},
				},
			},
			map[string]any{
				"required": []string{"contract_id", "authority", "expected_contract_version", "submission_material"},
				"properties": map[string]any{
					"contract_id": map[string]any{"type": "string"}, "authority": map[string]any{"type": "object"}, "expected_contract_version": map[string]any{"type": "string"}, "submission_material": map[string]any{"type": "object"},
					"claim_id": map[string]any{"type": "null"}, "fencing_token": map[string]any{"const": "0"}, "legacy_material": map[string]any{"type": "null"},
				},
			},
		}
		schema["x-wos-max-bytes"] = MaxPayloadBytes
	case reflect.TypeFor[application.ReportIssueWithBlockersCommand]():
		appendSchemaRequired(schema, "blockers")
	case reflect.TypeFor[application.ApplyDependencyChangesCommand](), reflect.TypeFor[application.DependencyChangesPreviewQuery]():
		appendSchemaRequired(schema, "expected_sources", "changes")
	}
}

func appendSchemaRequired(schema map[string]any, names ...string) {
	required, _ := schema["required"].([]string)
	for _, name := range names {
		found := false
		for _, existing := range required {
			found = found || existing == name
		}
		if !found {
			required = append(required, name)
		}
	}
	schema["required"] = required
}

func operationalTextBounds(schema map[string]any, maximumBytes int) {
	schema["minLength"], schema["maxLength"], schema["x-wos-max-bytes"] = 1, maximumBytes, maximumBytes
}

// Keep the lexical boundary exact, including the twenty-digit uint64 maximum.
// This does not alter any historical or signed decimal schema.
func operationalUint64Pattern(positive bool) string {
	const maximum = "18446744073709551615"
	alternatives := []string{"[1-9][0-9]{0,18}"}
	if !positive {
		alternatives = append(alternatives, "0")
	}
	for i := 1; i < len(maximum); i++ {
		if maximum[i] == '0' {
			continue
		}
		last := maximum[i] - 1
		digit := "0"
		if last > '0' {
			digit = fmt.Sprintf("[0-%c]", last)
		}
		alternatives = append(alternatives, maximum[:i]+digit+fmt.Sprintf("[0-9]{%d}", len(maximum)-i-1))
	}
	alternatives = append(alternatives, maximum)
	return "^(" + strings.Join(alternatives, "|") + ")$"
}
