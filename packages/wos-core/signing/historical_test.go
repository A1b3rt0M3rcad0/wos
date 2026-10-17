package signing

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
)

// Only tests can create protocol-2 fixtures. No legacy issuance API is exported.
func legacyTestEnvelope(t *testing.T, purpose PayloadType, value any) Envelope {
	t.Helper()
	raw, err := Canonical(value)
	if err != nil {
		t.Fatal(err)
	}
	return Envelope{purpose, base64.StdEncoding.EncodeToString(raw), []Signature{{"vector-key", base64.StdEncoding.EncodeToString(ed25519.Sign(vectorKey(), PAE(purpose, raw)))}}}
}

func TestIndependentLegacyVectorRemainsVerifiableAndCannotExecute(t *testing.T) {
	fixture, err := os.ReadFile("testdata/dsse-vector.json")
	if err != nil {
		t.Fatal(err)
	}
	var vector struct {
		PublicKey string   `json:"public_key_hex"`
		Canonical string   `json:"canonical_payload"`
		Digest    string   `json:"request_digest"`
		Envelope  Envelope `json:"envelope"`
	}
	if err = json.Unmarshal(fixture, &vector); err != nil {
		t.Fatal(err)
	}
	public, _ := hex.DecodeString(vector.PublicKey)
	payload, exact, err := DecodeHistorical[testPayload](vector.Envelope, WorkReturn, "vector-key", public)
	if err != nil || string(exact) != vector.Canonical || Digest(exact) != vector.Digest || payload.ProtocolVersion != 2 {
		t.Fatalf("historical proof changed: %v", err)
	}
	if _, err = Verify(vector.Envelope, WorkReturn, "vector-key", public); err == nil {
		t.Fatal("historical proof accepted at current mutation boundary")
	}
	if _, err = Sign(LegacyWorkReturn, payload, "vector-key", vectorKey()); err == nil {
		t.Fatal("legacy issuance is available")
	}
	if _, err = Sign(WorkReturn, payload, "vector-key", vectorKey()); err == nil {
		t.Fatal("historical payload silently upgraded to current issuance")
	}
	document, err := ToDocument(vector.Envelope)
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := document.Envelope()
	if err != nil || !reflect.DeepEqual(rebuilt, vector.Envelope) {
		t.Fatal("historical document changed its payload or proof", err)
	}
}

func TestHistoricalProjectionKeepsExactProofAndOpaqueContent(t *testing.T) {
	type scope struct {
		TenantID string `json:"tenant_id"`
	}
	type spec struct {
		Scope    scope                      `json:"scope"`
		Refs     []scope                    `json:"refs"`
		ByName   map[string]scope           `json:"by_name"`
		Metadata map[string]any             `json:"metadata"`
		Opaque   json.RawMessage            `json:"opaque"`
		Raw      map[string]json.RawMessage `json:"raw"`
	}
	value := map[string]any{
		"protocol_version": 2, "signer_key_id": "vector-key", "namespace_id": "legacy-tenant",
		"contract_kind": "execution", "contract_id": "contract", "work_item_id": "work",
		"spec": map[string]any{
			"scope":    map[string]any{"namespace_id": "legacy-tenant"},
			"refs":     []any{map[string]any{"namespace_id": "legacy-tenant"}},
			"by_name":  map[string]any{"namespace_id": map[string]any{"namespace_id": "legacy-tenant"}},
			"metadata": map[string]any{"namespace_id": "authored namespace_id", "tenant_id": "authored tenant_id"},
			"opaque":   map[string]any{"namespace_id": "opaque", "tenant_id": "<opaque>"},
			"raw":      map[string]any{"namespace_id": map[string]any{"namespace_id": "<raw>"}},
		},
	}
	envelope := legacyTestEnvelope(t, LegacyContractSpec, value)
	rawBefore, _ := base64.StdEncoding.DecodeString(envelope.Payload)
	decoded, exact, err := DecodeHistorical[SpecPayload[spec]](envelope, ContractSpec, "vector-key", vectorKey().Public().(ed25519.PublicKey))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(exact, rawBefore) || decoded.ProtocolVersion != 2 || decoded.TenantID != "legacy-tenant" || decoded.Spec.Scope.TenantID != "legacy-tenant" || decoded.Spec.Refs[0].TenantID != "legacy-tenant" || decoded.Spec.ByName["namespace_id"].TenantID != "legacy-tenant" {
		t.Fatal("historical binding or exact bytes lost")
	}
	if decoded.Spec.Metadata["namespace_id"] != "authored namespace_id" || decoded.Spec.Metadata["tenant_id"] != "authored tenant_id" || string(decoded.Spec.Opaque) != `{"namespace_id":"opaque","tenant_id":"<opaque>"}` || string(decoded.Spec.Raw["namespace_id"]) != `{"namespace_id":"<raw>"}` {
		t.Fatalf("opaque/authored content rewritten: %+v", decoded.Spec)
	}
	binding, err := ReadHistoricalBinding(exact)
	if err != nil || binding.TenantID != decoded.TenantID || binding.ProtocolVersion != 2 {
		t.Fatal("historical binding cannot be inspected", err)
	}
	if _, err = Sign(ContractSpec, decoded, "vector-key", vectorKey()); err == nil {
		t.Fatal("a read projection was accepted as current signed intent")
	}
}

