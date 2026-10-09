package commands

import (
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	"reflect"
	"testing"
)

func TestSignedSchemasUseVerifiedDTOFieldsAndFlattenedBinding(t *testing.T) {
	schema := SignedSchema(reflect.TypeFor[signing.WorkReturnPayload[a.SignedReturnMaterial]]())
	properties := schema["properties"].(map[string]any)
	if _, nested := properties["Binding"]; nested {
		t.Fatal("binding was nested instead of matching signed bytes")
	}
	if properties["protocol_version"].(map[string]any)["const"] != 2 || properties["expected_contract_version"].(map[string]any)["type"] != "string" {
		t.Fatal("binding/counter schema differs")
	}
	material := properties["material"].(map[string]any)["properties"].(map[string]any)
	artifact := material["artifacts"].(map[string]any)["items"].(map[string]any)["properties"].(map[string]any)
	if _, known := artifact["LocalKey"]; !known {
		t.Fatal("opaque signed records were renamed by the unsigned command normalizer")
	}
	if _, renamed := artifact["local_key"]; renamed {
		t.Fatal("schema advertises a field the verified DTO does not accept")
	}
	envelope := SignedSchema(reflect.TypeFor[signing.Envelope]())["properties"].(map[string]any)
	if envelope["signatures"].(map[string]any)["maxItems"] != 1 {
		t.Fatal("envelope schema permits multiple signatures")
	}
}

func TestSignedFocalTechnicalExpansionsHaveSeparateBase64Bounds(t *testing.T) {
	properties := SignedSchema(reflect.TypeFor[a.SignedStateResult]())["properties"].(map[string]any)
	for field, maximum := range map[string]int{"operation_result_payload": 1398104, "material_payload": 245760} {
		schema := properties[field].(map[string]any)
		if schema["contentEncoding"] != "base64" || schema["maxLength"] != maximum {
			t.Fatalf("%s inherited semantic scalar bound instead of technical expansion bound", field)
		}
	}
}

func TestSignedStateCounterSchemasMatchExactWireAndLeaveUnsignedSchemasIntact(t *testing.T) {
	for _, typ := range []reflect.Type{reflect.TypeFor[d.Version](), reflect.TypeFor[d.OutcomeRevision](), reflect.TypeFor[d.CriterionRevision]()} {
		if SignedSchema(typ)["type"] != "string" {
			t.Fatal("signed counter advertised as JSON number", typ)
		}
		if Schema(typ)["type"] != "integer" {
			t.Fatal("historical unsigned counter schema changed", typ)
		}
	}
	properties := SignedSchema(reflect.TypeFor[a.SignedStateResult]())["properties"].(map[string]any)
	if properties["protocol_version"].(map[string]any)["type"] != "integer" {
		t.Fatal("literal signed protocol marker advertised as string")
	}
	protocol := SignedSchema(reflect.TypeFor[d.NamespaceWorkProtocol]())["properties"].(map[string]any)
	if protocol["protocol_version"].(map[string]any)["type"] != "string" || protocol["writer_epoch"].(map[string]any)["type"] != "integer" {
		t.Fatal("Namespace CAS and epoch conflated")
	}
}
