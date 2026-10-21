package sqlite

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
	"path/filepath"
	"testing"
	"time"
)

type resolutionFixture struct {
	store interface {
		ports.SecurityStore
		ports.TransactionManager
	}
	service       *a.Service
	ctx           context.Context
	cc            d.CommandContext
	request       signing.WorkReturnPayload[a.SignedReturnMaterial]
	envelope      signing.Envelope
	private       ed25519.PrivateKey
	credential    d.ID
	server        d.ServerIdentity
	issuerPrivate ed25519.PrivateKey
}

func newResolutionFixture(t *testing.T, backend string) resolutionFixture {
	return newResolutionFixtureTTL(t, backend, 300)
}
func newResolutionFixtureTTL(t *testing.T, backend string, ttl int) resolutionFixture {
	t.Helper()
	floor := d.AcceptanceDirect
	hasCriteria := false
	var store interface {
		ports.SecurityStore
		ports.TransactionManager
	}
	if backend == "memory" {
		store = memory.New()
	} else {
		store = openTestStore(t, filepath.Join(t.TempDir(), "signed-acquisition.db"))
	}
	ctx := context.Background()
	now := time.Now().UTC()
	clock := sqliteFixedClock{now}
	ids := &sqliteSequenceIDs{prefix: "0199a556", next: 1}
	sec := a.SecurityService{Store: store, Clock: clock, IDs: ids}
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	ns := testID("0199a555-0000-7000-8000-000000000001")
	scope := d.Scope{NamespaceID: ns, OutcomeID: testID("0199a555-0000-7000-8000-000000000002")}
	must(sec.Bootstrap(ctx, ports.Namespace{ID: ns, Name: "signed-acquisition"}, "admin", "signed-acquisition-bootstrap-token-at-least-32"))
	admin, err := sec.Authenticate(ctx, "signed-acquisition-bootstrap-token-at-least-32")
	must(err)
	adminCtx := a.WithIdentity(ctx, admin)
	must(sec.SetGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "executor", Permissions: []ports.Permission{ports.PermissionStateRead, ports.PermissionWorkContractAcquire, ports.PermissionWorkWrite, ports.PermissionWorkContractReturn, ports.PermissionWorkCompleteDirect, ports.PermissionAssessmentWrite, ports.PermissionConclusionWrite, ports.PermissionRecordsWrite}}))
	must(sec.SetGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "another-executor", Permissions: []ports.Permission{ports.PermissionStateRead}}))
	credential, token, err := sec.IssueCredential(adminCtx, ns, "executor", d.ActorRef{Kind: d.ActorKindAgent, Provider: "test", ID: "executor"}, now.Add(time.Hour))
	must(err)
	identity, err := sec.Authenticate(ctx, token)
	must(err)
	agentCtx := a.WithIdentity(ctx, identity)
	pub, private, err := ed25519.GenerateKey(rand.Reader)
	must(err)
	fp, err := signing.Fingerprint(pub)
	must(err)
	server := d.ServerIdentity{ID: testID("0199a555-0000-7000-8000-000000000003"), IssuerKeyID: testID("0199a555-0000-7000-8000-000000000004"), PublicKey: base64.StdEncoding.EncodeToString(pub), Fingerprint: fp, CreatedAt: now}
	service, err := a.NewService(store, clock, ids)
	must(err)
	must(service.ConfigureSignedIssuer(ctx, acquisitionIssuer{server, private}, d.AcceptanceDirect))
	agentPub, agentPrivate, err := ed25519.GenerateKey(rand.Reader)
	must(err)
	agentFP, err := signing.Fingerprint(agentPub)
	must(err)
	agentKey := testID("0199a555-0000-7000-8000-000000000005")
	uow, err := store.Begin(ctx)
	must(err)
	registry := uow.(ports.SigningIdentityUnitOfWork).SigningIdentity()
	must(registry.SaveKey(ctx, d.SigningKey{ID: agentKey, NamespaceID: ns, PrincipalID: "executor", Purpose: "agent", Algorithm: "Ed25519", PublicKey: base64.StdEncoding.EncodeToString(agentPub), Fingerprint: agentFP, Version: 1, Status: "active", CreatedAt: now, UpdatedAt: now}, 0))
	foreignPub, _, err := ed25519.GenerateKey(rand.Reader)
	must(err)
	foreignFP, err := signing.Fingerprint(foreignPub)
	must(err)
	foreignKey := testID("0199a555-0000-7000-8000-000000000099")
	must(registry.SaveKey(ctx, d.SigningKey{ID: foreignKey, NamespaceID: ns, PrincipalID: "another-executor", Purpose: "agent", Algorithm: "Ed25519", PublicKey: base64.StdEncoding.EncodeToString(foreignPub), Fingerprint: foreignFP, Version: 1, Status: "active", CreatedAt: now, UpdatedAt: now}, 0))
	policy := d.CredentialPolicy{NamespaceID: ns, CredentialID: credential.ID, PrincipalID: "executor", Version: 1, PermittedOperations: []string{string(ports.PermissionStateRead), string(ports.PermissionWorkContractAcquire), string(ports.PermissionWorkWrite), string(ports.PermissionWorkContractReturn), string(ports.PermissionWorkCompleteDirect), string(ports.PermissionAssessmentWrite), string(ports.PermissionConclusionWrite), string(ports.PermissionRecordsWrite)}, AcceptanceFloor: floor, AllowedSigningKeyIDs: []d.ID{agentKey}, MaxActiveWorkContracts: 10, MaxActiveReviewContracts: 1, UpdatedAt: now}
	must(registry.SaveCredentialPolicy(ctx, policy, 0))
	outcome, err := d.NewOutcome(scope.OutcomeID, ns, "signed workload", "", "verified", d.PriorityNormal, now)
	must(err)
	outcome.Lifecycle = d.OutcomeLifecycleActive
	must(uow.Outcomes().Insert(ctx, outcome))
	works := []d.WorkItem{}
	for i := 0; i < 4; i++ {
		w, e := d.NewWorkItem(testID(fmt.Sprintf("0199a555-0000-7000-8000-%012x", 20+i)), scope, "precise signed task", "", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
		must(e)
		w.ContractsEnabled = true
		criterion, e := d.NewSuccessCriterion(testID(fmt.Sprintf("0199a555-0000-7000-8000-%012x", 100+i)), w.Ref(), "verified result", "", true, d.VerificationModeEvidenceReview)
		must(e)
		if hasCriteria {
			must(w.Criteria.Add(criterion))
		}
		must(insertWorkItemFixture(ctx, uow.Objectives(), uow.WorkItems(), &w))
		works = append(works, w)
	}
	protocol := uow.(ports.WorkProtocolUnitOfWork).WorkProtocol()
	state, err := protocol.Lock(ctx, ns, true)
	must(err)
	state.Phase = d.WorkProtocolSigned
	state.WriterEpoch = 2
	state.WritersDrained = true
	state.Version++
	state.UpdatedAt = now
	state.UpdatedBy = "operator"
	state.Reason = "test-only operator activation; public cutover remains separately gated"
	must(protocol.Save(ctx, state, 1))
	must(uow.Commit())

	w := works[0]
	cid, e := ids.NewID()
	must(e)
	cc := d.CommandContext{PrincipalID: identity.PrincipalID, Actor: identity.Actor, CommandID: cid, IdempotencyKey: "atomic-return-acquisition"}
	acquired, e := service.AcquireSignedWorkContract(agentCtx, cc, a.AcquireSignedWorkContractCommand{Scope: scope, WorkItemID: w.ID, ExpectedWorkItemVersion: w.Version, SignerKeyID: agentKey, TTLSeconds: ttl})
	must(e)
	c := acquired.Value.Contract
	work := acquired.Value.WorkItem
	rid, e := ids.NewID()
	must(e)
	request := signing.WorkReturnPayload[a.SignedReturnMaterial]{RequestBinding: signing.RequestBinding{Binding: signing.Binding{ProtocolVersion: 2, ServerID: server.ID.String(), NamespaceID: ns.String(), OutcomeID: scope.OutcomeID.String(), PrincipalID: identity.PrincipalID, SignerKeyID: agentKey.String()}, OperationKind: "work_return", RequestID: rid.String(), IdempotencyKey: "atomic-signed-return", ContractID: c.ID.String(), WorkItemID: work.ID.String(), ExecutionID: c.ExecutionID.String(), FencingToken: signing.Decimal(c.FencingToken), SpecDigest: c.SignedBinding.SpecificationDigest, AuthorityDigest: signing.Digest(acquired.Value.IssuedAuthority.Payload), ExpectedContractVersion: signing.Decimal(c.Version), ExpectedLeaseVersion: signing.Decimal(c.LeaseVersion), ExpectedWorkItemVersion: signing.Decimal(work.Version), PolicyRevision: signing.Decimal(policy.Version)}, CompletionIntent: "auto", Material: a.SignedReturnMaterial{Result: d.NewSignedResultMaterial(d.WorkResultMaterial{ContractID: c.ID, WorkItemID: work.ID, SpecDigest: c.SignedBinding.SpecificationDigest, Summary: "bounded implementation verified"}), Reason: "verified direct result"}}
	criterion := d.SuccessCriterion{}
	if hasCriteria {
		criterion = work.Criteria.Items[0]
	}
	request.Material.Artifacts = []a.SyncArtifactInput{{LocalKey: "build", Artifact: a.RegisterArtifactCommand{Scope: scope, ArtifactType: "source", Name: "tested commit", URI: "git:commit-tested", SourceVersion: "commit-tested"}}}
	request.Material.Evidence = []a.SyncEvidenceInput{{LocalKey: "test", ArtifactLocalKey: "build", Evidence: a.RegisterEvidenceCommand{Scope: scope, EvidenceType: d.EvidenceTypeTestResult, Description: "assertion log", SourceRef: d.SourceReference{Provider: "test", ID: "test-run"}, CapturedAt: now, SourceVersion: "commit-tested"}}}
	request.Material.ArtifactLocalKeys = []string{"build"}
	request.Material.EvidenceLocalKeys = []string{"test"}
	if hasCriteria {
		request.Material.EvidenceLocalKeys = nil // Criterion references attach their selected evidence without duplication.
		request.Material.CriterionLocalEvidence = []a.SignedLocalCriterionEvidence{{CriterionID: criterion.ID, CriterionRevision: signing.Decimal(criterion.Revision), EvidenceLocalKeys: []string{"test"}}}
	}
	if floor == d.AcceptanceDirect && hasCriteria {
		request.Material.Assessments = []a.SignedAssessmentInput{{CriterionID: criterion.ID, CriterionRevision: signing.Decimal(criterion.Revision), Result: d.AssessmentResultMet, Rationale: "the exact submitted test passed", EvidenceLocalKeys: []string{"test"}}}
	}
	envelope, e := signing.Sign(signing.WorkReturn, request, agentKey.String(), agentPrivate)
	must(e)
	returnCC := cc
	returnCC.CommandID = rid
	returnCC.IdempotencyKey = request.IdempotencyKey
	return resolutionFixture{store, service, agentCtx, returnCC, request, envelope, agentPrivate, credential.ID, server, private}
}
func TestSignedReturnResolutionRepository(t *testing.T) {
	for _, backend := range []string{"memory", "sql"} {
		t.Run(backend, func(t *testing.T) {
			f := newResolutionFixture(t, backend)
			now := time.Now().UTC()
			r := d.SignedReturnIntentResolution{Scope: d.Scope{NamespaceID: d.ID(f.request.NamespaceID), OutcomeID: d.ID(f.request.OutcomeID)}, PrincipalID: f.cc.PrincipalID, CredentialID: f.credential, OriginalCommandName: "ReturnSignedWork", OriginalIdempotencyKey: f.request.IdempotencyKey, OriginalRequestID: d.ID(f.request.RequestID), OriginalRequestDigest: signing.Digest([]byte("original")), OriginalEnvelopeDigest: signing.Digest([]byte("envelope")), ContractID: d.ID(f.request.ContractID), ResolutionRequestID: testID("0199a555-0000-7000-8000-000000000088"), ResolutionIdempotencyKey: "separate-resolution", ResolutionDigest: signing.Digest([]byte("resolution")), IssuerKeyID: f.server.IssuerKeyID, Decision: "closed_without_acceptance", Reason: "controlled repository contract", OutcomeRevision: 1, Receipt: json.RawMessage(`{"proof":"repository fixture"}`), RecordedAt: now}
			u, e := f.store.Begin(f.ctx)
			if e != nil {
				t.Fatal(e)
			}
			repo := u.(ports.SignedReturnResolutionUnitOfWork).SignedReturnResolutions()
			if e = repo.InsertTerminal(f.ctx, r); e != nil {
				u.Rollback()
				t.Fatal(e)
			}
			if e = u.Rollback(); e != nil {
				t.Fatal(e)
			}
			u, e = f.store.Begin(f.ctx)
			if e != nil {
				t.Fatal(e)
			}
			repo = u.(ports.SignedReturnResolutionUnitOfWork).SignedReturnResolutions()
			if _, e = repo.GetByOriginalIntent(f.ctx, r.Scope.NamespaceID, r.PrincipalID, r.CredentialID, r.OriginalCommandName, r.OriginalIdempotencyKey); e == nil {
				u.Rollback()
				t.Fatal("rollback retained terminal decision")
			}
			if e = repo.InsertTerminal(f.ctx, r); e != nil {
				u.Rollback()
				t.Fatal(e)
			}
			if e = u.Commit(); e != nil {
				t.Fatal(e)
			}
			u, e = f.store.Begin(f.ctx)
			if e != nil {
				t.Fatal(e)
			}
			defer u.Rollback()
			repo = u.(ports.SignedReturnResolutionUnitOfWork).SignedReturnResolutions()
			got, e := repo.GetByOriginalIntent(f.ctx, r.Scope.NamespaceID, r.PrincipalID, r.CredentialID, r.OriginalCommandName, r.OriginalIdempotencyKey)
			if e != nil || string(got.Receipt) != string(r.Receipt) {
				t.Fatalf("durable proof differs: %v", e)
			}
			if _, e = repo.GetByOriginalIntent(f.ctx, r.Scope.NamespaceID, "another-principal", r.CredentialID, r.OriginalCommandName, r.OriginalIdempotencyKey); e == nil {
				t.Fatal("foreign Principal observed a resolution")
			}
			r.Reason = "replacement is forbidden"
			if e = repo.InsertTerminal(f.ctx, r); e == nil {
				t.Fatal("terminal decision overwritten")
			}
		})
	}
}

