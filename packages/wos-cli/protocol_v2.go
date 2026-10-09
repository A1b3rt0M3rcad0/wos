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
	"strconv"
)

func protocolCommandV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, token []byte, identity a.SigningIdentityView, o options) (Output, error) {
	result := Output{Operation: "protocol"}
	if len(o.args) != 2 {
		return result, usage("protocol get|preflight|set|recover")
	}
	result.Operation += " " + o.args[1]
	scope, e := scopeV2(w, profile, o)
	if e != nil {
		return result, e
	}
	result.OutcomeID = scope.OutcomeID
	switch o.args[1] {
	case "get":
		value, e := client.GetNamespaceWorkProtocol(ctx, scope.NamespaceID)
		if e == nil {
			result.Data = value
		}
		return result, e
	case "preflight":
		value, e := client.ReadSignedState(ctx, a.SignedStateQuery{Scope: scope, Resource: "protocol_preflight"})
		result.Data = value.Readiness
		return result, e
	case "set":
		version, e := strconv.ParseUint(o.values["version"], 10, 64)
		if e != nil || version == 0 {
			return result, usage("exact protocol --version required")
		}
		phase := d.WorkProtocolPhase(o.values["phase"])
		if phase != d.WorkProtocolSignedDraining && phase != d.WorkProtocolSigned {
			return result, usage("schema-2 protocol set accepts explicit draining_to_signed_v2 or signed_contracts_v2")
		}
		if o.values["reason"] == "" {
			return result, usage("explicit operator --reason required")
		}
		if phase == d.WorkProtocolSigned && o.values["writers-drained"] != "true" {
			return result, usage("signed activation requires --writers-drained after actually retiring v1 writers and SQL credentials")
		}
		command := a.SetNamespaceWorkProtocolCommand{Scope: scope, ExpectedProtocolVersion: d.Version(version), Phase: phase, WritersDrained: o.values["writers-drained"] == "true", Reason: o.values["reason"]}
		raw, e := freezeProtocolV2(command)
		if e != nil {
			return result, e
		}
		fp, e := a.SignedCommandFingerprint(identity.Actor, command)
		if e != nil {
			return result, e
		}
		id, e := newLocalIDV2()
		if e != nil {
			return result, e
		}
		key, e := sdk.NewIdempotencyKey()
		if e != nil {
			return result, e
		}
		intent := PendingOperationV2{ID: id, Operation: "SetNamespaceWorkProtocol", State: "prepared", Scope: ProfilePendingScopeV2{NamespaceID: scope.NamespaceID, OutcomeID: scope.OutcomeID}, IdempotencyKey: key, Payload: base64.StdEncoding.EncodeToString(raw), PayloadDigest: signing.Digest(raw), RequestFingerprint: fp}
		e = mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
			if len(current.Local.PendingOperations) >= 10 {
				return fmt.Errorf("pending limit reached")
			}
			for _, pending := range current.Local.PendingOperations {
				if pending.Operation == intent.Operation {
					return fmt.Errorf("recover the original protocol intention before creating another")
				}
			}
			current.Local.PendingOperations = append(current.Local.PendingOperations, intent)
			return nil
		})
		if e != nil {
			return result, e
		}
		accepted, e := recoverProtocolV2(ctx, w, profile, client, token, intent)
		result.Committed = !accepted.CommandID.IsZero()
		if !accepted.CommandID.IsZero() {
			result.Data = accepted.Value
		}
		if e != nil {
			result.RequiresAction = "protocol recover retains exact phase/version/drain acknowledgement; never silently change CAS"
		}
		return result, e
	case "recover":
		current, e := w.LoadProfileV2(profile.Name)
		if e != nil {
			return result, e
		}
		if e = current.VerifyBinding(token); e != nil {
			return result, e
		}
		items := []any{}
		for _, intent := range current.Local.PendingOperations {
			if intent.Operation != "SetNamespaceWorkProtocol" {
				continue
			}
			accepted, e := recoverProtocolV2(ctx, w, profile, client, token, intent)
			result.Committed = result.Committed || !accepted.CommandID.IsZero()
			item := map[string]any{"intention_id": intent.ID}
			if !accepted.CommandID.IsZero() {
				item["command_id"] = accepted.CommandID
				item["protocol"] = accepted.Value.Protocol
			}
			items = append(items, item)
			result.Data = map[string]any{"items": items}
			if e != nil {
				result.RequiresAction = "preserve original protocol intention and inspect live preflight"
				return result, e
			}
		}
		result.Data = map[string]any{"items": items}
		return result, nil
	default:
		return result, usage("protocol get|preflight|set|recover")
	}
}
func recoverProtocolV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, token []byte, intent PendingOperationV2) (sdk.CommandResult[a.NamespaceWorkProtocolResult], error) {
	var accepted sdk.CommandResult[a.NamespaceWorkProtocolResult]
	raw, e := base64.StdEncoding.Strict().DecodeString(intent.Payload)
	if e != nil || signing.Digest(raw) != intent.PayloadDigest {
		return accepted, fmt.Errorf("frozen protocol intention differs")
	}
	command, e := decodeProtocolV2(raw)
	if e != nil {
		return accepted, e
	}
	if intent.Operation != "SetNamespaceWorkProtocol" || command.Scope.NamespaceID != intent.Scope.NamespaceID || command.Scope.OutcomeID != intent.Scope.OutcomeID || (command.Phase != d.WorkProtocolSignedDraining && command.Phase != d.WorkProtocolSigned) {
		return accepted, fmt.Errorf("protocol intention target/phase differs")
	}
	var snapshot PendingOperationV2
	e = mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
		index, e := pendingIndexV2(current, intent)
		if e != nil {
			return e
		}
		snapshot = current.Local.PendingOperations[index]
		switch snapshot.State {
		case "prepared", "sent_unknown":
			current.Local.PendingOperations[index].State = "sent_unknown"
		case "accepted_confirmed":
		default:
			return fmt.Errorf("protocol state requires explicit reconciliation")
		}
		return nil
	})
	if e != nil {
		return accepted, e
	}
	if snapshot.State == "accepted_confirmed" {
		raw, e := base64.StdEncoding.Strict().DecodeString(snapshot.Response)
		if e != nil {
			return accepted, e
		}
		if e = json.Unmarshal(raw, &accepted); e != nil {
			return accepted, e
		}
	} else {
		if snapshot.State == "sent_unknown" {
			operation, lookupErr := client.ReadSignedState(ctx, a.SignedStateQuery{Scope: command.Scope, Resource: "operation", IdempotencyKey: intent.IdempotencyKey})
			if lookupErr == nil {
				if operation.OperationName != intent.Operation || operation.OperationCommandID == nil || operation.OperationFingerprint != intent.RequestFingerprint {
					return accepted, fmt.Errorf("original protocol operation differs")
				}
				response, e := base64.StdEncoding.Strict().DecodeString(operation.OperationPayload)
				if e != nil || len(response) > d.MaxSignedOperationResultBytes || signing.Digest(response) != operation.PayloadDigest {
					return accepted, fmt.Errorf("original protocol response differs")
				}
				if e = json.Unmarshal(response, &accepted); e != nil {
					return accepted, e
				}
				if accepted.CommandID != *operation.OperationCommandID {
					return accepted, fmt.Errorf("original protocol command ID differs")
				}
			} else if !notFoundReturnV2(lookupErr) {
				return accepted, lookupErr
			}
		}
		if accepted.CommandID.IsZero() {
			accepted, e = client.SetNamespaceWorkProtocol(ctx, intent.IdempotencyKey, command)
			if e != nil {
				return accepted, e
			}
		}
		if e = validateProtocolAcceptanceV2(command, accepted); e != nil {
			return accepted, e
		}
		response, e := json.Marshal(accepted)
		if e != nil || len(response) > 256<<10 {
			return accepted, fmt.Errorf("accepted protocol response exceeds local bound")
		}
		e = mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
			index, e := pendingIndexV2(current, intent)
			if e != nil {
				return e
			}
			current.Local.PendingOperations[index].State = "accepted_confirmed"
			current.Local.PendingOperations[index].Response = base64.StdEncoding.EncodeToString(response)
			return nil
		})
		if e != nil {
			return accepted, e
		}
	}
	if e = validateProtocolAcceptanceV2(command, accepted); e != nil {
		return accepted, e
	}
	e = mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
		index, e := pendingIndexV2(current, intent)
		if e != nil {
			return e
		}
		current.Local.PendingOperations = append(current.Local.PendingOperations[:index], current.Local.PendingOperations[index+1:]...)
		return nil
	})
	return accepted, e
}

