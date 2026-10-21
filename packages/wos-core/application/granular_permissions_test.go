package application_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

func TestGranularPermissionCatalogAndValidation(t *testing.T) {
	permissions := ports.AllPermissions()
	names := domain.PermissionOperationNames()
	if len(permissions) != len(names) {
		t.Fatalf("AllPermissions count=%d, domain catalog count=%d", len(permissions), len(names))
	}
	for i, permission := range permissions {
		if !permission.Valid() || string(permission) != names[i] {
			t.Fatalf("permission[%d]=%q, want catalog entry %q", i, permission, names[i])
		}
	}
	permissions[0] = "changed"
	if ports.AllPermissions()[0].Valid() == false || ports.AllPermissions()[0] != ports.PermissionWorkContractReturn {
		t.Fatal("AllPermissions leaked its backing catalog")
	}
	if ports.Permission("task:invented").Valid() {
		t.Fatal("unknown permission is valid")
	}
	if err := ports.ValidatePermissionValues(nil); err != nil {
		t.Fatalf("an empty list should be structurally valid: %v", err)
	}
	if err := ports.ValidatePermissionValues([]ports.Permission{ports.PermissionTaskClaim, ports.PermissionTaskClaim}); err == nil {
		t.Fatal("duplicate permissions were accepted")
	}
	if err := ports.ValidatePermissionValues([]ports.Permission{"task:invented"}); err == nil {
		t.Fatal("unknown permission was accepted")
	}
	tooMany := make([]ports.Permission, ports.MaxPermissionValues+1)
	for i := range tooMany {
		tooMany[i] = ports.PermissionTaskClaim
	}
	if err := ports.ValidatePermissionValues(tooMany); err == nil || !strings.Contains(err.Error(), "128") {
		t.Fatalf("129 values must hit the explicit-list limit, got %v", err)
	}
	// The current catalog contains fewer than 128 distinct names. At the
	// 128-item boundary semantic validation therefore reaches duplicate checks;
	// the JSON schema independently publishes the 128-item ceiling.
	boundary := tooMany[:ports.MaxPermissionValues]
	if err := ports.ValidatePermissionValues(boundary); err == nil || strings.Contains(err.Error(), "exceeds 128") {
		t.Fatalf("128 values should pass the length check before duplicate validation, got %v", err)
	}
}

func TestCredentialPolicyPermissionListsAreExplicitAndEmptyDenies(t *testing.T) {
	policy := domain.CredentialPolicy{
		NamespaceID: domain.MustParseID("0199ff01-0000-7000-8000-000000000001"),
		CredentialID: domain.MustParseID("0199ff01-0000-7000-8000-000000000002"),
		PrincipalID: "policy-reader", Version: 1,
		AcceptanceFloor: domain.AcceptanceDirect,
		MaxActiveWorkContracts: 1, MaxActiveReviewContracts: 1,
		UpdatedAt: time.Date(2026, 10, 21, 0, 0, 0, 0, time.UTC),
	}
	if err := policy.Validate(); err != nil {
		t.Fatalf("empty policy permission list should be valid: %v", err)
	}
	if policy.Permits(string(ports.PermissionTaskClaim), domain.ID("")) {
		t.Fatal("an empty CredentialPolicy list granted an operation")
	}
	policy.PermittedOperations = []string{string(ports.PermissionTaskClaim), string(ports.PermissionTaskClaim)}
	if err := policy.Validate(); err == nil {
		t.Fatal("duplicate policy permissions were accepted")
	}
	policy.PermittedOperations = []string{"task:invented"}
	if err := policy.Validate(); err == nil {
		t.Fatal("unknown policy permission was accepted")
	}
	policy.PermittedOperations = make([]string, ports.MaxPermissionValues+1)
	for i := range policy.PermittedOperations {
		policy.PermittedOperations[i] = string(ports.PermissionTaskClaim)
	}
	if err := policy.Validate(); err == nil {
		t.Fatal("129 policy permission values were accepted")
	}
}

func TestGranularPermissionsAuthorizeOnlyTheRequestedOperation(t *testing.T) {
	ctx := context.Background()
	namespace := domain.MustParseID("0199ff01-0000-7000-8000-000000000011")
	store := memory.New()
	if err := store.PutNamespace(ctx, ports.Namespace{ID: namespace, Name: "Permissions"}); err != nil {
		t.Fatal(err)
	}
	security := application.SecurityService{Store: store}
	request := ports.AuthorizationRequest{NamespaceID: namespace, PrincipalID: "executor", Permission: ports.PermissionTaskClaim}
	if err := store.PutGrant(ctx, ports.NamespaceGrant{NamespaceID: namespace, PrincipalID: request.PrincipalID, Permissions: []ports.Permission{ports.PermissionTaskClaim}}); err != nil {
		t.Fatal(err)
	}
	if err := security.Authorize(ctx, request); err != nil {
		t.Fatalf("task:claim was not allowed: %v", err)
	}
	request.Permission = ports.PermissionTaskAccept
	if err := security.Authorize(ctx, request); err == nil {
		t.Fatal("task:claim implicitly authorized task:accept")
	}
	if err := store.PutGrant(ctx, ports.NamespaceGrant{NamespaceID: namespace, PrincipalID: request.PrincipalID, Permissions: []ports.Permission{ports.PermissionWorkWrite}}); err != nil {
		t.Fatal(err)
	}
	request.Permission = ports.PermissionTaskClaim
	if err := security.Authorize(ctx, request); err == nil {
		t.Fatal("legacy work:write was implicitly expanded before the compatibility migration")
	}
	if err := store.PutGrant(ctx, ports.NamespaceGrant{NamespaceID: namespace, PrincipalID: request.PrincipalID}); err != nil {
		t.Fatalf("empty grant should be accepted as deny-all: %v", err)
	}
	request.Permission = ports.PermissionTaskClaim
	if err := security.Authorize(ctx, request); err == nil {
		t.Fatal("empty grant authorized task:claim")
	}
}

func TestCommandAuthorizationUsesGranularAndCompoundCapabilities(t *testing.T) {
	tests := []struct {
		name string
		cmd  any
		want []ports.Permission
	}{
		{name: "ClaimWorkItem", cmd: application.ClaimWorkItemCommand{}, want: []ports.Permission{ports.PermissionTaskClaim}},
		{name: "CompleteWorkItem", cmd: application.CompleteWorkItemCommand{}, want: []ports.Permission{ports.PermissionTaskAccept}},
		{name: "ReportIssueWithBlocker", cmd: application.ReportIssueWithBlockerCommand{}, want: []ports.Permission{ports.PermissionIssueCreate, ports.PermissionBlockerCreate}},
		{name: "ResolveIssueAndBlockers", cmd: application.ResolveIssueAndBlockersCommand{}, want: []ports.Permission{ports.PermissionIssueResolve, ports.PermissionBlockerRelease}},
		{name: "AppendWorkActivity", cmd: application.AppendWorkActivityCommand{}, want: []ports.Permission{ports.PermissionRecordsWrite}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, ok := application.CommandAuthorizationPermissions(test.name, test.cmd)
			if !ok || len(got) != len(test.want) {
				t.Fatalf("requirements=%v registered=%v; want %v", got, ok, test.want)
			}
			for i := range got {
				if got[i] != test.want[i] {
					t.Fatalf("requirements=%v, want %v", got, test.want)
				}
			}
		})
	}
}
