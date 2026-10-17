package sqlite

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"path/filepath"
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
	if err := security.Bootstrap(context.Background(), ports.Tenant{ID: ns, Name: "Admin"}, "operator", token); err != nil {
		t.Fatal(err)
	}
	identity, err := security.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	ctx := application.WithIdentity(context.Background(), identity)
	command := ports.SecurityAdminCommand{TenantID: ns, ExpectedVersion: 1, Operation: "set_grant", Grant: &ports.TenantGrant{TenantID: ns, PrincipalID: "agent", Permissions: []ports.Permission{ports.PermissionStateRead}}}
	result, _, err := security.AdministrativeMutation(ctx, "admin-grant-key-0001", command)
	if err != nil || result.TenantVersion != 2 {
		t.Fatalf("grant: %+v %v", result, err)
	}
	replay, _, err := security.AdministrativeMutation(ctx, "admin-grant-key-0001", command)
	if err != nil || !replay.IdempotentReplay || replay.TenantVersion != 2 {
		t.Fatalf("replay: %+v %v", replay, err)
	}
	command.Grant.PrincipalID = "different"
	if _, _, err = security.AdministrativeMutation(ctx, "admin-grant-key-0001", command); securityErrorCode(err) != domain.ErrorCodeIdempotencyConflict {
		t.Fatalf("different intent: %v", err)
	}
	if _, _, err = security.AdministrativeMutation(ctx, "admin-stale-key-0001", command); securityErrorCode(err) != domain.ErrorCodeVersionConflict {
		t.Fatalf("stale grant: %v", err)
	}
	credential := ports.SecurityAdminCommand{TenantID: ns, ExpectedVersion: 2, Operation: "issue_credential", Credential: &ports.Credential{TenantID: ns, PrincipalID: "agent", Actor: domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "agent"}, ExpiresAt: clock.Now().Add(time.Hour)}}
	issued, secret, err := security.AdministrativeMutation(ctx, "admin-issue-key-0001", credential)
	if err != nil || secret == "" || issued.TenantVersion != 3 {
		t.Fatalf("issue: %+v %v", issued, err)
	}
	if _, err = security.Authenticate(context.Background(), secret); err != nil {
		t.Fatal(err)
	}
	replay, hidden, err := security.AdministrativeMutation(ctx, "admin-issue-key-0001", credential)
	if err != nil || hidden != "" || !replay.TokenOmitted || replay.Credential.ID != issued.Credential.ID {
		t.Fatalf("secret replay: %+v %v", replay, err)
	}
	// Competing physical connections use the same Tenant version guard.
	second := openTestStore(t, path)
	other := application.SecurityService{Store: second, Clock: clock, IDs: ids}
	var wait sync.WaitGroup
	results := make(chan error, 2)
	for i, s := range []application.SecurityService{security, other} {
		wait.Add(1)
		go func(i int, s application.SecurityService) {
			defer wait.Done()
			principal := []string{"human-one", "human-two"}[i]
			_, _, err := s.AdministrativeMutation(ctx, "concurrent-grant-"+principal, ports.SecurityAdminCommand{TenantID: ns, ExpectedVersion: 3, Operation: "set_grant", Grant: &ports.TenantGrant{TenantID: ns, PrincipalID: principal, Permissions: []ports.Permission{ports.PermissionStateRead}}})
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
	if err != nil || snapshot.TenantVersion != 4 || len(snapshot.Audit) != 3 || len(snapshot.Credentials) != 2 {
		t.Fatalf("audit: %+v %v", snapshot, err)
	}
	revoked, _, err := security.AdministrativeMutation(ctx, "admin-revoke-key-001", ports.SecurityAdminCommand{TenantID: ns, ExpectedVersion: 4, Operation: "revoke_credential", CredentialID: issued.Credential.ID})
	if err != nil || revoked.TenantVersion != 5 {
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
	if err != nil || snapshot.TenantVersion != 5 || len(snapshot.Audit) != 4 {
		t.Fatalf("restart: %+v %v", snapshot, err)
	}
	// Current permission must be checked before revealing the old receipt.
	if err = security.SetGrant(ctx, ports.TenantGrant{TenantID: ns, PrincipalID: "operator", Permissions: []ports.Permission{ports.PermissionStateRead}}); err != nil {
		t.Fatal(err)
	}
	if _, _, err = security.AdministrativeMutation(ctx, "admin-issue-key-0001", credential); securityErrorCode(err) != domain.ErrorCodeForbidden {
		t.Fatalf("revoked administration replay: %v", err)
	}
}

func securityErrorCode(err error) domain.ErrorCode { code, _ := domain.ErrorCodeOf(err); return code }

func TestTenantCreationCopiesOnlyGrantedPermissions(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "tenant.db"))
	security := application.SecurityService{Store: store, Clock: sqliteFixedClock{now: time.Now().UTC()}, IDs: &sqliteSequenceIDs{prefix: "0199d093", next: 1}}
	ns := testID("0199d093-0000-7000-8000-000000000010")
	token := "tenant-creation-credential-32-chars"
	if err := security.Bootstrap(context.Background(), ports.Tenant{ID: ns, Name: "Source"}, "operator", token); err != nil {
		t.Fatal(err)
	}
	identity, _ := security.Authenticate(context.Background(), token)
	ctx := application.WithIdentity(context.Background(), identity)
	permissions := []ports.Permission{ports.PermissionStateRead, ports.PermissionTenantAdmin, ports.PermissionInstallationProvision}
	if err := security.SetGrant(ctx, ports.TenantGrant{TenantID: ns, PrincipalID: "operator", Permissions: permissions}); err != nil {
		t.Fatal(err)
	}
	intent := application.AdministrativeIntent{TenantID: ns, ExpectedVersion: 1, Operation: "create_tenant", TenantName: "Created"}
	result, createdToken, err := security.Administer(ctx, "create-tenant-key-0001", intent)
	if err != nil || createdToken == "" || result.CreatedTenant == nil {
		t.Fatalf("create tenant: %+v %v", result, err)
	}
	grant, err := store.GetGrant(context.Background(), result.CreatedTenant.ID, "operator")
	if err != nil || len(grant.Permissions) != 2 {
		t.Fatalf("permission escalation: %+v %v", grant, err)
	}
	createdIdentity, err := security.Authenticate(context.Background(), createdToken)
	if err != nil {
		t.Fatal(err)
	}
	if err = security.Require(application.WithIdentity(context.Background(), createdIdentity), ns, ports.PermissionStateRead); securityErrorCode(err) != domain.ErrorCodeForbidden {
		t.Fatal("new credential crossed original scope", err)
	}
	replay, hidden, err := security.Administer(ctx, "create-tenant-key-0001", intent)
	if err != nil || hidden != "" || !replay.IdempotentReplay || replay.CreatedTenant.ID != result.CreatedTenant.ID {
		t.Fatalf("tenant replay: %+v %v", replay, err)
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
	if err := security.Bootstrap(context.Background(), ports.Tenant{ID: ns, Name: "Protected"}, "operator", token); err != nil {
		t.Fatal(err)
	}
	identity, _ := security.Authenticate(context.Background(), token)
	ctx := application.WithIdentity(context.Background(), identity)
	manager := &revokeBeforeTransaction{store: store, before: func() {
		_, _, err := security.AdministrativeMutation(ctx, "revoke-awaiting-tx-0001", ports.SecurityAdminCommand{TenantID: ns, ExpectedVersion: 1, Operation: "set_grant", Grant: &ports.TenantGrant{TenantID: ns, PrincipalID: "operator", Permissions: []ports.Permission{ports.PermissionStateRead, ports.PermissionTenantAdmin}}})
		if err != nil {
			t.Fatal(err)
		}
	}}
	service, _ := application.NewAuthorizedService(manager, clock, ids, security)
	cc := sqliteCommandContext("0199d091-0000-7000-8000-000000000011", "awaiting-command-0001")
	cc.PrincipalID = identity.PrincipalID
	cc.Actor = identity.Actor
	if _, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{TenantID: ns, Title: "Should roll back", DesiredState: "Denied", Priority: domain.PriorityNormal}); securityErrorCode(err) != domain.ErrorCodeForbidden {
		t.Fatalf("stale authorization committed: %v", err)
	}
	page, err := service.SearchOutcomes(ctx, ns, ports.OutcomeFilter{}, 25, "")
	if err != nil || len(page.Items) != 0 {
		t.Fatalf("unauthorized state persisted: %+v %v", page, err)
	}
}

