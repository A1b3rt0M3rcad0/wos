package memory

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
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
	if err := security.Store.PutGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "issuer", Permissions: []ports.Permission{ports.PermissionCredentialsIssue}}); err != nil {
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
	if err := security.Store.PutGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "worker", Permissions: []ports.Permission{ports.PermissionStateRead}}); err != nil {
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
	if err := security.Store.PutGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "worker", Permissions: []ports.Permission{}}); err != nil {
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

func TestCreateNamespaceRequiresRecoverableAdministration(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	store := New()
	ids := &credentialManagementIDs{next: 100}
	security := application.SecurityService{Store: store, Clock: credentialManagementClock{now: now}, IDs: ids}
	ns := domain.MustParseID("0199f802-0000-7000-8000-000000000001")
	token := "memory-namespace-recovery-token-at-least-32-bytes"
	if err := security.Bootstrap(ctx, ports.Namespace{ID: ns, Name: "Source"}, "operator", token); err != nil {
		t.Fatal(err)
	}
	identity, err := security.Authenticate(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	adminCtx := application.WithIdentity(ctx, identity)
	if err = store.PutGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "operator", Permissions: []ports.Permission{ports.PermissionStateRead, ports.PermissionNamespaceAdmin}}); err != nil {
		t.Fatal(err)
	}
	before, err := store.ReadSecurityAdminVersion(ctx, ns)
	if err != nil {
		t.Fatal(err)
	}
	deniedNamespaceID := domain.MustParseID(fmt.Sprintf("0199f802-0000-7000-8000-%012x", ids.next+1))
	intent := application.AdministrativeIntent{NamespaceID: ns, ExpectedVersion: before, Operation: "create_namespace", NamespaceName: "Unrecoverable"}
	result, secret, err := security.Administer(adminCtx, "create-unrecoverable-namespace-01", intent)
	if memorySecurityErrorCode(err) != domain.ErrorCodeInvalidTransition || secret != "" {
		t.Fatalf("creation without an administrative recovery path should roll back, result=%+v token_present=%t err=%v", result, secret != "", err)
	}
	if result.CreatedNamespace != nil {
		deniedNamespaceID = result.CreatedNamespace.ID
	}
	if _, err = store.ReadSecurityAdminVersion(ctx, deniedNamespaceID); memorySecurityErrorCode(err) != domain.ErrorCodeNotFound {
		t.Fatalf("rejected creation left a Namespace behind: %v", err)
	}
	after, err := store.ReadSecurityAdminVersion(ctx, ns)
	if err != nil || after != before {
		t.Fatalf("rejected creation changed its source Namespace version: before=%d after=%d err=%v", before, after, err)
	}
	if err = store.PutGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "operator", Permissions: []ports.Permission{ports.PermissionStateRead, ports.PermissionNamespaceAdmin, ports.PermissionPermissionsManage}}); err != nil {
		t.Fatal(err)
	}
	intent.NamespaceName = "Recoverable"
	result, secret, err = security.Administer(adminCtx, "create-recoverable-namespace-01", intent)
	if err != nil || secret == "" || result.CreatedNamespace == nil {
		t.Fatalf("creation with a recoverable administrator should succeed: %+v token_present=%t err=%v", result, secret != "", err)
	}
	grant, err := store.GetGrant(ctx, result.CreatedNamespace.ID, "operator")
	if err != nil || !slices.Contains(grant.Permissions, ports.PermissionPermissionsManage) {
		t.Fatalf("created Namespace lacks recoverable administrator permissions: %+v err=%v", grant, err)
	}
	snapshot, err := store.ReadSecurityAdmin(ctx, ns)
	if err != nil {
		t.Fatal(err)
	}
	createdAudit := findMemorySecurityAdminAudit(t, snapshot, "create_namespace", result.CreatedNamespace.ID.String())
	createdAfter := decodeMemorySecurityAuditObject(t, createdAudit.After)
	initialGrant, hasGrant := createdAfter["initial_grant"].(map[string]any)
	initialCredential, hasCredential := createdAfter["initial_credential"].(map[string]any)
	initialPolicy, hasPolicy := initialCredential["effective_policy"].(map[string]any)
	if !hasGrant || !hasCredential || !hasPolicy || initialGrant["permission_compatibility_version"] != float64(0) || initialPolicy["permission_compatibility_version"] != float64(0) {
		t.Fatalf("Namespace creation audit omitted the recoverable grant/policy projection: %s", createdAudit.After)
	}
	auditJSON, err := json.Marshal(snapshot.Audit)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(auditJSON), secret) || strings.Contains(string(auditJSON), "digest") {
		t.Fatal("security audit contains a credential secret or digest field")
	}
}