func resolutionIntent(t *testing.T, f resolutionFixture, reason string, suffix int) (d.CommandContext, a.ResolveSignedReturnIntentCommand) {
	t.Helper()
	raw, e := base64.StdEncoding.Strict().DecodeString(f.envelope.Payload)
	if e != nil {
		t.Fatal(e)
	}
	envBytes, e := signing.Canonical(f.envelope)
	if e != nil {
		t.Fatal(e)
	}
	p := signing.ReturnResolutionPayload{Binding: f.request.Binding, OperationKind: "resolve_return_intent", RequestID: fmt.Sprintf("0199a555-0000-7000-8000-%012x", 500+suffix), IdempotencyKey: fmt.Sprintf("resolve-original-%d", suffix), CredentialID: f.credential.String(), OriginalCommandName: "ReturnSignedWork", OriginalRequestID: f.request.RequestID, OriginalIdempotencyKey: f.request.IdempotencyKey, OriginalRequestDigest: signing.Digest(raw), OriginalEnvelopeDigest: signing.Digest(envBytes), ContractID: f.request.ContractID, PolicyRevision: f.request.PolicyRevision, Reason: reason}
	envelope, e := signing.Sign(signing.ReturnResolution, p, f.request.SignerKeyID, f.private)
	if e != nil {
		t.Fatal(e)
	}
	cc := f.cc
	cc.CommandID = d.ID(p.RequestID)
	cc.IdempotencyKey = p.IdempotencyKey
	return cc, a.ResolveSignedReturnIntentCommand{Scope: d.Scope{NamespaceID: d.ID(p.NamespaceID), OutcomeID: d.ID(p.OutcomeID)}, OriginalEnvelope: f.envelope, Envelope: envelope}
}
func TestSignedReturnTerminalResolution(t *testing.T) {
	for _, backend := range []string{"memory", "sql"} {
		t.Run(backend+"/closed-first", func(t *testing.T) {
			f := newResolutionFixture(t, backend)
			// Stable authority, invalid material: the frozen return is rejected without a commit.
			f.request.CompletionIntent = "invalid-completion"
			var e error
			f.envelope, e = signing.Sign(signing.WorkReturn, f.request, f.request.SignerKeyID, f.private)
			if e != nil {
				t.Fatal(e)
			}
			if _, e = f.service.ReturnSignedWork(f.ctx, f.cc, a.ReturnSignedWorkCommand{Envelope: f.envelope}); e == nil {
				t.Fatal("invalid material accepted")
			}
			cc, cmd := resolutionIntent(t, f, "discard invalid completion and prepare a corrected intention", 1)
			before, e := f.service.GetOutcomeState(f.ctx, cmd.Scope)
			if e != nil {
				t.Fatal(e)
			}
			result, e := f.service.ResolveSignedReturnIntent(f.ctx, cc, cmd)
			if e != nil {
				t.Fatal(e)
			}
			if result.Value.Decision != "closed_without_acceptance" || result.Value.ResolutionReceipt == nil || result.Value.AcceptanceReceipt != nil {
				t.Fatal("not a terminal negative proof")
			}
			document := *result.Value.ResolutionReceipt
			public, e := f.server.PublicBytes()
			if e != nil {
				t.Fatal(e)
			}
			env, e := document.Envelope()
			if e != nil {
				t.Fatal(e)
			}
			if _, e = signing.Verify(env, signing.ReturnResolutionReceipt, f.server.IssuerKeyID.String(), public); e != nil {
				t.Fatal(e)
			}
			replay, e := f.service.ResolveSignedReturnIntent(f.ctx, cc, cmd)
			if e != nil || !replay.IdempotentReplay || replay.Value.ResolutionReceipt == nil || string(replay.Value.ResolutionReceipt.Payload) != string(document.Payload) {
				t.Fatalf("resolution replay differs: %v", e)
			}
			cc2, cmd2 := resolutionIntent(t, f, "competing reason cannot overwrite", 2)
			if _, e = f.service.ResolveSignedReturnIntent(f.ctx, cc2, cmd2); e == nil {
				t.Fatal("second resolution replaced original")
			}
			if _, e = f.service.ReturnSignedWork(f.ctx, f.cc, a.ReturnSignedWorkCommand{Envelope: f.envelope}); e == nil {
				t.Fatal("closed return accepted")
			} else if code, _ := d.ErrorCodeOf(e); code != d.ErrorCodeSignedIntentClosed {
				t.Fatalf("wrong closed error: %v", e)
			}
			lookup, e := f.service.ReadSignedState(f.ctx, a.SignedStateQuery{Scope: cmd.Scope, Resource: "return_resolution", ContractKind: "execution", IdempotencyKey: f.request.IdempotencyKey})
			if e != nil || lookup.Envelope == nil {
				t.Fatalf("resolution lookup unavailable: %v", e)
			}
			after, e := f.service.GetOutcomeState(f.ctx, cmd.Scope)
			if e != nil {
				t.Fatal(e)
			}
			b, _ := json.Marshal(before)
			ar, _ := json.Marshal(after)
			if string(b) != string(ar) {
				t.Fatal("negative decision mutated Task, lease, assessment or revision")
			}
			// A separate explicit corrected return retains the same original authority/CAS.
			f.request.CompletionIntent = "auto"
			f.request.RequestID = "0199a555-0000-7000-8000-000000000777"
			f.request.IdempotencyKey = "corrected-new-intention"
			fresh, e := signing.Sign(signing.WorkReturn, f.request, f.request.SignerKeyID, f.private)
			if e != nil {
				t.Fatal(e)
			}
			newCC := f.cc
			newCC.CommandID = d.ID(f.request.RequestID)
			newCC.IdempotencyKey = f.request.IdempotencyKey
			if _, e = f.service.ReturnSignedWork(f.ctx, newCC, a.ReturnSignedWorkCommand{Envelope: fresh}); e != nil {
				t.Fatalf("explicit valid new intention failed: %v", e)
			}
		})
		t.Run(backend+"/accepted-first", func(t *testing.T) {
			f := newResolutionFixture(t, backend)
			accepted, e := f.service.ReturnSignedWork(f.ctx, f.cc, a.ReturnSignedWorkCommand{Envelope: f.envelope})
			if e != nil {
				t.Fatal(e)
			}
			cc, cmd := resolutionIntent(t, f, "response lost; server must preserve prior acceptance", 1)
			result, e := f.service.ResolveSignedReturnIntent(f.ctx, cc, cmd)
			if e != nil {
				t.Fatal(e)
			}
			if result.Value.Decision != "accepted" || result.Value.ResolutionReceipt != nil || result.Value.AcceptanceReceipt == nil || string(result.Value.AcceptanceReceipt.Payload) != string(accepted.Value.Receipt.Payload) {
				t.Fatal("accepted intention was abandoned or receipt replaced")
			}
			u, e := f.store.Begin(f.ctx)
			if e != nil {
				t.Fatal(e)
			}
			defer u.Rollback()
			if _, e = u.(ports.SignedReturnResolutionUnitOfWork).SignedReturnResolutions().GetByOriginalIntent(f.ctx, cmd.Scope.NamespaceID, f.cc.PrincipalID, f.credential, "ReturnSignedWork", f.request.IdempotencyKey); e == nil {
				t.Fatal("acceptance created a negative decision")
			}
		})
	}
}

