package mcptransport

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerSecurity(server *mcp.Server, options Options) {
	if options.Security == nil {
		return
	}
	mcp.AddTool(server, &mcp.Tool{Name: "wos_list_namespaces", Description: "List authorized namespaces; credential scope is enforced."}, func(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		ctx, cancel, err := requestContext(ctx, req, options)
		defer cancel()
		if err != nil {
			r, e := failure(err)
			return r, nil, e
		}
		value, err := options.Security.Namespaces(ctx)
		if err != nil {
			r, e := failure(err)
			return r, nil, e
		}
		r, e := success(map[string]any{"items": value})
		return r, nil, e
	})
	mcp.AddTool(server, &mcp.Tool{Name: "wos_get_namespace_administration", Description: "Read namespace version, grants, credential metadata and bounded security audit; namespace:admin required."}, func(ctx context.Context, req *mcp.CallToolRequest, q queryArgs) (*mcp.CallToolResult, any, error) {
		ctx, cancel, err := requestContext(ctx, req, options)
		defer cancel()
		if err != nil {
			r, e := failure(err)
			return r, nil, e
		}
		value, err := options.Security.AdministrativeSnapshot(ctx, q.NamespaceID)
		if err != nil {
			r, e := failure(err)
			return r, nil, e
		}
		r, e := success(value)
		return r, nil, e
	})
	type input struct {
		Key     string                           `json:"idempotency_key"`
		Command application.AdministrativeIntent `json:"command"`
	}
	mcp.AddTool(server, &mcp.Tool{Name: "wos_administer_namespace", Description: "Create namespace, set_grant, issue_credential or revoke_credential. Namespace CAS and idempotency required. Issued secret appears once; replay returns receipt with token_omitted. Namespace creation grants only the creator's current permissions."}, func(ctx context.Context, req *mcp.CallToolRequest, in input) (*mcp.CallToolResult, any, error) {
		ctx, cancel, err := requestContext(ctx, req, options)
		defer cancel()
		if err != nil {
			r, e := failure(err)
			return r, nil, e
		}
		value, token, err := options.Security.Administer(ctx, in.Key, in.Command)
		if err != nil {
			r, e := failure(err)
			return r, nil, e
		}
		r, e := success(map[string]any{"result": value, "token": token})
		return r, nil, e
	})
}
