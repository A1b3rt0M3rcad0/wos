package acceptance_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	cli "github.com/A1b3rt0M3rcad0/wos/packages/wos-cli"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

func signedCLIProfileJourney(t *testing.T, handler http.Handler, security *a.SecurityService, adminCtx context.Context, operator *sdk.Client, scope d.Scope, server d.ServerIdentity, permissions []ports.Permission) {
	t.Helper()
	ctx := context.Background()
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	must(security.SetGrant(adminCtx, ports.NamespaceGrant{NamespaceID: scope.NamespaceID, PrincipalID: "cli-worker", Permissions: permissions}))
	credential, token, e := security.IssueCredential(adminCtx, scope.NamespaceID, "cli-worker", d.ActorRef{Kind: d.ActorKindAgent, Provider: "cli-harness", ID: "cli-worker"}, time.Now().Add(time.Hour))
	must(e)
	identity, e := operator.SigningIdentity(ctx, nil)
	must(e)
	policy := d.CredentialPolicy{NamespaceID: scope.NamespaceID, CredentialID: credential.ID, PrincipalID: "cli-worker", AcceptanceFloor: d.AcceptanceIndependentReview, MaxActiveWorkContracts: 3, MaxActiveReviewContracts: 1}
	for _, p := range permissions {
		policy.PermittedOperations = append(policy.PermittedOperations, string(p))
	}
	configured, e := operator.SigningMutation(ctx, "cli-profile-policy", a.SigningSecurityIntent{NamespaceID: scope.NamespaceID, ExpectedNamespaceVersion: identity.NamespaceVersion, Operation: "set_credential_policy", CredentialPolicy: &policy})
	must(e)
	public, private, e := ed25519.GenerateKey(rand.Reader)
	must(e)
	fingerprint := signing.Digest(public)
	enrollment, e := operator.SigningMutation(ctx, "cli-profile-enrollment", a.SigningSecurityIntent{NamespaceID: scope.NamespaceID, ExpectedNamespaceVersion: configured.NamespaceVersion, Operation: "create_enrollment", CredentialID: credential.ID, PrincipalID: "cli-worker", ExpectedFingerprint: fingerprint})
	must(e)
	root := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	t.Setenv("LOCALAPPDATA", t.TempDir())
	t.Setenv("WOS_CLI_PROFILE_TEST_TOKEN", token)
	t.Setenv("WOS_CLI_PROFILE_TEST_KEY", base64.StdEncoding.EncodeToString(private.Seed()))
	var drop atomic.Bool
	drop.Store(true)
	var registrationCalls atomic.Int32
	var dropCheckout atomic.Bool
	var acquisitionCalls atomic.Int32
	var firstAcquireKey atomic.Value
	firstAcquireKey.Store("")
	endpoint := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {

		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/commands/acquire_next_signed_work_contract") {
			acquisitionCalls.Add(1)
			key := r.Header.Get("Idempotency-Key")
			firstAcquireKey.CompareAndSwap("", key)
			recorded := httptest.NewRecorder()
			handler.ServeHTTP(recorded, r)
			if dropCheckout.Load() && key != firstAcquireKey.Load().(string) && recorded.Code < 400 {
				connection, _, e := rw.(http.Hijacker).Hijack()
				if e != nil {
					t.Error(e)
					return
				}
				connection.Close()
				return
			}
			for key, values := range recorded.Header() {
				rw.Header()[key] = values
			}
			rw.WriteHeader(recorded.Code)
			_, _ = rw.Write(recorded.Body.Bytes())
			return
		}
		if r.Method == http.MethodPost && strings.HasPrefix(r.Header.Get("Idempotency-Key"), "profile-enrollment-") {
			registrationCalls.Add(1)
			// Fail the response only after the actual Core committed registration.
			recorded := httptest.NewRecorder()
			handler.ServeHTTP(recorded, r)
			if recorded.Code >= 400 {
				t.Errorf("registration failed before loss: %s", recorded.Body.String())
			}
			if drop.Load() {
				connection, _, e := rw.(http.Hijacker).Hijack()
				if e != nil {
					t.Error(e)
					return
				}
				connection.Close()
				return
			}
			for key, values := range recorded.Header() {
				rw.Header()[key] = values
			}
			rw.WriteHeader(recorded.Code)
			rw.Write(recorded.Body.Bytes())
			return
		}
		handler.ServeHTTP(rw, r)
	}))
	defer endpoint.Close()
	execute := func(expected int, args ...string) cli.Output {
		t.Helper()
		var out, diagnostics bytes.Buffer
		code := cli.Run(ctx, append(args, "--workspace", root, "--output", "json"), &out, &diagnostics)
		if code != expected {
			t.Fatalf("CLI %v: exit=%d expected=%d: %s %s", args, code, expected, out.String(), diagnostics.String())
		}
		var result cli.Output
		if e := json.Unmarshal(out.Bytes(), &result); e != nil {
			t.Fatalf("CLI %v emitted invalid JSON: %v; %s", args, e, diagnostics.String())
		}
		return result
	}
	execute(0, "init", "--workspace-schema", "2", "--server", endpoint.URL, "--server-id", server.ID.String(), "--namespace", scope.NamespaceID.String(), "--outcome", scope.OutcomeID.String())
	profile := cli.ProfileV2{SchemaVersion: 2, Kind: "WOSProfile", Name: "executor_cli", Binding: cli.ProfileBindingV2{ServerID: server.ID, ServerOrigin: endpoint.URL, NamespaceID: scope.NamespaceID, PrincipalID: "cli-worker", CredentialID: credential.ID, IssuerKeys: []cli.ProfileIssuerV2{{KeyID: server.IssuerKeyID, PublicKey: server.PublicKey, Fingerprint: server.Fingerprint}}}, Authentication: cli.ProfileAuthenticationV2{CredentialRef: "env:WOS_CLI_PROFILE_TEST_TOKEN"}, Signing: cli.ProfileSigningV2{KeyID: enrollment.Enrollment.KeyID, PrivateKeyRef: "env:WOS_CLI_PROFILE_TEST_KEY", PublicKeyFingerprint: fingerprint}, Lease: cli.LeaseConfig{RequestedTTLSeconds: 300}, Output: cli.OutputConfig{DefaultFormat: "json"}, Local: cli.ProfileLocalV2{SchemaVersion: 1, PendingOperations: []cli.PendingOperationV2{}}}
	proposal, e := cli.EncodeV2Document(profile)
	must(e)
	must(os.WriteFile(filepath.Join(root, "proposal.yaml"), proposal, 0600))
	execute(2, "profile", "create", "executor_cli", "--file", "proposal.yaml")
	if registrationCalls.Load() != 0 {
		t.Fatal("unapproved origin caused mutation")
	}
	execute(6, "profile", "create", "executor_cli", "--file", "proposal.yaml", "--server", endpoint.URL, "--server-id", server.ID.String(), "--issuer-fingerprint", server.Fingerprint, "--enrollment", enrollment.Enrollment.ID.String())
	workspace, e := cli.OpenWorkspace(root)
	must(e)
	defer workspace.Close()
	pending, e := workspace.LoadProfileV2("executor_cli")
	must(e)
	if len(pending.Local.PendingOperations) != 1 || pending.Local.PendingOperations[0].State != "sent_unknown" {
		t.Fatal("lost response discarded durable enrollment intent")
	}
	frozen := pending.Local.PendingOperations[0]
	execute(6, "profile", "remove", "executor_cli")
	beforeRecoveryCalls := registrationCalls.Load()
	drop.Store(false)
	result := execute(0, "profile", "recover", "executor_cli")
	if !result.Committed || registrationCalls.Load() != beforeRecoveryCalls+1 {
		t.Fatal("recovery did not replay one already committed intent")
	}
	recovered, e := workspace.LoadProfileV2("executor_cli")
	must(e)
	if len(recovered.Local.PendingOperations) != 0 {
		t.Fatal("confirmed registration stayed pending")
	}
	if frozen.IdempotencyKey == "" {
		t.Fatal("pending idempotency absent")
	}
	execute(0, "--profile", "executor_cli", "auth", "status")
	execute(0, "--profile", "executor_cli", "profile", "inspect")

	// Isolate the CLI tasks from the transport review fixture, using the actual
	// public creation path after signed activation rather than SQL-only fixtures.
	created, e := operator.CreateOutcome(ctx, "cli-batch-outcome", a.CreateOutcomeCommand{NamespaceID: scope.NamespaceID, Title: "CLI recovery", DesiredState: "two original grants materialized", Priority: d.PriorityNormal})
	must(e)
	batchScope := d.Scope{NamespaceID: scope.NamespaceID, OutcomeID: created.Value.ID}
	_, e = operator.AddCriterion(ctx, "cli-batch-outcome-criterion", a.AddCriterionCommand{Owner: created.Value.Ref(), ExpectedVersion: created.Value.Version, Title: "Both original contracts materialized", Required: true, VerificationMode: d.VerificationModeAttestation})
	must(e)
	_, e = operator.ActivateOutcome(ctx, "cli-batch-outcome-active", a.ActivateOutcomeCommand{Scope: batchScope, ExpectedVersion: created.Value.Version + 1})
	must(e)
	for i := 0; i < 2; i++ {
		work, e := operator.CreateWorkItem(ctx, fmt.Sprintf("cli-batch-task-%d", i), a.CreateWorkItemCommand{Scope: batchScope, Title: fmt.Sprintf("bounded CLI task %d", i), Priority: d.PriorityNormal, Lifecycle: d.WorkItemLifecycleTodo})
		must(e)
		if !work.Value.ContractsEnabled {
			t.Fatal("new Task in signed Namespace was not contract-enabled")
		}
	}
	dropCheckout.Store(true)
	batch := execute(6, "--profile", "executor_cli", "work", "checkout", "--next", "--count", "5", "--outcome", batchScope.OutcomeID.String())
	if !batch.Committed {
		t.Fatal("partial batch lost first accepted result")
	}
	unknown, e := workspace.LoadProfileV2("executor_cli")
	must(e)
	if len(unknown.Local.PendingOperations) != 4 || unknown.Local.PendingOperations[0].State != "sent_unknown" {
		t.Fatal("batch did not preserve second unknown and three prepared intentions")
	}
	for _, intent := range unknown.Local.PendingOperations {
		if intent.Operation != "AcquireNextSignedWorkContract" || intent.Scope.OutcomeID != batchScope.OutcomeID {
			t.Fatal("batch intention lost independent scope")
		}
	}
	entries, e := os.ReadDir(filepath.Join(root, ".wos/profiles/executor_cli/contract"))
	must(e)
	if len(entries) != 1 {
		t.Fatal("failure in second item discarded or duplicated first contract")
	}
	firstPath := filepath.Join(".wos/profiles/executor_cli/contract", entries[0].Name())
	firstRaw, e := workspace.ReadV2(firstPath)
	must(e)
	var firstFile cli.ContractFileV2
	must(cli.DecodeV2Document(firstRaw, &firstFile))
	firstFile.Execution.Progress.Summary = "local edit remains after another contract recovers"
	must(workspace.WriteV2(firstPath, firstFile, signing.Digest(firstRaw)))
	dropCheckout.Store(false)
	recoveredBatch := execute(0, "--profile", "executor_cli", "work", "recover", "--all-pending")
	if !recoveredBatch.Committed {
		t.Fatal("recovery did not reconcile accepted batch")
	}
	currentProfile, e := workspace.LoadProfileV2("executor_cli")
	must(e)
	if len(currentProfile.Local.PendingOperations) != 0 {
		t.Fatal("confirmed materialization retained pending intentions")
	}
	entries, e = os.ReadDir(filepath.Join(root, ".wos/profiles/executor_cli/contract"))
	must(e)
	if len(entries) != 2 {
		t.Fatal("recovery created more than two eligible contracts")
	}
	preserved, e := workspace.ReadV2(firstPath)
	must(e)
	must(cli.DecodeV2Document(preserved, &firstFile))
	if firstFile.Execution.Progress.Summary != "local edit remains after another contract recovers" {
		t.Fatal("recover overwrote edited draft")
	}
	beforeCalls := acquisitionCalls.Load()
	execute(0, "--profile", "executor_cli", "work", "recover", "--all-pending")
	execute(0, "--profile", "executor_cli", "work", "list")
	view, e := firstFile.VerifyIssued(currentProfile)
	must(e)
	shown := execute(0, "--profile", "executor_cli", "work", "show", view.Authority.ContractID, "--for-agent")
	shownJSON, e := json.Marshal(shown.Data)
	must(e)
	if bytes.Contains(shownJSON, []byte("proof")) || bytes.Contains(shownJSON, []byte("private_key_ref")) {
		t.Fatal("initial agent view includes harness proofs or secret references")
	}
	if acquisitionCalls.Load() != beforeCalls {
		t.Fatal("recover/show/list performed a new acquisition")
	}

	// Recreate the still-pending local intention from the materialize-before-remove
	// crash window, then revoke its original agent key. Recovery reads accepted
	// state and verifies issued proof; it must not call acquisition again.
	currentProfile.Local.PendingOperations = []cli.PendingOperationV2{unknown.Local.PendingOperations[0]}
	must(currentProfile.SealBinding([]byte(token)))
	currentRaw, e := workspace.ReadV2(".wos/profiles/executor_cli/profile.yaml")
	must(e)
	must(workspace.WriteV2(".wos/profiles/executor_cli/profile.yaml", currentProfile, signing.Digest(currentRaw)))
	operatorIdentity, e := operator.SigningIdentity(ctx, nil)
	must(e)
	_, e = operator.SigningMutation(ctx, "cli-revoke-original-key", a.SigningSecurityIntent{NamespaceID: scope.NamespaceID, ExpectedNamespaceVersion: operatorIdentity.NamespaceVersion, Operation: "revoke_signing_key", KeyID: currentProfile.Signing.KeyID, ExpectedVersion: 1, Reason: "exercise recovery of accepted state after key revocation"})
	must(e)
	execute(0, "--profile", "executor_cli", "work", "recover", "--all-pending")
	if acquisitionCalls.Load() != beforeCalls {
		t.Fatal("accepted recovery after key retirement called acquisition")
	}
	recovered, e = workspace.LoadProfileV2("executor_cli")
	must(e)
	if len(recovered.Local.PendingOperations) != 0 {
		t.Fatal("accepted historical result remained pending after reconciliation")
	}
	// Tampered local origin is rejected before transmitting any credential.
	current, e := workspace.ReadV2(".wos/profiles/executor_cli/profile.yaml")
	must(e)
	recovered.Binding.PrincipalID = "other-principal"
	must(workspace.WriteV2(".wos/profiles/executor_cli/profile.yaml", recovered, signing.Digest(current)))
	execute(6, "--profile", "executor_cli", "auth", "status")
	if registrationCalls.Load() != beforeRecoveryCalls+1 {
		t.Fatal("tampered profile created another enrollment")
	}
}

