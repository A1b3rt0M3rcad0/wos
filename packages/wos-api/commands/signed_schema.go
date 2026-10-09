package commands

import (
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"reflect"
	"strings"
	"time"
)

// SignedSchema describes exact JSON DTO spelling. Unlike command schemas these
// bytes are never normalized or renamed before their signature is verified.
func SignedSchema(t reflect.Type) map[string]any {
	if t.Kind() == reflect.Pointer {
		return map[string]any{"anyOf": []any{SignedSchema(t.Elem()), map[string]any{"type": "null"}}}
	}
	if t == reflect.TypeFor[signing.Decimal]() || t == reflect.TypeFor[d.FencingToken]() {
		return map[string]any{"type": "string", "pattern": "^(0|[1-9][0-9]{0,19})$"}
	}
	if t == reflect.TypeFor[time.Time]() {
		return map[string]any{"type": "string", "format": "date-time"}
	}
	if t == reflect.TypeFor[json.RawMessage]() {
		return map[string]any{"type": "object"}
	}
	switch t.Kind() {
	case reflect.Struct:
		fields := map[string]any{}
		required := []string{}
		for index := 0; index < t.NumField(); index++ {
			f := t.Field(index)
			if !f.IsExported() {
				continue
			}
			name := strings.Split(f.Tag.Get("json"), ",")[0]
			if name == "-" {
				continue
			}
			if f.Anonymous && name == "" {
				nested := SignedSchema(f.Type)
				for key, value := range nested["properties"].(map[string]any) {
					fields[key] = value
				}
				required = append(required, nested["required"].([]string)...)
				continue
			}
			if name == "" {
				name = f.Name
			}
			field := SignedSchema(f.Type)
			if strings.Contains(f.Tag.Get("json"), ",string") {
				field = map[string]any{"type": "string", "pattern": "^(0|[1-9][0-9]{0,19})$"}
			}
			if t == reflect.TypeFor[signing.Binding]() && name == "protocol_version" {
				field["const"] = 2
			}
			if t == reflect.TypeFor[signing.Envelope]() {
				if name == "payload" {
					field["maxLength"] = 245760
				}
				if name == "signatures" {
					field["minItems"], field["maxItems"] = 1, 1
				}
			}
			if t.Name() == "SignedStateResult" && t.PkgPath() == "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application" {
				if name == "operation_result_payload" {
					field = map[string]any{"type": "string", "contentEncoding": "base64", "maxLength": 1398104}
				}
				if name == "material_payload" {
					field = map[string]any{"type": "string", "contentEncoding": "base64", "maxLength": 245760}
				}
			}
			if f.Type.Kind() == reflect.Slice && (name == "artifacts" || name == "evidence" || name == "evidence_links" || name == "findings" || name == "correction_responses" || name == "execution_principals" || name == "execution_groups") {
				field["maxItems"] = 100
			}
			if t == reflect.TypeFor[d.ExecutionSpec]() {
				field["maxItems"] = 100
			}
			fields[name] = field
			if !strings.Contains(f.Tag.Get("json"), "omitempty") {
				required = append(required, name)
			}
		}
		return map[string]any{"type": "object", "properties": fields, "required": required, "additionalProperties": false}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": SignedSchema(t.Elem()), "maxItems": signing.MaxNodes}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": SignedSchema(t.Elem()), "maxProperties": signing.MaxNodes}
	case reflect.String:
		schema := map[string]any{"type": "string", "maxLength": signing.MaxScalarBytes}
		if t == reflect.TypeFor[d.ID]() {
			schema["pattern"] = "^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"
		}
		if t == reflect.TypeFor[signing.PayloadType]() {
			schema["enum"] = []string{string(signing.ContractSpec), string(signing.ContractAuthority), string(signing.WorkReturn), string(signing.ReviewReturn), string(signing.AcceptanceReceipt), string(signing.KeyEnrollment)}
		}
		return schema
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64, reflect.Uint32:
		return map[string]any{"type": "integer"}
	case reflect.Float64:
		return map[string]any{"type": "number"}
	default:
		return map[string]any{}
	}
}
