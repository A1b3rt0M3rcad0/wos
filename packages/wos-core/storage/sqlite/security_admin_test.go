package sqlite

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestSecurityAdministrationCASReceiptsSecretsAndRevocation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "admin.db")
	store := openTestStore(t, path)
	clock := sqliteFixedClock{now: time.Now().UTC()}
	ids := &sqliteSequenceIDs{prefix: "0199d099", next: 1}
	security := application.SecurityService{Store: store, Clock: clock, IDs: ids}
	ns := testID("0199d099-0000-7000-8000-000000000010")
	token := "security-admin-token-32-characters-long"
	if err := security.Bootstrap(context.Background(), ports.Namespace{ID: ns, Name: "Admin"}, "operator", token); err != nil {
		t.Fatal(err)
	}
	identity, err := security.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	ctx := application.WithIdentity(context.Background(), identity)
	command := ports.SecurityAdminCommand{NamespaceID: ns, ExpectedVersion: 1, Operation: "set_grant", Grant: &ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "agent", Permissions: []ports.Permission{ports.PermissionStateRead}}}
	result, _, err := security.AdministrativeMutation(ctx, "admin-grant-key-0001", command)
	if err != nil || result.NamespaceVersion != 2 {
		t.Fatalf("grant: %+v %v", result, err)
	}
	replay, _, err := security.AdministrativeMutation(ctx, "admin-grant-key-0001", command)
	if err != nil || !replay.IdempotentReplay || replay.NamespaceVersion != 2 {
		t.Fatalf("replay: %+v %v", replay, err)
	}
	command.Grant.PrincipalID = "different"
	if _, _, err = security.AdministrativeMutation(ctx, "admin-grant-key-0001", command); securityErrorCode(err) != domain.ErrorCodeIdempotencyConflict {
		t.Fatalf("different intent: %v", err)
	}
	if _, _, err = security.AdministrativeMutation(ctx, "admin-stale-key-0001", command); securityErrorCode(err) != domain.ErrorCodeVersionConflict {
		t.Fatalf("stale grant: %v", err)
	}
	credential := ports.SecurityAdminCommand{NamespaceID: ns, ExpectedVersion: 2, Operation: "issue_credential", Credential: &ports.Credential{NamespaceID: ns, PrincipalID: "agent", Actor: domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "agent"}, ExpiresAt: clock.Now().Add(time.Hour)}}
	issued, secret, err := security.AdministrativeMutation(ctx, "admin-issue-key-0001", credential)
	if err != nil || secret == "" || issued.NamespaceVersion != 3 {
		t.Fatalf("issue: %+v %v", issued, err)
	}
	if _, err = security.Authenticate(context.Background(), secret); err != nil {
		t.Fatal(err)
	}
	policy, err := store.CredentialPolicy(context.Background(), ns, issued.Credential.ID)
	if err != nil || !reflect.DeepEqual(policy.PermittedOperations, []string{string(ports.PermissionStateRead)}) {
		t.Fatalf("issued credential did not receive its frozen restrictive policy: %+v %v", policy, err)
	}
	if issued.CredentialMetadata == nil || issued.CredentialMetadata.Name != "Issued credential" || issued.CredentialMetadata.PolicyRevision != policy.Version {
		t.Fatalf("issuance metadata missing or inconsistent: %+v", issued.CredentialMetadata)
	}
	replay, hidden, err := security.AdministrativeMutation(ctx, "admin-issue-key-0001", credential)
	if err != nil || hidden != "" || !replay.TokenOmitted || replay.Credential.ID != issued.Credential.ID {
		t.Fatalf("secret replay: %+v %v", replay, err)
	}
	// Competing physical connections use the same Namespace version guard.
	second := openTestStore(t, path)
	other := application.SecurityService{Store: second, Clock: clock, IDs: ids}
	var wait sync.WaitGroup
	results := make(chan error, 2)
	for i, s := range []application.SecurityService{security, other} {
		wait.Add(1)
		go func(i int, s application.SecurityService) {
			defer wait.Done()
			principal := []string{"human-one", "human-two"}[i]
			_, _, err := s.AdministrativeMutation(ctx, "concurrent-grant-"+principal, ports.SecurityAdminCommand{NamespaceID: ns, ExpectedVersion: 3, Operation: "set_grant", Grant: &ports.NamespaceGrant{NamespaceID: ns, PrincipalID: principal, Permissions: []ports.Permission{ports.PermissionStateRead}}})
			results <- err
		}(i, s)
	}
	wait.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		} else if securityErrorCode(err) != domain.ErrorCodeVersionConflict && securityErrorCode(err) != domain.ErrorCodeTransactionConflict {
			t.Fatal(err)
		}
	}
	if success != 1 {
		t.Fatalf("concurrent commits: %d", success)
	}
	snapshot, err := security.AdministrativeSnapshot(ctx, ns)
	if err != nil || snapshot.NamespaceVersion != 4 || len(snapshot.Audit) != 3 || len(snapshot.Credentials) != 2 {
		t.Fatalf("audit: %+v %v", snapshot, err)
	}
	revoked, _, err := security.AdministrativeMutation(ctx, "admin-revoke-key-001", ports.SecurityAdminCommand{NamespaceID: ns, ExpectedVersion: 4, Operation: "revoke_credential", CredentialID: issued.Credential.ID})
	if err != nil || revoked.NamespaceVersion != 5 {
		t.Fatal(err)
	}
	if _, err = security.Authenticate(context.Background(), secret); err == nil {
		t.Fatal("revoked credential authenticated")
	}
	if err = store.Close(); err != nil {
		t.Fatal(err)
	}
	restarted := openTestStore(t, path)
	security.Store = restarted
	snapshot, err = security.AdministrativeSnapshot(ctx, ns)
	if err != nil || snapshot.NamespaceVersion != 5 || len(snapshot.Audit) != 4 {
		t.Fatalf("restart: %+v %v", snapshot, err)
	}
	// Current permission must be checked before revealing the old receipt.
	if err = security.SetGrant(ctx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "operator", Permissions: []ports.Permission{ports.PermissionStateRead}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err = security.AdministrativeMutation(ctx, "admin-issue-key-0001", credential); securityErrorCode(err) != domain.ErrorCodeForbidden {
		t.Fatalf("revoked administration replay: %v", err)
	}
}

