package acceptance_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"os"
	"testing"

	cli "github.com/A1b3rt0M3rcad0/wos/packages/wos-cli"
)

// Platform CI runs the existing actual HTTP/CLI lost-response journey through
// native protected references. Unconfigured hosts retain the env fixture and
// do not claim native credential-service execution.
func useNativeKeyringProfile(t *testing.T, profile *cli.ProfileV2, token string, private ed25519.PrivateKey) {
	t.Helper()
	if os.Getenv("WOS_TEST_NATIVE_KEYRING") != "1" {
		return
	}
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		t.Fatal(err)
	}
	name := "native_" + hex.EncodeToString(nonce)
	backend := cli.OSKeyring{}
	secrets := map[string]string{"api": token, "signing": base64.StdEncoding.EncodeToString(private.Seed())}
	for purpose, value := range secrets {
		user := name + "/" + purpose
		if _, err := backend.Get("wos", user); err == nil {
			t.Fatal("native fixture refuses an existing keyring entry")
		}
		if err := backend.Set("wos", user, value); err != nil {
			t.Fatal("actual native credential service unavailable")
		}
		t.Cleanup(func() {
			if err := backend.Delete("wos", user); err != nil {
				t.Error("native acceptance credential cleanup failed")
			}
		})
	}
	profile.Authentication.CredentialRef = "keyring:wos/" + name + "/api"
	profile.Signing.PrivateKeyRef = "keyring:wos/" + name + "/signing"
	t.Log("actual HTTP/CLI profile uses native OS protected references; no private secret is placed in process arguments or environment")
}
