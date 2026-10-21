package wossdk

import (
	"context"
	"fmt"
	"net/url"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func (c *Client) GetEntityWorkspaceView(ctx context.Context, scope domain.Scope, query application.WorkspaceViewQuery) (application.EntityWorkspaceView, error) {
	var result application.EntityWorkspaceView
	if err := scope.Validate(); err != nil {
		return result, err
	}
	if query.Subject.Ref.Scope != scope {
		return result, fmt.Errorf("workspace subject differs from requested scope")
	}
	err := c.do(ctx, "POST", scopePath(scope)+"/workspace-view", "", query, &result)
	return result, err
}

func (c *Client) GetWorkspaceCards(ctx context.Context, scope domain.Scope, query application.WorkspaceCardsQuery) (application.WorkspaceCardsPage, error) {
	var result application.WorkspaceCardsPage
	if err := scope.Validate(); err != nil {
		return result, err
	}
	for _, ref := range query.Refs {
		if ref.Scope != scope {
			return result, fmt.Errorf("card subject differs from requested scope")
		}
	}
	if query.ParentRef != nil && query.ParentRef.Scope != scope {
		return result, fmt.Errorf("card parent differs from requested scope")
	}
	err := c.do(ctx, "POST", scopePath(scope)+"/workspace-cards", "", query, &result)
	return result, err
}

func (c *Client) GetWorkspaceTimeline(ctx context.Context, scope domain.Scope, query application.WorkspaceTimelineQuery) (application.WorkspaceTimelinePage, error) {
	var result application.WorkspaceTimelinePage
	if err := scope.Validate(); err != nil {
		return result, err
	}
	values := url.Values{}
	if query.SubjectRef != nil {
		if query.SubjectRef.Scope != scope {
			return result, fmt.Errorf("timeline subject differs from requested scope")
		}
		values.Set("kind", string(query.SubjectRef.Kind))
		values.Set("entity_id", query.SubjectRef.ID.String())
	}
	if query.CriterionID != nil {
		values.Set("criterion_id", query.CriterionID.String())
	}
	if query.IncludeDescendants {
		values.Set("include_descendants", "true")
	}
	if query.Limit != 0 {
		values.Set("limit", fmt.Sprint(query.Limit))
	}
	if query.Cursor != "" {
		values.Set("cursor", query.Cursor)
	}
	err := c.do(ctx, "GET", scopePath(scope)+"/workspace-timeline?"+values.Encode(), "", nil, &result)
	return result, err
}

func (c *Client) GetEntityNote(ctx context.Context, scope domain.Scope, id domain.ID) (domain.EntityNote, error) {
	var result domain.EntityNote
	if err := scope.Validate(); err != nil {
		return result, err
	}
	if err := id.Validate(); err != nil {
		return result, err
	}
	err := c.do(ctx, "GET", scopePath(scope)+"/entity-notes/"+url.PathEscape(id.String()), "", nil, &result)
	return result, err
}
