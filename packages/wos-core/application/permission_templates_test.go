package application

import (
	"reflect"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func TestListPermissionTemplatesSnapshots(t *testing.T) {
	templates := ListPermissionTemplates()
	want := []ports.PermissionTemplate{
		{ID: ports.PermissionTemplateReader, Version: 1, NameKey: "permissionTemplateReader", DescriptionKey: "permissionTemplateReaderDescription", Permissions: []ports.Permission{ports.PermissionStateRead}},
		{ID: ports.PermissionTemplatePlanner, Version: 1, NameKey: "permissionTemplatePlanner", DescriptionKey: "permissionTemplatePlannerDescription", Permissions: []ports.Permission{
			ports.PermissionStateRead, ports.PermissionOutcomeWrite, ports.PermissionPlanningWrite,
			ports.PermissionObjectiveCreate, ports.PermissionObjectiveUpdate, ports.PermissionObjectiveStart,
			ports.PermissionObjectiveConfigureCompletion, ports.PermissionTaskCreate,
			ports.PermissionTaskUpdateDefinition, ports.PermissionTaskActivate, ports.PermissionTaskDefer,
			ports.PermissionTaskAssign, ports.PermissionCriterionDefine, ports.PermissionDependencyWrite,
		}},
		{ID: ports.PermissionTemplateExecutor, Version: 1, NameKey: "permissionTemplateExecutor", DescriptionKey: "permissionTemplateExecutorDescription", Permissions: []ports.Permission{
			ports.PermissionStateRead, ports.PermissionTaskClaim, ports.PermissionTaskRenew,
			ports.PermissionTaskRelease, ports.PermissionTaskDeliver, ports.PermissionRecordsWrite,
		}},
		{ID: ports.PermissionTemplateReviewer, Version: 1, NameKey: "permissionTemplateReviewer", DescriptionKey: "permissionTemplateReviewerDescription", Permissions: []ports.Permission{
			ports.PermissionStateRead, ports.PermissionTaskReview, ports.PermissionTaskAccept,
			ports.PermissionTaskRequestChanges, ports.PermissionCriterionAssess, ports.PermissionRecordsWrite,
		}},
		{ID: ports.PermissionTemplateIssueOperator, Version: 1, NameKey: "permissionTemplateIssueOperator", DescriptionKey: "permissionTemplateIssueOperatorDescription", Permissions: []ports.Permission{
			ports.PermissionStateRead, ports.PermissionIssueCreate, ports.PermissionIssueUpdate,
			ports.PermissionIssueAssign, ports.PermissionIssueInvestigate, ports.PermissionIssueResolve,
			ports.PermissionIssueReopen, ports.PermissionBlockerCreate, ports.PermissionBlockerRelease,
			ports.PermissionBlockerCancel, ports.PermissionRecordsWrite,
		}},
		{ID: ports.PermissionTemplateAdministrator, Version: 1, NameKey: "permissionTemplateAdministrator", DescriptionKey: "permissionTemplateAdministratorDescription", Permissions: []ports.Permission{
			ports.PermissionStateRead, ports.PermissionCredentialsRead, ports.PermissionCredentialsIssue,
			ports.PermissionCredentialsUpdate, ports.PermissionCredentialsRevoke,
			ports.PermissionCredentialsRotate, ports.PermissionAuditRead, ports.PermissionNamespaceAdmin,
			ports.PermissionActorDelegate, ports.PermissionPermissionsManage, ports.PermissionPolicyManage,
		}},
	}
	if !reflect.DeepEqual(templates, want) {
		t.Fatalf("permission template snapshot changed:\n got: %#v\nwant: %#v", templates, want)
	}
}

func TestPermissionTemplateCatalogIsDeterministicAndReturnsCopies(t *testing.T) {
	first := ListPermissionTemplates()
	second := ListPermissionTemplates()
	if !reflect.DeepEqual(first, second) {
		t.Fatal("template listing is not deterministic")
	}
	if len(first) == 0 {
		t.Fatal("template catalog is empty")
	}
	first[0].Permissions[0] = ports.PermissionPolicyManage
	first[0].ID = "changed"
	third := ListPermissionTemplates()
	if third[0].ID != ports.PermissionTemplateReader || third[0].Permissions[0] != ports.PermissionStateRead {
		t.Fatalf("mutating a returned template changed the catalog: %#v", third[0])
	}

	resolved, ok := ResolvePermissionTemplate(ports.PermissionTemplateExecutor, ports.PermissionTemplateVersion1)
	if !ok {
		t.Fatal("known version was not resolved")
	}
	resolved.Permissions[0] = ports.PermissionPolicyManage
	resolvedAgain, ok := ResolvePermissionTemplate(ports.PermissionTemplateExecutor, ports.PermissionTemplateVersion1)
	if !ok || resolvedAgain.Permissions[0] != ports.PermissionStateRead {
		t.Fatal("mutating resolved permissions changed the catalog")
	}
}

func TestResolvePermissionTemplateRequiresExactIDAndVersion(t *testing.T) {
	for _, test := range []struct {
		name    string
		id      ports.PermissionTemplateID
		version uint32
	}{
		{name: "unknown id", id: "owner", version: 1},
		{name: "unknown version", id: ports.PermissionTemplateReader, version: 2},
		{name: "empty id", id: "", version: 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			if template, ok := ResolvePermissionTemplate(test.id, test.version); ok {
				t.Fatalf("unexpectedly resolved template: %#v", template)
			}
		})
	}
}

