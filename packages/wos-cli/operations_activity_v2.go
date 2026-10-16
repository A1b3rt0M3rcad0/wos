package woscli

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/signing"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

func prepareActivityV2(ctx context.Context, w *Workspace, profile ProfileV2, token []byte, actor d.ActorRef, id d.ID, o options) (PendingOperationV2, error) {
	var intent PendingOperationV2
	e := mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
		if len(current.Local.PendingOperations) >= 10 {
			return fmt.Errorf("pending limit reached; recover existing intentions")
		}
		for _, pending := range current.Local.PendingOperations {
			if pending.ContractID != nil && *pending.ContractID == id {
				return fmt.Errorf("contract has an unresolved intention; recover it before recording another activity")
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
		if file.Execution == nil || view.Authority.ContractKind != "execution" {
			return usage("execution activities require an execution contract; review material has its own commands")
		}
		if file.Local.Pending != nil || file.Local.AcceptanceReceipt != nil {
			return fmt.Errorf("final return is frozen or accepted; preserve the original intention")
		}
		g := view.Authority
		scope := d.Scope{NamespaceID: current.Binding.NamespaceID, OutcomeID: d.ID(g.OutcomeID)}
		cmd := a.AppendWorkActivityCommand{Scope: scope, ContractID: id, SignerKeyID: current.Signing.KeyID, Authority: a.ContractAuthority{ExecutionID: d.ID(g.ExecutionID), FencingToken: d.FencingToken(g.FencingToken), SpecDigest: g.SpecDigest}, AuthorityDigest: signing.Digest(file.Issued.Authority.Payload), ExpectedContractVersion: d.Version(g.ContractVersion), Description: o.values["description"], Type: d.WorkActivityProgress}
		if len(o.args) < 3 || o.args[2] != "add" {
			return usage("work log/evidence/artifact add [reference]")
		}
		key, e := newLocalIDV2()
		if e != nil {
			return e
		}
		localKey := "record_" + strings.ReplaceAll(key.String(), "-", "")
		switch o.args[1] {
		case "log":
			if o.values["type"] != "" {
				cmd.Type = d.WorkActivityType(o.values["type"])
			}
			if !cmd.Type.Valid() {
				return usage("--type must be progress, discovery, problem, correction, validation or result")
			}
			ids, e := evidenceIDsOptionV2(o.values["evidence"])
			if e != nil {
				return e
			}
			cmd.EvidenceRefs = ids
		case "evidence":
			cmd.Type = d.WorkActivityValidation
			kind := d.EvidenceType(o.values["type"])
			if kind == "" {
				kind = d.EvidenceTypeSource
			}
			if !kind.Valid() || kind == d.EvidenceTypeMeasurement {
				return usage("--type must be test_result, inspection, attestation, source or external_evaluation; measurements require structured material")
			}
			if strings.TrimSpace(o.values["uri"]) == "" {
				return usage("evidence add requires --uri")
			}
			cmd.Evidence = []a.SyncEvidenceInput{{LocalKey: localKey, Evidence: a.RegisterEvidenceCommand{Scope: scope, EvidenceType: kind, Description: cmd.Description, SourceRef: d.SourceReference{Provider: "wos-cli", URI: o.values["uri"]}, CapturedAt: time.Now().UTC(), SourceVersion: o.values["source-version"], Checksum: o.values["content-hash"]}}}
			linkKey := localKey + "_link"
			link := a.SyncEvidenceLinkRecord{Scope: scope, TargetRef: d.EntityRef{Scope: scope, Kind: d.EntityKindWorkItem, ID: d.ID(g.WorkItemID)}, Stance: d.EvidenceStanceSupports, Rationale: cmd.Description}
			if ref := o.values["criterion"]; ref != "" {
				criterion, e := criterionReferenceV2(*current, view, ref)
				if e != nil {
					return e
				}
				link.CriterionID = &criterion.ID
				cmd.CriterionRefs = []d.WorkActivityCriterionRef{{CriterionID: criterion.ID, CriterionRevision: criterion.Revision}}
			}
			cmd.EvidenceLinks = []a.SyncEvidenceLinkInput{{LocalKey: linkKey, EvidenceLocalKey: localKey, Link: link}}
		case "artifact":
			cmd.Type = d.WorkActivityResult
			name, uri, artifactType := o.values["name"], o.values["uri"], o.values["artifact-type"]
			if strings.TrimSpace(name) == "" || strings.TrimSpace(uri) == "" || strings.TrimSpace(artifactType) == "" {
				return usage("artifact add requires --name, --uri and --artifact-type")
			}
			if cmd.Description == "" {
				cmd.Description = "Produced artifact: " + name
			}
			cmd.Artifacts = []a.SyncArtifactInput{{LocalKey: localKey, Artifact: a.RegisterArtifactCommand{Scope: scope, ArtifactType: artifactType, Name: name, URI: uri, Checksum: o.values["content-hash"], SourceVersion: o.values["source-version"]}}}
		default:
			return usage("unknown activity operation")
		}
		if strings.TrimSpace(cmd.Description) == "" || len(cmd.Description) > 16384 {
			return usage("--description must be nonempty and at most 16384 bytes")
		}
		raw, e := signing.Canonical(cmd)
		if e != nil {
			return e
		}
		fingerprint, e := a.SignedCommandFingerprint(actor, cmd)
		if e != nil {
			return e
		}
		idempotency, e := sdk.NewIdempotencyKey()
		if e != nil {
			return e
		}
		intent = PendingOperationV2{ID: key, Operation: "AppendWorkActivity", State: "prepared", Scope: ProfilePendingScopeV2{NamespaceID: scope.NamespaceID, OutcomeID: scope.OutcomeID}, ContractID: &id, IdempotencyKey: idempotency, Payload: base64.StdEncoding.EncodeToString(raw), PayloadDigest: signing.Digest(raw), RequestFingerprint: fingerprint}
		current.Local.PendingOperations = append(current.Local.PendingOperations, intent)
		return nil
	})
	return intent, e
}

