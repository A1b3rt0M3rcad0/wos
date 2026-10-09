package woscli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type Output struct {
	Operation      string    `json:"operation"`
	OutcomeID      domain.ID `json:"outcome_id,omitempty"`
	WorkItemID     domain.ID `json:"work_item_id,omitempty"`
	ContractID     domain.ID `json:"contract_id,omitempty"`
	Committed      bool      `json:"committed"`
	RequiresAction string    `json:"requires_action,omitempty"`
	ReceiptPath    string    `json:"receipt_path,omitempty"`
	Data           any       `json:"data,omitempty"`
	Error          string    `json:"error,omitempty"`
}
type exitError struct {
	code int
	err  error
}

func (e *exitError) Error() string { return e.err.Error() }
func (e *exitError) Unwrap() error { return e.err }
func usage(message string) error   { return &exitError{2, fmt.Errorf("%s", message)} }
func exitCode(err error) int {
	if err == nil {
		return 0
	}
	var local *LocalError
	if errors.As(err, &local) {
		return 9
	}
	var explicit *exitError
	if errors.As(err, &explicit) {
		return explicit.code
	}
	var remote *sdk.Error
	if errors.As(err, &remote) {
		switch remote.Code {
		case "transport_redirect", "transaction_conflict":
			return 6
		case "forbidden", "unauthorized", "invalid_credential":
			return 3
		case "version_conflict", "idempotency_conflict", "contract_spec_mismatch", "work_already_claimed":
			return 4
		case "contract_expired", "contract_revoked", "stale_execution", "invalid_fencing_token", "lease_expired":
			return 5
		case "submission_not_accepted", "criterion_not_satisfied", "criterion", "assessment", "criterion_error", "assessment_error":
			return 7
		case "invalid_argument", "invalid_id", "invalid_scope", "invalid_configuration", "invalid_config", "contract_protocol_required":
			return 2
		}
		if remote.Status == 401 || remote.Status == 403 {
			return 3
		}
		if remote.Status >= 500 {
			return 6
		}
		return 4
	}
	return 6
}

type options struct {
	values map[string]string
	args   []string
}

