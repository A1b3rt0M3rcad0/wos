package application

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

type issueCompletionFixture struct {
	t     *testing.T
	ctx   context.Context
	s     *Service
	store *memory.Store
	ids   *workspaceSnapshotIDs
	scope d.Scope
}

func newIssueCompletionFixture(t *testing.T) *issueCompletionFixture {
	t.Helper()
	f := &issueCompletionFixture{t: t, ctx: context.Background(), store: memory.New(), ids: &workspaceSnapshotIDs{}}
	var err error
	f.s, err = NewService(f.store, workspaceSnapshotClock{time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)}, f.ids)
	f.must(err)
	namespace, err := f.ids.NewID()
	f.must(err)
	out, err := f.s.CreateOutcome(f.ctx, f.cc(), CreateOutcomeCommand{NamespaceID: namespace, Title: "Issue completion", DesiredState: "Accept only a verified condition", Priority: d.PriorityNormal, InitialCriteria: []InitialCriterion{{Title: "Result defined", Required: true, VerificationMode: d.VerificationModeAttestation}}})
	f.must(err)
	f.scope = out.Value.Scope()
	_, err = f.s.ActivateOutcome(f.ctx, f.cc(), ActivateOutcomeCommand{Scope: f.scope, ExpectedVersion: out.Value.Version})
	f.must(err)
	return f
}

func (f *issueCompletionFixture) must(err error) {
	f.t.Helper()
	if err != nil {
		f.t.Fatal(err)
	}
}
func (f *issueCompletionFixture) cc() d.CommandContext {
	f.t.Helper()
	id, err := f.ids.NewID()
	f.must(err)
	return d.CommandContext{CommandID: id, PrincipalID: "owner", Actor: d.ActorRef{Kind: d.ActorKindHuman, Provider: "test", ID: "owner"}}
}
func (f *issueCompletionFixture) objective(title string, parent *d.ID, assessed bool) d.Objective {
	f.t.Helper()
	created, err := f.s.CreateObjective(f.ctx, f.cc(), CreateObjectiveCommand{Scope: f.scope, Title: title, ParentObjectiveID: parent, Priority: d.PriorityNormal, InitialCriteria: []InitialCriterion{{Title: "Verified independently of triage", Required: true, VerificationMode: d.VerificationModeAttestation}}})
	f.must(err)
	if assessed {
		criterion := created.Value.Criteria.Items[0]
		_, err = f.s.RecordCriterionAssessment(f.ctx, f.cc(), RecordCriterionAssessmentCommand{Owner: created.Value.Ref(), ExpectedVersion: created.Value.Version, CriterionID: criterion.ID, CriterionRevision: criterion.Revision, Result: d.AssessmentResultMet, Rationale: "The agreed condition was checked"})
		f.must(err)
	}
	current, err := f.s.GetObjective(f.ctx, f.scope, created.Value.ID)
	f.must(err)
	return current.Value
}
func (f *issueCompletionFixture) issue(owner d.ID, affected ...d.EntityRef) d.Issue {
	f.t.Helper()
	created, err := f.s.CreateIssue(f.ctx, f.cc(), CreateIssueCommand{Scope: f.scope, ObjectiveID: owner, Title: "A scoped problem", Severity: d.IssueSeverityMinor, AffectedRefs: affected})
	f.must(err)
	return created.Value
}
func (f *issueCompletionFixture) close(issue d.Issue, lifecycle d.IssueLifecycle) d.Issue {
	f.t.Helper()
	var result MutationResult[d.Issue]
	var err error
	if lifecycle == d.IssueLifecycleWontFix {
		result, err = f.s.MarkIssueWontFix(f.ctx, f.cc(), MarkIssueWontFixCommand{Scope: f.scope, IssueID: issue.ID, ExpectedVersion: issue.Version, ResolutionSummary: "Explicitly accepted triage; this is not criterion proof"})
	} else {
		result, err = f.s.ResolveIssue(f.ctx, f.cc(), ResolveIssueCommand{Scope: f.scope, IssueID: issue.ID, ExpectedVersion: issue.Version, ResolutionSummary: "The problem was resolved"})
	}
	f.must(err)
	return result.Value
}
func (f *issueCompletionFixture) achieve(owner d.Objective) error {
	f.t.Helper()
	_, err := f.s.AchieveObjective(f.ctx, f.cc(), AchieveObjectiveCommand{Scope: f.scope, ObjectiveID: owner.ID, ExpectedVersion: owner.Version, Reason: "Explicitly accept the verified condition"})
	return err
}

