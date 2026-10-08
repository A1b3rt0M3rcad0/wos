package mcptransport

import (
	"context"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerSignedQueries(server *mcp.Server, service *a.Service, options Options) {
	mcp.AddTool(server, &mcp.Tool{Name: "wos_read_signed_state", Description: "Read one authorized signed-v2 resource: execution, review, specification, authority, case, correction, fact, submission, receipt, cases or review_queue. Exact Outcome scope required. Metadata is compact; proof/material expansions are digest-bound. Receipt lookup is only for the authenticated Principal. Collections scan at most limit (1–100, default 25); follow next_cursor. Queue expiry is read-only and never grants authority."}, func(ctx context.Context, request *mcp.CallToolRequest, query a.SignedStateQuery) (*mcp.CallToolResult, any, error) {
		ctx, cancel, err := requestContext(ctx, request, options)
		defer cancel()
		if err != nil {
			result, e := failure(err)
			return result, nil, e
		}
		value, err := service.ReadSignedState(ctx, query)
		if err != nil {
			result, e := failure(err)
			return result, nil, e
		}
		result, e := success(value)
		return result, nil, e
	})
	mcp.AddTool(server, &mcp.Tool{Name: "wos_get_signed_trust", Description: "Read persistent public server identity and Namespace protocol metadata; no signing secret or another Outcome's state is returned."}, func(ctx context.Context, request *mcp.CallToolRequest, query queryArgs) (*mcp.CallToolResult, any, error) {
		ctx, cancel, err := requestContext(ctx, request, options)
		defer cancel()
		if err != nil {
			result, e := failure(err)
			return result, nil, e
		}
		value, err := service.ReadSignedState(ctx, a.SignedStateQuery{Scope: domain.Scope{NamespaceID: query.NamespaceID}, Resource: "trust"})
		if err != nil {
			result, e := failure(err)
			return result, nil, e
		}
		result, e := success(value)
		return result, nil, e
	})
}
