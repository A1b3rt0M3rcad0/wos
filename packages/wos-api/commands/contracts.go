package commands

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"io"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
)

func Schema(t reflect.Type) map[string]any                 { return typeSchema(t) }
func Normalize(raw []byte, t reflect.Type) ([]byte, error) { return normalizeInput(raw, t) }
func Decode(raw []byte, v any) error                       { return decodeStrict(raw, v) }
func decodeStrict(raw []byte, target any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return domain.WrapError(domain.ErrorCodeInvalidArgument, "invalid tool arguments", err)
	}
	if err := d.Decode(new(any)); err != io.EOF {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "tool arguments must contain exactly one JSON value")
	}
	return nil
}
func snake(s string) string {
	// Keep plural initialisms together, matching the existing REST JSON contract.
	s = strings.ReplaceAll(s, "IDs", "Ids")
	s = strings.ReplaceAll(s, "URLs", "Urls")
	r := []rune(s)
	var b strings.Builder
	for i, c := range r {
		if unicode.IsUpper(c) && i > 0 && (unicode.IsLower(r[i-1]) || (i+1 < len(r) && unicode.IsLower(r[i+1]))) {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(c))
	}
	return b.String()
}
func fieldName(f reflect.StructField) string {
	tag := strings.Split(f.Tag.Get("json"), ",")[0]
	if tag != "" {
		return tag
	}
	if f.Type == reflect.TypeFor[time.Duration]() {
		return snake(f.Name) + "_seconds"
	}
	return snake(f.Name)
}
func typeSchema(t reflect.Type) map[string]any {
	enumValues := map[reflect.Type][]string{
		reflect.TypeFor[domain.IssueSeverity]():      {string(domain.IssueSeverityCritical), string(domain.IssueSeverityMajor), string(domain.IssueSeverityMinor), string(domain.IssueSeverityInformational)},
		reflect.TypeFor[domain.BlockerPropagation](): {string(domain.BlockerPropagationDirect), string(domain.BlockerPropagationSubtree)},
		reflect.TypeFor[domain.VerificationMode]():   {string(domain.VerificationModeAttestation), string(domain.VerificationModeEvidenceReview), string(domain.VerificationModeExternalEvaluation)},
		reflect.TypeFor[domain.AssessmentResult]():   {string(domain.AssessmentResultMet), string(domain.AssessmentResultNotMet), string(domain.AssessmentResultInconclusive), string(domain.AssessmentResultWaived)},
		reflect.TypeFor[domain.EvidenceType]():       {string(domain.EvidenceTypeMeasurement), string(domain.EvidenceTypeTestResult), string(domain.EvidenceTypeInspection), string(domain.EvidenceTypeAttestation), string(domain.EvidenceTypeSource), string(domain.EvidenceTypeExternalEvaluation)},
	}
	if values, ok := enumValues[t]; ok {
		return map[string]any{"type": "string", "enum": values}
	}
	if t == reflect.TypeFor[domain.RoadmapScopeKind]() {
		return map[string]any{"type": "string", "enum": []string{"outcome", "objective"}}
	}
	if t == reflect.TypeFor[domain.ActorKind]() {
		return map[string]any{"type": "string", "enum": []string{"human", "agent", "service", "automation", "external_system"}}
	}

	if t == reflect.TypeFor[domain.ExternalContext]() {
		return map[string]any{"type": "object", "maxProperties": 64, "additionalProperties": map[string]any{"type": []string{"string", "number", "boolean", "null"}}}
	}
	if t == reflect.TypeFor[domain.EventPredicate]() {
		return predicateSchema(3)
	}
	if t.Kind() == reflect.Pointer {
		return typeSchema(t.Elem())
	}
	if t == reflect.TypeFor[time.Time]() {
		return map[string]any{"type": "string", "format": "date-time"}
	}
	if t == reflect.TypeFor[domain.EntityRef]() {
		return map[string]any{"type": "object", "additionalProperties": false, "required": []string{"namespace_id", "outcome_id", "kind", "id"}, "properties": map[string]any{"namespace_id": typeSchema(reflect.TypeFor[domain.ID]()), "outcome_id": typeSchema(reflect.TypeFor[domain.ID]()), "id": typeSchema(reflect.TypeFor[domain.ID]()), "kind": map[string]any{"type": "string"}}}
	}
	switch t.Kind() {
	case reflect.String:
		if t == reflect.TypeFor[domain.ID]() {
			return map[string]any{"type": "string", "pattern": "^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"}
		}
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64, reflect.Uint32:
		return map[string]any{"type": "integer"}
	case reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": typeSchema(t.Elem())}
	case reflect.Map:
		return map[string]any{"type": "object", "additionalProperties": typeSchema(t.Elem())}
	case reflect.Interface:
		return map[string]any{}
	case reflect.Struct:
		p := map[string]any{}
		required := []string{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() || fieldName(f) == "-" {
				continue
			}
			name := fieldName(f)
			p[name] = typeSchema(f.Type)
			if f.Tag.Get("wos") != "optional" && f.Type.Kind() != reflect.Pointer && !strings.Contains(f.Tag.Get("json"), "omitempty") && f.Name != "Description" && f.Name != "ResultSummary" && f.Type.Kind() != reflect.Slice {
				required = append(required, name)
			}
		}
		return map[string]any{"type": "object", "properties": p, "required": required, "additionalProperties": false}
	default:
		return map[string]any{}
	}
}

