package commands

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestPluralIdentifierArraysUsePublicRESTNames(t *testing.T) {
	id := domain.MustParseID("0199d210-0000-7000-8000-000000000001")
	for _, tc := range []struct {
		command any
		field   string
	}{
		{application.ConfigureTriggerCommand{}, "target_endpoint_ids"},
		{application.RecordCriterionAssessmentCommand{}, "evidence_ids"},
	} {
		t.Run(tc.field, func(t *testing.T) {
			typeOf := reflect.TypeOf(tc.command)
			props := Schema(typeOf)["properties"].(map[string]any)
			if _, ok := props[tc.field]; !ok {
				t.Fatalf("generated contract does not expose the existing REST field %s", tc.field)
			}
			raw, _ := json.Marshal(map[string]any{tc.field: []domain.ID{id}})
			normalized, err := Normalize(raw, typeOf)
			if err != nil {
				t.Fatal(err)
			}
			target := reflect.New(typeOf)
			if err := Decode(normalized, target.Interface()); err != nil {
				t.Fatal(err)
			}
			var value []domain.ID
			switch v := target.Interface().(type) {
			case *application.ConfigureTriggerCommand:
				value = v.TargetEndpointIDs
			case *application.RecordCriterionAssessmentCommand:
				value = v.EvidenceIDs
			}
			if len(value) != 1 || value[0] != id {
				t.Fatalf("public field did not bind the identifier array: %+v", value)
			}
		})
	}
}