func decodeActivityIntentV2(profile ProfileV2, intent PendingOperationV2) (a.AppendWorkActivityCommand, error) {
	var cmd a.AppendWorkActivityCommand
	raw, e := base64.StdEncoding.Strict().DecodeString(intent.Payload)
	if e != nil || intent.Operation != "AppendWorkActivity" || signing.Digest(raw) != intent.PayloadDigest {
		return cmd, fmt.Errorf("frozen activity payload differs")
	}
	if e = signing.DecodeStrict(raw, &cmd, signing.MaxPayloadBytes); e != nil {
		return cmd, e
	}
	if intent.ContractID == nil || cmd.ContractID != *intent.ContractID || cmd.Scope != (d.Scope{NamespaceID: intent.Scope.NamespaceID, OutcomeID: intent.Scope.OutcomeID}) || cmd.Scope.NamespaceID != profile.Binding.NamespaceID || cmd.SignerKeyID != profile.Signing.KeyID {
		return cmd, fmt.Errorf("frozen activity destination/contract differs")
	}
	return cmd, nil
}

func reconcileActivityV2(ctx context.Context, client *sdk.Client, intent PendingOperationV2) (sdk.CommandResult[a.WorkActivityResult], error) {
	var accepted sdk.CommandResult[a.WorkActivityResult]
	scope := d.Scope{NamespaceID: intent.Scope.NamespaceID, OutcomeID: intent.Scope.OutcomeID}
	state, e := client.ReadSignedState(ctx, a.SignedStateQuery{Scope: scope, Resource: "operation", IdempotencyKey: intent.IdempotencyKey})
	if e != nil {
		return accepted, e
	}
	if state.Scope == nil || *state.Scope != scope || state.OperationName != intent.Operation || state.OperationCommandID == nil || state.OperationFingerprint != intent.RequestFingerprint || intent.RequestFingerprint == "" {
		return accepted, fmt.Errorf("durable activity result differs from frozen intention")
	}
	raw, e := base64.StdEncoding.Strict().DecodeString(state.OperationPayload)
	if e != nil || len(raw) > d.MaxSignedOperationResultBytes || signing.Digest(raw) != state.PayloadDigest {
		return accepted, fmt.Errorf("durable activity result bytes differ")
	}
	if e = json.Unmarshal(raw, &accepted); e != nil {
		return accepted, e
	}
	if accepted.CommandID != *state.OperationCommandID {
		return accepted, fmt.Errorf("durable activity command identity differs")
	}
	return accepted, nil
}