func TestSignedReturnResolutionRaces(t *testing.T) {
	for _, backend := range []string{"memory", "sql"} {
		for _, mode := range []string{"return-vs-resolution", "same-resolution", "different-resolution"} {
			t.Run(backend+"/"+mode, func(t *testing.T) {
				f := newResolutionFixture(t, backend)
				cc, cmd := resolutionIntent(t, f, "explicit resolution race", 1)
				start := make(chan struct{})
				errors := make(chan error, 2)
				run := func(fn func() error) {
					<-start
					var e error
					for i := 0; i < 5; i++ {
						e = fn()
						code, _ := d.ErrorCodeOf(e)
						if code != d.ErrorCodeTransactionConflict {
							break
						}
					}
					errors <- e
				}
				go run(func() error { _, e := f.service.ResolveSignedReturnIntent(f.ctx, cc, cmd); return e })
				if mode == "return-vs-resolution" {
					go run(func() error {
						_, e := f.service.ReturnSignedWork(f.ctx, f.cc, a.ReturnSignedWorkCommand{Envelope: f.envelope})
						return e
					})
				} else {
					otherCC, other := cc, cmd
					if mode == "different-resolution" {
						otherCC, other = resolutionIntent(t, f, "competing terminal intention", 2)
					}
					go run(func() error { _, e := f.service.ResolveSignedReturnIntent(f.ctx, otherCC, other); return e })
				}
				close(start)
				first, second := <-errors, <-errors
				if first != nil && second != nil {
					t.Fatalf("neither contender committed: %v / %v", first, second)
				}
				for _, e := range []error{first, second} {
					if e != nil {
						code, _ := d.ErrorCodeOf(e)
						if code != d.ErrorCodeSignedIntentClosed && code != d.ErrorCodeIdempotencyConflict {
							t.Fatalf("unexpected race result: %v", e)
						}
					}
				}
				if mode == "same-resolution" && (first != nil || second != nil) {
					t.Fatalf("identical decision not replayed: %v / %v", first, second)
				}
				u, e := f.store.Begin(f.ctx)
				if e != nil {
					t.Fatal(e)
				}
				defer u.Rollback()
				_, accepted := u.(ports.SignedContractUnitOfWork).SignedContracts().Acceptance(f.ctx, cmd.Scope.NamespaceID, f.cc.PrincipalID, f.request.IdempotencyKey)
				_, closed := u.(ports.SignedReturnResolutionUnitOfWork).SignedReturnResolutions().GetByOriginalIntent(f.ctx, cmd.Scope.NamespaceID, f.cc.PrincipalID, f.credential, "ReturnSignedWork", f.request.IdempotencyKey)
				if (accepted == nil) == (closed == nil) {
					t.Fatal("acceptance and terminal decision must be exclusive")
				}
			})
		}
	}
}