func predicateSchema(depth int) map[string]any {
	properties := map[string]any{"field": map[string]any{"type": "string"}, "op": map[string]any{"type": "string"}, "value": map[string]any{}}
	if depth > 1 {
		properties["all"] = map[string]any{"type": "array", "maxItems": 20, "items": predicateSchema(depth - 1)}
		properties["any"] = map[string]any{"type": "array", "maxItems": 20, "items": predicateSchema(depth - 1)}
	}
	return map[string]any{"type": "object", "additionalProperties": false, "properties": properties}
}
func normalizeInput(raw []byte, t reflect.Type) ([]byte, error) {
	var v any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if err := d.Decode(&v); err != nil {
		return nil, err
	}
	value, err := normalizeValue(v, t)
	if err != nil {
		return nil, err
	}
	return json.Marshal(value)
}
func normalizeValue(value any, t reflect.Type) (any, error) {
	if value == nil {
		return nil, nil
	}
	if t.Kind() == reflect.Pointer {
		return normalizeValue(value, t.Elem())
	}
	if t == reflect.TypeFor[time.Time]() || t == reflect.TypeFor[domain.EntityRef]() {
		return value, nil
	}
	if t == reflect.TypeFor[time.Duration]() {
		n, ok := value.(json.Number)
		if !ok {
			return nil, fmt.Errorf("TTL seconds must be an integer")
		}
		seconds, err := strconv.ParseInt(string(n), 10, 64)
		if err != nil || seconds < 1 || seconds > 86400 {
			return nil, fmt.Errorf("TTL seconds must be between 1 and 86400")
		}
		return seconds * int64(time.Second), nil
	}
	switch t.Kind() {
	case reflect.Struct:
		m, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("object required")
		}
		out := map[string]any{}
		known := map[string]bool{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			name := fieldName(f)
			known[name] = true
			if v, ok := m[name]; ok {
				transformed, err := normalizeValue(v, f.Type)
				if err != nil {
					return nil, err
				}
				key := strings.Split(f.Tag.Get("json"), ",")[0]
				if key == "" {
					key = f.Name
				}
				out[key] = transformed
			}
		}
		for k := range m {
			if !known[k] {
				return nil, fmt.Errorf("unknown field %q", k)
			}
		}
		return out, nil
	case reflect.Slice, reflect.Array:
		a, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("array required")
		}
		for i := range a {
			v, err := normalizeValue(a[i], t.Elem())
			if err != nil {
				return nil, err
			}
			a[i] = v
		}
		return a, nil
	default:
		return value, nil
	}
}
