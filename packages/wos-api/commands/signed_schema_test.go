package commands

import (
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
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