func TestObjectiveIssueCompletionLifecycleAndCanonicalMatrix(t *testing.T) {
	for _, tc := range []struct {
		name      string
		lifecycle d.IssueLifecycle
		canonical d.IssueLifecycle
		allowed   bool
	}{
		{"open", d.IssueLifecycleOpen, "", false}, {"investigating", d.IssueLifecycleInvestigating, "", false},
		{"resolved", d.IssueLifecycleResolved, "", true}, {"wont_fix", d.IssueLifecycleWontFix, "", true},
		{"duplicate_open", d.IssueLifecycleDuplicate, d.IssueLifecycleOpen, false},
		{"duplicate_resolved", d.IssueLifecycleDuplicate, d.IssueLifecycleResolved, true},
		{"duplicate_wont_fix", d.IssueLifecycleDuplicate, d.IssueLifecycleWontFix, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newIssueCompletionFixture(t)
			owner := f.objective("Owner", nil, true)
			issue := f.issue(owner.ID)
			switch tc.lifecycle {
			case d.IssueLifecycleInvestigating:
				r, err := f.s.StartIssueInvestigation(f.ctx, f.cc(), StartIssueInvestigationCommand{Scope: f.scope, IssueID: issue.ID, ExpectedVersion: issue.Version})
				f.must(err)
				issue = r.Value
			case d.IssueLifecycleResolved, d.IssueLifecycleWontFix:
				issue = f.close(issue, tc.lifecycle)
			case d.IssueLifecycleDuplicate:
				other := f.objective("Canonical owner", nil, true)
				canonical := f.issue(other.ID)
				if tc.canonical != d.IssueLifecycleOpen {
					canonical = f.close(canonical, tc.canonical)
				}
				// Two hops, crossing Objective ownership, still belong to the
				// selected source Issue's owner for this gate.
				middle := f.issue(other.ID)
				_, err := f.s.MarkIssueDuplicate(f.ctx, f.cc(), MarkIssueDuplicateCommand{Scope: f.scope, IssueID: middle.ID, ExpectedVersion: middle.Version, DuplicateOfID: canonical.ID})
				f.must(err)
				r, err := f.s.MarkIssueDuplicate(f.ctx, f.cc(), MarkIssueDuplicateCommand{Scope: f.scope, IssueID: issue.ID, ExpectedVersion: issue.Version, DuplicateOfID: middle.ID})
				f.must(err)
				issue = r.Value
			}
			before, err := f.s.GetOutcomeState(f.ctx, f.scope)
			f.must(err)
			events := f.store.SnapshotDomainEvents(f.scope)
			view, err := f.s.GetEntityWorkspaceView(f.ctx, f.scope, WorkspaceViewQuery{Subject: WorkspaceSubject{Kind: "entity", Ref: owner.Ref()}})
			f.must(err)
			var capability *WorkspaceCapability
			for i := range view.Capabilities {
				if view.Capabilities[i].ActionID == "AchieveObjective" {
					capability = &view.Capabilities[i]
				}
			}
			if capability == nil || (capability.Availability == "available") != tc.allowed {
				t.Fatalf("capability does not share Issue gate: %+v", capability)
			}
			err = f.achieve(owner)
			if tc.allowed {
				f.must(err)
				unchanged, e := f.s.GetIssue(f.ctx, f.scope, issue.ID)
				f.must(e)
				if !reflect.DeepEqual(unchanged.Value, issue) {
					t.Fatal("achievement changed Issue disposition")
				}
				return
			}
			var coded *d.Error
			if !errors.As(err, &coded) || coded.Code != d.ErrorCodePreconditionFailed || coded.Details == nil || coded.Details.ReasonCode != "objective_issue_pending" || coded.Details.SubjectRef == nil || *coded.Details.SubjectRef != owner.Ref() || len(coded.Details.RelatedRefs) == 0 || coded.Details.RelatedRefs[0] != issue.Ref() {
				t.Fatalf("missing structured owner gate: %v", err)
			}
			after, e := f.s.GetOutcomeState(f.ctx, f.scope)
			f.must(e)
			if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(events, f.store.SnapshotDomainEvents(f.scope)) {
				t.Fatal("rejected achievement mutated state, versions, revision or events")
			}
		})
	}
}

