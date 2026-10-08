// Package signing implements the public WOS DSSE 1.0.2 / JCS / Ed25519 codec.
// It owns no persistence, transport, secret store or authorization decisions.
package signing

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"strconv"
	"unicode/utf8"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
)

const (
	MaxPayloadBytes  = 180 * 1024
	MaxEnvelopeBytes = 256 * 1024
	MaxDepth         = 16
	MaxNodes         = 10000
	MaxScalarBytes   = 64 * 1024
	ProtocolVersion  = 2
)

type PayloadType string

const (
	ContractSpec      PayloadType = "application/vnd.wos.contract-spec.v2+json"
	ContractAuthority PayloadType = "application/vnd.wos.contract-authority.v2+json"
	WorkReturn        PayloadType = "application/vnd.wos.work-return.v2+json"
	ReviewReturn      PayloadType = "application/vnd.wos.review-return.v2+json"
	AcceptanceReceipt PayloadType = "application/vnd.wos.acceptance-receipt.v2+json"
	KeyEnrollment     PayloadType = "application/vnd.wos.key-enrollment.v2+json"
)

func (t PayloadType) Valid() bool {
	switch t {
	case ContractSpec, ContractAuthority, WorkReturn, ReviewReturn, AcceptanceReceipt, KeyEnrollment:
		return true
	}
	return false
}

type Error struct {
	Code    string
	Message string
}

func (e *Error) Error() string           { return e.Code + ": " + e.Message }
func invalid(code, message string) error { return &Error{code, message} }

type Signature struct {
	KeyID string `json:"keyid"`
	Sig   string `json:"sig"`
}
type Envelope struct {
	PayloadType PayloadType `json:"payloadType"`
	Payload     string      `json:"payload"`
	Signatures  []Signature `json:"signatures"`
}

// Document is the local WOS representation. Payload is a legible JSON/YAML mapping,
// not the base64 payload field of a literal DSSE envelope.
type Proof struct {
	PayloadType PayloadType `json:"payload_type"`
	KeyID       string      `json:"key_id"`
	Signature   string      `json:"signature"`
}
type Document struct {
	Payload json.RawMessage `json:"payload"`
	Proof   Proof           `json:"proof"`
}

// PAE is the DSSE protocol message; Ed25519 signs these bytes, not a digest.
func PAE(t PayloadType, payload []byte) []byte {
	return append([]byte(fmt.Sprintf("DSSEv1 %d %s %d ", len(t), t, len(payload))), payload...)
}
func Digest(payload []byte) string {
	s := sha256.Sum256(payload)
	return "sha256:" + hex.EncodeToString(s[:])
}
func Fingerprint(public ed25519.PublicKey) (string, error) {
	if len(public) != ed25519.PublicKeySize {
		return "", invalid("invalid_signing_key", "invalid public key length")
	}
	return Digest(public), nil
}

func Canonical(v any) ([]byte, error) {
	raw, err := json.Marshal(v)
	if err != nil {
		return nil, invalid("invalid_payload", "cannot serialize payload")
	}
	return CanonicalJSON(raw)
}
func CanonicalJSON(raw []byte) ([]byte, error) {
	if err := validateJSON(raw, MaxPayloadBytes); err != nil {
		return nil, err
	}
	out, err := jsoncanonicalizer.Transform(raw)
	if err != nil {
		return nil, invalid("invalid_payload", "payload is not interoperable JCS JSON")
	}
	if len(out) > MaxPayloadBytes {
		return nil, invalid("payload_limit", "payload exceeds byte limit")
	}
	return out, nil
}
func strictBase64(s string, max int) ([]byte, error) {
	if len(s) > base64.StdEncoding.EncodedLen(max) {
		return nil, invalid("payload_limit", "encoded field exceeds limit")
	}
	b, err := base64.StdEncoding.Strict().DecodeString(s)
	if err != nil || len(b) > max || base64.StdEncoding.EncodeToString(b) != s {
		return nil, invalid("invalid_payload", "noncanonical or invalid base64")
	}
	return b, nil
}
func identity(raw []byte, key string) error {
	var h struct {
		Version int    `json:"protocol_version"`
		Key     string `json:"signer_key_id"`
	}
	if json.Unmarshal(raw, &h) != nil || h.Version != ProtocolVersion || h.Key != key || key == "" || len(key) > 256 {
		return invalid("signature_payload_mismatch", "signed protocol/key identity differs")
	}
	return nil
}
func Sign(t PayloadType, v any, keyID string, private ed25519.PrivateKey) (Envelope, error) {
	raw, err := Canonical(v)
	if err != nil {
		return Envelope{}, err
	}
	return SignCanonical(t, raw, keyID, private)
}
func SignCanonical(t PayloadType, raw []byte, keyID string, private ed25519.PrivateKey) (Envelope, error) {
	if !t.Valid() {
		return Envelope{}, invalid("signature_payload_mismatch", "unsupported payload purpose")
	}
	canonical, err := CanonicalJSON(raw)
	if err != nil {
		return Envelope{}, err
	}
	if !bytes.Equal(raw, canonical) {
		return Envelope{}, invalid("invalid_payload", "signed payload must be canonical")
	}
	if err := identity(raw, keyID); err != nil {
		return Envelope{}, err
	}
	if len(private) != ed25519.PrivateKeySize {
		return Envelope{}, invalid("invalid_signing_key", "invalid private key length")
	}
	derived := ed25519.NewKeyFromSeed(private[:ed25519.SeedSize])
	if !bytes.Equal(derived, private) {
		return Envelope{}, invalid("invalid_signing_key", "inconsistent private key")
	}
	sig := ed25519.Sign(private, PAE(t, raw))
	env := Envelope{t, base64.StdEncoding.EncodeToString(raw), []Signature{{keyID, base64.StdEncoding.EncodeToString(sig)}}}
	if b, _ := json.Marshal(env); len(b) > MaxEnvelopeBytes {
		return Envelope{}, invalid("payload_limit", "envelope exceeds request limit")
	}
	return env, nil
}

