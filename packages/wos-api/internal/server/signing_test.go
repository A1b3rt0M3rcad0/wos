package server

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"reflect"
	"strings"
	"testing"
)

func TestSignedServerIdentityPersistsAndRejectsSilentKeyReplacement(t *testing.T) {
	forEachRuntimeStorage(t, func(t *testing.T, cfg Config) {
		seed := make([]byte, ed25519.SeedSize)
		if _, err := rand.Read(seed); err != nil {
			t.Fatal(err)
		}
		t.Setenv("WOS_TEST_SIGNER_SEED", base64.StdEncoding.EncodeToString(seed))
		cfg.Signing.SeedEnv = "WOS_TEST_SIGNER_SEED"
		cfg.Auth.Mode = AuthModeAPIToken
		cfg.Auth.BootstrapToken = strings.Repeat("a", 40)
		cfg.Auth.BootstrapNamespaceID = "0199a444-0000-7000-8000-000000000001"
		cfg.Auth.BootstrapNamespaceName = "signed-runtime"
		runtime, err := OpenRuntime(cfg)
		if err != nil {
			t.Fatal(err)
		}
		uow, err := runtime.store.Begin(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		identity, err := uow.(ports.ServerIdentityUnitOfWork).ServerIdentity().Server(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		_ = uow.Rollback()
		_ = runtime.Close()
		runtime, err = OpenRuntime(cfg)
		if err != nil {
			t.Fatal(err)
		}
		uow, err = runtime.store.Begin(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		restored, err := uow.(ports.ServerIdentityUnitOfWork).ServerIdentity().Server(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		_ = uow.Rollback()
		_ = runtime.Close()
		if !reflect.DeepEqual(identity, restored) {
			t.Fatal("server trust changed on restart")
		}
		private := ed25519.NewKeyFromSeed(seed)
		local := localIssuer{identity: identity, private: private}
		payload := signing.Binding{ProtocolVersion: 2, ServerID: identity.ID.String(), NamespaceID: cfg.Auth.BootstrapNamespaceID, PrincipalID: "executor", SignerKeyID: identity.IssuerKeyID.String()}
		raw, err := signing.Canonical(payload)
		if err != nil {
			t.Fatal(err)
		}
		proofRaw, err := local.SignCanonical(string(signing.ContractAuthority), raw)
		if err != nil {
			t.Fatal(err)
		}
		var proof signing.Proof
		if err = json.Unmarshal(proofRaw, &proof); err != nil {
			t.Fatal(err)
		}
		envelope, err := (signing.Document{Payload: raw, Proof: proof}).Envelope()
		if err != nil {
			t.Fatal(err)
		}
		public, err := identity.PublicBytes()
		if err != nil {
			t.Fatal(err)
		}
		if _, err = signing.Verify(envelope, signing.ContractAuthority, identity.IssuerKeyID.String(), public); err != nil {
			t.Fatal(err)
		}
		replacement := make([]byte, ed25519.SeedSize)
		if _, err = rand.Read(replacement); err != nil {
			t.Fatal(err)
		}
		t.Setenv("WOS_TEST_SIGNER_SEED", base64.StdEncoding.EncodeToString(replacement))
		if runtime, err = OpenRuntime(cfg); err == nil {
			runtime.Close()
			t.Fatal("silent replacement of pinned issuer accepted")
		}
		t.Setenv("WOS_TEST_SIGNER_SEED", "")
		if runtime, err = OpenRuntime(cfg); err == nil {
			runtime.Close()
			t.Fatal("missing referenced seed silently downgraded")
		}
	})
}
