package server

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

func TestSignedRuntimeWithoutSignerIsNotReadyButRetainsHistory(t *testing.T) {
	forEachRuntimeStorage(t, func(t *testing.T, cfg Config) {
		ctx := context.Background()
		must := func(e error) {
			t.Helper()
			if e != nil {
				t.Fatal(e)
			}
		}
		seed := make([]byte, ed25519.SeedSize)
		seed[0] = 81
		t.Setenv("WOS_TEST_READINESS_ISSUER_SEED", base64.StdEncoding.EncodeToString(seed))
		cfg.Signing.SeedEnv = "WOS_TEST_READINESS_ISSUER_SEED"
		cfg.Auth.Mode = AuthModeAPIToken
		cfg.Auth.BootstrapToken = strings.Repeat("q", 40)
		cfg.Auth.BootstrapNamespaceID = "0199a779-0000-7000-8000-000000000001"
		cfg.Auth.BootstrapNamespaceName = "signed readiness"
		runtime, e := OpenRuntime(cfg)
		must(e)
		defer runtime.Close()
		endpoint := httptest.NewServer(runtime.Handler())
		client, e := sdk.New(endpoint.URL, cfg.Auth.BootstrapToken, endpoint.Client())
		must(e)
		namespace := d.MustParseID(cfg.Auth.BootstrapNamespaceID)
		identity, e := client.SigningIdentity(ctx, nil)
		must(e)
		security := a.SecurityService{Store: runtime.store, Clock: systemClock{}, IDs: uuidV7Generator{}}
		admin, e := security.Authenticate(ctx, cfg.Auth.BootstrapToken)
		must(e)
		must(security.SetGrant(a.WithIdentity(ctx, admin), ports.NamespaceGrant{NamespaceID: namespace, PrincipalID: identity.PrincipalID, Permissions: []ports.Permission{ports.PermissionStateRead, ports.PermissionNamespaceAdmin, ports.PermissionWorkContractAcquire, ports.PermissionSigningKeyEnroll, ports.PermissionOutcomeWrite, ports.PermissionWorkWrite}}))
		identity, e = client.SigningIdentity(ctx, nil)
		must(e)
		policy := d.CredentialPolicy{NamespaceID: namespace, CredentialID: identity.CredentialID, PrincipalID: identity.PrincipalID, AcceptanceFloor: d.AcceptanceDirect, MaxActiveWorkContracts: 3, MaxActiveReviewContracts: 1, PermittedOperations: []string{string(ports.PermissionStateRead), string(ports.PermissionNamespaceAdmin), string(ports.PermissionWorkContractAcquire), string(ports.PermissionSigningKeyEnroll), string(ports.PermissionOutcomeWrite), string(ports.PermissionWorkWrite)}}
		configured, e := client.SigningMutation(ctx, "readiness-credential-policy", a.SigningSecurityIntent{NamespaceID: namespace, ExpectedNamespaceVersion: identity.NamespaceVersion, Operation: "set_credential_policy", CredentialPolicy: &policy})
		must(e)
		public, private, e := ed25519.GenerateKey(rand.Reader)
		must(e)
		enrollment, e := client.SigningMutation(ctx, "readiness-key-enrollment", a.SigningSecurityIntent{NamespaceID: namespace, ExpectedNamespaceVersion: configured.NamespaceVersion, Operation: "create_enrollment", CredentialID: identity.CredentialID, PrincipalID: identity.PrincipalID, ExpectedFingerprint: signing.Digest(public)})
		must(e)
		challenge := enrollment.Enrollment
		proof, e := signing.Sign(signing.KeyEnrollment, signing.EnrollmentPayload{Binding: signing.Binding{ProtocolVersion: 2, ServerID: identity.ServerID, NamespaceID: namespace.String(), PrincipalID: identity.PrincipalID, SignerKeyID: challenge.KeyID.String()}, EnrollmentID: challenge.ID.String(), Nonce: challenge.Nonce, Purpose: challenge.Purpose, PublicKey: base64.StdEncoding.EncodeToString(public), ExpiresAt: challenge.ExpiresAt.UTC().Format(time.RFC3339Nano)}, challenge.KeyID.String(), private)
		must(e)
		registered, e := client.SigningMutation(ctx, "readiness-key-registration", a.SigningSecurityIntent{NamespaceID: namespace, Operation: "register_signing_key", EnrollmentID: challenge.ID, Proof: &proof})
		must(e)
		identity, e = client.SigningIdentity(ctx, nil)
		must(e)
		_, e = client.SigningMutation(ctx, "readiness-namespace-policy", a.SigningSecurityIntent{NamespaceID: namespace, ExpectedNamespaceVersion: identity.NamespaceVersion, Operation: "set_acceptance_policy", AcceptancePolicy: &d.WorkAcceptancePolicy{NamespaceID: namespace, AcceptanceFloor: d.AcceptanceDirect, MaxActiveWorkContracts: 3, MaxActiveReviewContracts: 1}})
		must(e)
		created, e := client.CreateOutcome(ctx, "readiness-outcome", a.CreateOutcomeCommand{NamespaceID: namespace, Title: "Persisted signed history", DesiredState: "History remains readable", Priority: d.PriorityNormal})
		must(e)
		scope := created.Value.Scope()
		for i, phase := range []d.WorkProtocolPhase{d.WorkProtocolDraining, d.WorkProtocolContracts, d.WorkProtocolSignedDraining, d.WorkProtocolSigned} {
			_, e = client.SetNamespaceWorkProtocol(ctx, "readiness-cutover-"+string(phase), a.SetNamespaceWorkProtocolCommand{Scope: scope, ExpectedProtocolVersion: d.Version(i + 1), Phase: phase, WritersDrained: phase == d.WorkProtocolContracts || phase == d.WorkProtocolSigned, Reason: "explicit fixture writer drain"})
			must(e)
		}
		work, e := client.CreateWorkItem(ctx, "readiness-work-task", a.CreateWorkItemCommand{Scope: scope, Title: "No emission without signer", Priority: d.PriorityNormal, Lifecycle: d.WorkItemLifecycleTodo})
		must(e)
		endpoint.Close()
		must(runtime.Close())
		cfg.Signing.SeedEnv = ""
		restored, e := OpenRuntime(cfg)
		must(e)
		defer restored.Close()
		endpoint = httptest.NewServer(restored.Handler())
		defer endpoint.Close()
		for path, expected := range map[string]int{"/readyz": http.StatusServiceUnavailable, "/livez": http.StatusOK} {
			response, e := endpoint.Client().Get(endpoint.URL + path)
			must(e)
			response.Body.Close()
			if response.StatusCode != expected {
				t.Fatalf("%s=%d expected %d", path, response.StatusCode, expected)
			}
		}
		client, e = sdk.New(endpoint.URL, cfg.Auth.BootstrapToken, endpoint.Client())
		must(e)
		trust, e := client.GetSignedTrust(ctx, namespace)
		must(e)
		if trust.Server == nil || trust.Server.ID.String() != identity.ServerID || trust.Protocol == nil || trust.Protocol.Phase != d.WorkProtocolSigned {
			t.Fatal("missing signer changed persisted public identity/protocol")
		}
		preflight, e := client.ReadSignedState(ctx, a.SignedStateQuery{Scope: scope, Resource: "protocol_preflight"})
		must(e)
		if preflight.Readiness == nil || preflight.Readiness.IssuerReady {
			t.Fatal("missing signer claimed issuance readiness")
		}
		_, e = client.AcquireSignedWorkContract(ctx, "readiness-no-signer-acquisition", a.AcquireSignedWorkContractCommand{Scope: scope, WorkItemID: work.Value.ID, ExpectedWorkItemVersion: work.Value.Version, SignerKeyID: registered.Key.ID, TTLSeconds: 300})
		var denied *sdk.Error
		if !errors.As(e, &denied) || denied.Code != string(d.ErrorCodeInvalidConfig) {
			t.Fatalf("new issuance did not explicitly refuse missing signer: %v", e)
		}
	})
}
