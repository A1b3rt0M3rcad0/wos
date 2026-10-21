package memory

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type credentialManagementClock struct{ now time.Time }

func (c credentialManagementClock) Now() time.Time { return c.now }

type credentialManagementIDs struct{ next uint64 }

func (g *credentialManagementIDs) NewID() (domain.ID, error) {
	g.next++
	return domain.ParseID(fmt.Sprintf("0199f801-0000-7000-8000-%012x", g.next))
}

func TestCredentialIssuanceIsAtomicIdempotentAndSecretFree(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	store := New()
	security := application.SecurityService{Store: store, Clock: credentialManagementClock{now: now}, IDs: &credentialManagementIDs{}}
	ns := domain.MustParseID("0199f800-0000-7000-8000-000000000001")
	adminToken := "memory-credential-admin-token-with-32-bytes"
	if err := security.Bootstrap(ctx, ports.Namespace{ID: ns, Name: "credential-management"}, "admin", adminToken); err != nil {
		t.Fatal(err)
	}
	admin, err := security.Authenticate(ctx, adminToken)
	if err != nil {
		t.Fatal(err)
	}
	adminCtx := application.WithIdentity(ctx, admin)
	if err := security.SetGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "issuer", Permissions: []ports.Permission{ports.PermissionCredentialsIssue}}); err != nil {
		t.Fatal(err)
	}
	_, issuerToken, err := security.IssueCredential(adminCtx, ns, "issuer", domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "issuer"}, now.Add(12*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	issuerIdentity, err := security.Authenticate(ctx, issuerToken)
	if err != nil {
		t.Fatal(err)
	}
	issuerCtx := application.WithIdentity(ctx, issuerIdentity)
	if err := security.SetGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "worker", Permissions: []ports.Permission{ports.PermissionStateRead}}); err != nil {
		t.Fatal(err)
	}
	// Keep the issuer's actor identity so this exercises credentials:issue
	// without also requiring actor:delegate.
	workerActor := issuerIdentity.Actor
	version, err := store.ReadSecurityAdminVersion(ctx, ns)
	if err != nil {
		t.Fatal(err)
	}
	before, err := store.ReadSecurityAdmin(ctx, ns)
	if err != nil {
		t.Fatal(err)
	}
	failedCommand := ports.SecurityAdminCommand{
		NamespaceID: ns, ExpectedVersion: version, Operation: "issue_credential",
		Credential:     &ports.Credential{NamespaceID: ns, PrincipalID: "worker", Actor: workerActor, ExpiresAt: now.Add(time.Hour)},
		CredentialName: "must roll back", CredentialPermissions: []ports.Permission{ports.PermissionStateRead},
		AllowedOutcomeIDs: []domain.ID{domain.MustParseID("0199f800-0000-7000-8000-000000000099")},
	}
	if _, _, err := security.AdministrativeMutation(issuerCtx, "memory-credential-denied-0001", failedCommand); err == nil {
		t.Fatal("invalid Outcome scope was issued")
	}
	afterFailure, err := store.ReadSecurityAdmin(ctx, ns)
	if err != nil {
		t.Fatal(err)
	}
	if afterFailure.NamespaceVersion != before.NamespaceVersion || len(afterFailure.Credentials) != len(before.Credentials) || len(afterFailure.CredentialMetadata) != len(before.CredentialMetadata) || len(afterFailure.Audit) != len(before.Audit) {
		t.Fatalf("denied issuance partially committed: before=%+v after=%+v", before, afterFailure)
	}
	store.mu.Lock()
	_, failedReceiptExists := store.security.Receipts[securityKey(ns, "issuer")+"/memory-credential-denied-0001"]
	store.mu.Unlock()
	if failedReceiptExists {
		t.Fatal("denied issuance persisted a receipt")
	}

	version, err = store.ReadSecurityAdminVersion(ctx, ns)
	if err != nil {
		t.Fatal(err)
	}
	result, token, err := security.IssueCredentialWithIdempotencyKey(issuerCtx, "memory-credential-issued-0001", version, ns, "worker", workerActor, now.Add(time.Hour))
	if err != nil || token == "" || result.Credential == nil || result.CredentialMetadata == nil {
		t.Fatalf("issuance failed: result=%+v token_present=%t err=%v", result, token != "", err)
	}
	if result.CredentialMetadata.Name != "Issued credential" || result.CredentialMetadata.CreatedByPrincipalID != "issuer" {
		t.Fatalf("unexpected safe metadata: %+v", result.CredentialMetadata)
	}
	policyUnit, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	policy, err := policyUnit.(ports.SigningIdentityUnitOfWork).SigningIdentity().CredentialPolicy(ctx, ns, result.Credential.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := policyUnit.Rollback(); err != nil {
		t.Fatal(err)
	}
	if len(policy.PermittedOperations) != 1 || policy.PermittedOperations[0] != string(ports.PermissionStateRead) || len(policy.AllowedOutcomeIDs) != 0 || len(policy.AllowedSigningKeyIDs) != 0 {
		t.Fatalf("omitted scopes or frozen grant differ: %+v", policy)
	}
	if policy.PermitsKey(domain.MustParseID("0199f800-0000-7000-8000-000000000098")) {
		t.Fatal("empty signing-key allowlist unexpectedly permits a key")
	}
	replayed, replayToken, err := security.IssueCredentialWithIdempotencyKey(issuerCtx, "memory-credential-issued-0001", version, ns, "worker", workerActor, now.Add(time.Hour))
	if err != nil || !replayed.IdempotentReplay || !replayed.TokenOmitted || replayToken != "" || replayed.Credential == nil || replayed.Credential.ID != result.Credential.ID {
		t.Fatalf("idempotent issuance replay was not safe: result=%+v token_present=%t err=%v", replayed, replayToken != "", err)
	}
	// A lost response can be retried after the target Principal's grant changes.
	// The authorized issuer still needs the safe receipt to identify the key for
	// explicit revocation; replay must not re-prepare a policy from the new grant.
	if err := security.SetGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "worker", Permissions: []ports.Permission{}}); err != nil {
		t.Fatal(err)
	}
	replayedAfterGrantChange, tokenAfterGrantChange, err := security.IssueCredentialWithIdempotencyKey(issuerCtx, "memory-credential-issued-0001", version, ns, "worker", workerActor, now.Add(time.Hour))
	if err != nil || tokenAfterGrantChange != "" || !replayedAfterGrantChange.IdempotentReplay || replayedAfterGrantChange.Credential == nil || replayedAfterGrantChange.Credential.ID != result.Credential.ID {
		t.Fatalf("receipt was not recoverable after target grant change: result=%+v token_present=%t err=%v", replayedAfterGrantChange, tokenAfterGrantChange != "", err)
	}
	final, err := store.ReadSecurityAdmin(ctx, ns)
	if err != nil {
		t.Fatal(err)
	}
	if final.NamespaceVersion != version+1 || len(final.Credentials) != len(before.Credentials)+1 || len(final.CredentialMetadata) != len(before.CredentialMetadata)+1 || len(final.Audit) != len(before.Audit)+1 {
		t.Fatalf("replay duplicated issuance records: %+v", final)
	}
	encoded, err := json.Marshal(final)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), token) || strings.Contains(string(encoded), result.Credential.Digest) {
		t.Fatal("administrative snapshot contains bearer material")
	}
	store.mu.Lock()
	receipt := store.security.Receipts[securityKey(ns, "issuer")+"/memory-credential-issued-0001"]
	store.mu.Unlock()
	if strings.Contains(string(receipt.Payload), token) || strings.Contains(string(receipt.Payload), result.Credential.Digest) {
		t.Fatal("durable receipt contains bearer material")
	}
}
