package mcptransport

import (
	"context"
	"reflect"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type workspaceViewArgs struct {
	Scope domain.Scope                   `json:"scope"`
	Query application.WorkspaceViewQuery `json:"query"`
}
type workspaceCardsArgs struct {
	Scope domain.Scope                    `json:"scope"`
	Query application.WorkspaceCardsQuery `json:"query"`
}
type workspaceTimelineArgs struct {
	Scope              domain.Scope      `json:"scope"`
	SubjectRef         *domain.EntityRef `json:"subject_ref,omitempty"`
	CriterionID        *domain.ID        `json:"criterion_id,omitempty"`
	IncludeDescendants bool              `json:"include_descendants,omitempty"`
	Limit              int               `json:"limit,omitempty"`
	Cursor             string            `json:"cursor,omitempty"`
}
type entityNoteArgs struct {
	Scope  domain.Scope `json:"scope"`
	NoteID domain.ID    `json:"note_id"`
}

func registerWorkspaceQueries(server *mcp.Server, service *application.Service, options Options) {
	registerWorkspaceQuery(server, options, "wos_get_workspace_view", "Read one coherent authorized entity, Criterion, Roadmap revision or execution-record workspace. Facet coverage and cursors distinguish incomplete data from absence. Versions are exact decimal strings. Capabilities explain current conditions and never grant authority.", func(ctx context.Context, input workspaceViewArgs) (application.EntityWorkspaceView, error) {
		return service.GetEntityWorkspaceView(ctx, input.Scope, input.Query)
	})
	registerWorkspaceQuery(server, options, "wos_get_workspace_cards", "Read at most 100 compact card summaries at one Outcome revision, using explicit refs OR one structural parent with pagination. Never hydrate an inspector per card. Unknown execution authority remains unknown; completion, assessment and acceptance remain separate.", func(ctx context.Context, input workspaceCardsArgs) (application.WorkspaceCardsPage, error) {
		return service.GetWorkspaceCards(ctx, input.Scope, input.Query)
	})
	registerWorkspaceQuery(server, options, "wos_get_workspace_timeline", "Read at most 20 recent authorized facts with a frozen historical frontier and stable revision/event-index ordering. Subject defaults to the Outcome. Follow next_cursor; descendant origins and documentary observations remain explicit. Reading never completes or approves work.", func(ctx context.Context, input workspaceTimelineArgs) (application.WorkspaceTimelinePage, error) {
		return service.GetWorkspaceTimeline(ctx, input.Scope, application.WorkspaceTimelineQuery{SubjectRef: input.SubjectRef, CriterionID: input.CriterionID, IncludeDescendants: input.IncludeDescendants, Limit: input.Limit, Cursor: input.Cursor})
	})
	registerWorkspaceQuery(server, options, "wos_get_entity_note", "Read a complete authorized immutable documentary note, including actual recording Principal, declared Actor and correction reference. A note is not signed execution activity, a criterion assessment or acceptance.", func(ctx context.Context, input entityNoteArgs) (domain.EntityNote, error) {
		return service.GetEntityNote(ctx, input.Scope, input.NoteID)
	})
}

func registerWorkspaceQuery[I, O any](server *mcp.Server, options Options, name, description string, run func(context.Context, I) (O, error)) {
	inputSchema := commands.SignedSchema(reflect.TypeFor[I]())
	properties := inputSchema["properties"].(map[string]any)
	switch reflect.TypeFor[I]() {
	case reflect.TypeFor[workspaceViewArgs]():
		properties["query"] = commands.WorkspaceReadSchema(reflect.TypeFor[application.WorkspaceViewQuery]())
	case reflect.TypeFor[workspaceCardsArgs]():
		properties["query"] = commands.WorkspaceReadSchema(reflect.TypeFor[application.WorkspaceCardsQuery]())
	case reflect.TypeFor[workspaceTimelineArgs]():
		properties["limit"] = map[string]any{"type": "integer", "minimum": 1, "maximum": application.MaxWorkspaceTimelineLimit, "default": application.MaxWorkspaceTimelineLimit}
	}
	server.AddTool(&mcp.Tool{Name: name, Description: description, InputSchema: inputSchema, OutputSchema: commands.SignedSchema(reflect.TypeFor[O]()), Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, cancel, err := requestContext(ctx, request, options)
		defer cancel()
		if err != nil {
			return failure(err)
		}
		if len(request.Params.Arguments) > MaxPayloadBytes {
			return failure(domain.NewError(domain.ErrorCodeInvalidArgument, "workspace query exceeds 256 KiB"))
		}
		var input I
		if err = decodeStrict(request.Params.Arguments, &input); err != nil {
			return failure(err)
		}
		value, err := run(ctx, input)
		if err != nil {
			return failure(err)
		}
		return success(value)
	})
}
