package mcptransport

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func registerSecurity(server *mcp.Server, options Options) {
	if options.Security == nil {
		return
	}
	registerSigningIdentity(server, options)
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
	mcp.AddTool(server, &mcp.Tool{Name: "wos_get_namespace_administration", Description: "Read namespace version, grants, credential metadata and bounded security audit; credentials:read and audit:read required."}, func(ctx context.Context, req *mcp.CallToolRequest, q queryArgs) (*mcp.CallToolResult, any, error) {
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
	type credentialQuery struct {
		NamespaceID domain.ID        `json:"namespace_id"`
		Limit       int              `json:"limit,omitempty"`
		Cursor      domain.ID        `json:"cursor,omitempty"`
		PrincipalID string           `json:"principal_id,omitempty"`
		Status      string           `json:"status,omitempty"`
		ActorKind   domain.ActorKind `json:"actor_kind,omitempty"`
		ActorID     string           `json:"actor_id,omitempty"`
	}
	mcp.AddTool(server, &mcp.Tool{Name: "wos_list_credentials", Description: "List safe credential metadata and effective policy with bounded pagination; credential secrets are never returned."}, func(ctx context.Context, req *mcp.CallToolRequest, q credentialQuery) (*mcp.CallToolResult, any, error) {
		ctx, cancel, err := requestContext(ctx, req, options)
		defer cancel()
		if err != nil {
			r, e := failure(err)
			return r, nil, e
		}
		value, err := options.Security.ListCredentials(ctx, q.NamespaceID, ports.CredentialManagementQuery{Limit: q.Limit, Cursor: q.Cursor, PrincipalID: q.PrincipalID, Status: q.Status, ActorKind: q.ActorKind, ActorID: q.ActorID})
		if err != nil {
			r, e := failure(err)
			return r, nil, e
		}
		r, e := success(value)
		return r, nil, e
	})
	type credentialDetailQuery struct {
		NamespaceID  domain.ID `json:"namespace_id"`
		CredentialID domain.ID `json:"credential_id"`
	}
	mcp.AddTool(server, &mcp.Tool{Name: "wos_get_credential", Description: "Read one credential's safe metadata and effective policy without exposing bearer material."}, func(ctx context.Context, req *mcp.CallToolRequest, q credentialDetailQuery) (*mcp.CallToolResult, any, error) {
		ctx, cancel, err := requestContext(ctx, req, options)
		defer cancel()
		if err != nil {
			r, e := failure(err)
			return r, nil, e
		}
		value, err := options.Security.GetCredential(ctx, q.NamespaceID, q.CredentialID)
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
	mcp.AddTool(server, &mcp.Tool{Name: "wos_administer_namespace", Description: "Create namespaces, manage grants, issue/update/rotate/revoke credentials. Expected Namespace and record revisions plus an idempotency key are required. A rotated secret appears once; replay omits it and never transfers work leases."}, func(ctx context.Context, req *mcp.CallToolRequest, in input) (*mcp.CallToolResult, any, error) {
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
