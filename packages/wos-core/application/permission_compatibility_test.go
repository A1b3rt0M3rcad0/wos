package application

import (
	"reflect"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func TestPermissionCompatibilityCrosswalkIsExactAndFinite(t *testing.T) {
	cases := []struct {
		command string
		current ports.Permission
		legacy  ports.Permission
	}{
		{"CreateWorkItem", ports.PermissionTaskCreate, ports.PermissionWorkWrite},
		{"CreatePlanItems", ports.PermissionObjectiveCreate, ports.PermissionOutcomeWrite},
		{"CreatePlanItems", ports.PermissionTaskCreate, ports.PermissionWorkWrite},
		{"CreatePlanItems", ports.PermissionCriterionDefine, ports.PermissionOutcomeWrite},
		{"ReturnSignedWork", ports.PermissionTaskDeliver, ports.PermissionWorkContractReturn},
		{"ReturnSignedReview", ports.PermissionTaskReview, ports.PermissionWorkReviewDecide},
		{"SetNamespaceWorkProtocol", ports.PermissionPolicyManage, ports.PermissionNamespaceAdmin},
		{"SetCredentialPolicy", ports.PermissionPolicyManage, ports.PermissionNamespaceAdmin},
		{"SetAcceptancePolicy", ports.PermissionPolicyManage, ports.PermissionNamespaceAdmin},
		{"SetReviewGroup", ports.PermissionPolicyManage, ports.PermissionNamespaceAdmin},
		{"RelateIssueToWorkItem", ports.PermissionIssueUpdate, ports.PermissionWorkWrite},
	}
	for _, test := range cases {
		t.Run(test.command+"/"+string(test.current), func(t *testing.T) {
			got := legacyPermissionForCommand(test.command, test.current)
			if got != test.legacy {
				t.Fatalf("legacy permission = %q, want %q", got, test.legacy)
			}
		})
	}

	// The old PlanningWrite primary permission stays exact, while an Objective
	// or Task created by the compound command keeps its historical effect right.
	if got := legacyPermissionForCommand("CreatePlanItems", ports.PermissionPlanningWrite); got != "" {
		t.Fatalf("unchanged planning permission acquired a legacy alias: %q", got)
	}
	if got := legacyPermissionForCommand("ConfigureOutcomeOperationalProfile", ports.PermissionPolicyManage); got != "" {
		t.Fatalf("new command inherited a legacy alias: %q", got)
	}
	if got := legacyPermissionForCommand("FutureWorkCommand", ports.PermissionTaskAccept); got != "" {
		t.Fatalf("unregistered command inherited a legacy alias: %q", got)
	}
}

func TestPermissionCompatibilityIntersectsGrantAndPolicyWithoutCrossMixing(t *testing.T) {
	grant := []ports.Permission{ports.PermissionWorkWrite}
	policy := []ports.Permission{ports.PermissionOutcomeWrite}
	if got := effectivePermissionIntersection(grant, policy, true); len(got) != 0 {
		t.Fatalf("unrelated legacy grant and policy rights were mixed: %v", got)
	}

	policy = []ports.Permission{ports.PermissionWorkWrite}
	got := effectivePermissionIntersection(grant, policy, true)
	want := legacyCompatibilityCapabilities(grant)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("effective compatibility set = %v, want grant/policy intersection %v", got, want)
	}
	for _, forbidden := range []ports.Permission{
		ports.PermissionCredentialsIssue,
		ports.PermissionSigningKeyEnroll,
		ports.PermissionSigningKeyRotate,
		ports.PermissionSigningKeyRevoke,
		ports.PermissionWorkAdminComplete,
		ports.PermissionWorkAdminCancel,
	} {
		for _, capability := range got {
			if capability == forbidden {
				t.Fatalf("legacy work:write expanded to unrelated capability %q", forbidden)
			}
		}
	}

	// Existing explicit granular permissions remain exact, with no compatibility
	// expansion required. Missing policy remains unrestricted only at the policy
	// layer, matching the pre-existing credential policy contract.
	current := effectivePermissionIntersection([]ports.Permission{ports.PermissionTaskClaim}, nil, false)
	if !reflect.DeepEqual(current, []ports.Permission{ports.PermissionTaskClaim}) {
		t.Fatalf("explicit grant changed during canonicalization: %v", current)
	}
}

func TestMigratedNamespaceAdminMapsOnlyReviewedAdministrationCommands(t *testing.T) {
	grant := []ports.Permission{ports.PermissionNamespaceAdmin}
	policy := []ports.Permission{ports.PermissionNamespaceAdmin}
	got := effectivePermissionSet(grant, ports.PermissionCompatibilityLegacyV1, policy, ports.PermissionCompatibilityLegacyV1, true)
	want := []ports.Permission{ports.PermissionNamespaceAdmin, ports.PermissionPolicyManage, ports.PermissionTaskIntervene}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("migrated admin permissions = %v, want %v", got, want)
	}
	if legacyPermissionForCommand("FutureCredentialOperation", ports.PermissionPolicyManage) != "" {
		t.Fatal("future credential operation inherited a legacy admin alias")
	}
}
