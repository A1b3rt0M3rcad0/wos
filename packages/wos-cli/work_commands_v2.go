package woscli

import (
	"context"
	"fmt"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"path/filepath"
	"sort"
	"strconv"
)

func scopeV2(w *Workspace, profile ProfileV2, o options) (d.Scope, error) {
	project, e := w.LoadProjectV2()
	if e != nil {
		return d.Scope{}, e
	}
	scope := d.Scope{NamespaceID: profile.Binding.NamespaceID, OutcomeID: project.Scope.DefaultOutcomeID}
	if value := o.values["outcome"]; value != "" {
		id, e := d.ParseID(value)
		if e != nil {
			return scope, e
		}
		scope.OutcomeID = id
	}
	if value := o.values["namespace"]; value != "" && value != scope.NamespaceID.String() {
		return scope, usage("profile Namespace is immutable")
	}
	return scope, scope.Validate()
}
func loadContractV2(w *Workspace, profile ProfileV2, id d.ID) (ContractFileV2, ContractViewV2, string, []byte, error) {
	var file ContractFileV2
	var view ContractViewV2
	if id.Validate() != nil {
		return file, view, "", nil, usage("valid contract ID required")
	}
	path, e := w.DocumentPath(filepath.Join(".wos/profiles", profile.Name, "contract", id.String()))
	if e != nil {
		return file, view, path, nil, e
	}
	raw, e := w.ReadV2(path)
	if e != nil {
		return file, view, path, raw, e
	}
	if _, legacy, e := readLegacyUnsignedV2(w, profile, id); e != nil {
		return file, view, path, raw, e
	} else if legacy {
		return file, view, path, raw, fmt.Errorf("legacy_unsigned record cannot authorize signed mutation; original v1 recovery remains in the preserved source workspace")
	}
	if e = DecodeV2Document(raw, &file); e != nil {
		return file, view, path, raw, e
	}
	view, e = file.VerifyIssued(profile)
	if e == nil && view.Authority.ContractID != id.String() {
		e = fmt.Errorf("contract filename/issued ID differs")
	}
	return file, view, path, raw, e
}
func compactAcquisitionV2(profile ProfileV2, value acquisitionAcceptedV2) any {
	item := map[string]any{"acquired": value.Acquired, "search_complete": value.SearchComplete, "next_cursor": value.NextCursor, "reasons": value.Reasons}
	if !value.CommandID.IsZero() {
		item["command_id"] = value.CommandID
	}
	if value.Contract != nil {
		view, e := value.Contract.VerifyIssued(profile)
		if e == nil {
			item["contract_id"] = view.Authority.ContractID
			item["work_item_id"] = view.Authority.WorkItemID
			item["outcome_id"] = view.Authority.OutcomeID
			item["path"] = contractPathV2(profile.Name, d.ID(view.Authority.ContractID))
			item["expires_at"] = view.Authority.ExpiresAt
			item["acceptance_mode"] = view.Authority.AcceptanceMode
		}
	}
	return item
}
func workCommandV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, token []byte, identity a.SigningIdentityView, o options) (Output, error) {
	kind := "execution"
	prefix := "work"
	if o.args[0] == "review" {
		kind = "review"
		prefix = "review"
	}
	result := Output{Operation: prefix}
	if len(o.args) < 2 {
		return result, usage("work checkout|recover|list|show <contract-id>")
	}
	result.Operation = prefix + " " + o.args[1]
	switch o.args[1] {
	case "sign", "send", "finish":
		return returnCommandV2(ctx, w, profile, client, token, identity, o)
	case "renew", "resume", "refresh", "keepalive":
		return leaseCommandV2(ctx, w, profile, client, token, identity, o)
	case "checkout":
		if kind == "review" {
			return checkoutReviewV2(ctx, w, profile, client, token, identity, o)
		}
		return checkoutWorkV2(ctx, w, profile, client, token, identity, o)
	case "recover":
		if len(o.args) != 2 {
			return result, usage("work recover --all-pending")
		}
		// Snapshot only existing intentions. No fresh search or compensation is created.
		release, e := w.LockV2(ctx, profile.Name, "")
		if e != nil {
			return result, e
		}
		current, e := w.LoadProfileV2(profile.Name)
		if e == nil {
			e = current.VerifyBinding(token)
		}
		release()
		if e != nil {
			return result, e
		}
		items := []any{}
		result.Data = map[string]any{"items": items}
		for _, intent := range current.Local.PendingOperations {
			if returnPendingV2(intent.Operation) && ((kind == "execution" && intent.Operation == "ReturnSignedWork") || (kind == "review" && intent.Operation == "ReturnSignedReview")) {
				if intent.State == "prepared_signed" {
					items = append(items, map[string]any{"intention_id": intent.ID, "state": "prepared_signed", "requires_action": "send explicitly"})
					result.Data = map[string]any{"items": items}
					continue
				}
				_, committed, e := recoverReturnV2(ctx, w, profile, client, token, intent, false)
				result.Committed = result.Committed || committed
				items = append(items, map[string]any{"intention_id": intent.ID, "remote_committed": committed, "cleanup_pending": committed && e != nil})
				result.Data = map[string]any{"items": items}
				if e != nil {
					result.RequiresAction = "preserve frozen return and reconcile its original receipt"
					return result, e
				}
				continue
			}
			isLease := leaseOperationV2(intent.Operation)
			leaseKindMatches := isLease && ((kind == "execution" && (intent.Operation == "RenewSignedWorkContract" || intent.Operation == "ResumeSignedWorkContract")) || (kind == "review" && (intent.Operation == "RenewSignedReviewContract" || intent.Operation == "ResumeSignedReviewContract")))
			if leaseKindMatches {
				accepted, e := recoverLeaseV2(ctx, w, profile, client, token, intent)
				if !accepted.CommandID.IsZero() {
					result.Committed = true
				}
				item := map[string]any{"intention_id": intent.ID, "lease_materialized": e == nil}
				if !accepted.CommandID.IsZero() {
					item["command_id"] = accepted.CommandID
				}
				items = append(items, item)
				result.Data = map[string]any{"items": items}
				if e != nil {
					result.RequiresAction = "preserve and reconcile original lease intention"
					return result, e
				}
				continue
			}
			matches := intent.Operation == "AcquireSignedWorkContract" || intent.Operation == "AcquireNextSignedWorkContract"
			if kind == "review" {
				matches = intent.Operation == "AcquireSignedReviewContract" || intent.Operation == "AcquireNextSignedReviewContract"
			}
			if !matches {
				continue
			}
			accepted, e := recoverAcquisitionV2(ctx, w, profile, client, token, intent)
			if accepted.CommandID != "" {
				result.Committed = true
			}
			items = append(items, map[string]any{"intention_id": intent.ID, "result": compactAcquisitionV2(profile, accepted)})
			result.Data = map[string]any{"items": items}
			if e != nil {
				result.RequiresAction = "work recover preserves and replays existing frozen intentions"
				return result, e
			}
		}
		return result, nil
	case "list":
		ids, e := contractIDsV2(w, profile.Name)
		if e != nil {
			return result, e
		}
		sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
		items := []any{}
		for _, id := range ids {
			legacy, isLegacy, e := readLegacyUnsignedV2(w, profile, id)
			if e != nil {
				return result, e
			}
			if isLegacy {
				if kind == "execution" {
					items = append(items, legacyUnsignedSummaryV2(legacy))
				}
				continue
			}
			_, view, _, _, e := loadContractV2(w, profile, id)
			if e != nil {
				return result, e
			}
			if view.Authority.ContractKind != kind {
				continue
			}
			items = append(items, map[string]any{"contract_id": id, "work_item_id": view.Authority.WorkItemID, "outcome_id": view.Authority.OutcomeID, "expires_at": view.Authority.ExpiresAt})
		}
		result.Data = map[string]any{"profile": profile.Name, "items": items, "authority": "obtain fresh focal state before mutation"}
		return result, nil
	case "show":
		if len(o.args) != 3 {
			return result, usage("work show <contract-id> --for-agent")
		}
		id, e := d.ParseID(o.args[2])
		if e != nil {
			return result, e
		}
		legacy, isLegacy, e := readLegacyUnsignedV2(w, profile, id)
		if e != nil {
			return result, e
		}
		if isLegacy {
			if kind != "execution" {
				return result, usage("legacy unsigned record is not a review contract")
			}
			result.ContractID = id
			result.WorkItemID = legacy.Contract.Contract.WorkItemID
			result.OutcomeID = legacy.Contract.Contract.Scope.OutcomeID
			result.Data = legacyUnsignedSummaryV2(legacy)
			result.RequiresAction = "read-only historical record; use the original v1 workspace for supported legacy recovery; never fabricate signed authority"
			return result, nil
		}
		file, view, path, _, e := loadContractV2(w, profile, id)
		if e != nil {
			return result, e
		}
		if view.Authority.ContractKind != kind {
			return result, usage("contract kind differs from selected command")
		}
		scope := d.Scope{NamespaceID: profile.Binding.NamespaceID, OutcomeID: d.ID(view.Authority.OutcomeID)}
		state, e := client.ReadSignedState(ctx, a.SignedStateQuery{Scope: scope, Resource: kind, ID: id})
		if e != nil {
			return result, e
		}
		result.ContractID = id
		result.WorkItemID = d.ID(view.Authority.WorkItemID)
		result.OutcomeID = scope.OutcomeID
		// All frozen instructions/constraints/criteria are kept; proofs and history
		// are harness-only. Showing a contract never renews or takes over execution.
		pendingLease := false
		for _, pending := range profile.Local.PendingOperations {
			if pending.ContractID != nil && *pending.ContractID == id && leaseOperationV2(pending.Operation) {
				pendingLease = true
			}
		}
		if kind == "review" {
			specification, e := reviewAgentSpecificationV2(profile, view)
			if e != nil {
				return result, e
			}
			result.Data = map[string]any{"profile": profile.Name, "path": path, "specification": specification, "spec_digest": view.Authority.SpecDigest, "status": state.Contract, "progress": file.Review.Progress, "decision": file.Review.Decision, "pending_final_return": file.Local.Pending != nil, "pending_lease_operation": pendingLease}
			return result, nil
		}
		result.Data = map[string]any{"profile": profile.Name, "path": path, "specification": view.WorkSpecification.Spec, "spec_digest": view.Authority.SpecDigest, "status": state.Contract, "progress": file.Execution.Progress, "completion_intent": file.Execution.CompletionIntent, "pending_final_return": file.Local.Pending != nil, "pending_lease_operation": pendingLease}
		return result, nil
	default:
		return result, usage("signed work operation is not implemented; preserve its contract")
	}
}
func checkoutWorkV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, token []byte, identity a.SigningIdentityView, o options) (Output, error) {
	result := Output{Operation: "work checkout"}
	scope, e := scopeV2(w, profile, o)
	if e != nil {
		return result, e
	}
	result.OutcomeID = scope.OutcomeID
	count := 1
	if value := o.values["count"]; value != "" {
		count, e = strconv.Atoi(value)
		if e != nil || count < 1 || count > 10 {
			return result, usage("--count must be 1..10 and is distinct from --limit")
		}
	}
	ttl := profile.Lease.RequestedTTLSeconds
	if value := o.values["ttl"]; value != "" {
		ttl, e = strconv.Atoi(value)
		if e != nil || ttl < 30 || ttl > 3600 {
			return result, usage("TTL must be 30..3600 seconds")
		}
	}
	operation := "AcquireSignedWorkContract"
	var command any
	if o.values["next"] == "true" {
		if len(o.args) != 2 {
			return result, usage("work checkout --next --count N")
		}
		operation = "AcquireNextSignedWorkContract"
		limit := 25
		if value := o.values["limit"]; value != "" {
			limit, e = strconv.Atoi(value)
			if e != nil || limit < 1 || limit > 100 {
				return result, usage("scan limit must be 1..100")
			}
		}
		command = a.AcquireNextSignedWorkContractCommand{Scope: scope, SignerKeyID: profile.Signing.KeyID, TTLSeconds: ttl, Limit: limit, Cursor: o.values["cursor"]}
	} else {
		if count != 1 || len(o.args) != 3 {
			return result, usage("work checkout <work-item-id> --version V, or --next --count N")
		}
		work, e := d.ParseID(o.args[2])
		if e != nil {
			return result, e
		}
		version, e := strconv.ParseUint(o.values["version"], 10, 64)
		if e != nil || version == 0 {
			return result, usage("exact --version required")
		}
		cmd := a.AcquireSignedWorkContractCommand{Scope: scope, SignerKeyID: profile.Signing.KeyID, WorkItemID: work, ExpectedWorkItemVersion: d.Version(version), TTLSeconds: ttl}
		if value := o.values["previous-review"]; value != "" {
			id, e := d.ParseID(value)
			if e != nil {
				return result, e
			}
			cmd.PreviousReviewCaseID = &id
		}
		command = cmd
	}

	// Persist the entire bounded set before sending the first request. Each item
	// has independent frozen bytes and idempotency; it is never restarted as a batch.
	intents, e := prepareAcquisitionBatchV2(ctx, w, profile, token, operation, scope, identity.Actor, command, count)
	if e != nil {
		return result, e
	}
	items := []any{}
	acquired := 0
	for _, intent := range intents {
		accepted, e := recoverAcquisitionV2(ctx, w, profile, client, token, intent)
		if accepted.CommandID != "" {
			result.Committed = true
		}
		if accepted.Acquired {
			acquired++
		}
		items = append(items, map[string]any{"intention_id": intent.ID, "result": compactAcquisitionV2(profile, accepted)})
		result.Data = map[string]any{"requested": count, "acquired": acquired, "items": items, "partial": acquired < count}
		if e != nil {
			result.RequiresAction = "work recover --all-pending replays this batch's frozen intentions; accepted contracts are preserved"
			return result, e
		}
	}
	return result, nil
}
