package application

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// requireObjectiveIssuesClosed uses owner/lifecycle-filtered reads in the caller's
// guarded write or coherent read transaction. Pending totals include all matches,
// independently of the page. Every owned duplicate is checked because its
// canonical Issue can still be pending. Only ObjectiveID determines ownership.
func requireObjectiveIssuesClosed(ctx context.Context, u ports.UnitOfWork, owner domain.EntityRef) error {
	issues, _, err := issueBlockerRepositories(u)
	if err != nil {
		return err
	}
	pending, count, err := issues.ListPendingIssuesByObjective(ctx, owner.Scope, owner.ID, "", 1)
	if err != nil {
		return err
	}
	if count < 0 || len(pending) > 1 || (count == 0) != (len(pending) == 0) {
		return domain.NewError(domain.ErrorCodeIssue, "inconsistent pending Objective Issue page")
	}
	if count > 0 {
		issue := pending[0]
		if err := validateOwnedIssueCandidate(issue, owner); err != nil {
			return err
		}
		if issue.Lifecycle != domain.IssueLifecycleOpen && issue.Lifecycle != domain.IssueLifecycleInvestigating {
			return domain.NewError(domain.ErrorCodeIssue, "pending Objective Issue page contains a non-pending Issue")
		}
		return objectiveIssuePending(owner, issue, issue)
	}

	const pageSize = 100
	after, seen, total := "", 0, -1
	for {
		page, count, err := issues.ListDuplicateIssuesByObjective(ctx, owner.Scope, owner.ID, after, pageSize)
		if err != nil {
			return err
		}
		if total == -1 {
			total = count
		}
		if count < 0 || count != total || len(page) > pageSize || len(page) > total-seen || (seen < total && len(page) == 0) {
			return domain.NewError(domain.ErrorCodeIssue, "inconsistent duplicate Objective Issue page")
		}
		for _, issue := range page {
			if err := validateOwnedIssueCandidate(issue, owner); err != nil {
				return err
			}
			if issue.Lifecycle != domain.IssueLifecycleDuplicate || issue.ID.String() <= after {
				return domain.NewError(domain.ErrorCodeIssue, "duplicate Objective Issue page is not an ordered duplicate page")
			}
			closed, canonical, err := issueDuplicateChainClosed(ctx, issues, owner.Scope, issue)
			if err != nil {
				return err
			}
			if !closed {
				return objectiveIssuePending(owner, issue, canonical)
			}
			after = issue.ID.String()
		}
		seen += len(page)
		if seen == total {
			return nil
		}
	}
}

func validateOwnedIssueCandidate(issue domain.Issue, owner domain.EntityRef) error {
	if issue.Scope != owner.Scope {
		return domain.NewError(domain.ErrorCodeInvalidScope, "owned Issue does not belong to the Objective scope")
	}
	if issue.ObjectiveID == nil || *issue.ObjectiveID != owner.ID {
		return domain.NewError(domain.ErrorCodeIssue, "Objective Issue page contains an Issue owned elsewhere")
	}
	return issue.Validate()
}

func objectiveIssuePending(owner domain.EntityRef, issue, canonical domain.Issue) error {
	related := []domain.EntityRef{issue.Ref()}
	if canonical.ID != issue.ID {
		related = append(related, canonical.Ref())
	}
	return domain.NewDetailedError(domain.ErrorCodePreconditionFailed, "close the Objective's pending Issues before achieving it", domain.ErrorDetails{
		ReasonCode: "objective_issue_pending", SubjectRef: &owner, RelatedRefs: related,
		Remedies: []domain.ErrorRemedy{{ActionID: "inspect", SubjectRef: refPointer(issue.Ref())}},
	})
}
