package commands

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestWorkspaceCompactReceiptRetainsIdentityWithoutDocumentaryPayload(t *testing.T) {
	ref := domain.EntityRef{Scope: domain.Scope{NamespaceID: domain.MustParseID("0199d210-0000-7000-8000-000000000001"), OutcomeID: domain.MustParseID("0199d210-0000-7000-8000-000000000002")}, Kind: domain.EntityKindWorkItem, ID: domain.MustParseID("0199d210-0000-7000-8000-000000000003")}
	criterion := domain.MustParseID("0199d210-0000-7000-8000-000000000004")
	description := strings.Repeat("documentary-content-should-not-be-in-a-receipt", 10000)
	raw, _ := json.Marshal(map[string]any{"command_id": criterion, "outcome_revision": json.Number("9007199254740993"), "idempotent_replay": true, "value": map[string]any{
		"refs": map[string]domain.EntityRef{"draft": ref}, "items": []any{map[string]any{"local_key": "draft", "ref": ref, "version": "9007199254740993", "criteria": []any{map[string]any{"id": criterion, "description": description}}}},
	}})
	receipt, err := CommitReceipt(json.RawMessage(raw))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(receipt)
	if err != nil || len(encoded) > 2048 || strings.Contains(string(encoded), "documentary-content") {
		t.Fatalf("receipt copied authored content: bytes=%d %v", len(encoded), err)
	}
	var actual struct {
		CreatedRefs  map[string]domain.EntityRef `json:"created_refs"`
		CreatedItems []struct {
			Version      string      `json:"version"`
			CriterionIDs []domain.ID `json:"criterion_ids"`
		} `json:"created_items"`
		OutcomeRevision json.Number `json:"outcome_revision"`
	}
	if err = json.Unmarshal(encoded, &actual); err != nil || actual.CreatedRefs["draft"] != ref || actual.CreatedItems[0].Version != "9007199254740993" || actual.CreatedItems[0].CriterionIDs[0] != criterion || string(actual.OutcomeRevision) != "9007199254740993" {
		t.Fatalf("identity or precision lost: %s %v", encoded, err)
	}
	compound := map[string]any{"command_id": criterion, "outcome_revision": 3, "idempotent_replay": false, "value": map[string]any{
		"evidence": map[string]any{"id": criterion, "description": description}, "artifact": map[string]any{"id": ref.ID, "uri": description}, "observation": map[string]any{"id": criterion, "body": description}, "decision": map[string]any{"id": criterion, "proposal": description}, "relation": map[string]any{"id": ref.ID}, "links": []any{map[string]any{"id": criterion, "rationale": description}},
	}}
	receipt, err = CommitReceipt(compound)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ = json.Marshal(receipt)
	if len(encoded) > 2048 || strings.Contains(string(encoded), "documentary-content") {
		t.Fatal("compound receipt retained authored payload")
	}
	for _, name := range []string{"evidence_id", "artifact_id", "observation_id", "decision_id", "relation_id", "evidence_link_ids"} {
		if _, ok := receipt[name]; !ok {
			t.Fatal("created identity missing", name)
		}
	}
}
