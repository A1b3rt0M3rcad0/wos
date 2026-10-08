package wossdk

import (
	"context"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"net/http"
	"net/url"
	"strconv"
)

func (c *Client) ReadSignedState(ctx context.Context, query a.SignedStateQuery) (a.SignedStateResult, error) {
	var result a.SignedStateResult
	if err := query.Scope.Validate(); err != nil {
		return result, err
	}
	parameters := url.Values{}
	if query.ContractKind != "" {
		parameters.Set("contract_kind", query.ContractKind)
	}
	if query.Digest != "" {
		parameters.Set("digest", query.Digest)
	}
	if query.IdempotencyKey != "" {
		parameters.Set("idempotency_key", query.IdempotencyKey)
	}
	if query.Cursor != "" {
		parameters.Set("cursor", query.Cursor)
	}
	if query.Limit != 0 {
		parameters.Set("limit", strconv.Itoa(query.Limit))
	}
	path := "/namespaces/" + query.Scope.NamespaceID.String() + "/outcomes/" + query.Scope.OutcomeID.String() + "/signed-state/" + url.PathEscape(query.Resource)
	if !query.ID.IsZero() {
		path += "/" + query.ID.String()
	}
	if len(parameters) > 0 {
		path += "?" + parameters.Encode()
	}
	err := c.do(ctx, http.MethodGet, path, "", nil, &result)
	return result, err
}
func (c *Client) GetSignedTrust(ctx context.Context, namespace d.ID) (a.SignedStateResult, error) {
	var result a.SignedStateResult
	if err := namespace.Validate(); err != nil {
		return result, err
	}
	err := c.do(ctx, http.MethodGet, "/namespaces/"+namespace.String()+"/signed-trust", "", nil, &result)
	return result, err
}
