package acceptance_test

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	ht "github.com/A1b3rt0M3rcad0/wos/packages/wos-api/http"
	mt "github.com/A1b3rt0M3rcad0/wos/packages/wos-api/mcp"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type transportIssuer struct {
	identity d.ServerIdentity
	private  ed25519.PrivateKey
}

func (i transportIssuer) Identity() d.ServerIdentity { return i.identity }
func (i transportIssuer) SignCanonical(purpose string, raw []byte) (json.RawMessage, error) {
	envelope, err := signing.SignCanonical(signing.PayloadType(purpose), raw, i.identity.IssuerKeyID.String(), i.private)
	if err != nil {
		return nil, err
	}
	document, err := signing.ToDocument(envelope)
	if err != nil {
		return nil, err
	}
	return json.Marshal(document.Proof)
}

func TestSignedPublicEnrollmentHTTPAcquireMCPReturnSDKRead(t *testing.T) {
	for _, backend := range []string{"memory", "sqlite", "postgres"} {
		t.Run(backend, func(t *testing.T) {
			var store interface {
				ports.TransactionManager
				ports.SecurityStore
			}
			switch backend {
			case "memory":
				store = memory.New()
			case "sqlite":
				s, e := sqlite.Open(filepath.Join(t.TempDir(), "signed.db"), sqlite.Options{MigrateOnOpen: true})
				if e != nil {
					t.Fatal(e)
				}
				t.Cleanup(func() { s.Close() })
				store = s
			case "postgres":
				if os.Getenv("WOS_TEST_POSTGRES_DSN") == "" {
					t.Skip("real PostgreSQL required")
				}
				s, e := postgres.Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), postgres.Options{MigrateOnOpen: true, Schema: fmt.Sprintf("signed_transport_%d", time.Now().UnixNano())})
				if e != nil {
					t.Fatal(e)
				}
				t.Cleanup(func() { s.Close() })
				store = s
			}
			signedTransportJourney(t, store)
		})
	}
}
func signedTransportJourney(t *testing.T, store interface {
	ports.TransactionManager
	ports.SecurityStore
}) {
	t.Helper()
	ctx := context.Background()
	generator := &ids{}
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	ns := d.MustParseID("0199ff31-0000-7000-8000-000000000001")
	scope := d.Scope{NamespaceID: ns, OutcomeID: d.MustParseID("0199ff31-0000-7000-8000-000000000002")}
	now := time.Now().UTC()
	sec := &a.SecurityService{Store: store, Clock: clock{}, IDs: generator}
	const adminToken = "signed-public-admin-credential-at-least-32"
	must(sec.Bootstrap(ctx, ports.Namespace{ID: ns, Name: "signed transport"}, "operator", adminToken))
	admin, e := sec.Authenticate(ctx, adminToken)
	must(e)
	adminCtx := a.WithIdentity(ctx, admin)
	permissions := []ports.Permission{ports.PermissionStateRead, ports.PermissionSigningKeyEnroll, ports.PermissionWorkContractAcquire, ports.PermissionWorkContractReturn, ports.PermissionWorkCompleteDirect, ports.PermissionWorkWrite, ports.PermissionAssessmentWrite, ports.PermissionConclusionWrite}
	must(sec.SetGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "worker", Permissions: permissions}))
	credential, token, e := sec.IssueCredential(adminCtx, ns, "worker", d.ActorRef{Kind: d.ActorKindAgent, Provider: "harness", ID: "worker"}, now.Add(time.Hour))
	must(e)
	service, e := a.NewAuthorizedService(store, clock{}, generator, sec)
	must(e)
	pub, private, e := ed25519.GenerateKey(rand.Reader)
	must(e)
	fp, e := signing.Fingerprint(pub)
	must(e)
	server := d.ServerIdentity{ID: d.MustParseID("0199ff31-0000-7000-8000-000000000003"), IssuerKeyID: d.MustParseID("0199ff31-0000-7000-8000-000000000004"), PublicKey: base64.StdEncoding.EncodeToString(pub), Fingerprint: fp, CreatedAt: now}
	must(service.ConfigureSignedIssuer(ctx, transportIssuer{server, private}, d.AcceptanceDirect))
	sec.ServerID = server.ID.String()
	resolve := func(ctx context.Context, headers http.Header) (a.Identity, error) {
		return sec.Authenticate(ctx, strings.TrimPrefix(headers.Get("Authorization"), "Bearer "))
	}
	handler, e := ht.New(ht.Options{Prefix: "/api/v1", Service: service, IDs: generator, Security: sec, RequestTimeout: 10 * time.Second, ResolveIdentity: func(r *http.Request) (a.Identity, error) { return resolve(r.Context(), r.Header) }})
	must(e)
	protocol, e := mt.New(service, generator, mt.Options{Security: sec, ResolveIdentity: resolve, RequestTimeout: 10 * time.Second})
	must(e)
	mux := http.NewServeMux()
	mux.Handle("/mcp", mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return protocol }, &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true}))
	mux.Handle("/", handler)
	endpoint := httptest.NewServer(mux)
	defer endpoint.Close()
	operator, e := sdk.New(endpoint.URL, adminToken, endpoint.Client())
	must(e)
	worker, e := sdk.New(endpoint.URL, token, endpoint.Client())
	must(e)
	identity, e := operator.SigningIdentity(ctx, nil)
	must(e)
	policy := d.CredentialPolicy{NamespaceID: ns, CredentialID: credential.ID, PrincipalID: "worker", AcceptanceFloor: d.AcceptanceIndependentReview, MaxActiveWorkContracts: 3, MaxActiveReviewContracts: 1}
	for _, p := range permissions {
		policy.PermittedOperations = append(policy.PermittedOperations, string(p))
	}
	configured, e := operator.SigningMutation(ctx, "transport-set-signing-policy", a.SigningSecurityIntent{NamespaceID: ns, ExpectedNamespaceVersion: identity.NamespaceVersion, Operation: "set_credential_policy", CredentialPolicy: &policy})
	must(e)
	agentPub, agentPrivate, e := ed25519.GenerateKey(rand.Reader)
	must(e)
	agentFP, e := signing.Fingerprint(agentPub)
	must(e)
	enrollment, e := operator.SigningMutation(ctx, "transport-enroll-signing-key", a.SigningSecurityIntent{NamespaceID: ns, ExpectedNamespaceVersion: configured.NamespaceVersion, Operation: "create_enrollment", CredentialID: credential.ID, PrincipalID: "worker", ExpectedFingerprint: agentFP})
	must(e)
	own, e := worker.SigningIdentity(ctx, &enrollment.Enrollment.ID)
	must(e)
	if own.ServerID != server.ID.String() || own.CredentialID != credential.ID || own.Enrollment == nil {
		t.Fatal("public identity binding differs")
	}
	proof, e := signing.Sign(signing.KeyEnrollment, signing.EnrollmentPayload{Binding: signing.Binding{ProtocolVersion: 2, ServerID: server.ID.String(), NamespaceID: ns.String(), PrincipalID: "worker", SignerKeyID: own.Enrollment.KeyID.String()}, EnrollmentID: own.Enrollment.ID.String(), Nonce: own.Enrollment.Nonce, Purpose: own.Enrollment.Purpose, PublicKey: base64.StdEncoding.EncodeToString(agentPub), ExpiresAt: own.Enrollment.ExpiresAt.UTC().Format(time.RFC3339Nano)}, own.Enrollment.KeyID.String(), agentPrivate)
	must(e)
	// This identity resolver uses a real credential on every MCP request.
	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "signed-harness", Version: "2"}, nil)
	authenticatedHTTP := *endpoint.Client()
	authenticatedHTTP.Transport = credentialTransport{base: endpoint.Client().Transport, token: token}
	session, e := mcpClient.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: endpoint.URL + "/mcp", HTTPClient: &authenticatedHTTP, DisableStandaloneSSE: true}, nil)
	must(e)
	defer session.Close()
	call := func(name string, args any, target any) {
		t.Helper()
		r, e := session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: args})
		must(e)
		if r.IsError {
			t.Fatalf("%s: %s", name, r.StructuredContent)
		}
		raw, e := json.Marshal(r.StructuredContent)
		must(e)
		must(json.Unmarshal(raw, target))
	}
	var registered a.SigningSecurityResult
	call("wos_manage_signing_identity", map[string]any{"idempotency_key": "transport-register-signing-key", "command": map[string]any{"namespace_id": ns, "operation": "register_signing_key", "expected_namespace_version": "0", "expected_version": "0", "enrollment_id": own.Enrollment.ID, "proof": proof}}, &registered)
	if registered.Key == nil || registered.Key.PrincipalID != "worker" {
		t.Fatal("MCP registration changed ownership")
	}
	u, e := store.Begin(ctx)
	must(e)
	outcome, e := d.NewOutcome(scope.OutcomeID, ns, "Signed parity", "", "Exact scoped delivery", d.PriorityNormal, now)
	must(e)
	outcome.Lifecycle = d.OutcomeLifecycleActive
	must(u.Outcomes().Insert(ctx, outcome))
	work, e := d.NewWorkItem(d.MustParseID("0199ff31-0000-7000-8000-000000000005"), scope, "Bounded task", "", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
	must(e)
	work.ContractsEnabled = true
	work.LastFencingToken = 9007199254740993
	must(u.WorkItems().Insert(ctx, work))
	p := u.(ports.WorkProtocolUnitOfWork).WorkProtocol()
	phase, e := p.Lock(ctx, ns, true)
	must(e)
	previous := phase.Version
	phase.Phase = d.WorkProtocolSigned
	phase.WriterEpoch = 2
	phase.WritersDrained = true
	phase.Version++
	phase.UpdatedAt = now
	phase.UpdatedBy = "test operator"
	phase.Reason = "fixture activation; public cutover tested separately"
	must(p.Save(ctx, phase, previous))
	must(u.Commit())
	signedCLIProfileJourney(t, handler, sec, adminCtx, operator, scope, server, permissions)
	acquired, e := worker.AcquireSignedWorkContract(ctx, "transport-signed-acquisition", a.AcquireSignedWorkContractCommand{Scope: scope, WorkItemID: work.ID, ExpectedWorkItemVersion: work.Version, SignerKeyID: registered.Key.ID, TTLSeconds: 300})
	must(e)
	operation, e := worker.ReadSignedState(ctx, a.SignedStateQuery{Scope: scope, Resource: "operation", IdempotencyKey: "transport-signed-acquisition"})
	must(e)
	originalResponse, e := base64.StdEncoding.Strict().DecodeString(operation.OperationPayload)
	must(e)
	var originalAcquisition sdk.CommandResult[a.WorkContractResult]
	must(json.Unmarshal(originalResponse, &originalAcquisition))
	if operation.OperationCommandID == nil || *operation.OperationCommandID != acquired.CommandID || operation.PayloadDigest != signing.Digest(originalResponse) || originalAcquisition.Value.Contract.ID != acquired.Value.Contract.ID {
		t.Fatal("HTTP durable acquisition expansion changed original response")
	}
	c := acquired.Value.Contract
	requestID, e := generator.NewID()
	must(e)
	request := signing.WorkReturnPayload[a.SignedReturnMaterial]{RequestBinding: signing.RequestBinding{Binding: signing.Binding{ProtocolVersion: 2, ServerID: server.ID.String(), NamespaceID: ns.String(), OutcomeID: scope.OutcomeID.String(), PrincipalID: "worker", SignerKeyID: registered.Key.ID.String()}, OperationKind: "work_return", RequestID: requestID.String(), IdempotencyKey: "transport-signed-return", ContractID: c.ID.String(), WorkItemID: work.ID.String(), ExecutionID: c.ExecutionID.String(), FencingToken: signing.Decimal(c.FencingToken), SpecDigest: c.SignedBinding.SpecificationDigest, AuthorityDigest: signing.Digest(acquired.Value.IssuedAuthority.Payload), ExpectedContractVersion: signing.Decimal(c.Version), ExpectedLeaseVersion: signing.Decimal(c.LeaseVersion), ExpectedWorkItemVersion: signing.Decimal(acquired.Value.WorkItem.Version), PolicyRevision: signing.Decimal(registered.CredentialPolicy.Version)}, CompletionIntent: "auto", Material: a.SignedReturnMaterial{Result: d.NewSignedResultMaterial(d.WorkResultMaterial{ContractID: c.ID, WorkItemID: work.ID, SpecDigest: c.SignedBinding.SpecificationDigest, Summary: "same signed bytes across transports"}), Reason: "no required criteria in bounded fixture"}}
	envelope, e := signing.Sign(signing.WorkReturn, request, registered.Key.ID.String(), agentPrivate)
	must(e)
	var returned a.MutationResult[a.SignedReturnResult]
	call("wos_return_signed_work", map[string]any{"idempotency_key": request.IdempotencyKey, "command": map[string]any{"envelope": envelope}}, &returned)
	receiptEnvelope, e := returned.Value.Receipt.Envelope()
	must(e)
	receipt, _, e := signing.Decode[signing.ReceiptPayload](receiptEnvelope, signing.AcceptanceReceipt, server.IssuerKeyID.String(), pub)
	must(e)
	if !receipt.LocalObligationClosed || receipt.WorkItemLifecycle != "in_progress" {
		t.Fatal("MCP return did not atomically close executor obligation")
	}
	state, e := worker.ReadSignedState(ctx, a.SignedStateQuery{Scope: scope, Resource: "execution", ID: c.ID})
	must(e)
	if state.Contract == nil || state.Contract.LeaseValid || uint64(state.Contract.FencingToken) != 9007199254740994 {
		t.Fatal("HTTP metadata lost exact fencing/closed state")
	}
	var replay a.SignedStateResult
	call("wos_read_signed_state", a.SignedStateQuery{Scope: scope, Resource: "receipt", IdempotencyKey: request.IdempotencyKey}, &replay)
	if replay.Envelope == nil || replay.Envelope.Payload != receiptEnvelope.Payload {
		t.Fatal("cross-transport receipt changed signed bytes")
	}
	_, e = worker.CompleteWorkItem(ctx, "transport-unsigned-complete-bypass", a.CompleteWorkItemCommand{Scope: scope, WorkItemID: work.ID, ExpectedVersion: d.Version(state.Contract.WorkItemVersion), ClaimID: c.ID, FencingToken: uint64(c.FencingToken), ResultSummary: "unsigned bypass", Reason: "must be rejected"})
	var typed *sdk.Error
	if !errors.As(e, &typed) || typed.Code != string(d.ErrorCodeSignedProtocolRequired) {
		t.Fatalf("unsigned v1 completion did not hit protocol guard: %v", e)
	}
	// A separate reviewer uses only its own credential/key/lease, after the
	// original execution bearer has been revoked. No executor session is reused.
	reviewPermissions := []ports.Permission{ports.PermissionStateRead, ports.PermissionSigningKeyEnroll, ports.PermissionWorkReviewAcquire, ports.PermissionWorkReviewDecide, ports.PermissionAssessmentWrite, ports.PermissionConclusionWrite}
	must(sec.SetGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "reviewer", Permissions: reviewPermissions}))
	reviewCredential, reviewToken, e := sec.IssueCredential(adminCtx, ns, "reviewer", d.ActorRef{Kind: d.ActorKindAgent, Provider: "harness", ID: "reviewer"}, now.Add(time.Hour))
	must(e)
	reviewer, e := sdk.New(endpoint.URL, reviewToken, endpoint.Client())
	must(e)
	adminView, e := operator.SigningIdentity(ctx, nil)
	must(e)
	reviewPolicy := d.CredentialPolicy{NamespaceID: ns, CredentialID: reviewCredential.ID, PrincipalID: "reviewer", AcceptanceFloor: d.AcceptanceIndependentReview, MaxActiveWorkContracts: 3, MaxActiveReviewContracts: 1}
	for _, permission := range reviewPermissions {
		reviewPolicy.PermittedOperations = append(reviewPolicy.PermittedOperations, string(permission))
	}
	reviewConfigured, e := operator.SigningMutation(ctx, "transport-review-policy", a.SigningSecurityIntent{NamespaceID: ns, ExpectedNamespaceVersion: adminView.NamespaceVersion, Operation: "set_credential_policy", CredentialPolicy: &reviewPolicy})
	must(e)
	reviewPublic, reviewPrivate, e := ed25519.GenerateKey(rand.Reader)
	must(e)
	reviewFP, e := signing.Fingerprint(reviewPublic)
	must(e)
	reviewEnrollment, e := operator.SigningMutation(ctx, "transport-review-enrollment", a.SigningSecurityIntent{NamespaceID: ns, ExpectedNamespaceVersion: reviewConfigured.NamespaceVersion, Operation: "create_enrollment", CredentialID: reviewCredential.ID, PrincipalID: "reviewer", ExpectedFingerprint: reviewFP})
	must(e)
	challenge := reviewEnrollment.Enrollment
	reviewProof, e := signing.Sign(signing.KeyEnrollment, signing.EnrollmentPayload{Binding: signing.Binding{ProtocolVersion: 2, ServerID: server.ID.String(), NamespaceID: ns.String(), PrincipalID: "reviewer", SignerKeyID: challenge.KeyID.String()}, EnrollmentID: challenge.ID.String(), Nonce: challenge.Nonce, Purpose: challenge.Purpose, PublicKey: base64.StdEncoding.EncodeToString(reviewPublic), ExpiresAt: challenge.ExpiresAt.UTC().Format(time.RFC3339Nano)}, challenge.KeyID.String(), reviewPrivate)
	must(e)
	reviewRegistered, e := reviewer.SigningMutation(ctx, "transport-review-register", a.SigningSecurityIntent{NamespaceID: ns, Operation: "register_signing_key", EnrollmentID: challenge.ID, Proof: &reviewProof})
	must(e)
	must(sec.RevokeCredential(adminCtx, ns, credential.ID))
	caseView, e := reviewer.ReadSignedState(ctx, a.SignedStateQuery{Scope: scope, Resource: "case", ID: d.ID(receipt.ReviewCaseID)})
	must(e)
	if caseView.ReviewCase == nil {
		t.Fatal("delivery lost independent review case")
	}
	assigned := signedCLIReviewCheckout(t, mux, scope, server, reviewCredential, reviewToken, *reviewRegistered.Key, reviewPrivate, reviewer)
	must(e)
	rc := assigned.Value.Contract
	reviewRequestID, e := generator.NewID()
	must(e)
	decision := signing.ReviewReturnPayload[a.SignedReviewMaterial]{RequestBinding: signing.RequestBinding{Binding: signing.Binding{ProtocolVersion: 2, ServerID: server.ID.String(), NamespaceID: ns.String(), OutcomeID: scope.OutcomeID.String(), PrincipalID: "reviewer", SignerKeyID: reviewRegistered.Key.ID.String()}, OperationKind: "review_return", RequestID: reviewRequestID.String(), IdempotencyKey: "transport-sdk-review-approve", ContractID: rc.ID.String(), WorkItemID: work.ID.String(), ExecutionID: rc.ExecutionID.String(), FencingToken: signing.Decimal(rc.FencingToken), SpecDigest: rc.Binding.SpecificationDigest, AuthorityDigest: signing.Digest(assigned.Value.IssuedAuthority.Payload), ExpectedContractVersion: signing.Decimal(rc.Version), ExpectedLeaseVersion: signing.Decimal(rc.LeaseVersion), ExpectedWorkItemVersion: assigned.Value.WorkItemVersion, PolicyRevision: signing.Decimal(reviewRegistered.CredentialPolicy.Version)}, ExpectedReviewCaseVersion: signing.Decimal(assigned.Value.Case.Version), ReviewCaseID: assigned.Value.Case.ID.String(), SubmissionID: receipt.SubmissionID, SubmissionDigest: receipt.SubmissionDigest, Decision: "approved", Material: a.SignedReviewMaterial{Reason: "independent review of exact submitted result; no mandatory criteria"}}
	decisionEnvelope, e := signing.Sign(signing.ReviewReturn, decision, reviewRegistered.Key.ID.String(), reviewPrivate)
	must(e)
	approved, e := reviewer.ReturnSignedReview(ctx, decision.IdempotencyKey, a.ReturnSignedReviewCommand{Envelope: decisionEnvelope})
	must(e)
	approvedEnvelope, e := approved.Value.Receipt.Envelope()
	must(e)
	approvedReceipt, _, e := signing.Decode[signing.ReceiptPayload](approvedEnvelope, signing.AcceptanceReceipt, server.IssuerKeyID.String(), pub)
	must(e)
	if approvedReceipt.WorkItemLifecycle != "done" || approvedReceipt.Disposition != "review_approved" {
		t.Fatal("independent SDK reviewer failed to complete delivered Task")
	}
	finalState, e := reviewer.ReadSignedState(ctx, a.SignedStateQuery{Scope: scope, Resource: "execution", ID: c.ID})
	must(e)
	if finalState.Contract == nil || finalState.Contract.CredentialID != credential.ID {
		t.Fatal("public signed execution metadata lost original CID")
	}
	if finalState.Contract.Status != d.ContractDelivered || finalState.Contract.WorkItemLifecycle != d.WorkItemLifecycleDone || finalState.Contract.LeaseValid {
		t.Fatal("HTTP projection rewrote original delivery or lost independent completion")
	}

}

type credentialTransport struct {
	base  http.RoundTripper
	token string
}

func (t credentialTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	r = r.Clone(r.Context())
	r.Header.Set("Authorization", "Bearer "+t.token)
	base := t.base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(r)
}
