package woscli

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"time"
)

func (w *Workspace) drafts(state State, dir string) (CheckpointDraft, ResultDraft, error) {
	var checkpoint CheckpointDraft
	var result ResultDraft
	var contract ContractDocument
	contractPath, err := w.DocumentPath(filepath.Join(dir, "contract"))
	if err != nil {
		return checkpoint, result, usage(err.Error())
	}
	if err = w.ReadDocument(contractPath, &contract); err != nil {
		return checkpoint, result, usage(err.Error())
	}
	c := contract.Contract
	if contract.SchemaVersion != 1 || contract.Kind != "WorkContract" || c.ID != state.Contract.ID || c.Scope != state.Scope || c.WorkItemID != state.Contract.WorkItemID || c.SpecDigest != state.Contract.SpecDigest || c.HolderPrincipalID != state.Contract.HolderPrincipalID {
		return checkpoint, result, usage("contract document binding differs from server-issued base")
	}
	if err = c.Validate(); err != nil {
		return checkpoint, result, usage(err.Error())
	}
	digest, err := domain.SemanticDigest(domain.NormalizeContractSpec(c.Spec))
	if err != nil || digest != state.Contract.SpecDigest {
		return checkpoint, result, usage("immutable contract spec differs")
	}
	path, err := w.DocumentPath(filepath.Join(dir, "checkpoint"))
	if err != nil {
		return checkpoint, result, usage(err.Error())
	}
	if err = w.ReadDocument(path, &checkpoint); err != nil {
		return checkpoint, result, usage(err.Error())
	}
	if checkpoint.SchemaVersion != 1 || checkpoint.Kind != "WorkCheckpointDraft" || checkpoint.ContractID != c.ID || checkpoint.SpecDigest != c.SpecDigest {
		return checkpoint, result, usage("checkpoint schema or binding differs")
	}
	path, err = w.DocumentPath(filepath.Join(dir, "result"))
	if err != nil {
		return checkpoint, result, usage(err.Error())
	}
	if err = w.ReadDocument(path, &result); err != nil {
		return checkpoint, result, usage(err.Error())
	}
	if result.SchemaVersion != 1 || result.Kind != "WorkResult" || result.ContractID != c.ID || result.SpecDigest != c.SpecDigest || result.Material.ContractID != c.ID || result.Material.WorkItemID != c.WorkItemID || result.Material.SpecDigest != c.SpecDigest {
		return checkpoint, result, usage("result schema or binding differs")
	}
	if len(result.Artifacts)+len(result.Evidence)+len(result.EvidenceLinks) > 100 {
		return checkpoint, result, usage("at most 100 documentary records per sync")
	}
	return checkpoint, result, nil
}
func authority(state State) application.ContractAuthority {
	return application.ContractAuthority{ExecutionID: state.Contract.ExecutionID, FencingToken: state.Contract.FencingToken, SpecDigest: state.Contract.SpecDigest}
}
func (w *Workspace) applyResult(config Config, dir string, raw json.RawMessage) (State, error) {
	var result sdk.CommandResult[application.WorkContractResult]
	if err := json.Unmarshal(raw, &result); err != nil {
		return State{}, err
	}
	if result.ResultOmitted {
		return State{}, fmt.Errorf("committed result omitted; recover using contract IDs")
	}
	if err := result.Value.Contract.Validate(); err != nil {
		return State{}, err
	}
	// Preserve accumulated mappings and refuse to regress state when replaying older receipts.
	var old State
	if data, err := w.Read(filepath.Join(dir, "state.json")); err == nil {
		if err = json.Unmarshal(data, &old); err != nil {
			return State{}, err
		}
		if old.Contract.ID != result.Value.Contract.ID {
			return State{}, fmt.Errorf("receipt belongs to another contract")
		}
		if old.Contract.Version > result.Value.Contract.Version || old.Contract.LeaseVersion > result.Value.Contract.LeaseVersion {
			return old, nil
		}
		if result.Value.LocalKeys == nil {
			result.Value.LocalKeys = map[string]domain.ID{}
		}
		for key, id := range old.LocalKeys {
			if _, ok := result.Value.LocalKeys[key]; !ok {
				result.Value.LocalKeys[key] = id
			}
		}
	}
	if _, err := w.materialize(config, result.Value, result.CommandID); err != nil {
		return State{}, err
	}
	data, err := w.Read(filepath.Join(dir, "state.json"))
	if err != nil {
		return State{}, err
	}
	var state State
	err = json.Unmarshal(data, &state)
	return state, err
}
func liveAuthority(ctx context.Context, client *sdk.Client, state State) (application.WorkContractView, error) {
	got, err := client.GetWorkContract(ctx, state.Scope, state.Contract.ID)
	if err != nil {
		return application.WorkContractView{}, err
	}
	c := got.Value.Contract
	if got.Value.EffectiveStatus != domain.ContractActive {
		return got.Value, &exitError{5, fmt.Errorf("contract authority is %s", got.Value.EffectiveStatus)}
	}
	if c.ExecutionID != state.Contract.ExecutionID || c.FencingToken != state.Contract.FencingToken || c.HolderPrincipalID != state.Contract.HolderPrincipalID {
		return got.Value, &exitError{5, fmt.Errorf("execution changed; explicit resume/takeover is required")}
	}
	if c.SpecDigest != state.Contract.SpecDigest {
		return got.Value, &exitError{4, fmt.Errorf("server spec binding differs")}
	}
	return got.Value, nil
}
func (w *Workspace) confirmedMutation(ctx context.Context, client *sdk.Client, config Config, state State, dir, name string, cmd any, result Output) (State, Output, error) {
	raw, receipt, err := w.Mutate(ctx, client, config, state.Scope, dir, name, cmd)
	result.Data = raw
	result.ReceiptPath = receipt
	result.Committed = err == nil
	if err != nil {
		return state, result, err
	}
	updated, err := w.applyResult(config, dir, raw)
	if err != nil {
		return state, result, &LocalError{Err: err, Committed: true, ReceiptPath: receipt}
	}
	if sync, ok := cmd.(application.SyncWorkContractCommand); ok {
		if err = recordLocalDigests(w, dir, &updated, sync); err != nil {
			return state, result, &LocalError{Err: err, Committed: true, ReceiptPath: receipt}
		}
	}
	return updated, result, nil
}
func workWorkspaceCommand(ctx context.Context, w *Workspace, client *sdk.Client, config Config, scope domain.Scope, state State, dir string, o options, result Output) (Output, error) {
	operation := o.args[1]
	if operation == "keepalive" {
		if o.values["foreground"] != "true" {
			return result, usage("keepalive requires --foreground")
		}
		return keepalive(ctx, w, client, config, state, dir, o, result)
	}
	if operation == "recover" && o.values["break-lock"] == "true" {
		path := filepath.Join(dir, "workspace.lock")
		if err := w.check(path); err != nil {
			return result, &LocalError{Err: err}
		}
		if err := w.root.Remove(path); err != nil && !os.IsNotExist(err) {
			return result, &LocalError{Err: err}
		}
	}
	unlock, err := w.Lock(dir)
	if err != nil {
		return result, &LocalError{Err: err}
	}
	defer unlock()
	if operation == "recover" {
		return recoverContract(ctx, w, client, config, state, dir, result)
	}
	if operation == "validate" || operation == "diff" {
		checkpoint, draft, err := w.drafts(state, dir)
		if err != nil {
			return result, err
		}
		result.Data = map[string]any{"checkpoint": checkpoint, "result": draft, "base_contract_version": state.Contract.Version, "base_lease_version": state.Contract.LeaseVersion, "local_validation_only": true}
		if operation == "diff" {
			view, err := client.GetWorkContract(ctx, scope, state.Contract.ID)
			if err != nil {
				return result, err
			}
			result.Data = map[string]any{"local_checkpoint": checkpoint, "local_result": draft, "server": view.Value, "base_differs": view.Value.Contract.Version != state.Contract.Version}
		}
		return result, nil
	}
	if err = w.ensureNoPending(dir); err != nil {
		return result, err
	}
	if operation == "refresh" || operation == "resume" && o.values["takeover"] != "true" {
		view, err := liveAuthority(ctx, client, state)
		if err != nil {
			return result, err
		}
		work, err := client.GetWorkContext(ctx, scope, state.Contract.WorkItemID)
		if err != nil {
			return result, err
		}
		if work.OutcomeRevision != viewRevision(ctx, client, scope, state.Contract.ID, view.Contract) {
			return result, &exitError{4, fmt.Errorf("state changed during refresh; read again")}
		}
		state.Contract = view.Contract
		state.WorkItemVersion = work.Work.Version
		if err = w.saveJSON(filepath.Join(dir, "state.json"), state); err != nil {
			return result, &LocalError{Err: err}
		}
		result.Data = view
		return result, nil
	}
	if operation == "keepalive" {
		if o.values["foreground"] != "true" {
			return result, usage("keepalive requires --foreground")
		}
		return keepalive(ctx, w, client, config, state, dir, o, result)
	}
	if _, err = liveAuthority(ctx, client, state); err != nil {
		return result, err
	}
	ttl := config.Lease.RequestedTTLSeconds
	if value := o.values["ttl"]; value != "" {
		ttl, err = strconv.Atoi(value)
		if err != nil {
			return result, usage("invalid ttl")
		}
	}
	switch operation {
	case "renew":
		_, result, err = w.confirmedMutation(ctx, client, config, state, dir, "renew_work_contract", application.RenewWorkContractCommand{Scope: scope, ContractID: state.Contract.ID, Authority: authority(state), ExpectedLeaseVersion: state.Contract.LeaseVersion, TTLSeconds: ttl}, result)
		return result, err
	case "resume":
		if o.values["takeover"] != "true" {
			return result, usage("use explicit --takeover")
		}
		_, result, err = w.confirmedMutation(ctx, client, config, state, dir, "resume_work_contract", application.ResumeWorkContractCommand{Scope: scope, ContractID: state.Contract.ID, Authority: authority(state), ExpectedLeaseVersion: state.Contract.LeaseVersion}, result)
		return result, err
	case "checkpoint", "sync", "submit", "finalize", "finish":
		checkpoint, draft, err := w.drafts(state, dir)
		if err != nil {
			return result, err
		}
		if err = validateMappedRecords(state, draft); err != nil {
			return result, err
		}
		if operation == "checkpoint" || operation == "sync" || operation == "finish" {
			sync := application.SyncWorkContractCommand{Scope: scope, ContractID: state.Contract.ID, Authority: authority(state), ExpectedContractVersion: state.Contract.Version, Checkpoint: checkpoint.Checkpoint}
			if operation != "checkpoint" {
				sync.Artifacts, sync.Evidence, sync.EvidenceLinks = unsyncedRecords(state, draft)
			}
			state, result, err = w.confirmedMutation(ctx, client, config, state, dir, "sync_work_contract", sync, result)
			if err != nil {
				return result, err
			}
			if operation != "finish" {
				return result, nil
			}
		}
		if operation == "submit" || operation == "finish" {
			material, err := resolvedMaterial(state, draft)
			if err != nil {
				return result, err
			}
			state, result, err = w.confirmedMutation(ctx, client, config, state, dir, "submit_work_result", application.SubmitWorkResultCommand{Scope: scope, ContractID: state.Contract.ID, Authority: authority(state), ExpectedContractVersion: state.Contract.Version, Material: material, SupersedesSubmissionID: state.Contract.LatestSubmissionID}, result)
			if err != nil {
				return result, err
			}
			if operation == "submit" {
				result.RequiresAction = "review_or_explicit_finalize"
				return result, nil
			}
		}
		submission := state.Contract.LatestSubmissionID
		if value := o.values["submission"]; value != "" {
			id, err := domain.ParseID(value)
			if err != nil {
				return result, usage(err.Error())
			}
			submission = &id
		}
		if submission == nil {
			return result, usage("finalize requires --submission or an observed submitted result")
		}
		// Assessment mutates WorkItem version separately; explicitly observe that live version
		// without changing the contract content intent or execution generation.
		work, err := client.GetWorkContext(ctx, scope, state.Contract.WorkItemID)
		if err != nil {
			return result, err
		}
		if work.Contract == nil || work.Contract.Contract.Version != state.Contract.Version {
			return result, &exitError{4, fmt.Errorf("contract changed; compare and refresh before new intent")}
		}
		reason := o.values["reason"]
		if reason == "" {
			reason = draft.Material.Summary
		}
		_, result, err = w.confirmedMutation(ctx, client, config, state, dir, "finalize_work_contract", application.FinalizeWorkContractCommand{Scope: scope, ContractID: state.Contract.ID, Authority: authority(state), ExpectedContractVersion: state.Contract.Version, ExpectedWorkItemVersion: work.Work.Version, SubmissionID: *submission, Reason: reason}, result)
		if exitCode(err) == 7 {
			result.RequiresAction = "review_required"
			result.Committed = operation == "finish"
		}
		return result, err
	}
	return result, usage("unknown work operation")
}

