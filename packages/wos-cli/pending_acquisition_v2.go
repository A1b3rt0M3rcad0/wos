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
	"io"
	"os"
	"path/filepath"
	"strings"
)

type acquisitionAcceptedV2 struct {
	Acquired       bool            `json:"acquired"`
	Contract       *ContractFileV2 `json:"contract,omitempty"`
	CommandID      d.ID            `json:"command_id"`
	SearchComplete bool            `json:"search_complete"`
	NextCursor     string          `json:"next_cursor,omitempty"`
	Reasons        []string        `json:"reasons"`
}

func mutateProfileV2(ctx context.Context, w *Workspace, expected ProfileV2, token []byte, fn func(*ProfileV2) error) error {
	release, e := w.LockV2(ctx, expected.Name, "")
	if e != nil {
		return e
	}
	defer release()
	current, raw, e := w.readProfileSnapshotV2(expected.Name)
	if e != nil {
		return e
	}
	if e = current.VerifyBinding(token); e != nil {
		return e
	}
	if current.Local.BindingMAC != expected.Local.BindingMAC {
		return fmt.Errorf("profile binding changed; pending intention preserved")
	}
	if e = fn(&current); e != nil {
		return e
	}
	if e = current.SealBinding(token); e != nil {
		return e
	}
	return w.WriteV2(profilePathV2(current.Name), current, signing.Digest(raw))
}
func pendingIndexV2(profile *ProfileV2, intent PendingOperationV2) (int, error) {
	for i, current := range profile.Local.PendingOperations {
		if current.ID != intent.ID {
			continue
		}
		if intent.ContractID != nil && (current.ContractID == nil || *current.ContractID != *intent.ContractID) {
			return -1, fmt.Errorf("pending contract target changed")
		}
		if current.Operation != intent.Operation || current.Scope != intent.Scope || current.IdempotencyKey != intent.IdempotencyKey || current.Payload != intent.Payload || current.PayloadDigest != intent.PayloadDigest || current.RequestFingerprint != intent.RequestFingerprint {
			return -1, fmt.Errorf("pending intention changed")
		}
		return i, nil
	}
	return -1, fmt.Errorf("pending intention no longer present; do not create a replacement acquisition")
}
func prepareAcquisitionBatchV2(ctx context.Context, w *Workspace, profile ProfileV2, token []byte, operation string, scope d.Scope, actor d.ActorRef, command any, count int) ([]PendingOperationV2, error) {
	if count < 1 || count > 10 {
		return nil, usage("acquisition count must be 1..10")
	}
	if e := scope.Validate(); e != nil {
		return nil, e
	}
	raw, e := signing.Canonical(command)
	if e != nil {
		return nil, e
	}
	fingerprint, e := a.SignedCommandFingerprint(actor, command)
	if e != nil {
		return nil, e
	}
	intents := make([]PendingOperationV2, 0, count)
	for i := 0; i < count; i++ {
		id, e := newLocalIDV2()
		if e != nil {
			return nil, e
		}
		key, e := sdk.NewIdempotencyKey()
		if e != nil {
			return nil, e
		}
		intents = append(intents, PendingOperationV2{ID: id, Operation: operation, State: "prepared", Scope: ProfilePendingScopeV2{NamespaceID: scope.NamespaceID, OutcomeID: scope.OutcomeID}, IdempotencyKey: key, Payload: base64.StdEncoding.EncodeToString(raw), PayloadDigest: signing.Digest(raw), RequestFingerprint: fingerprint})
	}
	e = mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
		if len(current.Local.PendingOperations)+count > 10 {
			return fmt.Errorf("pending limit reached; recover existing intentions")
		}
		files, e := contractIDsV2(w, current.Name)
		if e != nil {
			return e
		}
		if len(files) >= 100 {
			return fmt.Errorf("local contract limit reached")
		}
		current.Local.PendingOperations = append(current.Local.PendingOperations, intents...)
		return nil
	})
	return intents, e
}
func contractPathV2(profile string, id d.ID) string {
	return filepath.Join(".wos/profiles", profile, "contract", id.String()+".yaml")
}
func contractIDsV2(w *Workspace, profile string) ([]d.ID, error) {
	path := filepath.Join(".wos/profiles", profile, "contract")
	if e := w.check(path); e != nil {
		return nil, e
	}
	dir, e := w.root.Open(path)
	if os.IsNotExist(e) {
		return []d.ID{}, nil
	}
	if e != nil {
		return nil, e
	}
	defer dir.Close()
	entries, e := dir.ReadDir(112)
	if e != nil && e != io.EOF {
		return nil, e
	}
	if len(entries) >= 112 {
		return nil, fmt.Errorf("contract directory scan bound reached; preserve files for reconciliation")
	}
	ids := []d.ID{}
	seen := map[d.ID]bool{}
	for _, entry := range entries {
		if isContractStageV2(entry.Name()) {
			stage := filepath.Join(path, entry.Name())
			if _, _, e = w.readStageV2(stage, 1); e != nil {
				if os.IsNotExist(e) {
					continue
				}
				// A published two-alias pair is normalized by the ordinary
				// confined reader before another batch item scans this folder.
				base := strings.TrimSuffix(strings.Split(entry.Name(), ".materialize-")[0], ".yaml")
				target, resolveError := w.DocumentPath(filepath.Join(path, base))
				if resolveError != nil {
					return nil, resolveError
				}
				if _, e = w.ReadV2(target); e != nil {
					return nil, e
				}
				if _, _, e = w.readStageV2(stage, 1); e != nil && !os.IsNotExist(e) {
					return nil, e
				}
			}
			continue // technical stage is not execution material or agent context
		}

		extension := filepath.Ext(entry.Name())
		id := d.ID(entry.Name()[:len(entry.Name())-len(extension)])
		if (extension != ".yaml" && extension != ".yml") || entry.IsDir() || entry.Type()&os.ModeSymlink != 0 || id.Validate() != nil || seen[id] {
			return nil, fmt.Errorf("unrecognized or ambiguous contract entry preserved")
		}
		seen[id] = true
		if _, e = w.ReadV2(filepath.Join(path, entry.Name())); e != nil {
			return nil, e
		}
		ids = append(ids, id)
	}
	if len(ids) > 100 {
		return nil, fmt.Errorf("local contract limit exceeded")
	}
	return ids, nil
}
func executionAcquisitionV2(profile ProfileV2, scope d.Scope, commandID d.ID, result a.WorkContractResult) (acquisitionAcceptedV2, error) {
	zero := acquisitionAcceptedV2{}
	c := result.Contract
	if c.Scope != scope || c.HolderPrincipalID != profile.Binding.PrincipalID || c.SignedBinding == nil || c.SignedBinding.CredentialID != profile.Binding.CredentialID || result.IssuedSpecification == nil || result.IssuedAuthority == nil {
		return zero, fmt.Errorf("acquisition response binding differs; intention preserved")
	}
	file := ContractFileV2{SchemaVersion: 2, Kind: "WOSContractFile", Issued: IssuedDocumentsV2{Specification: *result.IssuedSpecification, Authority: *result.IssuedAuthority}, Local: ContractLocalV2{SchemaVersion: 1, Profile: profile.Name, WorkItemVersion: signing.Decimal(result.WorkItem.Version)}, Execution: &ExecutionDraftV2{CompletionIntent: "auto", Material: a.SignedReturnMaterial{Result: d.NewSignedResultMaterial(d.WorkResultMaterial{ContractID: c.ID, WorkItemID: c.WorkItemID, SpecDigest: c.SignedBinding.SpecificationDigest})}}}
	view, e := file.VerifyIssued(profile)
	if e != nil {
		return zero, e
	}
	if view.Authority.ContractID != c.ID.String() || view.Authority.OutcomeID != scope.OutcomeID.String() {
		return zero, fmt.Errorf("issued acquisition target differs")
	}
	return acquisitionAcceptedV2{Acquired: true, Contract: &file, CommandID: commandID, SearchComplete: true, Reasons: []string{}}, nil
}
func reviewAcquisitionV2(profile ProfileV2, scope d.Scope, commandID d.ID, result a.SignedReviewContractResult) (acquisitionAcceptedV2, error) {
	c := result.Contract
	if c.Scope != scope || c.HolderPrincipalID != profile.Binding.PrincipalID || c.Binding.CredentialID != profile.Binding.CredentialID || result.IssuedSpecification == nil || result.Case.ID != c.CaseID {
		return acquisitionAcceptedV2{}, fmt.Errorf("review acquisition response binding differs")
	}
	file := ContractFileV2{SchemaVersion: 2, Kind: "WOSContractFile", Issued: IssuedDocumentsV2{Specification: *result.IssuedSpecification, Authority: result.IssuedAuthority}, Local: ContractLocalV2{SchemaVersion: 1, Profile: profile.Name, WorkItemVersion: result.WorkItemVersion, ReviewCaseVersion: signing.Decimal(result.Case.Version)}, Review: &ReviewDraftV2{Decision: "inconclusive", Material: a.SignedReviewMaterial{}}}
	view, e := file.VerifyIssued(profile)
	if e != nil {
		return acquisitionAcceptedV2{}, e
	}
	if view.Authority.ContractID != c.ID.String() || view.Authority.ContractKind != "review" || view.ReviewSpecification.Spec.ReviewCaseID != c.CaseID || view.ReviewSpecification.Spec.SubmissionID != c.SubmissionID {
		return acquisitionAcceptedV2{}, fmt.Errorf("issued review target differs")
	}
	return acquisitionAcceptedV2{Acquired: true, Contract: &file, CommandID: commandID, SearchComplete: true, Reasons: []string{}}, nil
}
func reviewSearchAcquisitionV2(profile ProfileV2, scope d.Scope, id d.ID, result a.SignedReviewAcquisition) (acquisitionAcceptedV2, error) {
	if !result.Acquired {
		return acquisitionAcceptedV2{CommandID: id, SearchComplete: result.SearchComplete, NextCursor: result.NextCursor, Reasons: result.Reasons}, nil
	}
	if result.Result == nil {
		return acquisitionAcceptedV2{}, fmt.Errorf("acquired review result absent")
	}
	value, e := reviewAcquisitionV2(profile, scope, id, *result.Result)
	value.SearchComplete = result.SearchComplete
	value.NextCursor = result.NextCursor
	value.Reasons = result.Reasons
	return value, e
}
func callAcquisitionV2(ctx context.Context, client *sdk.Client, profile ProfileV2, pending PendingOperationV2) (acquisitionAcceptedV2, error) {
	var zero acquisitionAcceptedV2
	raw, e := base64.StdEncoding.Strict().DecodeString(pending.Payload)
	if e != nil || signing.Digest(raw) != pending.PayloadDigest {
		return zero, fmt.Errorf("pending payload differs")
	}
	scope := d.Scope{NamespaceID: pending.Scope.NamespaceID, OutcomeID: pending.Scope.OutcomeID}
	switch pending.Operation {
	case "AcquireSignedWorkContract":
		var cmd a.AcquireSignedWorkContractCommand
		if e = signing.DecodeStrict(raw, &cmd, signing.MaxPayloadBytes); e != nil {
			return zero, e
		}
		if cmd.Scope != scope || cmd.SignerKeyID != profile.Signing.KeyID {
			return zero, fmt.Errorf("pending command binding differs")
		}
		result, e := client.AcquireSignedWorkContract(ctx, pending.IdempotencyKey, cmd)
		if e != nil {
			return zero, e
		}
		if result.ResultOmitted {
			return zero, fmt.Errorf("acquisition result omitted; preserve intention")
		}
		return executionAcquisitionV2(profile, scope, result.CommandID, result.Value)
	case "AcquireNextSignedWorkContract":
		var cmd a.AcquireNextSignedWorkContractCommand
		if e = signing.DecodeStrict(raw, &cmd, signing.MaxPayloadBytes); e != nil {
			return zero, e
		}
		if cmd.Scope != scope || cmd.SignerKeyID != profile.Signing.KeyID {
			return zero, fmt.Errorf("pending search binding differs")
		}
		result, e := client.AcquireNextSignedWorkContract(ctx, pending.IdempotencyKey, cmd)
		if e != nil {
			return zero, e
		}
		if result.ResultOmitted {
			return zero, fmt.Errorf("search result omitted; preserve intention")
		}
		if !result.Value.Acquired {
			return acquisitionAcceptedV2{CommandID: result.CommandID, SearchComplete: result.Value.SearchComplete, NextCursor: result.Value.NextCursor, Reasons: result.Value.Reasons}, nil
		}
		if result.Value.Result == nil {
			return zero, fmt.Errorf("acquired search lacks contract")
		}
		accepted, e := executionAcquisitionV2(profile, scope, result.CommandID, *result.Value.Result)
		accepted.SearchComplete = result.Value.SearchComplete
		accepted.NextCursor = result.Value.NextCursor
		accepted.Reasons = result.Value.Reasons
		return accepted, e
	case "AcquireSignedReviewContract":
		var cmd a.AcquireSignedReviewContractCommand
		if e = signing.DecodeStrict(raw, &cmd, signing.MaxPayloadBytes); e != nil {
			return zero, e
		}
		if cmd.Scope != scope || cmd.SignerKeyID != profile.Signing.KeyID {
			return zero, fmt.Errorf("pending review binding differs")
		}
		result, e := client.AcquireSignedReviewContract(ctx, pending.IdempotencyKey, cmd)
		if e != nil {
			return zero, e
		}
		if result.ResultOmitted {
			return zero, fmt.Errorf("review result omitted; preserve intention")
		}
		return reviewAcquisitionV2(profile, scope, result.CommandID, result.Value)
	case "AcquireNextSignedReviewContract":
		var cmd a.AcquireNextSignedReviewContractCommand
		if e = signing.DecodeStrict(raw, &cmd, signing.MaxPayloadBytes); e != nil {
			return zero, e
		}
		if cmd.Scope != scope || cmd.SignerKeyID != profile.Signing.KeyID {
			return zero, fmt.Errorf("pending review search binding differs")
		}
		result, e := client.AcquireNextSignedReviewContract(ctx, pending.IdempotencyKey, cmd)
		if e != nil {
			return zero, e
		}
		if result.ResultOmitted {
			return zero, fmt.Errorf("review search result omitted; preserve intention")
		}
		return reviewSearchAcquisitionV2(profile, scope, result.CommandID, result.Value)
	default:
		return zero, fmt.Errorf("unsupported frozen operation preserved")
	}
}
func recoverAcquisitionV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, token []byte, intent PendingOperationV2) (acquisitionAcceptedV2, error) {
	var accepted acquisitionAcceptedV2
	snapshot := intent
	lookupAccepted := false
	e := mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
		index, e := pendingIndexV2(current, intent)
		if e != nil {
			return e
		}
		snapshot = current.Local.PendingOperations[index]
		lookupAccepted = snapshot.State == "sent_unknown"
		if snapshot.State == "accepted_unmaterialized" {
			return nil
		}
		if snapshot.State != "prepared" && snapshot.State != "sent_unknown" {
			return fmt.Errorf("pending state requires explicit reconciliation")
		}
		current.Local.PendingOperations[index].State = "sent_unknown"
		snapshot.State = "sent_unknown"
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
		if e = json.Unmarshal(raw, &accepted); e != nil {
			return accepted, e
		}
	} else {
		if lookupAccepted {
			accepted, e = reconcileAcquisitionV2(ctx, client, profile, snapshot)
			if e != nil {
				var remote *sdk.Error
				if !errors.As(e, &remote) || remote.Code != "not_found" {
					return accepted, e
				}
				accepted, e = callAcquisitionV2(ctx, client, profile, snapshot)
			}
		} else {
			accepted, e = callAcquisitionV2(ctx, client, profile, snapshot)
		}
		if e != nil {
			return accepted, e
		}
		raw, e := json.Marshal(accepted)
		if e != nil {
			return accepted, e
		}
		if len(raw) > 256<<10 {
			return accepted, fmt.Errorf("accepted compact response exceeds bound; original server intention remains recoverable")
		}
		e = mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
			index, e := pendingIndexV2(current, intent)
			if e != nil {
				return e
			}
			next := &current.Local.PendingOperations[index]
			if next.State == "accepted_unmaterialized" && next.Response != base64.StdEncoding.EncodeToString(raw) {
				return fmt.Errorf("concurrent accepted response differs")
			}
			next.State = "accepted_unmaterialized"
			next.Response = base64.StdEncoding.EncodeToString(raw)
			if accepted.Contract != nil {
				view, e := accepted.Contract.VerifyIssued(profile)
				if e != nil {
					return e
				}
				id := d.ID(view.Authority.ContractID)
				next.ContractID = &id
			}
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
		if accepted.Acquired {
			if accepted.Contract == nil {
				return fmt.Errorf("accepted contract absent")
			}
			view, e := accepted.Contract.VerifyIssued(*current)
			if e != nil {
				return e
			}
			if view.Authority.OutcomeID != intent.Scope.OutcomeID.String() {
				return fmt.Errorf("accepted Outcome differs")
			}
			id := d.ID(view.Authority.ContractID)
			release, e := w.LockV2(ctx, current.Name, id.String())
			if e != nil {
				return e
			}
			defer release()
			if e = w.recoverContractStageV2(*current, id, intent.ID, *accepted.Contract); e != nil {
				return e
			}
			path, e := w.DocumentPath(filepath.Join(".wos/profiles", current.Name, "contract", id.String()))
			if e != nil {
				return e
			}
			raw, e := w.ReadV2(path)
			if e == nil {
				var existing ContractFileV2
				if e = DecodeV2Document(raw, &existing); e != nil {
					return e
				}
				other, e := existing.VerifyIssued(*current)
				if e != nil {
					return e
				}
				originalSpec, _ := json.Marshal(accepted.Contract.Issued.Specification)
				existingSpec, _ := json.Marshal(existing.Issued.Specification)
				if other.Authority.ContractID != id.String() || string(originalSpec) != string(existingSpec) {
					return fmt.Errorf("existing contract differs; edited file preserved")
				}
				// Never replace an already materialized draft or a newer authority.
			} else if os.IsNotExist(e) {
				files, e := contractIDsV2(w, current.Name)
				if e != nil {
					return e
				}
				if len(files) >= 100 {
					return fmt.Errorf("local contract limit reached; accepted intention preserved")
				}
				if e = w.createAcceptedContractV2(*current, id, intent.ID, *accepted.Contract); e != nil {
					return e
				}
			} else {
				return e
			}
		}
		current.Local.PendingOperations = append(current.Local.PendingOperations[:index], current.Local.PendingOperations[index+1:]...)
		return nil
	})
	return accepted, e
}

