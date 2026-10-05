package mcptransport

import (
	"context"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type queryArgs struct {
	Kind        domain.EntityKind `json:"kind,omitempty"`
	Depth       int               `json:"depth,omitempty"`
	Direction   string            `json:"direction,omitempty"`
	NamespaceID domain.ID         `json:"namespace_id"`
	OutcomeID   domain.ID         `json:"outcome_id,omitempty"`
	Limit       int               `json:"limit,omitempty"`
	Cursor      string            `json:"cursor,omitempty"`
	Section     string            `json:"section,omitempty"`
	Text        string            `json:"text,omitempty"`
	Lifecycle   string            `json:"lifecycle,omitempty"`
	Archived    *bool             `json:"archived,omitempty"`
	Priority    domain.Priority   `json:"priority,omitempty"`
	EntityID    domain.ID         `json:"entity_id,omitempty"`
	PrincipalID string            `json:"principal_id,omitempty"`
	CommandID   domain.ID         `json:"command_id,omitempty"`
	EventType   string            `json:"event_type,omitempty"`
}

func registerQueries(server *mcp.Server, s *application.Service, options Options) {
	for _, name := range []string{"wos_search_outcomes", "wos_get_continuity", "wos_get_continuity_section", "wos_get_timeline", "wos_list_ready_work", "wos_get_work_context", "wos_get_outcome_graph", "wos_get_entity"} {
		name := name
		mcp.AddTool(server, &mcp.Tool{Name: name, Description: "Read authorized WOS state; limit 1–100, default 25. Expand omissions with section cursors; candidates never authorize execution."}, func(ctx context.Context, req *mcp.CallToolRequest, q queryArgs) (*mcp.CallToolResult, any, error) {
			ctx, cancel, err := requestContext(ctx, req, options)
			defer cancel()
			if err != nil {
				r, e := failure(err)
				return r, nil, e
			}
			scope := domain.Scope{NamespaceID: q.NamespaceID, OutcomeID: q.OutcomeID}
			var value any
			switch name {
			case "wos_get_work_context":
				value, err = s.GetWorkContext(ctx, scope, q.EntityID)
			case "wos_get_outcome_graph":
				depth := q.Depth
				if depth == 0 {
					depth = 2
				}
				value, err = s.GetOutcomeGraph(ctx, scope, application.GraphQuery{Limit: q.Limit, Cursor: q.Cursor, Depth: depth, Direction: q.Direction})
			case "wos_get_entity":
				switch q.Kind {
				case domain.EntityKindOutcome:
					value, err = s.GetOutcome(ctx, scope)
				case domain.EntityKindObjective:
					value, err = s.GetObjective(ctx, scope, q.EntityID)
				case domain.EntityKindWorkItem:
					value, err = s.GetWorkItem(ctx, scope, q.EntityID)
				case domain.EntityKindIssue:
					value, err = s.GetIssue(ctx, scope, q.EntityID)
				case domain.EntityKindBlocker:
					value, err = s.GetBlocker(ctx, scope, q.EntityID)
				case domain.EntityKindArtifact:
					value, err = s.GetArtifact(ctx, scope, q.EntityID)
				case domain.EntityKindEvidence:
					value, err = s.GetEvidence(ctx, scope, q.EntityID)
				case domain.EntityKindDecision:
					value, err = s.GetDecision(ctx, scope, q.EntityID)
				case domain.EntityKindRoadmap:
					value, err = s.GetRoadmap(ctx, scope, q.EntityID)
				default:
					err = domain.NewError(domain.ErrorCodeInvalidEntityKind, "unsupported entity kind")
				}
			case "wos_search_outcomes":
				value, err = s.SearchOutcomes(ctx, q.NamespaceID, ports.OutcomeFilter{Text: q.Text, Lifecycle: q.Lifecycle, Archived: q.Archived, Priority: q.Priority}, q.Limit, q.Cursor)
			case "wos_get_continuity":
				value, err = s.GetContinuity(ctx, scope, q.Limit)
			case "wos_get_continuity_section":
				value, err = s.GetContinuitySection(ctx, scope, q.Section, q.Limit, q.Cursor)
			case "wos_get_timeline":
				value, err = s.GetTimeline(ctx, scope, application.TimelineQuery{Limit: q.Limit, Cursor: q.Cursor, EntityID: q.EntityID, PrincipalID: q.PrincipalID, CommandID: q.CommandID, EventType: q.EventType})
			case "wos_list_ready_work":
				value, err = s.GetContinuitySection(ctx, scope, "ready_work", q.Limit, q.Cursor)
			}
			if err != nil {
				r, e := failure(err)
				return r, nil, e
			}
			r, e := success(value)
			return r, nil, e
		})
	}
	_ = json.Valid
}
