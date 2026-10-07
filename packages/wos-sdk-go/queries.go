package wossdk

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"net/url"
)

func (c *Client) DiscoverOutcomes(ctx context.Context, ns domain.ID, filter ports.OutcomeFilter, limit int, cursor string) (application.OutcomePage, error) {
	var result application.OutcomePage
	query := url.Values{"limit": {fmt.Sprint(limit)}, "cursor": {cursor}, "text": {filter.Text}, "lifecycle": {filter.Lifecycle}, "priority": {string(filter.Priority)}, "creator_principal_id": {filter.CreatorPrincipalID}, "external_provider": {filter.ExternalProvider}, "external_kind": {filter.ExternalKind}, "external_id": {filter.ExternalID}}
	if filter.Archived != nil {
		query.Set("archived", fmt.Sprint(*filter.Archived))
	}
	if filter.Owner != nil {
		query.Set("owner_kind", string(filter.Owner.Kind))
		query.Set("owner_provider", filter.Owner.Provider)
		query.Set("owner_id", filter.Owner.ID)
	}
	if len(filter.ExternalContext) > 0 {
		raw, err := json.Marshal(filter.ExternalContext)
		if err != nil {
			return result, err
		}
		query.Set("external_context", string(raw))
	}
	err := c.do(ctx, "GET", "/namespaces/"+ns.String()+"/outcomes?"+query.Encode(), "", nil, &result)
	return result, err
}
func (c *Client) GetTimeline(ctx context.Context, scope domain.Scope, q application.TimelineQuery) (application.TimelinePage, error) {
	var result application.TimelinePage
	query := url.Values{"limit": {fmt.Sprint(q.Limit)}, "cursor": {q.Cursor}, "principal_id": {q.PrincipalID}, "command_id": {string(q.CommandID)}, "entity_id": {string(q.EntityID)}, "event_type": {q.EventType}}
	err := c.do(ctx, "GET", scopePath(scope)+"/timeline?"+query.Encode(), "", nil, &result)
	return result, err
}
func (c *Client) GetOutcomeGraph(ctx context.Context, scope domain.Scope, q application.GraphQuery) (application.GraphPage, error) {
	var result application.GraphPage
	query := url.Values{"limit": {fmt.Sprint(q.Limit)}, "depth": {fmt.Sprint(q.Depth)}, "cursor": {q.Cursor}, "direction": {q.Direction}}
	if q.Root != nil {
		query.Set("root_kind", string(q.Root.Kind))
		query.Set("root_id", q.Root.ID.String())
	}
	for i, kind := range q.Kinds {
		if i == 0 {
			query.Set("kinds", kind)
		} else {
			query.Set("kinds", query.Get("kinds")+","+kind)
		}
	}
	err := c.do(ctx, "GET", scopePath(scope)+"/graph?"+query.Encode(), "", nil, &result)
	return result, err
}
func (c *Client) AdministerNamespace(ctx context.Context, key string, intent application.AdministrativeIntent) (ports.SecurityAdminResult, string, error) {
	var result struct {
		Result ports.SecurityAdminResult `json:"result"`
		Token  string                    `json:"token"`
	}
	if err := domain.ValidateIdempotencyKey(key); err != nil {
		return result.Result, "", err
	}
	err := c.do(ctx, "POST", "/security/commands", key, intent, &result)
	return result.Result, result.Token, err
}