// Local v2 counters are quoted decimal strings even when the unchanged v1
// administrative DTO uses numeric JSON. Never round a protocol CAS through JS.
type frozenProtocolV2 struct {
	Scope                   d.Scope             `json:"scope"`
	ExpectedProtocolVersion signing.Decimal     `json:"expected_protocol_version"`
	Phase                   d.WorkProtocolPhase `json:"phase"`
	WritersDrained          bool                `json:"writers_drained"`
	Reason                  string              `json:"reason"`
}

func freezeProtocolV2(command a.SetNamespaceWorkProtocolCommand) ([]byte, error) {
	if command.LeasePolicy != nil {
		return nil, fmt.Errorf("v2 cutover intent does not alter lease policy")
	}
	return signing.Canonical(frozenProtocolV2{command.Scope, signing.Decimal(command.ExpectedProtocolVersion), command.Phase, command.WritersDrained, command.Reason})
}
func decodeProtocolV2(raw []byte) (a.SetNamespaceWorkProtocolCommand, error) {
	var value frozenProtocolV2
	if e := signing.DecodeStrict(raw, &value, 256<<10); e != nil {
		return a.SetNamespaceWorkProtocolCommand{}, e
	}
	return a.SetNamespaceWorkProtocolCommand{Scope: value.Scope, ExpectedProtocolVersion: d.Version(value.ExpectedProtocolVersion), Phase: value.Phase, WritersDrained: value.WritersDrained, Reason: value.Reason}, nil
}

func validateProtocolAcceptanceV2(command a.SetNamespaceWorkProtocolCommand, accepted sdk.CommandResult[a.NamespaceWorkProtocolResult]) error {
	if accepted.CommandID.Validate() != nil || accepted.ResultOmitted || accepted.Value.Protocol.NamespaceID != command.Scope.NamespaceID || accepted.Value.Protocol.Phase != command.Phase || accepted.Value.Protocol.Version != command.ExpectedProtocolVersion+1 {
		return fmt.Errorf("accepted protocol result target/version differs")
	}
	return nil
}
