// Generates the executable catalog contract; legacy REST routes remain in openapi.yaml.
package main

import (
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"os"
	"strings"
)

func main() {
	schemas := map[string]any{}
	paths := map[string]any{}
	header := map[string]any{"name": "Idempotency-Key", "in": "header", "required": true, "schema": map[string]any{"type": "string", "minLength": 16, "maxLength": 128}}
	for _, d := range commands.NewCatalog(nil, nil).Descriptors() {
		name := "Catalog_" + d.Name
		schemas[name] = d.Schema
		paths["/api/v1/commands/"+d.Name] = map[string]any{"post": map[string]any{"operationId": "command_" + d.Name, "description": "Application command. Retry an uncertain result with identical intent/key; reconcile stale versions before a new intent. Payload cap 256 KiB.", "parameters": []any{header}, "requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "object", "required": []string{"command"}, "additionalProperties": false, "properties": map[string]any{"command": map[string]any{"$ref": "#/components/schemas/" + name}}}}}}, "responses": map[string]any{"200": map[string]any{"description": "Persisted result or commit receipt (result_omitted=true); includes command_id, outcome_revision and idempotent_replay."}, "400": map[string]any{"description": "Invalid intent or key."}, "401": map[string]any{"description": "Credential absent, revoked or expired."}, "403": map[string]any{"description": "Namespace or permission denied."}, "409": map[string]any{"description": "Version, transaction, idempotency or lease conflict; never silently change intent/version."}, "413": map[string]any{"description": "Payload exceeds byte limit."}}}}
	}
	for _, path := range []string{"/api/v1/capabilities", "/api/v1/namespaces/{namespace_id}/commands/{command_id}", "/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/work-contracts", "/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/work-contracts/{contract_id}", "/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/work-contracts/{contract_id}/spec", "/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/work-contracts/{contract_id}/checkpoints", "/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/work-contracts/{contract_id}/submissions", "/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/work-submissions/{submission_id}", "/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/available-work"} {
		parameters := []any{}
		for _, segment := range strings.Split(path, "/") {
			if strings.HasPrefix(segment, "{") {
				parameters = append(parameters, map[string]any{"name": strings.Trim(segment, "{}"), "in": "path", "required": true, "schema": map[string]any{"type": "string"}})
			}
		}
		paths[path] = map[string]any{"get": map[string]any{"description": "Authorized execution-contract read. Collections accept limit (1..100) and scope-bound cursor. Specs are immutable resources. Effective expiry is read-only; receipt lookup is restricted to authenticated principal.", "parameters": parameters, "responses": map[string]any{"200": map[string]any{"description": "Typed result, page or retained receipt; omission metadata remains explicit."}, "403": map[string]any{"description": "Authorization denied."}, "404": map[string]any{"description": "Not found in scope or receipt not retained."}}}}
	}
	document := map[string]any{"openapi": "3.1.0", "info": map[string]any{"title": "WOS generated command catalog", "version": "0.1.0-dev"}, "paths": paths, "security": []any{map[string]any{"bearerAuth": []string{}}, map[string]any{"browserSession": []string{}}}, "components": map[string]any{"schemas": schemas, "securitySchemes": map[string]any{"bearerAuth": map[string]any{"type": "http", "scheme": "bearer"}, "browserSession": map[string]any{"type": "apiKey", "in": "cookie", "name": "wos_session"}}}}
	b, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		panic(err)
	}
	b = append(b, '\n')
	if err = os.WriteFile("packages/wos-api/commands.openapi.json", b, 0644); err != nil {
		panic(err)
	}
}
