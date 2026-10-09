package woscli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

type leaseAcceptedV2 struct {
	CommandID         d.ID             `json:"command_id"`
	ContractID        d.ID             `json:"contract_id"`
	Authority         signing.Document `json:"authority"`
	WorkItemVersion   signing.Decimal  `json:"work_item_version"`
	ReviewCaseVersion signing.Decimal  `json:"review_case_version,omitempty"`
}

func leaseOperationV2(operation string) bool {
	switch operation {
	case "RenewSignedWorkContract", "ResumeSignedWorkContract", "RenewSignedReviewContract", "ResumeSignedReviewContract":
		return true
	}
	return false
}
func prepareLeaseV2(ctx context.Context, w *Workspace, profile ProfileV2, token []byte, actor d.ActorRef, id d.ID, kind string, resume bool, ttl int) (PendingOperationV2, error) {
	var intent PendingOperationV2
	e := mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
		if len(current.Local.PendingOperations) >= 10 {
			return fmt.Errorf("pending limit reached; recover original intentions")
		}
		for _, p := range current.Local.PendingOperations {
			if p.ContractID != nil && *p.ContractID == id {
				return fmt.Errorf("contract has unresolved intention; recover it before lease mutation")
			}
		}
		release, e := w.LockV2(ctx, current.Name, id.String())
		if e != nil {
			return e
		}
		defer release()
		file, view, _, _, e := loadContractV2(w, *current, id)
		if e != nil {
			return e
		}
		if view.Authority.ContractKind != kind || file.Local.Pending != nil || file.Local.AcceptanceReceipt != nil {
			return fmt.Errorf("lease mutation paused for kind mismatch or pending/accepted final return")
		}
		scope := d.Scope{NamespaceID: current.Binding.NamespaceID, OutcomeID: d.ID(view.Authority.OutcomeID)}
		authority := a.ContractAuthority{ExecutionID: d.ID(view.Authority.ExecutionID), FencingToken: d.FencingToken(view.Authority.FencingToken), SpecDigest: view.Authority.SpecDigest}
		digest := signing.Digest(file.Issued.Authority.Payload)
		var command any
		operation := "RenewSignedWorkContract"
		if kind == "execution" {
			if resume {
				operation = "ResumeSignedWorkContract"
				command = a.ResumeSignedWorkContractCommand{Scope: scope, ContractID: id, SignerKeyID: current.Signing.KeyID, Authority: authority, AuthorityDigest: digest, ExpectedLeaseVersion: d.Version(view.Authority.LeaseVersion)}
			} else {
				command = a.RenewSignedWorkContractCommand{Scope: scope, ContractID: id, SignerKeyID: current.Signing.KeyID, Authority: authority, AuthorityDigest: digest, ExpectedLeaseVersion: d.Version(view.Authority.LeaseVersion), TTLSeconds: ttl}
			}
		} else {
			operation = "RenewSignedReviewContract"
			if resume {
				operation = "ResumeSignedReviewContract"
				command = a.ResumeSignedReviewContractCommand{Scope: scope, ContractID: id, SignerKeyID: current.Signing.KeyID, Authority: authority, AuthorityDigest: digest, ExpectedLeaseVersion: d.Version(view.Authority.LeaseVersion)}
			} else {
				command = a.RenewSignedReviewContractCommand{Scope: scope, ContractID: id, SignerKeyID: current.Signing.KeyID, Authority: authority, AuthorityDigest: digest, ExpectedLeaseVersion: d.Version(view.Authority.LeaseVersion), TTLSeconds: ttl}
			}
		}
		raw, e := signing.Canonical(command)
		if e != nil {
			return e
		}
		fp, e := a.SignedCommandFingerprint(actor, command)
		if e != nil {
			return e
		}
		localID, e := newLocalIDV2()
		if e != nil {
			return e
		}
		key, e := sdk.NewIdempotencyKey()
		if e != nil {
			return e
		}
		intent = PendingOperationV2{ID: localID, Operation: operation, State: "prepared", Scope: ProfilePendingScopeV2{NamespaceID: scope.NamespaceID, OutcomeID: scope.OutcomeID}, ContractID: &id, IdempotencyKey: key, Payload: base64.StdEncoding.EncodeToString(raw), PayloadDigest: signing.Digest(raw), RequestFingerprint: fp}
		current.Local.PendingOperations = append(current.Local.PendingOperations, intent)
		return nil
	})
	return intent, e
}
func leaseWorkResultV2(profile ProfileV2, intent PendingOperationV2, id d.ID, result a.WorkContractResult) (leaseAcceptedV2, error) {
	c := result.Contract
	if intent.ContractID == nil || c.ID != *intent.ContractID || c.Scope.NamespaceID != intent.Scope.NamespaceID || c.Scope.OutcomeID != intent.Scope.OutcomeID || c.HolderPrincipalID != profile.Binding.PrincipalID || c.SignedBinding == nil || c.SignedBinding.CredentialID != profile.Binding.CredentialID || result.IssuedAuthority == nil {
		return leaseAcceptedV2{}, fmt.Errorf("accepted lease binding differs")
	}
	return leaseAcceptedV2{CommandID: id, ContractID: c.ID, Authority: *result.IssuedAuthority, WorkItemVersion: signing.Decimal(result.WorkItem.Version)}, nil
}
func leaseReviewResultV2(profile ProfileV2, intent PendingOperationV2, id d.ID, result a.SignedReviewContractResult) (leaseAcceptedV2, error) {
	c := result.Contract
	if intent.ContractID == nil || c.ID != *intent.ContractID || c.Scope.NamespaceID != intent.Scope.NamespaceID || c.Scope.OutcomeID != intent.Scope.OutcomeID || c.HolderPrincipalID != profile.Binding.PrincipalID || c.Binding.CredentialID != profile.Binding.CredentialID {
		return leaseAcceptedV2{}, fmt.Errorf("accepted review lease binding differs")
	}
	return leaseAcceptedV2{CommandID: id, ContractID: c.ID, Authority: result.IssuedAuthority, WorkItemVersion: result.WorkItemVersion, ReviewCaseVersion: signing.Decimal(result.Case.Version)}, nil
}
func callLeaseV2(ctx context.Context, client *sdk.Client, profile ProfileV2, intent PendingOperationV2) (leaseAcceptedV2, error) {
	var zero leaseAcceptedV2
	raw, e := base64.StdEncoding.Strict().DecodeString(intent.Payload)
	if e != nil || signing.Digest(raw) != intent.PayloadDigest {
		return zero, fmt.Errorf("frozen lease bytes differ")
	}
	validate := func(scope d.Scope, id, key d.ID) error {
		if intent.ContractID == nil || *intent.ContractID != id || scope.NamespaceID != intent.Scope.NamespaceID || scope.OutcomeID != intent.Scope.OutcomeID || key != profile.Signing.KeyID {
			return fmt.Errorf("frozen lease target/key differs")
		}
		return nil
	}
	switch intent.Operation {
	case "RenewSignedWorkContract":
		var cmd a.RenewSignedWorkContractCommand
		if e = signing.DecodeStrict(raw, &cmd, signing.MaxPayloadBytes); e != nil {
			return zero, e
		}
		if e = validate(cmd.Scope, cmd.ContractID, cmd.SignerKeyID); e != nil {
			return zero, e
		}
		r, e := client.RenewSignedWorkContract(ctx, intent.IdempotencyKey, cmd)
		if e != nil {
			return zero, e
		}
		if r.ResultOmitted {
			return zero, fmt.Errorf("lease response omitted")
		}
		return leaseWorkResultV2(profile, intent, r.CommandID, r.Value)
	case "ResumeSignedWorkContract":
		var cmd a.ResumeSignedWorkContractCommand
		if e = signing.DecodeStrict(raw, &cmd, signing.MaxPayloadBytes); e != nil {
			return zero, e
		}
		if e = validate(cmd.Scope, cmd.ContractID, cmd.SignerKeyID); e != nil {
			return zero, e
		}
		r, e := client.ResumeSignedWorkContract(ctx, intent.IdempotencyKey, cmd)
		if e != nil {
			return zero, e
		}
		if r.ResultOmitted {
			return zero, fmt.Errorf("takeover response omitted")
		}
		return leaseWorkResultV2(profile, intent, r.CommandID, r.Value)
	case "RenewSignedReviewContract":
		var cmd a.RenewSignedReviewContractCommand
		if e = signing.DecodeStrict(raw, &cmd, signing.MaxPayloadBytes); e != nil {
			return zero, e
		}
		if e = validate(cmd.Scope, cmd.ContractID, cmd.SignerKeyID); e != nil {
			return zero, e
		}
		r, e := client.RenewSignedReviewContract(ctx, intent.IdempotencyKey, cmd)
		if e != nil {
			return zero, e
		}
		if r.ResultOmitted {
			return zero, fmt.Errorf("review lease response omitted")
		}
		return leaseReviewResultV2(profile, intent, r.CommandID, r.Value)
	case "ResumeSignedReviewContract":
		var cmd a.ResumeSignedReviewContractCommand
		if e = signing.DecodeStrict(raw, &cmd, signing.MaxPayloadBytes); e != nil {
			return zero, e
		}
		if e = validate(cmd.Scope, cmd.ContractID, cmd.SignerKeyID); e != nil {
			return zero, e
		}
		r, e := client.ResumeSignedReviewContract(ctx, intent.IdempotencyKey, cmd)
		if e != nil {
			return zero, e
		}
		if r.ResultOmitted {
			return zero, fmt.Errorf("review takeover response omitted")
		}
		return leaseReviewResultV2(profile, intent, r.CommandID, r.Value)
	}
	return zero, fmt.Errorf("unsupported lease intention")
}
func reconcileLeaseV2(ctx context.Context, client *sdk.Client, profile ProfileV2, intent PendingOperationV2) (leaseAcceptedV2, error) {
	var zero leaseAcceptedV2
	state, e := client.ReadSignedState(ctx, a.SignedStateQuery{Scope: d.Scope{NamespaceID: intent.Scope.NamespaceID, OutcomeID: intent.Scope.OutcomeID}, Resource: "operation", IdempotencyKey: intent.IdempotencyKey})
	if e != nil {
		return zero, e
	}
	if state.OperationName != intent.Operation || state.OperationCommandID == nil || intent.RequestFingerprint == "" || state.OperationFingerprint != intent.RequestFingerprint || state.Scope == nil || state.Scope.NamespaceID != intent.Scope.NamespaceID || state.Scope.OutcomeID != intent.Scope.OutcomeID {
		return zero, fmt.Errorf("durable lease result differs from original intention")
	}
	raw, e := base64.StdEncoding.Strict().DecodeString(state.OperationPayload)
	if e != nil || len(raw) > d.MaxSignedOperationResultBytes || signing.Digest(raw) != state.PayloadDigest {
		return zero, fmt.Errorf("durable lease bytes differ")
	}
	switch intent.Operation {
	case "RenewSignedWorkContract", "ResumeSignedWorkContract":
		var r sdk.CommandResult[a.WorkContractResult]
		if e = json.Unmarshal(raw, &r); e != nil {
			return zero, e
		}
		if r.CommandID != *state.OperationCommandID {
			return zero, fmt.Errorf("original lease command ID differs")
		}
		return leaseWorkResultV2(profile, intent, r.CommandID, r.Value)
	case "RenewSignedReviewContract", "ResumeSignedReviewContract":
		var r sdk.CommandResult[a.SignedReviewContractResult]
		if e = json.Unmarshal(raw, &r); e != nil {
			return zero, e
		}
		if r.CommandID != *state.OperationCommandID {
			return zero, fmt.Errorf("original review lease command ID differs")
		}
		return leaseReviewResultV2(profile, intent, r.CommandID, r.Value)
	}
	return zero, fmt.Errorf("unsupported lease operation")
}
func recoverLeaseV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, token []byte, intent PendingOperationV2) (leaseAcceptedV2, error) {
	var accepted leaseAcceptedV2
	snapshot := intent
	lookup := false
	e := mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
		index, e := pendingIndexV2(current, intent)
		if e != nil {
			return e
		}
		snapshot = current.Local.PendingOperations[index]
		switch snapshot.State {
		case "prepared", "sent_unknown":
			lookup = snapshot.State == "sent_unknown"
			current.Local.PendingOperations[index].State = "sent_unknown"
		case "accepted_unmaterialized":
		default:
			return fmt.Errorf("lease state requires explicit reconciliation")
		}
		return nil
	})
	if e != nil {
		return accepted, e
	}
	if snapshot.State == "accepted_unmaterialized" {
		raw, e := base64.StdEncoding.Strict().DecodeString(snapshot.Response)
		if e != nil {
			return accepted, e
		}
		if e = signing.DecodeStrict(raw, &accepted, 256<<10); e != nil {
			return accepted, e
		}
	} else {
		if lookup {
			accepted, e = reconcileLeaseV2(ctx, client, profile, intent)
			if e != nil {
				code, _ := d.ErrorCodeOf(e)
				remote, ok := e.(*sdk.Error)
				if code != d.ErrorCodeNotFound && (!ok || remote.Code != string(d.ErrorCodeNotFound)) {
					return accepted, e
				}
				accepted, e = callLeaseV2(ctx, client, profile, intent)
			}
		} else {
			accepted, e = callLeaseV2(ctx, client, profile, intent)
		}
		if e != nil {
			return accepted, e
		}
		raw, e := json.Marshal(accepted)
		if e != nil || len(raw) > 256<<10 {
			return accepted, fmt.Errorf("accepted lease exceeds local technical bound")
		}
		e = mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
			index, e := pendingIndexV2(current, intent)
			if e != nil {
				return e
			}
			current.Local.PendingOperations[index].State = "accepted_unmaterialized"
			current.Local.PendingOperations[index].Response = base64.StdEncoding.EncodeToString(raw)
			return nil
		})
		if e != nil {
			return accepted, e
		}
	}
	e = mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
		index, e := pendingIndexV2(current, intent)
		if e != nil {
			return e
		}
		if intent.ContractID == nil || accepted.ContractID != *intent.ContractID {
			return fmt.Errorf("accepted lease local target differs")
		}
		release, e := w.LockV2(ctx, current.Name, accepted.ContractID.String())
		if e != nil {
			return e
		}
		defer release()
		file, view, path, observed, e := loadContractV2(w, *current, accepted.ContractID)
		if e != nil {
			return e
		}
		if file.Local.Pending != nil || file.Local.AcceptanceReceipt != nil {
			return fmt.Errorf("pending return pauses lease materialization")
		}
		raw, e := base64.StdEncoding.Strict().DecodeString(intent.Payload)
		if e != nil {
			return e
		}
		var frozen struct{ AuthorityDigest string }
		if e = json.Unmarshal(raw, &frozen); e != nil {
			return e
		}
		oldDigest := signing.Digest(file.Issued.Authority.Payload)
		newDigest := signing.Digest(accepted.Authority.Payload)
		if oldDigest != frozen.AuthorityDigest && oldDigest != newDigest {
			return fmt.Errorf("local authority changed; preserve draft and accepted intention")
		}
		candidate := file
		candidate.Issued.Authority = accepted.Authority
		updated, e := candidate.VerifyIssued(*current)
		if e != nil {
			return e
		}
		if updated.Authority.ContractID != view.Authority.ContractID || updated.Authority.ContractKind != view.Authority.ContractKind || updated.Authority.OutcomeID != intent.Scope.OutcomeID.String() || (updated.Authority.LeaseVersion <= view.Authority.LeaseVersion && oldDigest != newDigest) {
			return fmt.Errorf("accepted authority target/version differs")
		}
		candidate.Local.WorkItemVersion = accepted.WorkItemVersion
		candidate.Local.ReviewCaseVersion = accepted.ReviewCaseVersion
		if e = w.WriteV2(path, candidate, signing.Digest(observed)); e != nil {
			return e
		}
		current.Local.PendingOperations = append(current.Local.PendingOperations[:index], current.Local.PendingOperations[index+1:]...)
		return nil
	})
	return accepted, e
}
