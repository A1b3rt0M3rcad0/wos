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
}
