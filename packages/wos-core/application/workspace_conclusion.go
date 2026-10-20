package application

import (
	"context"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// Compatible impact remains advisory; strict-work adds explicit operational gates.
func (s *Service) workspaceConclusionImpact(ctx context.Context, f *continuityReadFrame, v *EntityWorkspaceView, limit int) error {
	facts, err := f.repo.(ports.WorkspaceReadRepository).WorkspaceConclusionImpact(ctx, v.Subject.Ref.Scope, v.Subject.Ref, limit)
	if err != nil {
		return err
	}
	impact := &WorkspaceConclusionImpact{OpenObjectives: facts.OpenObjectives, OpenTasks: facts.OpenWorkItems,
		OpenItems: []EntitySummary{}, Consequences: []string{"child_lifecycles_unchanged", "child_proof_unchanged"}}
	for _, item := range facts.Items {
		impact.OpenItems = append(impact.OpenItems, summary(item.Ref, item.Version, item.Title, item.Lifecycle, item.Priority))
	}
	if v.Subject.Ref.Kind == d.EntityKindOutcome {
		impact.Consequences = append(impact.Consequences, "outcome_operational_scope_closed")
	} else {
		impact.Consequences = append(impact.Consequences, "objective_closed_to_new_children", "descendant_workflow_unchanged")
	}
	parent, err := completionOutcome(ctx, f.u, v.Subject.Ref.Scope)
	if err != nil {
		return err
	}
	impact.CompletionPolicy = parent.CompletionPolicy.Effective()
	if impact.CompletionPolicy == d.CompletionPolicyStrictWork {
		open, cancelled, e := strictCompletionWork(ctx, f.u, v.Subject.Ref)
		if e != nil {
			return e
		}
		impact.RequiredOpenWork = open
		if len(cancelled) > 0 {
			for start := 0; start < len(cancelled); start += 100 {
				nodes, e := f.repo.(ports.GraphReadRepository).GraphNodes(ctx, v.Subject.Ref.Scope, cancelled[start:min(start+100, len(cancelled))])
				if e != nil {
					return e
				}
				for _, item := range nodes {
					impact.CancelledItems = append(impact.CancelledItems, summary(item.Ref, item.Version, item.Title, item.Lifecycle, item.Priority))
				}
			}
		}
	}
	if v.Subject.Ref.Kind == d.EntityKindObjective {
		report := evaluateObjectiveIssues(ctx, f.u, v.Subject.Ref, limit, true)
		if err := ctx.Err(); err != nil {
			return err
		}
		f.objectiveIssues = &report
		impact.PendingIssueRefs = append([]d.EntityRef(nil), report.pendingRefs...)
		coverage := WorkspaceCoverage{Status: "partial", ReturnedCount: workspaceCount(len(report.pendingRefs)), Reason: "objective_issue_check_incomplete"}
		if report.complete {
			impact.PendingIssues = workspaceCount(report.pendingCount)
			coverage.TotalCount = workspaceCount(report.pendingCount)
			coverage.Status = "complete"
			coverage.Reason = ""
			if len(report.pendingRefs) < report.pendingCount {
				coverage.Status = "partial"
				coverage.Reason = "bounded_pending_issue_sample; inspect owned Issues and their canonical targets"
			}
		}
		v.Coverage["conclusion_impact_issues"] = coverage
	}
	v.ConclusionImpact = impact
	total := facts.OpenObjectives + facts.OpenWorkItems
	coverage := WorkspaceCoverage{Status: "complete", ReturnedCount: workspaceCount(len(impact.OpenItems)), TotalCount: workspaceCount(total)}
	if len(impact.OpenItems) < total {
		coverage.Status = "partial"
		coverage.Reason = "bounded_open_descendant_sample; use Plan and Work to inspect remaining items"
	}
	v.Coverage["conclusion_impact"] = coverage
	return nil
}

// Legacy callers may omit this additive preview check. When supplied, a human's
// impact review is pinned to the same guarded state that is about to conclude.
// Canonical current proof/preparation checks run first so their remedies remain
// actionable; this never refreshes the caller's frozen intention automatically.
func validateConclusionImpactPreview(ctx context.Context, u ports.UnitOfWork, subject d.EntityRef, observed *d.OutcomeRevision) error {
	if observed == nil {
		return nil
	}
	coordination, err := u.Coordination().LockOutcome(ctx, subject.Scope)
	if err != nil {
		return err
	}
	if coordination.Revision == *observed {
		return nil
	}
	return d.NewDetailedError(d.ErrorCodePreconditionFailed, "the Outcome changed after the conclusion impact review; review current open work before confirming again", d.ErrorDetails{
		ReasonCode: "conclusion_impact_changed", SubjectRef: &subject, FieldPaths: []string{"observed_outcome_revision"},
		Remedies: []d.ErrorRemedy{{ActionID: "review_conclusion_impact", SubjectRef: &subject}},
	})
}
