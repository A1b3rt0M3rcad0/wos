package application

import (
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"strings"
	"testing"
)

func TestWorkspaceProjectionRoundTripPreservesExactNestedVersions(t *testing.T) {
	const high = d.Version(9007199254740993)
	id := d.MustParseID("0199a555-0000-7000-8000-000000000001")
	scope := d.Scope{NamespaceID: id, OutcomeID: id}
	ref := d.EntityRef{Scope: scope, Kind: d.EntityKindWorkItem, ID: id}
	v := EntityWorkspaceView{Subject: WorkspaceSubject{Kind: "entity", Ref: ref}, EntityVersion: high, OutcomeRevision: d.OutcomeRevision(high), Entity: map[string]any{"version": high, "last_fencing_token": uint64(high)}, Placement: WorkspacePlacement{Outcome: EntitySummary{Ref: ref, Version: high}}, Criteria: []WorkspaceCriterion{{Definition: d.SuccessCriterion{ID: id, OwnerRef: ref, Revision: d.CriterionRevision(high)}}}, Attention: &WorkAttention{WorkItemID: id, OperationalState: d.WorkItemOperationalState{Ref: ref}, OutcomeRevision: d.OutcomeRevision(high)}, Capabilities: []WorkspaceCapability{{Subject: WorkspaceSubject{Kind: "entity", Ref: ref}, ObservedVersion: high}}}
	raw, e := json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	if strings.Contains(string(raw), ":9007199254740993") {
		t.Fatalf("unsafe numeric version: %s", raw)
	}
	var got EntityWorkspaceView
	if e = json.Unmarshal(raw, &got); e != nil {
		t.Fatal(e)
	}
	if got.EntityVersion != high || got.OutcomeRevision != d.OutcomeRevision(high) || got.Placement.Outcome.Version != high || got.Criteria[0].Definition.Revision != d.CriterionRevision(high) || got.Capabilities[0].ObservedVersion != high || got.Attention.OutcomeRevision != d.OutcomeRevision(high) {
		t.Fatalf("projection version changed: %+v", got)
	}
	entity, ok := got.Entity.(map[string]any)
	if !ok || entity["version"] != "9007199254740993" {
		t.Fatalf("interface record lost exact string: %#v", got.Entity)
	}
	v.Entity = map[string]any{"version": high, "external_context": map[string]any{"schema_version": json.Number("9007199254740993")}, "measurement": json.RawMessage(`{"version":3}`)}
	raw, e = json.Marshal(v)
	if e != nil {
		t.Fatal(e)
	}
	if !strings.Contains(string(raw), `"schema_version":9007199254740993`) || !strings.Contains(string(raw), `"measurement":{"version":3}`) {
		t.Fatalf("authored data was rewritten: %s", raw)
	}
	cards := WorkspaceCardsPage{Items: []WorkspaceCardSummary{{Ref: ref, Version: high}}, OutcomeRevision: d.OutcomeRevision(high)}
	raw, e = json.Marshal(cards)
	if e != nil {
		t.Fatal(e)
	}
	var cardGot WorkspaceCardsPage
	if e = json.Unmarshal(raw, &cardGot); e != nil {
		t.Fatal(e)
	}
	if cardGot.Items[0].Version != high {
		t.Fatal("card version rounded")
	}
}

func TestWorkspaceSubjectRejectsConfusedAndForeignSelectors(t *testing.T) {
	id := d.MustParseID("0199a555-0000-7000-8000-000000000001")
	other := d.MustParseID("0199a555-0000-7000-8000-000000000002")
	scope := d.Scope{NamespaceID: id, OutcomeID: id}
	ref := d.EntityRef{Scope: scope, Kind: d.EntityKindWorkItem, ID: id}
	for _, subject := range []WorkspaceSubject{{Kind: "entity", Ref: ref, CriterionID: &other}, {Kind: "criterion", Ref: ref}, {Kind: "execution_record", Ref: ref, RecordID: &other, RecordKind: "note"}, {Kind: "roadmap_revision", Ref: ref, RevisionNumber: 1}, {Kind: "entity", Ref: d.EntityRef{Scope: d.Scope{NamespaceID: other, OutcomeID: id}, Kind: d.EntityKindWorkItem, ID: id}}} {
		if e := subject.Validate(scope); e == nil {
			t.Fatalf("confused selector accepted: %+v", subject)
		}
	}
}