func TestCredentialIssueIsAtomicAndWorksWithIssueOnlyAuthority(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "atomic-credential.db"))
	clock := sqliteFixedClock{now: time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)}
	security := application.SecurityService{Store: store, Clock: clock, IDs: &sqliteSequenceIDs{prefix: "0199d088", next: 1}}
	ns := testID("0199d088-0000-7000-8000-000000000010")
	adminToken := "atomic-credential-admin-token-at-least-32"
	if err := security.Bootstrap(context.Background(), ports.Namespace{ID: ns, Name: "Atomic credential issuance"}, "admin", adminToken); err != nil {
		t.Fatal(err)
	}
	adminIdentity, err := security.Authenticate(context.Background(), adminToken)
	if err != nil {
		t.Fatal(err)
	}
	adminCtx := application.WithIdentity(context.Background(), adminIdentity)
	if err = security.SetGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "issuer", Permissions: []ports.Permission{ports.PermissionCredentialsIssue}}); err != nil {
		t.Fatal(err)
	}
	if err = security.SetGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "target", Permissions: []ports.Permission{ports.PermissionStateRead}}); err != nil {
		t.Fatal(err)
	}
	issuerActor := domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "test", ID: "issuer"}
	issuerCredential, issuerToken, err := security.IssueCredential(adminCtx, ns, "issuer", issuerActor, clock.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	issuerIdentity, err := security.Authenticate(context.Background(), issuerToken)
	if err != nil {
		t.Fatal(err)
	}
	issuerCtx := application.WithIdentity(context.Background(), issuerIdentity)
	if err = security.Require(issuerCtx, ns, ports.PermissionCredentialsIssue); err != nil {
		policy, _ := store.CredentialPolicy(context.Background(), ns, issuerCredential.ID)
		t.Fatalf("issued issue-only credential cannot exercise its frozen permission: policy=%+v identity=%+v err=%v", policy, issuerIdentity, err)
	}
	if _, err = security.AdministrativeSnapshot(issuerCtx, ns); securityErrorCode(err) != domain.ErrorCodeForbidden {
		t.Fatalf("credentials:issue-only Principal unexpectedly read administration: %v", err)
	}
	// Inject a failure after credential and policy inserts. The transaction must
	// leave no credential, policy, metadata, namespace revision, audit, or receipt.
	if _, err = store.db.Exec(`CREATE TRIGGER reject_credential_metadata BEFORE INSERT ON credential_metadata BEGIN SELECT RAISE(ABORT, 'injected metadata failure'); END`); err != nil {
		t.Fatal(err)
	}
	issue := ports.SecurityAdminCommand{NamespaceID: ns, Operation: "issue_credential", Credential: &ports.Credential{NamespaceID: ns, PrincipalID: "target", Actor: issuerActor, ExpiresAt: clock.Now().Add(time.Hour)}}
	beforeVersion, err := store.ReadSecurityAdminVersion(context.Background(), ns)
	if err != nil {
		t.Fatal(err)
	}
	issue.ExpectedVersion = beforeVersion
	if _, _, err = security.AdministrativeMutation(issuerCtx, "atomic-credential-retry-0001", issue); err == nil {
		t.Fatal("injected metadata failure unexpectedly committed")
	}
	assertCounts := func(expectedCredentials, expectedPolicies, expectedMetadata, expectedAudit, expectedReceipts int) {
		t.Helper()
		var credentials, policies, metadata, audit, receipts int
		queries := []struct {
			query string
			dst   *int
		}{
			{`SELECT count(*) FROM credentials WHERE namespace_id=? AND principal_id='target'`, &credentials},
			{`SELECT count(*) FROM credential_policies WHERE namespace_id=? AND principal_id='target'`, &policies},
			{`SELECT count(*) FROM credential_metadata WHERE namespace_id=?`, &metadata},
			{`SELECT count(*) FROM security_audit WHERE namespace_id=? AND operation='issue_credential'`, &audit},
			{`SELECT count(*) FROM security_admin_results WHERE namespace_id=? AND idempotency_key='atomic-credential-retry-0001'`, &receipts},
		}
		for _, q := range queries {
			if err := store.db.QueryRow(q.query, ns.String()).Scan(q.dst); err != nil {
				t.Fatal(err)
			}
		}
		if credentials != expectedCredentials || policies != expectedPolicies || metadata != expectedMetadata || audit != expectedAudit || receipts != expectedReceipts {
			t.Fatalf("atomic counts credentials/policies/metadata/audit/receipts=%d/%d/%d/%d/%d, want %d/%d/%d/%d/%d", credentials, policies, metadata, audit, receipts, expectedCredentials, expectedPolicies, expectedMetadata, expectedAudit, expectedReceipts)
		}
	}
	assertCounts(0, 0, 2, 1, 0) // the bootstrap and issuer metadata already exist
	afterVersion, err := store.ReadSecurityAdminVersion(context.Background(), ns)
	if err != nil || afterVersion != beforeVersion {
		t.Fatalf("failed issuance changed Namespace version: %d => %d (%v)", beforeVersion, afterVersion, err)
	}
	if _, err = store.db.Exec(`DROP TRIGGER reject_credential_metadata`); err != nil {
		t.Fatal(err)
	}
	result, secret, err := security.AdministrativeMutation(issuerCtx, "atomic-credential-retry-0001", issue)
	if err != nil || secret == "" || result.Credential == nil || result.CredentialMetadata == nil {
		t.Fatalf("retry after rollback: %+v %v", result, err)
	}
	assertCounts(1, 1, 3, 2, 1)
	replay, hidden, err := security.AdministrativeMutation(issuerCtx, "atomic-credential-retry-0001", issue)
	if err != nil || hidden != "" || !replay.IdempotentReplay || !replay.TokenOmitted {
		t.Fatalf("replay leaked token or failed: %+v %v", replay, err)
	}
	assertCounts(1, 1, 3, 2, 1)
	var receipt string
	if err = store.db.QueryRow(`SELECT result_json FROM security_admin_results WHERE namespace_id=? AND idempotency_key='atomic-credential-retry-0001'`, ns.String()).Scan(&receipt); err != nil {
		t.Fatal(err)
	}
	var audit string
	if err = store.db.QueryRow(`SELECT target FROM security_audit WHERE namespace_id=? AND operation='issue_credential' ORDER BY namespace_version DESC LIMIT 1`, ns.String()).Scan(&audit); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(receipt, secret) || strings.Contains(receipt, "digest") || strings.Contains(audit, secret) {
		t.Fatal("bearer token or credential digest entered durable receipt/audit")
	}
	if _, err = security.Authenticate(context.Background(), secret); err != nil {
		t.Fatalf("issued retry token did not authenticate: %v", err)
	}
	if err = security.SetGrant(adminCtx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "target", Permissions: []ports.Permission{}}); err != nil {
		t.Fatal(err)
	}
	replayAfterGrantChange, hiddenAfterGrantChange, err := security.AdministrativeMutation(issuerCtx, "atomic-credential-retry-0001", issue)
	if err != nil || hiddenAfterGrantChange != "" || !replayAfterGrantChange.IdempotentReplay || replayAfterGrantChange.Credential == nil || replayAfterGrantChange.Credential.ID != result.Credential.ID {
		t.Fatalf("receipt replay failed after target grant change: %+v %v", replayAfterGrantChange, err)
	}
	if _, directToken, directErr := security.IssueCredential(issuerCtx, ns, "target", issuerActor, clock.Now().Add(time.Hour)); directErr != nil || directToken == "" {
		t.Fatalf("credentials:issue-only Principal could not use the legacy atomic issuance entry point: %v", directErr)
	}
	_ = issuerCredential
}

