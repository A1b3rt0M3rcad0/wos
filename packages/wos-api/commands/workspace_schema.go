package commands

import (
	"reflect"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func workspaceFieldBounds(owner reflect.Type, name string, schema map[string]any) {
	if name == "permissions" || name == "permitted_operations" {
		schema["maxItems"] = domain.MaxPermissionValues
	}
	if name == "initial_criteria" {
		schema["maxItems"] = application.MaxInitialCriteria
	}
	switch owner {
	case reflect.TypeFor[application.CreatePlanItemsCommand]():
		if name == "items" {
			schema["minItems"], schema["maxItems"] = 1, application.MaxPlanItems
		}
	case reflect.TypeFor[application.AppendEntityNoteCommand]():
		if name == "body" {
			schema["minLength"], schema["maxLength"], schema["x-wos-max-bytes"] = 1, domain.MaxEntityNoteBodyBytes, domain.MaxEntityNoteBodyBytes
		}
		if name == "artifact_refs" || name == "evidence_refs" {
			schema["maxItems"] = domain.MaxEntityNoteReferences
		}
	case reflect.TypeFor[application.RegisterEvidenceWithLinksCommand]():
		if name == "links" {
			schema["minItems"], schema["maxItems"] = 1, 100
		}
	}
}

// WorkspaceReadSchema describes exact DTO spelling for the additive read
// surfaces; old command and signed-protocol schemas retain their contracts.
func WorkspaceReadSchema(t reflect.Type) map[string]any {
	schema := SignedSchema(t)
	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		return schema
	}
	switch t {
	case reflect.TypeFor[application.WorkspaceViewQuery]():
		properties["subject"] = WorkspaceReadSchema(reflect.TypeFor[application.WorkspaceSubject]())
		properties["limit"] = map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "default": 20}
	case reflect.TypeFor[application.WorkspaceSubject]():
		properties["kind"] = map[string]any{"type": "string", "enum": []string{"entity", "criterion", "roadmap_revision", "execution_record"}}
		properties["record_kind"] = map[string]any{"type": "string", "enum": []string{"contract", "submission", "review_case"}}
	case reflect.TypeFor[application.WorkspaceCardsQuery]():
		properties["refs"].(map[string]any)["maxItems"] = 100
		properties["limit"] = map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "default": 50}
	}
	return schema
}