func TestMaterializePermissionTemplateFreezesPresetAndMarksEditsCustom(t *testing.T) {
	preset, err := MaterializePermissionTemplate(PermissionTemplateSelection{
		TemplateID:      ports.PermissionTemplateExecutor,
		TemplateVersion: ports.PermissionTemplateVersion1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if preset.Custom || preset.TemplateID != ports.PermissionTemplateExecutor || preset.TemplateVersion != ports.PermissionTemplateVersion1 {
		t.Fatalf("preset provenance is incorrect: %#v", preset)
	}
	if !reflect.DeepEqual(preset.Permissions, templatePermissionsSlice(t, ports.PermissionTemplateExecutor)) {
		t.Fatalf("preset was not materialized exactly: %#v", preset.Permissions)
	}

	// Editing an issued/materialized value remains a concrete list. A changed
	// list loses preset identity and cannot follow a later template revision.
	edited, err := MaterializePermissionTemplate(PermissionTemplateSelection{
		TemplateID:      ports.PermissionTemplateExecutor,
		TemplateVersion: ports.PermissionTemplateVersion1,
		EditedPermissions: []ports.Permission{
			ports.PermissionTaskDeliver,
			ports.PermissionStateRead,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	wantEdited := []ports.Permission{ports.PermissionStateRead, ports.PermissionTaskDeliver}
	if !edited.Custom || edited.TemplateID != "" || edited.TemplateVersion != 0 || !reflect.DeepEqual(edited.Permissions, wantEdited) {
		t.Fatalf("edited set was not frozen as custom: %#v", edited)
	}
	// A semantically identical set is still the preset even when its incoming
	// list used a different order.
	unchanged, err := MaterializePermissionTemplate(PermissionTemplateSelection{
		TemplateID:      ports.PermissionTemplateReader,
		TemplateVersion: ports.PermissionTemplateVersion1,
		EditedPermissions: []ports.Permission{
			ports.PermissionStateRead,
		},
	})
	if err != nil || unchanged.Custom || unchanged.TemplateID != ports.PermissionTemplateReader {
		t.Fatalf("unchanged preset list should preserve preset provenance: %#v, %v", unchanged, err)
	}

	custom, err := MaterializePermissionTemplate(PermissionTemplateSelection{
		EditedPermissions: []ports.Permission{ports.PermissionTaskClaim},
	})
	if err != nil || !custom.Custom || custom.TemplateID != "" || custom.TemplateVersion != 0 {
		t.Fatalf("standalone custom permissions were not identified: %#v, %v", custom, err)
	}
	custom.Permissions[0] = ports.PermissionPolicyManage
	if templatePermissionsSlice(t, ports.PermissionTemplateExecutor)[0] != ports.PermissionStateRead {
		t.Fatal("materialized permission list shares memory with the catalog")
	}
}

func TestPreviewPermissionTemplateIntersectsGrantAndCredentialRestriction(t *testing.T) {
	preview, err := PreviewPermissionTemplate(PermissionTemplatePreviewInput{
		Selection: PermissionTemplateSelection{
			TemplateID:      ports.PermissionTemplatePlanner,
			TemplateVersion: ports.PermissionTemplateVersion1,
		},
		NamespaceGrantPermissions: []ports.Permission{
			ports.PermissionTaskCreate,
			ports.PermissionStateRead,
			ports.PermissionCriterionDefine,
			ports.PermissionTaskDeliver,
		},
		CredentialPolicyPermissions: []ports.Permission{
			ports.PermissionTaskCreate,
			ports.PermissionStateRead,
		},
		CredentialPolicyPresent: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if preview.Materialization.Custom || preview.Materialization.TemplateID != ports.PermissionTemplatePlanner {
		t.Fatalf("preview lost selected preset provenance: %#v", preview.Materialization)
	}
	wantEffective := []ports.Permission{ports.PermissionStateRead, ports.PermissionTaskCreate}
	if !reflect.DeepEqual(preview.EffectivePermissions, wantEffective) {
		t.Fatalf("effective permissions = %v, want %v", preview.EffectivePermissions, wantEffective)
	}
	wantDeniedByGrant := []ports.Permission{
		ports.PermissionDependencyWrite,
		ports.PermissionObjectiveConfigureCompletion,
		ports.PermissionObjectiveCreate,
		ports.PermissionObjectiveStart,
		ports.PermissionObjectiveUpdate,
		ports.PermissionOutcomeWrite,
		ports.PermissionPlanningWrite,
		ports.PermissionTaskActivate,
		ports.PermissionTaskAssign,
		ports.PermissionTaskDefer,
		ports.PermissionTaskUpdateDefinition,
	}
	if !reflect.DeepEqual(preview.ExcludedByNamespaceGrant, wantDeniedByGrant) {
		t.Fatalf("excluded by Namespace grant = %v, want %v", preview.ExcludedByNamespaceGrant, wantDeniedByGrant)
	}
	if !reflect.DeepEqual(preview.ExcludedByCredentialPolicy, []ports.Permission{ports.PermissionCriterionDefine}) {
		t.Fatalf("excluded by credential policy = %v, want criterion:define", preview.ExcludedByCredentialPolicy)
	}

	// Without a credential policy, only the namespace-specific grant bounds the
	// result. An explicitly present empty policy denies all permissions.
	withoutPolicy, err := PreviewPermissionTemplate(PermissionTemplatePreviewInput{
		Selection:                 PermissionTemplateSelection{TemplateID: ports.PermissionTemplateReader, TemplateVersion: 1},
		NamespaceGrantPermissions: []ports.Permission{ports.PermissionStateRead},
	})
	if err != nil || !reflect.DeepEqual(withoutPolicy.EffectivePermissions, []ports.Permission{ports.PermissionStateRead}) {
		t.Fatalf("absent credential policy changed the grant preview: %#v, %v", withoutPolicy, err)
	}
	emptyPolicy, err := PreviewPermissionTemplate(PermissionTemplatePreviewInput{
		Selection:                 PermissionTemplateSelection{TemplateID: ports.PermissionTemplateReader, TemplateVersion: 1},
		NamespaceGrantPermissions: []ports.Permission{ports.PermissionStateRead},
		CredentialPolicyPresent:   true,
	})
	if err != nil || len(emptyPolicy.EffectivePermissions) != 0 || !reflect.DeepEqual(emptyPolicy.ExcludedByCredentialPolicy, []ports.Permission{ports.PermissionStateRead}) {
		t.Fatalf("present empty credential policy must deny all: %#v, %v", emptyPolicy, err)
	}
}

func TestPreviewPermissionTemplateRejectsUnknownAndInvalidInputs(t *testing.T) {
	tests := []struct {
		name  string
		input PermissionTemplatePreviewInput
	}{
		{
			name: "unknown template version",
			input: PermissionTemplatePreviewInput{
				Selection: PermissionTemplateSelection{TemplateID: ports.PermissionTemplateReader, TemplateVersion: 2},
			},
		},
		{
			name: "unknown principal grant permission",
			input: PermissionTemplatePreviewInput{
				Selection:                 PermissionTemplateSelection{TemplateID: ports.PermissionTemplateReader, TemplateVersion: 1},
				NamespaceGrantPermissions: []ports.Permission{"task:future"},
			},
		},
		{
			name: "policy values without policy",
			input: PermissionTemplatePreviewInput{
				Selection:                   PermissionTemplateSelection{TemplateID: ports.PermissionTemplateReader, TemplateVersion: 1},
				CredentialPolicyPermissions: []ports.Permission{ports.PermissionStateRead},
			},
		},
		{
			name: "duplicate edited permission",
			input: PermissionTemplatePreviewInput{
				Selection: PermissionTemplateSelection{
					TemplateID:        ports.PermissionTemplateReader,
					TemplateVersion:   1,
					EditedPermissions: []ports.Permission{ports.PermissionStateRead, ports.PermissionStateRead},
				},
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := PreviewPermissionTemplate(test.input); err == nil {
				t.Fatal("expected invalid preview input to be rejected")
			}
		})
	}
}

func TestPermissionTemplatesUseValidUniqueExplicitPermissions(t *testing.T) {
	for _, template := range ListPermissionTemplates() {
		t.Run(string(template.ID), func(t *testing.T) {
			if err := ports.ValidatePermissionValues(template.Permissions); err != nil {
				t.Fatalf("template permissions are invalid: %v", err)
			}
		})
	}
}

func TestPermissionTemplatesKeepExecutionReviewAndAdministrationSeparate(t *testing.T) {
	executor := templatePermissions(t, ports.PermissionTemplateExecutor)
	for _, forbidden := range []ports.Permission{
		ports.PermissionTaskAccept,
		ports.PermissionTaskReview,
		ports.PermissionCriterionAssess,
		ports.PermissionWorkAdminComplete,
		ports.PermissionWorkCompleteDirect,
		ports.PermissionTaskIntervene,
		ports.PermissionPermissionsManage,
		ports.PermissionPolicyManage,
	} {
		if executor[forbidden] {
			t.Errorf("executor unexpectedly includes %q", forbidden)
		}
	}

	reviewer := templatePermissions(t, ports.PermissionTemplateReviewer)
	for _, forbidden := range []ports.Permission{
		ports.PermissionTaskClaim,
		ports.PermissionTaskRenew,
		ports.PermissionTaskRelease,
		ports.PermissionTaskDeliver,
		ports.PermissionTaskActivate,
		ports.PermissionTaskIntervene,
		ports.PermissionWorkAdminComplete,
		ports.PermissionWorkReviewAcquire,
	} {
		if reviewer[forbidden] {
			t.Errorf("reviewer unexpectedly includes execution/admin permission %q", forbidden)
		}
	}

	issueOperator := templatePermissions(t, ports.PermissionTemplateIssueOperator)
	if issueOperator[ports.PermissionTaskAccept] || issueOperator[ports.PermissionTaskDeliver] {
		t.Fatal("issue operator unexpectedly includes task execution or acceptance")
	}

	administrator := templatePermissions(t, ports.PermissionTemplateAdministrator)
	for _, forbidden := range []ports.Permission{
		ports.PermissionTaskAccept,
		ports.PermissionTaskDeliver,
		ports.PermissionTaskReview,
		ports.PermissionCriterionWaive,
		ports.PermissionAssessmentWaive,
		ports.PermissionWorkCompleteDirect,
		ports.PermissionWorkAdminComplete,
		ports.PermissionTaskIntervene,
	} {
		if administrator[forbidden] {
			t.Errorf("administrator unexpectedly includes bypass/review authority %q", forbidden)
		}
	}
}

func templatePermissions(t *testing.T, id ports.PermissionTemplateID) map[ports.Permission]bool {
	t.Helper()
	template, ok := ResolvePermissionTemplate(id, ports.PermissionTemplateVersion1)
	if !ok {
		t.Fatalf("template %q was not found", id)
	}
	permissions := make(map[ports.Permission]bool, len(template.Permissions))
	for _, permission := range template.Permissions {
		permissions[permission] = true
	}
	return permissions
}

func templatePermissionsSlice(t *testing.T, id ports.PermissionTemplateID) []ports.Permission {
	t.Helper()
	template, ok := ResolvePermissionTemplate(id, ports.PermissionTemplateVersion1)
	if !ok {
		t.Fatalf("template %q was not found", id)
	}
	return template.Permissions
}