func TestSignedReturnResolutionRevalidatesAuthority(t *testing.T) {
	for _, backend := range []string{"memory", "sql"} {
		for _, probe := range []string{"credential", "signer", "digest", "scope", "issuer", "revoked", "retired-key", "revoked-contract"} {
			t.Run(backend+"/"+probe, func(t *testing.T) {
				f := newResolutionFixture(t, backend)
				cc, cmd := resolutionIntent(t, f, "must not bypass current authority", 1)
				raw, e := base64.StdEncoding.Strict().DecodeString(cmd.Envelope.Payload)
				if e != nil {
					t.Fatal(e)
				}
				var p signing.ReturnResolutionPayload
				if e = signing.DecodeStrict(raw, &p, 32<<10); e != nil {
					t.Fatal(e)
				}
				foreign := "0199a555-0000-7000-8000-000000000099"
				switch probe {
				case "credential":
					p.CredentialID = foreign
				case "signer":
					p.SignerKeyID = foreign
				case "digest":
					p.OriginalRequestDigest = signing.Digest([]byte("wrong exact bytes"))
				case "scope":
					p.OutcomeID = foreign
				case "issuer":
					p.ServerID = foreign
				case "revoked":
					if e = f.store.RevokeCredential(f.ctx, f.credential); e != nil {
						t.Fatal(e)
					}
				case "retired-key":
					u, e := f.store.Begin(f.ctx)
					if e != nil {
						t.Fatal(e)
					}
					registry := u.(ports.SigningIdentityUnitOfWork).SigningIdentity()
					k, e := registry.Key(f.ctx, cmd.Scope.NamespaceID, d.ID(p.SignerKeyID))
					if e != nil {
						u.Rollback()
						t.Fatal(e)
					}
					k.Status = "retired"
					k.Version++
					k.UpdatedAt = k.UpdatedAt.Add(time.Second)
					if e = registry.SaveKey(f.ctx, k, k.Version-1); e != nil {
						u.Rollback()
						t.Fatal(e)
					}
					if e = u.Commit(); e != nil {
						t.Fatal(e)
					}
				case "revoked-contract":
					u, e := f.store.Begin(f.ctx)
					if e != nil {
						t.Fatal(e)
					}
					repo := u.(ports.SignedContractUnitOfWork).SignedWorkContracts()
					c, e := repo.Get(f.ctx, cmd.Scope, d.ID(p.ContractID))
					if e != nil {
						u.Rollback()
						t.Fatal(e)
					}
					originalVersion, originalLease := c.Version, c.LeaseVersion
					if e = c.Revoke("operator", "explicit revoked authority fixture", c.Version, c.AcquiredAt); e != nil {
						u.Rollback()
						t.Fatal(e)
					}
					if e = repo.Save(f.ctx, c, originalVersion, originalLease); e != nil {
						u.Rollback()
						t.Fatal(e)
					}
					if e = u.Commit(); e != nil {
						t.Fatal(e)
					}
				}
				cmd.Envelope, e = signing.Sign(signing.ReturnResolution, p, p.SignerKeyID, f.private)
				if e != nil {
					t.Fatal(e)
				}
				if _, e = f.service.ResolveSignedReturnIntent(f.ctx, cc, cmd); e == nil {
					t.Fatal("changed authority accepted")
				}
				u, e := f.store.Begin(f.ctx)
				if e != nil {
					t.Fatal(e)
				}
				defer u.Rollback()
				if _, e = u.(ports.SignedReturnResolutionUnitOfWork).SignedReturnResolutions().GetByOriginalIntent(f.ctx, cmd.Scope.NamespaceID, f.cc.PrincipalID, f.credential, "ReturnSignedWork", f.request.IdempotencyKey); e == nil {
					t.Fatal("invalid authority persisted terminal decision")
				}
			})
		}
	}
}

