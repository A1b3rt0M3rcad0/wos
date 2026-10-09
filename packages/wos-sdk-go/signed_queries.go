package wossdk

import (
	"context"
	"encoding/base64"
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
	maximum := int64(256 << 10)
	if query.Resource == "operation" {
		// The registry stores at most 1 MiB; base64 expands it by 4/3.
		// Leave bounded room for the focal metadata, without widening v1 reads.
		maximum = int64(base64.StdEncoding.EncodedLen(d.MaxSignedOperationResultBytes) + (64 << 10))
	}
	err := c.doBounded(ctx, http.MethodGet, path, "", nil, &result, maximum)
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
