package signing

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

type testPayload struct {
	ProtocolVersion int    `json:"protocol_version"`
	SignerKeyID     string `json:"signer_key_id"`
	FencingToken    string `json:"fencing_token"`
	Summary         string `json:"summary"`
}

func vectorKey() ed25519.PrivateKey {
	seed, _ := hex.DecodeString("9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60")
	return ed25519.NewKeyFromSeed(seed)
}
func fixturePayload() testPayload {
	return testPayload{2, "vector-key", "18446744073709551615", "ação ☃"}
}
func TestIndependentEd25519DSSEVector(t *testing.T) {
	raw, err := os.ReadFile("testdata/dsse-vector.json")
	if err != nil {
		t.Fatal(err)
	}
	var v struct {
		PublicKey string   `json:"public_key_hex"`
		Canonical string   `json:"canonical_payload"`
		PAE       string   `json:"pae_base64"`
		Digest    string   `json:"request_digest"`
		Envelope  Envelope `json:"envelope"`
	}
	if err = json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	public, _ := hex.DecodeString(v.PublicKey)
	env, err := Sign(WorkReturn, fixturePayload(), "vector-key", vectorKey())
	if err != nil {
		t.Fatal(err)
	}
	if env.Payload != v.Envelope.Payload || env.Signatures[0] != v.Envelope.Signatures[0] {
		t.Fatal("signature differs from independent Python Ed25519/DSSE vector")
	}
	payload, err := Verify(v.Envelope, WorkReturn, "vector-key", public)
	if err != nil || string(payload) != v.Canonical || Digest(payload) != v.Digest {
		t.Fatal("invalid vector", err)
	}
	if base64.StdEncoding.EncodeToString(PAE(WorkReturn, payload)) != v.PAE {
		t.Fatal("DSSE byte lengths/prefix mismatch")
	}
	decoded, verified, err := Decode[testPayload](env, WorkReturn, "vector-key", public)
	if err != nil || decoded != fixturePayload() || !bytes.Equal(payload, verified) {
		t.Fatal("not the verified DTO", err)
	}
}
func TestSignedDocumentPresentationRoundTrip(t *testing.T) {
	env, _ := Sign(WorkReturn, fixturePayload(), "vector-key", vectorKey())
	doc, err := ToDocument(env)
	if err != nil {
		t.Fatal(err)
	}
	doc.Payload = json.RawMessage("{ \"summary\":\"ação ☃\",\"signer_key_id\":\"vector-key\",\"protocol_version\":2,\"fencing_token\":\"18446744073709551615\" }")
	rebuilt, err := doc.Envelope()
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Verify(rebuilt, WorkReturn, "vector-key", vectorKey().Public().(ed25519.PublicKey)); err != nil {
		t.Fatal("presentation changed authenticated bytes", err)
	}
	doc.Payload = bytes.ReplaceAll(doc.Payload, []byte("ação"), []byte("ação"))
	rebuilt, _ = doc.Envelope()
	if _, err = Verify(rebuilt, WorkReturn, "vector-key", vectorKey().Public().(ed25519.PublicKey)); err == nil {
		t.Fatal("Unicode content was normalized silently")
	}
}
func TestRejectAmbiguousJSONAndComplexity(t *testing.T) {
	cases := []string{`{"x":1,"x":2}`, `{"x":1,"\u0078":2}`, `{"x":"\ud800"}`, `{"x":"\ud800\u0000"}`, `{"x":"\udc00"}`, `{"x":9007199254740993}`, `{"x":1e100}`, `{} {}`, `{"x":NaN}`, `{"x":"` + strings.Repeat("a", MaxScalarBytes+1) + `"}`, strings.Repeat("[", MaxDepth+1) + "0" + strings.Repeat("]", MaxDepth+1)}
	for _, raw := range cases {
		if _, err := CanonicalJSON([]byte(raw)); err == nil {
			t.Fatalf("accepted ambiguous/unsafe JSON %.80s", raw)
		}
	}
	if _, err := CanonicalJSON([]byte("{\"x\":\"\xff\"}")); err == nil {
		t.Fatal("accepted invalid UTF-8")
	}
	if _, err := CanonicalJSON([]byte(`{"x":"\ud83d\ude00"}`)); err != nil {
		t.Fatal("rejected valid surrogate pair", err)
	}
}
func TestPurposeIdentityAndTampering(t *testing.T) {
	key := vectorKey()
	pub := key.Public().(ed25519.PublicKey)
	original, _ := Sign(WorkReturn, fixturePayload(), "vector-key", key)
	for _, mutate := range []func(*Envelope){
		func(e *Envelope) { e.PayloadType = ReviewReturn },
		func(e *Envelope) { e.Signatures[0].KeyID = "another-key" },
		func(e *Envelope) { e.Signatures = append(e.Signatures, e.Signatures[0]) },
		func(e *Envelope) { e.Signatures[0].Sig = base64.StdEncoding.EncodeToString(make([]byte, 64)) },
		func(e *Envelope) {
			e.Payload = base64.StdEncoding.EncodeToString([]byte(`{"protocol_version":2,"signer_key_id":"vector-key","summary":"changed"}`))
		},
		func(e *Envelope) { e.Payload = "\n" + e.Payload },
	} {
		e := original
		e.Signatures = append([]Signature(nil), original.Signatures...)
		mutate(&e)
		if _, err := Verify(e, WorkReturn, "vector-key", pub); err == nil {
			t.Fatal("tampered/purpose/identity accepted")
		}
	}
	changed := fixturePayload()
	changed.SignerKeyID = "other"
	if _, err := Sign(WorkReturn, changed, "vector-key", key); err == nil {
		t.Fatal("inner signer mismatch")
	}
	for _, n := range []int{0, 1, 31, 33, 63, 65} {
		if _, err := Sign(WorkReturn, fixturePayload(), "vector-key", make([]byte, n)); err == nil {
			t.Fatal("invalid private length")
		}
		if _, err := Verify(original, WorkReturn, "vector-key", make([]byte, n)); err == nil {
			t.Fatal("invalid public length")
		}
	}
	key[63] ^= 1
	if _, err := Sign(WorkReturn, fixturePayload(), "vector-key", key); err == nil {
		t.Fatal("inconsistent private key")
	}
}
func TestStrictVerifiedDTOAndLargeTechnicalEnvelope(t *testing.T) {
	p := map[string]any{"protocol_version": 2, "signer_key_id": "vector-key", "surprise": "unsigned companion must not execute"}
	env, err := Sign(WorkReturn, p, "vector-key", vectorKey())
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := Decode[testPayload](env, WorkReturn, "vector-key", vectorKey().Public().(ed25519.PublicKey)); err == nil {
		t.Fatal("unknown field accepted by strict verified DTO")
	}
	p = map[string]any{"protocol_version": 2, "signer_key_id": "vector-key", "values": []string{strings.Repeat("x", 60000), strings.Repeat("y", 60000)}}
	env, err = Sign(WorkReturn, p, "vector-key", vectorKey())
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(env)
	parsed, err := DecodeEnvelope(raw)
	if err != nil {
		t.Fatal("technical base64 incorrectly limited by semantic scalar bound", err)
	}
	if _, err = Verify(parsed, WorkReturn, "vector-key", vectorKey().Public().(ed25519.PublicKey)); err != nil {
		t.Fatal(err)
	}
	raw = append(raw[:len(raw)-1], []byte(`,"command":{"summary":"different"}}`)...)
	if _, err = DecodeEnvelope(raw); err == nil {
		t.Fatal("unsigned companion accepted")
	}
	p["values"] = []string{strings.Repeat("x", 60000), strings.Repeat("y", 60000), strings.Repeat("z", 60000), "overflow" + strings.Repeat("q", 5000)}
	if _, err = Sign(WorkReturn, p, "vector-key", vectorKey()); err == nil {
		t.Fatal("oversized return accepted")
	}
}
func FuzzCanonical(f *testing.F) {
	for _, s := range []string{`{"x":1}`, `{"x":"\ud800"}`, `{"x":1,"x":2}`, `{"protocol_version":2,"signer_key_id":"key"}`} {
		f.Add([]byte(s))
	}
	f.Fuzz(func(t *testing.T, raw []byte) {
		out, err := CanonicalJSON(raw)
		if err == nil {
			again, err := CanonicalJSON(out)
			if err != nil || !bytes.Equal(out, again) {
				t.Fatal("non-idempotent canonical bytes")
			}
		}
	})
}
func FuzzEnvelope(f *testing.F) {
	env, _ := Sign(WorkReturn, fixturePayload(), "vector-key", vectorKey())
	raw, _ := json.Marshal(env)
	f.Add(raw)
	f.Add([]byte(`{}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		env, err := DecodeEnvelope(raw)
		if err == nil {
			_, _ = Verify(env, WorkReturn, "vector-key", vectorKey().Public().(ed25519.PublicKey))
		}
	})
}

func TestExactNewProtocolCounters(t *testing.T) {
	for _, raw := range []string{`"9007199254740993"`, `"18446744073709551615"`, `"0"`} {
		var n Decimal
		if err := json.Unmarshal([]byte(raw), &n); err != nil {
			t.Fatal(err)
		}
		b, _ := json.Marshal(n)
		if string(b) != raw {
			t.Fatal("lost exact counter")
		}
	}
	for _, raw := range []string{`9007199254740993`, `"18446744073709551616"`, `"01"`, `"+1"`, `"-1"`, `"1e2"`} {
		var n Decimal
		if json.Unmarshal([]byte(raw), &n) == nil {
			t.Fatal("accepted inexact/overflow counter", raw)
		}
	}
}
