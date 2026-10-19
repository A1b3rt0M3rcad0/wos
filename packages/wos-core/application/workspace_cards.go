package application

import (
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sort"
	"strings"
	"unicode/utf8"
)

func workspaceText(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	value = value[:limit]
	for !utf8.ValidString(value) {
		value = value[:len(value)-1]
	}
	return value
}
func workspaceCardKey(ref d.EntityRef) string { return ref.Kind.String() + ":" + ref.ID.String() }

func workspaceCursorFilter(ctx context.Context, query any) string {
	identity, _ := IdentityFromContext(ctx)
	// Optional domain.ID values must be strings: their MarshalText intentionally
	// rejects zero IDs. The cursor remains bound to the authenticated reader.
	return filterHash(struct {
		Query                    any       `json:"query"`
		PrincipalID              string    `json:"principal_id"`
		CredentialID             string    `json:"credential_id"`
		CredentialPolicyRevision d.Version `json:"credential_policy_revision"`
	}{query, identity.PrincipalID, string(identity.CredentialID), identity.CredentialPolicyRevision})
}

func (s *Service) GetWorkspaceCards(ctx context.Context, scope d.Scope, q WorkspaceCardsQuery) (WorkspaceCardsPage, error) {
	zero := WorkspaceCardsPage{}
	if (q.ParentRef == nil) == (len(q.Refs) == 0) {
		return zero, d.NewError(d.ErrorCodeInvalidArgument, "supply either refs or parent_ref")
	}
	if len(q.Refs) > 100 {
		return zero, d.NewError(d.ErrorCodeInvalidArgument, "workspace card batch exceeds 100 references")
	}
	limit := q.Limit
	if limit == 0 {
		limit = 50
	}
	if limit < 1 || limit > 100 {
		return zero, d.NewError(d.ErrorCodeInvalidArgument, "workspace card limit must be between 1 and 100")
	}
	refs := append([]d.EntityRef{}, q.Refs...)
	seen := map[d.EntityRef]bool{}
	for _, ref := range refs {
		if e := ref.Validate(); e != nil || ref.Scope != scope {
			return zero, d.NewError(d.ErrorCodeInvalidEntityRef, "card references must belong to the Outcome")
		}
		if seen[ref] {
			return zero, d.NewError(d.ErrorCodeInvalidArgument, "duplicate card reference")
		}
		seen[ref] = true
	}
	if q.ParentRef != nil {
		if e := q.ParentRef.Validate(); e != nil || q.ParentRef.Scope != scope || (q.ParentRef.Kind != d.EntityKindOutcome && q.ParentRef.Kind != d.EntityKindObjective) {
			return zero, d.NewError(d.ErrorCodeInvalidEntityRef, "structural parent must be an Outcome or Objective in this scope")
		}
	} else if q.Cursor != "" {
		return zero, d.NewError(d.ErrorCodeInvalidArgument, "a reference batch has no cursor")
	}
	frame, handled, e := s.openWorkspaceFrame(ctx, scope)
	if e != nil {
		return zero, e
	}
	if !handled {
		return zero, d.NewError(d.ErrorCodeInvalidConfig, "workspace read adapter is unavailable")
	}
	defer frame.u.Rollback()
	repo, ok := frame.repo.(ports.WorkspaceReadRepository)
	if !ok {
		return zero, d.NewError(d.ErrorCodeInvalidConfig, "workspace read adapter is unavailable")
	}
	graph, ok := frame.repo.(ports.GraphReadRepository)
	if !ok {
		return zero, d.NewError(d.ErrorCodeInvalidConfig, "workspace graph adapter is unavailable")
	}
	page := WorkspaceCardsPage{Items: []WorkspaceCardSummary{}, OutcomeRevision: frame.revision, EvaluatedAt: frame.at, Coverage: WorkspaceCoverage{Status: "complete"}}
	after := ""
	hash := workspaceCursorFilter(ctx, q.ParentRef)
	if q.ParentRef != nil {
		parent, e := graph.GraphNodes(ctx, scope, []d.EntityRef{*q.ParentRef})
		if e != nil {
			return zero, e
		}
		if len(parent) != 1 {
			return zero, d.NewError(d.ErrorCodeNotFound, "structural parent not found")
		}
		if q.Cursor != "" {
			c, e := decodeCursor(q.Cursor, scope.NamespaceID, scope.OutcomeID, "workspace_cards", hash)
			if e != nil {
				return zero, e
			}
			if c.Revision != frame.revision {
				return zero, d.NewError(d.ErrorCodePreconditionFailed, "snapshot_changed")
			}
			after = c.Key
		}
		refs, e = repo.WorkspaceChildren(ctx, scope, *q.ParentRef, after, limit+1)
		if e != nil {
			return zero, e
		}
		if len(refs) > limit {
			refs = refs[:limit]
			page.Coverage.Status = "partial"
			page.Coverage.NextCursor = encodeCursor(queryCursor{Namespace: scope.NamespaceID, Outcome: scope.OutcomeID, Section: "workspace_cards", Filter: hash, Revision: frame.revision, Key: workspaceCardKey(refs[len(refs)-1])})
			page.Coverage.Reason = "direct_children_page"
		}
	}
	if len(refs) > 0 {
		summaries, e := graph.GraphNodes(ctx, scope, refs)
		if e != nil {
			return zero, e
		}
		if len(summaries) != len(refs) {
			return zero, d.NewError(d.ErrorCodeNotFound, "one or more workspace records were not found")
		}
		facts, e := repo.WorkspaceCards(ctx, scope, refs)
		if e != nil {
			return zero, e
		}
		byRef := map[d.EntityRef]ports.WorkspaceCardFacts{}
		for _, f := range facts {
			byRef[f.Ref] = f
		}
		for _, head := range summaries {
			f := byRef[head.Ref]
			v := WorkspaceCardSummary{Ref: head.Ref, Version: head.Version, Title: head.Title, Lifecycle: head.Lifecycle, DisplayState: head.Lifecycle, Priority: head.Priority, Description: workspaceText(f.Description, 240), ParentRef: f.ParentRef, RequiredForOutcome: f.RequiredForOutcome, Participants: []WorkspacePerson{}, AuthorityState: "not_applicable"}
			if head.Ref.Kind == d.EntityKindWorkItem {
				v.AuthorityState = "unknown"
			}
			role := "declared_owner"
			if head.Ref.Kind == d.EntityKindWorkItem {
				role = "declared_assignee"
			}
			for _, a := range f.Actors {
				actor := a
				v.Participants = append(v.Participants, WorkspacePerson{Role: role, Actor: &actor, Source: "entity_actor_links"})
			}
			if head.Ref.Kind == d.EntityKindOutcome || head.Ref.Kind == d.EntityKindObjective || head.Ref.Kind == d.EntityKindWorkItem {
				counts := WorkspaceCriteriaSummary{}
				for _, c := range f.Criteria {
					if c.Status != d.CriterionStatusActive {
						continue
					}
					counts.Active++
					if c.Required {
						counts.Required++
					}
					if c.AssessmentApplicable && c.AssessmentResult == d.AssessmentResultMet {
						counts.Met++
					} else if c.AssessmentApplicable && c.AssessmentResult == d.AssessmentResultWaived {
						counts.Waived++
					} else {
						counts.Pending++
					}
				}
				v.Criteria = &counts
			}
			if f.PendingReview {
				v.DisplayState = "awaiting_review"
				v.AttentionReason = "awaiting_independent_review"
			} else if f.CorrectionRequired {
				v.DisplayState = "corrections_requested"
				v.AttentionReason = "corrections_requested"
			}
			page.Items = append(page.Items, v)
		}
	}
	sort.Slice(page.Items, func(i, j int) bool {
		return strings.Compare(workspaceCardKey(page.Items[i].Ref), workspaceCardKey(page.Items[j].Ref)) < 0
	})
	page.Coverage.ReturnedCount = workspaceCount(len(page.Items))
	if q.ParentRef == nil {
		page.Coverage.TotalCount = workspaceCount(len(page.Items))
	}
	raw, e := json.Marshal(page)
	if e != nil {
		return zero, e
	}
	if len(raw) > MaxSnapshotBytes {
		return zero, d.NewError(d.ErrorCodeGraphLimitExceeded, "workspace card response exceeds byte limit; reduce the batch")
	}
	return page, nil
}
