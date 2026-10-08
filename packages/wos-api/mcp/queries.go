package mcptransport

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type queryArgs struct {
	ExternalContext    domain.ExternalContext   `json:"external_context,omitempty"`
	CreatorPrincipalID string                   `json:"creator_principal_id,omitempty"`
	Root               *domain.EntityRef        `json:"root,omitempty"`
	Kinds              []string                 `json:"kinds,omitempty"`
	CriterionID        domain.ID                `json:"criterion_id,omitempty"`
	ConclusionID       domain.ID                `json:"conclusion_id,omitempty"`
	RevisionNumber     uint64                   `json:"revision_number,omitempty"`
	PlanScope          *domain.RoadmapPlanScope `json:"plan_scope,omitempty"`
	ExternalProvider   string                   `json:"external_provider,omitempty"`
	ExternalKind       string                   `json:"external_kind,omitempty"`
	ExternalID         string                   `json:"external_id,omitempty"`
	Owner              *domain.ActorRef         `json:"owner,omitempty"`
	Kind               domain.EntityKind        `json:"kind,omitempty"`
	Depth              int                      `json:"depth,omitempty"`
	Direction          string                   `json:"direction,omitempty"`
	NamespaceID        domain.ID                `json:"namespace_id"`
	OutcomeID          domain.ID                `json:"outcome_id,omitempty"`
	Limit              int                      `json:"limit,omitempty"`
	Cursor             string                   `json:"cursor,omitempty"`
	Section            string                   `json:"section,omitempty"`
	Text               string                   `json:"text,omitempty"`
	Lifecycle          string                   `json:"lifecycle,omitempty"`
	Archived           *bool                    `json:"archived,omitempty"`
	Priority           domain.Priority          `json:"priority,omitempty"`
	EntityID           domain.ID                `json:"entity_id,omitempty"`
	PrincipalID        string                   `json:"principal_id,omitempty"`
	CommandID          domain.ID                `json:"command_id,omitempty"`
	EventType          string                   `json:"event_type,omitempty"`
}

func registerQueries(server *mcp.Server, s *application.Service, options Options) {
	for _, name := range []string{"wos_list_available_work", "wos_capabilities", "wos_get_work_contract", "wos_get_work_contract_spec", "wos_list_work_contracts", "wos_list_contract_checkpoints", "wos_list_work_submissions", "wos_get_work_submission", "wos_get_command_receipt", "wos_get_criterion_history", "wos_list_conclusions", "wos_get_conclusion", "wos_get_roadmap_revision", "wos_list_roadmaps", "wos_get_active_roadmap_slot", "wos_list_roadmap_activation_history", "wos_list_triggers", "wos_list_deliveries", "wos_list_trigger_firings", "wos_search_outcomes", "wos_get_continuity", "wos_get_continuity_section", "wos_get_timeline", "wos_list_ready_work", "wos_get_work_context", "wos_get_outcome_graph", "wos_get_entity"} {
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
			case "wos_list_available_work":
				value, err = s.ListAvailableWork(ctx, scope, q.Limit, q.Cursor)
			case "wos_capabilities":
				value = application.ContractCapabilities()
			case "wos_get_work_contract", "wos_get_work_contract_spec":
				value, err = s.GetWorkContract(ctx, scope, q.EntityID)
			case "wos_list_work_contracts":
				value, err = s.WorkContractHistory(ctx, scope, ports.ContractFilter{Limit: q.Limit, WorkItemID: q.EntityID, HolderPrincipalID: q.PrincipalID, Status: domain.ContractStatus(q.Lifecycle)}, q.Cursor)
			case "wos_list_contract_checkpoints":
				value, err = s.ContractRecords(ctx, scope, q.EntityID, "checkpoints", q.Limit, q.Cursor)
			case "wos_list_work_submissions":
				value, err = s.ContractRecords(ctx, scope, q.EntityID, "submissions", q.Limit, q.Cursor)
			case "wos_get_work_submission":
				value, err = s.GetWorkSubmission(ctx, scope, q.EntityID)
			case "wos_get_command_receipt":
				value, err = s.GetCommandReceipt(ctx, q.NamespaceID, q.CommandID)

			case "wos_get_criterion_history":
				value, err = s.GetCriterionHistory(ctx, domain.EntityRef{Scope: scope, Kind: q.Kind, ID: q.EntityID}, q.CriterionID)
			case "wos_list_conclusions":
				value, err = s.ListConclusions(ctx, domain.EntityRef{Scope: scope, Kind: q.Kind, ID: q.EntityID})
			case "wos_get_conclusion":
				value, err = s.GetConclusion(ctx, domain.EntityRef{Scope: scope, Kind: q.Kind, ID: q.EntityID}, q.ConclusionID)
			case "wos_get_roadmap_revision":
				value, err = s.GetRoadmapRevision(ctx, scope, q.EntityID, q.RevisionNumber)
			case "wos_list_roadmaps":
				var items []domain.Roadmap
				var revision domain.OutcomeRevision
				items, revision, err = s.ListRoadmaps(ctx, scope)
				value = map[string]any{"items": items, "outcome_revision": revision}
			case "wos_get_active_roadmap_slot", "wos_list_roadmap_activation_history":
				plan := domain.RoadmapPlanScope{Kind: domain.RoadmapScopeOutcome, ID: q.OutcomeID}
				if q.PlanScope != nil {
					plan = *q.PlanScope
				}
				var revision domain.OutcomeRevision
				var data any
				if name == "wos_get_active_roadmap_slot" {
					data, revision, err = s.GetActiveRoadmapSlot(ctx, scope, plan)
				} else {
					data, revision, err = s.ListRoadmapActivationHistory(ctx, scope, plan)
				}
				value = map[string]any{"value": data, "outcome_revision": revision}
			case "wos_list_triggers":
				var items []domain.Trigger
				var revision domain.OutcomeRevision
				items, revision, err = s.ListTriggers(ctx, scope)
				value = map[string]any{"items": items, "outcome_revision": revision}
			case "wos_list_deliveries":
				value, err = s.ListDeliveries(ctx, scope, q.Limit, q.Cursor)
			case "wos_list_trigger_firings":
				value, err = s.ListTriggerFirings(ctx, scope, q.Limit, q.Cursor)
			case "wos_get_work_context":
				value, err = s.GetWorkContext(ctx, scope, q.EntityID)
			case "wos_get_outcome_graph":
				depth := q.Depth
				if depth == 0 {
					depth = 2
				}
				value, err = s.GetOutcomeGraph(ctx, scope, application.GraphQuery{Root: q.Root, Kinds: q.Kinds, Limit: q.Limit, Cursor: q.Cursor, Depth: depth, Direction: q.Direction})
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
				value, err = s.SearchOutcomes(ctx, q.NamespaceID, ports.OutcomeFilter{ExternalContext: q.ExternalContext, CreatorPrincipalID: q.CreatorPrincipalID, ExternalProvider: q.ExternalProvider, ExternalKind: q.ExternalKind, ExternalID: q.ExternalID, Owner: q.Owner, Text: q.Text, Lifecycle: q.Lifecycle, Archived: q.Archived, Priority: q.Priority}, q.Limit, q.Cursor)
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
}
