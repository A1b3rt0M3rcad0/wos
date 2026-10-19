package woscli

import (
	"context"
	"strings"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	sdk "github.com/A1b3rt0M3rcad0/wos/packages/wos-sdk-go"
)

func workspaceInspectCommand(ctx context.Context, client *sdk.Client, scope domain.Scope, o options) (Output, error) {
	result := Output{Operation: strings.Join(o.args, " "), OutcomeID: scope.OutcomeID}
	if len(o.args) < 2 {
		return result, usage("inspect get|cards|timeline --file FILE | inspect note NOTE_ID")
	}
	if err := scope.Validate(); err != nil {
		return result, usage(err.Error())
	}
	if o.args[1] == "note" {
		if len(o.args) != 3 {
			return result, usage("inspect note NOTE_ID")
		}
		id, err := domain.ParseID(o.args[2])
		if err != nil {
			return result, usage(err.Error())
		}
		value, err := client.GetEntityNote(ctx, scope, id)
		result.Data = value
		return result, err
	}
	if len(o.args) != 2 {
		return result, usage("inspect get|cards|timeline --file FILE")
	}
	raw, err := workspaceInputFile(o.values["file"])
	if err != nil {
		return result, err
	}
	switch o.args[1] {
	case "get":
		var query application.WorkspaceViewQuery
		if err = commands.Decode(raw, &query); err != nil {
			return result, usage(err.Error())
		}
		if err = validateWorkspaceReadScope(scope, query.Subject.Ref); err != nil {
			return result, usage(err.Error())
		}
		result.Data, err = client.GetEntityWorkspaceView(ctx, scope, query)
	case "cards":
		var query application.WorkspaceCardsQuery
		if err = commands.Decode(raw, &query); err != nil {
			return result, usage(err.Error())
		}
		for _, ref := range query.Refs {
			if err = validateWorkspaceReadScope(scope, ref); err != nil {
				return result, usage(err.Error())
			}
		}
		if query.ParentRef != nil {
			if err = validateWorkspaceReadScope(scope, *query.ParentRef); err != nil {
				return result, usage(err.Error())
			}
		}
		result.Data, err = client.GetWorkspaceCards(ctx, scope, query)
	case "timeline":
		var input struct {
			SubjectRef         *domain.EntityRef `json:"subject_ref,omitempty"`
			CriterionID        *domain.ID        `json:"criterion_id,omitempty"`
			IncludeDescendants bool              `json:"include_descendants,omitempty"`
			Limit              int               `json:"limit,omitempty"`
			Cursor             string            `json:"cursor,omitempty"`
		}
		if err = commands.Decode(raw, &input); err != nil {
			return result, usage(err.Error())
		}
		if input.SubjectRef != nil {
			if err = validateWorkspaceReadScope(scope, *input.SubjectRef); err != nil {
				return result, usage(err.Error())
			}
		}
		result.Data, err = client.GetWorkspaceTimeline(ctx, scope, application.WorkspaceTimelineQuery{SubjectRef: input.SubjectRef, CriterionID: input.CriterionID, IncludeDescendants: input.IncludeDescendants, Limit: input.Limit, Cursor: input.Cursor})
	default:
		return result, usage("inspect get|cards|timeline --file FILE | inspect note NOTE_ID")
	}
	return result, err
}
