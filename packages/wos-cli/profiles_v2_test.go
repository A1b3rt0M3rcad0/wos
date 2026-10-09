package woscli

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func profileFixtureV2(t *testing.T, name string) (ProjectV2, ProfileV2, []byte) {
	t.Helper()
	id := func(n string) d.ID { return d.MustParseID("0199ac10-0000-7000-8000-00000000000" + n) }
	public := ed25519.NewKeyFromSeed(make([]byte, 32)).Public().(ed25519.PublicKey)
	token := []byte("test-profile-authentication-credential-at-least-32")
	project := ProjectV2{SchemaVersion: 2, Kind: "WOSProject", Name: "profile test", Connection: ProjectConnectionV2{ServerURL: "http://127.0.0.1:8080", ExpectedServerID: id("1")}, Scope: WorkspaceScope{NamespaceID: id("2"), DefaultOutcomeID: id("3")}, Workspace: ProjectLayoutV2{ProfilesDirectory: "profiles"}}
	profile := ProfileV2{SchemaVersion: 2, Kind: "WOSProfile", Name: name, Binding: ProfileBindingV2{ServerID: id("1"), ServerOrigin: project.Connection.ServerURL, NamespaceID: id("2"), PrincipalID: "executor", CredentialID: id("4"), IssuerKeys: []ProfileIssuerV2{{KeyID: id("5"), PublicKey: base64.StdEncoding.EncodeToString(public), Fingerprint: signing.Digest(public)}}}, Authentication: ProfileAuthenticationV2{CredentialRef: "env:TEST_WOS_TOKEN"}, Signing: ProfileSigningV2{KeyID: id("6"), PrivateKeyRef: "env:TEST_WOS_SIGNING", PublicKeyFingerprint: signing.Digest(public)}, Lease: LeaseConfig{RequestedTTLSeconds: 300}, Output: OutputConfig{DefaultFormat: "json"}, Local: ProfileLocalV2{SchemaVersion: 1, PendingOperations: []PendingOperationV2{}}}
	if e := profile.SealBinding(token); e != nil {
		t.Fatal(e)
	}
	return project, profile, token
}
func TestProfileBindingRejectsDestinationCredentialKeyAndPendingTampering(t *testing.T) {
	_, profile, token := profileFixtureV2(t, "executor_a")
	if e := profile.VerifyBinding(token); e != nil {
		t.Fatal(e)
	}
	variants := []ProfileV2{}
	changed := profile
	changed.Binding.ServerOrigin = "https://attacker.example"
	variants = append(variants, changed)
	changed = profile
	changed.Authentication.CredentialRef = "env:OTHER_TOKEN"
	variants = append(variants, changed)
	changed = profile
	changed.Binding.PrincipalID = "another-principal"
	variants = append(variants, changed)
	changed = profile
	changed.Signing.PrivateKeyRef = "keyring:wos/other/signing"
	variants = append(variants, changed)
	changed = profile
	changed.Local.PendingOperations = []PendingOperationV2{{Operation: "AcquireSignedWorkContract", IdempotencyKey: "substitute-another-intent"}}
	variants = append(variants, changed)
	for _, tampered := range variants {
		if e := tampered.VerifyBinding(token); e == nil {
			t.Fatal("changed profile trusted without explicit onboarding")
		}
	}
	if e := profile.VerifyBinding([]byte("different-current-token-at-least-32")); e == nil {
		t.Fatal("credential replacement silently rebound profile")
	}
	profile.Output.DefaultFormat = "text"
	profile.Lease.RequestedTTLSeconds = 30
	if e := profile.VerifyBinding(token); e != nil {
		t.Fatal("presentation/TTL preference changed identity proof")
	}
}
func TestProfileSelectionCaseSafetyAndObservedEditorCAS(t *testing.T) {
	root := t.TempDir()
	workspace, e := OpenWorkspace(root)
	if e != nil {
		t.Fatal(e)
	}
	defer workspace.Close()
	project, profile, token := profileFixtureV2(t, "executor_a")
	if e = workspace.CreateV2(".wos/project.yaml", project); e != nil {
		t.Fatal(e)
	}
	path := ".wos/profiles/executor_a/profile.yaml"
	if e = workspace.CreateV2(path, profile); e != nil {
		t.Fatal(e)
	}
	if _, e = workspace.SelectProfileV2("", ""); e != nil {
		t.Fatal(e)
	}
	other := profile
	other.Name = "reviewer_b"
	if e = other.SealBinding(token); e != nil {
		t.Fatal(e)
	}
	if e = workspace.CreateV2(".wos/profiles/reviewer_b/profile.yaml", other); e != nil {
		t.Fatal(e)
	}
	if _, e = workspace.SelectProfileV2("", ""); e == nil {
		t.Fatal("multiple profiles implicitly selected")
	}
	selected, e := workspace.SelectProfileV2("executor_a", "reviewer_b")
	if e != nil || selected != "executor_a" {
		t.Fatal("flag did not precede process selection")
	}
	if _, e = workspace.SelectProfileV2("EXECUTOR_A", ""); e == nil {
		t.Fatal("profile selection silently folded case")
	}
	loaded, e := workspace.LoadProfileV2(selected)
	if e != nil {
		t.Fatal(e)
	}
	if e = loaded.VerifyBinding(token); e != nil {
		t.Fatal(e)
	}
	before, e := workspace.ReadV2(path)
	if e != nil {
		t.Fatal(e)
	}
	edited := profile
	edited.Output.DefaultFormat = "text"
	raw, e := EncodeV2Document(edited)
	if e != nil {
		t.Fatal(e)
	}
	if e = os.WriteFile(filepath.Join(root, path), raw, 0600); e != nil {
		t.Fatal(e)
	}
	if e = workspace.WriteV2(path, profile, signing.Digest(before)); e == nil {
		t.Fatal("CLI overwrote observed external edit")
	}
	preserved, e := workspace.ReadV2(path)
	if e != nil || !bytes.Equal(raw, preserved) {
		t.Fatal("conflict lost edited bytes")
	}
	if e = workspace.CreateV2(path, profile); e == nil {
		t.Fatal("exclusive initial creation overwrote existing profile")
	}
	if e = os.Mkdir(filepath.Join(root, ".wos/profiles/Executor_A"), 0700); e != nil {
		if runtime.GOOS == "windows" {
			return
		}
		t.Fatal(e)
	}
	if _, e = workspace.ProfileNamesV2(); e == nil {
		t.Fatal("case-colliding profile directories accepted")
	}
}
func TestV2PresentationLimitsAndExactFieldNames(t *testing.T) {
	_, profile, token := profileFixtureV2(t, "executor_a")
	body := []byte(`{"a":"` + strings.Repeat("x", 40000) + `","b":"` + strings.Repeat("y", 30000) + `"}`)
	profile.Local.PendingOperations = []PendingOperationV2{{ID: d.MustParseID("0199ac10-0000-7000-8000-000000000007"), Scope: ProfilePendingScopeV2{NamespaceID: profile.Binding.NamespaceID, OutcomeID: d.MustParseID("0199ac10-0000-7000-8000-000000000003")}, Operation: "AcquireSignedWorkContract", State: "prepared", IdempotencyKey: "bounded-frozen-acquisition", Payload: base64.StdEncoding.EncodeToString(body), PayloadDigest: signing.Digest(body)}}
	if e := profile.SealBinding(token); e != nil {
		t.Fatal(e)
	}
	raw, e := EncodeV2Document(profile)
	if e != nil {
		t.Fatal(e)
	}
	var decoded ProfileV2
	if e = DecodeV2Document(raw, &decoded); e != nil {
		t.Fatal(e)
	}
	if e = decoded.VerifyBinding(token); e != nil {
		t.Fatal(e)
	}
	if decoded.Local.PendingOperations[0].Payload != profile.Local.PendingOperations[0].Payload {
		t.Fatal("technical payload was truncated or recanonicalized")
	}
	if e = DecodeV2Document(bytes.Replace(raw, []byte("schema_version:"), []byte("Schema_version:"), 1), &decoded); e == nil {
		t.Fatal("case alias accepted")
	}
	if _, e = YAMLJSONV2([]byte("name: " + strings.Repeat("x", 65537))); e == nil {
		t.Fatal("semantic scalar cap bypassed")
	}
	if _, e = YAMLJSONV2([]byte(strings.Repeat(" ", MaxLocalDocumentV2+1))); e == nil {
		t.Fatal("document size cap bypassed")
	}
	if _, e = YAMLJSONV2([]byte("name: &a value\nkind: *a\n")); e == nil {
		t.Fatal("YAML aliases accepted")
	}
	if _, e = YAMLJSONV2([]byte("schema_version: 2\nschema_version: 1\n")); e == nil {
		t.Fatal("duplicate keys accepted")
	}
	for _, name := range []string{"../escape", "CON", "LPT1", "trailing.", "with:colon", "space name"} {
		if validProfileName(name) {
			t.Fatalf("unsafe profile name %q", name)
		}
	}
}
func TestV2RegularFileAndMountedSecretConfinement(t *testing.T) {
	root := t.TempDir()
	workspace, e := OpenWorkspace(root)
	if e != nil {
		t.Fatal(e)
	}
	defer workspace.Close()
	project, _, _ := profileFixtureV2(t, "executor_a")
	if e = workspace.CreateV2(".wos/project.yaml", project); e != nil {
		t.Fatal(e)
	}
	linked := filepath.Join(root, "linked.yaml")
	if e = os.Link(filepath.Join(root, ".wos/project.yaml"), linked); e == nil {
		if _, e = workspace.ReadV2("linked.yaml"); e == nil {
			t.Fatal("hardlinked operational document read")
		}
		os.Remove(linked)
	}
	if _, e = workspace.ReadV2("../outside"); e == nil {
		t.Fatal("path traversal accepted")
	}
	outside := filepath.Join(t.TempDir(), "secret")
	if e = os.WriteFile(outside, []byte("synthetic-secret-value"), 0600); e != nil {
		t.Fatal(e)
	}
	outside, e = filepath.EvalSymlinks(outside)
	if e != nil {
		t.Fatal(e)
	}
	protectSecretFixtureV2(t, outside)
	resolver := SecretResolver{WorkspaceRoot: root}
	secret, e := resolver.Read("mounted:" + outside)
	if e != nil {
		t.Fatal(e)
	}
	clear(secret)
	if _, e = resolver.Read("mounted:" + filepath.Join(root, ".wos/project.yaml")); e == nil {
		t.Fatal("repository-contained secret accepted")
	}
	if runtime.GOOS != "windows" {
		if e = os.Chmod(outside, 0644); e != nil {
			t.Fatal(e)
		}
		if _, e = resolver.Read("mounted:" + outside); e == nil {
			t.Fatal("world-readable mounted secret accepted")
		}
	}
	if e = os.Symlink(outside, filepath.Join(root, "symlink.yaml")); e == nil {
		if _, e = workspace.ReadV2("symlink.yaml"); e == nil {
			t.Fatal("symlink document read")
		}
	}
}
