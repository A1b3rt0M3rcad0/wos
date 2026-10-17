package woscli

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

func oldProfileFixture(t *testing.T, profile ProfileV2, token []byte) []byte {
	t.Helper()
	type legacyBinding struct {
		ServerID     d.ID              `json:"server_id"`
		ServerOrigin string            `json:"server_origin"`
		NamespaceID  d.ID              `json:"namespace_id"`
		PrincipalID  string            `json:"principal_id"`
		CredentialID d.ID              `json:"credential_id"`
		IssuerKeys   []ProfileIssuerV2 `json:"issuer_keys"`
	}
	legacy := legacyBinding{profile.Binding.ServerID, profile.Binding.ServerOrigin, profile.Binding.TenantID, profile.Binding.PrincipalID, profile.Binding.CredentialID, profile.Binding.IssuerKeys}
	message, err := signing.Canonical(struct {
		SchemaVersion  int                     `json:"schema_version"`
		Name           string                  `json:"name"`
		Binding        legacyBinding           `json:"binding"`
		Authentication ProfileAuthenticationV2 `json:"authentication"`
		Signing        ProfileSigningV2        `json:"signing"`
	}{2, profile.Name, legacy, profile.Authentication, profile.Signing})
	if err != nil {
		t.Fatal(err)
	}
	projected := profile
	mac := hmac.New(sha256.New, token)
	mac.Write([]byte("WOS local profile destination v2\x00"))
	mac.Write(message)
	projected.Local.BindingMAC = hex.EncodeToString(mac.Sum(nil))
	pending, err := json.Marshal(struct {
		Name                string                 `json:"name"`
		BindingMAC          string                 `json:"binding_mac"`
		Operations          []PendingOperationV2   `json:"operations"`
		PriorIssuerBindings []PriorIssuerBindingV2 `json:"prior_issuer_bindings,omitempty"`
	}{profile.Name, projected.Local.BindingMAC, profile.Local.PendingOperations, profile.Local.PriorIssuerBindings})
	if err != nil {
		t.Fatal(err)
	}
	mac = hmac.New(sha256.New, token)
	mac.Write([]byte("WOS local profile pending v2\x00"))
	mac.Write(pending)
	projected.Local.PendingMAC = hex.EncodeToString(mac.Sum(nil))
	raw, err := json.Marshal(projected)
	if err != nil {
		t.Fatal(err)
	}
	return bytes.ReplaceAll(raw, []byte(`"tenant_id"`), []byte(`"namespace_id"`))
}

// The test builds historical DSSE directly. Production signing deliberately
// refuses to issue protocol-2 envelopes, even when reading one is supported.
func oldDocumentFixture(t *testing.T, document signing.Document, purpose signing.PayloadType) signing.Document {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(document.Payload, &fields); err != nil {
		t.Fatal(err)
	}
	fields["protocol_version"] = json.RawMessage(`2`)
	fields["namespace_id"] = fields["tenant_id"]
	delete(fields, "tenant_id")
	raw, err := signing.Canonical(fields)
	if err != nil {
		t.Fatal(err)
	}
	private := ed25519.NewKeyFromSeed(make([]byte, 32))
	pae := []byte(fmt.Sprintf("DSSEv1 %d %s %d %s", len(purpose), purpose, len(raw), raw))
	document.Payload = raw
	document.Proof.PayloadType = purpose
	document.Proof.Signature = base64.StdEncoding.EncodeToString(ed25519.Sign(private, pae))
	return document
}

