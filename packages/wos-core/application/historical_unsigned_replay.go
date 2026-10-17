package application

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"strings"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// A terminology cutover cannot invalidate an already committed unsigned receipt.
// Consult the old fingerprint only after a current-fingerprint conflict, and
// accept only a completed replay. A missing/expired old receipt never authorizes
// execution using the historical spelling. Signed intentions have no alias.
func reserveCommandIntent(ctx context.Context, store ports.IdempotencyStore, identity domain.IdempotencyIdentity, meta commandMetadata, actor domain.ActorRef, command any) (domain.IdempotencyReservation, error) {
	reservation, err := store.Reserve(ctx, identity, meta.Fingerprint)
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeIdempotencyConflict || signedCommand(meta.Name) || meta.Name == "SetTenantWorkProtocol" {
		return reservation, err
	}
	oldPayload, changed, compatibilityError := historicalUnsignedPayload(meta.Payload, reflect.TypeOf(command))
	if compatibilityError != nil {
		return domain.IdempotencyReservation{}, compatibilityError
	}
	if !changed {
		return reservation, err
	}
	oldFingerprint, compatibilityError := fingerprintCommand(meta.Name, actor, oldPayload)
	if compatibilityError != nil {
		return domain.IdempotencyReservation{}, compatibilityError
	}
	previous, replayError := store.Reserve(ctx, identity, oldFingerprint)
	if replayError == nil && previous.IsReplay() {
		return previous, nil
	}
	if replayError != nil {
		return domain.IdempotencyReservation{}, replayError
	}
	// The caller rolls back this transaction. In particular, never commit a
	// reservation newly created after a receipt expired between the two reads.
	return domain.IdempotencyReservation{}, err
}

func fingerprintCommand(name string, actor domain.ActorRef, payload json.RawMessage) (string, error) {
	encoded, err := json.Marshal(struct {
		CommandName string          `json:"command_name"`
		Actor       domain.ActorRef `json:"actor_ref"`
		Payload     json.RawMessage `json:"payload"`
	}{name, actor, payload})
	if err != nil {
		return "", domain.WrapError(domain.ErrorCodeInvalidArgument, "command fingerprint cannot be encoded", err)
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), nil
}

// Preserve encoding/json's original struct field ordering exactly. Only keys
// owned by typed DTO fields are projected; maps, metadata, RawMessage and user
// strings remain opaque. Re-encoding into a map would change historic hashes.
func historicalUnsignedPayload(raw []byte, typ reflect.Type) ([]byte, bool, error) {
	type replacement struct {
		start, end int
		key        string
	}
	replacements := []replacement{}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var visit func(reflect.Type) error
	visit = func(typ reflect.Type) error {
		for typ != nil && typ.Kind() == reflect.Pointer {
			typ = typ.Elem()
		}
		if typ == reflect.TypeFor[json.RawMessage]() {
			typ = nil
		}
		token, err := decoder.Token()
		if err != nil {
			return err
		}
		switch token {
		case json.Delim('{'):
			for decoder.More() {
				start := int(decoder.InputOffset())
				token, err := decoder.Token()
				if err != nil {
					return err
				}
				key := token.(string)
				end := int(decoder.InputOffset())
				field, found := historicalJSONField(typ, key)
				var child reflect.Type
				if found {
					child = field.Type
					if field.Name == "TenantID" {
						oldKey := map[string]string{"tenant_id": "namespace_id", "TenantID": "NamespaceID"}[key]
						if oldKey != "" {
							start += bytes.IndexByte(raw[start:end], '"')
							replacements = append(replacements, replacement{start, end, oldKey})
						}
					}
				}
				if err := visit(child); err != nil {
					return err
				}
			}
			_, err := decoder.Token()
			return err
		case json.Delim('['):
			var child reflect.Type
			if typ != nil && (typ.Kind() == reflect.Slice || typ.Kind() == reflect.Array) {
				child = typ.Elem()
			}
			for decoder.More() {
				if err := visit(child); err != nil {
					return err
				}
			}
			_, err := decoder.Token()
			return err
		}
		return nil
	}
	if err := visit(typ); err != nil {
		return nil, false, domain.WrapError(domain.ErrorCodeInvalidArgument, "historical command fingerprint cannot be projected", err)
	}
	if len(replacements) == 0 {
		return raw, false, nil
	}
	var output bytes.Buffer
	position := 0
	for _, replacement := range replacements {
		output.Write(raw[position:replacement.start])
		key, _ := json.Marshal(replacement.key)
		output.Write(key)
		position = replacement.end
	}
	output.Write(raw[position:])
	return output.Bytes(), true, nil
}

func historicalJSONField(typ reflect.Type, key string) (reflect.StructField, bool) {
	for typ != nil && typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ == nil || typ.Kind() != reflect.Struct {
		return reflect.StructField{}, false
	}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.PkgPath != "" {
			continue
		}
		name := strings.Split(field.Tag.Get("json"), ",")[0]
		if name == "-" {
			continue
		}
		if name == "" && field.Anonymous {
			continue
		}
		if name == "" {
			name = field.Name
		}
		if name == key {
			return field, true
		}
	}
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Anonymous && field.PkgPath == "" && strings.Split(field.Tag.Get("json"), ",")[0] == "" {
			if inherited, found := historicalJSONField(field.Type, key); found {
				return inherited, true
			}
		}
	}
	return reflect.StructField{}, false
}
