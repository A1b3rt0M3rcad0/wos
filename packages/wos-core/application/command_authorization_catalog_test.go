package application_test

import (
	"strings"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func commandNameFromPublicID(name string) string {
	parts := strings.Split(name, "_")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, "")
}

func TestAuthorizationCatalogCoversEveryPublicCommand(t *testing.T) {
	descriptors := commands.NewCatalog(nil, nil).Descriptors()
	if len(descriptors) != 111 {
		t.Fatalf("public command catalog has %d descriptors, want 111", len(descriptors))
	}
	for _, descriptor := range descriptors {
		t.Run(descriptor.Name, func(t *testing.T) {
			requirement, ok := application.CommandAuthorizationForPresentation(commandNameFromPublicID(descriptor.Name))
			if !ok {
				t.Fatal("public command has no explicit authorization requirement")
			}
			if !requirement.PrimaryPermission.Valid() {
				t.Fatalf("invalid primary permission %q", requirement.PrimaryPermission)
			}
		})
	}
}

func TestAuthorizationCatalogFailsClosedForUnknownNamesAndMismatchedTypes(t *testing.T) {
	for _, name := range []string{
		"UnregisteredIssue",
		"UpdateOutcomeArtifact",
		"CreateRoadmapIssue",
		"SetTriggerWhatever",
		"CreateObjectiveTypo",
	} {
		t.Run(name, func(t *testing.T) {
			if _, ok := application.CommandAuthorizationForPresentation(name); ok {
				t.Fatal("unknown command unexpectedly has a permission")
			}
			if permission := application.CommandPermissionForPresentation(name); permission != "" {
				t.Fatalf("unknown command received permission %q", permission)
			}
			if _, ok := application.CommandAuthorizationPermissions(name, struct{}{}); ok {
				t.Fatal("unknown command unexpectedly has authorization requirements")
			}
		})
	}
	if _, ok := application.CommandAuthorizationPermissions("CreateOutcome", application.CreateWorkItemCommand{}); ok {
		t.Fatal("a registered name accepted a different command type")
	}
}

func TestAuthorizationCatalogConditionalRequirements(t *testing.T) {
	tests := []struct {
		name string
		cmd  any
		want []ports.Permission
	}{
		{
			name: "CreatePlanItems",
			cmd: application.CreatePlanItemsCommand{Items: []application.PlanItemInput{
				{Kind: domain.EntityKindObjective},
				{Kind: domain.EntityKindWorkItem, InitialCriteria: []application.InitialCriterion{{Title: "proof"}}},
			}},
			want: []ports.Permission{ports.PermissionPlanningWrite, ports.PermissionObjectiveCreate, ports.PermissionTaskCreate, ports.PermissionCriterionDefine},
		},
		{
			name: "CreateWorkItem without criteria",
			cmd:  application.CreateWorkItemCommand{},
			want: []ports.Permission{ports.PermissionTaskCreate},
		},
		{
			name: "CreateWorkItem with initial criteria",
			cmd:  application.CreateWorkItemCommand{InitialCriteria: []application.InitialCriterion{{Title: "proof"}}},
			want: []ports.Permission{ports.PermissionTaskCreate, ports.PermissionCriterionDefine},
		},
		{
			name: "RecordCriterionAssessment met",
			cmd:  application.RecordCriterionAssessmentCommand{Result: domain.AssessmentResultMet},
			want: []ports.Permission{ports.PermissionCriterionAssess},
		},
		{
			name: "RecordCriterionAssessment waived",
			cmd:  application.RecordCriterionAssessmentCommand{Result: domain.AssessmentResultWaived},
			want: []ports.Permission{ports.PermissionCriterionAssess, ports.PermissionCriterionWaive},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			name := test.name
			if name == "CreateWorkItem without criteria" || name == "CreateWorkItem with initial criteria" {
				name = "CreateWorkItem"
			} else if name == "RecordCriterionAssessment met" || name == "RecordCriterionAssessment waived" {
				name = "RecordCriterionAssessment"
			}
			got, ok := application.CommandAuthorizationPermissions(name, test.cmd)
			if !ok {
				t.Fatal("registered command did not resolve authorization requirements")
			}
			if len(got) != len(test.want) {
				t.Fatalf("permissions = %v, want %v", got, test.want)
			}
			for i := range got {
				if got[i] != test.want[i] {
					t.Fatalf("permissions = %v, want %v", got, test.want)
				}
			}
		})
	}
}
