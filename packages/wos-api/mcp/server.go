package mcptransport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const MaxPayloadBytes = 256 << 10

type Options struct {
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
	var v map[string]json.RawMessage
	if err = json.Unmarshal(b, &v); err != nil {
		return failure(err)
	}
	receipt := map[string]any{"command_id": v["command_id"], "outcome_revision": v["outcome_revision"], "idempotent_replay": v["idempotent_replay"], "result_omitted": true, "recovery": "mutation committed; consult bounded continuity for current state"}
	var entity map[string]json.RawMessage
	if json.Unmarshal(v["value"], &entity) == nil {
		receipt["entity_id"] = entity["id"]
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
func decodeStrict(raw []byte, target any) error {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.DisallowUnknownFields()
	if err := d.Decode(target); err != nil {
		return domain.WrapError(domain.ErrorCodeInvalidArgument, "invalid tool arguments", err)
	}
	return nil
}
func snake(s string) string {
	r := []rune(s)
	var b strings.Builder
	for i, c := range r {
		if unicode.IsUpper(c) && i > 0 && (unicode.IsLower(r[i-1]) || (i+1 < len(r) && unicode.IsLower(r[i+1]))) {
			b.WriteByte('_')
		}
		b.WriteRune(unicode.ToLower(c))
	}
	return b.String()
}
func fieldName(f reflect.StructField) string {
	tag := strings.Split(f.Tag.Get("json"), ",")[0]
	if tag != "" {
		return tag
	}
	if f.Type == reflect.TypeFor[time.Duration]() {
		return snake(f.Name) + "_seconds"
	}
	return snake(f.Name)
}
func typeSchema(t reflect.Type) map[string]any {
	if t.Kind() == reflect.Pointer {
		return typeSchema(t.Elem())
	}
	if t == reflect.TypeFor[time.Time]() {
		return map[string]any{"type": "string", "format": "date-time"}
	}
	if t == reflect.TypeFor[domain.EntityRef]() {
		return map[string]any{"type": "object", "additionalProperties": false, "required": []string{"namespace_id", "outcome_id", "kind", "id"}, "properties": map[string]any{"namespace_id": typeSchema(reflect.TypeFor[domain.ID]()), "outcome_id": typeSchema(reflect.TypeFor[domain.ID]()), "id": typeSchema(reflect.TypeFor[domain.ID]()), "kind": map[string]any{"type": "string"}}}
	}
	switch t.Kind() {
	case reflect.String:
		if t == reflect.TypeFor[domain.ID]() {
			return map[string]any{"type": "string", "pattern": "^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$"}
		}
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int64, reflect.Uint, reflect.Uint64, reflect.Uint32:
		return map[string]any{"type": "integer"}
	case reflect.Float64:
		return map[string]any{"type": "number"}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": typeSchema(t.Elem())}
	case reflect.Map, reflect.Interface:
		return map[string]any{}
	case reflect.Struct:
		p := map[string]any{}
		required := []string{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() || fieldName(f) == "-" {
				continue
			}
			name := fieldName(f)
			p[name] = typeSchema(f.Type)
			if f.Type.Kind() != reflect.Pointer && !strings.Contains(f.Tag.Get("json"), "omitempty") && f.Name != "Description" && f.Name != "Reason" && f.Name != "ResultSummary" && f.Type.Kind() != reflect.Slice {
				required = append(required, name)
			}
		}
		return map[string]any{"type": "object", "properties": p, "required": required, "additionalProperties": false}
	default:
		return map[string]any{}
	}
}
func normalizeInput(raw []byte, t reflect.Type) ([]byte, error) {
	var v any
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	if err := d.Decode(&v); err != nil {
		return nil, err
	}
	value, err := normalizeValue(v, t)
	if err != nil {
		return nil, err
	}
	return json.Marshal(value)
}
func normalizeValue(value any, t reflect.Type) (any, error) {
	if value == nil {
		return nil, nil
	}
	if t.Kind() == reflect.Pointer {
		return normalizeValue(value, t.Elem())
	}
	if t == reflect.TypeFor[time.Time]() || t == reflect.TypeFor[domain.EntityRef]() {
		return value, nil
	}
	if t == reflect.TypeFor[time.Duration]() {
		n, ok := value.(json.Number)
		if !ok {
			return nil, fmt.Errorf("TTL seconds must be an integer")
		}
		seconds, err := strconv.ParseInt(string(n), 10, 64)
		if err != nil || seconds < 1 || seconds > 86400 {
			return nil, fmt.Errorf("TTL seconds must be between 1 and 86400")
		}
		return seconds * int64(time.Second), nil
	}
	switch t.Kind() {
	case reflect.Struct:
		m, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("object required")
		}
		out := map[string]any{}
		known := map[string]bool{}
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			if !f.IsExported() {
				continue
			}
			name := fieldName(f)
			known[name] = true
			if v, ok := m[name]; ok {
				transformed, err := normalizeValue(v, f.Type)
				if err != nil {
					return nil, err
				}
				key := strings.Split(f.Tag.Get("json"), ",")[0]
				if key == "" {
					key = f.Name
				}
				out[key] = transformed
			}
		}
		for k := range m {
			if !known[k] {
				return nil, fmt.Errorf("unknown field %q", k)
			}
		}
		return out, nil
	case reflect.Slice, reflect.Array:
		a, ok := value.([]any)
		if !ok {
			return nil, fmt.Errorf("array required")
		}
		for i := range a {
			v, err := normalizeValue(a[i], t.Elem())
			if err != nil {
				return nil, err
			}
			a[i] = v
		}
		return a, nil
	default:
		return value, nil
	}
}
