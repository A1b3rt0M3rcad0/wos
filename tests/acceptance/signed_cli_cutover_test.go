package acceptance_test

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	cli "github.com/A1b3rt0M3rcad0/wos/packages/wos-cli"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func signedCLICutoverJourney(t *testing.T, handler http.Handler, security *a.SecurityService, adminCtx context.Context, operator *sdk.Client, scope d.Scope, server d.ServerIdentity, adminToken string) {
	t.Helper()
	ctx := context.Background()
	must := func(e error) {
		t.Helper()
		if e != nil {
			_, file, line, _ := runtime.Caller(1)
			t.Fatalf("%s:%d: %v", file, line, e)
		}
	}
	own, e := operator.SigningIdentity(ctx, nil)
	must(e)
	permissions := []ports.Permission{ports.PermissionStateRead, ports.PermissionNamespaceAdmin, ports.PermissionSigningKeyEnroll, ports.PermissionActorDelegate, ports.PermissionOutcomeWrite, ports.PermissionWorkWrite, ports.PermissionPlanningWrite, ports.PermissionAssessmentWrite, ports.PermissionWorkContractRevoke}
	must(security.SetGrant(adminCtx, ports.NamespaceGrant{NamespaceID: scope.NamespaceID, PrincipalID: own.PrincipalID, Permissions: permissions}))
	own, e = operator.SigningIdentity(ctx, nil)
	must(e)
	policy := d.CredentialPolicy{NamespaceID: scope.NamespaceID, CredentialID: own.CredentialID, PrincipalID: own.PrincipalID, AcceptanceFloor: d.AcceptanceIndependentReview, MaxActiveWorkContracts: 3, MaxActiveReviewContracts: 1}
	for _, permission := range permissions {
		policy.PermittedOperations = append(policy.PermittedOperations, string(permission))
	}
	configured, e := operator.SigningMutation(ctx, "cutover-cli-admin-policy", a.SigningSecurityIntent{NamespaceID: scope.NamespaceID, ExpectedNamespaceVersion: own.NamespaceVersion, Operation: "set_credential_policy", CredentialPolicy: &policy})
	must(e)
	public, private, e := ed25519.GenerateKey(rand.Reader)
	must(e)
	enrolled, e := operator.SigningMutation(ctx, "cutover-cli-admin-enrollment", a.SigningSecurityIntent{NamespaceID: scope.NamespaceID, ExpectedNamespaceVersion: configured.NamespaceVersion, Operation: "create_enrollment", CredentialID: own.CredentialID, PrincipalID: own.PrincipalID, ExpectedFingerprint: signing.Digest(public)})
	must(e)
	challenge := enrolled.Enrollment
	proof, e := signing.Sign(signing.KeyEnrollment, signing.EnrollmentPayload{Binding: signing.Binding{ProtocolVersion: 2, ServerID: server.ID.String(), NamespaceID: scope.NamespaceID.String(), PrincipalID: own.PrincipalID, SignerKeyID: challenge.KeyID.String()}, EnrollmentID: challenge.ID.String(), Nonce: challenge.Nonce, Purpose: challenge.Purpose, PublicKey: base64.StdEncoding.EncodeToString(public), ExpiresAt: challenge.ExpiresAt.UTC().Format(time.RFC3339Nano)}, challenge.KeyID.String(), private)
	must(e)
	registered, e := operator.SigningMutation(ctx, "cutover-cli-admin-registration", a.SigningSecurityIntent{NamespaceID: scope.NamespaceID, Operation: "register_signing_key", EnrollmentID: challenge.ID, Proof: &proof})
	must(e)
	own, e = operator.SigningIdentity(ctx, nil)
	must(e)
	_, e = operator.SigningMutation(ctx, "cutover-cli-namespace-floor", a.SigningSecurityIntent{NamespaceID: scope.NamespaceID, ExpectedNamespaceVersion: own.NamespaceVersion, Operation: "set_acceptance_policy", AcceptancePolicy: &d.WorkAcceptancePolicy{NamespaceID: scope.NamespaceID, AcceptanceFloor: d.AcceptanceIndependentReview, MaxActiveWorkContracts: 3, MaxActiveReviewContracts: 1}})
	must(e)
	old, e := operator.GetNamespaceWorkProtocol(ctx, scope.NamespaceID)
	must(e)
	drain, e := operator.SetNamespaceWorkProtocol(ctx, "cutover-cli-drain-legacy", a.SetNamespaceWorkProtocolCommand{Scope: scope, ExpectedProtocolVersion: old.Version, Phase: d.WorkProtocolDraining, Reason: "retire legacy fixture writers"})
	must(e)
	_, e = operator.SetNamespaceWorkProtocol(ctx, "cutover-cli-activate-v1", a.SetNamespaceWorkProtocolCommand{Scope: scope, ExpectedProtocolVersion: drain.Value.Protocol.Version, Phase: d.WorkProtocolContracts, WritersDrained: true, Reason: "legacy writers explicitly drained"})
	must(e)
	var loseActivation atomic.Bool
	var activationCalls atomic.Int32
	endpoint := httptest.NewServer(http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/commands/set_namespace_work_protocol") {
			activationCalls.Add(1)
			recorded := httptest.NewRecorder()
			handler.ServeHTTP(recorded, r)
			if loseActivation.Load() && recorded.Code < 400 {
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
	root := t.TempDir()
	t.Setenv("WOS_CLI_CUTOVER_TOKEN", adminToken)
	t.Setenv("WOS_CLI_CUTOVER_KEY", base64.StdEncoding.EncodeToString(private.Seed()))
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	run := func(expected int, args ...string) cli.Output {
		t.Helper()
		var out, diagnostic bytes.Buffer
		code := cli.Run(ctx, append(args, "--workspace", root, "--output", "json"), &out, &diagnostic)
		if code != expected {
			t.Fatalf("cutover CLI %v exit %d expected %d: %s %s", args, code, expected, out.String(), diagnostic.String())
		}
		var result cli.Output
		if e := json.Unmarshal(out.Bytes(), &result); e != nil {
			t.Fatalf("cutover CLI %v invalid output: %v output=%q diagnostic=%q", args, e, out.String(), diagnostic.String())
		}
		return result
	}
	run(0, "init", "--workspace-schema", "2", "--server", endpoint.URL, "--server-id", server.ID.String(), "--namespace", scope.NamespaceID.String(), "--outcome", scope.OutcomeID.String())
	w, e := cli.OpenWorkspace(root)
	must(e)
	defer w.Close()
	profile := cli.ProfileV2{SchemaVersion: 2, Kind: "WOSProfile", Name: "operator", Binding: cli.ProfileBindingV2{ServerID: server.ID, ServerOrigin: endpoint.URL, NamespaceID: scope.NamespaceID, PrincipalID: own.PrincipalID, CredentialID: own.CredentialID, IssuerKeys: []cli.ProfileIssuerV2{{KeyID: server.IssuerKeyID, PublicKey: server.PublicKey, Fingerprint: server.Fingerprint}}}, Authentication: cli.ProfileAuthenticationV2{CredentialRef: "env:WOS_CLI_CUTOVER_TOKEN"}, Signing: cli.ProfileSigningV2{KeyID: registered.Key.ID, PrivateKeyRef: "env:WOS_CLI_CUTOVER_KEY", PublicKeyFingerprint: registered.Key.Fingerprint}, Lease: cli.LeaseConfig{RequestedTTLSeconds: 300}, Output: cli.OutputConfig{DefaultFormat: "json"}, Local: cli.ProfileLocalV2{SchemaVersion: 1, PendingOperations: []cli.PendingOperationV2{}}}
	must(profile.SealBinding([]byte(adminToken)))
	must(w.CreateV2(".wos/profiles/operator/profile.yaml", profile))
	phase, e := operator.GetNamespaceWorkProtocol(ctx, scope.NamespaceID)
	must(e)
	version := func(v d.Version) string { return strconv.FormatUint(uint64(v), 10) }
	run(0, "protocol", "set", "--version", version(phase.Version), "--phase", "draining_to_signed_v2", "--reason", "retire v1 fixture writers")
	run(0, "protocol", "preflight")
	phase, e = operator.GetNamespaceWorkProtocol(ctx, scope.NamespaceID)
	must(e)
	loseActivation.Store(true)
	run(6, "protocol", "set", "--version", version(phase.Version), "--phase", "signed_contracts_v2", "--writers-drained", "--reason", "v1 fixture writers and SQL credentials retired")
	pending, e := w.LoadProfileV2("operator")
	must(e)
	if len(pending.Local.PendingOperations) != 1 || pending.Local.PendingOperations[0].State != "sent_unknown" {
		t.Fatal("uncertain activation lost original intention")
	}
	before := activationCalls.Load()
	loseActivation.Store(false)
	recovered := run(0, "protocol", "recover")
	if !recovered.Committed || activationCalls.Load() != before {
		t.Fatal("activation recovery executed another protocol change")
	}
	final, e := operator.GetNamespaceWorkProtocol(ctx, scope.NamespaceID)
	must(e)
	if final.Phase != d.WorkProtocolSigned || final.WriterEpoch != 2 || final.Version != phase.Version+1 {
		t.Fatal("cutover recovery advanced wrong protocol state")
	}
	cleared, e := w.LoadProfileV2("operator")
	must(e)
	if len(cleared.Local.PendingOperations) != 0 {
		t.Fatal("accepted activation retained pending intention")
	}
}