func validateActivityAcceptedV2(profile ProfileV2, cmd a.AppendWorkActivityCommand, accepted sdk.CommandResult[a.WorkActivityResult]) error {
	if accepted.ResultOmitted || accepted.CommandID.IsZero() {
		return fmt.Errorf("activity result omitted; preserve original intention")
	}
	activity := accepted.Value.Activity
	if e := activity.Validate(); e != nil {
		return e
	}
	if activity.Scope != cmd.Scope || activity.ContractID != cmd.ContractID || activity.PrincipalID != profile.Binding.PrincipalID || activity.Type != cmd.Type || activity.Description != strings.TrimSpace(cmd.Description) {
		return fmt.Errorf("accepted activity target differs")
	}
	if uint64(accepted.OutcomeRevision) != activity.Sequence {
		return fmt.Errorf("accepted activity chronology differs from command receipt")
	}
	for _, item := range accepted.Value.Evidence {
		if item.Scope != cmd.Scope {
			return fmt.Errorf("accepted evidence target differs")
		}
	}
	for _, item := range accepted.Value.Artifacts {
		if item.Scope != cmd.Scope {
			return fmt.Errorf("accepted artifact target differs")
		}
	}
	return nil
}

func appendUniqueIDV2(ids []d.ID, id d.ID) []d.ID {
	for _, existing := range ids {
		if existing == id {
			return ids
		}
	}
	return append(ids, id)
}

func applyActivityMaterialV2(file *ContractFileV2, accepted a.WorkActivityResult) error {
	if file.Execution == nil || file.Local.Pending != nil || file.Local.AcceptanceReceipt != nil {
		return fmt.Errorf("activity material cannot replace a frozen or non-execution draft")
	}
	material := &file.Execution.Material.Result
	if material.ContractID != accepted.Activity.ContractID || material.WorkItemID != accepted.Activity.WorkItemID {
		return fmt.Errorf("activity material binding differs")
	}
	for _, id := range accepted.Activity.EvidenceRefs {
		material.EvidenceIDs = appendUniqueIDV2(material.EvidenceIDs, id)
	}
	for _, artifact := range accepted.Activity.ArtifactVersions {
		found := false
		for _, existing := range material.Artifacts {
			if existing.ArtifactID == artifact.ArtifactID {
				if existing != artifact {
					return fmt.Errorf("artifact exact version changed; explicit reconciliation required")
				}
				found = true
			}
		}
		if !found {
			material.Artifacts = append(material.Artifacts, artifact)
		}
	}
	for _, ref := range accepted.Activity.CriterionRefs {
		index := -1
		for i, existing := range material.CriterionEvidence {
			if existing.CriterionID == ref.CriterionID {
				if existing.CriterionRevision != ref.CriterionRevision {
					return fmt.Errorf("criterion revision changed; preserve material")
				}
				index = i
			}
		}
		if index < 0 {
			material.CriterionEvidence = append(material.CriterionEvidence, d.SignedCriterionEvidence{CriterionID: ref.CriterionID, CriterionRevision: ref.CriterionRevision, EvidenceIDs: []d.ID{}})
			index = len(material.CriterionEvidence) - 1
		}
		for _, id := range accepted.Activity.EvidenceRefs {
			material.CriterionEvidence[index].EvidenceIDs = appendUniqueIDV2(material.CriterionEvidence[index].EvidenceIDs, id)
		}
	}
	return nil
}

