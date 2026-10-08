package mcptransport

import (
	"context"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-api/commands"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/http"
	"reflect"
	"strings"
	"time"
)

const MaxPayloadBytes = 256 << 10

type Options struct {
	Security        *application.SecurityService
	ResolveIdentity func(context.Context, http.Header) (application.Identity, error)
	LocalIdentity   *application.Identity
	RequestTimeout  time.Duration
}

func New(service *application.Service, ids ports.IDGenerator, options Options) (*mcp.Server, error) {
	if service == nil || ids == nil || (options.ResolveIdentity == nil && options.LocalIdentity == nil) {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "MCP requires application, IDs and an identity resolver")
	}
	if options.RequestTimeout <= 0 {
		options.RequestTimeout = 15 * time.Second
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "wos", Version: "0.1.0-dev"}, &mcp.ServerOptions{Instructions: "WOS preserves shared outcome state. Discover outcomes, read bounded continuity, explicitly claim work and preserve fencing tokens. Candidates are informational; clients decide execution. A timeout may follow a successful commit: retry the SAME intent and idempotency key. Published plans and proof history are immutable."})
	registerCommands(server, service, ids, options)
	registerQueries(server, service, options)
	registerResources(server, service, options)
	registerSecurity(server, options)
	return server, nil
}
func identityContext(ctx context.Context, headers http.Header, options Options) (context.Context, error) {
	var id application.Identity
	var err error
	if options.ResolveIdentity != nil {
		id, err = options.ResolveIdentity(ctx, headers)
	} else {
		id = *options.LocalIdentity
	}
	if err != nil {
		return ctx, err
	}
	return application.WithIdentity(ctx, id), nil
}
func requestContext(ctx context.Context, req *mcp.CallToolRequest, options Options) (context.Context, context.CancelFunc, error) {
	headers := http.Header{}
	if req.Extra != nil {
		headers = req.Extra.Header
	}
	ctx, err := identityContext(ctx, headers, options)
	ctx, cancel := context.WithTimeout(ctx, options.RequestTimeout)
	return ctx, cancel, err
}
func failure(err error) (*mcp.CallToolResult, error) {
	code, _ := domain.ErrorCodeOf(err)
	if code == "" {
		code = domain.ErrorCodeInvalidArgument
	}
	recovery := "inspect the error and correct the request"
	switch code {
	case domain.ErrorCodeVersionConflict:
		recovery = "read current state and reconcile intent before using a new expected_version"
	case domain.ErrorCodeContractExpired, domain.ErrorCodeContractRevoked, domain.ErrorCodeStaleExecution:
		recovery = "stop using this execution authority; reconcile current contract before explicit recovery"
	case domain.ErrorCodeContractSpecMismatch:
		recovery = "reload immutable canonical spec, preserving local drafts"
	case domain.ErrorCodeSubmissionNotAccepted:
		recovery = "review exact submission and criterion obligations before finalizing"
	case domain.ErrorCodeLease:
		recovery = "read operational state; preserve claim_id and fencing_token or reclaim explicitly"
	case domain.ErrorCodeIdempotencyConflict:
		recovery = "reuse a key only for the identical intent"
	}
	b, _ := json.Marshal(map[string]any{"error": map[string]any{"code": code, "message": err.Error(), "recovery": recovery}})
	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}, StructuredContent: json.RawMessage(b)}, nil
}
func success(value any) (*mcp.CallToolResult, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return failure(err)
	}
	if len(b) > MaxPayloadBytes {
		return failure(domain.NewError(domain.ErrorCodeGraphLimitExceeded, "query result exceeds byte limit; use bounded continuity and section cursors"))
	}
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(b)}}, StructuredContent: json.RawMessage(b)}, nil
}
func mutationSuccess(value any) (*mcp.CallToolResult, error) {
	b, err := json.Marshal(value)
	if err != nil {
		return failure(err)
	}
	if len(b) <= MaxPayloadBytes {
		return success(value)
	}
	receipt, err := commands.CommitReceipt(value)
	if err != nil {
		return failure(err)
	}
	return success(receipt)
}
func registerCommand[C any](server *mcp.Server, ids ports.IDGenerator, options Options, name string, run func(context.Context, domain.CommandContext, C) (any, error)) {
	schema := map[string]any{"type": "object", "additionalProperties": false, "required": []string{"idempotency_key", "command"}, "properties": map[string]any{"idempotency_key": map[string]any{"type": "string", "minLength": 16, "maxLength": 128}, "correlation_id": map[string]any{"type": "string", "maxLength": 256}, "command": typeSchema(reflect.TypeFor[C]())}}
	server.AddTool(&mcp.Tool{Name: name, Description: "Execute " + strings.TrimPrefix(name, "wos_") + " through the same Application service as HTTP. Optimistic versions and fencing are mandatory where specified. TTL uses seconds.", InputSchema: schema}, func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		ctx, cancel, err := requestContext(ctx, req, options)
		defer cancel()
		if err != nil {
			return failure(err)
		}
		if len(req.Params.Arguments) > MaxPayloadBytes {
			return failure(domain.NewError(domain.ErrorCodeInvalidArgument, "payload exceeds 256 KiB"))
		}
		var input struct {
			Key         string          `json:"idempotency_key"`
			Correlation string          `json:"correlation_id"`
			Command     json.RawMessage `json:"command"`
		}
		if err := decodeStrict(req.Params.Arguments, &input); err != nil {
			return failure(err)
		}
		if err := domain.ValidateIdempotencyKey(input.Key); err != nil {
			return failure(err)
		}
		raw, err := normalizeInput(input.Command, reflect.TypeFor[C]())
		if err != nil {
			return failure(err)
		}
		var command C
		if err = decodeStrict(raw, &command); err != nil {
			return failure(err)
		}
		id, err := ids.NewID()
		if err != nil {
			return failure(err)
		}
		identity, _ := application.IdentityFromContext(ctx)
		cc := domain.CommandContext{PrincipalID: identity.PrincipalID, Actor: identity.Actor, IdempotencyKey: input.Key, CommandID: id, CorrelationID: input.Correlation}
		value, err := run(ctx, cc, command)
		if err != nil {
			return failure(err)
		}
		return mutationSuccess(value)
	})
}
func typeSchema(t reflect.Type) map[string]any                  { return commands.Schema(t) }
func normalizeInput(raw []byte, t reflect.Type) ([]byte, error) { return commands.Normalize(raw, t) }
func decodeStrict(raw []byte, target any) error                 { return commands.Decode(raw, target) }