func TestTenantAdministratorCannotAcquireInstallationAuthority(t *testing.T) {
	store := openTestStore(t, filepath.Join(t.TempDir(), "operator-boundary.db"))
	clock := sqliteFixedClock{now: time.Now().UTC()}
	security := application.SecurityService{Store: store, Clock: clock, IDs: &sqliteSequenceIDs{prefix: "0199d098", next: 1}}
	ns := testID("0199d098-0000-7000-8000-000000000010")
	if err := security.Bootstrap(context.Background(), ports.Tenant{ID: ns, Name: "Shared"}, "operator", "installation-boundary-bootstrap-credential"); err != nil {
		t.Fatal(err)
	}
	operator, err := security.Authenticate(context.Background(), "installation-boundary-bootstrap-credential")
	if err != nil {
		t.Fatal(err)
	}
	operatorCtx := application.WithIdentity(context.Background(), operator)
	if err = security.RequireInstallation(operatorCtx, ports.PermissionInstallationObserve); err != nil {
		t.Fatal(err)
	}
	if err = security.SetGrant(operatorCtx, ports.TenantGrant{TenantID: ns, PrincipalID: "tenant-admin", Permissions: []ports.Permission{ports.PermissionStateRead, ports.PermissionTenantAdmin, ports.PermissionActorDelegate}}); err != nil {
		t.Fatal(err)
	}
	_, token, err := security.IssueCredential(operatorCtx, ns, "tenant-admin", domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "test", ID: "tenant"}, clock.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	tenant, err := security.Authenticate(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	tenantCtx := application.WithIdentity(context.Background(), tenant)
	if _, err = security.AdministrativeSnapshot(tenantCtx, ns); err != nil {
		t.Fatal("ordinary administration must remain available", err)
	}
	commands := []ports.SecurityAdminCommand{
		{TenantID: ns, ExpectedVersion: 1, Operation: "create_tenant", TenantName: "Unauthorized"},
		{TenantID: ns, ExpectedVersion: 1, Operation: "set_grant", Grant: &ports.TenantGrant{TenantID: ns, PrincipalID: "tenant-admin", Permissions: []ports.Permission{ports.PermissionTenantAdmin, ports.PermissionInstallationProvision}}},
		{TenantID: ns, ExpectedVersion: 1, Operation: "set_grant", Grant: &ports.TenantGrant{TenantID: ns, PrincipalID: "operator", Permissions: []ports.Permission{ports.PermissionStateRead}}},
		{TenantID: ns, ExpectedVersion: 1, Operation: "issue_credential", Credential: &ports.Credential{TenantID: ns, PrincipalID: "operator", Actor: tenant.Actor, ExpiresAt: clock.Now().Add(time.Hour)}},
		{TenantID: ns, ExpectedVersion: 1, Operation: "revoke_credential", CredentialID: operator.CredentialID},
	}
	for i, command := range commands {
		_, secret, err := security.AdministrativeMutation(tenantCtx, "operator-boundary-key-0001", command)
		if securityErrorCode(err) != domain.ErrorCodeForbidden || secret != "" {
			t.Fatalf("command %d escaped operator boundary: %v", i, err)
		}
		// A direct storage caller cannot bypass the transaction's second check.
		_, err = store.ApplySecurityAdmin(tenantCtx, ports.SecurityAdminRequest{Command: command, PrincipalID: tenant.PrincipalID, Actor: tenant.Actor, CredentialDigest: tenant.CredentialDigest, IdempotencyKey: "operator-boundary-key-0002", Fingerprint: "test", Now: clock.Now()})
		if securityErrorCode(err) != domain.ErrorCodeForbidden {
			t.Fatalf("transaction %d escaped operator boundary: %v", i, err)
		}
	}
	if err = security.RequireInstallation(tenantCtx, ports.PermissionInstallationObserve); securityErrorCode(err) != domain.ErrorCodeForbidden {
		t.Fatal("tenant read installation metrics", err)
	}
	if _, _, err = security.IssueCredential(tenantCtx, ns, "operator", tenant.Actor, clock.Now().Add(time.Hour)); securityErrorCode(err) != domain.ErrorCodeForbidden {
		t.Fatal("composition helper minted operator credential", err)
	}
	if err = security.RevokeCredential(tenantCtx, ns, operator.CredentialID); securityErrorCode(err) != domain.ErrorCodeForbidden {
		t.Fatal("composition helper revoked operator", err)
	}
	result, _, err := security.AdministrativeMutation(operatorCtx, "operator-authorized-create-0001", commands[0])
	if err != nil {
		t.Fatal(err)
	}
	createdGrant, err := store.GetGrant(context.Background(), result.CreatedTenant.ID, operator.PrincipalID)
	if err != nil || ports.HasInstallationPermissions(createdGrant.Permissions) {
		t.Fatalf("new Tenant copied installation authority: %+v %v", createdGrant, err)
	}
	// Revocation must protect an already known metrics identity on every request.
	if err = security.SetGrant(operatorCtx, ports.TenantGrant{TenantID: ns, PrincipalID: "tenant-admin", Permissions: []ports.Permission{ports.PermissionStateRead, ports.PermissionTenantAdmin, ports.PermissionInstallationObserve}}); err != nil {
		t.Fatal(err)
	}
	if err = security.RequireInstallation(tenantCtx, ports.PermissionInstallationObserve); err != nil {
		t.Fatal(err)
	}
	u, err := store.Begin(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	policy := domain.CredentialPolicy{TenantID: ns, CredentialID: tenant.CredentialID, PrincipalID: tenant.PrincipalID, Version: 1, PermittedOperations: []string{string(ports.PermissionStateRead), string(ports.PermissionTenantAdmin), string(ports.PermissionInstallationObserve)}, AllowedOutcomeIDs: []domain.ID{testID("0199d098-0000-7000-8000-000000000099")}, AcceptanceFloor: domain.AcceptanceDirect, MaxActiveWorkContracts: 3, MaxActiveReviewContracts: 1, UpdatedAt: clock.Now()}
	if err = u.(ports.SigningIdentityUnitOfWork).SigningIdentity().SaveCredentialPolicy(context.Background(), policy, 0); err != nil {
		u.Rollback()
		t.Fatal(err)
	}
	if err = u.Commit(); err != nil {
		t.Fatal(err)
	}
	if err = security.RequireInstallation(tenantCtx, ports.PermissionInstallationObserve); securityErrorCode(err) != domain.ErrorCodeForbidden {
		t.Fatal("Outcome-restricted observer read installation metrics", err)
	}
	if err = security.SetGrant(operatorCtx, ports.TenantGrant{TenantID: ns, PrincipalID: "tenant-admin", Permissions: []ports.Permission{ports.PermissionStateRead, ports.PermissionTenantAdmin}}); err != nil {
		t.Fatal(err)
	}
	if err = security.RequireInstallation(tenantCtx, ports.PermissionInstallationObserve); securityErrorCode(err) != domain.ErrorCodeForbidden {
		t.Fatal("revoked observer retained authority", err)
	}
}
