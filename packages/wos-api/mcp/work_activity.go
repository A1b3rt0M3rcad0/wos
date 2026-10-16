package mcptransport

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type workActivityArgs struct {
	NamespaceID domain.ID `json:"namespace_id"`
	OutcomeID   domain.ID `json:"outcome_id"`
	WorkItemID  domain.ID `json:"work_item_id,omitempty"`
	ContractID  domain.ID `json:"contract_id,omitempty"`
	Limit       int       `json:"limit,omitempty"`
	Cursor      string    `json:"cursor,omitempty"`
	Descending  bool      `json:"descending,omitempty"`
}

type workAttentionArgs struct {
	NamespaceID domain.ID `json:"namespace_id"`
	OutcomeID   domain.ID `json:"outcome_id"`
	WorkItemID  domain.ID `json:"work_item_id"`
}

func registerWorkActivityQueries(server *mcp.Server, service *application.Service, options Options) {
	mcp.AddTool(server, &mcp.Tool{Name: "wos_list_work_activity", Description: "Read authorized immutable work activity in an Outcome. Optional work_item_id and contract_id restrict the history; descending=true reads newest first and its cursor continues toward older records. Limit 1–100, default 25; follow the scope/filter-bound next_cursor. Activity records observations, not completion, criterion assessment, or execution authority."}, func(ctx context.Context, request *mcp.CallToolRequest, query workActivityArgs) (*mcp.CallToolResult, any, error) {
		ctx, cancel, err := requestContext(ctx, request, options)
		defer cancel()
		if err != nil {
			result, e := failure(err)
			return result, nil, e
		}
		value, err := service.ListWorkActivity(ctx, domain.Scope{NamespaceID: query.NamespaceID, OutcomeID: query.OutcomeID}, ports.WorkActivityFilter{WorkItemID: query.WorkItemID, ContractID: query.ContractID, Descending: query.Descending}, query.Limit, query.Cursor)
		if err != nil {
			result, e := failure(err)
			return result, nil, e
		}
		result, e := success(value)
		return result, nil, e
	})
	mcp.AddTool(server, &mcp.Tool{Name: "wos_get_work_attention", Description: "Read the current authorized WorkItem attention projection. Signals explain conditions requiring participant attention; reading does not change lifecycle, renew authority, or complete work."}, func(ctx context.Context, request *mcp.CallToolRequest, query workAttentionArgs) (*mcp.CallToolResult, any, error) {
		ctx, cancel, err := requestContext(ctx, request, options)
		defer cancel()
		if err != nil {
			result, e := failure(err)
			return result, nil, e
		}
		value, err := service.GetWorkAttention(ctx, domain.Scope{NamespaceID: query.NamespaceID, OutcomeID: query.OutcomeID}, query.WorkItemID)
		if err != nil {
			result, e := failure(err)
			return result, nil, e
		}
		result, e := success(value)
		return result, nil, e
	})
}
