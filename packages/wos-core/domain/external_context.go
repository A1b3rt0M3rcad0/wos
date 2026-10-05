package domain

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"time"
)

// ExternalContext is a consumer-owned flat address. It never grants access.
type ExternalContext map[string]any

func (c ExternalContext) Validate() error {
	if len(c) > 64 {
		return NewError(ErrorCodeInvalidArgument, "external context exceeds 64 keys")
	}
	for key, value := range c {
		if strings.TrimSpace(key) == "" || len(key) > 256 || strings.HasPrefix(key, "wos.") || strings.ContainsRune(key, 0) {
			return NewError(ErrorCodeInvalidArgument, "invalid or reserved external context key")
		}
		if _, err := CanonicalContextValue(value); err != nil {
			return err
		}
	}
	b, err := json.Marshal(c)
	if err != nil || len(b) > 8192 {
		return NewError(ErrorCodeInvalidArgument, "external context exceeds 8 KiB or contains invalid values")
	}
	return nil
}

// CanonicalContextValue preserves scalar type, normalizing equivalent JSON numbers.
func CanonicalContextValue(value any) (string, error) {
	if value == nil {
		return "null", nil
	}
	switch reflect.ValueOf(value).Kind() {
	case reflect.String, reflect.Bool, reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Float32, reflect.Float64:
	default:
		return "", NewError(ErrorCodeInvalidArgument, "external context requires scalar values")
	}
	b, err := json.Marshal(value)
	if err != nil {
		return "", NewError(ErrorCodeInvalidArgument, "external context requires scalar values")
	}
	var normalized any
	if json.Unmarshal(b, &normalized) != nil {
		return "", NewError(ErrorCodeInvalidArgument, "invalid context value")
	}
	switch v := normalized.(type) {
	case string, bool:
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) || math.Abs(v) > 9007199254740991 {
			return "", NewError(ErrorCodeInvalidArgument, "context number outside interoperable range")
		}
	default:
		return "", NewError(ErrorCodeInvalidArgument, "external context requires string, boolean, number or null")
	}
	b, err = json.Marshal(normalized)
	return string(b), err
}
func (c ExternalContext) Clone() ExternalContext {
	if c == nil {
		return nil
	}
	copy := ExternalContext{}
	for k, v := range c {
		copy[k] = v
	}
	return copy
}
func (o *Outcome) ReplaceExternalContext(value ExternalContext, now time.Time) error {
	if err := value.Validate(); err != nil {
		return err
	}
	if err := o.RecordExternalReferenceChange(now); err != nil {
		return err
	}
	o.ExternalContext = value.Clone()
	return nil
}