// Verify returns exactly the canonical bytes verified. Registered key lookup,
// authorization, scope and lease remain the caller's explicit responsibility.
func Verify(env Envelope, purpose PayloadType, keyID string, public ed25519.PublicKey) ([]byte, error) {
	if !purpose.Valid() || env.PayloadType != purpose {
		return nil, invalid("signature_payload_mismatch", "wrong payload purpose")
	}
	if len(public) != ed25519.PublicKeySize {
		return nil, invalid("invalid_signing_key", "invalid public key length")
	}
	if len(env.Signatures) != 1 || env.Signatures[0].KeyID != keyID {
		return nil, invalid("signature_payload_mismatch", "exactly one expected identity signature required")
	}
	if b, _ := json.Marshal(env); len(b) > MaxEnvelopeBytes {
		return nil, invalid("payload_limit", "envelope exceeds request limit")
	}
	raw, err := strictBase64(env.Payload, MaxPayloadBytes)
	if err != nil {
		return nil, err
	}
	canonical, err := CanonicalJSON(raw)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(raw, canonical) {
		return nil, invalid("invalid_payload", "noncanonical signed payload")
	}
	if err = identity(raw, keyID); err != nil {
		return nil, err
	}
	sig, err := strictBase64(env.Signatures[0].Sig, ed25519.SignatureSize)
	if err != nil || len(sig) != ed25519.SignatureSize {
		return nil, invalid("invalid_signature", "invalid signature length/encoding")
	}
	if !ed25519.Verify(public, PAE(purpose, raw), sig) {
		return nil, invalid("invalid_signature", "signature verification failed")
	}
	return raw, nil
}

// Decode verifies first and then decodes solely those bytes into a strict DTO.
func Decode[T any](env Envelope, purpose PayloadType, keyID string, public ed25519.PublicKey) (T, []byte, error) {
	var v T
	raw, err := Verify(env, purpose, keyID, public)
	if err != nil {
		return v, nil, err
	}
	if err = DecodeStrict(raw, &v, MaxPayloadBytes); err != nil {
		return v, nil, err
	}
	return v, raw, nil
}
func DecodeEnvelope(raw []byte) (Envelope, error) {
	var env Envelope
	err := decodeStrict(raw, &env, MaxEnvelopeBytes, base64.StdEncoding.EncodedLen(MaxPayloadBytes))
	return env, err
}
func DecodeStrict(raw []byte, destination any, maxBytes int) error {
	return decodeStrict(raw, destination, maxBytes, MaxScalarBytes)
}
func decodeStrict(raw []byte, destination any, maxBytes, maxScalar int) error {
	if err := validateJSONBounds(raw, maxBytes, maxScalar); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.DisallowUnknownFields()
	decoder.UseNumber()
	if err := decoder.Decode(destination); err != nil {
		return invalid("invalid_payload", "payload does not match strict schema")
	}
	if decoder.Decode(new(any)) != io.EOF {
		return invalid("invalid_payload", "trailing JSON document")
	}
	return nil
}
func ToDocument(env Envelope) (Document, error) {
	if !env.PayloadType.Valid() || len(env.Signatures) != 1 {
		return Document{}, invalid("invalid_payload", "invalid envelope structure")
	}
	raw, err := strictBase64(env.Payload, MaxPayloadBytes)
	if err != nil {
		return Document{}, err
	}
	canonical, err := CanonicalJSON(raw)
	if err != nil || !bytes.Equal(raw, canonical) {
		return Document{}, invalid("invalid_payload", "noncanonical envelope payload")
	}
	return Document{raw, Proof{env.PayloadType, env.Signatures[0].KeyID, env.Signatures[0].Sig}}, nil
}
func (d Document) Envelope() (Envelope, error) {
	raw, err := CanonicalJSON(d.Payload)
	if err != nil {
		return Envelope{}, err
	}
	if !d.Proof.PayloadType.Valid() {
		return Envelope{}, invalid("signature_payload_mismatch", "invalid document purpose")
	}
	return Envelope{d.Proof.PayloadType, base64.StdEncoding.EncodeToString(raw), []Signature{{d.Proof.KeyID, d.Proof.Signature}}}, nil
}

