package commands

import (
	"encoding/json"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"reflect"
	"strings"
	"time"
)

// Encode uses public command names and seconds, preserving the Core DTO fingerprint representation.
func Encode(value any) ([]byte, error) {
	v, err := encodeValue(reflect.ValueOf(value))
	if err != nil {
		return nil, err
	}
	return json.Marshal(v)
}
func encodeValue(v reflect.Value) (any, error) {
	if !v.IsValid() {
		return nil, nil
	}
	if v.Kind() == reflect.Interface || v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return nil, nil
		}
		return encodeValue(v.Elem())
	}
	if v.Type() == reflect.TypeFor[time.Time]() || v.Type() == reflect.TypeFor[domain.EntityRef]() {
		return v.Interface(), nil
	}
	if v.Type() == reflect.TypeFor[time.Duration]() {
		d := time.Duration(v.Int())
		if d%time.Second != 0 {
			return nil, fmt.Errorf("TTL must use whole seconds")
		}
		return int64(d / time.Second), nil
	}
	switch v.Kind() {
	case reflect.Struct:
		result := map[string]any{}
		for i := 0; i < v.NumField(); i++ {
			f := v.Type().Field(i)
			if !f.IsExported() || fieldName(f) == "-" {
				continue
			}
			fv := v.Field(i)
			if (strings.Contains(f.Tag.Get("json"), "omitempty") || fv.Kind() == reflect.Pointer || fv.Kind() == reflect.Slice) && fv.IsZero() {
				continue
			}
			encoded, err := encodeValue(fv)
			if err != nil {
				return nil, err
			}
			if strings.Contains(f.Tag.Get("json"), ",string") {
				encoded = fmt.Sprint(encoded)
			}
			result[fieldName(f)] = encoded
		}
		return result, nil
	case reflect.Slice, reflect.Array:
		result := make([]any, v.Len())
		for i := range result {
			item, err := encodeValue(v.Index(i))
			if err != nil {
				return nil, err
			}
			result[i] = item
		}
		return result, nil
	default:
		return v.Interface(), nil
	}
}
