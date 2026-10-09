package woscli

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"os"
	"os/exec"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

func nativeKeyringProfile(t *testing.T, name string) {
	t.Helper()
	_, profile, expectedToken := profileFixtureV2(t, name)
	defer clear(expectedToken)
	profile.Authentication.CredentialRef = "keyring:wos/" + name + "/api"
	profile.Signing.PrivateKeyRef = "keyring:wos/" + name + "/signing"
	if err := profile.SealBinding(expectedToken); err != nil {
		t.Fatal(err)
	}
	resolver := SecretResolver{WorkspaceRoot: t.TempDir(), Keyring: OSKeyring{}}
	token, err := resolver.Read(profile.Authentication.CredentialRef)
	if err != nil {
		t.Fatal("native keyring credential reference unavailable")
	}
	defer clear(token)
	if !bytes.Equal(token, expectedToken) {
		t.Fatal("native keyring credential differs")
	}
	if err = profile.VerifyBinding(token); err != nil {
		t.Fatal("native keyring credential did not authenticate the exact profile")
	}
	private, err := readProfilePrivateV2(profile, resolver)
	if err != nil {
		t.Fatal("native keyring signing reference unavailable or mismatched")
	}
	defer clear(private)
	envelope, err := signing.Sign(signing.WorkReturn, map[string]any{"summary": "Native protected signer", "protocol_version": 2}, profile.Signing.KeyID.String(), private)
	if err != nil {
		t.Fatal("native protected signing failed")
	}
	if _, err = signing.Verify(envelope, signing.WorkReturn, profile.Signing.KeyID.String(), private.Public().(ed25519.PublicKey)); err != nil {
		t.Fatal("native protected signature verification failed")
	}
}

func TestNativeOSKeyringProtectedProfileV2(t *testing.T) {
	if os.Getenv("WOS_TEST_NATIVE_KEYRING") != "1" {
		t.Skip("requires actual OS credential service; platform CI sets WOS_TEST_NATIVE_KEYRING=1")
	}
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatal(err)
	}
	name := "native_" + hex.EncodeToString(nonce)
	backend := OSKeyring{}
	_, _, token := profileFixtureV2(t, name)
	defer clear(token)
	values := map[string][]byte{"api": token, "signing": []byte(base64.StdEncoding.EncodeToString(make([]byte, 32)))}
	for purpose, value := range values {
		user := name + "/" + purpose
		if _, err := backend.Get("wos", user); err == nil {
			t.Fatal("refusing to overwrite an existing native credential")
		}
		if err := backend.Set("wos", user, string(value)); err != nil {
			t.Fatal("native credential service cannot provision a protected fixture")
		}
		t.Cleanup(func() {
			if err := backend.Delete("wos", user); err != nil {
				t.Error("native credential fixture cleanup failed")
			}
		})
	}
	nativeKeyringProfile(t, name)
	// The same references survive a real process boundary; no private bytes are
	// passed in process arguments or environment variables.
	child := exec.Command(os.Args[0], "-test.run=^TestNativeOSKeyringChildV2$")
	child.Env = append(os.Environ(), "WOS_NATIVE_KEYRING_CHILD_PROFILE="+name)
	if output, err := child.CombinedOutput(); err != nil {
		t.Fatalf("native keyring child failed: %v; %s", err, output)
	}
}

func TestNativeOSKeyringChildV2(t *testing.T) {
	name := os.Getenv("WOS_NATIVE_KEYRING_CHILD_PROFILE")
	if name == "" {
		t.Skip("native child fixture only")
	}
	if !validProfileName(name) {
		t.Fatal("invalid native child profile")
	}
	nativeKeyringProfile(t, name)
}
