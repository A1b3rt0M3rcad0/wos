package application

import (
	"encoding/json"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
)

func TestSignedReturnReplayPermissionsFailClosedOnReceiptBindingMismatch(t *testing.T) {
	scope := domain.Scope{
		NamespaceID: "0199e200-0000-7000-8000-000000000001",
		OutcomeID:   "0199e200-0000-7000-8000-000000000002",
	}
	contractID := domain.ID("0199e200-0000-7000-8000-000000000003")
	requestID := domain.ID("0199e200-0000-7000-8000-000000000004")
	workItemID := domain.ID("0199e200-0000-7000-8000-000000000005")
	const idempotencyKey = "signed-return-replay-001"
	const requestDigest = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

	encodeStored := func(receipt signing.ReceiptPayload) domain.StoredCommandResult {
		t.Helper()
		payload, err := json.Marshal(receipt)
		if err != nil {
			t.Fatal(err)
		}
		response, err := json.Marshal(MutationResult[SignedReturnResult]{
			CommandID: "0199e200-0000-7000-8000-000000000006",
			Value:     SignedReturnResult{Receipt: signing.Document{Payload: payload}},
		})
		if err != nil {
			t.Fatal(err)
		}
		return domain.StoredCommandResult{ResponseJSON: response}
	}

	valid := signing.ReceiptPayload{
		Binding:        signing.Binding{NamespaceID: scope.NamespaceID.String(), OutcomeID: scope.OutcomeID.String()},
		RequestID:      requestID.String(),
		IdempotencyKey: idempotencyKey,
		RequestDigest:  requestDigest,
		ContractID:     contractID.String(),
		WorkItemID:     workItemID.String(),
		Accepted:       true,
		Disposition:    "completed",
	}
	permissions, err := signedReturnStoredCompletionPermissions(scope, contractID, requestID, idempotencyKey, workItemID, requestDigest, encodeStored(valid))
	if err != nil {
		t.Fatalf("valid receipt rejected: %v", err)
	}
	if len(permissions) != len(signedDirectCompletionPermissions) {
		t.Fatalf("valid completion permissions = %v", permissions)
	}
	for i := range permissions {
		if permissions[i] != signedDirectCompletionPermissions[i] {
			t.Fatalf("valid completion permissions = %v, want %v", permissions, signedDirectCompletionPermissions)
		}
	}

	tests := []struct {
		name   string
		mutate func(*signing.ReceiptPayload)
	}{
		{name: "namespace", mutate: func(receipt *signing.ReceiptPayload) { receipt.NamespaceID = "0199e200-0000-7000-8000-000000000099" }},
		{name: "outcome", mutate: func(receipt *signing.ReceiptPayload) { receipt.OutcomeID = "0199e200-0000-7000-8000-000000000099" }},
		{name: "contract", mutate: func(receipt *signing.ReceiptPayload) { receipt.ContractID = "0199e200-0000-7000-8000-000000000099" }},
		{name: "task", mutate: func(receipt *signing.ReceiptPayload) { receipt.WorkItemID = "0199e200-0000-7000-8000-000000000099" }},
		{name: "request", mutate: func(receipt *signing.ReceiptPayload) { receipt.RequestID = "0199e200-0000-7000-8000-000000000099" }},
		{name: "idempotency key", mutate: func(receipt *signing.ReceiptPayload) { receipt.IdempotencyKey = "another-key" }},
		{name: "request digest", mutate: func(receipt *signing.ReceiptPayload) {
			receipt.RequestDigest = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
		}},
		{name: "unaccepted", mutate: func(receipt *signing.ReceiptPayload) { receipt.Accepted = false }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			receipt := valid
			test.mutate(&receipt)
			if _, err := signedReturnStoredCompletionPermissions(scope, contractID, requestID, idempotencyKey, workItemID, requestDigest, encodeStored(receipt)); err == nil {
				t.Fatal("mismatched receipt binding did not fail closed")
			}
		})
	}

	valid.Disposition = "delivered_for_review"
	permissions, err = signedReturnStoredCompletionPermissions(scope, contractID, requestID, idempotencyKey, workItemID, requestDigest, encodeStored(valid))
	if err != nil || len(permissions) != 0 {
		t.Fatalf("review delivery replay permissions = %v, err=%v; want no completion rights", permissions, err)
	}
}

func TestSignedReturnReplayPermissionsRejectUnknownDisposition(t *testing.T) {
	scope := domain.Scope{NamespaceID: "0199e200-0000-7000-8000-000000000001", OutcomeID: "0199e200-0000-7000-8000-000000000002"}
	contractID := domain.ID("0199e200-0000-7000-8000-000000000003")
	requestID := domain.ID("0199e200-0000-7000-8000-000000000004")
	workItemID := domain.ID("0199e200-0000-7000-8000-000000000005")
	payload, err := json.Marshal(signing.ReceiptPayload{
		Binding:        signing.Binding{NamespaceID: scope.NamespaceID.String(), OutcomeID: scope.OutcomeID.String()},
		RequestID:      requestID.String(),
		IdempotencyKey: "signed-return-replay-001",
		RequestDigest:  "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		ContractID:     contractID.String(),
		WorkItemID:     workItemID.String(),
		Accepted:       true,
		Disposition:    "unknown",
	})
	if err != nil {
		t.Fatal(err)
	}
	response, err := json.Marshal(MutationResult[SignedReturnResult]{
		CommandID: "0199e200-0000-7000-8000-000000000006",
		Value:     SignedReturnResult{Receipt: signing.Document{Payload: payload}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := signedReturnStoredCompletionPermissions(scope, contractID, requestID, "signed-return-replay-001", workItemID, "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", domain.StoredCommandResult{ResponseJSON: response}); err == nil {
		t.Fatal("unknown replay disposition did not fail closed")
	}
}
