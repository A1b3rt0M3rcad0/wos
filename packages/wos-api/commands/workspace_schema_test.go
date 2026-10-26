package commands

import (
	"reflect"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestWorkspaceEligibilityContractMatchesCanonicalRules(t *testing.T) {
	rules := application.ContractCapabilities().WorkspaceRules
	if rules.Version != 1 || !reflect.DeepEqual(rules.EvidenceLinkTargetKinds, domain.EvidenceLinkTargetKinds()) {
		t.Fatal("capability target eligibility drifted from Core")
	}
	for kind, values := range rules.BlockerPropagation {
		if !reflect.DeepEqual(values, domain.BlockerPropagationOptions(kind)) {
			t.Fatal("propagation eligibility drifted", kind)
		}
	}
	legacyProperties := Schema(reflect.TypeFor[application.CreateEvidenceLinkCommand]())["properties"].(map[string]any)
	legacyKind := legacyProperties["target_ref"].(map[string]any)["properties"].(map[string]any)["kind"].(map[string]any)
	if _, constrained := legacyKind["enum"]; constrained {
		t.Fatal("released CreateEvidenceLink wire schema acquired a new constraint")
	}
	for _, typ := range []reflect.Type{reflect.TypeFor[application.EvidenceLinkInput]()} {
		properties := Schema(typ)["properties"].(map[string]any)
		values := properties["target_ref"].(map[string]any)["properties"].(map[string]any)["kind"].(map[string]any)["enum"].([]string)
		if len(values) != len(rules.EvidenceLinkTargetKinds) {
			t.Fatal("transport and capability target sets differ")
		}
		for _, value := range values {
			if !domain.EvidenceLinkTargetAllowed(domain.EntityKind(value)) {
				t.Fatal("invalid target published", value)
			}
		}
		if domain.EvidenceLinkTargetAllowed(domain.EntityKindBlocker) || !domain.EvidenceLinkTargetAllowed(domain.EntityKindDecision) {
			t.Fatal("documentary target semantics changed")
		}
	}
}

func TestRegisteredSettersRequireNullableFieldAndExactVersion(t *testing.T) {
	for name, field := range map[string]string{
		"set_work_item_execution_spec": "execution_spec", "set_work_item_not_before": "not_before",
		"set_work_item_objective": "objective_id", "set_objective_parent": "parent_objective_id",
	} {
		t.Run(name, func(t *testing.T) {
			var descriptor Descriptor
			for _, item := range NewCatalog(nil, nil).Descriptors() {
				if item.Name == name {
					descriptor = item
					break
				}
			}
			if descriptor.Name == "" {
				t.Fatal("typed setter has no executable catalog registration")
			}
			properties := descriptor.Schema["properties"].(map[string]any)
			if properties["expected_version"].(map[string]any)["type"] != "string" {
				t.Fatal("setter version is not exact decimal")
			}
			if _, ok := properties[field].(map[string]any)["anyOf"]; !ok {
				t.Fatal("setter schema cannot express explicit null")
			}
			found := false
			for _, required := range descriptor.Schema["required"].([]string) {
				found = found || required == field
			}
			if !found {
				t.Fatal("setter omission can accidentally clear a field")
			}
		})
	}
}
