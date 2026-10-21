package commands

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func FuzzPublicCommandRoundTrip(f *testing.F) {
	f.Add("public title", int64(900))
	scope := domain.Scope{NamespaceID: domain.MustParseID("0199d094-0000-7000-8000-000000000001"), OutcomeID: domain.MustParseID("0199d094-0000-7000-8000-000000000002")}
	f.Fuzz(func(t *testing.T, title string, seconds int64) {
		if seconds < 0 || seconds > 86400 || len(title) > 4096 {
			return
		}
		cmd := application.ClaimWorkItemCommand{Scope: scope, WorkItemID: domain.MustParseID("0199d094-0000-7000-8000-000000000003"), ExpectedVersion: 1, TTL: time.Duration(seconds) * time.Second}
		raw, err := Encode(cmd)
		if err != nil {
			t.Fatal(err)
		}
		normalized, err := Normalize(raw, reflect.TypeFor[application.ClaimWorkItemCommand]())
		if err != nil {
			t.Fatal(err)
		}
		var decoded application.ClaimWorkItemCommand
		if err = json.Unmarshal(normalized, &decoded); err != nil || decoded != cmd {
			t.Fatalf("round trip: %+v %v", decoded, err)
		}
		schema := Schema(reflect.TypeFor[application.ConfigureTriggerCommand]())
		if _, err = json.Marshal(schema); err != nil {
			t.Fatal(err)
		}
	})
}

func TestPermissionListSchemasPublishThe128ValueLimit(t *testing.T) {
	assertMaxItems := func(label string, schema map[string]any, want int) {
		t.Helper()
		got, ok := schema["maxItems"].(int)
		if !ok || got != want {
			t.Fatalf("%s maxItems=%v, want %d", label, schema["maxItems"], want)
		}
	}
	adminSchema := Schema(reflect.TypeFor[application.AdministrativeIntent]())
	adminProps := adminSchema["properties"].(map[string]any)
	assertMaxItems("administrative permissions", adminProps["permissions"].(map[string]any), domain.MaxPermissionValues)

	signingSchema := Schema(reflect.TypeFor[application.SigningSecurityIntent]())
	signingProps := signingSchema["properties"].(map[string]any)
	policySchema := signingProps["credential_policy"].(map[string]any)
	policyProps := policySchema["properties"].(map[string]any)
	assertMaxItems("credential permitted_operations", policyProps["permitted_operations"].(map[string]any), domain.MaxPermissionValues)
}

func TestEvidenceOptionalMetadataPreservesCoreSerialization(t *testing.T) {
	schema := Schema(reflect.TypeFor[application.RegisterEvidenceCommand]())
	for _, name := range schema["required"].([]string) {
		if name == "checksum" || name == "source_version" {
			t.Fatalf("optional metadata became required: %s", name)
		}
	}
	// Existing Core receipts fingerprint this representation. Schema metadata must
	// not rename/omit Go JSON fields and invalidate an old idempotency receipt.
	raw, err := json.Marshal(application.RegisterEvidenceCommand{Scope: domain.Scope{NamespaceID: domain.MustParseID("01a11790-0000-7000-8000-000000000001"), OutcomeID: domain.MustParseID("01a11790-0000-7000-8000-000000000002")}})
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]any
	if err = json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	if fields["Checksum"] != "" || fields["SourceVersion"] != "" {
		t.Fatal("legacy Core fingerprint shape changed")
	}
}

func TestWorkActivitySchemaPreservesRequiredDescriptionAndDecimalPrecision(t *testing.T) {
	schema := Schema(reflect.TypeFor[application.AppendWorkActivityCommand]())
	properties := schema["properties"].(map[string]any)
	foundDescription := false
	for _, name := range schema["required"].([]string) {
		foundDescription = foundDescription || name == "description"
	}
	if !foundDescription || properties["expected_contract_version"].(map[string]any)["type"] != "string" {
		t.Fatal("activity contract must require description and quoted version")
	}
	if types := properties["type"].(map[string]any)["enum"].([]string); len(types) != 6 || types[0] != "progress" || types[5] != "result" {
		t.Fatalf("activity type contract: %+v", types)
	}
	command := application.AppendWorkActivityCommand{
		Scope:      domain.Scope{NamespaceID: domain.MustParseID("0199d094-0000-7000-8000-000000000001"), OutcomeID: domain.MustParseID("0199d094-0000-7000-8000-000000000002")},
		ContractID: domain.MustParseID("0199d094-0000-7000-8000-000000000003"), SignerKeyID: domain.MustParseID("0199d094-0000-7000-8000-000000000004"),
		Authority:       application.ContractAuthority{ExecutionID: domain.MustParseID("0199d094-0000-7000-8000-000000000005"), FencingToken: 9007199254740993, SpecDigest: "issued-specification"},
		AuthorityDigest: "issued-authority", ExpectedContractVersion: 9007199254740993, Type: domain.WorkActivityProgress, Description: "A bounded observation",
		CriterionRefs: []domain.WorkActivityCriterionRef{{CriterionID: domain.MustParseID("0199d094-0000-7000-8000-000000000006"), CriterionRevision: 9007199254740993}},
	}
	raw, err := Encode(command)
	if err != nil {
		t.Fatal(err)
	}
	var wire map[string]any
	if err = json.Unmarshal(raw, &wire); err != nil {
		t.Fatal(err)
	}
	if wire["expected_contract_version"] != "9007199254740993" || wire["authority"].(map[string]any)["fencing_token"] != "9007199254740993" || wire["criterion_refs"].([]any)[0].(map[string]any)["criterion_revision"] != "9007199254740993" {
		t.Fatalf("activity command lost decimal precision: %s", raw)
	}
	normalized, err := Normalize(raw, reflect.TypeFor[application.AppendWorkActivityCommand]())
	if err != nil {
		t.Fatal(err)
	}
	var decoded application.AppendWorkActivityCommand
	if err = json.Unmarshal(normalized, &decoded); err != nil || !reflect.DeepEqual(command, decoded) {
		t.Fatalf("activity command round trip: %+v %v", decoded, err)
	}
}
