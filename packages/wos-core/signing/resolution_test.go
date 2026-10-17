package signing

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"testing"
)

func FuzzReturnResolution(f *testing.F) {
	f.Add([]byte(`{"protocol_version":3,"server_id":"fixture-server","tenant_id":"fixture-tenant","principal_id":"fixture-principal","signer_key_id":"resolution-test-key","original_request_digest":"sha256:0000000000000000000000000000000000000000000000000000000000000000"}`))
	private := ed25519.NewKeyFromSeed(make([]byte, 32))
	public := private.Public().(ed25519.PublicKey)
	f.Fuzz(func(t *testing.T, raw []byte) {
		var p ReturnResolutionPayload
		if DecodeStrict(raw, &p, 32<<10) != nil {
			return
		}
		// This codec property does not grant authority; Application separately
		// validates every identity, digest, policy, lease and original envelope.
		p.SignerKeyID = "resolution-test-key"
		canonical, e := Canonical(p)
		if e != nil {
			return
		}
		env, e := SignCanonical(ReturnResolution, canonical, p.SignerKeyID, private)
		if e != nil {
			return
		}
		verified, e := Verify(env, ReturnResolution, p.SignerKeyID, public)
		if e != nil || !bytes.Equal(verified, canonical) {
			t.Fatalf("resolution round trip differs: %v", e)
		}
		if _, e = Verify(env, AcceptanceReceipt, p.SignerKeyID, public); e == nil {
			t.Fatal("resolution accepted as an acceptance receipt")
		}
		mutated := append([]byte(nil), canonical...)
		mutated[len(mutated)-1] ^= 1
		env.Payload = base64.StdEncoding.EncodeToString(mutated)
		if _, e = Verify(env, ReturnResolution, p.SignerKeyID, public); e == nil {
			t.Fatal("changed resolution bytes verified")
		}
	})
}
