// Generates the executable catalog contract; legacy REST routes remain in openapi.yaml.
package main

import (
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"os"
	"reflect"
	"strings"
)

func main() {
	version, err := os.ReadFile("VERSION")
	if err != nil {
		panic(err)
	}
	schemas := map[string]any{}
	for name, typed := range map[string]reflect.Type{
		"SignedEnvelopeV2":            reflect.TypeFor[signing.Envelope](),
		"SignedWorkSpecificationV2":   reflect.TypeFor[signing.SpecPayload[domain.SignedWorkSpec]](),
		"SignedReviewSpecificationV2": reflect.TypeFor[signing.SpecPayload[application.SignedReviewSpec]](),
		"SignedAuthorityV2":           reflect.TypeFor[signing.AuthorityPayload](),
		"SignedWorkReturnV2":          reflect.TypeFor[signing.WorkReturnPayload[application.SignedReturnMaterial]](),
		"SignedReviewReturnV2":        reflect.TypeFor[signing.ReviewReturnPayload[application.SignedReviewMaterial]](),
		"SignedAcceptanceReceiptV2":   reflect.TypeFor[signing.ReceiptPayload](),
		"SignedKeyEnrollmentV2":       reflect.TypeFor[signing.EnrollmentPayload](),
	} {
		schemas[name] = commands.SignedSchema(typed)
	}
	paths := map[string]any{}
	schemas["ReferenceCandidate"] = commands.Schema(reflect.TypeFor[ports.ReferenceCandidate]())
	schemas["ReferencePage"] = commands.Schema(reflect.TypeFor[application.ReferencePage]())
	refsParameters := []any{}
	for _, p := range []string{"namespace_id", "outcome_id"} {
		refsParameters = append(refsParameters, map[string]any{"name": p, "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}})
	}
	for _, p := range []string{"kind", "query", "lifecycle", "priority", "cursor", "id"} {
		refsParameters = append(refsParameters, map[string]any{"name": p, "in": "query", "required": p == "kind", "schema": map[string]any{"type": "string"}})
	}
	refsParameters = append(refsParameters, map[string]any{"name": "limit", "in": "query", "schema": map[string]any{"type": "integer", "minimum": 1, "maximum": 100, "default": 25}})
	paths["/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/references"] = map[string]any{"get": map[string]any{"operationId": "searchReferences", "description": "Authorized bounded reference metadata. kind is a comma-separated allowlist. Query uses ASCII case-insensitive substring matching, preserving Unicode otherwise. Deterministic kind/ID keysets bind scope, filters and Outcome revision. id resolves a selected candidate independently of paging; cannot combine with query/cursor. Results never grant mutation authority.", "parameters": refsParameters, "responses": map[string]any{"200": map[string]any{"description": "Bounded candidates and revision-bound next cursor", "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/ReferencePage"}}}}, "422": map[string]any{"description": "Invalid kind, filter, ID, limit or cursor binding"}, "403": map[string]any{"description": "Current authorization denied"}, "409": map[string]any{"description": "Outcome changed; explicitly reload search while preserving selected IDs"}}}}

	paths["/api/v1/namespaces/{namespace_id}/effective-permissions"] = map[string]any{"get": map[string]any{"operationId": "effectivePermissions", "description": "Current authenticated permission hints for presentation, optionally scoped to outcome_id. Does not grant execution authority. Credentials and grants are rechecked.", "parameters": []any{map[string]any{"name": "namespace_id", "in": "path", "required": true, "schema": map[string]any{"type": "string", "format": "uuid"}}, map[string]any{"name": "outcome_id", "in": "query", "schema": map[string]any{"type": "string", "format": "uuid"}}}, "responses": map[string]any{"200": map[string]any{"description": "Bounded permission names; execution_authority=false"}, "403": map[string]any{"description": "Current authorization denied"}}}}
	schemas["SigningSecurityIntentV2"] = commands.Schema(reflect.TypeFor[application.SigningSecurityIntent]())
	schemas["SignedStateQueryV2"] = commands.SignedSchema(reflect.TypeFor[application.SignedStateQuery]())
	schemas["SignedStateResultV2"] = commands.SignedSchema(reflect.TypeFor[application.SignedStateResult]())

	header := map[string]any{"name": "Idempotency-Key", "in": "header", "required": true, "schema": map[string]any{"type": "string", "minLength": 16, "maxLength": 128}}
	for _, d := range commands.NewCatalog(nil, nil).Descriptors() {
		name := "Catalog_" + d.Name
		schemas[name] = d.Schema
		paths["/api/v1/commands/"+d.Name] = map[string]any{"post": map[string]any{"operationId": "command_" + d.Name, "description": "Application command. Retry an uncertain result with identical intent/key; reconcile stale versions before a new intent. Payload cap 256 KiB.", "parameters": []any{header}, "requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"type": "object", "required": []string{"command"}, "additionalProperties": false, "properties": map[string]any{"command": map[string]any{"$ref": "#/components/schemas/" + name}}}}}}, "responses": map[string]any{"200": map[string]any{"description": "Persisted result or commit receipt (result_omitted=true); includes command_id, outcome_revision and idempotent_replay."}, "400": map[string]any{"description": "Invalid intent or key."}, "401": map[string]any{"description": "Credential absent, revoked or expired."}, "403": map[string]any{"description": "Namespace or permission denied."}, "409": map[string]any{"description": "Version, transaction, idempotency or lease conflict; never silently change intent/version."}, "413": map[string]any{"description": "Payload exceeds byte limit."}}}}
	}
	paths["/api/v1/security/signing-commands"] = map[string]any{"post": map[string]any{"description": "Transactional signing identity administration; one-use enrollment and possession proof are mandatory for registration. Versions are quoted decimal strings. No private key is transmitted.", "parameters": []any{header}, "requestBody": map[string]any{"required": true, "content": map[string]any{"application/json": map[string]any{"schema": map[string]any{"$ref": "#/components/schemas/SigningSecurityIntentV2"}}}}, "responses": map[string]any{"200": map[string]any{"description": "Persisted signing identity result or exact idempotent replay."}, "403": map[string]any{"description": "Current credential, policy or proof denied."}, "409": map[string]any{"description": "CAS or intent mismatch."}}}}
	for _, path := range []string{"/api/v1/security/signing-identity", "/api/v1/namespaces/{namespace_id}/signed-trust", "/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/signed-state/{resource}", "/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/signed-state/{resource}/{entity_id}", "/api/v1/capabilities", "/api/v1/namespaces/{namespace_id}/commands/{command_id}", "/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/work-contracts", "/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/work-contracts/{contract_id}", "/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/work-contracts/{contract_id}/spec", "/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/work-contracts/{contract_id}/checkpoints", "/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/work-contracts/{contract_id}/submissions", "/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/work-submissions/{submission_id}", "/api/v1/namespaces/{namespace_id}/outcomes/{outcome_id}/available-work"} {
		parameters := []any{}
		for _, segment := range strings.Split(path, "/") {
			if strings.HasPrefix(segment, "{") {
				parameters = append(parameters, map[string]any{"name": strings.Trim(segment, "{}"), "in": "path", "required": true, "schema": map[string]any{"type": "string"}})
			}
		}
		paths[path] = map[string]any{"get": map[string]any{"description": "Authorized execution-contract read. Collections accept limit (1..100) and scope-bound cursor. Specs are immutable resources. Effective expiry is read-only; receipt lookup is restricted to authenticated principal.", "parameters": parameters, "responses": map[string]any{"200": map[string]any{"description": "Typed result, page or retained receipt; omission metadata remains explicit."}, "403": map[string]any{"description": "Authorization denied."}, "404": map[string]any{"description": "Not found in scope or receipt not retained."}}}}
	}
	document := map[string]any{"openapi": "3.1.0", "info": map[string]any{"title": "WOS generated command catalog", "version": strings.TrimSpace(string(version))}, "paths": paths, "security": []any{map[string]any{"bearerAuth": []string{}}, map[string]any{"browserSession": []string{}}}, "components": map[string]any{"schemas": schemas, "securitySchemes": map[string]any{"bearerAuth": map[string]any{"type": "http", "scheme": "bearer"}, "browserSession": map[string]any{"type": "apiKey", "in": "cookie", "name": "wos_session"}}}}
	b, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		panic(err)
	}
	b = append(b, '\n')
	if err = os.WriteFile("packages/wos-api/commands.openapi.json", b, 0644); err != nil {
		panic(err)
	}
}
