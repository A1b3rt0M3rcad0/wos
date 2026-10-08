package local

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"testing"
)

func TestContractPermissionsRespectLocalPrincipalAndAdministrativeOptIn(t *testing.T) {
	for _, admin := range []bool{false, true} {
		auth, _ := NewAuthorizer("local-owner", admin)
		request := ports.AuthorizationRequest{NamespaceID: domain.MustParseID("0199ff01-0000-7000-8000-000000000001"), PrincipalID: "local-owner", Permission: ports.PermissionWorkContractAcquire}
		if err := auth.Authorize(context.Background(), request); err != nil {
			t.Fatal("local acquisition should follow ordinary work permission", err)
		}
		request.Permission = ports.PermissionWorkContractRevoke
		if err := auth.Authorize(context.Background(), request); (err == nil) != admin {
			t.Fatal("revocation requires administrative opt-in", admin, err)
		}
		request.Permission = ports.PermissionNamespaceAdmin
		if auth.Authorize(context.Background(), request) == nil {
			t.Fatal("local contract permissions must not grant Namespace administration")
		}
		request.PrincipalID = "another-principal"
		request.Permission = ports.PermissionWorkContractAcquire
		if auth.Authorize(context.Background(), request) == nil {
			t.Fatal("local acquisition accepted another principal")
		}
	}
}