// Refresh consistency is checked with a second bounded contract read; no stale
// execution generation is replaced implicitly.
func viewRevision(ctx context.Context, client *sdk.Client, scope domain.Scope, id domain.ID, c domain.WorkContract) domain.OutcomeRevision {
	v, err := client.GetWorkContract(ctx, scope, id)
	if err != nil || v.Value.Contract.Version != c.Version || v.Value.Contract.LeaseVersion != c.LeaseVersion {
		return 0
	}
	return v.OutcomeRevision
}
func unsyncedRecords(state State, draft ResultDraft) ([]application.SyncArtifactInput, []application.SyncEvidenceInput, []application.SyncEvidenceLinkInput) {
	artifacts := []application.SyncArtifactInput{}
	evidence := []application.SyncEvidenceInput{}
	links := []application.SyncEvidenceLinkInput{}
	for _, a := range draft.Artifacts {
		if _, exists := state.LocalKeys[a.LocalKey]; !exists {
			artifacts = append(artifacts, a)
		}
	}
	for _, e := range draft.Evidence {
		if _, exists := state.LocalKeys[e.LocalKey]; exists {
			continue
		}
		if id, exists := state.LocalKeys[e.ArtifactLocalKey]; exists {
			e.Evidence.ArtifactID = &id
			e.ArtifactLocalKey = ""
		}
		evidence = append(evidence, e)
	}
	for _, l := range draft.EvidenceLinks {
		if _, exists := state.LocalKeys[l.LocalKey]; exists {
			continue
		}
		if id, exists := state.LocalKeys[l.EvidenceLocalKey]; exists {
			l.Link.EvidenceID = &id
			l.EvidenceLocalKey = ""
		}
		links = append(links, l)
	}
	return artifacts, evidence, links
}
func resolvedMaterial(state State, draft ResultDraft) (domain.WorkResultMaterial, error) {
	material := domain.NormalizeResultMaterial(draft.Material)
	for _, a := range draft.Artifacts {
		id, exists := state.LocalKeys[a.LocalKey]
		if !exists {
			return material, usage("unsynced artifact local_key " + a.LocalKey)
		}
		material.Artifacts = append(material.Artifacts, domain.SubmissionArtifact{ArtifactID: id, SourceVersion: a.Artifact.SourceVersion, Checksum: a.Artifact.Checksum})
	}
	for _, e := range draft.Evidence {
		id, exists := state.LocalKeys[e.LocalKey]
		if !exists {
			return material, usage("unsynced evidence local_key " + e.LocalKey)
		}
		material.EvidenceIDs = append(material.EvidenceIDs, id)
	}
	for _, c := range draft.CriterionEvidenceKeys {
		ref := domain.SubmissionCriterionEvidence{CriterionID: c.CriterionID, CriterionRevision: c.CriterionRevision, EvidenceIDs: []domain.ID{}}
		for _, key := range c.EvidenceKeys {
			id, exists := state.LocalKeys[key]
			if !exists {
				return material, usage("unsynced criterion evidence key " + key)
			}
			ref.EvidenceIDs = append(ref.EvidenceIDs, id)
		}
		material.CriterionEvidence = append(material.CriterionEvidence, ref)
	}
	return material, nil
}
func keepalive(ctx context.Context, w *Workspace, client *sdk.Client, config Config, state State, dir string, o options, result Output) (Output, error) {
	ttl := config.Lease.RequestedTTLSeconds
	interval := time.Duration(ttl/3) * time.Second
	if value := o.values["interval"]; value != "" {
		seconds, err := strconv.Atoi(value)
		if err != nil || seconds < 1 || seconds >= ttl {
			return result, usage("interval must be positive and below TTL")
		}
		interval = time.Duration(seconds) * time.Second
	}
	timer := time.NewTicker(interval)
	defer timer.Stop()
	renewed := 0
	for {
		select {
		case <-ctx.Done():
			result.Data = map[string]any{"renewals": renewed, "stopped": true}
			return result, nil
		default:
		}
		release, err := w.Lock(dir)
		if err != nil {
			return result, &LocalError{Err: err}
		}
		latest, _, err := w.loadState(config, state.Scope, state.Contract.WorkItemID, state.Contract.ID.String())
		if err != nil {
			release()
			return result, err
		}
		state = latest
		if err = w.ensureNoPending(dir); err != nil {
			release()
			return result, err
		}
		if _, err = liveAuthority(ctx, client, state); err != nil {
			release()
			return result, err
		}
		updated, out, err := w.confirmedMutation(ctx, client, config, state, dir, "renew_work_contract", application.RenewWorkContractCommand{Scope: state.Scope, ContractID: state.Contract.ID, Authority: authority(state), ExpectedLeaseVersion: state.Contract.LeaseVersion, TTLSeconds: ttl}, result)
		release()
		if err != nil {
			return out, err
		}
		state = updated
		result = out
		renewed++
		select {
		case <-ctx.Done():
			result.Data = map[string]any{"renewals": renewed, "stopped": true, "contract_id": state.Contract.ID}
			return result, nil
		case <-timer.C:
		}
	}
}

