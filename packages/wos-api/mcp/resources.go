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

func registerResources(server *mcp.Server, s *application.Service, options Options) {
	registerSignedResources(server, s, options)
	server.AddResourceTemplate(&mcp.ResourceTemplate{Name: "outcome_continuity", Description: "Authorized compact continuity, schema 1. Use tool section cursors to expand omissions.", URITemplate: "wos://namespaces/{namespace_id}/outcomes/{outcome_id}/continuity", MIMEType: "application/json"}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		headers := http.Header{}
		if req.Extra != nil {
			headers = req.Extra.Header
		}
		ctx, err := identityContext(ctx, headers, options)
		if err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(ctx, options.RequestTimeout)
		defer cancel()
		u, err := url.Parse(req.Params.URI)
		if err != nil {
			return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid resource URI")
		}
		parts := strings.Split(strings.Trim(u.Path, "/"), "/")
		if u.Scheme != "wos" || u.Host != "namespaces" || u.RawQuery != "" || u.Fragment != "" || len(parts) != 4 || parts[1] != "outcomes" || parts[3] != "continuity" {
			return nil, domain.NewError(domain.ErrorCodeInvalidArgument, "invalid continuity resource URI")
		}
		ns, err := domain.ParseID(parts[0])
		if err != nil {
			return nil, err
		}
		id, err := domain.ParseID(parts[2])
		if err != nil {
			return nil, err
		}
		result, err := s.GetContinuity(ctx, domain.Scope{NamespaceID: ns, OutcomeID: id}, 25)
		if err != nil {
			return nil, err
		}
		b, err := json.Marshal(result)
		if err != nil {
			return nil, err
		}
		return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: req.Params.URI, MIMEType: "application/json", Text: string(b)}}}, nil
	})
}
