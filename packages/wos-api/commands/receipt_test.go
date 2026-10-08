package commands

import (
	"encoding/json"
	"testing"
)

func TestCommitReceiptKeepsAcquisitionIdentityForBoundedRecovery(t *testing.T) {
	raw := json.RawMessage(`{"command_id":"command","outcome_revision":8,"value":{"acquired":true,"result":{"contract":{"id":"contract","work_item_id":"work","version":3,"lease_version":4,"fencing_token":"9007199254740993","scope":{"namespace_id":"namespace","outcome_id":"outcome"}},"work_item":{"private":"omit"}}}}`)
	receipt, err := CommitReceipt(raw)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(receipt)
	var decoded map[string]any
	json.Unmarshal(encoded, &decoded)
	if decoded["contract_id"] != "contract" || decoded["work_item_id"] != "work" || decoded["fencing_token"] != "9007199254740993" || decoded["result_omitted"] != true {
		t.Fatalf("recovery identity missing %s", encoded)
	}
	if _, ok := decoded["work_item"]; ok {
		t.Fatal("receipt leaked full result")
	}
}