func TestObjectiveIssueCompletionUsesOnlyStructuralOwner(t *testing.T) {
	f := newIssueCompletionFixture(t)
	parent := f.objective("Parent", nil, true)
	child := f.objective("Child", &parent.ID, true)
	other := f.objective("Other", nil, true)
	f.issue(child.ID, parent.Ref())
	f.issue(other.ID, parent.Ref())
	owned := f.issue(parent.ID, other.Ref())
	if err := f.achieve(parent); err == nil {
		t.Fatal("owned Issue was ignored because affected_refs points elsewhere")
	}
	f.close(owned, d.IssueLifecycleResolved)
	f.must(f.achieve(parent))
	// Compatible structural completion does not inherit every descendant Issue.
	current, err := f.s.GetObjective(f.ctx, f.scope, child.ID)
	f.must(err)
	if current.Value.Lifecycle != child.Lifecycle {
		t.Fatal("parent achievement changed child lifecycle")
	}
}

func TestObjectiveIssueCompletionDoesNotUseWontFixAsCriterionProof(t *testing.T) {
	f := newIssueCompletionFixture(t)
	owner := f.objective("Unverified condition", nil, false)
	f.close(f.issue(owner.ID), d.IssueLifecycleWontFix)
	before, err := f.s.GetOutcomeState(f.ctx, f.scope)
	f.must(err)
	events := f.store.SnapshotDomainEvents(f.scope)
	if err := f.achieve(owner); err == nil {
		t.Fatal("triage substituted for criterion assessment")
	}
	after, err := f.s.GetOutcomeState(f.ctx, f.scope)
	f.must(err)
	if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(events, f.store.SnapshotDomainEvents(f.scope)) {
		t.Fatal("failed criterion gate changed state")
	}
}

func TestObjectiveIssueCompletionInvalidPersistedChainDoesNotMutate(t *testing.T) {
	f := newIssueCompletionFixture(t)
	owner := f.objective("Owner with legacy corruption", nil, true)
	other := f.objective("Other canonical owner", nil, true)
	first, second := f.issue(owner.ID), f.issue(other.ID)
	// Model a corrupt persisted cycle. The public duplicate command rejects
	// this graph, but the achievement gate must still fail closed on old data.
	f.must(first.MarkDuplicate(second.ID, first.UpdatedAt))
	f.must(second.MarkDuplicate(first.ID, second.UpdatedAt))
	u, err := f.store.Begin(f.ctx)
	f.must(err)
	defer u.Rollback()
	r := u.(ports.IssueBlockerUnitOfWork).Issues()
	f.must(r.Save(f.ctx, first, first.Version-1))
	f.must(r.Save(f.ctx, second, second.Version-1))
	f.must(u.Commit())
	before, err := f.s.GetOutcomeState(f.ctx, f.scope)
	f.must(err)
	events := f.store.SnapshotDomainEvents(f.scope)
	if err := f.achieve(owner); err == nil {
		t.Fatal("corrupt persisted duplicate cycle permitted achievement")
	}
	after, err := f.s.GetOutcomeState(f.ctx, f.scope)
	f.must(err)
	if !reflect.DeepEqual(before, after) || !reflect.DeepEqual(events, f.store.SnapshotDomainEvents(f.scope)) {
		t.Fatal("invalid duplicate chain changed state, versions, revision or events")
	}
}

type completionIssueRepository struct {
	duplicateChainIssues
	candidates []d.Issue
}

func (r completionIssueRepository) ListByOutcome(context.Context, d.Scope) ([]d.Issue, error) {
	return nil, errors.New("the achievement gate must not scan all Outcome Issues")
}

func (r completionIssueRepository) ListPendingIssuesByObjective(_ context.Context, scope d.Scope, objective d.ID, after string, limit int) ([]d.Issue, int, error) {
	return r.ownerPage(scope, objective, after, limit, d.IssueLifecycleOpen, d.IssueLifecycleInvestigating)
}

func (r completionIssueRepository) ListDuplicateIssuesByObjective(_ context.Context, scope d.Scope, objective d.ID, after string, limit int) ([]d.Issue, int, error) {
	return r.ownerPage(scope, objective, after, limit, d.IssueLifecycleDuplicate)
}

