package woscli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

func TestProfileTrustChild(t *testing.T) {
	root := os.Getenv("WOS_PROFILE_TRUST_CHILD")
	if root == "" {
		return
	}
	var out, diagnostic bytes.Buffer
	code := Run(context.Background(), []string{"profile", "trust", "--workspace", root, "--profile", "executor_a", "--issuer-fingerprint", os.Getenv("WOS_PROFILE_TRUST_FP"), "--output", "json"}, &out, &diagnostic)
	if code != 0 {
		t.Fatalf("trust child %d: %s %s", code, out.String(), diagnostic.String())
	}
	os.Exit(23)
}

func TestExplicitProfileTrustRetainsFrozenIntentionsAcrossNativeProcessDeath(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	project, profile, token, file, intent, oldReceipt := returnFixtureV2(t)
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	w, e := OpenWorkspace(t.TempDir())
	must(e)
	defer w.Close()
	seed := make([]byte, ed25519.SeedSize)
	seed[0] = 9
	private := ed25519.NewKeyFromSeed(seed)
	public := private.Public().(ed25519.PublicKey)
	fp, e := signing.Fingerprint(public)
	must(e)
	newKey := d.MustParseID("0199ac10-0000-7000-8000-000000000020")
	server := d.ServerIdentity{ID: profile.Binding.ServerID, IssuerKeyID: newKey, PublicKey: base64.StdEncoding.EncodeToString(public), Fingerprint: fp, CreatedAt: time.Now().UTC()}
	endpoint := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Error("pin approval attempted remote mutation")
			rw.WriteHeader(500)
			return
		}
		// Real requests must not hold a profile lock while waiting on transport.
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		release, e := w.LockV2(ctx, profile.Name, "")
		if e != nil {
			t.Error("trust RPC held profile lock", e)
			rw.WriteHeader(500)
			return
		}
		release()
		switch r.URL.Path {
		case "/api/v1/security/signing-identity":
			if e := json.NewEncoder(rw).Encode(a.SigningIdentityView{Actor: d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "executor"}, ServerID: profile.Binding.ServerID.String(), NamespaceID: profile.Binding.NamespaceID, PrincipalID: profile.Binding.PrincipalID, CredentialID: profile.Binding.CredentialID}); e != nil {
				t.Error("identity fixture encoding failed", e)
			}
		default:
			if e := json.NewEncoder(rw).Encode(a.SignedStateResult{ProtocolVersion: 2, NamespaceID: profile.Binding.NamespaceID, Resource: "trust", Server: &server}); e != nil {
				t.Error("trust fixture encoding failed", e)
			}
		}
	}))
	defer endpoint.Close()
	project.Connection.ServerURL = endpoint.URL
	profile.Binding.ServerOrigin = endpoint.URL
	must(profile.SealBinding(token))
	var frozen FrozenReturnV2
	raw, e := base64.StdEncoding.Strict().DecodeString(intent.Payload)
	must(e)
	must(json.Unmarshal(raw, &frozen))
	must(sealFrozenV2(profile, token, &frozen))
	file.Local.Pending = &frozen
	raw, e = json.Marshal(frozen)
	must(e)
	intent.Payload = base64.StdEncoding.EncodeToString(raw)
	intent.PayloadDigest = signing.Digest(raw)
	profile.Local.PendingOperations = []PendingOperationV2{intent}
	must(profile.SealBinding(token))
	t.Setenv("TEST_WOS_TOKEN", string(token))
	must(w.CreateV2(".wos/project.yaml", project))
	must(w.CreateV2(profilePathV2(profile.Name), profile))
	// Native aliases must work for a profile mutation, not only for discovery.
	profilePath := filepath.Join(".wos/profiles", profile.Name, "profile.yml")
	must(os.Rename(filepath.Join(w.canonicalPath, profilePathV2(profile.Name)), filepath.Join(w.canonicalPath, profilePath)))
	contractPath := contractPathV2(profile.Name, *intent.ContractID)
	must(w.CreateV2(contractPath, file))
	originalFile, e := w.ReadV2(contractPath)
	must(e)
	originalIntents, e := json.Marshal(profile.Local.PendingOperations)
	must(e)
	var rejected, diagnostic bytes.Buffer
	bad := signing.Digest([]byte("unapproved replacement"))
	code := Run(context.Background(), []string{"profile", "trust", "--workspace", w.canonicalPath, "--profile", profile.Name, "--issuer-fingerprint", bad, "--output", "json"}, &rejected, &diagnostic)
	if code == 0 || !bytes.Contains(rejected.Bytes(), []byte("explicitly approved")) {
		t.Fatalf("unapproved pin rejection missing: %d %s %s", code, rejected.String(), diagnostic.String())
	}
	unchanged, e := w.LoadProfileV2(profile.Name)
	must(e)
	if unchanged.Local.BindingMAC != profile.Local.BindingMAC {
		t.Fatal("unapproved pin changed local binding")
	}
	child := exec.Command(os.Args[0], "-test.run=^TestProfileTrustChild$")
	child.Env = append(os.Environ(), "WOS_PROFILE_TRUST_CHILD="+w.canonicalPath, "WOS_PROFILE_TRUST_FP="+fp)
	output, e := child.CombinedOutput()
	var exit *exec.ExitError
	if !errors.As(e, &exit) || exit.ExitCode() != 23 {
		t.Fatalf("child did not die after durable trust write: %v %s", e, output)
	}
	approved, e := w.LoadProfileV2(profile.Name)
	must(e)
	must(approved.VerifyBinding(token))
	if len(approved.Binding.IssuerKeys) != 2 || len(approved.Local.PriorIssuerBindings) != 1 || approved.Binding.IssuerKeys[0] != profile.Binding.IssuerKeys[0] {
		t.Fatal("historical trust lost")
	}
	afterIntents, e := json.Marshal(approved.Local.PendingOperations)
	must(e)
	afterFile, e := w.ReadV2(contractPath)
	must(e)
	if !bytes.Equal(afterIntents, originalIntents) || !bytes.Equal(afterFile, originalFile) {
		t.Fatal("trust approval rewrote a frozen original intention or contract")
	}
	recovered, binding, e := decodeReturnIntentV2(approved, token, approved.Local.PendingOperations[0])
	must(e)
	if !reflect.DeepEqual(recovered, frozen) {
		t.Fatal("original frozen payload or signature changed")
	}
	_, e = verifyFrozenV2(approved, token, *file.Local.Pending)
	must(e)
	_, e = verifyReturnReceiptV2(approved, frozen, binding, oldReceipt)
	must(e)
	var receipt signing.ReceiptPayload
	must(signing.DecodeStrict(oldReceipt.Payload, &receipt, signing.MaxPayloadBytes))
	receipt.SignerKeyID = newKey.String()
	envelope, e := signing.Sign(signing.AcceptanceReceipt, receipt, newKey.String(), private)
	must(e)
	newReceipt, e := signing.ToDocument(envelope)
	must(e)
	_, e = verifyReturnReceiptV2(approved, frozen, binding, newReceipt)
	must(e)
	// A repeated approval has no new write and keeps annotations intact.
	approvedRaw, e := w.ReadV2(profilePath)
	must(e)
	annotated := append(append([]byte{}, approvedRaw...), []byte("\n# operator annotation retained\n")...)
	must(os.WriteFile(filepath.Join(w.canonicalPath, profilePath), annotated, 0600))
	var out bytes.Buffer
	code = Run(context.Background(), []string{"profile", "trust", "--workspace", w.canonicalPath, "--profile", profile.Name, "--issuer-fingerprint", fp, "--output", "json"}, &out, &diagnostic)
	if code != 0 {
		t.Fatalf("repeat approval %d %s %s", code, out.String(), diagnostic.String())
	}
	observed, e := w.ReadV2(profilePath)
	must(e)
	if !bytes.Equal(observed, annotated) {
		t.Fatal("repeated approval rewrote the profile")
	}
	if _, e = os.Stat(filepath.Join(w.canonicalPath, profilePathV2(profile.Name))); !os.IsNotExist(e) {
		t.Fatal("profile.yml update created another yaml alias")
	}
	forged := approved
	forged.Local.PriorIssuerBindings = nil
	if _, e = verifyFrozenV2(forged, token, frozen); e == nil {
		t.Fatal("unauthenticated lineage accepted")
	}
	moved := approved
	moved.Binding.ServerOrigin = "http://127.0.0.1:8080"
	must(moved.SealBinding(token))
	if _, e = verifyFrozenV2(moved, token, frozen); e == nil {
		t.Fatal("old return moved to another origin through trust lineage")
	}
}