func TestSecurityAuditProjectionReconstructsGrantAndCredentialChanges(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	store := New()
	security := application.SecurityService{Store: store, Clock: credentialManagementClock{now: now}, IDs: &credentialManagementIDs{next: 200}}
	ns := domain.MustParseID("0199f803-0000-7000-8000-000000000001")
	adminToken := "memory-security-audit-token-at-least-32-bytes"
	if err := security.Bootstrap(ctx, ports.Namespace{ID: ns, Name: "Security audit"}, "operator", adminToken); err != nil {
		t.Fatal(err)
	}
	identity, err := security.Authenticate(ctx, adminToken)
	if err != nil {
		t.Fatal(err)
	}
	adminCtx := application.WithIdentity(ctx, identity)
	agentActor := domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "audit-agent"}
	if err = store.PutGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "agent", Permissions: []ports.Permission{ports.PermissionStateRead}}); err != nil {
		t.Fatal(err)
	}
	allowedOutcome := domain.MustParseID("0199f803-0000-7000-8000-000000000020")
	allowedKey := domain.MustParseID("0199f803-0000-7000-8000-000000000021")
	outcomeService, err := application.NewService(store, credentialManagementClock{now: now}, &credentialManagementIDs{next: 299})
	if err != nil {
		t.Fatal(err)
	}
	createdOutcome, err := outcomeService.CreateOutcome(ctx, domain.CommandContext{PrincipalID: "operator", Actor: domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "test", ID: "operator"}, CommandID: domain.MustParseID("0199f803-0000-7000-8000-000000000030")}, application.CreateOutcomeCommand{NamespaceID: ns, Title: "Scoped Outcome", DesiredState: "Audit captures its scope", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	allowedOutcome = createdOutcome.Value.ID
	privateKey := ed25519.NewKeyFromSeed(make([]byte, ed25519.SeedSize))
	publicKey := privateKey.Public().(ed25519.PublicKey)
	fingerprint, err := signing.Fingerprint(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	keyUnit, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	keyRepo := keyUnit.(ports.SigningIdentityUnitOfWork).SigningIdentity()
	if err = keyRepo.SaveKey(ctx, domain.SigningKey{ID: allowedKey, NamespaceID: ns, PrincipalID: "agent", Purpose: "agent", Algorithm: "Ed25519", PublicKey: base64.StdEncoding.EncodeToString(publicKey), Fingerprint: fingerprint, Version: 1, Status: "active", CreatedAt: now, UpdatedAt: now}, 0); err != nil {
		_ = keyUnit.Rollback()
		t.Fatal(err)
	}
	if err = keyUnit.Commit(); err != nil {
		t.Fatal(err)
	}
	expires := now.Add(4 * time.Hour)
	version, err := store.ReadSecurityAdminVersion(ctx, ns)
	if err != nil {
		t.Fatal(err)
	}
	issued, issuedToken, err := security.Administer(adminCtx, "audit-issue-agent-credential-01", application.AdministrativeIntent{
		NamespaceID: ns, ExpectedVersion: version, Operation: "issue_credential", PrincipalID: "agent", Actor: &agentActor,
		CredentialName: "Audit agent", Permissions: []ports.Permission{ports.PermissionStateRead},
		AllowedOutcomeIDs: []domain.ID{allowedOutcome}, AllowedSigningKeyIDs: []domain.ID{allowedKey}, ExpiresAt: &expires,
	})
	if err != nil || issued.Credential == nil || issuedToken == "" {
		t.Fatalf("issue auditable credential: %+v token_present=%t err=%v", issued, issuedToken != "", err)
	}
	credentialID := issued.Credential.ID
	page, err := security.ListCredentials(adminCtx, ns, ports.CredentialManagementQuery{Limit: 100})
	if err != nil || len(page.Items) != 2 {
		t.Fatalf("list issued credential: %+v err=%v", page, err)
	}
	var current ports.CredentialAdministrativeRecord
	for _, item := range page.Items {
		if item.Credential.ID == credentialID {
			current = item
		}
	}
	if current.Credential.ID.IsZero() {
		t.Fatal("issued credential missing from administrative projection")
	}
	policy := current.EffectivePolicy
	policy.AcceptanceFloor = domain.AcceptanceIndependentReview
	policy.PermissionCompatibilityVersion = domain.PermissionCompatibilityLegacyV1
	policy.AllowedOutcomeIDs = []domain.ID{allowedOutcome}
	policy.AllowedSigningKeyIDs = []domain.ID{allowedKey}
	policy.MaxActiveWorkContracts = 9
	policy.MaxActiveReviewContracts = 4
	version, err = store.ReadSecurityAdminVersion(ctx, ns)
	if err != nil {
		t.Fatal(err)
	}
	policyUpdate, err := security.UpdateCredential(adminCtx, "audit-policy-update-agent-01", application.CredentialUpdateIntent{
		NamespaceID: ns, CredentialID: credentialID, ExpectedNamespaceVersion: version,
		ExpectedMetadataRevision: current.Metadata.MetadataRevision, ExpectedPolicyRevision: current.EffectivePolicy.Version,
		Name: current.Metadata.Name, Description: current.Metadata.Description, ExpiresAt: &current.Credential.ExpiresAt,
		Policy: &policy, Reason: "Record the complete bounded policy",
	})
	if err != nil || policyUpdate.Credential == nil {
		t.Fatalf("update credential policy: %+v %v", policyUpdate, err)
	}
	snapshot, err := store.ReadSecurityAdmin(ctx, ns)
	if err != nil {
		t.Fatal(err)
	}
	policyAudit := findMemorySecurityAdminAudit(t, snapshot, "update_credential", credentialID.String())
	policyAfter := decodeMemorySecurityAuditObject(t, policyAudit.After)
	effectivePolicy, ok := policyAfter["effective_policy"].(map[string]any)
	if !ok {
		t.Fatalf("updated credential audit omitted effective policy: %s", policyAudit.After)
	}
	if effectivePolicy["acceptance_floor"] != string(domain.AcceptanceIndependentReview) || effectivePolicy["permission_compatibility_version"] != float64(0) || policyAfter["grant_permission_compatibility_version"] != float64(0) || effectivePolicy["max_active_work_contracts"] != float64(9) || effectivePolicy["max_active_review_contracts"] != float64(4) || !reflect.DeepEqual(effectivePolicy["allowed_outcome_ids"], []any{allowedOutcome.String()}) || !reflect.DeepEqual(effectivePolicy["allowed_signing_key_ids"], []any{allowedKey.String()}) {
		t.Fatalf("audit omitted policy scope or limits: %s", policyAudit.After)
	}
	if !reflect.DeepEqual(policyAfter["permissions"], []any{string(ports.PermissionStateRead)}) {
		t.Fatalf("audit omitted effective operations: %s", policyAudit.After)
	}

	page, err = security.ListCredentials(adminCtx, ns, ports.CredentialManagementQuery{Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range page.Items {
		if item.Credential.ID == credentialID {
			current = item
		}
	}
	version, err = store.ReadSecurityAdminVersion(ctx, ns)
	if err != nil {
		t.Fatal(err)
	}
	_, err = security.UpdateCredential(adminCtx, "audit-metadata-update-agent-01", application.CredentialUpdateIntent{
		NamespaceID: ns, CredentialID: credentialID, ExpectedNamespaceVersion: version,
		ExpectedMetadataRevision: current.Metadata.MetadataRevision, ExpectedPolicyRevision: current.EffectivePolicy.Version,
		Name: "Renamed audit agent", Description: current.Metadata.Description, ExpiresAt: &current.Credential.ExpiresAt,
		Reason: "Rename without changing authority",
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err = store.ReadSecurityAdmin(ctx, ns)
	if err != nil {
		t.Fatal(err)
	}
	metadataAudit := findMemorySecurityAdminAudit(t, snapshot, "update_credential", credentialID.String())
	metadataBefore := decodeMemorySecurityAuditObject(t, metadataAudit.Before)
	metadataAfter := decodeMemorySecurityAuditObject(t, metadataAudit.After)
	if !reflect.DeepEqual(metadataBefore["effective_policy"], metadataAfter["effective_policy"]) {
		t.Fatalf("metadata-only audit failed to preserve the complete effective policy: before=%s after=%s", metadataAudit.Before, metadataAudit.After)
	}

	version, err = store.ReadSecurityAdminVersion(ctx, ns)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = security.AdministrativeMutation(adminCtx, "audit-revoke-agent-credential-01", ports.SecurityAdminCommand{
		NamespaceID: ns, ExpectedVersion: version, Operation: "revoke_credential", CredentialID: credentialID, Reason: "Retire the agent credential",
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err = store.ReadSecurityAdmin(ctx, ns)
	if err != nil {
		t.Fatal(err)
	}
	revokeAudit := findMemorySecurityAdminAudit(t, snapshot, "revoke_credential", credentialID.String())
	revokeBefore := decodeMemorySecurityAuditObject(t, revokeAudit.Before)
	revokeAfter := decodeMemorySecurityAuditObject(t, revokeAudit.After)
	if revokeBefore["status"] != "active" || revokeAfter["status"] != "revoked" || !reflect.DeepEqual(revokeBefore["effective_policy"], revokeAfter["effective_policy"]) {
		t.Fatalf("revocation audit lost credential policy state: before=%s after=%s", revokeAudit.Before, revokeAudit.After)
	}

	store.mu.Lock()
	grant := store.security.Grants[securityKey(ns, "agent")]
	grant.PermissionCompatibilityVersion = 1
	store.security.Grants[securityKey(ns, "agent")] = grant
	store.mu.Unlock()
	version, err = store.ReadSecurityAdminVersion(ctx, ns)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = security.AdministrativeMutation(adminCtx, "audit-normalize-agent-grant-01", ports.SecurityAdminCommand{
		NamespaceID: ns, ExpectedVersion: version, Operation: "set_grant",
		Grant: &ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "agent", Permissions: []ports.Permission{ports.PermissionStateRead}},
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err = store.ReadSecurityAdmin(ctx, ns)
	if err != nil {
		t.Fatal(err)
	}
	grantAudit := findMemorySecurityAdminAudit(t, snapshot, "set_grant", "agent")
	grantBefore := decodeMemorySecurityAuditObject(t, grantAudit.Before)
	grantAfter := decodeMemorySecurityAuditObject(t, grantAudit.After)
	if grantBefore["permission_compatibility_version"] != float64(1) || grantAfter["permission_compatibility_version"] != float64(0) {
		t.Fatalf("grant audit omitted compatibility transition: before=%s after=%s", grantAudit.Before, grantAudit.After)
	}
}

func memorySecurityErrorCode(err error) domain.ErrorCode {
	code, _ := domain.ErrorCodeOf(err)
	return code
}

func findMemorySecurityAdminAudit(t *testing.T, snapshot ports.SecurityAdminSnapshot, operation, target string) ports.SecurityAudit {
	t.Helper()
	for _, entry := range snapshot.Audit {
		if entry.Operation == operation && entry.Target == target {
			return entry
		}
	}
	t.Fatalf("audit entry %q for %q not found", operation, target)
	return ports.SecurityAudit{}
}

func decodeMemorySecurityAuditObject(t *testing.T, raw json.RawMessage) map[string]any {
	t.Helper()
	var state map[string]any
	if err := json.Unmarshal(raw, &state); err != nil {
		t.Fatalf("decode security audit state %q: %v", raw, err)
	}
	return state
}
