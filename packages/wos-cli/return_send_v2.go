package woscli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"os"
	"path/filepath"
)

func returnPendingV2(operation string) bool {
	return operation == "ReturnSignedWork" || operation == "ReturnSignedReview"
}
func decodeReturnIntentV2(profile ProfileV2, token []byte, intent PendingOperationV2) (FrozenReturnV2, signing.RequestBinding, error) {
	var frozen FrozenReturnV2
	var binding signing.RequestBinding
	raw, e := base64.StdEncoding.Strict().DecodeString(intent.Payload)
	if e != nil || len(raw) > 256<<10 || signing.Digest(raw) != intent.PayloadDigest {
		return frozen, binding, fmt.Errorf("frozen intention bytes differ")
	}
	if e = signing.DecodeStrict(raw, &frozen, 256<<10); e != nil {
		return frozen, binding, e
	}
	binding, e = verifyFrozenV2(profile, token, frozen)
	if e != nil {
		return frozen, binding, e
	}
	if intent.ID.String() != binding.RequestID || intent.ContractID == nil || intent.ContractID.String() != binding.ContractID || intent.Scope.NamespaceID.String() != binding.NamespaceID || intent.Scope.OutcomeID.String() != binding.OutcomeID || intent.IdempotencyKey != binding.IdempotencyKey || !returnPendingV2(intent.Operation) || (intent.Operation == "ReturnSignedReview") != (frozen.Envelope.PayloadType == signing.ReviewReturn) {
		return frozen, binding, fmt.Errorf("profile return intention binding differs")
	}
	return frozen, binding, nil
}
func verifyReturnReceiptV2(profile ProfileV2, frozen FrozenReturnV2, binding signing.RequestBinding, document signing.Document) (signing.ReceiptPayload, error) {
	var receipt signing.ReceiptPayload
	raw, e := verifyProfileDocumentV2(profile, document, signing.AcceptanceReceipt)
	if e != nil {
		return receipt, e
	}
	if e = signing.DecodeStrict(raw, &receipt, signing.MaxPayloadBytes); e != nil {
		return receipt, e
	}
	if receipt.ProtocolVersion != 2 || receipt.ServerID != binding.ServerID || receipt.NamespaceID != binding.NamespaceID || receipt.OutcomeID != binding.OutcomeID || receipt.PrincipalID != binding.PrincipalID || receipt.ContractID != binding.ContractID || receipt.WorkItemID != binding.WorkItemID || receipt.RequestID != binding.RequestID || receipt.IdempotencyKey != binding.IdempotencyKey || receipt.RequestDigest != frozen.RequestDigest || !receipt.Accepted || !receipt.LocalObligationClosed {
		return receipt, fmt.Errorf("acceptance receipt does not close this exact intention")
	}
	if frozen.Envelope.PayloadType == signing.WorkReturn {
		if receipt.ContractStatus == "completed" {
			if receipt.Disposition != "completed" {
				return receipt, fmt.Errorf("completion disposition differs")
			}
		} else if receipt.ContractStatus == "delivered" {
			if receipt.Disposition != "delivered_for_review" {
				return receipt, fmt.Errorf("delivery disposition differs")
			}
		} else {
			return receipt, fmt.Errorf("execution obligation remains open")
		}
	} else {
		if receipt.ContractStatus != "completed" {
			return receipt, fmt.Errorf("review obligation remains open")
		}
		payload, _ := base64.StdEncoding.Strict().DecodeString(frozen.Envelope.Payload)
		var request signing.ReviewReturnPayload[a.SignedReviewMaterial]
		if e = signing.DecodeStrict(payload, &request, 180<<10); e != nil {
			return receipt, e
		}
		if receipt.ReviewCaseID != request.ReviewCaseID || receipt.SubmissionID != request.SubmissionID || receipt.SubmissionDigest != request.SubmissionDigest || receipt.DecisionID == "" {
			return receipt, fmt.Errorf("review receipt target differs")
		}
		expected := map[string]string{"approved": "review_approved", "changes_requested": "changes_requested", "inconclusive": "review_inconclusive"}[request.Decision]
		if expected == "" || receipt.Disposition != expected {
			return receipt, fmt.Errorf("review receipt disposition differs")
		}
	}
	return receipt, nil
}
func lookupReturnReceiptV2(ctx context.Context, client *sdk.Client, profile ProfileV2, intent PendingOperationV2, frozen FrozenReturnV2, binding signing.RequestBinding) (signing.Document, error) {
	var document signing.Document
	state, e := client.ReadSignedState(ctx, a.SignedStateQuery{Scope: d.Scope{NamespaceID: intent.Scope.NamespaceID, OutcomeID: intent.Scope.OutcomeID}, Resource: "receipt", IdempotencyKey: intent.IdempotencyKey})
	if e != nil {
		return document, e
	}
	if state.Envelope == nil {
		return document, fmt.Errorf("acceptance proof unavailable")
	}
	document, e = signing.ToDocument(*state.Envelope)
	if e != nil {
		return document, e
	}
	_, e = verifyReturnReceiptV2(profile, frozen, binding, document)
	return document, e
}
func notFoundReturnV2(e error) bool {
	code, _ := d.ErrorCodeOf(e)
	var remote *sdk.Error
	return code == d.ErrorCodeNotFound || errors.As(e, &remote) && remote.Code == string(d.ErrorCodeNotFound)
}