func validateJSON(raw []byte, maxBytes int) error {
	return validateJSONBounds(raw, maxBytes, MaxScalarBytes)
}
func validateJSONBounds(raw []byte, maxBytes, maxScalar int) error {
	if len(raw) == 0 || len(raw) > maxBytes {
		return invalid("payload_limit", "JSON exceeds byte limit")
	}
	if !utf8.Valid(raw) || !validSurrogates(raw) {
		return invalid("invalid_payload", "invalid Unicode")
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	nodes := 0
	var visit func(int) error
	visit = func(depth int) error {
		nodes++
		if nodes > MaxNodes || depth > MaxDepth {
			return invalid("payload_limit", "JSON complexity exceeds limit")
		}
		token, err := decoder.Token()
		if err != nil {
			return invalid("invalid_payload", "malformed JSON")
		}
		switch v := token.(type) {
		case json.Delim:
			if v != '{' && v != '[' {
				return invalid("invalid_payload", "unexpected JSON delimiter")
			}
			seen := map[string]bool{}
			for decoder.More() {
				if v == '{' {
					key, err := decoder.Token()
					if err != nil {
						return invalid("invalid_payload", "malformed JSON key")
					}
					s, ok := key.(string)
					if !ok || seen[s] || len(s) > maxScalar {
						return invalid("invalid_payload", "duplicate/invalid JSON key")
					}
					seen[s] = true
					nodes++
				}
				if err := visit(depth + 1); err != nil {
					return err
				}
			}
			end, err := decoder.Token()
			if err != nil || (v == '{' && end != json.Delim('}')) || (v == '[' && end != json.Delim(']')) {
				return invalid("invalid_payload", "unclosed JSON value")
			}
		case string:
			if len(v) > maxScalar {
				return invalid("payload_limit", "JSON scalar exceeds limit")
			}
		case json.Number:
			f, err := strconv.ParseFloat(string(v), 64)
			if err != nil || math.IsInf(f, 0) || math.IsNaN(f) || (math.Trunc(f) == f && math.Abs(f) > 9007199254740991) {
				return invalid("invalid_payload", "integer/number requires interoperable representation; use decimal strings")
			}
		}
		return nil
	}
	if err := visit(1); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		return invalid("invalid_payload", "trailing JSON value")
	}
	return nil
}

// encoding/json replaces lone UTF-16 surrogates. Reject them before that loss.
func validSurrogates(raw []byte) bool {
	in := false
	for i := 0; i < len(raw); i++ {
		if raw[i] == '"' {
			in = !in
			continue
		}
		if !in || raw[i] != '\\' {
			continue
		}
		i++
		if i >= len(raw) {
			return false
		}
		if raw[i] != 'u' {
			continue
		}
		if i+4 >= len(raw) {
			return false
		}
		v, err := strconv.ParseUint(string(raw[i+1:i+5]), 16, 16)
		if err != nil {
			return false
		}
		i += 4
		if v >= 0xdc00 && v <= 0xdfff {
			return false
		}
		if v < 0xd800 || v > 0xdbff {
			continue
		}
		if i+6 >= len(raw) || raw[i+1] != '\\' || raw[i+2] != 'u' {
			return false
		}
		low, err := strconv.ParseUint(string(raw[i+3:i+7]), 16, 16)
		if err != nil || low < 0xdc00 || low > 0xdfff {
			return false
		}
		i += 6
	}
	return true
}
