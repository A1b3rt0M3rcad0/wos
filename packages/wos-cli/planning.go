package woscli

import (
	"context"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
	"path/filepath"
	"strings"
)

func planningCommand(ctx context.Context, w *Workspace, client *sdk.Client, config Config, scope domain.Scope, o options) (Output, error) {
	result := Output{Operation: strings.Join(o.args, " "), OutcomeID: scope.OutcomeID}
	if len(o.args) < 2 {
		return result, usage("planning subcommand required")
	}
	entity, verb := o.args[0], o.args[1]
	base := outcomePath(scope)
	if entity == "protocol" && verb == "get" {
		v, e := client.GetNamespaceWorkProtocol(ctx, scope.NamespaceID)
		result.Data = v
		return result, e
	}
	if verb == "list" {
		switch entity {
		case "outcome":
			page, err := client.SearchOutcomes(ctx, scope.NamespaceID, "", 25, o.values["cursor"])
			result.Data = page
			return result, err
		case "work", "objective":
			kind := "work_item"
			if entity == "objective" {
				kind = "objective"
			}
			page, err := client.GetOutcomeGraph(ctx, scope, application.GraphQuery{Kinds: []string{kind}, Limit: 25, Depth: 1, Cursor: o.values["cursor"]})
			result.Data = page
			return result, err
		case "roadmap":
			raw, err := client.QueryJSON(ctx, base+"/roadmaps")
			result.Data = raw
			return result, err
		case "review":
			id, err := domain.ParseID(o.values["contract"])
			if err != nil {
				return result, usage("review list requires --contract")
			}
			page, err := client.ContractRecords(ctx, scope, id, "submissions", 25, o.values["cursor"])
			result.Data = page
			return result, err
		}
	}
	if verb == "get" {
		if entity == "outcome" {
			if len(o.args) == 3 {
				id, err := domain.ParseID(o.args[2])
				if err != nil {
					return result, usage(err.Error())
				}
				base = outcomePath(domain.Scope{NamespaceID: scope.NamespaceID, OutcomeID: id})
			}
			raw, err := client.QueryJSON(ctx, base)
			result.Data = raw
			return result, err
		}
		if len(o.args) != 3 {
			return result, usage("entity get requires ID")
		}
		id, err := domain.ParseID(o.args[2])
		if err != nil {
			return result, usage(err.Error())
		}
		if entity == "review" {
			v, err := client.GetWorkSubmission(ctx, scope, id)
			result.Data = v
			return result, err
		}
		path := ""
		switch entity {
		case "objective":
			path = "objectives"
		case "work":
			path = "work-items"
		case "roadmap":
			path = "roadmaps"
		}
		if path == "" {
			return result, usage("unknown entity")
		}
		raw, err := client.QueryJSON(ctx, base+"/"+path+"/"+id.String())
		result.Data = raw
		return result, err
	}
	name := ""
	switch entity {
	case "protocol":
		if verb == "set" {
			name = "set_namespace_work_protocol"
		}
		if verb == "reconcile" {
			name = "reconcile_expired_work_contracts"
		}
	case "outcome":
		if verb == "create" {
			name = "create_outcome"
		}
	case "objective":
		if verb == "create" {
			name = "create_objective"
		}
	case "work":
		if verb == "create" {
			name = "create_work_item"
		}
	case "review":
		switch verb {
		case "assess":
			name = "record_criterion_assessment"
		case "attest":
			name = "attest_criterion"
		}
	case "roadmap":
		switch verb {
		case "create":
			name = "create_roadmap"
		case "draft":
			name = "open_roadmap_draft"
		case "replace":
			name = "replace_roadmap_draft"
		case "publish":
			name = "publish_roadmap_draft"
		case "activate":
			name = "activate_roadmap_revision"
		case "deactivate":
			name = "deactivate_roadmap_revision"
		case "discard":
			name = "discard_roadmap_draft"
		}
	}
	if name == "" {
		return result, usage("unknown planning mutation; use documented verb and --file")
	}
	file := o.values["file"]
	if file == "" {
		return result, usage("planning/review mutations require --file with typed public command YAML")
	}
	raw, err := w.Read(file)
	if err != nil {
		return result, &LocalError{Err: err}
	}
	payload, err := YAMLJSON(raw)
	if err != nil {
		return result, usage(err.Error())
	}
	payload, err = commands.NewCatalog(nil, nil).Validate(name, payload)
	if err != nil {
		return result, usage(err.Error())
	}
	var object any
	if err = json.Unmarshal(payload, &object); err != nil {
		return result, usage(err.Error())
	}
	if err = checkPlanningBinding(object, scope, name == "create_outcome"); err != nil {
		return result, err
	}
	dir := filepath.Join(".wos", "planning", scope.OutcomeID.String())
	release, err := w.Lock(dir)
	if err != nil {
		return result, &LocalError{Err: err}
	}
	defer release()
	if err = w.ensureNoPending(dir); err != nil {
		return result, err
	}
	response, receipt, err := w.Mutate(ctx, client, config, scope, dir, name, json.RawMessage(payload))
	result.Data = response
	result.ReceiptPath = receipt
	result.Committed = err == nil
	return result, err
}
func checkPlanningBinding(v any, scope domain.Scope, creatingOutcome bool) error {
	switch x := v.(type) {
	case map[string]any:
		for key, value := range x {
			if key == "namespace_id" && value != scope.NamespaceID.String() {
				return usage("planning Namespace differs from trusted binding")
			}
			if key == "outcome_id" && !creatingOutcome && value != scope.OutcomeID.String() {
				return usage("planning Outcome differs from explicit scope")
			}
			if err := checkPlanningBinding(value, scope, creatingOutcome); err != nil {
				return err
			}
		}
	case []any:
		for _, value := range x {
			if err := checkPlanningBinding(value, scope, creatingOutcome); err != nil {
				return err
			}
		}
	}
	return nil
}
