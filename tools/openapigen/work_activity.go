package main

import (
	"reflect"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
)

func workActivityContracts(paths, schemas map[string]any) {
	schemas["WorkActivityPage"] = commands.Schema(reflect.TypeFor[application.WorkActivityPage]())
	schemas["WorkAttention"] = commands.Schema(reflect.TypeFor[application.WorkAttention]())
	scope := func() []any {
		return []any{
			map[string]any{"name": "namespace_id", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}},
			map[string]any{"name": "outcome_id", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}},
		}
	}
	responses := func(schema, description string) map[string]any {
		return map[string]any{
			"200": map[string]any{"description": description, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/" + schema}}}},
			"400": map[string]any{"description": "Malformed identifier or query format."},
			"403": map[string]any{"description": "Current Outcome authorization denied."},
			"404": map[string]any{"description": "Referenced work not found in scope."},
			"422": map[string]any{"description": "Invalid ID, limit, filter or cursor."},
		}
	}
	parameters := scope()
	for _, name := range []string{"work_item_id", "contract_id"} {
		parameters = append(parameters, map[string]any{"name": name, "in": "query", "schema": map[string]any{"type": "string", "format": "uuid"}})
	}
	parameters = append(parameters,
		map[string]any{"name": "descending", "in": "query", "description": "Read newest activity first; its cursor continues toward older records.", "schema": map[string]any{"type": "boolean", "default": false}},
		map[string]any{"name": "limit", "in": "query", "schema": map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "default": 25}},
		map[string]any{"name": "cursor", "in": "query", "schema": map[string]any{"type": "string"}},
	)
	paths["/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/work-activity"] = map[string]any{"get": map[string]any{
		"operationId": "listWorkActivity", "description": "Immutable authorized Outcome work activity. Optional WorkItem/contract filters and opaque scope/filter-bound cursors. Observations never complete work, assess criteria or grant execution authority. Cache-Control: no-store.",
		"parameters": parameters, "responses": responses("WorkActivityPage", "Bounded activity history and continuation cursor."),
	}}
	parameters = append(scope(), map[string]any{"name": "work_item_id", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}})
	paths["/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/work-items/{work_item_id}/attention"] = map[string]any{"get": map[string]any{
		"operationId": "getWorkAttention", "description": "Current authorized WorkItem attention projection. Reading never changes lifecycle, renews authority or completes work. Cache-Control: no-store.",
		"parameters": parameters, "responses": responses("WorkAttention", "Current attention signals and Outcome revision."),
	}}
}
