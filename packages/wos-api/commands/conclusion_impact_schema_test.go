package commands

import (
	"encoding/json"
	"reflect"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestObservedImpactRevisionIsOptionalExactAndPreservesLegacyIntention(t *testing.T) {
	scope := domain.Scope{
		NamespaceID: domain.MustParseID("0199ee00-0000-7000-8000-000000000001"),
		OutcomeID:   domain.MustParseID("0199ee00-0000-7000-8000-000000000002"),
	}
	objectiveID := domain.MustParseID("0199ee00-0000-7000-8000-000000000003")
	blockedRef := domain.EntityRef{Scope: scope, Kind: domain.EntityKindWorkItem, ID: objectiveID}
	const reason = "Explicitly accept the reviewed result and its remaining scope"
	for _, fixture := range []struct {
		name   string
		value  any
		legacy any
	}{
		{"achieve_outcome", application.AchieveOutcomeCommand{Scope: scope, ExpectedVersion: 17, Reason: reason}, struct {
			Scope           domain.Scope
			ExpectedVersion domain.Version
			Reason          string
		}{scope, 17, reason}},
		{"achieve_objective", application.AchieveObjectiveCommand{Scope: scope, ObjectiveID: objectiveID, ExpectedVersion: 17, Reason: reason}, struct {
			Scope           domain.Scope
			ObjectiveID     domain.ID
			ExpectedVersion domain.Version
			Reason          string
		}{scope, objectiveID, 17, reason}},
		{"report_issue_with_blocker", application.ReportIssueWithBlockerCommand{
			Scope: scope, Title: "A recorded operational problem", Description: reason,
			Severity: domain.IssueSeverityMinor, AffectedRefs: []domain.EntityRef{blockedRef},
			BlockedRef: blockedRef, BlockerDescription: "Explicitly prevents the selected Task", Propagation: domain.BlockerPropagationDirect,
		}, struct {
			Scope              domain.Scope
			Title              string
			Description        string
			Severity           domain.IssueSeverity
			AffectedRefs       []domain.EntityRef
			BlockedRef         domain.EntityRef
			BlockerDescription string
			Propagation        domain.BlockerPropagation
		}{scope, "A recorded operational problem", reason, domain.IssueSeverityMinor, []domain.EntityRef{blockedRef}, blockedRef, "Explicitly prevents the selected Task", domain.BlockerPropagationDirect}},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			before, err := json.Marshal(fixture.legacy)
			if err != nil {
				t.Fatal(err)
			}
			after, err := json.Marshal(fixture.value)
			if err != nil || string(before) != string(after) {
				t.Fatalf("old Core intention representation changed: before=%s after=%s err=%v", before, after, err)
			}
			payload, err := Encode(fixture.value)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err = json.Unmarshal(payload, &fields); err != nil {
				t.Fatal(err)
			}
			if _, exists := fields["observed_outcome_revision"]; exists {
				t.Fatalf("old omitted field changed: %s", payload)
			}
			typ := reflect.TypeOf(fixture.value)
			if _, hasVersion := typ.FieldByName("ExpectedVersion"); hasVersion && string(fields["expected_version"]) != "17" {
				t.Fatalf("old numeric version changed: %s", payload)
			}
			normalized, err := Normalize(payload, typ)
			if err != nil {
				t.Fatalf("old command shape rejected: %v", err)
			}
			decoded := reflect.New(typ)
			if err = Decode(normalized, decoded.Interface()); err != nil || !reflect.DeepEqual(decoded.Elem().Interface(), fixture.value) {
				t.Fatalf("old command did not round trip: %s %v", normalized, err)
			}
			withObserved := reflect.New(typ).Elem()
			withObserved.Set(reflect.ValueOf(fixture.value))
			revision := domain.OutcomeRevision(9007199254740993)
			withObserved.FieldByName("ObservedOutcomeRevision").Set(reflect.ValueOf(&revision))
			payload, err = Encode(withObserved.Interface())
			if err != nil {
				t.Fatal(err)
			}
			if err = json.Unmarshal(payload, &fields); err != nil || string(fields["observed_outcome_revision"]) != `"9007199254740993"` {
				t.Fatalf("observed revision lost exact decimal form: %s %v", payload, err)
			}
			normalized, err = Normalize(payload, typ)
			if err != nil {
				t.Fatal(err)
			}
			err = Decode(normalized, decoded.Interface())
			observed := decoded.Elem().FieldByName("ObservedOutcomeRevision").Interface().(*domain.OutcomeRevision)
			if err != nil || observed == nil || *observed != revision {
				t.Fatalf("observed revision did not round trip: %s %v", normalized, err)
			}
			fields["observed_outcome_revision"] = json.RawMessage(`9007199254740993`)
			invalid, _ := json.Marshal(fields)
			if _, err = Normalize(invalid, typ); err == nil {
				t.Fatal("new observed revision accepted a numeric wire value")
			}
			properties := Schema(typ)["properties"].(map[string]any)
			if properties["observed_outcome_revision"].(map[string]any)["type"] != "string" {
				t.Fatal("new observed revision schema is not exact decimal")
			}
			for _, required := range Schema(typ)["required"].([]string) {
				if required == "observed_outcome_revision" {
					t.Fatal("existing conclusion clients acquired a new required input")
				}
			}
		})
	}
}

func TestWorkspacePreviewQueryRequiresExplicitNullableParent(t *testing.T) {
	schema := WorkspaceReadSchema(reflect.TypeFor[application.WorkspaceViewQuery]())
	properties := schema["properties"].(map[string]any)
	preview := properties["placement_preview"].(map[string]any)["anyOf"].([]any)[0].(map[string]any)
	parent := preview["properties"].(map[string]any)["parent_id"].(map[string]any)
	if required := preview["required"].([]string); len(required) != 1 || required[0] != "parent_id" {
		t.Fatalf("preview schema loses explicit parent selection: %#v", preview)
	}
	if parent["anyOf"].([]any)[1].(map[string]any)["type"] != "null" {
		t.Fatal("preview schema cannot express the deliberate Outcome parent")
	}
	for _, required := range schema["required"].([]string) {
		if required == "placement_preview" || required == "blocker_preview" {
			t.Fatal("existing workspace reads acquired a required preview")
		}
	}
	for _, raw := range []string{
		`{"placement_preview":{}}`,
		`{"placement_preview":{"parent_id":null,"unexpected":"value"}}`,
	} {
		var query application.WorkspaceViewQuery
		if err := json.Unmarshal([]byte(raw), &query); err == nil {
			t.Fatalf("typed HTTP/MCP binding accepted an ambiguous parent query: %s", raw)
		}
	}
	var query application.WorkspaceViewQuery
	if err := json.Unmarshal([]byte(`{"placement_preview":{"parent_id":null}}`), &query); err != nil || query.PlacementPreview == nil || query.PlacementPreview.ParentID != nil {
		t.Fatalf("explicit Outcome preview was rejected: %+v %v", query, err)
	}
	query = application.WorkspaceViewQuery{}
	if err := json.Unmarshal([]byte(`{"subject":{"kind":"entity","ref":{"namespace_id":"0199ee00-0000-7000-8000-000000000001","outcome_id":"0199ee00-0000-7000-8000-000000000002","kind":"outcome","id":"0199ee00-0000-7000-8000-000000000002"}}}`), &query); err != nil || query.PlacementPreview != nil || query.BlockerPreview != nil {
		t.Fatalf("old query acquired an implicit preview: %+v %v", query, err)
	}
}
