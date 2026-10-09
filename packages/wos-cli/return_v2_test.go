package woscli

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func returnFixtureV2(t *testing.T) (ProjectV2, ProfileV2, []byte, ContractFileV2, PendingOperationV2, signing.Document) {
	t.Helper()
	project, profile, token := profileFixtureV2(t, "executor_a")
	_, file := contractFixtureV2(t)
	file.Execution.CompletionIntent = "auto"
	view, e := file.VerifyIssued(profile)
	if e != nil {
		t.Fatal(e)
	}
	g := view.Authority
	b := signing.RequestBinding{Binding: signing.Binding{ProtocolVersion: 2, ServerID: g.ServerID, NamespaceID: g.NamespaceID, OutcomeID: g.OutcomeID, PrincipalID: g.PrincipalID, SignerKeyID: profile.Signing.KeyID.String()}, RequestID: "0199ac10-0000-7000-8000-000000000011", IdempotencyKey: "frozen-return-fixture", OperationKind: "work_return", ContractID: g.ContractID, WorkItemID: g.WorkItemID, ExecutionID: g.ExecutionID, FencingToken: g.FencingToken, SpecDigest: g.SpecDigest, AuthorityDigest: signing.Digest(file.Issued.Authority.Payload), ExpectedContractVersion: g.ContractVersion, ExpectedLeaseVersion: g.LeaseVersion, ExpectedWorkItemVersion: file.Local.WorkItemVersion, PolicyRevision: 1}
	private := ed25519.NewKeyFromSeed(make([]byte, 32))
	env, e := signing.Sign(signing.WorkReturn, signing.WorkReturnPayload[a.SignedReturnMaterial]{RequestBinding: b, CompletionIntent: "auto", Material: file.Execution.Material}, profile.Signing.KeyID.String(), private)
	if e != nil {
		t.Fatal(e)
	}
	draft, e := file.DraftDigest()
	if e != nil {
		t.Fatal(e)
	}
	raw, _ := base64.StdEncoding.Strict().DecodeString(env.Payload)
	frozen := FrozenReturnV2{State: "prepared_signed", Envelope: env, DraftDigest: draft, RequestDigest: signing.Digest(raw), CredentialID: profile.Binding.CredentialID, SignerPublicKey: base64.StdEncoding.EncodeToString(private.Public().(ed25519.PublicKey))}
	if e = sealFrozenV2(profile, token, &frozen); e != nil {
		t.Fatal(e)
	}
	file.Local.Pending = &frozen
	intent, e := returnIntentV2(profile, frozen, b, d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "executor"})
	if e != nil {
		t.Fatal(e)
	}
	receipt := signing.ReceiptPayload{Binding: signing.Binding{ProtocolVersion: 2, ServerID: b.ServerID, NamespaceID: b.NamespaceID, OutcomeID: b.OutcomeID, PrincipalID: b.PrincipalID, SignerKeyID: profile.Binding.IssuerKeys[0].KeyID.String()}, RequestID: b.RequestID, IdempotencyKey: b.IdempotencyKey, RequestDigest: frozen.RequestDigest, ContractID: b.ContractID, WorkItemID: b.WorkItemID, Accepted: true, LocalObligationClosed: true, ContractStatus: "delivered", Disposition: "delivered_for_review"}
	envelope, e := signing.Sign(signing.AcceptanceReceipt, receipt, receipt.SignerKeyID, private)
	if e != nil {
		t.Fatal(e)
	}
	doc, e := signing.ToDocument(envelope)
	if e != nil {
		t.Fatal(e)
	}
	response, e := json.Marshal(doc)
	if e != nil {
		t.Fatal(e)
	}
	intent.State = "accepted_confirmed"
	intent.Response = base64.StdEncoding.EncodeToString(response)
	profile.Local.PendingOperations = []PendingOperationV2{intent}
	if e = profile.SealBinding(token); e != nil {
		t.Fatal(e)
	}
	return project, profile, token, file, intent, doc
}
func TestFrozenReturnRejectsCopiedCredentialAndTamperedEnvelope(t *testing.T) {
	_, profile, token, file, intent, receipt := returnFixtureV2(t)
	frozen, b, e := decodeReturnIntentV2(profile, token, intent)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = verifyReturnReceiptV2(profile, frozen, b, receipt); e != nil {
		t.Fatal(e)
	}
	other := profile
	other.Binding.CredentialID = d.ID("0199ac10-0000-7000-8000-000000000012")
	if e = other.SealBinding(token); e != nil {
		t.Fatal(e)
	}
	if _, e = verifyFrozenV2(other, token, *file.Local.Pending); e == nil {
		t.Fatal("copied return became authority of another CID")
	}
	changed := *file.Local.Pending
	changed.Envelope.Signatures = append([]signing.Signature(nil), changed.Envelope.Signatures...)
	changed.Envelope.Signatures[0].Sig = base64.StdEncoding.EncodeToString(make([]byte, 64))
	if e = sealFrozenV2(profile, token, &changed); e != nil {
		t.Fatal(e)
	}
	if _, e = verifyFrozenV2(profile, token, changed); e == nil {
		t.Fatal("local MAC substituted for signature verification")
	}
	wrong := b
	wrong.RequestID = "0199ac10-0000-7000-8000-000000000013"
	if _, e = verifyReturnReceiptV2(profile, frozen, wrong, receipt); e == nil {
		t.Fatal("another request's receipt accepted")
	}
	open := receipt
	var body signing.ReceiptPayload
	if e = signing.DecodeStrict(open.Payload, &body, signing.MaxPayloadBytes); e != nil {
		t.Fatal(e)
	}
	body.LocalObligationClosed = false
	private := ed25519.NewKeyFromSeed(make([]byte, 32))
	env, e := signing.Sign(signing.AcceptanceReceipt, body, body.SignerKeyID, private)
	if e != nil {
		t.Fatal(e)
	}
	open, e = signing.ToDocument(env)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = verifyReturnReceiptV2(profile, frozen, b, open); e == nil {
		t.Fatal("open obligation allowed deletion")
	}
}
func TestReceiptCleanupPreservesUnconfirmedEditsAndUnrelatedFiles(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	project, profile, token, file, intent, receipt := returnFixtureV2(t)
	w, e := OpenWorkspace(t.TempDir())
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	must(w.CreateV2(".wos/project.yaml", project))
	must(w.CreateV2(profilePathV2(profile.Name), profile))
	path := contractPathV2(profile.Name, *intent.ContractID)
	original := file.Execution.Progress
	file.Execution.Progress.Summary = "unconfirmed human change"
	must(w.CreateV2(path, file))
	must(w.CreateV2("deliverable.yaml", map[string]string{"summary": "user artifact"}))
	if e = cleanupReturnV2(context.Background(), w, profile, token, intent, receipt); e == nil {
		t.Fatal("edited draft was deleted")
	}
	raw, e := w.ReadV2(path)
	must(e)
	must(DecodeV2Document(raw, &file))
	if file.Execution.Progress.Summary != "unconfirmed human change" || file.Local.AcceptanceReceipt == nil {
		t.Fatal("edit or signed receipt lost")
	}
	file.Execution.Progress = original
	must(w.WriteV2(path, file, signing.Digest(raw)))
	must(cleanupReturnV2(context.Background(), w, profile, token, intent, receipt))
	if _, e = w.ReadV2(path); !errors.Is(e, os.ErrNotExist) {
		t.Fatal("closed unchanged contract remains")
	}
	if _, e = w.ReadV2("deliverable.yaml"); e != nil {
		t.Fatal("unrelated artifact was removed")
	}
	p, e := w.LoadProfileV2(profile.Name)
	must(e)
	if len(p.Local.PendingOperations) != 0 {
		t.Fatal("confirmed pending was not removed")
	}
}
func TestV2ReturnCleanupChild(t *testing.T) {
	root := os.Getenv("WOS_RETURN_CLEANUP_FIXTURE")
	if root == "" {
		return
	}
	_, profile, token, _, intent, receipt := returnFixtureV2(t)
	w, e := OpenWorkspace(root)
	if e != nil {
		t.Fatal(e)
	}
	defer w.Close()
	point := os.Getenv("WOS_RETURN_CLEANUP_POINT")
	e = cleanupReturnWithHookV2(context.Background(), w, profile, token, intent, receipt, func(stage string) error {
		if stage == point {
			os.Exit(23)
		}
		return nil
	})
	if e != nil {
		t.Fatal(e)
	}
}
func TestReceiptCleanupRecoversNativeProcessDeath(t *testing.T) {
	for _, point := range []string{"receipt_persisted", "unlinked"} {
		t.Run(point, func(t *testing.T) {
			t.Setenv("XDG_CACHE_HOME", t.TempDir())
			t.Setenv("LOCALAPPDATA", t.TempDir())
			project, profile, token, file, intent, receipt := returnFixtureV2(t)
			root := t.TempDir()
			w, e := OpenWorkspace(root)
			if e != nil {
				t.Fatal(e)
			}
			defer w.Close()
			must := func(e error) {
				t.Helper()
				if e != nil {
					t.Fatal(e)
				}
			}
			must(w.CreateV2(".wos/project.yaml", project))
			must(w.CreateV2(profilePathV2(profile.Name), profile))
			path := contractPathV2(profile.Name, *intent.ContractID)
			must(w.CreateV2(path, file))
			child := exec.Command(os.Args[0], "-test.run=^TestV2ReturnCleanupChild$")
			child.Env = append(os.Environ(), "WOS_RETURN_CLEANUP_FIXTURE="+root, "WOS_RETURN_CLEANUP_POINT="+point)
			output, e := child.CombinedOutput()
			var exit *exec.ExitError
			if !errors.As(e, &exit) || exit.ExitCode() != 23 {
				t.Fatalf("child did not die at %s: %v %s", point, e, output)
			}
			must(cleanupReturnV2(context.Background(), w, profile, token, intent, receipt))
			if _, e = os.Stat(filepath.Join(root, path)); !os.IsNotExist(e) {
				t.Fatal("cleanup recovery retained closed file")
			}
			p, e := w.LoadProfileV2(profile.Name)
			must(e)
			if len(p.Local.PendingOperations) != 0 {
				t.Fatal("post-unlink recovery lost original receipt")
			}
		})
	}
}
