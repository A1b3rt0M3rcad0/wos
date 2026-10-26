package commands

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func TestPublicCatalogExposesConditionalAuthorizationRequirements(t *testing.T) {
	catalog := NewCatalog(nil, nil)
	base := `"scope":{"namespace_id":"018f0a10-0000-7000-8000-000000000001","outcome_id":"018f0a10-0000-7000-8000-000000000002"}`
	criteria := `"initial_criteria":[{"title":"Proof exists","required":true,"verification_mode":"manual"}]`
	tests := []struct {
		name string
		body string
		want []ports.Permission
	}{
		{
			name: "create_plan_items",
			body: `{` + base + `,"items":[{"local_key":"objective","kind":"objective","title":"Objective"},{"local_key":"task","kind":"work_item","title":"Task",` + criteria + `}]}`,
			want: []ports.Permission{ports.PermissionPlanningWrite, ports.PermissionObjectiveCreate, ports.PermissionTaskCreate, ports.PermissionCriterionDefine},
		},
		{
			name: "create_work_item",
			body: `{` + base + `,"title":"Task",` + criteria + `}`,
			want: []ports.Permission{ports.PermissionTaskCreate, ports.PermissionCriterionDefine},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := catalog.AuthorizationRequirements(test.name, json.RawMessage(test.body))
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, test.want) {
				t.Fatalf("authorization requirements = %v, want %v", got, test.want)
			}
		})
	}
}

func TestPublicCatalogUnknownAuthorizationRequirementFailsClosed(t *testing.T) {
	_, err := NewCatalog(nil, nil).AuthorizationRequirements("update_outcome_artifact", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("unknown command returned authorization requirements")
	}
}
