package mcptransport

import (
	"context"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"net/http"
	"net/url"
	"strings"
)

func registerSignedResources(server *mcp.Server, service *application.Service, options Options) {
	server.AddResourceTemplate(&mcp.ResourceTemplate{Name: "signed_state", Description: "One scoped signed-v2 resource. Compact metadata by default; specification, authority, fact or submission can be expanded with an expected digest. Collections use the focal query tool for bounded pagination.", URITemplate: "wos://namespaces/{namespace_id}/outcomes/{outcome_id}/signed-state/{resource}/{entity_id}", MIMEType: "application/json"}, func(ctx context.Context, request *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		headers := http.Header{}
		if request.Extra != nil {
			headers = request.Extra.Header
		}
		ctx, err := identityContext(ctx, headers, options)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(ctx, options.RequestTimeout)
		defer cancel()
		u, err := url.Parse(request.Params.URI)
		if err != nil {
			return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid signed resource URI")
		}
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if u.Scheme != "wos" || u.Host != "namespaces" || u.User != nil || u.Fragment != "" || len(parts) != 6 || parts[1] != "outcomes" || parts[3] != "signed-state" {
			return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid signed resource URI")
		}
		ns, err := domain.ParseID(parts[0])
		if err != nil {
			return nil, err
		}
		outcome, err := domain.ParseID(parts[2])
		if err != nil {
			return nil, err
		}
		id, err := domain.ParseID(parts[5])
		if err != nil {
			return nil, err
		}
		parameters, err := url.ParseQuery(u.RawQuery)
		if err != nil {
			return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid signed resource query")
		}
		for key, values := range parameters {
			if (key != "digest" && key != "contract_kind") || len(values) != 1 {
				return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "unknown or repeated signed resource query")
			}
		}
		value, err := service.ReadSignedState(ctx, application.SignedStateQuery{Scope: domain.Scope{NamespaceID: ns, OutcomeID: outcome}, Resource: parts[4], ID: id, Digest: parameters.Get("digest"), ContractKind: parameters.Get("contract_kind")})
		if err != nil {
			return nil, err
		}
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: request.Params.URI, MIMEType: "application/json", Text: string(raw)}}}, nil
	})
}