// Only an authenticated original file can repair a lost local profile write.
// Neither an unsigned _local hint nor an issuer receipt binds another CID.
func findReturnIntentV2(ctx context.Context, w *Workspace, profile ProfileV2, token []byte, actor d.ActorRef, id d.ID, kind string) (PendingOperationV2, error) {
	var intent PendingOperationV2
	e := mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
		for _, candidate := range current.Local.PendingOperations {
			if candidate.ContractID != nil && *candidate.ContractID == id {
				if candidate.Operation != returnOperationV2(kind) {
					return fmt.Errorf("another unresolved intention requires recovery")
				}
				intent = candidate
				_, _, e := decodeReturnIntentV2(*current, token, intent)
				return e
			}
		}
		unlock, e := w.LockV2(ctx, current.Name, id.String())
		if e != nil {
			return e
		}
		defer unlock()
		file, view, _, _, e := loadContractV2(w, *current, id)
		if e != nil {
			return e
		}
		if view.Authority.ContractKind != kind || file.Local.Pending == nil {
			return fmt.Errorf("sign the contract before sending")
		}
		binding, e := verifyFrozenV2(*current, token, *file.Local.Pending)
		if e != nil {
			return e
		}
		if binding.ContractID != id.String() || file.Local.Pending.State != "prepared_signed" {
			return fmt.Errorf("missing profile intention requires original-state reconciliation")
		}
		if len(current.Local.PendingOperations) >= 10 {
			return fmt.Errorf("pending limit reached")
		}
		intent, e = returnIntentV2(*current, *file.Local.Pending, binding, actor)
		if e != nil {
			return e
		}
		current.Local.PendingOperations = append(current.Local.PendingOperations, intent)
		return nil
	})
	return intent, e
}
func updateReturnStateV2(ctx context.Context, w *Workspace, profile ProfileV2, token []byte, intent PendingOperationV2, state string, receipt *signing.Document) error {
	return mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
		index, e := pendingIndexV2(current, intent)
		if e != nil {
			return e
		}
		frozen, binding, e := decodeReturnIntentV2(*current, token, intent)
		if e != nil {
			return e
		}
		response := ""
		if receipt != nil {
			if _, e = verifyReturnReceiptV2(*current, frozen, binding, *receipt); e != nil {
				return e
			}
			raw, e := json.Marshal(receipt)
			if e != nil {
				return e
			}
			response = base64.StdEncoding.EncodeToString(raw)
		}
		// Persist confirmed acceptance in the profile before file edits/cleanup.
		current.Local.PendingOperations[index].State = state
		if response != "" {
			current.Local.PendingOperations[index].Response = response
		}
		return nil
	})
}
func returnReceiptFromIntentV2(profile ProfileV2, token []byte, intent PendingOperationV2) (signing.Document, error) {
	var document signing.Document
	frozen, binding, e := decodeReturnIntentV2(profile, token, intent)
	if e != nil {
		return document, e
	}
	raw, e := base64.StdEncoding.Strict().DecodeString(intent.Response)
	if e != nil {
		return document, e
	}
	if e = signing.DecodeStrict(raw, &document, 256<<10); e != nil {
		return document, e
	}
	_, e = verifyReturnReceiptV2(profile, frozen, binding, document)
	return document, e
}
func cleanupReturnV2(ctx context.Context, w *Workspace, profile ProfileV2, token []byte, intent PendingOperationV2, receipt signing.Document) error {
	return cleanupReturnWithHookV2(ctx, w, profile, token, intent, receipt, nil)
}

