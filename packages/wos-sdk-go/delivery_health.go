package wossdk

import (
	"context"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"net/http"
	"net/url"
)

func (c *Client) GetDeliveryHealth(ctx context.Context, q a.DeliveryHealthQuery) (a.DeliveryHealthResult, error) {
	var result a.DeliveryHealthResult
	if e := q.NamespaceID.Validate(); e != nil {
		return result, e
	}
	path := "/namespaces/" + q.NamespaceID.String() + "/operations/delivery-health"
	if q.OutcomeID != nil {
		if e := q.OutcomeID.Validate(); e != nil {
			return result, e
		}
		path += "?outcome_id=" + url.QueryEscape(q.OutcomeID.String())
	}
	e := c.do(ctx, http.MethodGet, path, "", nil, &result)
	return result, e
}