// An uncertain append replays the original canonical payload/key. Accepted
// server history is retained even when local publication needs recovery.
func recoverActivityV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, token []byte, intent PendingOperationV2) (sdk.CommandResult[a.WorkActivityResult], error) {
	var accepted sdk.CommandResult[a.WorkActivityResult]
	cmd, e := decodeActivityIntentV2(profile, intent)
	if e != nil {
		return accepted, e
	}
	snapshot := intent
	lookup := false
	e = mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
		index, e := pendingIndexV2(current, intent)
		if e != nil {
			return e
		}
		snapshot = current.Local.PendingOperations[index]
		if snapshot.State == "accepted_unmaterialized" {
			return nil
		}
		if snapshot.State != "prepared" && snapshot.State != "sent_unknown" {
			return fmt.Errorf("activity intention state requires explicit reconciliation")
		}
		lookup = snapshot.State == "sent_unknown"
		current.Local.PendingOperations[index].State = "sent_unknown"
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
		if lookup {
			accepted, e = reconcileActivityV2(ctx, client, intent)
			if e != nil {
				var remote *sdk.Error
				if !errors.As(e, &remote) || remote.Code != "not_found" {
					return accepted, e
				}
				accepted, e = client.AppendWorkActivity(ctx, intent.IdempotencyKey, cmd)
			}
		} else {
			accepted, e = client.AppendWorkActivity(ctx, intent.IdempotencyKey, cmd)
		}
		if e != nil {
			return accepted, e
		}
		if accepted.ResultOmitted {
			accepted, e = reconcileActivityV2(ctx, client, intent)
			if e != nil {
				return sdk.CommandResult[a.WorkActivityResult]{}, e
			}
		}
		if e = validateActivityAcceptedV2(profile, cmd, accepted); e != nil {
			return sdk.CommandResult[a.WorkActivityResult]{}, e
		}
		// The full original result remains in the server's durable registry.
		// Local publication needs only its validated Activity and command identity;
		// documentary arrays duplicate potentially large captured version metadata.
		cached := accepted
		cached.Value = a.WorkActivityResult{Activity: accepted.Value.Activity}
		raw, e := json.Marshal(cached)
		if e != nil {
			return accepted, e
		}
		if len(raw) > 256<<10 {
			return accepted, fmt.Errorf("activity response exceeds local bound; original result remains recoverable")
		}
		e = mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
			index, e := pendingIndexV2(current, intent)
			if e != nil {
				return e
			}
			next := &current.Local.PendingOperations[index]
			if next.State == "accepted_unmaterialized" && next.Response != base64.StdEncoding.EncodeToString(raw) {
				return fmt.Errorf("concurrent activity result differs")
			}
			next.State, next.Response = "accepted_unmaterialized", base64.StdEncoding.EncodeToString(raw)
			return nil
		})
		if e != nil {
			return accepted, e
		}
	}
	if e = validateActivityAcceptedV2(profile, cmd, accepted); e != nil {
		return sdk.CommandResult[a.WorkActivityResult]{}, e
	}
	e = mutateProfileV2(ctx, w, profile, token, func(current *ProfileV2) error {
		index, e := pendingIndexV2(current, intent)
		if e != nil {
			return e
		}
		release, e := w.LockV2(ctx, current.Name, cmd.ContractID.String())
		if e != nil {
			return e
		}
		defer release()
		file, view, path, raw, e := loadContractV2(w, *current, cmd.ContractID)
		if e != nil {
			return e
		}
		if view.Authority.WorkItemID != accepted.Value.Activity.WorkItemID.String() || view.Authority.SpecDigest != cmd.Authority.SpecDigest {
			return fmt.Errorf("local activity contract target changed")
		}
		if e = applyActivityMaterialV2(&file, accepted.Value); e != nil {
			return e
		}
		if e = w.WriteV2(path, file, signing.Digest(raw)); e != nil {
			return e
		}
		current.Local.PendingOperations = append(current.Local.PendingOperations[:index], current.Local.PendingOperations[index+1:]...)
		return nil
	})
	return accepted, e
}

func activityCommandV2(ctx context.Context, w *Workspace, profile ProfileV2, client *sdk.Client, token []byte, actor d.ActorRef, o options) (Output, error) {
	result := Output{Operation: strings.Join(o.args[:min(len(o.args), 3)], " ")}
	if len(o.args) < 3 || o.args[2] != "add" {
		return result, usage("work log/evidence/artifact add [reference]")
	}
	id, e := contractOptionV2(w, profile, "execution", o, 3)
	if e != nil {
		return result, e
	}
	result.ContractID = id
	intent, e := prepareActivityV2(ctx, w, profile, token, actor, id, o)
	if e != nil {
		return result, e
	}
	accepted, e := recoverActivityV2(ctx, w, profile, client, token, intent)
	result.Committed = !accepted.CommandID.IsZero()
	if result.Committed {
		result.Data = map[string]any{"activity": accepted.Value.Activity, "command_id": accepted.CommandID, "remote_committed": true, "local_saved": e == nil, "evidence": accepted.Value.Evidence, "artifacts": accepted.Value.Artifacts}
	}
	if e != nil {
		result.RequiresAction = "work recover replays this exact activity intention; preserve its payload and key"
	}
	return result, e
}
