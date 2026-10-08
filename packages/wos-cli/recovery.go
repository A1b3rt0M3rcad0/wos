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
	"sort"
)

func (w *Workspace) intentPaths(dir string) ([]string, error) {
	path := filepath.Join(dir, "outbox")
	if err := w.check(path); err != nil {
		return nil, err
	}
	f, err := w.root.Open(path)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, err
	}
	defer f.Close()
	entries, err := f.ReadDir(-1)
	if err != nil {
		return nil, err
	}
	paths := []string{}
	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".json" {
			paths = append(paths, filepath.Join(path, entry.Name()))
		}
	}
	sort.Strings(paths)
	return paths, nil
}
func recoveryResult(ctx context.Context, client *sdk.Client, config Config, raw json.RawMessage) (application.WorkContractResult, domain.ID, error) {
	var response sdk.CommandResult[application.WorkContractResult]
	if err := json.Unmarshal(raw, &response); err != nil {
		return application.WorkContractResult{}, "", err
	}
	if response.ResultOmitted {
		var thin struct {
			ContractID domain.ID    `json:"contract_id"`
			WorkItemID domain.ID    `json:"work_item_id"`
			Scope      domain.Scope `json:"scope"`
		}
		if err := json.Unmarshal(raw, &thin); err != nil {
			return response.Value, response.CommandID, err
		}
		if err := thin.ContractID.Validate(); err != nil {
			return response.Value, response.CommandID, fmt.Errorf("receipt lacks recoverable contract identity")
		}
		if thin.Scope.NamespaceID != config.Scope.NamespaceID {
			return response.Value, response.CommandID, fmt.Errorf("receipt namespace mismatch")
		}
		view, err := client.GetWorkContract(ctx, thin.Scope, thin.ContractID)
		if err != nil {
			return response.Value, response.CommandID, err
		}
		work, err := client.GetWorkContext(ctx, thin.Scope, thin.WorkItemID)
		if err != nil {
			return response.Value, response.CommandID, err
		}
		response.Value = application.WorkContractResult{Contract: view.Value.Contract, WorkItem: work.Work, EvaluatedAt: view.Value.EvaluatedAt}
	}
	if response.Value.Contract.ID.IsZero() {
		var next sdk.CommandResult[application.WorkContractAcquisition]
		if err := json.Unmarshal(raw, &next); err != nil {
			return response.Value, response.CommandID, err
		}
		if !next.Value.Acquired || next.Value.Result == nil {
			return response.Value, response.CommandID, &exitError{8, fmt.Errorf("original acquisition did not acquire work")}
		}
		response.Value = *next.Value.Result
		response.CommandID = next.CommandID
	}
	if err := response.Value.Contract.Validate(); err != nil {
		return response.Value, response.CommandID, err
	}
	return response.Value, response.CommandID, nil
}
func recoverAcquisition(ctx context.Context, w *Workspace, client *sdk.Client, config Config, scope domain.Scope, work domain.ID, next bool, o options, result Output) (Output, error) {
	stem := work.String()
	if next {
		stem = "next"
	}
	dir := filepath.Join(".wos", "checkout", scope.OutcomeID.String(), stem)
	if o.values["break-lock"] == "true" {
		if err := w.check(filepath.Join(dir, "workspace.lock")); err != nil {
			return result, &LocalError{Err: err}
		}
		if err := w.root.Remove(filepath.Join(dir, "workspace.lock")); err != nil && !os.IsNotExist(err) {
			return result, &LocalError{Err: err}
		}
	}
	unlock, err := w.Lock(dir)
	if err != nil {
		return result, &LocalError{Err: err}
	}
	defer unlock()
	paths, err := w.intentPaths(dir)
	if err != nil {
		return result, &LocalError{Err: err}
	}
	if len(paths) == 0 {
		return result, usage("no acquisition journal exists; recovery never acquires new work")
	}
	type acquisitionIntent struct {
		path   string
		intent Intent
	}
	ordered := []acquisitionIntent{}
	for _, path := range paths {
		raw, e := w.Read(path)
		if e != nil {
			return result, &LocalError{Err: e}
		}
		var intent Intent
		if e = json.Unmarshal(raw, &intent); e != nil {
			return result, &LocalError{Err: e}
		}
		ordered = append(ordered, acquisitionIntent{path, intent})
	}
	sort.SliceStable(ordered, func(i, j int) bool { return ordered[i].intent.PreparedAt.Before(ordered[j].intent.PreparedAt) })
	paths = paths[:0]
	for _, item := range ordered {
		paths = append(paths, item.path)
	}
	recovered := []any{}
	for _, path := range paths {
		data, err := w.Read(path)
		if err != nil {
			return result, &LocalError{Err: err}
		}
		var intent Intent
		if err = json.Unmarshal(data, &intent); err != nil {
			return result, &LocalError{Err: err}
		}
		if intent.State == "rejected" {
			continue
		}
		raw, receipt, err := w.Replay(ctx, client, config, path)
		result.ReceiptPath = receipt
		if err != nil {
			return result, err
		}
		acquired, commandID, err := recoveryResult(ctx, client, config, raw)
		if exitCode(err) == 8 {
			continue
		}
		if err != nil {
			return result, &LocalError{Err: err, Committed: true, ReceiptPath: receipt}
		}
		if !next && acquired.Contract.WorkItemID != work {
			return result, usage("acquisition journal work binding differs")
		}
		destination := stateDir(acquired.Contract.WorkItemID, acquired.Contract.ID)
		release, err := w.Lock(destination)
		if err != nil {
			return result, &LocalError{Err: err, Committed: true, ReceiptPath: receipt}
		}
		encoded, encodeErr := json.Marshal(sdk.CommandResult[application.WorkContractResult]{Value: acquired, CommandID: commandID})
		var writeErr error
		if encodeErr != nil {
			writeErr = encodeErr
		} else {
			_, writeErr = w.applyResult(config, destination, encoded)
		}
		release()
		if writeErr != nil {
			return result, &LocalError{Err: writeErr, Committed: true, ReceiptPath: receipt}
		}
		result.Committed = true
		result.ContractID = acquired.Contract.ID
		result.WorkItemID = acquired.Contract.WorkItemID
		recovered = append(recovered, map[string]any{"contract_id": acquired.Contract.ID, "work_item_id": acquired.Contract.WorkItemID, "workspace": destination})
	}
	result.Data = recovered
	result.RequiresAction = "verify_current_authority_before_execution"
	return result, nil
}
func recoverContract(ctx context.Context, w *Workspace, client *sdk.Client, config Config, state State, dir string, result Output) (Output, error) {
	paths, err := w.intentPaths(dir)
	if err != nil {
		return result, &LocalError{Err: err}
	}
	recovered := 0
	// Order by prepare time, not random idempotency key; old receipts must never
	// regress a later contract/lease/work observation.
	type pending struct {
		path   string
		intent Intent
	}
	items := []pending{}
	for _, path := range paths {
		raw, err := w.Read(path)
		if err != nil {
			return result, &LocalError{Err: err}
		}
		var intent Intent
		if err = json.Unmarshal(raw, &intent); err != nil {
			return result, &LocalError{Err: err}
		}
		if intent.State != "rejected" {
			items = append(items, pending{path, intent})
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].intent.PreparedAt.Before(items[j].intent.PreparedAt) })
	for _, item := range items {
		raw, receipt, err := w.Replay(ctx, client, config, item.path)
		result.ReceiptPath = receipt
		if err != nil {
			return result, err
		}
		next, err := w.applyResult(config, dir, raw)
		if err != nil {
			return result, &LocalError{Err: err, Committed: true, ReceiptPath: receipt}
		}
		state = next
		if item.intent.Command == "sync_work_contract" {
			cmd, err := decodeSyncPayload(item.intent.Payload)
			if err != nil {
				return result, &LocalError{Err: err}
			}
			if err = recordLocalDigests(w, dir, &state, cmd); err != nil {
				return result, &LocalError{Err: err, Committed: true}
			}
		}
		result.Committed = true
		recovered++
	}
	view, err := client.GetWorkContract(ctx, state.Scope, state.Contract.ID)
	if err != nil {
		return result, err
	}
	result.Data = map[string]any{"recovered_intents": recovered, "server": view.Value, "local_drafts_preserved": true}
	if view.Value.EffectiveStatus != domain.ContractActive {
		result.RequiresAction = "authority_ended"
	}
	return result, nil
}