func securityErrorCode(err error) domain.ErrorCode { code, _ := domain.ErrorCodeOf(err); return code }

func TestNamespaceCreationCopiesOnlyGrantedPermissions(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "namespace.db"))
	security := application.SecurityService{Store: store, Clock: sqliteFixedClock{now: time.Now().UTC()}, IDs: &sqliteSequenceIDs{prefix: "0199d093", next: 1}}
	ns := testID("0199d093-0000-7000-8000-000000000010")
	token := "namespace-creation-credential-32-chars"
	if err := security.Bootstrap(context.Background(), ports.Namespace{ID: ns, Name: "Source"}, "operator", token); err != nil {
		t.Fatal(err)
	}
	identity, _ := security.Authenticate(context.Background(), token)
	ctx := application.WithIdentity(context.Background(), identity)
	permissions := []ports.Permission{ports.PermissionStateRead, ports.PermissionNamespaceAdmin, ports.PermissionPermissionsManage}
	if err := security.SetGrant(ctx, ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "operator", Permissions: permissions}); err != nil {
		t.Fatal(err)
	}
	intent := application.AdministrativeIntent{NamespaceID: ns, ExpectedVersion: 1, Operation: "create_namespace", NamespaceName: "Created"}
	result, createdToken, err := security.Administer(ctx, "create-namespace-key-0001", intent)
	if err != nil || createdToken == "" || result.CreatedNamespace == nil {
		t.Fatalf("create namespace: %+v %v", result, err)
	}
	grant, err := store.GetGrant(context.Background(), result.CreatedNamespace.ID, "operator")
	if err != nil || len(grant.Permissions) != 3 {
		t.Fatalf("permission escalation: %+v %v", grant, err)
	}
	createdIdentity, err := security.Authenticate(context.Background(), createdToken)
	if err != nil {
		t.Fatal(err)
	}
	if err = security.Require(application.WithIdentity(context.Background(), createdIdentity), ns, ports.PermissionStateRead); securityErrorCode(err) != domain.ErrorCodeForbidden {
		t.Fatal("new credential crossed original scope", err)
	}
	replay, hidden, err := security.Administer(ctx, "create-namespace-key-0001", intent)
	if err != nil || hidden != "" || !replay.IdempotentReplay || replay.CreatedNamespace.ID != result.CreatedNamespace.ID {
		t.Fatalf("namespace replay: %+v %v", replay, err)
	}
}