func TestHistoricalPurposeAndProtocolCannotBeMixed(t *testing.T) {
	public := vectorKey().Public().(ed25519.PublicKey)
	for _, tc := range []struct {
		purpose PayloadType
		version int
		field   string
	}{
		{LegacyWorkReturn, 3, "namespace_id"},
		{WorkReturn, 2, "tenant_id"},
		{LegacyWorkReturn, 2, "tenant_id"},
		{WorkReturn, 3, "namespace_id"},
		{LegacyReviewReturn, 2, "namespace_id"},
	} {
		value := map[string]any{"protocol_version": tc.version, "signer_key_id": "vector-key", tc.field: "tenant"}
		envelope := legacyTestEnvelope(t, tc.purpose, value)
		if _, err := VerifyHistorical(envelope, WorkReturn, "vector-key", public); err == nil {
			t.Fatalf("accepted mismatched purpose/version/field: %+v", tc)
		}
	}
	valid := legacyTestEnvelope(t, LegacyWorkReturn, map[string]any{"protocol_version": 2, "signer_key_id": "vector-key", "namespace_id": "tenant"})
	if _, err := VerifyHistorical(valid, LegacyWorkReturn, "vector-key", public); err == nil {
		t.Fatal("caller bypassed explicit current-purpose matching")
	}
}

func TestHistoricalProjectionRejectsAmbiguousAndWrongVersionFields(t *testing.T) {
	type nested struct {
		TenantID string `json:"tenant_id"`
	}
	type payload struct {
		Binding
		Scope nested `json:"scope"`
	}
	for _, raw := range []string{
		`{"protocol_version":2,"namespace_id":"one","tenant_id":"one"}`,
		`{"protocol_version":2,"tenant_id":"one"}`,
		`{"protocol_version":2,"scope":{"tenant_id":"one"}}`,
		`{"protocol_version":2,"scope":{"namespace_id":"one","tenant_id":"one"}}`,
		`{"protocol_version":2,"scope":{"Namespace_id":"one"}}`,
		`{"protocol_version":2,"namespace_id":"one","namespace_id":"two"}`,
		`{"protocol_version":3,"namespace_id":"one"}`,
		`{"protocol_version":3,"scope":{"namespace_id":"one"}}`,
		`{"protocol_version":1,"namespace_id":"one"}`,
		`{"namespace_id":"one"}`,
	} {
		var decoded payload
		if err := DecodeHistoricalPayload([]byte(raw), &decoded, 4096); err == nil {
			t.Fatalf("accepted ambiguous historical projection: %s", raw)
		}
	}
	var decoded payload
	if err := DecodeHistoricalPayload([]byte(`{"protocol_version":3,"tenant_id":"current","scope":{"tenant_id":"current"}}`), &decoded, 4096); err != nil || decoded.TenantID != "current" || decoded.ProtocolVersion != 3 {
		t.Fatal("current historical read failed", err)
	}
}

func TestStoredJSONProjectionPreservesLargeHistoriesAndRawMessages(t *testing.T) {
	type entry struct {
		TenantID string `json:"tenant_id"`
	}
	type record struct {
		TenantID string          `json:"tenant_id"`
		History  []entry         `json:"history"`
		Content  string          `json:"content"`
		Proof    json.RawMessage `json:"proof"`
	}
	proof := `{ "namespace_id": "signed", "tenant_id": "<authored>", "escaped": "\u003c" }`
	history := strings.Repeat(`{"namespace_id":"tenant"},`, MaxNodes)
	raw := []byte(`{"namespace_id":"tenant","history":[` + strings.TrimSuffix(history, ",") + `],"content":"` + strings.Repeat("x", MaxScalarBytes+1) + `","proof":` + proof + `,"newer_field":"ignored as before"}`)
	before := bytes.Clone(raw)
	var stored record
	if err := DecodeStoredJSON(raw, &stored, 0); err != nil {
		t.Fatal("signed-envelope limits leaked into stored aggregate decoder", err)
	}
	if stored.TenantID != "tenant" || len(stored.History) != MaxNodes || stored.History[len(stored.History)-1].TenantID != "tenant" || string(stored.Proof) != proof || !bytes.Equal(raw, before) {
		t.Fatal("stored history/proof/source bytes changed")
	}
	if err := DecodeStoredJSON(raw, &stored, 4096); err == nil {
		t.Fatal("ignored explicit stored byte limit")
	}
	for _, invalid := range []string{
		`{"tenant_id":"one","namespace_id":"one"}`,
		`{"namespace_id":"one","namespace_id":"one"}`,
		`{"namespace_id":"one"} {}`,
	} {
		if err := DecodeStoredJSON([]byte(invalid), &stored, 0); err == nil {
			t.Fatalf("accepted ambiguous stored document: %s", invalid)
		}
	}
}
