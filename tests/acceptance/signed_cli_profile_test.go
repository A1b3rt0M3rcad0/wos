package acceptance_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
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
	endpoint := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
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
		must(json.Unmarshal(out.Bytes(), &result))
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
