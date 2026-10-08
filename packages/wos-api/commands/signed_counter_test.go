package commands

import (
	"encoding/json"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"math"
	"reflect"
	"testing"
)

func TestSignedCommandVersionsAreExactStrings(t *testing.T) {
	id := d.MustParseID("0199a555-0000-7000-8000-000000000001")
	cmd := a.AcquireSignedWorkContractCommand{Scope: d.Scope{NamespaceID: id, OutcomeID: id}, WorkItemID: id, SignerKeyID: id, ExpectedWorkItemVersion: d.Version(math.MaxUint64)}
	raw, err := Encode(cmd)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err = json.Unmarshal(raw, &wire); err != nil || wire["expected_work_item_version"] != "18446744073709551615" {
		t.Fatal("SDK/public encoder rounded or emitted numeric version", string(raw), err)
	}
	normalized, err := Normalize(raw, reflect.TypeOf(cmd))
	if err != nil {
		t.Fatal(err)
	}
	var decoded a.AcquireSignedWorkContractCommand
	if err = Decode(normalized, &decoded); err != nil || decoded.ExpectedWorkItemVersion != cmd.ExpectedWorkItemVersion {
		t.Fatal("wire round trip lost exact version", err)
	}
	wire["expected_work_item_version"] = json.Number("9007199254740993")
	numeric, err := json.Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = Normalize(numeric, reflect.TypeOf(cmd)); err == nil {
		t.Fatal("signed expected version accepted a numeric counter")
	}
	schema := Schema(reflect.TypeOf(cmd))["properties"].(map[string]any)
	if schema["expected_work_item_version"].(map[string]any)["type"] != "string" || Schema(reflect.TypeFor[signing.Decimal]())["type"] != "string" {
		t.Fatal("schema contradicts exact decimal representation")
	}
	legacy, err := Encode(a.ClaimWorkItemCommand{Scope: cmd.Scope, WorkItemID: id, ExpectedVersion: 1})
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(legacy, &wire); err != nil {
		t.Fatal(err)
	}
	if _, numeric := wire["expected_version"].(float64); !numeric {
		t.Fatal("v1 encoder changed")
	}
}