func parseOptions(args []string) (options, error) {
	o := options{values: map[string]string{}}
	booleans := map[string]bool{"signing-key-stdin": true, "token-stdin": true, "generate-signing-key": true, "all-pending": true, "all-active": true, "once": true, "for-agent": true, "next": true, "dry-run": true, "takeover": true, "foreground": true, "break-lock": true, "writers-drained": true}
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "--") {
			o.args = append(o.args, arg)
			continue
		}
		key, value, has := strings.Cut(strings.TrimPrefix(arg, "--"), "=")
		allowed := map[string]bool{"destination": true, "to": true, "signing-key-stdin": true, "workspace-schema": true, "token-stdin": true, "generate-signing-key": true, "credential-ref": true, "private-key-ref": true, "profile": true, "server-id": true, "issuer-fingerprint": true, "enrollment": true, "name": true, "workspace": true, "output": true, "server": true, "namespace": true, "outcome": true, "credential-env": true, "version": true, "all-pending": true, "all-active": true, "once": true, "for-agent": true, "next": true, "dry-run": true, "takeover": true, "foreground": true, "break-lock": true, "limit": true, "cursor": true, "work": true, "contract": true, "reason": true, "submission": true, "file": true, "ttl": true, "interval": true, "count": true, "previous-review": true, "command": true, "completion": true, "phase": true, "writers-drained": true}
		if !allowed[key] {
			return o, usage("unknown flag --" + key)
		}
		if key == "" {
			return o, usage("invalid flag")
		}
		if !has {
			if booleans[key] {
				value = "true"
			} else {
				if i+1 >= len(args) || strings.HasPrefix(args[i+1], "--") {
					return o, usage("flag --" + key + " requires a value")
				}
				i++
				value = args[i]
			}
		}
		if _, exists := o.values[key]; exists {
			return o, usage("duplicate flag --" + key)
		}
		o.values[key] = value
	}
	if format := o.values["output"]; format != "" && format != "json" && format != "text" {
		return o, usage("output must be text/json")
	}
	return o, nil
}
func Run(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 1 && args[0] == "--version" {
		args = []string{"version"}
	}
	o, err := parseOptions(args)
	result := Output{}
	if err == nil {
		if o.values["break-lock"] == "true" && (len(o.args) < 2 || o.args[0] != "work" || o.args[1] != "recover") {
			err = usage("--break-lock is only allowed with explicit work recover")
		}
		if len(o.args) == 0 {
			err = usage("usage: wosctl init|auth status|capabilities|work|contract|outcome|objective|roadmap|review")
		} else {
			result.Operation = strings.Join(o.args, " ")
			if err == nil {
				result, err = run(ctx, o)
			}
		}
	}
	code := exitCode(err)
	if err != nil {
		result.Error = err.Error()
		var local *LocalError
		if errors.As(err, &local) {
			result.Committed = local.Committed
			result.ReceiptPath = local.ReceiptPath
		}
		fmt.Fprintln(stderr, result.Error)
	}
	if o.values["output"] == "json" {
		if encodeError := json.NewEncoder(stdout).Encode(result); encodeError != nil {
			fmt.Fprintln(stderr, "cannot write CLI result:", encodeError)
			return 6
		}
	} else {
		if result.Data != nil {
			data, encodeError := json.MarshalIndent(result.Data, "", "  ")
			if encodeError != nil {
				fmt.Fprintln(stderr, "cannot encode CLI result:", encodeError)
				return 6
			}
			fmt.Fprintln(stdout, string(data))
		}
		if result.ReceiptPath != "" {
			fmt.Fprintln(stdout, "Receipt:", result.ReceiptPath)
		}
	}
	return code
}
func loadClient(w *Workspace) (Config, *sdk.Client, error) {
	var config Config
	path, err := w.DocumentPath(".wos/config")
	if err != nil {
		return config, nil, usage(err.Error())
	}
	if err = w.ReadDocument(path, &config); err != nil {
		return config, nil, usage(err.Error())
	}
	if err = config.Validate(); err != nil {
		return config, nil, usage(err.Error())
	}
	raw, err := w.Read(".wos/trust.json")
	if err != nil {
		return config, nil, usage("workspace trust binding absent; run init")
	}
	var binding Destination
	if err = json.Unmarshal(raw, &binding); err != nil {
		return config, nil, usage("invalid trust binding")
	}
	if binding != config.Destination() {
		return config, nil, usage("configuration destination changed; restore trusted server/namespace/credential reference")
	}
	token := os.Getenv(strings.TrimPrefix(config.Connection.CredentialRef, "env:"))
	client, err := sdk.New(config.Connection.ServerURL, token, nil)
	return config, client, err
}
func scopeFor(config Config, o options) (domain.Scope, error) {
	scope := config.DefaultScope()
	if v := o.values["outcome"]; v != "" {
		id, err := domain.ParseID(v)
		if err != nil {
			return scope, usage(err.Error())
		}
		scope.OutcomeID = id
	}
	return scope, nil
}
func outcomePath(scope domain.Scope) string {
	return "/namespaces/" + scope.NamespaceID.String() + "/outcomes/" + scope.OutcomeID.String()
}
func run(ctx context.Context, o options) (Output, error) {
	result := Output{Operation: strings.Join(o.args, " ")}
	if len(o.args) == 1 && o.args[0] == "version" {
		result.Data = map[string]any{"version": Version, "commit": Commit, "built_at": BuiltAt, "work_protocol": "contracts_v1", "workspace_schema": 1}
		return result, nil
	}
	root := o.values["workspace"]
	if root == "" {
		root = os.Getenv("WOS_WORKSPACE")
		if root == "" {
			root = "."
		}
	}
	w, err := OpenWorkspace(root)
	if err != nil {
		return result, &LocalError{Err: err}
	}
	defer w.Close()
	if o.args[0] == "workspace" {
		return workspaceMigrationCommandV2(ctx, w, o)
	}
	if o.args[0] == "project" {
		return projectInitializeV2(w, o)
	}
	if o.args[0] == "profile" {
		return profileCommandV2(ctx, w, o, defaultSecretResolverV2(w))
	}
	if o.args[0] == "init" && o.values["workspace-schema"] == "2" {
		o.args = []string{"project", "init"}
		return projectInitializeV2(w, o)
	}
	if o.args[0] == "init" && o.values["workspace-schema"] != "" && o.values["workspace-schema"] != "1" {
		return result, usage("workspace-schema must be 1 or 2")
	}
	if o.args[0] == "init" {
		return initialize(ctx, w, o)
	}
	_, projectYAML := w.root.Stat(".wos/project.yaml")
	_, projectYML := w.root.Stat(".wos/project.yml")
	if projectYAML == nil || projectYML == nil || o.values["profile"] != "" || os.Getenv("WOS_PROFILE") != "" {
		return runWorkspaceV2(ctx, w, o)
	}
	config, client, err := loadClient(w)
	if err != nil {
		return result, err
	}
	if o.values["output"] == "" {
		o.values["output"] = config.Output.DefaultFormat
	}
	scope, err := scopeFor(config, o)
	if err != nil {
		return result, err
	}
	result.OutcomeID = scope.OutcomeID
	switch o.args[0] {
	case "capabilities":
		v, err := client.Capabilities(ctx)
		result.Data = v
		return result, err
	case "auth":
		if len(o.args) != 2 || o.args[1] != "status" {
			return result, usage("use auth status")
		}
		raw, err := client.QueryJSON(ctx, "/identity")
		result.Data = raw
		return result, err
	case "work":
		return workCommand(ctx, w, client, config, scope, o)
	case "contract":
		if len(o.args) < 2 {
			return result, usage("contract list|get|history|revoke")
		}
		switch o.args[1] {
		case "list", "history":
			limit := 25
			if value := o.values["limit"]; value != "" {
				limit, err = strconv.Atoi(value)
				if err != nil {
					return result, usage("invalid limit")
				}
			}
			f := ports.ContractFilter{Limit: limit}
			if v := o.values["work"]; v != "" {
				f.WorkItemID, err = domain.ParseID(v)
				if err != nil {
					return result, usage(err.Error())
				}
			}
			v, err := client.WorkContractHistory(ctx, scope, f, o.values["cursor"])
			result.Data = v
			return result, err
		case "get", "revoke":
			if len(o.args) != 3 {
				return result, usage("contract ID required")
			}
			id, err := domain.ParseID(o.args[2])
			if err != nil {
				return result, usage(err.Error())
			}
			result.ContractID = id
			if o.args[1] == "get" {
				v, err := client.GetWorkContract(ctx, scope, id)
				result.Data = v
				return result, err
			}
			reason := o.values["reason"]
			version, err := strconv.ParseUint(o.values["version"], 10, 64)
			if err != nil || reason == "" {
				return result, usage("revocation requires --reason and --version")
			}
			unlock, err := w.Lock(filepath.Join(".wos", "admin", id.String()))
			if err != nil {
				return result, &LocalError{Err: err}
			}
			defer unlock()
			raw, receipt, err := w.Mutate(ctx, client, config, scope, filepath.Join(".wos", "admin", id.String()), "revoke_work_contract", application.RevokeWorkContractCommand{Scope: scope, ContractID: id, ExpectedContractVersion: domain.Version(version), Reason: reason})
			result.Data = raw
			result.ReceiptPath = receipt
			result.Committed = err == nil
			return result, err
		}
	case "outcome", "objective", "roadmap", "review", "protocol":
		return planningCommand(ctx, w, client, config, scope, o)
	}
	return result, usage("unknown command")
}
func initialize(ctx context.Context, w *Workspace, o options) (Output, error) {
	result := Output{Operation: "init"}
	ns, err := domain.ParseID(o.values["namespace"])
	if err != nil {
		return result, usage("init requires --namespace UUIDv7")
	}
	outcome, err := domain.ParseID(o.values["outcome"])
	if err != nil {
		return result, usage("init requires --outcome UUIDv7")
	}
	variable := o.values["credential-env"]
	if variable == "" {
		variable = "WOS_TOKEN"
	}
	config := Config{SchemaVersion: 1, Kind: "WOSWorkspace", Profile: "default", Connection: ConnectionConfig{o.values["server"], "env:" + variable}, Scope: WorkspaceScope{ns, outcome}, Workspace: WorkspaceConfig{".wos/work", "yaml"}, Lease: LeaseConfig{300}, Output: OutputConfig{"json"}}
	if err = config.Validate(); err != nil {
		return result, usage(err.Error())
	}
	if _, err = w.Read(".wos/trust.json"); err == nil {
		return result, usage("workspace already initialized; existing trust and pending intents are preserved")
	}
	client, err := sdk.New(config.Connection.ServerURL, os.Getenv(variable), nil)
	if err != nil {
		return result, usage(err.Error())
	}
	capabilities, err := client.Capabilities(ctx)
	if err != nil {
		return result, err
	}
	if len(capabilities.YAMLVersions) == 0 || capabilities.YAMLVersions[0] != 1 {
		return result, usage("server lacks YAML schema 1")
	}
	if _, err = client.QueryJSON(ctx, outcomePath(config.DefaultScope())); err != nil {
		return result, err
	}
	if err = w.WriteDocument(".wos/config.yaml", config); err != nil {
		return result, &LocalError{Err: err}
	}
	if err = w.saveJSON(".wos/trust.json", config.Destination()); err != nil {
		return result, &LocalError{Err: err}
	}
	if err = w.AtomicWrite(".wos/.gitignore", []byte("work/\ncheckout/\nplanning/\nadmin/\ntrust.json\n")); err != nil {
		return result, &LocalError{Err: err}
	}
	result.Data = map[string]any{"server_url": config.Connection.ServerURL, "namespace_id": ns, "outcome_id": outcome, "credential_ref": config.Connection.CredentialRef}
	return result, nil
}
