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
)

func TestCentralProtectedReferencesProvisionWithoutKeyringOrPlaintext(t *testing.T) {
	project, template, token := profileFixtureV2(t, "executor")
	enrollment := d.MustParseID("0199ac10-0000-7000-8000-000000000007")
	challenge := d.SigningEnrollment{ID: enrollment, NamespaceID: template.Binding.NamespaceID, PrincipalID: template.Binding.PrincipalID, CredentialID: template.Binding.CredentialID, KeyID: template.Signing.KeyID, Purpose: "agent", Nonce: "authorized-reference-fixture", ExpiresAt: time.Now().Add(time.Minute), ExpectedFingerprint: template.Signing.PublicKeyFingerprint}
	calls := 0
	endpoint := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer "+string(token) {
			t.Error("protected credential was not supplied to approved origin")
		}
		switch r.URL.Path {
		case "/api/v1/security/signing-identity":
			json.NewEncoder(rw).Encode(a.SigningIdentityView{Actor: d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "executor"}, ServerID: project.Connection.ExpectedServerID.String(), NamespaceID: project.Scope.NamespaceID, PrincipalID: template.Binding.PrincipalID, CredentialID: template.Binding.CredentialID, Enrollment: &challenge})
		default:
			json.NewEncoder(rw).Encode(a.SignedStateResult{NamespaceID: project.Scope.NamespaceID, Resource: "trust", ProtocolVersion: 2, Server: &d.ServerIdentity{ID: project.Connection.ExpectedServerID, IssuerKeyID: template.Binding.IssuerKeys[0].KeyID, PublicKey: template.Binding.IssuerKeys[0].PublicKey, Fingerprint: template.Binding.IssuerKeys[0].Fingerprint}})
		}
	}))
	defer endpoint.Close()
	project.Connection.ServerURL = endpoint.URL
	project.Scope.DefaultOutcomeID = ""
	w, e := OpenWorkspace(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	if e = w.CreateV2(".wos/project.yaml", project); e != nil {
		t.Fatal(e)
	}
	resolver := SecretResolver{WorkspaceRoot: w.canonicalPath, Environment: func(name string) string {
		if name == "PROVISION_TOKEN" {
			return string(token)
		}
		if name == "PROVISION_SIGNING" {
			return base64.StdEncoding.EncodeToString(make([]byte, 32))
		}
		return ""
	}}
	o := options{values: map[string]string{"credential-ref": "env:PROVISION_TOKEN", "private-key-ref": "env:PROVISION_SIGNING", "server": endpoint.URL, "server-id": project.Connection.ExpectedServerID.String(), "issuer-fingerprint": template.Binding.IssuerKeys[0].Fingerprint, "enrollment": enrollment.String()}}
	profile, e := prepareReferencedProfileV2(context.Background(), w, o, "executor", resolver)
	if e != nil {
		t.Fatal(e)
	}
	if profile.Signing.PublicKeyFingerprint != template.Signing.PublicKeyFingerprint || profile.Authentication.CredentialRef != "env:PROVISION_TOKEN" || profile.Signing.PrivateKeyRef != "env:PROVISION_SIGNING" {
		t.Fatal("protected references or key identity changed")
	}
	raw, e := EncodeV2Document(profile)
	if e != nil || strings.Contains(string(raw), string(token)) || strings.Contains(string(raw), base64.StdEncoding.EncodeToString(make([]byte, 32))) {
		t.Fatal("profile exported credential/private seed", e)
	}
	before := calls
	o.values["server"] = "https://unapproved.example"
	if _, e = prepareReferencedProfileV2(context.Background(), w, o, "executor", resolver); e == nil || calls != before {
		t.Fatal("unapproved origin received credential")
	}
	o.values["server"] = endpoint.URL
	challenge.ExpectedFingerprint = "sha256:" + strings.Repeat("1", 64)
	if _, e = prepareReferencedProfileV2(context.Background(), w, o, "executor", resolver); e == nil {
		t.Fatal("enrollment accepted a different signing fingerprint")
	}
	challenge.ExpectedFingerprint = template.Signing.PublicKeyFingerprint
	o.values["token-stdin"] = "true"
	before = calls
	if _, e = prepareReferencedProfileV2(context.Background(), w, o, "executor", resolver); e == nil || calls != before {
		t.Fatal("ambiguous reference/stdin modes accepted")
	}
}