func TestTenantProfileMigrationAuthenticatesOriginalAndPreservesHistoricalProofs(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	project, profile, token := profileFixtureV2(t, "executor_a")
	_, file := contractFixtureV2(t)
	file.Issued.Specification = oldDocumentFixture(t, file.Issued.Specification, signing.LegacyContractSpec)
	var authority map[string]json.RawMessage
	if err := json.Unmarshal(file.Issued.Authority.Payload, &authority); err != nil {
		t.Fatal(err)
	}
	authority["spec_digest"], _ = json.Marshal(signing.Digest(file.Issued.Specification.Payload))
	file.Issued.Authority.Payload, _ = json.Marshal(authority)
	file.Issued.Authority = oldDocumentFixture(t, file.Issued.Authority, signing.LegacyContractAuthority)
	original := oldProfileFixture(t, profile, token)
	projectRaw, _ := json.Marshal(project)
	projectRaw = bytes.ReplaceAll(projectRaw, []byte(`"tenant_id"`), []byte(`"namespace_id"`))
	w, err := OpenWorkspace(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer w.Close()
	if err = w.writeRawV2(".wos/project.yaml", projectRaw, "", true); err != nil {
		t.Fatal(err)
	}
	path := profilePathV2(profile.Name)
	if err = w.writeRawV2(path, original, "", true); err != nil {
		t.Fatal(err)
	}
	view, err := file.VerifyIssued(profile)
	if err != nil {
		t.Fatal(err)
	}
	if view.Authority.ProtocolVersion != 2 || view.Authority.TenantID != profile.Binding.TenantID.String() {
		t.Fatal("legacy read projection changed protocol or scope")
	}
	contractPath := contractPathV2(profile.Name, d.MustParseID(view.Authority.ContractID))
	if err = w.CreateV2(contractPath, file); err != nil {
		t.Fatal(err)
	}
	contractBefore, err := w.ReadV2(contractPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEST_WOS_TOKEN", string(token))
	if _, err = migrateTenantProfile(context.Background(), w, profile.Name, defaultSecretResolverV2(w)); err != nil {
		t.Fatal(err)
	}
	current, err := w.LoadProfileV2(profile.Name)
	if err != nil {
		t.Fatal(err)
	}
	if err = current.VerifyBinding(token); err != nil {
		t.Fatal(err)
	}
	contractAfter, _ := w.ReadV2(contractPath)
	if !bytes.Equal(contractBefore, contractAfter) {
		t.Fatal("migration rewrote historical proof bytes")
	}
	preserved, err := w.ReadV2(".wos/tenant-migration/executor_a/profile.original")
	if err != nil || !bytes.Equal(preserved, original) {
		t.Fatal("original local profile lost", err)
	}
	if _, err = prepareReturnV2(context.Background(), w, current, nil, token, a.SigningIdentityView{}, d.MustParseID(view.Authority.ContractID), "execution", ""); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatal("historical contract was allowed to create a current mutation", err)
	}
}

func TestTenantProfileMigrationRejectsPendingAndTampering(t *testing.T) {
	_, profile, token := profileFixtureV2(t, "executor_a")
	raw := oldProfileFixture(t, profile, token)
	projected, message, err := legacyProfileProjection(raw)
	if err != nil {
		t.Fatal(err)
	}
	if err = verifyLegacyProfileMAC(projected, message, token); err != nil {
		t.Fatal(err)
	}
	modified := bytes.Replace(raw, []byte("executor\""), []byte("attacker\""), 1)
	projected, message, err = legacyProfileProjection(modified)
	if err != nil {
		t.Fatal(err)
	}
	if verifyLegacyProfileMAC(projected, message, token) == nil {
		t.Fatal("modified legacy identity trusted")
	}
	pending := bytes.Replace(raw, []byte(`"pending_operations":[]`), []byte(`"pending_operations":[{"payload":"frozen-original"}]`), 1)
	if _, _, err = legacyProfileProjection(pending); err == nil || !strings.Contains(err.Error(), "reconciled") {
		t.Fatal("pending old intent migrated", err)
	}
	var request FrozenReturnV2
	request.Envelope.PayloadType = signing.LegacyWorkReturn
	if _, err = verifyFrozenV2(profile, token, request); err == nil || !strings.Contains(err.Error(), "protocol-2") {
		t.Fatal("legacy replay was not explicitly refused", err)
	}
}

func TestHistoricalContractCannotMixProtocolVersions(t *testing.T) {
	profile, file := contractFixtureV2(t)
	file.Issued.Authority = oldDocumentFixture(t, file.Issued.Authority, signing.LegacyContractAuthority)
	if _, err := file.VerifyIssued(profile); err == nil {
		t.Fatal("mixed protocol specification and authority accepted")
	}
}

func TestLegacyUnsignedTenantProjectionRetainsOriginalDigestAndBackupBytes(t *testing.T) {
	_, config, result := journalFixture(t)
	_, profile, _ := profileFixtureV2(t, "executor_a")
	profile.Binding.ServerOrigin = config.Connection.ServerURL
	profile.Binding.TenantID = config.Scope.TenantID
	contract := result.Contract
	contract.Spec.ExecutionSpec.ContextRefs = []d.EntityRef{{Scope: contract.Scope, Kind: d.EntityKindWorkItem, ID: contract.WorkItemID}}
	contract.Spec = d.NormalizeContractSpec(contract.Spec)
	oldSpec, _ := json.Marshal(contract.Spec)
	oldSpec = bytes.ReplaceAll(oldSpec, []byte(`"tenant_id"`), []byte(`"namespace_id"`))
	digest, err := d.SemanticDigest(json.RawMessage(oldSpec))
	if err != nil {
		t.Fatal(err)
	}
	contract.SpecDigest = digest
	backup := []byte(`{"namespace_id":"original-byte-string"}`)
	file := LegacyUnsignedFileV2{SchemaVersion: 2, Kind: "WOSLegacyUnsignedContract", Protocol: "legacy_unsigned", SourceOrigin: config.Connection.ServerURL, SourceTenantID: config.Scope.TenantID, Contract: ContractDocument{SchemaVersion: 1, Kind: "WorkContract", Contract: contract}, Originals: []LegacyOriginalV2{{Path: "contract.yaml", Digest: signing.Digest(backup), Chunks: []string{base64.StdEncoding.EncodeToString(backup)}}}}
	raw, err := json.Marshal(file)
	if err != nil {
		t.Fatal(err)
	}
	raw = bytes.ReplaceAll(raw, []byte(`"tenant_id"`), []byte(`"namespace_id"`))
	raw = bytes.ReplaceAll(raw, []byte(`"source_tenant_id"`), []byte(`"source_namespace_id"`))
	decoded, legacy, err := decodeLegacyUnsignedV2(raw, profile, contract.ID)
	if err != nil || !legacy {
		t.Fatal("original unsigned snapshot unreadable", err)
	}
	if decoded.Contract.Contract.SpecDigest != digest || decoded.Contract.Contract.Scope.TenantID != profile.Binding.TenantID || decoded.Originals[0].Chunks[0] != file.Originals[0].Chunks[0] {
		t.Fatal("legacy read changed immutable material")
	}
}
