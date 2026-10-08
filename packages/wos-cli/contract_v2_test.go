package woscli

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"testing"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

func contractFixtureV2(t *testing.T) (ProfileV2, ContractFileV2) {
	t.Helper()
	project, profile, _ := profileFixtureV2(t, "executor_a")
	contract := d.MustParseID("0199ac10-0000-7000-8000-000000000008")
	work := d.MustParseID("0199ac10-0000-7000-8000-000000000009")
	private := ed25519.NewKeyFromSeed(make([]byte, 32))
	binding := signing.Binding{ProtocolVersion: 2, ServerID: profile.Binding.ServerID.String(), NamespaceID: profile.Binding.NamespaceID.String(), OutcomeID: project.Scope.DefaultOutcomeID.String(), PrincipalID: profile.Binding.PrincipalID, SignerKeyID: profile.Binding.IssuerKeys[0].KeyID.String()}
	payload := signing.SpecPayload[d.SignedWorkSpec]{Binding: binding, ContractKind: "execution", ContractID: contract.String(), WorkItemID: work.String(), Spec: d.SignedWorkSpec{Title: "Implement only the bounded change", Description: "Preserve existing behavior", ExecutionSpec: d.ExecutionSpec{Instructions: []string{"Run targeted checks"}}, PolicyRevision: 1}}
	document := func(purpose signing.PayloadType, body any) signing.Document {
		t.Helper()
		env, e := signing.Sign(purpose, body, binding.SignerKeyID, private)
		if e != nil {
			t.Fatal(e)
		}
		doc, e := signing.ToDocument(env)
		if e != nil {
			t.Fatal(e)
		}
		return doc
	}
	spec := document(signing.ContractSpec, payload)
	authority := document(signing.ContractAuthority, signing.AuthorityPayload{Binding: binding, ContractKind: "execution", ContractID: contract.String(), WorkItemID: work.String(), ExecutionID: contract.String(), FencingToken: 9007199254741003, ContractVersion: 1, LeaseVersion: 1, SpecDigest: signing.Digest(spec.Payload), PolicyRevision: 1, AllowedSigningKeyIDs: []string{profile.Signing.KeyID.String()}})
	return profile, ContractFileV2{SchemaVersion: 2, Kind: "WOSContractFile", Issued: IssuedDocumentsV2{spec, authority}, Execution: &ExecutionDraftV2{CompletionIntent: "complete", Material: a.SignedReturnMaterial{Result: d.SignedResultMaterial{ContractID: contract, WorkItemID: work, SpecDigest: signing.Digest(spec.Payload), Summary: "Targeted result"}}}, Local: ContractLocalV2{SchemaVersion: 1, Profile: profile.Name, WorkItemVersion: 9007199254741005}}
}

func TestV2SingleContractYAMLPreservesProofCountersAndDraftIsolation(t *testing.T) {
	profile, file := contractFixtureV2(t)
	before, e := file.DraftDigest()
	if e != nil {
		t.Fatal(e)
	}
	raw, e := EncodeV2Document(file)
	if e != nil {
		t.Fatal(e)
	}
	if !bytes.Contains(raw, []byte("work_item_id:")) || !bytes.Contains(raw, []byte(`"9007199254741005"`)) {
		t.Fatal("human names or exact counter lost")
	}
	var restored ContractFileV2
	if e = DecodeV2Document(raw, &restored); e != nil {
		t.Fatal(e)
	}
	view, e := restored.VerifyIssued(profile)
	if e != nil {
		t.Fatal(e)
	}
	if view.Authority.FencingToken != 9007199254741003 {
		t.Fatal("fencing token rounded")
	}
	after, e := restored.DraftDigest()
	if e != nil || before != after {
		t.Fatal("editable material changed in YAML round trip")
	}
	restored.Execution.Progress.Summary = "Work in progress"
	if _, e = restored.VerifyIssued(profile); e != nil {
		t.Fatal("draft edit changed issuer proof")
	}
	var spec signing.SpecPayload[d.SignedWorkSpec]
	if e = json.Unmarshal(restored.Issued.Specification.Payload, &spec); e != nil {
		t.Fatal(e)
	}
	spec.Spec.Title = "Forged instructions"
	restored.Issued.Specification.Payload, _ = json.Marshal(spec)
	if _, e = restored.VerifyIssued(profile); e == nil {
		t.Fatal("modified issued spec trusted")
	}
	if e = DecodeV2Document(bytes.Replace(raw, []byte("summary:"), []byte("Summary:"), 1), &restored); e == nil {
		t.Fatal("editable case alias accepted")
	}
}

type failingKeyringV2 struct{}

func (failingKeyringV2) Get(string, string) (string, error) {
	return "", errors.New("secret-value-from-backend")
}
func (failingKeyringV2) Set(string, string, string) error {
	return errors.New("secret-value-from-backend")
}
func (failingKeyringV2) Delete(string, string) error { return errors.New("secret-value-from-backend") }
func TestProfileSecretErrorsRedactedAndNoFallback(t *testing.T) {
	for _, backend := range []KeyringBackend{nil, failingKeyringV2{}} {
		resolver := SecretResolver{WorkspaceRoot: t.TempDir(), Keyring: backend}
		_, e := resolver.Read("keyring:wos/executor_a/api")
		if e == nil || bytes.Contains([]byte(e.Error()), []byte("secret-value-from-backend")) {
			t.Fatal("backend failure leaked or fell back")
		}
		if e = resolver.Store("keyring:wos/executor_a/signing", []byte("private-value")); e == nil || bytes.Contains([]byte(e.Error()), []byte("secret-value-from-backend")) {
			t.Fatal("provisioning failure leaked or fell back")
		}
		if e = resolver.Store("env:WOS_PRIVATE", []byte("private-value")); e == nil {
			t.Fatal("implicit env provisioning")
		}
	}
}