type revokeBeforeTransaction struct {
	store  ports.TransactionManager
	before func()
}

func (t *revokeBeforeTransaction) Begin(ctx context.Context) (ports.UnitOfWork, error) {
	if t.before != nil {
		hook := t.before
		t.before = nil
		hook()
	}
	return t.store.Begin(ctx)
}
func TestPermissionRevokedWhileAwaitingTransactionCannotMutate(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "transaction-auth.db"))
	clock := sqliteFixedClock{now: time.Now().UTC()}
	ids := &sqliteSequenceIDs{prefix: "0199d091", next: 1}
	security := application.SecurityService{Store: store, Clock: clock, IDs: ids}
	ns := testID("0199d091-0000-7000-8000-000000000010")
	token := "transaction-auth-token-at-least-32-chars"
	if err := security.Bootstrap(context.Background(), ports.Namespace{ID: ns, Name: "Protected"}, "operator", token); err != nil {
		t.Fatal(err)
	}
	identity, _ := security.Authenticate(context.Background(), token)
	ctx := application.WithIdentity(context.Background(), identity)
	manager := &revokeBeforeTransaction{store: store, before: func() {
		_, _, err := security.AdministrativeMutation(ctx, "revoke-awaiting-tx-0001", ports.SecurityAdminCommand{NamespaceID: ns, ExpectedVersion: 1, Operation: "set_grant", Grant: &ports.NamespaceGrant{NamespaceID: ns, PrincipalID: "operator", Permissions: []ports.Permission{ports.PermissionStateRead, ports.PermissionNamespaceAdmin}}})
		if err != nil {
			t.Fatal(err)
		}
	}}
	service, _ := application.NewAuthorizedService(manager, clock, ids, security)
	cc := sqliteCommandContext("0199d091-0000-7000-8000-000000000011", "awaiting-command-0001")
	cc.PrincipalID = identity.PrincipalID
	cc.Actor = identity.Actor
	if _, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{NamespaceID: ns, Title: "Should roll back", DesiredState: "Denied", Priority: domain.PriorityNormal}); securityErrorCode(err) != domain.ErrorCodeForbidden {
		t.Fatalf("stale authorization committed: %v", err)
	}
	page, err := service.SearchOutcomes(ctx, ns, ports.OutcomeFilter{}, 25, "")
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("unauthorized state persisted: %+v %v", page, err)
	}
}
