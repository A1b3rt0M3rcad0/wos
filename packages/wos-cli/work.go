package woscli

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"os"
	"path/filepath"
	"strconv"
)

type State struct {
	SchemaVersion          int                  `json:"workspace_schema_version"`
	Destination            Destination          `json:"destination"`
	Scope                  domain.Scope         `json:"scope"`
	Contract               domain.WorkContract  `json:"contract"`
	WorkItemVersion        domain.Version       `json:"work_item_version"`
	LocalKeys              map[string]domain.ID `json:"local_keys"`
	LastConfirmedCommandID domain.ID            `json:"last_confirmed_command_id,omitempty"`
}

func stateDir(work, contract domain.ID) string {
	return filepath.Join(".wos", "work", work.String(), contract.String())
}
func (w *Workspace) materialize(config Config, result application.WorkContractResult, commandID domain.ID) (string, error) {
	c := result.Contract
	dir := stateDir(c.WorkItemID, c.ID)
	if err := w.Mkdir(dir); err != nil {
		return dir, err
	}
	path, err := w.DocumentPath(filepath.Join(dir, "contract"))
	if err != nil {
		return dir, err
	}
	if _, err = w.Read(path); os.IsNotExist(err) {
		if err = w.WriteDocument(path, ContractDocument{1, "WorkContract", c}); err != nil {
			return dir, err
		}
	} else if err != nil {
		return dir, err
	}
	checkpointPath, err := w.DocumentPath(filepath.Join(dir, "checkpoint"))
	if err != nil {
		return dir, err
	}
	if _, err = w.Read(checkpointPath); os.IsNotExist(err) {
		draft := CheckpointDraft{SchemaVersion: 1, Kind: "WorkCheckpointDraft", ContractID: c.ID, SpecDigest: c.SpecDigest, Checkpoint: application.ContractCheckpointInput{Summary: "Record material progress", Completed: []string{}, Pending: []string{}, Unknown: []string{}}}
		if err = w.WriteDocument(checkpointPath, draft); err != nil {
			return dir, err
		}
	} else if err != nil {
		return dir, err
	}
	resultPath, err := w.DocumentPath(filepath.Join(dir, "result"))
	if err != nil {
		return dir, err
	}
	if _, err = w.Read(resultPath); os.IsNotExist(err) {
		draft := ResultDraft{SchemaVersion: 1, Kind: "WorkResult", ContractID: c.ID, SpecDigest: c.SpecDigest, Material: domain.NormalizeResultMaterial(domain.WorkResultMaterial{ContractID: c.ID, WorkItemID: c.WorkItemID, SpecDigest: c.SpecDigest, Summary: "Describe delivered result"}), Artifacts: []application.SyncArtifactInput{}, Evidence: []application.SyncEvidenceInput{}, EvidenceLinks: []application.SyncEvidenceLinkInput{}, CriterionEvidenceKeys: []CriterionEvidenceKeys{}}
		if err = w.WriteDocument(resultPath, draft); err != nil {
			return dir, err
		}
	} else if err != nil {
		return dir, err
	}
	state := State{SchemaVersion: 1, Destination: config.Destination(), Scope: c.Scope, Contract: c, WorkItemVersion: result.WorkItem.Version, LocalKeys: result.LocalKeys, LastConfirmedCommandID: commandID}
	oldRaw, oldErr := w.Read(filepath.Join(dir, "state.json"))
	if oldErr == nil {
		var old State
		if err = json.Unmarshal(oldRaw, &old); err != nil {
			return dir, err
		}
		if state.LocalKeys == nil {
			state.LocalKeys = old.LocalKeys
		}
	}
	if err = w.saveJSON(filepath.Join(dir, "state.json"), state); err != nil {
		return dir, err
	}
	if err = w.saveJSON(filepath.Join(".wos", "work", c.WorkItemID.String(), "current.json"), map[string]domain.ID{"contract_id": c.ID}); err != nil {
		return dir, err
	}
	return dir, nil
}
func (w *Workspace) loadState(config Config, scope domain.Scope, work domain.ID, contractFlag string) (State, string, error) {
	var state State
	var contract domain.ID
	var err error
	if contractFlag != "" {
		contract, err = domain.ParseID(contractFlag)
		if err != nil {
			return state, "", usage(err.Error())
		}
	} else {
		raw, err := w.Read(filepath.Join(".wos", "work", work.String(), "current.json"))
		if err != nil {
			return state, "", &LocalError{Err: err}
		}
		var pointer struct {
			ContractID domain.ID `json:"contract_id"`
		}
		if err = json.Unmarshal(raw, &pointer); err != nil {
			return state, "", &LocalError{Err: err}
		}
		contract = pointer.ContractID
	}
	if err = contract.Validate(); err != nil {
		return state, "", usage(err.Error())
	}
	dir := stateDir(work, contract)
	raw, err := w.Read(filepath.Join(dir, "state.json"))
	if err != nil {
		return state, dir, &LocalError{Err: err}
	}
	if err = json.Unmarshal(raw, &state); err != nil {
		return state, dir, &LocalError{Err: err}
	}
	if state.SchemaVersion != 1 || state.Destination != config.Destination() || state.Scope != scope || state.Contract.ID != contract || state.Contract.WorkItemID != work {
		return state, dir, usage("local state binding differs from trusted scope")
	}
	return state, dir, nil
}
func workCommand(ctx context.Context, w *Workspace, client *sdk.Client, config Config, scope domain.Scope, o options) (Output, error) {
	result := Output{Operation: "work", OutcomeID: scope.OutcomeID}
	if len(o.args) < 2 {
		return result, usage("work subcommand required")
	}
	operation := o.args[1]
	result.Operation = "work " + operation
	if operation == "list" || operation == "get" || operation == "create" {
		return planningCommand(ctx, w, client, config, scope, o)
	}
	if operation == "checkout" {
		if o.values["dry-run"] == "true" {
			page, err := client.ListAvailableWork(ctx, scope, 25, o.values["cursor"])
			result.Data = page
			result.RequiresAction = "dry_run_does_not_reserve_work"
			return result, err
		}
		dir := filepath.Join(".wos", "checkout", scope.OutcomeID.String(), "next")
		name := "acquire_next_work_contract"
		var cmd any = application.AcquireNextWorkContractCommand{Scope: scope, TTLSeconds: config.Lease.RequestedTTLSeconds, Limit: 25, Cursor: o.values["cursor"]}
		if o.values["next"] != "true" {
			if len(o.args) != 3 {
				return result, usage("checkout requires work ID or --next")
			}
			id, err := domain.ParseID(o.args[2])
			if err != nil {
				return result, usage(err.Error())
			}
			result.WorkItemID = id
			dir = filepath.Join(".wos", "checkout", scope.OutcomeID.String(), id.String())
			version, err := strconv.ParseUint(o.values["version"], 10, 64)
			if err != nil {
				return result, usage("specific checkout requires --version")
			}
			name = "acquire_work_contract"
			cmd = application.AcquireWorkContractCommand{Scope: scope, WorkItemID: id, ExpectedWorkItemVersion: domain.Version(version), TTLSeconds: config.Lease.RequestedTTLSeconds}
		}
		unlock, err := w.Lock(dir)
		if err != nil {
			return result, &LocalError{Err: err}
		}
		defer unlock()
		if err = w.ensureNoPending(dir); err != nil {
			return result, err
		}
		raw, receipt, err := w.Mutate(ctx, client, config, scope, dir, name, cmd)
		result.ReceiptPath = receipt
		result.Committed = err == nil
		result.Data = raw
		if err != nil {
			return result, err
		}
		var acquired sdk.CommandResult[application.WorkContractResult]
		if name == "acquire_next_work_contract" {
			var next sdk.CommandResult[application.WorkContractAcquisition]
			if err = json.Unmarshal(raw, &next); err != nil {
				return result, &LocalError{Err: err, Committed: true, ReceiptPath: receipt}
			}
			if !next.Value.Acquired {
				result.RequiresAction = "no_eligible_work"
				return result, &exitError{8, fmt.Errorf("no eligible work in scanned page; inspect search_complete and next_cursor")}
			}
			acquired.Value = *next.Value.Result
			acquired.CommandID = next.CommandID
			acquired.ResultOmitted = next.ResultOmitted
		} else {
			if err = json.Unmarshal(raw, &acquired); err != nil {
				return result, &LocalError{Err: err, Committed: true, ReceiptPath: receipt}
			}
		}
		if acquired.ResultOmitted {
			return result, &LocalError{Err: fmt.Errorf("committed response omitted; recover exact acquisition before executing"), Committed: true, ReceiptPath: receipt}
		}
		result.ContractID = acquired.Value.Contract.ID
		result.WorkItemID = acquired.Value.Contract.WorkItemID
		path, err := w.materialize(config, acquired.Value, acquired.CommandID)
		if err != nil {
			return result, &LocalError{Err: err, Committed: true, ReceiptPath: receipt}
		}
		result.Data = map[string]any{"workspace": path, "contract": acquired.Value.Contract, "recovery": acquired.Value.Recovery}
		return result, nil
	}
	if len(o.args) != 3 {
		return result, usage("work operation requires work ID")
	}
	id, err := domain.ParseID(o.args[2])
	if err != nil {
		return result, usage(err.Error())
	}
	result.WorkItemID = id
	state, dir, err := w.loadState(config, scope, id, o.values["contract"])
	if err != nil {
		return result, err
	}
	result.ContractID = state.Contract.ID
	if operation == "status" {
		view, err := client.GetWorkContract(ctx, scope, state.Contract.ID)
		result.Data = view
		return result, err
	}
	return workWorkspaceCommand(ctx, w, client, config, scope, state, dir, o, result)
}
func (w *Workspace) ensureNoPending(dir string) error {
	path := filepath.Join(dir, "outbox")
	if err := w.check(path); err != nil {
		return &LocalError{Err: err}
	}
	f, err := w.root.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return &LocalError{Err: err}
	}
	defer f.Close()
	entries, err := f.ReadDir(-1)
	if err != nil {
		return &LocalError{Err: err}
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		raw, err := w.Read(filepath.Join(path, entry.Name()))
		if err != nil {
			return &LocalError{Err: err}
		}
		var intent Intent
		if err = json.Unmarshal(raw, &intent); err != nil {
			return &LocalError{Err: err}
		}
		if intent.State == "prepared" || intent.State == "sent_unknown" {
			return &exitError{6, fmt.Errorf("pending intent %s; recover the original payload first", entry.Name())}
		}
	}
	return nil
}
func workWorkspaceCommand(ctx context.Context, w *Workspace, client *sdk.Client, config Config, scope domain.Scope, state State, dir string, o options, result Output) (Output, error) {
	return result, usage("workspace mutation is not yet available")
}
func planningCommand(ctx context.Context, w *Workspace, client *sdk.Client, config Config, scope domain.Scope, o options) (Output, error) {
	return Output{Operation: "planning", OutcomeID: scope.OutcomeID}, usage("planning wrappers are not yet available")
}