func (r completionIssueRepository) ownerPage(scope d.Scope, objective d.ID, after string, limit int, lifecycle ...d.IssueLifecycle) ([]d.Issue, int, error) {
	var page []d.Issue
	total := 0
	for _, issue := range r.candidates {
		if issue.Scope != scope || issue.ObjectiveID == nil || *issue.ObjectiveID != objective {
			continue
		}
		for _, state := range lifecycle {
			if issue.Lifecycle == state {
				total++
				if issue.ID.String() > after {
					page = append(page, issue)
				}
				break
			}
		}
	}
	sort.Slice(page, func(i, j int) bool { return page[i].ID < page[j].ID })
	if len(page) > limit {
		page = page[:limit]
	}
	return page, total, nil
}

type completionIssueUnit struct {
	ports.UnitOfWork
	issues ports.IssueRepository
}

func (u completionIssueUnit) Issues() ports.IssueRepository     { return u.issues }
func (u completionIssueUnit) Blockers() ports.BlockerRepository { return nil }

func TestObjectiveIssueCompletionFailsClosedForInvalidDuplicateChains(t *testing.T) {
	scope := d.Scope{NamespaceID: d.MustParseID("0199f222-0000-7000-8000-000000000001"), OutcomeID: d.MustParseID("0199f222-0000-7000-8000-000000000002")}
	owner := d.EntityRef{Scope: scope, Kind: d.EntityKindObjective, ID: d.MustParseID("0199f222-0000-7000-8000-000000000003")}
	key := func(n int) d.ID { return d.MustParseID(fmt.Sprintf("0199f223-0000-7000-8000-%012x", n)) }
	for _, kind := range []string{"missing", "cycle", "inconsistent", "over_limit"} {
		t.Run(kind, func(t *testing.T) {
			first := duplicateChainIssue(t, scope, key(1), owner.ID)
			next := key(2)
			first.Lifecycle = d.IssueLifecycleDuplicate
			first.DuplicateOfIssueID = &next
			repo := completionIssueRepository{duplicateChainIssues: duplicateChainIssues{first.ID: first}, candidates: []d.Issue{first}}
			second := duplicateChainIssue(t, scope, next, owner.ID)
			switch kind {
			case "cycle":
				second.Lifecycle = d.IssueLifecycleDuplicate
				second.DuplicateOfIssueID = &first.ID
				repo.duplicateChainIssues[next] = second
			case "inconsistent":
				second.DuplicateOfIssueID = &first.ID
				repo.duplicateChainIssues[next] = second
			case "over_limit":
				for index := 2; index <= maxIssueDuplicateDepth+2; index++ {
					issue := duplicateChainIssue(t, scope, key(index), owner.ID)
					target := key(index + 1)
					issue.Lifecycle = d.IssueLifecycleDuplicate
					issue.DuplicateOfIssueID = &target
					repo.duplicateChainIssues[issue.ID] = issue
				}
			}
			if err := requireObjectiveIssuesClosed(context.Background(), completionIssueUnit{issues: repo}, owner); err == nil {
				t.Fatal("invalid duplicate chain permitted achievement")
			}
		})
	}
}

func TestObjectiveIssueCompletionChecksEveryOwnedDuplicateBeyondFirstPage(t *testing.T) {
	for _, closed := range []bool{false, true} {
		t.Run(fmt.Sprintf("last_canonical_closed_%v", closed), func(t *testing.T) {
			scope := d.Scope{NamespaceID: d.MustParseID("0199f224-0000-7000-8000-000000000001"), OutcomeID: d.MustParseID("0199f224-0000-7000-8000-000000000002")}
			owner := d.EntityRef{Scope: scope, Kind: d.EntityKindObjective, ID: d.MustParseID("0199f224-0000-7000-8000-000000000003")}
			other := d.MustParseID("0199f224-0000-7000-8000-000000000004")
			key := func(n int) d.ID { return d.MustParseID(fmt.Sprintf("0199f225-0000-7000-8000-%012x", n)) }
			repo := completionIssueRepository{duplicateChainIssues: duplicateChainIssues{}}
			resolved := duplicateChainIssue(t, scope, key(200), other)
			if err := resolved.Resolve("Canonical closed independently", resolved.UpdatedAt); err != nil {
				t.Fatal(err)
			}
			repo.duplicateChainIssues[resolved.ID] = resolved
			for index := 1; index <= 105; index++ {
				issue := duplicateChainIssue(t, scope, key(index), owner.ID)
				if err := issue.MarkDuplicate(resolved.ID, issue.UpdatedAt); err != nil {
					t.Fatal(err)
				}
				repo.candidates = append(repo.candidates, issue)
			}
			canonical := duplicateChainIssue(t, scope, key(201), other)
			if closed {
				if err := canonical.Resolve("Last canonical resolved", canonical.UpdatedAt); err != nil {
					t.Fatal(err)
				}
			}
			duplicate := duplicateChainIssue(t, scope, key(106), owner.ID)
			if err := duplicate.MarkDuplicate(canonical.ID, duplicate.UpdatedAt); err != nil {
				t.Fatal(err)
			}
			repo.candidates = append(repo.candidates, duplicate)
			repo.duplicateChainIssues[canonical.ID] = canonical
			err := requireObjectiveIssuesClosed(context.Background(), completionIssueUnit{issues: repo}, owner)
			if closed {
				if err != nil {
					t.Fatalf("complete closed duplicate pages prevented achievement: %v", err)
				}
				return
			}
			var coded *d.Error
			if !errors.As(err, &coded) || coded.Details == nil || coded.Details.ReasonCode != "objective_issue_pending" || len(coded.Details.RelatedRefs) == 0 || coded.Details.RelatedRefs[0] != duplicate.Ref() {
				t.Fatalf("a duplicate beyond the first page bypassed the gate: %v", err)
			}
		})
	}
}

