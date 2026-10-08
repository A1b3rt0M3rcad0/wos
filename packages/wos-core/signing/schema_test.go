package signing

import (
	"crypto/ed25519"
	"testing"
)

func TestSignedSchemaRejectsCaseAliasesAtEveryLevel(t *testing.T) {
	type Nested struct {
		Reason string `json:"reason"`
	}
	type Payload struct {
		Binding
		Material Nested   `json:"material"`
		Items    []Nested `json:"items"`
	}
	for _, raw := range []string{
		`{"protocol_version":2,"Protocol_Version":2}`,
		`{"Protocol_Version":2}`,
		`{"material":{"Reason":"alternate"}}`,
		`{"material":{"reason":"original","REASON":"alternate"}}`,
		`{"items":[{"REASON":"alternate"}]}`,
	} {
		var value Payload
		if DecodeStrict([]byte(raw), &value, 4096) == nil {
			t.Fatalf("accepted alias: %s", raw)
		}
	}
	var value Payload
	if err := DecodeStrict([]byte(`{"protocol_version":2,"material":{"reason":"exact"},"items":[{"reason":"exact"}]}`), &value, 4096); err != nil {
		t.Fatal(err)
	}
}

func TestValidSignatureDoesNotAuthorizeAmbiguousTypedPayload(t *testing.T) {
	envelope, err := Sign(WorkReturn, map[string]any{"protocol_version": 2, "signer_key_id": "vector-key", "fencing_token": "1", "summary": "declared", "Summary": "alternate"}, "vector-key", vectorKey())
	if err != nil {
		t.Fatal(err)
	}
	public := vectorKey().Public().(ed25519.PublicKey)
	if _, err = Verify(envelope, WorkReturn, "vector-key", public); err != nil {
		t.Fatal("fixture must have a valid signature", err)
	}
	if _, _, err = Decode[testPayload](envelope, WorkReturn, "vector-key", public); err == nil {
		t.Fatal("valid signature bypassed the exact typed schema")
	}
}
