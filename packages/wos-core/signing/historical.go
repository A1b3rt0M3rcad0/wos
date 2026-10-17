package signing

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"io"
	"reflect"
	"sort"
	"strings"
)

const legacyProtocolVersion = 2

// These purposes identify immutable, previously issued protocol-2 documents.
// Sign and SignCanonical deliberately cannot issue new documents with them.
const (
	LegacyContractSpec            PayloadType = "application/vnd.wos.contract-spec.v2+json"
	LegacyContractAuthority       PayloadType = "application/vnd.wos.contract-authority.v2+json"
	LegacyWorkReturn              PayloadType = "application/vnd.wos.work-return.v2+json"
	LegacyReviewReturn            PayloadType = "application/vnd.wos.review-return.v2+json"
	LegacyAcceptanceReceipt       PayloadType = "application/vnd.wos.acceptance-receipt.v2+json"
	LegacyReturnResolution        PayloadType = "application/vnd.wos.return-resolution.v2+json"
	LegacyReturnResolutionReceipt PayloadType = "application/vnd.wos.return-resolution-receipt.v2+json"
	LegacyKeyEnrollment           PayloadType = "application/vnd.wos.key-enrollment.v2+json"
)

func legacyPurpose(current PayloadType) PayloadType {
	switch current {
	case ContractSpec:
		return LegacyContractSpec
	case ContractAuthority:
		return LegacyContractAuthority
	case WorkReturn:
		return LegacyWorkReturn
	case ReviewReturn:
		return LegacyReviewReturn
	case AcceptanceReceipt:
		return LegacyAcceptanceReceipt
	case ReturnResolution:
		return LegacyReturnResolution
	case ReturnResolutionReceipt:
		return LegacyReturnResolutionReceipt
	case KeyEnrollment:
		return LegacyKeyEnrollment
	}
	return ""
}

func readablePurpose(purpose PayloadType) bool {
	if purpose.Valid() {
		return true
	}
	switch purpose {
	case LegacyContractSpec, LegacyContractAuthority, LegacyWorkReturn, LegacyReviewReturn,
		LegacyAcceptanceReceipt, LegacyReturnResolution, LegacyReturnResolutionReceipt, LegacyKeyEnrollment:
		return true
	}
	return false
}

// VerifyHistorical verifies existing records only. The caller supplies the
// current purpose; an exact protocol-2 purpose of that same kind is also accepted.
// It never upgrades a payload, changes the authenticated bytes, or authorizes a
// legacy mutation. Mutation entry points must continue to use Verify.
func VerifyHistorical(env Envelope, purpose PayloadType, keyID string, public ed25519.PublicKey) ([]byte, error) {
	if !purpose.Valid() {
		return nil, invalid("signature_payload_mismatch", "expected current payload purpose")
	}
	if env.PayloadType == purpose {
		return Verify(env, purpose, keyID, public)
	}
	if env.PayloadType != legacyPurpose(purpose) {
		return nil, invalid("signature_payload_mismatch", "wrong historical payload purpose")
	}
	return verifyVersion(env, env.PayloadType, keyID, public, legacyProtocolVersion)
}

// DecodeHistorical returns a typed read projection and the exact verified bytes.
// The projection retains its original ProtocolVersion. It must not be serialized
// over the stored record, signed again, or used to resume a protocol-2 mutation.
func DecodeHistorical[T any](env Envelope, purpose PayloadType, keyID string, public ed25519.PublicKey) (T, []byte, error) {
	var result T
	raw, err := VerifyHistorical(env, purpose, keyID, public)
	if err != nil {
		return result, nil, err
	}
	if err = DecodeHistoricalPayload(raw, &result, MaxPayloadBytes); err != nil {
		return result, nil, err
	}
	return result, raw, nil
}

