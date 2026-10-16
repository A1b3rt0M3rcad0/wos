package wossdk

import (
	"context"
	"fmt"
	"net/url"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// ListWorkActivity reads an immutable, paginated history without granting authority.
func (c *Client) ListWorkActivity(ctx context.Context, scope domain.Scope, filter ports.WorkActivityFilter, limit int, cursor string) (application.WorkActivityPage, error) {
	var value application.WorkActivityPage
	if err := scope.Validate(); err != nil {
		return value, err
	}
	query := url.Values{"cursor": {cursor}}
	if filter.Descending {
		query.Set("descending", "true")
	}
	if limit != 0 {
		query.Set("limit", fmt.Sprint(limit))
	}
	for name, id := range map[string]domain.ID{"work_item_id": filter.WorkItemID, "contract_id": filter.ContractID} {
		if !id.IsZero() {
			if err := id.Validate(); err != nil {
				return value, err
			}
			query.Set(name, id.String())
		}
	}
	err := c.do(ctx, "GET", scopePath(scope)+"/work-activity?"+query.Encode(), "", nil, &value)
	return value, err
}

// GetWorkAttention reads current coordination signals; it does not mutate work.
func (c *Client) GetWorkAttention(ctx context.Context, scope domain.Scope, id domain.ID) (application.WorkAttention, error) {
	var value application.WorkAttention
	if err := scope.Validate(); err != nil {
		return value, err
	}
	if err := id.Validate(); err != nil {
		return value, err
	}
	err := c.do(ctx, "GET", scopePath(scope)+"/work-items/"+id.String()+"/attention", "", nil, &value)
	return value, err
}
