package mcptransport

import (
	"context"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerDeliveryHealth(server *mcp.Server, service *a.Service, options Options) {
	mcp.AddTool(server, &mcp.Tool{Name: "wos_delivery_health", Description: "Read scoped durable webhook queue counts/ages and this process worker status. Requires current Namespace administration; optional Outcome restriction is enforced. No event body, receiver URL or secret is returned. API readiness is independent; no delivery/receiver SLA is certified."}, func(ctx context.Context, request *mcp.CallToolRequest, q a.DeliveryHealthQuery) (*mcp.CallToolResult, any, error) {
		ctx, cancel, e := requestContext(ctx, request, options)
		defer cancel()
		if e != nil {
			result, e := failure(e)
			return result, nil, e
		}
		value, e := service.GetDeliveryHealth(ctx, q)
		if e != nil {
			result, e := failure(e)
			return result, nil, e
		}
		result, e := success(value)
		return result, nil, e
	})
}
