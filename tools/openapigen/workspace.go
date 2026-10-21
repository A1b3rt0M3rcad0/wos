package main

import (
	"reflect"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func workspaceContracts(paths, schemas map[string]any) {
	for name, typ := range map[string]reflect.Type{
		"WorkspaceViewQuery":    reflect.TypeFor[application.WorkspaceViewQuery](),
		"EntityWorkspaceView":   reflect.TypeFor[application.EntityWorkspaceView](),
		"WorkspaceCardsQuery":   reflect.TypeFor[application.WorkspaceCardsQuery](),
		"WorkspaceCardsPage":    reflect.TypeFor[application.WorkspaceCardsPage](),
		"WorkspaceTimelinePage": reflect.TypeFor[application.WorkspaceTimelinePage](),
		"EntityNote":            reflect.TypeFor[domain.EntityNote](),
		"WorkspaceRules":        reflect.TypeFor[application.WorkspaceRules](),
		"ErrorDetails":          reflect.TypeFor[domain.ErrorDetails](),
	} {
		schemas[name] = commands.SignedSchema(typ)
	}
	schemas["WorkspaceViewQuery"] = commands.WorkspaceReadSchema(reflect.TypeFor[application.WorkspaceViewQuery]())
	schemas["WorkspaceCardsQuery"] = commands.WorkspaceReadSchema(reflect.TypeFor[application.WorkspaceCardsQuery]())
	for name, schema := range commands.WorkspaceCommandResults() {
		schemas["Result_"+name] = schema
	}
	base := "/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}"
	scope := func() []any {
		return []any{
			map[string]any{"name": "namespace_id", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}},
			map[string]any{"name": "outcome_id", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}},
		}
	}
	responses := func(schema, description string) map[string]any {
		return map[string]any{
			"200": map[string]any{"description": description, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/" + schema}}}},
			"400": map[string]any{"description": "Malformed request, subject or cursor."},
			"403": map[string]any{"description": "Current scoped authorization denied."},
			"404": map[string]any{"description": "Subject not found in the authorized scope."},
			"409": map[string]any{"description": "Historical structural context changed; explicitly restart the affected read."},
			"422": map[string]any{"description": "Invalid subject, page, references or cursor binding."},
		}
	}
	for _, read := range []struct{ path, operation, input, output, description string }{
		{"workspace-view", "getEntityWorkspaceView", "WorkspaceViewQuery", "EntityWorkspaceView", "Read-only POST for one structured subject at a coherent authorized Outcome snapshot. Exact decimal versions/revisions; facet coverage/cursors distinguish incomplete data from absence. Capability hints never authorize mutations. No idempotency key or side effects."},
		{"workspace-cards", "getWorkspaceCards", "WorkspaceCardsQuery", "WorkspaceCardsPage", "Read-only POST: at most 100 explicit refs OR one structural parent with bounded pagination. One authorized Outcome revision per response; compact metadata without per-card inspector or history hydration. Unknown authority is explicit."},
	} {
		paths[base+"/"+read.path] = map[string]any{"post": map[string]any{
			"operationId": read.operation, "description": read.description + " Cache-Control: no-store; request and response limit 256 KiB.", "parameters": scope(),
			"requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/" + read.input}}}},
			"responses":   responses(read.output, "Coherent bounded workspace projection."),
		}}
	}
	parameters := scope()
	for _, field := range []string{"kind", "entity_id", "criterion_id", "cursor"} {
		parameters = append(parameters, map[string]any{"name": field, "in": "query", "schema": map[string]any{"type": "string"}})
	}
	parameters = append(parameters,
		map[string]any{"name": "include_descendants", "in": "query", "schema": map[string]any{"type": "boolean", "default": false}},
		map[string]any{"name": "limit", "in": "query", "schema": map[string]any{"type": "integer", "minimum": 1, "maximum": 20, "default": 20}},
	)
	paths[base+"/workspace-timeline"] = map[string]any{"get": map[string]any{"operationId": "getWorkspaceTimeline", "description": "Newest-first authorized documentary/operational facts, at most 20 previews. Optional kind and entity_id select a subject; otherwise the Outcome. Stable revision/event-index ordering, frozen historical frontier and source provenance. Follow next_cursor; structural membership changes require an explicit restart. Existing /timeline is unchanged. Cache-Control: no-store.", "parameters": parameters, "responses": responses("WorkspaceTimelinePage", "Frozen historical page with explicit origin and update indication.")}}
	parameters = append(scope(), map[string]any{"name": "note_id", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}})
	paths[base+"/entity-notes/{note_id}"] = map[string]any{"get": map[string]any{"operationId": "getEntityNote", "description": "Read a complete scoped immutable documentary note (body at most 16 KiB). Authorship, correction lineage and record time remain explicit. This does not grant signed execution authority or assess criteria. Cache-Control: no-store.", "parameters": parameters, "responses": responses("EntityNote", "Complete authorized immutable note.")}}
}
