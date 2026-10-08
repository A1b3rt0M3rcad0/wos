package signing

import (
	"encoding/json"
	"reflect"
	"strings"
)

// encoding/json accepts case-insensitive aliases for struct fields. Signed
// schemas require their exact spelling, including inside arrays and embeddings.
func exactJSONFields(raw []byte, target reflect.Type) error {
	if target == nil {
		return invalid("invalid_payload", "typed destination required")
	}
	for target.Kind() == reflect.Pointer {
		target = target.Elem()
	}
	if string(raw) == "null" {
		return nil
	}
	switch target.Kind() {
	case reflect.Struct:
		// Custom codecs own their non-object schema (for example time.Time).
		if reflect.PointerTo(target).Implements(reflect.TypeFor[json.Unmarshaler]()) {
			return nil
		}
		var object map[string]json.RawMessage
		if json.Unmarshal(raw, &object) != nil {
			return invalid("invalid_payload", "object required by schema")
		}
		fields := map[string]reflect.Type{}
		exactStructFields(target, fields)
		for name, value := range object {
			field, known := fields[name]
			if !known {
				return invalid("invalid_payload", "field does not match exact schema")
			}
			if err := exactJSONFields(value, field); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		// RawMessage is intentionally opaque; its purpose-specific schema is
		// validated only after proof verification and before executing its DTO.
		if target == reflect.TypeFor[json.RawMessage]() || target.Elem().Kind() == reflect.Uint8 {
			return nil
		}
		var array []json.RawMessage
		if json.Unmarshal(raw, &array) != nil {
			return invalid("invalid_payload", "array required by schema")
		}
		for _, value := range array {
			if err := exactJSONFields(value, target.Elem()); err != nil {
				return err
			}
		}
	case reflect.Map:
		var object map[string]json.RawMessage
		if json.Unmarshal(raw, &object) != nil {
			return invalid("invalid_payload", "mapping required by schema")
		}
		for _, value := range object {
			if err := exactJSONFields(value, target.Elem()); err != nil {
				return err
			}
		}
	}
	return nil
}
func exactStructFields(target reflect.Type, fields map[string]reflect.Type) {
	for i := 0; i < target.NumField(); i++ {
		field := target.Field(i)
		if !field.IsExported() {
			continue
		}
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "-" {
			continue
		}
		nested := field.Type
		for nested.Kind() == reflect.Pointer {
			nested = nested.Elem()
		}
		if field.Anonymous && name == "" && nested.Kind() == reflect.Struct {
			exactStructFields(nested, fields)
			continue
		}
		if name == "" {
			name = field.Name
		}
		fields[name] = field.Type
	}
}