func cleanupReturnWithHookV2(ctx context.Context, w *Workspace, profile ProfileV2, token []byte, intent PendingOperationV2, receipt signing.Document, hook func(string) error) error {
	return mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
		index, e := pendingIndexV2(current, intent)
		if e != nil {
			return e
		}
		frozen, binding, e := decodeReturnIntentV2(*current, token, intent)
		if e != nil {
			return e
		}
		if _, e = verifyReturnReceiptV2(*current, frozen, binding, receipt); e != nil {
			return e
		}
		// Receipt must have already been persisted, so death after unlink can resume.
		persisted, e := returnReceiptFromIntentV2(*current, token, current.Local.PendingOperations[index])
		if e != nil {
			return e
		}
		if signing.Digest(persisted.Payload) != signing.Digest(receipt.Payload) {
			return fmt.Errorf("persisted receipt differs")
		}
		for i, p := range current.Local.PendingOperations {
			if i != index && p.ContractID != nil && *p.ContractID == *intent.ContractID {
				return fmt.Errorf("another pending intention prevents cleanup")
			}
		}
		unlock, e := w.LockV2(ctx, current.Name, binding.ContractID)
		if e != nil {
			return e
		}
		defer unlock()
		file, view, path, observed, e := loadContractV2(w, *current, *intent.ContractID)
		if errors.Is(e, os.ErrNotExist) {
			current.Local.PendingOperations = append(current.Local.PendingOperations[:index], current.Local.PendingOperations[index+1:]...)
			return nil
		}
		if e != nil {
			return e
		}
		if view.Authority.ContractID != binding.ContractID || file.Local.Pending == nil {
			return fmt.Errorf("local frozen return missing or replaced; preserve file")
		}
		localBinding, e := verifyFrozenV2(*current, token, *file.Local.Pending)
		if e != nil {
			return e
		}
		if localBinding.RequestID != binding.RequestID || file.Local.Pending.RequestDigest != frozen.RequestDigest {
			return fmt.Errorf("local frozen return replaced; preserve file")
		}
		draft, e := file.DraftDigest()
		if e != nil {
			return e
		}
		file.Local.AcceptanceReceipt = &receipt
		file.Local.Pending.State = "cleanup_pending"
		encoded, e := json.Marshal(receipt)
		if e != nil {
			return e
		}
		file.Local.Pending.Response = base64.StdEncoding.EncodeToString(encoded)
		if e = sealFrozenV2(*current, token, file.Local.Pending); e != nil {
			return e
		}
		if e = w.WriteV2(path, file, signing.Digest(observed)); e != nil {
			return e
		}
		if hook != nil {
			if e = hook("receipt_persisted"); e != nil {
				return e
			}
		}
		if draft != frozen.DraftDigest {
			return fmt.Errorf("remote return accepted; unconfirmed draft edits preserved, cleanup pending")
		}
		// Re-read immediately before the confined single-file unlink. Cooperative
		// writers share these locks; this is not isolation from a hostile OS writer.
		latest, _, _, bytes, e := loadContractV2(w, *current, *intent.ContractID)
		if e != nil {
			return e
		}
		latestDraft, e := latest.DraftDigest()
		if e != nil || latestDraft != draft || latest.Local.Pending == nil || latest.Local.Pending.RequestDigest != frozen.RequestDigest {
			return fmt.Errorf("draft changed before cleanup; preserve file")
		}
		expected, e := EncodeV2Document(file)
		if e != nil || signing.Digest(bytes) != signing.Digest(expected) {
			return fmt.Errorf("local file changed before cleanup; preserve file")
		}
		if e = w.root.Remove(path); e != nil {
			return e
		}
		if hook != nil {
			if e = hook("unlinked"); e != nil {
				return e
			}
		}
		if e = w.syncV2Directory(filepath.Dir(path)); e != nil {
			return e
		}
		current.Local.PendingOperations = append(current.Local.PendingOperations[:index], current.Local.PendingOperations[index+1:]...)
		return nil
	})
}
func recoverReturnV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, token []byte, intent PendingOperationV2, allowPrepared bool) (signing.Document, bool, error) {
	var receipt signing.Document
	frozen, binding, e := decodeReturnIntentV2(profile, token, intent)
	if e != nil {
		return receipt, false, e
	}
	if intent.State == "prepared_signed" && !allowPrepared {
		return receipt, false, fmt.Errorf("prepared signature has not been sent; use send explicitly")
	}
	if intent.State == "accepted_confirmed" || intent.State == "cleanup_pending" {
		receipt, e = returnReceiptFromIntentV2(profile, token, intent)
		if e != nil {
			return receipt, false, e
		}
		e = cleanupReturnV2(ctx, w, profile, token, intent, receipt)
		return receipt, true, e
	}
	if intent.State != "prepared_signed" && intent.State != "sent_unknown" {
		return receipt, false, fmt.Errorf("unsupported return recovery state")
	}
	if e = updateReturnStateV2(ctx, w, profile, token, intent, "sent_unknown", nil); e != nil {
		return receipt, false, e
	}
	receipt, e = lookupReturnReceiptV2(ctx, client, profile, intent, frozen, binding)
	if e != nil && !notFoundReturnV2(e) {
		return receipt, false, e
	}
	if e != nil {
		var result sdk.CommandResult[a.SignedReturnResult]
		if intent.Operation == "ReturnSignedWork" {
			result, e = client.ReturnSignedWork(ctx, intent.IdempotencyKey, a.ReturnSignedWorkCommand{Envelope: frozen.Envelope})
		} else {
			result, e = client.ReturnSignedReview(ctx, intent.IdempotencyKey, a.ReturnSignedReviewCommand{Envelope: frozen.Envelope})
		}
		if e != nil {
			return receipt, false, e
		}
		if result.ResultOmitted {
			receipt, e = lookupReturnReceiptV2(ctx, client, profile, intent, frozen, binding)
		} else {
			receipt = result.Value.Receipt
			_, e = verifyReturnReceiptV2(profile, frozen, binding, receipt)
		}
		if e != nil {
			return receipt, false, e
		}
	}
	if e = updateReturnStateV2(ctx, w, profile, token, intent, "accepted_confirmed", &receipt); e != nil {
		return receipt, true, e
	}
	e = cleanupReturnV2(ctx, w, profile, token, intent, receipt)
	return receipt, true, e
}
func returnCommandV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, token []byte, identity a.SigningIdentityView, o options) (Output, error) {
	result := Output{Operation: o.args[0] + " " + o.args[1]}
	if len(o.args) != 3 {
		return result, usage("sign/send/finish requires one contract ID")
	}
	id, e := d.ParseID(o.args[2])
	if e != nil {
		return result, e
	}
	result.ContractID = id
	kind := "execution"
	if o.args[0] == "review" {
		kind = "review"
	}
	if o.args[1] == "sign" {
		intent, e := prepareReturnV2(ctx, w, profile, client, token, identity, id, kind, o.values["completion"])
		if e == nil {
			result.Data = map[string]any{"contract_id": id, "intention_id": intent.ID, "state": "prepared_signed", "sent": false}
		}
		return result, e
	}
	if o.args[1] == "finish" {
		file, _, _, _, readErr := loadContractV2(w, profile, id)
		if readErr == nil && file.Local.Pending == nil {
			_, e = prepareReturnV2(ctx, w, profile, client, token, identity, id, kind, o.values["completion"])
			if e != nil {
				return result, e
			}
		}
	}
	intent, e := findReturnIntentV2(ctx, w, profile, token, identity.Actor, id, kind)
	if e != nil {
		return result, e
	}
	receipt, committed, e := recoverReturnV2(ctx, w, profile, client, token, intent, true)
	result.Committed = committed
	if committed {
		result.Data = map[string]any{"contract_id": id, "remote_committed": true, "cleanup_pending": e != nil, "acceptance_digest": signing.Digest(receipt.Payload)}
	}
	if e != nil {
		result.RequiresAction = "preserve original frozen return; recover it without resigning, renewing or changing CAS"
	}
	return result, e
}
