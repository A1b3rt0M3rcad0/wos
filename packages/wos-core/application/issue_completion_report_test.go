package application

import (
	"context"
	"reflect"
	"testing"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type countedObjectiveIssueRepository struct {
	ports.IssueRepository
	pendingReads, duplicateReads, chainReads int
}

func (r *countedObjectiveIssueRepository) Get(ctx context.Context, scope d.Scope, id d.ID) (d.Issue, error) {
	r.chainReads++
	return r.IssueRepository.Get(ctx, scope, id)
}
func (r *countedObjectiveIssueRepository) ListPendingIssuesByObjective(ctx context.Context, scope d.Scope, objective d.ID, after string, limit int) ([]d.Issue, int, error) {
	r.pendingReads++
	return r.IssueRepository.ListPendingIssuesByObjective(ctx, scope, objective, after, limit)
}
func (r *countedObjectiveIssueRepository) ListDuplicateIssuesByObjective(ctx context.Context, scope d.Scope, objective d.ID, after string, limit int) ([]d.Issue, int, error) {
	r.duplicateReads++
	return r.IssueRepository.ListDuplicateIssuesByObjective(ctx, scope, objective, after, limit)
}

type objectiveIssueReadUnit struct {
	ports.IssueBlockerUnitOfWork
	ports.WorkProtocolUnitOfWork
	repository ports.IssueRepository
}

func (u objectiveIssueReadUnit) Issues() ports.IssueRepository { return u.repository }

func TestWorkspaceIssueReportReusesDuplicateTraversalForCapabilities(t *testing.T) {
	f := newIssueCompletionFixture(t)
	owner := f.objective("Selected verified condition", nil, true)
	other := f.objective("Canonical issue owner", nil, true)
	canonical := f.close(f.issue(other.ID), d.IssueLifecycleResolved)
	// Persist more than one duplicate page without generating unrelated command
	// history. Every source still has a valid structural owner and canonical ref.
	write, err := f.store.Begin(f.ctx)
	f.must(err)
	issues := write.(ports.IssueBlockerUnitOfWork).Issues()
	for n := 0; n < 105; n++ {
		id, err := f.ids.NewID()
		f.must(err)
		issue, err := d.NewIssueForObjective(id, f.scope, &owner.ID, "Owned duplicate source", "", d.IssueSeverityMinor, nil, canonical.ReportedBy, canonical.UpdatedAt)
		f.must(err)
		f.must(issue.MarkDuplicate(canonical.ID, canonical.UpdatedAt))
		f.must(issues.Insert(f.ctx, issue))
	}
	f.must(write.Commit())

	frame, handled, err := f.s.openWorkspaceFrame(f.ctx, f.scope)
	f.must(err)
	if !handled || frame == nil {
		t.Fatal("workspace snapshot frame unavailable")
	}
	defer frame.u.Rollback()
	counted := &countedObjectiveIssueRepository{IssueRepository: frame.u.(ports.IssueBlockerUnitOfWork).Issues()}
	frame.u = objectiveIssueReadUnit{IssueBlockerUnitOfWork: frame.u.(ports.IssueBlockerUnitOfWork), WorkProtocolUnitOfWork: frame.u.(ports.WorkProtocolUnitOfWork), repository: counted}
	view := EntityWorkspaceView{Subject: WorkspaceSubject{Kind: "entity", Ref: owner.Ref()}, EntityVersion: owner.Version, OutcomeRevision: frame.revision, Coverage: map[string]WorkspaceCoverage{}}
	f.must(f.s.workspaceConclusionImpact(f.ctx, frame, &view, 3))
	if frame.objectiveIssues == nil || !frame.objectiveIssues.complete || view.ConclusionImpact.PendingIssues == nil || *view.ConclusionImpact.PendingIssues != 0 || counted.pendingReads != 1 || counted.duplicateReads != 2 || counted.chainReads != 105 {
		t.Fatalf("preview did not classify both duplicate pages once: report=%+v reads=%+v", frame.objectiveIssues, counted)
	}
	outcome, err := frame.u.Outcomes().Get(f.ctx, f.scope.NamespaceID, f.scope.OutcomeID)
	f.must(err)
	capabilities, err := f.s.workspaceCapabilities(f.ctx, frame, &view, owner, nil, outcome)
	f.must(err)
	view.Capabilities = capabilities
	if workspaceIssueAchieveCapability(t, view).Availability != "available" || counted.pendingReads != 1 || counted.duplicateReads != 2 || counted.chainReads != 105 {
		t.Fatalf("capability repeated an exhaustive Issue traversal: reads=%+v", counted)
	}
	// Other callers can evaluate capabilities without a conclusion report. They
	// must run the same predicate rather than interpreting an absent cache as zero.
	frame.objectiveIssues = nil
	capabilities, err = f.s.workspaceCapabilities(f.ctx, frame, &view, owner, nil, outcome)
	f.must(err)
	view.Capabilities = capabilities
	if workspaceIssueAchieveCapability(t, view).Availability != "available" || counted.pendingReads != 2 || counted.duplicateReads != 4 || counted.chainReads != 210 {
		t.Fatalf("uncached capability did not recheck the full predicate: reads=%+v", counted)
	}
}

func TestWorkspaceIssueReportIncompletePagesOmitExactCountAndPreserveGateReason(t *testing.T) {
	f := newIssueCompletionFixture(t)
	owner := f.objective("Verified owner", nil, true)
	pending := f.issue(owner.ID)
	canonical := f.close(f.issue(owner.ID), d.IssueLifecycleResolved)
	duplicate := f.issue(owner.ID)
	f.must(duplicate.MarkDuplicate(canonical.ID, duplicate.UpdatedAt))
	closedDuplicate := completionIssueRepository{duplicateChainIssues: duplicateChainIssues{canonical.ID: canonical}, candidates: []d.Issue{duplicate}}
	pendingBase := completionIssueRepository{duplicateChainIssues: duplicateChainIssues{}, candidates: []d.Issue{pending}}
	for _, tc := range []struct {
		name string
		repo completionIssuePageRepository
	}{
		{"underfilled_pending_sample", completionIssuePageRepository{completionIssueRepository: pendingBase, pending: func() ([]d.Issue, int, error) { return []d.Issue{pending}, 3, nil }}},
		{"missing_pending_witness", completionIssuePageRepository{completionIssueRepository: pendingBase, pending: func() ([]d.Issue, int, error) { return nil, 1, nil }}},
		{"missing_duplicate_tail", completionIssuePageRepository{completionIssueRepository: closedDuplicate, duplicate: func(after string) ([]d.Issue, int, error) {
			if after == "" {
				return []d.Issue{duplicate}, 2, nil
			}
			return nil, 2, nil
		}}},
		{"changing_duplicate_total", completionIssuePageRepository{completionIssueRepository: closedDuplicate, duplicate: func(after string) ([]d.Issue, int, error) {
			if after == "" {
				return []d.Issue{duplicate}, 2, nil
			}
			return nil, 1, nil
		}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			frame, handled, err := f.s.openWorkspaceFrame(f.ctx, f.scope)
			f.must(err)
			if !handled || frame == nil {
				t.Fatal("workspace snapshot frame unavailable")
			}
			defer frame.u.Rollback()
			frame.u = objectiveIssueReadUnit{IssueBlockerUnitOfWork: frame.u.(ports.IssueBlockerUnitOfWork), WorkProtocolUnitOfWork: frame.u.(ports.WorkProtocolUnitOfWork), repository: tc.repo}
			view := EntityWorkspaceView{Subject: WorkspaceSubject{Kind: "entity", Ref: owner.Ref()}, EntityVersion: owner.Version, OutcomeRevision: frame.revision, Coverage: map[string]WorkspaceCoverage{}}
			f.must(f.s.workspaceConclusionImpact(f.ctx, frame, &view, 3))
			coverage := view.Coverage["conclusion_impact_issues"]
			if view.ConclusionImpact.PendingIssues != nil || coverage.Status != "partial" || coverage.TotalCount != nil || coverage.Reason == "" {
				t.Fatalf("incomplete repository page invented an exact count: impact=%+v coverage=%+v", view.ConclusionImpact, coverage)
			}
			fast := f.s.checkObjectiveAchievement(f.ctx, frame.u, owner, "owner")
			cached := f.s.checkObjectiveAchievementWithIssueReport(f.ctx, frame.u, owner, "owner", frame.objectiveIssues)
			if fast == nil || !reflect.DeepEqual(fast, cached) {
				t.Fatalf("cached preview and command gate diverged: fast=%v cached=%v", fast, cached)
			}
		})
	}
}

func TestObjectiveIssueReportCannotAuthorizeAnotherOwnerOrIncompleteClosure(t *testing.T) {
	f := newIssueCompletionFixture(t)
	owner := f.objective("Selected owner", nil, true)
	other := f.objective("Different owner", nil, true)
	pending := f.issue(owner.ID)
	repo := completionIssueRepository{duplicateChainIssues: duplicateChainIssues{}, candidates: []d.Issue{pending}}
	u := completionIssueUnit{issues: repo}
	wrongOwner := objectiveIssueReport{owner: other.Ref(), complete: true}
	if err := requireObjectiveIssuesClosedWithReport(f.ctx, u, owner.Ref(), &wrongOwner); err == nil {
		t.Fatal("a different owner's complete report bypassed the selected Issue gate")
	}
	incomplete := objectiveIssueReport{owner: owner.Ref()}
	if err := requireObjectiveIssuesClosedWithReport(f.ctx, u, owner.Ref(), &incomplete); err == nil {
		t.Fatal("an incomplete report without a known failure was treated as closure")
	}
}