func TestSignedReturnResolutionSurvivesCleanRestore(t *testing.T) {
	f := newResolutionFixture(t, "sql")
	cc, cmd := resolutionIntent(t, f, "terminal proof must survive a clean database restore", 1)
	closed, e := f.service.ResolveSignedReturnIntent(f.ctx, cc, cmd)
	if e != nil {
		t.Fatal(e)
	}
	source := f.store.(*Store)
	restored := reopenIntegrationFixture(t, source, source.Path())
	service, e := a.NewService(restored, sqliteFixedClock{time.Now().UTC()}, &sqliteSequenceIDs{prefix: "0199a557", next: 1})
	if e != nil {
		t.Fatal(e)
	}
	observed, e := service.ReadSignedState(f.ctx, a.SignedStateQuery{Scope: cmd.Scope, Resource: "return_resolution", ContractKind: "execution", IdempotencyKey: f.request.IdempotencyKey})
	if e != nil {
		t.Fatal(e)
	}
	if observed.Envelope == nil {
		t.Fatal("restored terminal proof missing")
	}
	doc, e := signing.ToDocument(*observed.Envelope)
	if e != nil {
		t.Fatal(e)
	}
	if string(doc.Payload) != string(closed.Value.ResolutionReceipt.Payload) || doc.Proof != closed.Value.ResolutionReceipt.Proof {
		t.Fatal("restore replaced signed proof")
	}
	_, e = service.ReturnSignedWork(f.ctx, f.cc, a.ReturnSignedWorkCommand{Envelope: f.envelope})
	if code, _ := d.ErrorCodeOf(e); code != d.ErrorCodeSignedIntentClosed {
		t.Fatalf("restore resurrected closed return: %v", e)
	}
	if e = restored.RevokeCredential(f.ctx, f.credential); e != nil {
		t.Fatal(e)
	}
	if _, e = service.ReadSignedState(f.ctx, a.SignedStateQuery{Scope: cmd.Scope, Resource: "return_resolution", ContractKind: "execution", IdempotencyKey: f.request.IdempotencyKey}); e == nil {
		t.Fatal("restored terminal proof bypassed current credential revocation")
	}
}