func recordLocalDigests(w *Workspace, dir string, state *State, cmd application.SyncWorkContractCommand) error {
	if state.LocalKeyDigests == nil {
		state.LocalKeyDigests = map[string]string{}
	}
	store := func(key string, v any) error {
		raw, err := commands.Encode(v)
		if err != nil {
			return err
		}
		digest, err := domain.SemanticDigest(json.RawMessage(raw))
		if err != nil {
			return err
		}
		state.LocalKeyDigests[key] = digest
		return nil
	}
	for _, a := range cmd.Artifacts {
		if err := store(a.LocalKey, a); err != nil {
			return err
		}
	}
	for _, e := range cmd.Evidence {
		if id, exists := state.LocalKeys[e.ArtifactLocalKey]; exists {
			e.Evidence.ArtifactID = &id
			e.ArtifactLocalKey = ""
		}
		if err := store(e.LocalKey, e); err != nil {
			return err
		}
	}
	for _, l := range cmd.EvidenceLinks {
		if id, exists := state.LocalKeys[l.EvidenceLocalKey]; exists {
			l.Link.EvidenceID = &id
			l.EvidenceLocalKey = ""
		}
		if err := store(l.LocalKey, l); err != nil {
			return err
		}
	}
	return w.saveJSON(filepath.Join(dir, "state.json"), state)
}
func validateMappedRecords(state State, draft ResultDraft) error {
	check := func(key string, v any) error {
		if _, exists := state.LocalKeys[key]; !exists {
			return nil
		}
		expected, exists := state.LocalKeyDigests[key]
		if !exists {
			return &exitError{4, fmt.Errorf("local_key %s lacks its immutable base; recover journal first", key)}
		}
		raw, err := commands.Encode(v)
		if err != nil {
			return usage(err.Error())
		}
		digest, err := domain.SemanticDigest(json.RawMessage(raw))
		if err != nil {
			return usage(err.Error())
		}
		if digest != expected {
			return &exitError{4, fmt.Errorf("registered local_key %s changed; use a new key for new material", key)}
		}
		return nil
	}
	for _, a := range draft.Artifacts {
		if err := check(a.LocalKey, a); err != nil {
			return err
		}
	}
	for _, e := range draft.Evidence {
		if id, exists := state.LocalKeys[e.ArtifactLocalKey]; exists {
			e.Evidence.ArtifactID = &id
			e.ArtifactLocalKey = ""
		}
		if err := check(e.LocalKey, e); err != nil {
			return err
		}
	}
	for _, l := range draft.EvidenceLinks {
		if id, exists := state.LocalKeys[l.EvidenceLocalKey]; exists {
			l.Link.EvidenceID = &id
			l.EvidenceLocalKey = ""
		}
		if err := check(l.LocalKey, l); err != nil {
			return err
		}
	}
	return nil
}
func decodeSyncPayload(payload FrozenPayload) (application.SyncWorkContractCommand, error) {
	var cmd application.SyncWorkContractCommand
	normalized, err := commands.Normalize([]byte(payload), reflect.TypeFor[application.SyncWorkContractCommand]())
	if err != nil {
		return cmd, err
	}
	err = commands.Decode(normalized, &cmd)
	return cmd, err
}
