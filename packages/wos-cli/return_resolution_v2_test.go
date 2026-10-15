package woscli

import (
	"context"
	"crypto/ed25519"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

func resolutionLocalFixture(t *testing.T) (*Workspace, ProfileV2, []byte, PendingOperationV2, signing.Document) {
	return resolutionLocalFixtureAt(t, t.TempDir())
}
func resolutionLocalFixtureAt(t *testing.T, root string) (*Workspace, ProfileV2, []byte, PendingOperationV2, signing.Document) {
	t.Helper()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	project, profile, token, file, intent, _ := returnFixtureV2(t)
	intent.State, intent.Response = "sent_unknown", ""
	profile.Local.PendingOperations = []PendingOperationV2{intent}
	if e := profile.SealBinding(token); e != nil {
		t.Fatal(e)
	}
	w, e := OpenWorkspace(root)
	if e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() { w.Close() })
	for path, value := range map[string]any{".wos/project.yaml": project, profilePathV2(profile.Name): profile, contractPathV2(profile.Name, *intent.ContractID): file} {
		if e = w.CreateV2(path, value); e != nil {
			t.Fatal(e)
		}
	}
	private := ed25519.NewKeyFromSeed(make([]byte, 32))
	identity := a.SigningIdentityView{PrincipalID: profile.Binding.PrincipalID, CredentialID: profile.Binding.CredentialID, Policy: &d.CredentialPolicy{Version: 1}}
	prepared, e := prepareResolutionV2(context.Background(), w, profile, token, intent, identity, "close an explicitly rejected return", private)
	if e != nil {
		t.Fatal(e)
	}
	_, _, r, e := decodeResolutionV2(profile, token, prepared)
	if e != nil {
		t.Fatal(e)
	}
	p := signing.ReturnResolutionReceiptPayload{Binding: r.Binding, CredentialID: r.CredentialID, RequestID: r.RequestID, IdempotencyKey: r.IdempotencyKey, RequestDigest: prepared.Resolution.RequestDigest, OriginalCommandName: r.OriginalCommandName, OriginalRequestID: r.OriginalRequestID, OriginalIdempotencyKey: r.OriginalIdempotencyKey, OriginalRequestDigest: r.OriginalRequestDigest, OriginalEnvelopeDigest: r.OriginalEnvelopeDigest, ContractID: r.ContractID, Decision: "closed_without_acceptance", Reason: r.Reason, OutcomeRevision: 1, ResolvedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	p.SignerKeyID = profile.Binding.IssuerKeys[0].KeyID.String()
	env, e := signing.Sign(signing.ReturnResolutionReceipt, p, p.SignerKeyID, private)
	if e != nil {
		t.Fatal(e)
	}
	doc, e := signing.ToDocument(env)
	if e != nil {
		t.Fatal(e)
	}
	return w, profile, token, prepared, doc
}

func TestReturnResolutionPreservesFrozenIntentAndVerifiesProof(t *testing.T) {
	w, profile, token, intent, doc := resolutionLocalFixture(t)
	if e := verifyResolutionReceiptV2(profile, token, intent, doc); e != nil {
		t.Fatal(e)
	}
	current, e := w.LoadProfileV2(profile.Name)
	if e != nil {
		t.Fatal(e)
	}
	if len(current.Local.PendingOperations) != 1 || current.Local.PendingOperations[0].Payload != intent.Payload {
		t.Fatal("resolution changed the original frozen bytes")
	}
	identity := a.SigningIdentityView{PrincipalID: profile.Binding.PrincipalID, CredentialID: profile.Binding.CredentialID, Policy: &d.CredentialPolicy{Version: 1}}
	if _, e = prepareResolutionV2(context.Background(), w, current, token, current.Local.PendingOperations[0], identity, "different reason", ed25519.NewKeyFromSeed(make([]byte, 32))); e == nil {
		t.Fatal("existing resolution was replaced")
	}
	var p signing.ReturnResolutionReceiptPayload
	if e = signing.DecodeStrict(doc.Payload, &p, 32<<10); e != nil {
		t.Fatal(e)
	}
	p.OriginalRequestDigest = signing.Digest([]byte("another original"))
	env, e := signing.Sign(signing.ReturnResolutionReceipt, p, p.SignerKeyID, ed25519.NewKeyFromSeed(make([]byte, 32)))
	if e != nil {
		t.Fatal(e)
	}
	wrong, e := signing.ToDocument(env)
	if e != nil {
		t.Fatal(e)
	}
	if e = verifyResolutionReceiptV2(profile, token, intent, wrong); e == nil {
		t.Fatal("mathematically valid proof closed another return")
	}
}

func TestReturnResolutionRecoversLostResponseWithoutRepostingOriginal(t *testing.T) {
	w, profile, token, intent, doc := resolutionLocalFixture(t)
	var committed atomic.Bool
	var posts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		rw.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodPost {
			posts.Add(1)
			if !strings.HasSuffix(r.URL.Path, "/resolve_signed_return_intent") {
				t.Error("recovery sent the original closed return")
			}
			committed.Store(true)
			conn, _, e := rw.(http.Hijacker).Hijack()
			if e != nil {
				t.Error(e)
				return
			}
			conn.Close()
			return
		}
		if committed.Load() && strings.Contains(r.URL.Path, "/return_resolution") {
			env, e := doc.Envelope()
			if e != nil {
				t.Error(e)
				return
			}
			if e = json.NewEncoder(rw).Encode(a.SignedStateResult{ProtocolVersion: 2, NamespaceID: profile.Binding.NamespaceID, Resource: "return_resolution", OutcomeRevision: 1, EvaluatedAt: time.Now().UTC(), Envelope: &env}); e != nil {
				t.Error(e)
			}
			return
		}
		rw.WriteHeader(404)
		_, _ = rw.Write([]byte(`{"error":{"code":"not_found","message":"controlled absent proof"}}`))
	}))
	defer server.Close()
	client, e := sdk.New(server.URL, string(token), server.Client())
	if e != nil {
		t.Fatal(e)
	}
	_, confirmed, e := recoverReturnV2(context.Background(), w, profile, client, token, intent, false)
	if e == nil || confirmed || posts.Load() != 1 {
		t.Fatalf("lost response was falsely resolved: confirmed=%v posts=%d error=%v", confirmed, posts.Load(), e)
	}
	current, e := w.LoadProfileV2(profile.Name)
	if e != nil {
		t.Fatal(e)
	}
	if len(current.Local.PendingOperations) != 1 || current.Local.PendingOperations[0].Resolution.State != "resolution_unknown" {
		t.Fatal("uncertain resolution discarded")
	}
	_, confirmed, e = recoverReturnV2(context.Background(), w, current, client, token, current.Local.PendingOperations[0], false)
	if e != nil || !confirmed || posts.Load() != 1 {
		t.Fatalf("proof recovery failed: confirmed=%v posts=%d error=%v", confirmed, posts.Load(), e)
	}
	file, _, _, _, e := loadContractV2(w, profile, *intent.ContractID)
	if e != nil {
		t.Fatal(e)
	}
	if file.Local.Pending != nil || file.Execution == nil {
		t.Fatal("terminal cleanup removed draft or retained old signature")
	}
	current, e = w.LoadProfileV2(profile.Name)
	if e != nil {
		t.Fatal(e)
	}
	if len(current.Local.PendingOperations) != 0 {
		t.Fatal("verified terminal resolution remained pending")
	}
}

