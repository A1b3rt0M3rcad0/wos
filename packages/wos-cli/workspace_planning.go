package woscli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

// This allowlist is deliberately planning/documentary-only. Signed execution,
// acquisition, review and administrative override retain their own channels.
var workspacePlanningCommands = map[string]bool{
	"create_outcome": true, "create_objective": true, "create_work_item": true,
	"create_plan_items": true, "set_work_item_execution_spec": true,
	"set_work_item_not_before": true, "set_work_item_objective": true,
	"set_objective_parent": true, "set_outcome_owners": true,
	"set_objective_owners": true, "set_work_item_assignees": true,
	"append_entity_note": true, "register_evidence_with_links": true,
	"register_artifact_in_context": true, "propose_decision_in_context": true,
}

// A user-selected input file is separate from the private installation store.
// It never selects the destination, authenticated identity or idempotency key.
func workspaceInputFile(path string) ([]byte, error) {
	if path == "" {
		return nil, usage("--file must identify a typed JSON or YAML input")
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, usage(err.Error())
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() {
		return nil, usage("input must be a regular file")
	}
	raw, err := io.ReadAll(io.LimitReader(file, commands.MaxPayloadBytes+1))
	if err != nil {
		return nil, usage(err.Error())
	}
	if len(raw) > commands.MaxPayloadBytes {
		return nil, usage("input exceeds the 256 KiB payload limit")
	}
	payload, err := YAMLJSON(raw)
	if err != nil {
		return nil, usage(err.Error())
	}
	if len(payload) > commands.MaxPayloadBytes {
		return nil, usage("normalized input exceeds the 256 KiB payload limit")
	}
	return payload, nil
}

func workspacePlanningCommand(ctx context.Context, w *Workspace, client *sdk.Client, config Config, scope domain.Scope, credential domain.ID, o options) (Output, error) {
	result := Output{Operation: strings.Join(o.args, " "), OutcomeID: scope.OutcomeID}
	if len(o.args) != 2 || (o.args[1] != "apply" && o.args[1] != "recover") {
		return result, usage("planning apply --command NAME --file FILE | planning recover; specify --outcome")
	}
	if err := scope.Validate(); err != nil {
		return result, usage(err.Error())
	}
	identity := "legacy"
	releaseProfile := func() {}
	if !credential.IsZero() {
		identity = credential.String()
		var err error
		releaseProfile, err = w.LockV2(ctx, config.Profile, "")
		if err != nil {
			return result, err
		}
	}
	defer func() { releaseProfile() }()
	unlockProfile := func() { releaseProfile(); releaseProfile = func() {} }
	dir := filepath.Join(".wos", "workspace-planning", identity, scope.OutcomeID.String())
	release, err := w.Lock(dir)
	if err != nil {
		return result, &LocalError{Err: err}
	}
	defer release()
	if o.args[1] == "recover" {
		paths, err := w.intentPaths(dir)
		if err != nil {
			return result, &LocalError{Err: err}
		}
		items := []any{}
		for _, path := range paths {
			raw, err := w.Read(path)
			if err != nil {
				return result, &LocalError{Err: err}
			}
			var intent Intent
			if err = json.Unmarshal(raw, &intent); err != nil {
				return result, &LocalError{Err: err}
			}
			if !workspacePlanningCommands[intent.Command] || intent.Scope != scope {
				return result, usage("planning journal command or scope differs; preserve the original intention")
			}
			if intent.State == "rejected" {
				items = append(items, map[string]any{"idempotency_key": intent.IdempotencyKey, "state": intent.State})
				continue
			}
			if _, err = validateWorkspacePlanningPayload(intent.Command, []byte(intent.Payload), scope); err != nil {
				return result, err
			}
			unlockProfile()
			response, receipt, err := w.Replay(ctx, client, config, path)
			result.ReceiptPath = receipt
			if err != nil {
				result.Data = map[string]any{"items": items}
				result.RequiresAction = "Recover the original planning intention with the same profile and Outcome."
				return result, err
			}
			result.Committed = true
			items = append(items, map[string]any{"command": intent.Command, "response": response, "receipt_path": receipt})
		}
		result.Data = map[string]any{"items": items}
		return result, nil
	}
	name := o.values["command"]
	if !workspacePlanningCommands[name] {
		return result, usage("command is not a supported planning/documentary operation; execution and review require their authorized channels")
	}
	if err = w.ensureWorkspacePlanningSettled(dir); err != nil {
		result.RequiresAction = "Run planning recover before creating another intention."
		return result, err
	}
	raw, err := workspaceInputFile(o.values["file"])
	if err != nil {
		return result, err
	}
	payload, err := validateWorkspacePlanningPayload(name, raw, scope)
	if err != nil {
		return result, err
	}
	path, _, err := w.Prepare(config, scope, dir, name, json.RawMessage(payload))
	if err != nil {
		return result, err
	}
	// Profile removal now observes the durable pending intention. Network I/O
	// never holds the profile lock; the journal lock protects its exact replay.
	unlockProfile()
	response, receipt, err := w.Replay(ctx, client, config, path)
	result.Data, result.ReceiptPath, result.Committed = response, receipt, err == nil
	if err != nil {
		result.RequiresAction = "Run planning recover to reconcile the original frozen payload before creating another intention."
	}
	return result, err
}

func (w *Workspace) ensureNoWorkspacePlanningPending(credential domain.ID) error {
	dir := filepath.Join(".wos", "workspace-planning", credential.String())
	entries, err := w.readDirectoryV2(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() || domain.ID(entry.Name()).Validate() != nil {
			return fmt.Errorf("unrecognized planning journal content preserved")
		}
		if err = w.ensureWorkspacePlanningSettled(filepath.Join(dir, entry.Name())); err != nil {
			return fmt.Errorf("planning intentions must be reconciled before profile removal: %w", err)
		}
	}
	return nil
}

func (w *Workspace) ensureWorkspacePlanningSettled(dir string) error {
	paths, err := w.intentPaths(dir)
	if err != nil {
		return err
	}
	for _, path := range paths {
		raw, err := w.Read(path)
		if err != nil {
			return err
		}
		var intent Intent
		if err = json.Unmarshal(raw, &intent); err != nil {
			return err
		}
		if !workspacePlanningCommands[intent.Command] {
			return fmt.Errorf("unrecognized planning command preserved")
		}
		switch intent.State {
		case "confirmed", "rejected":
		case "prepared", "sent_unknown":
			return fmt.Errorf("pending planning intention; recover the original payload first")
		default:
			return fmt.Errorf("unknown planning intention state; preserve and inspect the original journal")
		}
	}
	return nil
}

func validateWorkspacePlanningPayload(name string, raw []byte, scope domain.Scope) ([]byte, error) {
	payload, err := commands.NewCatalog(nil, nil).Validate(name, raw)
	if err != nil {
		return nil, usage(err.Error())
	}
	var object any
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.UseNumber()
	if err = decoder.Decode(&object); err != nil {
		return nil, usage(err.Error())
	}
	if err = checkPlanningBinding(object, scope, name == "create_outcome"); err != nil {
		return nil, err
	}
	return payload, nil
}

func workspaceProfileConfig(profile ProfileV2, scope domain.Scope) Config {
	return Config{
		SchemaVersion: 1, Kind: "WOSWorkspace", Profile: profile.Name,
		Connection: ConnectionConfig{ServerURL: profile.Binding.ServerOrigin, CredentialRef: profile.Authentication.CredentialRef},
		Scope:      WorkspaceScope{NamespaceID: scope.NamespaceID, DefaultOutcomeID: scope.OutcomeID},
	}
}

func validateWorkspaceReadScope(scope domain.Scope, ref domain.EntityRef) error {
	if ref.Scope != scope {
		return fmt.Errorf("workspace subject differs from the selected profile and Outcome")
	}
	return nil
}
