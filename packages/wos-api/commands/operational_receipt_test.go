package commands_test

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"github.com/google/jsonschema-go/jsonschema"
)

func operationalSchemaID(n byte) domain.ID {
	const prefix = "0199ed00-0000-7000-8000-0000000000"
	const hex = "0123456789abcdef"
	return domain.MustParseID(prefix + string([]byte{hex[n/16], hex[n%16]}))
}

func operationalSchemaScope() domain.Scope {
	return domain.Scope{NamespaceID: operationalSchemaID(1), OutcomeID: operationalSchemaID(2)}
}

func resolveOperationalSchema(t *testing.T, schema map[string]any) *jsonschema.Resolved {
	t.Helper()
	raw, err := json.Marshal(schema)
	if err != nil {
		t.Fatal(err)
	}
	var typed jsonschema.Schema
	if err := json.Unmarshal(raw, &typed); err != nil {
		t.Fatal(err)
	}
	resolved, err := typed.Resolve(nil)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

func TestOperationalCompactReceiptRetainsNativeReviewIdentityThroughTypedRecovery(t *testing.T) {
	scope := operationalSchemaScope()
	commandID, workID, deliveryID, withdrawalID := operationalSchemaID(10), operationalSchemaID(3), operationalSchemaID(8), operationalSchemaID(9)
	digest := "sha256:" + strings.Repeat("a", 64)
	private := strings.Repeat("authored-material-must-not-be-a-receipt", 10000)
	for _, replay := range []bool{false, true} {
		value := map[string]any{
			"command_id": commandID, "outcome_revision": json.Number("9007199254740993"), "idempotent_replay": replay,
			"value": map[string]any{
				"work_item":  map[string]any{"id": workID, "scope": scope, "description": private},
				"delivery":   map[string]any{"id": deliveryID, "scope": scope, "work_item_id": workID, "round": "18446744073709551615", "delivery_digest": digest, "material": private},
				"withdrawal": map[string]any{"id": withdrawalID, "scope": scope, "work_item_id": workID, "delivery_id": deliveryID, "delivery_digest": digest, "round": "18446744073709551615", "reason": private},
			},
		}
		receipt, err := commands.CommitReceipt(value)
		if err != nil {
			t.Fatal(err)
		}
		raw, err := json.Marshal(receipt)
		if err != nil || len(raw) > 2048 || strings.Contains(string(raw), "authored-material") {
			t.Fatalf("receipt retained private or unbounded result material: %d %v", len(raw), err)
		}
		var recovered wossdk.CommandResult[application.TaskDeliveryWithdrawalResult]
		if err := json.Unmarshal(raw, &recovered); err != nil {
			t.Fatal(err)
		}
		if !recovered.ResultOmitted || recovered.IdempotentReplay != replay || recovered.CommandID != commandID || recovered.OutcomeRevision != 9007199254740993 || recovered.DeliveryID == nil || *recovered.DeliveryID != deliveryID || recovered.DeliveryDigest != digest || recovered.DeliveryRound != math.MaxUint64 || recovered.WithdrawalID == nil || *recovered.WithdrawalID != withdrawalID {
			t.Fatalf("typed recovery lost its exact canonical review target: %+v", recovered)
		}
		if recovered.Value.Delivery.ID != "" || recovered.Value.WorkItem.ID != "" {
			t.Fatal("compact identity was presented as a complete command result")
		}
		validator := resolveOperationalSchema(t, commands.WorkspaceCommandResults()["withdraw_task_delivery"])
		var instance map[string]any
		_ = json.Unmarshal(raw, &instance)
		if err := validator.Validate(instance); err != nil {
			t.Fatal("compact operational receipt disagrees with its output schema", err)
		}
	}
}

func TestOperationalCompactCompletionReceiptRetainsOnlyBoundAuthorityIdentity(t *testing.T) {
	scope, objectiveID, authorityID := operationalSchemaScope(), operationalSchemaID(3), operationalSchemaID(9)
	value := map[string]any{
		"command_id": operationalSchemaID(10), "outcome_revision": 3, "idempotent_replay": false,
		"value": map[string]any{
			"objective": map[string]any{"id": objectiveID, "scope": scope},
			"authority": map[string]any{"id": authorityID, "scope": scope, "objective_id": objectiveID, "delegator_credential_id": operationalSchemaID(8), "reason": "private delegation material"},
		},
	}
	receipt, err := commands.CommitReceipt(value)
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(receipt)
	var recovered wossdk.CommandResult[application.ObjectiveCompletionModeChange]
	if err := json.Unmarshal(raw, &recovered); err != nil || recovered.CompletionAuthorityID == nil || *recovered.CompletionAuthorityID != authorityID || strings.Contains(string(raw), "delegator_credential") || strings.Contains(string(raw), "private delegation") {
		t.Fatalf("completion identity missing or private delegation leaked: %s %v", raw, err)
	}
	value["value"].(map[string]any)["authority"].(map[string]any)["objective_id"] = operationalSchemaID(11)
	receipt, err = commands.CommitReceipt(value)
	if err != nil {
		t.Fatal(err)
	}
	if _, exists := receipt["completion_authority_id"]; exists {
		t.Fatal("an authority from another Objective was exposed as the recovery target")
	}
}

func TestOperationalCompactReceiptDoesNotConfuseIntegrationDeliveryOrDifferentWithdrawal(t *testing.T) {
	value := map[string]any{"command_id": operationalSchemaID(10), "outcome_revision": 3, "idempotent_replay": false, "value": map[string]any{"delivery": map[string]any{"id": operationalSchemaID(8), "status": "pending", "payload": "integration-only"}}}
	receipt, err := commands.CommitReceipt(value)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"delivery_id", "delivery_digest", "delivery_round"} {
		if _, exists := receipt[key]; exists {
			t.Fatal("integration delivery was interpreted as a TaskDelivery", key)
		}
	}
	result := value["value"].(map[string]any)
	delivery := map[string]any{"id": operationalSchemaID(8), "scope": operationalSchemaScope(), "work_item_id": operationalSchemaID(3), "round": "1", "delivery_digest": "sha256:" + strings.Repeat("a", 64)}
	result["delivery"] = delivery
	withdrawal := map[string]any{}
	for key, val := range delivery {
		withdrawal[key] = val
	}
	withdrawal["id"], withdrawal["delivery_id"] = operationalSchemaID(9), operationalSchemaID(11)
	result["withdrawal"] = withdrawal
	receipt, err = commands.CommitReceipt(value)
	if err != nil {
		t.Fatal(err)
	}
	if receipt["delivery_id"] != domain.ID(operationalSchemaID(8)) {
		t.Fatal("valid current delivery identity was lost")
	}
	if _, exists := receipt["withdrawal_id"]; exists {
		t.Fatal("withdrawal for a different delivery was exposed")
	}
	before, _ := json.Marshal(receipt)
	replayed, err := commands.CommitReceipt(value)
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(replayed)
	if !reflect.DeepEqual(before, after) {
		t.Fatal("deterministic receipt compaction changed its review target")
	}
}