func TestReturnResolutionCrashCleanupPreservesEveryDraft(t *testing.T) {
	for _, stage := range []string{"draft_archived", "archive_persisted", "draft_unfrozen"} {
		t.Run(stage, func(t *testing.T) {
			w, profile, token, intent, doc := resolutionLocalFixture(t)
			path := contractPathV2(profile.Name, *intent.ContractID)
			file, _, _, raw, e := loadContractV2(w, profile, *intent.ContractID)
			if e != nil {
				t.Fatal(e)
			}
			file.Execution.Progress.Summary = "human edit after original signing"
			if e = w.WriteV2(path, file, signing.Digest(raw)); e != nil {
				t.Fatal(e)
			}
			if e = cleanupResolutionV2(context.Background(), w, profile, token, intent, doc, nil); e == nil {
				t.Fatal("cleanup without persisted proof permitted")
			}
			if e = updateResolutionStateV2(context.Background(), w, profile, token, intent, "abandoned_confirmed", &doc); e != nil {
				t.Fatal(e)
			}
			crash := errors.New("controlled process boundary")
			e = cleanupResolutionV2(context.Background(), w, profile, token, intent, doc, func(at string) error {
				if at == stage {
					return crash
				}
				return nil
			})
			if !errors.Is(e, crash) {
				t.Fatalf("crash stage did not fire: %v", e)
			}
			current, e := w.LoadProfileV2(profile.Name)
			if e != nil {
				t.Fatal(e)
			}
			if len(current.Local.PendingOperations) != 1 || current.Local.PendingOperations[0].Resolution.Response == "" {
				t.Fatal("crash lost terminal proof")
			}
			if e = cleanupResolutionV2(context.Background(), w, current, token, current.Local.PendingOperations[0], doc, nil); e != nil {
				t.Fatal(e)
			}
			file, _, _, _, e = loadContractV2(w, profile, *intent.ContractID)
			if e != nil {
				t.Fatal(e)
			}
			if file.Local.Pending != nil || file.Execution.Progress.Summary != "human edit after original signing" {
				t.Fatal("cleanup lost human edits")
			}
			archivePath := filepath.Join(".wos/profiles", profile.Name, "history", "return-"+intent.ID.String()+".yaml")
			archiveRaw, e := w.ReadV2(archivePath)
			if e != nil {
				t.Fatal(e)
			}
			var archive returnResolutionArchiveV2
			if e = DecodeV2Document(archiveRaw, &archive); e != nil {
				t.Fatal(e)
			}
			draft, e := w.ReadV2(archive.ContractPath)
			if e != nil {
				t.Fatal(e)
			}
			if signing.Digest(draft) != archive.ContractDigest {
				t.Fatal("exact archived draft bytes lost")
			}
			var saved ContractFileV2
			if e = DecodeV2Document(draft, &saved); e != nil {
				t.Fatal(e)
			}
			if saved.Local.Pending == nil || saved.Execution.Progress.Summary != "human edit after original signing" {
				t.Fatal("archive omitted original signature or edited draft")
			}
		})
	}
}