type resolutionRealClock struct{}

func (resolutionRealClock) Now() time.Time { return time.Now().UTC() }
func TestSignedReturnResolutionRejectsActualExpiredLease(t *testing.T) {
	for _, backend := range []string{"memory", "sql"} {
		t.Run(backend, func(t *testing.T) {
			t.Parallel()
			f := newResolutionFixtureTTL(t, backend, 30)
			service, e := a.NewService(f.store, resolutionRealClock{}, &sqliteSequenceIDs{prefix: "0199a558", next: 1})
			if e != nil {
				t.Fatal(e)
			}
			if e = service.ConfigureSignedIssuer(f.ctx, acquisitionIssuer{f.server, f.issuerPrivate}, d.AcceptanceDirect); e != nil {
				t.Fatal(e)
			}
			cc, cmd := resolutionIntent(t, f, "expired authority is never bypassed by closure", 1)
			u, e := f.store.Begin(f.ctx)
			if e != nil {
				t.Fatal(e)
			}
			contract, e := u.(ports.SignedContractUnitOfWork).SignedWorkContracts().Get(f.ctx, cmd.Scope, d.ID(f.request.ContractID))
			if e != nil {
				u.Rollback()
				t.Fatal(e)
			}
			before, e := u.Coordination().LockOutcome(f.ctx, cmd.Scope)
			if e != nil {
				u.Rollback()
				t.Fatal(e)
			}
			u.Rollback()
			// Wait for the actual persisted deadline. No row, signature or database clock
			// is modified, including in the generated PostgreSQL test.
			timer := time.NewTimer(max(0, time.Until(contract.ExpiresAt.Add(50*time.Millisecond))))
			defer timer.Stop()
			<-timer.C
			_, e = service.ResolveSignedReturnIntent(f.ctx, cc, cmd)
			if code, _ := d.ErrorCodeOf(e); code != d.ErrorCodeContractExpired {
				t.Fatalf("expired authority was bypassed: %v", e)
			}
			_, e = service.ReturnSignedWork(f.ctx, f.cc, a.ReturnSignedWorkCommand{Envelope: f.envelope})
			if code, _ := d.ErrorCodeOf(e); code != d.ErrorCodeContractExpired {
				t.Fatalf("expired return was accepted: %v", e)
			}
			u, e = f.store.Begin(f.ctx)
			if e != nil {
				t.Fatal(e)
			}
			defer u.Rollback()
			after, e := u.Coordination().LockOutcome(f.ctx, cmd.Scope)
			if e != nil {
				t.Fatal(e)
			}
			current, e := u.(ports.SignedContractUnitOfWork).SignedWorkContracts().Get(f.ctx, cmd.Scope, contract.ID)
			if e != nil {
				t.Fatal(e)
			}
			b, _ := json.Marshal(contract)
			ar, _ := json.Marshal(current)
			if before.Revision != after.Revision || string(b) != string(ar) {
				t.Fatal("read-time expiry or rejected closure mutated authority/revision")
			}
			if _, e = u.(ports.SignedReturnResolutionUnitOfWork).SignedReturnResolutions().GetByOriginalIntent(f.ctx, cmd.Scope.NamespaceID, f.cc.PrincipalID, f.credential, "ReturnSignedWork", f.request.IdempotencyKey); e == nil {
				t.Fatal("expired lease created a terminal decision")
			}
		})
	}
}
