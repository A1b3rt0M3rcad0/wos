package application

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func TestSecurityServiceDoesNotExposeUnauditedAdminWriteWrappers(t *testing.T) {
	typeOfSecurityService := reflect.TypeOf(SecurityService{})
	for _, name := range []string{"SetGrant", "RevokeCredential"} {
		if _, exists := typeOfSecurityService.MethodByName(name); exists {
			t.Fatalf("SecurityService.%s bypasses the audited CAS administrative mutation pipeline", name)
		}
	}
}

func TestPrepareCredentialIssuanceMaterializesAndBoundsTemplate(t *testing.T) {
	namespaceID := domain.MustParseID("0199f901-0000-7000-8000-000000000001")
	credentialID := domain.MustParseID("0199f901-0000-7000-8000-000000000002")
	actor := domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "worker"}
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	grant := ports.NamespaceGrant{
		NamespaceID: namespaceID,
		PrincipalID: "worker-principal",
		Permissions: []ports.Permission{
			ports.PermissionTaskClaim,
			ports.PermissionTaskRenew,
			ports.PermissionStateRead,
			ports.PermissionCredentialsIssue,
		},
	}
	prepared, err := PrepareCredentialIssuance(CredentialIssuanceSelection{
		NamespaceID: namespaceID, CredentialID: credentialID, PrincipalID: grant.PrincipalID,
		Actor: actor, ExpiresAt: now.Add(time.Hour), Name: "Build worker",
		TemplateID: ports.PermissionTemplateExecutor, TemplateVersion: ports.PermissionTemplateVersion1,
		CreatedByPrincipalID: "admin", CreatedByActor: domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "test", ID: "admin"},
	}, grant, now)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		string(ports.PermissionStateRead),
		string(ports.PermissionTaskClaim),
		string(ports.PermissionTaskRenew),
	}
	if !reflect.DeepEqual(prepared.Policy.PermittedOperations, want) {
		t.Fatalf("materialized policy = %v, want intersection %v", prepared.Policy.PermittedOperations, want)
	}
	if prepared.Policy.Version != 1 || prepared.Metadata.PolicyRevision != 1 || prepared.Metadata.RoleTemplateID != ports.PermissionTemplateExecutor || prepared.Metadata.RoleTemplateVersion != ports.PermissionTemplateVersion1 {
		t.Fatalf("policy/provenance not frozen: policy=%+v metadata=%+v", prepared.Policy, prepared.Metadata)
	}
	if !reflect.DeepEqual(prepared.Preview.ExcludedByNamespaceGrant, []ports.Permission{ports.PermissionRecordsWrite, ports.PermissionTaskDeliver, ports.PermissionTaskRelease}) {
		t.Fatalf("template exclusions = %v", prepared.Preview.ExcludedByNamespaceGrant)
	}
	if !prepared.Policy.UpdatedAt.Equal(now) || !prepared.Metadata.CreatedAt.Equal(now) || prepared.Metadata.Name != "Build worker" {
		t.Fatalf("creation metadata = %+v", prepared.Metadata)
	}
}

func TestPrepareCredentialIssuanceCustomListHasNoTemplateProvenance(t *testing.T) {
	namespaceID := domain.MustParseID("0199f902-0000-7000-8000-000000000001")
	credentialID := domain.MustParseID("0199f902-0000-7000-8000-000000000002")
	actor := domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "worker"}
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	grant := ports.NamespaceGrant{NamespaceID: namespaceID, PrincipalID: "worker", Permissions: []ports.Permission{
		ports.PermissionTaskClaim, ports.PermissionStateRead, ports.PermissionTaskRenew,
	}}
	selection := CredentialIssuanceSelection{
		NamespaceID: namespaceID, CredentialID: credentialID, PrincipalID: grant.PrincipalID,
		Actor: actor, ExpiresAt: now.Add(time.Hour), EditedPermissions: []ports.Permission{ports.PermissionTaskClaim, ports.PermissionStateRead},
		CreatedByPrincipalID: "admin", CreatedByActor: domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "test", ID: "admin"},
	}
	prepared, err := PrepareCredentialIssuance(selection, grant, now)
	if err != nil {
		t.Fatal(err)
	}
	if !prepared.Preview.Materialization.Custom || prepared.Metadata.RoleTemplateID != "" || prepared.Metadata.RoleTemplateVersion != 0 {
		t.Fatalf("custom materialization retained template provenance: %+v", prepared.Metadata)
	}
	if !reflect.DeepEqual(prepared.Policy.PermittedOperations, []string{string(ports.PermissionStateRead), string(ports.PermissionTaskClaim)}) {
		t.Fatalf("custom policy = %v", prepared.Policy.PermittedOperations)
	}
}