// The key was enrolled over the public API before this fixture. Installation
// records only protected references; checkout uses actual authenticated HTTP.
func signedCLIReviewCheckout(t *testing.T, handler http.Handler, scope d.Scope, server d.ServerIdentity, credential ports.Credential, token string, key d.SigningKey, private ed25519.PrivateKey, reviewer *sdk.Client) sdk.CommandResult[a.SignedReviewContractResult] {
	t.Helper()
	ctx := context.Background()
	must := func(e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
	root := t.TempDir()
	t.Setenv("WOS_CLI_REVIEW_TOKEN", token)
	t.Setenv("WOS_CLI_REVIEW_KEY", base64.StdEncoding.EncodeToString(private.Seed()))
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	var drop atomic.Bool
	drop.Store(true)
	var calls atomic.Int32
	endpoint := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/commands/acquire_next_signed_review_contract") {
			calls.Add(1)
			recorded := httptest.NewRecorder()
			handler.ServeHTTP(recorded, r)
			if drop.Load() && recorded.Code < 400 {
				connection, _, e := rw.(http.Hijacker).Hijack()
				if e != nil {
					t.Error(e)
					return
				}
				connection.Close()
				return
			}
			for k, vs := range recorded.Header() {
				rw.Header()[k] = vs
			}
			rw.WriteHeader(recorded.Code)
			_, _ = rw.Write(recorded.Body.Bytes())
			return
		}
		handler.ServeHTTP(rw, r)
	}))
	defer endpoint.Close()
	execute := func(expected int, args ...string) cli.Output {
		t.Helper()
		var out, diagnostics bytes.Buffer
		code := cli.Run(ctx, append(args, "--workspace", root, "--output", "json"), &out, &diagnostics)
		if code != expected {
			t.Fatalf("review CLI %v exit %d expected %d: %s %s", args, code, expected, out.String(), diagnostics.String())
		}
		var result cli.Output
		must(json.Unmarshal(out.Bytes(), &result))
		return result
	}
	execute(0, "init", "--workspace-schema", "2", "--server", endpoint.URL, "--server-id", server.ID.String(), "--namespace", scope.NamespaceID.String(), "--outcome", scope.OutcomeID.String())
	w, e := cli.OpenWorkspace(root)
	must(e)
	defer w.Close()
	profile := cli.ProfileV2{SchemaVersion: 2, Kind: "WOSProfile", Name: "reviewer", Binding: cli.ProfileBindingV2{ServerID: server.ID, ServerOrigin: endpoint.URL, NamespaceID: scope.NamespaceID, PrincipalID: credential.PrincipalID, CredentialID: credential.ID, IssuerKeys: []cli.ProfileIssuerV2{{KeyID: server.IssuerKeyID, PublicKey: server.PublicKey, Fingerprint: server.Fingerprint}}}, Authentication: cli.ProfileAuthenticationV2{CredentialRef: "env:WOS_CLI_REVIEW_TOKEN"}, Signing: cli.ProfileSigningV2{KeyID: key.ID, PrivateKeyRef: "env:WOS_CLI_REVIEW_KEY", PublicKeyFingerprint: key.Fingerprint}, Lease: cli.LeaseConfig{RequestedTTLSeconds: 300}, Output: cli.OutputConfig{DefaultFormat: "json"}, Local: cli.ProfileLocalV2{SchemaVersion: 1, PendingOperations: []cli.PendingOperationV2{}}}
	must(profile.SealBinding([]byte(token)))
	must(w.CreateV2(".wos/profiles/reviewer/profile.yaml", profile))
	execute(6, "review", "checkout", "--next")
	pending, e := w.LoadProfileV2("reviewer")
	must(e)
	if len(pending.Local.PendingOperations) != 1 || pending.Local.PendingOperations[0].State != "sent_unknown" {
		t.Fatal("review lost-response intention discarded")
	}
	frozen := pending.Local.PendingOperations[0]
	before := calls.Load()
	drop.Store(false)
	execute(0, "review", "recover", "--all-pending")
	if calls.Load() != before {
		t.Fatal("review recovery unnecessarily acquired again")
	}
	operation, e := reviewer.ReadSignedState(ctx, a.SignedStateQuery{Scope: scope, Resource: "operation", IdempotencyKey: frozen.IdempotencyKey})
	must(e)
	raw, e := base64.StdEncoding.Strict().DecodeString(operation.OperationPayload)
	must(e)
	var search sdk.CommandResult[a.SignedReviewAcquisition]
	must(json.Unmarshal(raw, &search))
	if !search.Value.Acquired || search.Value.Result == nil {
		t.Fatal("review acquisition absent")
	}
	id := search.Value.Result.Contract.ID
	path := ".wos/profiles/reviewer/contract/" + id.String() + ".yaml"
	document, e := w.ReadV2(path)
	must(e)
	var file cli.ContractFileV2
	must(cli.DecodeV2Document(document, &file))
	if file.Review == nil || file.Review.Decision != "inconclusive" || file.Local.ReviewCaseVersion != signing.Decimal(search.Value.Result.Case.Version) {
		t.Fatal("review file invented approval or lost CAS")
	}
	file.Review.Progress.Summary = "independent reviewer draft preserved"
	must(w.WriteV2(path, file, signing.Digest(document)))
	shown := execute(0, "review", "show", id.String(), "--for-agent")
	agentJSON, e := json.Marshal(shown.Data)
	must(e)
	if bytes.Contains(agentJSON, []byte("proof")) || bytes.Contains(agentJSON, []byte("private_key_ref")) {
		t.Fatal("review agent projection expanded technical secrets/proofs")
	}
	execute(0, "review", "list")
	execute(0, "review", "recover")
	if calls.Load() != before {
		t.Fatal("review show/list/recover acquired fresh work")
	}
	return sdk.CommandResult[a.SignedReviewContractResult]{Value: *search.Value.Result, CommandID: search.CommandID, OutcomeRevision: search.OutcomeRevision}
}
