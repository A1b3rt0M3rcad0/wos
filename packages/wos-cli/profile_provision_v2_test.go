package woscli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	keyring "github.com/zalando/go-keyring"
)

type provisionKeyringV2 struct {
	values map[string]string
	writes int
}

func (k *provisionKeyringV2) Get(_, name string) (string, error) {
	if value, ok := k.values[name]; ok {
		return value, nil
	}
	return "", keyring.ErrNotFound
}
func (k *provisionKeyringV2) Set(_, name, value string) error {
	k.writes++
	k.values[name] = value
	return nil
}
func (k *provisionKeyringV2) Delete(_, name string) error { delete(k.values, name); return nil }
func TestProfileProtectedProvisioningReusesKeyAndRejectsUnapprovedPins(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	project, template, token := profileFixtureV2(t, "executor_a")
	enrollment := d.MustParseID("0199ac10-0000-7000-8000-000000000007")
	challenge := d.SigningEnrollment{ID: enrollment, NamespaceID: template.Binding.NamespaceID, PrincipalID: template.Binding.PrincipalID, CredentialID: template.Binding.CredentialID, KeyID: template.Signing.KeyID, Purpose: "agent", Nonce: "authorized-fixture", ExpiresAt: time.Now().Add(time.Minute)}
	calls := 0
	endpoint := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		calls++
		switch r.URL.Path {
		case "/api/v1/security/signing-identity":
			json.NewEncoder(rw).Encode(a.SigningIdentityView{Actor: d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "executor"}, ServerID: project.Connection.ExpectedServerID.String(), NamespaceID: project.Scope.NamespaceID, PrincipalID: template.Binding.PrincipalID, CredentialID: template.Binding.CredentialID, Enrollment: &challenge})
		default:
			json.NewEncoder(rw).Encode(a.SignedStateResult{NamespaceID: project.Scope.NamespaceID, Resource: "trust", ProtocolVersion: 2, Server: &d.ServerIdentity{ID: project.Connection.ExpectedServerID, IssuerKeyID: template.Binding.IssuerKeys[0].KeyID, PublicKey: template.Binding.IssuerKeys[0].PublicKey, Fingerprint: template.Binding.IssuerKeys[0].Fingerprint}})
		}
	}))
	defer endpoint.Close()
	project.Connection.ServerURL = endpoint.URL
	w, e := OpenWorkspace(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	if e = w.CreateV2(".wos/project.yaml", project); e != nil {
		t.Fatal(e)
	}
	backend := &provisionKeyringV2{values: map[string]string{}}
	resolver := SecretResolver{WorkspaceRoot: w.canonicalPath, Keyring: backend}
	options := options{values: map[string]string{"token-stdin": "true", "generate-signing-key": "true", "server": endpoint.URL, "server-id": project.Connection.ExpectedServerID.String(), "issuer-fingerprint": template.Binding.IssuerKeys[0].Fingerprint, "enrollment": enrollment.String()}}
	rejected := options
	rejected.values = map[string]string{}
	for k, v := range options.values {
		rejected.values[k] = v
	}
	rejected.values["server"] = "https://unapproved.example"
	if _, e = prepareGeneratedProfileV2(context.Background(), w, rejected, "executor_a", resolver, strings.NewReader(string(token))); e == nil || calls != 0 || backend.writes != 0 {
		t.Fatal("unapproved destination transmitted credential or provisioned secret")
	}
	rejected.values["server"] = endpoint.URL
	rejected.values["issuer-fingerprint"] = "sha256:" + strings.Repeat("1", 64)
	if _, e = prepareGeneratedProfileV2(context.Background(), w, rejected, "executor_a", resolver, strings.NewReader(string(token))); e == nil || backend.writes != 0 {
		t.Fatal("unapproved issuer provisioned secret")
	}
	profile, e := prepareGeneratedProfileV2(context.Background(), w, options, "executor_a", resolver, strings.NewReader(string(token)+"\n"))
	if e != nil {
		t.Fatal(e)
	}
	if backend.writes != 2 {
		t.Fatal("key and token not provisioned in explicit keyring")
	}
	recovered, e := prepareGeneratedProfileV2(context.Background(), w, options, "executor_a", resolver, strings.NewReader(string(token)))
	if e != nil {
		t.Fatal(e)
	}
	if recovered.Signing.PublicKeyFingerprint != profile.Signing.PublicKeyFingerprint || backend.writes != 2 {
		t.Fatal("interrupted provisioning regenerated or overwrote key")
	}
	if _, e = prepareGeneratedProfileV2(context.Background(), w, options, "executor_a", resolver, strings.NewReader("replacement-token-at-least-thirty-two-characters")); e == nil || backend.writes != 2 {
		t.Fatal("credential entry silently overwritten")
	}
	unavailable := resolver
	unavailable.Keyring = nil
	if _, e = prepareGeneratedProfileV2(context.Background(), w, options, "another_profile", unavailable, strings.NewReader(string(token))); e == nil {
		t.Fatal("plaintext fallback when keyring unavailable")
	}
	importOptions := options
	importOptions.values = map[string]string{}
	for key, value := range options.values {
		importOptions.values[key] = value
	}
	importOptions.values["generate-signing-key"] = "false"
	importOptions.values["token-stdin"] = "false"
	importOptions.values["signing-key-stdin"] = "true"
	importOptions.values["credential-ref"] = "env:PROTECTED_IMPORT_TOKEN"
	resolver.Environment = func(string) string { return string(token) }
	seed := base64.StdEncoding.EncodeToString(make([]byte, 32))
	imported, e := prepareGeneratedProfileV2(context.Background(), w, importOptions, "imported_seed", resolver, strings.NewReader(seed+"\n"))
	if e != nil {
		t.Fatal(e)
	}
	private, e := readProfilePrivateV2(imported, resolver)
	if e != nil {
		t.Fatal(e)
	}
	clear(private)
	if backend.values["imported_seed/signing"] != seed || backend.writes != 3 {
		t.Fatal("dedicated seed input was not imported exactly once")
	}
	importOptions.values["token-stdin"] = "true"
	if _, e = prepareGeneratedProfileV2(context.Background(), w, importOptions, "mixed_input", resolver, strings.NewReader(seed)); e == nil || backend.writes != 3 {
		t.Fatal("token and seed ambiguously shared stdin")
	}
}
