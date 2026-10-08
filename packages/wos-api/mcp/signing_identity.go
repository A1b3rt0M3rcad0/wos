package mcptransport

import (
	"context"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"reflect"
)

func registerSigningIdentity(server *mcp.Server, options Options) {
	schema := map[string]any{"type": "object", "additionalProperties": false, "required": []string{"idempotency_key", "command"}, "properties": map[string]any{"idempotency_key": map[string]any{"type": "string", "minLength": 16, "maxLength": 128}, "command": typeSchema(reflect.TypeFor[application.SigningSecurityIntent]())}}
	server.AddTool(&mcp.Tool{Name: "wos_manage_signing_identity", Description: "Administer scoped signing policy or one-use key enrollment. Registration requires a signed possession proof; tokens alone cannot replace a key. CAS versions are decimal strings. No private key is transmitted.", InputSchema: schema}, func(ctx context.Context, request *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, cancel, err := requestContext(ctx, request, options)
		defer cancel()
		if err != nil {
			return failure(err)
		}
		if len(request.Params.Arguments) > MaxPayloadBytes {
			return failure(domain.NewError(domain.ErrorCodeInvalidArgument, "payload exceeds 256 KiB"))
		}
		var input struct {
			Key     string          `json:"idempotency_key"`
			Command json.RawMessage `json:"command"`
		}
		if err = decodeStrict(request.Params.Arguments, &input); err != nil {
			return failure(err)
		}
		raw, err := normalizeInput(input.Command, reflect.TypeFor[application.SigningSecurityIntent]())
		if err != nil {
			return failure(err)
		}
		var intent application.SigningSecurityIntent
		if err = decodeStrict(raw, &intent); err != nil {
			return failure(err)
		}
		result, err := options.Security.SigningMutation(ctx, input.Key, intent)
		if err != nil {
			return failure(err)
		}
		return mutationSuccess(result)
	})
	type identityQuery struct {
		EnrollmentID *domain.ID `json:"enrollment_id,omitempty"`
	}
	mcp.AddTool(server, &mcp.Tool{Name: "wos_get_signing_identity", Description: "Read the authenticated credential's public identity, policy, bounded key metadata and optionally its enrollment challenge. No private key or bearer token is returned."}, func(ctx context.Context, request *mcp.CallToolRequest, query identityQuery) (*mcp.CallToolResult, any, error) {
		ctx, cancel, err := requestContext(ctx, request, options)
		defer cancel()
		if err != nil {
			r, e := failure(err)
			return r, nil, e
		}
		result, err := options.Security.SigningIdentity(ctx, query.EnrollmentID)
		if err != nil {
			r, e := failure(err)
			return r, nil, e
		}
		r, e := success(result)
		return r, nil, e
	})
}
