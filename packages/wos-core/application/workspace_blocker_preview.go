package application

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type WorkspaceBlockerPreviewQuery struct {
	Propagation d.BlockerPropagation `json:"propagation"`
}

type WorkspaceBlockerPreview struct {
	Propagation        d.BlockerPropagation `json:"propagation"`
	TargetRef          d.EntityRef          `json:"target_ref"`
	TargetIncluded     bool                 `json:"target_included"`
	AffectedOutcomes   int                  `json:"affected_outcomes"`
	AffectedObjectives int                  `json:"affected_objectives"`
	AffectedTasks      int                  `json:"affected_tasks"`
	AffectedItems      []EntitySummary      `json:"affected_items"`
}

// This opt-in projection observes the existing canonical block scope. It never
// creates an Issue/Blocker or changes a child's lifecycle, proof or authority.
func (s *Service) workspaceBlockerPreview(ctx context.Context, f *continuityReadFrame, v *EntityWorkspaceView, selected any, outcome d.Outcome, q *WorkspaceBlockerPreviewQuery) (*WorkspaceBlockerPreview, error) {
	if v.Subject.Kind != "entity" {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "blocker preview requires an entity target")
	}
	ref := v.Subject.Ref
	if !d.BlockerPropagationAllowed(ref.Kind, q.Propagation) {
		return nil, d.NewDetailedError(d.ErrorCodeBlocker, "propagation is not supported for the blocker target", d.ErrorDetails{ReasonCode: "invalid_blocker_propagation", SubjectRef: &ref, FieldPaths: []string{"blocker_preview.propagation"}})
	}
	if err := validateOpenOutcome(outcome); err != nil {
		return nil, err
	}
	var priority d.Priority
	switch value := selected.(type) {
	case d.Outcome:
		priority = value.Priority
	case d.Objective:
		priority = value.Priority
	case d.WorkItem:
		priority = value.Priority
	default:
		return nil, d.NewError(d.ErrorCodeBlocker, "unsupported blocker target kind")
	}
	if err := validateBlockerTargetState(ref, v.Header.Lifecycle, ref.Kind == d.EntityKindOutcome && outcome.IsArchived()); err != nil {
		return nil, err
	}
	preview := &WorkspaceBlockerPreview{Propagation: q.Propagation, TargetRef: ref, TargetIncluded: true, AffectedItems: []EntitySummary{summary(ref, v.EntityVersion, v.Header.Title, v.Header.Lifecycle, priority)}}
	switch ref.Kind {
	case d.EntityKindOutcome:
		preview.AffectedOutcomes = 1
	case d.EntityKindObjective:
		preview.AffectedObjectives = 1
	case d.EntityKindWorkItem:
		preview.AffectedTasks = 1
	}
	if q.Propagation == d.BlockerPropagationSubtree {
		facts, err := f.repo.(ports.WorkspaceReadRepository).WorkspaceConclusionImpact(ctx, ref.Scope, ref, 20)
		if err != nil {
			return nil, err
		}
		preview.AffectedObjectives += facts.OpenObjectives
		preview.AffectedTasks += facts.OpenWorkItems
		// The explicit target occupies one slot; the remaining sample is bounded.
		for _, item := range facts.Items[:min(19, len(facts.Items))] {
			preview.AffectedItems = append(preview.AffectedItems, summary(item.Ref, item.Version, item.Title, item.Lifecycle, item.Priority))
		}
	}
	total := preview.AffectedOutcomes + preview.AffectedObjectives + preview.AffectedTasks
	coverage := WorkspaceCoverage{Status: "complete", ReturnedCount: workspaceCount(len(preview.AffectedItems)), TotalCount: workspaceCount(total)}
	if len(preview.AffectedItems) < total {
		coverage.Status = "partial"
		coverage.Reason = "bounded_open_blocker_scope_sample; use Plan and Work to inspect remaining items"
	}
	v.Coverage["blocker_preview"] = coverage
	return preview, nil
}