func reconcileAcquisitionV2(ctx context.Context, client *sdk.Client, profile ProfileV2, intent PendingOperationV2) (acquisitionAcceptedV2, error) {
	var zero acquisitionAcceptedV2
	scope := d.Scope{NamespaceID: intent.Scope.NamespaceID, OutcomeID: intent.Scope.OutcomeID}
	result, e := client.ReadSignedState(ctx, a.SignedStateQuery{Scope: scope, Resource: "operation", IdempotencyKey: intent.IdempotencyKey})
	if e != nil {
		return zero, e
	}
	if result.Scope == nil || *result.Scope != scope || result.OperationName != intent.Operation || result.OperationCommandID == nil || intent.RequestFingerprint == "" || result.OperationFingerprint != intent.RequestFingerprint {
		return zero, fmt.Errorf("durable result differs from frozen intention; preserve it for explicit reconciliation")
	}
	raw, e := base64.StdEncoding.Strict().DecodeString(result.OperationPayload)
	if e != nil || len(raw) > d.MaxSignedOperationResultBytes || signing.Digest(raw) != result.PayloadDigest {
		return zero, fmt.Errorf("durable response bytes/digest differ")
	}
	switch intent.Operation {
	case "AcquireSignedWorkContract":
		var accepted sdk.CommandResult[a.WorkContractResult]
		if e = json.Unmarshal(raw, &accepted); e != nil {
			return zero, e
		}
		if accepted.CommandID != *result.OperationCommandID {
			return zero, fmt.Errorf("durable command identity differs")
		}
		return executionAcquisitionV2(profile, scope, accepted.CommandID, accepted.Value)
	case "AcquireNextSignedWorkContract":
		var accepted sdk.CommandResult[a.WorkContractAcquisition]
		if e = json.Unmarshal(raw, &accepted); e != nil {
			return zero, e
		}
		if accepted.CommandID != *result.OperationCommandID {
			return zero, fmt.Errorf("durable search identity differs")
		}
		if !accepted.Value.Acquired {
			return acquisitionAcceptedV2{CommandID: accepted.CommandID, SearchComplete: accepted.Value.SearchComplete, NextCursor: accepted.Value.NextCursor, Reasons: accepted.Value.Reasons}, nil
		}
		if accepted.Value.Result == nil {
			return zero, fmt.Errorf("durable acquired contract absent")
		}
		value, e := executionAcquisitionV2(profile, scope, accepted.CommandID, *accepted.Value.Result)
		value.SearchComplete = accepted.Value.SearchComplete
		value.NextCursor = accepted.Value.NextCursor
		value.Reasons = accepted.Value.Reasons
		return value, e
	case "AcquireSignedReviewContract":
		var accepted sdk.CommandResult[a.SignedReviewContractResult]
		if e = json.Unmarshal(raw, &accepted); e != nil {
			return zero, e
		}
		if accepted.CommandID != *result.OperationCommandID {
			return zero, fmt.Errorf("durable review identity differs")
		}
		return reviewAcquisitionV2(profile, scope, accepted.CommandID, accepted.Value)
	case "AcquireNextSignedReviewContract":
		var accepted sdk.CommandResult[a.SignedReviewAcquisition]
		if e = json.Unmarshal(raw, &accepted); e != nil {
			return zero, e
		}
		if accepted.CommandID != *result.OperationCommandID {
			return zero, fmt.Errorf("durable review search identity differs")
		}
		return reviewSearchAcquisitionV2(profile, scope, accepted.CommandID, accepted.Value)
	default:
		return zero, fmt.Errorf("unsupported frozen recovery operation")
	}
}
