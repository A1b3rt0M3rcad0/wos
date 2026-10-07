package commands

import (
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"reflect"
	"testing"
	"time"
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
