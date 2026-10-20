package application

import (
	"context"
	"sort"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// The report is scoped to one Objective in the caller's transaction. A preview
// exhaustively classifies owned duplicate sources while retaining a bounded
// sample. Commands may stop at the first failed precondition. Both preserve the
// same first gate error; incomplete classification never establishes closure.
type objectiveIssueReport struct {
	owner        domain.EntityRef
	pendingCount int
	pendingRefs  []domain.EntityRef
	complete     bool
	gateError    error
}

func (r objectiveIssueReport) requireClosed() error {
	if r.gateError != nil {
		return r.gateError
	}
	if !r.complete || r.pendingCount != 0 {
		return domain.NewError(domain.ErrorCodeIssue, "Objective Issue closure check is incomplete")
	}
	return nil
}

func requireObjectiveIssuesClosed(ctx context.Context, u ports.UnitOfWork, owner domain.EntityRef) error {
	return requireObjectiveIssuesClosedWithReport(ctx, u, owner, nil)
}

func requireObjectiveIssuesClosedWithReport(ctx context.Context, u ports.UnitOfWork, owner domain.EntityRef, report *objectiveIssueReport) error {
	if report == nil || report.owner != owner {
		current := evaluateObjectiveIssues(ctx, u, owner, 1, false)
		report = &current
	}
	return report.requireClosed()
}

// Repository reads are filtered by structural owner and lifecycle. Pending
// totals include all matches independently of the page; every duplicate source
// must be classified before the pending count or closure can be called complete.
func evaluateObjectiveIssues(ctx context.Context, u ports.UnitOfWork, owner domain.EntityRef, sampleLimit int, exhaustive bool) objectiveIssueReport {
	report := objectiveIssueReport{owner: owner}
	fail := func(err error) objectiveIssueReport {
		if report.gateError == nil {
			report.gateError = err
		}
		return report
	}
	issues, _, err := issueBlockerRepositories(u)
	if err != nil {
		return fail(err)
	}
	sampleLimit = max(1, min(sampleLimit, 20))
	pending, count, err := issues.ListPendingIssuesByObjective(ctx, owner.Scope, owner.ID, "", sampleLimit)
	if err != nil {
		return fail(err)
	}
	if count < 0 || len(pending) > sampleLimit || len(pending) > count || (count == 0) != (len(pending) == 0) {
		return fail(domain.NewError(domain.ErrorCodeIssue, "inconsistent pending Objective Issue page"))
	}
	report.pendingCount = count
	afterPending := ""
	for _, issue := range pending {
		if err := validateOwnedIssueCandidate(issue, owner); err != nil {
			return fail(err)
		}
		if (issue.Lifecycle != domain.IssueLifecycleOpen && issue.Lifecycle != domain.IssueLifecycleInvestigating) || issue.ID.String() <= afterPending {
			return fail(domain.NewError(domain.ErrorCodeIssue, "pending Objective Issue page is not an ordered pending page"))
		}
		if report.gateError == nil {
			report.gateError = objectiveIssuePending(owner, issue, issue)
		}
		report.addPendingSample(issue.Ref(), sampleLimit)
		afterPending = issue.ID.String()
		if !exhaustive {
			return report
		}
	}
	if len(pending) != min(count, sampleLimit) {
		return fail(domain.NewError(domain.ErrorCodeIssue, "incomplete pending Objective Issue sample"))
	}

	const pageSize = 100
	after, seen, total := "", 0, -1
	for {
		page, count, err := issues.ListDuplicateIssuesByObjective(ctx, owner.Scope, owner.ID, after, pageSize)
		if err != nil {
			return fail(err)
		}
		if total == -1 {
			total = count
		}
		if count < 0 || count != total || len(page) > pageSize || len(page) > total-seen || (seen < total && len(page) == 0) {
			return fail(domain.NewError(domain.ErrorCodeIssue, "inconsistent duplicate Objective Issue page"))
		}
		for _, issue := range page {
			if err := validateOwnedIssueCandidate(issue, owner); err != nil {
				return fail(err)
			}
			if issue.Lifecycle != domain.IssueLifecycleDuplicate || issue.ID.String() <= after {
				return fail(domain.NewError(domain.ErrorCodeIssue, "duplicate Objective Issue page is not an ordered duplicate page"))
			}
			closed, canonical, err := issueDuplicateChainClosed(ctx, issues, owner.Scope, issue)
			if err != nil {
				return fail(err)
			}
			if !closed {
				report.pendingCount++
				if report.gateError == nil {
					report.gateError = objectiveIssuePending(owner, issue, canonical)
				}
				report.addPendingSample(issue.Ref(), sampleLimit)
				if !exhaustive {
					return report
				}
			}
			after = issue.ID.String()
		}
		seen += len(page)
		if seen == total {
			report.complete = true
			return report
		}
	}
}

func (r *objectiveIssueReport) addPendingSample(ref domain.EntityRef, limit int) {
	index := sort.Search(len(r.pendingRefs), func(i int) bool { return r.pendingRefs[i].ID >= ref.ID })
	if index >= limit || (index < len(r.pendingRefs) && r.pendingRefs[index] == ref) {
		return
	}
	if len(r.pendingRefs) < limit {
		r.pendingRefs = append(r.pendingRefs, domain.EntityRef{})
	}
	copy(r.pendingRefs[index+1:], r.pendingRefs[index:])
	r.pendingRefs[index] = ref
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