type completionIssuePageRepository struct {
	completionIssueRepository
	pending   func() ([]d.Issue, int, error)
	duplicate func(after string) ([]d.Issue, int, error)
}

func (r completionIssuePageRepository) ListPendingIssuesByObjective(ctx context.Context, scope d.Scope, objective d.ID, after string, limit int) ([]d.Issue, int, error) {
	if r.pending != nil {
		return r.pending()
	}
	return r.completionIssueRepository.ListPendingIssuesByObjective(ctx, scope, objective, after, limit)
}

func (r completionIssuePageRepository) ListDuplicateIssuesByObjective(ctx context.Context, scope d.Scope, objective d.ID, after string, limit int) ([]d.Issue, int, error) {
	if r.duplicate != nil {
		return r.duplicate(after)
	}
	return r.completionIssueRepository.ListDuplicateIssuesByObjective(ctx, scope, objective, after, limit)
}

func TestObjectiveIssueCompletionDoesNotTreatIncompleteOrInconsistentPagesAsClosed(t *testing.T) {
	scope := d.Scope{NamespaceID: d.MustParseID("0199f226-0000-7000-8000-000000000001"), OutcomeID: d.MustParseID("0199f226-0000-7000-8000-000000000002")}
	owner := d.EntityRef{Scope: scope, Kind: d.EntityKindObjective, ID: d.MustParseID("0199f226-0000-7000-8000-000000000003")}
	canonical := duplicateChainIssue(t, scope, d.MustParseID("0199f226-0000-7000-8000-000000000005"), owner.ID)
	if err := canonical.Resolve("Closed canonical", canonical.UpdatedAt); err != nil {
		t.Fatal(err)
	}
	duplicate := duplicateChainIssue(t, scope, d.MustParseID("0199f226-0000-7000-8000-000000000004"), owner.ID)
	if err := duplicate.MarkDuplicate(canonical.ID, duplicate.UpdatedAt); err != nil {
		t.Fatal(err)
	}
	base := completionIssueRepository{duplicateChainIssues: duplicateChainIssues{canonical.ID: canonical}}
	for _, tc := range []struct {
		name string
		repo completionIssuePageRepository
	}{
		{"missing_pending_witness", completionIssuePageRepository{completionIssueRepository: base, pending: func() ([]d.Issue, int, error) { return nil, 1, nil }}},
		{"missing_duplicate_tail", completionIssuePageRepository{completionIssueRepository: base, duplicate: func(after string) ([]d.Issue, int, error) {
			if after == "" {
				return []d.Issue{duplicate}, 2, nil
			}
			return nil, 2, nil
		}}},
		{"repeated_duplicate_cursor", completionIssuePageRepository{completionIssueRepository: base, duplicate: func(string) ([]d.Issue, int, error) { return []d.Issue{duplicate}, 2, nil }}},
		{"changed_duplicate_total", completionIssuePageRepository{completionIssueRepository: base, duplicate: func(after string) ([]d.Issue, int, error) {
			if after == "" {
				return []d.Issue{duplicate}, 2, nil
			}
			return nil, 1, nil
		}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if err := requireObjectiveIssuesClosed(context.Background(), completionIssueUnit{issues: tc.repo}, owner); err == nil {
				t.Fatal("incomplete or inconsistent owner pages permitted achievement")
			}
		})
	}
}