// ReadHistoricalBinding extracts an already verified record's common binding.
// Additional fields are intentionally left for the purpose-specific strict DTO.
// This function performs no signature verification or authorization.
func ReadHistoricalBinding(raw []byte) (Binding, error) {
	var binding Binding
	if err := validateJSON(raw, MaxPayloadBytes); err != nil {
		return binding, err
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(raw, &fields) != nil || fields == nil || json.Unmarshal(fields["protocol_version"], &binding.ProtocolVersion) != nil {
		return binding, invalid("invalid_payload", "signed protocol version is required")
	}
	for name := range fields {
		for _, exact := range []string{"protocol_version", "server_id", "namespace_id", "tenant_id", "outcome_id", "principal_id", "signer_key_id"} {
			if name != exact && strings.EqualFold(name, exact) {
				return Binding{}, invalid("invalid_payload", "signed binding field does not match exact schema")
			}
		}
	}
	switch binding.ProtocolVersion {
	case ProtocolVersion:
		if _, found := fields["namespace_id"]; found {
			return Binding{}, invalid("invalid_payload", "legacy scope field in current payload")
		}
		if err := json.Unmarshal(raw, &binding); err != nil {
			return Binding{}, invalid("invalid_payload", "invalid signed binding")
		}
	case legacyProtocolVersion:
		if _, found := fields["tenant_id"]; found {
			return Binding{}, invalid("invalid_payload", "current scope field in historical payload")
		}
		var legacy struct {
			ProtocolVersion int    `json:"protocol_version"`
			ServerID        string `json:"server_id"`
			NamespaceID     string `json:"namespace_id"`
			OutcomeID       string `json:"outcome_id"`
			PrincipalID     string `json:"principal_id"`
			SignerKeyID     string `json:"signer_key_id"`
		}
		if err := json.Unmarshal(raw, &legacy); err != nil {
			return Binding{}, invalid("invalid_payload", "invalid historical signed binding")
		}
		binding = Binding{legacy.ProtocolVersion, legacy.ServerID, legacy.NamespaceID, legacy.OutcomeID, legacy.PrincipalID, legacy.SignerKeyID}
	default:
		return Binding{}, invalid("invalid_payload", "unsupported signed protocol version")
	}
	return binding, nil
}

// DecodeHistoricalPayload decodes an already verified or trusted stored payload.
// Only protocol-2 DTO fields receive a legacy-name projection. Maps, arbitrary
// metadata, strings and opaque RawMessage values retain their original content.
// The source bytes and protocol version are never modified.
func DecodeHistoricalPayload(raw []byte, destination any, maxBytes int) error {
	if err := validateJSON(raw, maxBytes); err != nil {
		return err
	}
	var version struct {
		ProtocolVersion int `json:"protocol_version"`
	}
	if json.Unmarshal(raw, &version) != nil {
		return invalid("invalid_payload", "invalid historical payload")
	}
	switch version.ProtocolVersion {
	case ProtocolVersion:
		return DecodeStrict(raw, destination, maxBytes)
	case legacyProtocolVersion:
		projected, err := projectStoredFields(raw, reflect.TypeOf(destination), true)
		if err != nil {
			return err
		}
		return DecodeStrict(projected, destination, maxBytes)
	default:
		return invalid("invalid_payload", "unsupported signed protocol version")
	}
}

// DecodeStoredJSON reads trusted persistent application JSON written before or
// after the Tenant terminology cutover. It is never an incoming-command decoder.
// Historical proof bytes must still use VerifyHistorical; this helper supplies
// no proof, authorization, idempotency reinterpretation, or permission to rewrite
// immutable records. DTO fields can use either historical or current spelling,
// but both spellings together are rejected even when their values agree.
func DecodeStoredJSON(raw []byte, destination any, maxBytes int) error {
	if maxBytes > 0 && len(raw) > maxBytes {
		return invalid("payload_limit", "stored JSON exceeds byte limit")
	}
	if err := validateStoredJSONKeys(raw); err != nil {
		return err
	}
	projected, err := projectStoredFields(raw, reflect.TypeOf(destination), false)
	if err != nil {
		return err
	}
	// Persistent aggregates are not signed-payload envelopes. Preserve the prior
	// JSON decoder's capacity and tolerance for fields added by stored versions;
	// protocol payload depth/node/scalar limits must not truncate large histories.
	return json.Unmarshal(projected, destination)
}

func validateStoredJSONKeys(raw []byte) error {
	// Retain encoding/json's own syntax and nesting constraints before walking
	// duplicate fields, without importing the tighter signed-envelope limits.
	if !json.Valid(raw) {
		return invalid("invalid_payload", "invalid stored JSON")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var visit func() error
	visit = func() error {
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		opening, composite := token.(json.Delim)
		if !composite {
			return nil
		}
		if opening != '{' && opening != '[' {
			return invalid("invalid_payload", "invalid stored JSON delimiter")
		}
		seen := map[string]bool{}
		for decoder.More() {
			if opening == '{' {
				key, err := decoder.Token()
				if err != nil {
					return err
				}
				name, ok := key.(string)
				if !ok || seen[name] {
					return invalid("invalid_payload", "duplicate stored JSON field")
				}
				seen[name] = true
			}
			if err := visit(); err != nil {
				return err
			}
		}
		_, err = decoder.Token()
		return err
	}
	if err := visit(); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return invalid("invalid_payload", "trailing stored JSON value")
	}
	return nil
}

func legacyFieldName(name string) string {
	name = strings.ReplaceAll(name, "tenant", "namespace")
	name = strings.ReplaceAll(name, "Tenant", "Namespace")
	return strings.ReplaceAll(name, "TENANT", "NAMESPACE")
}

func projectStoredFields(raw []byte, target reflect.Type, legacyOnly bool) ([]byte, error) {
	if target == nil {
		return nil, invalid("invalid_payload", "typed destination required")
	}
	for target.Kind() == reflect.Pointer {
		target = target.Elem()
	}
	if string(raw) == "null" || target == reflect.TypeFor[json.RawMessage]() {
		return raw, nil
	}
	// Scalar custom codecs (time, decimal counters) own their representation.
	// Object DTO codecs can still contain scope fields: WorkItem's codec, for
	// example, only handles an exact fencing counter and delegates other fields.
	if reflect.PointerTo(target).Implements(reflect.TypeFor[json.Unmarshaler]()) && (target.Kind() != reflect.Struct || len(bytes.TrimSpace(raw)) == 0 || bytes.TrimSpace(raw)[0] != '{') {
		return raw, nil
	}
	switch target.Kind() {
	case reflect.Struct:
		var fields map[string]json.RawMessage
		if json.Unmarshal(raw, &fields) != nil || fields == nil {
			return nil, invalid("invalid_payload", "object required by stored schema")
		}
		types := map[string]reflect.Type{}
		exactStructFields(target, types)
		for name, typ := range types {
			oldName := legacyFieldName(name)
			current, hasCurrent := fields[name]
			if oldName != name {
				old, hasOld := fields[oldName]
				if hasCurrent && (hasOld || legacyOnly) {
					return nil, invalid("invalid_payload", "scope field spelling differs from stored schema")
				}
				if hasOld {
					current, hasCurrent = old, true
					delete(fields, oldName)
				}
			}
			if !hasCurrent {
				continue
			}
			value, err := projectStoredFields(current, typ, legacyOnly)
			if err != nil {
				return nil, err
			}
			fields[name] = value
		}
		return marshalRawObject(fields), nil
	case reflect.Slice, reflect.Array:
		if target.Elem().Kind() == reflect.Uint8 {
			return raw, nil
		}
		var values []json.RawMessage
		if json.Unmarshal(raw, &values) != nil {
			return nil, invalid("invalid_payload", "array required by stored schema")
		}
		for i, value := range values {
			projected, err := projectStoredFields(value, target.Elem(), legacyOnly)
			if err != nil {
				return nil, err
			}
			values[i] = projected
		}
		var projected bytes.Buffer
		projected.WriteByte('[')
		for i, value := range values {
			if i > 0 {
				projected.WriteByte(',')
			}
			projected.Write(value)
		}
		projected.WriteByte(']')
		return projected.Bytes(), nil
	case reflect.Map:
		var values map[string]json.RawMessage
		if json.Unmarshal(raw, &values) != nil {
			return nil, invalid("invalid_payload", "mapping required by stored schema")
		}
		for name, value := range values {
			projected, err := projectStoredFields(value, target.Elem(), legacyOnly)
			if err != nil {
				return nil, err
			}
			values[name] = projected
		}
		return marshalRawObject(values), nil
	default:
		return raw, nil
	}
}

// Raw values are copied verbatim so an opaque signed document embedded inside
// an application record does not acquire whitespace or escaping changes.
func marshalRawObject(fields map[string]json.RawMessage) []byte {
	keys := make([]string, 0, len(fields))
	for name := range fields {
		keys = append(keys, name)
	}
	sort.Strings(keys)
	var result bytes.Buffer
	result.WriteByte('{')
	for i, name := range keys {
		if i > 0 {
			result.WriteByte(',')
		}
		var quoted bytes.Buffer
		encoder := json.NewEncoder(&quoted)
		encoder.SetEscapeHTML(false)
		_ = encoder.Encode(name)
		result.Write(bytes.TrimSuffix(quoted.Bytes(), []byte{'\n'}))
		result.WriteByte(':')
		result.Write(fields[name])
	}
	result.WriteByte('}')
	return result.Bytes()
}
