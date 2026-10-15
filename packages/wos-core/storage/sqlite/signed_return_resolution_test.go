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
	service    *a.Service
	ctx        context.Context
	cc         d.CommandContext
	request    signing.WorkReturnPayload[a.SignedReturnMaterial]
	envelope   signing.Envelope
	private    ed25519.PrivateKey
	credential d.ID
	server     d.ServerIdentity
}

func newResolutionFixture(t *testing.T, backend string) resolutionFixture {
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
		must(uow.WorkItems().Insert(ctx, w))
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
	acquired, e := service.AcquireSignedWorkContract(agentCtx, cc, a.AcquireSignedWorkContractCommand{Scope: scope, WorkItemID: w.ID, ExpectedWorkItemVersion: w.Version, SignerKeyID: agentKey, TTLSeconds: 300})
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
	return resolutionFixture{store, service, agentCtx, returnCC, request, envelope, agentPrivate, credential.ID, server}
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
