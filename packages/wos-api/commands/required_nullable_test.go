package commands

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestRequiredNullableSettersPreserveClearAndRejectOmission(t *testing.T) {
	type setter struct {
		ExpectedVersion domain.Version `json:"expected_version,string"`
		ObjectiveID     *domain.ID     `wos:"required"`
		NotBefore       *time.Time     `wos:"required"`
	}
	command := setter{ExpectedVersion: 9007199254740993}
	raw, err := Encode(command)
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"expected_version":"9007199254740993","not_before":null,"objective_id":null}` {
		t.Fatalf("explicit clears or exact version lost: %s", raw)
	}
	normalized, err := Normalize(raw, reflect.TypeFor[setter]())
	if err != nil {
		t.Fatal(err)
	}
	var decoded setter
	if err = Decode(normalized, &decoded); err != nil || !reflect.DeepEqual(command, decoded) {
		t.Fatalf("clear round trip: %+v %v", decoded, err)
	}
	for _, invalid := range []string{
		`{"expected_version":"9007199254740993","not_before":null}`,
		`{"expected_version":"9007199254740993","objective_id":null}`,
		`{"expected_version":9007199254740993,"objective_id":null,"not_before":null}`,
	} {
		if _, err = Normalize([]byte(invalid), reflect.TypeFor[setter]()); err == nil {
			t.Fatalf("omission or imprecise version accepted: %s", invalid)
		}
	}
	properties := Schema(reflect.TypeFor[setter]())["properties"].(map[string]any)
	for _, key := range []string{"objective_id", "not_before"} {
		alternatives := properties[key].(map[string]any)["anyOf"].([]any)
		if alternatives[1].(map[string]any)["type"] != "null" {
			t.Fatalf("schema hides explicit clear: %s", key)
		}
	}
}

func TestNullableSetterChangeRetainsLegacyOptionalEncoding(t *testing.T) {
	command := application.UpdateWorkItemCommand{Scope: domain.Scope{NamespaceID: domain.MustParseID("0199d094-0000-7000-8000-000000000001"), OutcomeID: domain.MustParseID("0199d094-0000-7000-8000-000000000002")}, WorkItemID: domain.MustParseID("0199d094-0000-7000-8000-000000000003"), ExpectedVersion: 9007199254740993}
	raw, err := Encode(command)
	if err != nil {
		t.Fatal(err)
	}
	var properties map[string]json.RawMessage
	if err = json.Unmarshal(raw, &properties); err != nil {
		t.Fatal(err)
	}
	if string(properties["expected_version"]) != "9007199254740993" {
		t.Fatal("legacy numeric contract changed", string(raw))
	}
	if strings.Contains(string(raw), `"title"`) || strings.Contains(string(raw), `"description"`) {
		t.Fatal("legacy omitted optional fields changed", string(raw))
	}
}