func TestReturnResolutionCorruptionFailsClosed(t *testing.T) {
	w, profile, token, intent, doc := resolutionLocalFixture(t)
	if e := updateResolutionStateV2(context.Background(), w, profile, token, intent, "abandoned_confirmed", &doc); e != nil {
		t.Fatal(e)
	}
	stop := errors.New("stop after durable draft archive")
	if e := cleanupResolutionV2(context.Background(), w, profile, token, intent, doc, func(at string) error {
		if at == "draft_archived" {
			return stop
		}
		return nil
	}); !errors.Is(e, stop) {
		t.Fatal(e)
	}
	draftPath := filepath.Join(".wos/profiles", profile.Name, "history", "return-"+intent.ID.String()+".contract.yaml")
	raw, e := w.ReadV2(draftPath)
	if e != nil {
		t.Fatal(e)
	}
	if e = w.writeRawV2(draftPath, []byte("corrupt: archive"), signing.Digest(raw), false); e != nil {
		t.Fatal(e)
	}
	if e = cleanupResolutionV2(context.Background(), w, profile, token, intent, doc, nil); e == nil {
		t.Fatal("corrupt archive authorized cleanup")
	}
	file, _, _, _, e := loadContractV2(w, profile, *intent.ContractID)
	if e != nil {
		t.Fatal(e)
	}
	if file.Local.Pending == nil {
		t.Fatal("corruption discarded original signature")
	}
}