func TestPrepareCredentialIssuanceLegacySelectionFreezesCurrentGrant(t *testing.T) {
	namespaceID := domain.MustParseID("0199f903-0000-7000-8000-000000000001")
	credentialID := domain.MustParseID("0199f903-0000-7000-8000-000000000002")
	actor := domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "test", ID: "reader"}
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	grant := ports.NamespaceGrant{NamespaceID: namespaceID, PrincipalID: "reader", Permissions: []ports.Permission{
		ports.PermissionStateRead, ports.PermissionAuditRead,
	}}
	prepared, err := PrepareCredentialIssuance(CredentialIssuanceSelection{
		NamespaceID: namespaceID, CredentialID: credentialID, PrincipalID: grant.PrincipalID,
		Actor: actor, ExpiresAt: now.Add(time.Hour),
		CreatedByPrincipalID: "admin", CreatedByActor: domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "test", ID: "admin"},
	}, grant, now)
	if err != nil {
		t.Fatal(err)
	}
	if !prepared.Preview.Materialization.Custom || !reflect.DeepEqual(prepared.Policy.PermittedOperations, []string{string(ports.PermissionAuditRead), string(ports.PermissionStateRead)}) {
		t.Fatalf("legacy grant was not frozen as custom: %+v", prepared)
	}
}

func TestPrepareCredentialIssuanceRejectsInvalidPolicyIntent(t *testing.T) {
	namespaceID := domain.MustParseID("0199f904-0000-7000-8000-000000000001")
	credentialID := domain.MustParseID("0199f904-0000-7000-8000-000000000002")
	actor := domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "worker"}
	now := time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)
	grant := ports.NamespaceGrant{NamespaceID: namespaceID, PrincipalID: "worker", Permissions: []ports.Permission{ports.PermissionStateRead}}
	base := CredentialIssuanceSelection{
		NamespaceID: namespaceID, CredentialID: credentialID, PrincipalID: grant.PrincipalID,
		Actor: actor, ExpiresAt: now.Add(time.Hour), TemplateID: ports.PermissionTemplateReader,
		TemplateVersion:      ports.PermissionTemplateVersion1,
		CreatedByPrincipalID: "admin", CreatedByActor: domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "test", ID: "admin"},
	}
	cases := []struct {
		name   string
		mutate func(*CredentialIssuanceSelection)
	}{
		{"unknown template version", func(s *CredentialIssuanceSelection) { s.TemplateVersion++ }},
		{"overlong name", func(s *CredentialIssuanceSelection) { s.Name = strings.Repeat("n", 101) }},
		{"overlong description", func(s *CredentialIssuanceSelection) { s.Description = strings.Repeat("d", 1001) }},
		{"expired credential", func(s *CredentialIssuanceSelection) { s.ExpiresAt = now }},
		{"duplicate outcome scope", func(s *CredentialIssuanceSelection) { id := namespaceID; s.AllowedOutcomeIDs = []domain.ID{id, id} }},
		{"invalid custom permission", func(s *CredentialIssuanceSelection) {
			s.TemplateID = ""
			s.TemplateVersion = 0
			s.EditedPermissions = []ports.Permission{"unknown:permission"}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			selection := base
			tc.mutate(&selection)
			if _, err := PrepareCredentialIssuance(selection, grant, now); err == nil {
				t.Fatal("invalid issuance intent accepted")
			}
		})
	}
}
