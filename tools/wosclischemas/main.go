package main

import (
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	cli "github.com/A1b3rt0M3rcad0/wos/packages/wos-cli"
	"os"
	"reflect"
)

func main() {
	documents := map[string]reflect.Type{"config": reflect.TypeFor[cli.Config](), "contract": reflect.TypeFor[cli.ContractDocument](), "checkpoint": reflect.TypeFor[cli.CheckpointDraft](), "result": reflect.TypeFor[cli.ResultDraft]()}
	for name, t := range documents {
		schema := commands.Schema(t)
		schema["$schema"] = "https://json-schema.org/draft/2020-12/schema"
		schema["$id"] = "https://wos.dev/schemas/workspace/v1/" + name
		schema["description"] = "WOS restricted YAML 1.2 mapping, schema 1; max 256 KiB, depth 16. Server remains authoritative."
		properties := schema["properties"].(map[string]any)
		properties["schema_version"] = map[string]any{"type": "integer", "const": 1}
		data, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			panic(err)
		}
		if err = os.WriteFile("packages/wos-cli/schemas/"+name+".schema.json", append(data, '\n'), 0644); err != nil {
			panic(err)
		}
	}
	for name, t := range map[string]reflect.Type{"project-v2": reflect.TypeFor[cli.ProjectV2](), "profile-v2": reflect.TypeFor[cli.ProfileV2](), "contract-v2": reflect.TypeFor[cli.ContractFileV2]()} {
		schema := commands.SignedSchema(t)
		schema["$schema"] = "https://json-schema.org/draft/2020-12/schema"
		schema["$id"] = "https://wos.dev/schemas/workspace/v2/" + name
		schema["description"] = "Restricted YAML mapping; max 1 MiB, depth 16, 10000 nodes. Semantic scalars max 64 KiB; explicitly identified technical base64 fields have separate byte bounds. Issued documents must be cryptographically verified. Profiles do not grant authority."
		properties := schema["properties"].(map[string]any)
		properties["schema_version"] = map[string]any{"type": "integer", "const": 2}
		switch name {
		case "profile-v2":
			local := properties["_local"].(map[string]any)["properties"].(map[string]any)
			pending := local["pending_operations"].(map[string]any)
			pending["maxItems"] = 10
			fields := pending["items"].(map[string]any)["properties"].(map[string]any)
			fields["request_fingerprint"] = map[string]any{"type": "string", "pattern": "^[a-f0-9]{64}$", "minLength": 64, "maxLength": 64}
			for _, key := range []string{"payload", "response"} {
				fields[key] = map[string]any{"type": "string", "contentEncoding": "base64", "maxLength": 349528}
			}
		case "contract-v2":
			// Editable drafts use human command names; signed payloads use exact
			// JSON spelling and must never be normalized through command schemas.
			for field, typ := range map[string]reflect.Type{"execution": reflect.TypeFor[cli.ExecutionDraftV2](), "review": reflect.TypeFor[cli.ReviewDraftV2]()} {
				properties[field] = map[string]any{"anyOf": []any{commands.Schema(typ), map[string]any{"type": "null"}}}
			}
			schema["oneOf"] = []any{map[string]any{"required": []string{"execution"}, "not": map[string]any{"required": []string{"review"}}}, map[string]any{"required": []string{"review"}, "not": map[string]any{"required": []string{"execution"}}}}
			local := properties["_local"].(map[string]any)["properties"].(map[string]any)
			pending := local["pending"].(map[string]any)["anyOf"].([]any)[0].(map[string]any)["properties"].(map[string]any)
			pending["response"] = map[string]any{"type": "string", "contentEncoding": "base64", "maxLength": 349528}
		}
		data, err := json.MarshalIndent(schema, "", "  ")
		if err != nil {
			panic(err)
		}
		if err = os.WriteFile("packages/wos-cli/schemas/"+name+".schema.json", append(data, '\n'), 0644); err != nil {
			panic(err)
		}
	}
}
