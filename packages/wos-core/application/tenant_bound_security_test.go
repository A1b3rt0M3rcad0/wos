package application

import (
	"context"
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type boundTenantSecurityStore struct {
	ports.SecurityStore
	tenant      d.ID
	credential  ports.Credential
	permissions []ports.Permission
}

func (s boundTenantSecurityStore) BoundTenantID() d.ID { return s.tenant }
func (s boundTenantSecurityStore) GetCredential(context.Context, string) (ports.Credential, error) {
	return s.credential, nil
}
func (s boundTenantSecurityStore) GetGrant(context.Context, d.ID, string) (ports.TenantGrant, error) {
	return ports.TenantGrant{TenantID: s.tenant, PrincipalID: s.credential.PrincipalID, Permissions: s.permissions}, nil
}

func TestBoundTenantCreationRequiresInstallationProvisioning(t *testing.T) {
	tenant := d.MustParseID("0199f390-0000-7000-8000-000000000011")
	credential := ports.Credential{TenantID: tenant, ID: d.MustParseID("0199f390-0000-7000-8000-000000000012"), PrincipalID: "operator", Digest: TokenDigest("operator test token"), ExpiresAt: policyReadFaultClock{}.Now().Add(time.Hour)}
	store := boundTenantSecurityStore{tenant: tenant, credential: credential, permissions: []ports.Permission{ports.PermissionTenantAdmin, ports.PermissionInstallationProvision}}
	security := SecurityService{Store: store, Clock: policyReadFaultClock{}}
	ctx := WithIdentity(context.Background(), Identity{TenantID: tenant, PrincipalID: credential.PrincipalID, CredentialDigest: credential.Digest, CredentialID: credential.ID})
	_, secret, err := security.AdministrativeMutation(ctx, "another-tenant", ports.SecurityAdminCommand{TenantID: tenant, Operation: "create_tenant", ExpectedVersion: 1, TenantName: "Another account"})
	if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodePreconditionFailed || secret != "" {
		t.Fatalf("bound store allowed creation or lost explicit provisioning error: %s %v", code, err)
	}
	// An unauthorized caller must still receive the authorization result before
	// learning about provisioning behavior. No ID generation or write is reached.
	store.permissions = []ports.Permission{ports.PermissionTenantAdmin}
	security.Store = store
	_, _, err = security.AdministrativeMutation(ctx, "another-tenant", ports.SecurityAdminCommand{TenantID: tenant, Operation: "create_tenant", ExpectedVersion: 1, TenantName: "Another account"})
	if code, _ := d.ErrorCodeOf(err); code != d.ErrorCodeForbidden {
		t.Fatalf("provisioning guard bypassed operator authorization: %v", err)
	}
}
