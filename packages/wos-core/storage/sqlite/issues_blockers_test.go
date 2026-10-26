package sqlite

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

func TestListDuplicateIssuesByObjectiveUsesStableScopedPages(t *testing.T) {
	for _, backend := range []string{"memory", "sql"} {
		t.Run(backend, func(t *testing.T) {
			ctx := context.Background()
			var store ports.TransactionManager
			if backend == "memory" {
				store = memory.New()
			} else {
				store = openTestStore(t, filepath.Join(t.TempDir(), "duplicate-issues.db"))
			}
			must := func(err error) {
				t.Helper()
				if err != nil {
					t.Fatal(err)
				}
			}
			id := func(n int) domain.ID { return domain.MustParseID(fmt.Sprintf("0199f160-0000-7000-8000-%012x", n)) }
			now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
			scopes := []domain.Scope{{NamespaceID: id(1), OutcomeID: id(2)}, {NamespaceID: id(1), OutcomeID: id(3)}, {NamespaceID: id(4), OutcomeID: id(5)}}
			write, err := store.Begin(ctx)
			must(err)
			var outcomes []domain.Outcome
			var owners []domain.Objective
			for i, scope := range scopes {
				outcome, err := domain.NewOutcome(scope.OutcomeID, scope.NamespaceID, "Scoped duplicate Issues", "", "Canonical issues must be checked independently", domain.PriorityNormal, now)
				must(err)
				must(write.Outcomes().Insert(ctx, outcome))
				owner, err := domain.NewObjective(id(10+i), scope, "Issue owner", "", domain.PriorityNormal, false, now)
				must(err)
				must(write.Objectives().Insert(ctx, owner))
				outcomes, owners = append(outcomes, outcome), append(owners, owner)
			}
			otherOwner, err := domain.NewObjective(id(13), scopes[0], "Other owner", "", domain.PriorityNormal, false, now)
			must(err)
			must(write.Objectives().Insert(ctx, otherOwner))
			actor := domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "test", ID: "reporter"}
			newIssue := func(n int, owner domain.Objective) domain.Issue {
				t.Helper()
				issue, err := domain.NewIssueForObjective(id(n), owner.Scope, &owner.ID, "Operational issue", "", domain.IssueSeverityMinor, nil, actor, now)
				must(err)
				return issue
			}
			repo := write.(ports.IssueBlockerUnitOfWork).Issues()
			var canonicals []domain.Issue
			for i, owner := range owners {
				canonical := newIssue(90+i, owner)
				must(canonical.Resolve("Reviewed canonical resolution", now.Add(time.Minute)))
				must(repo.Insert(ctx, canonical))
				canonicals = append(canonicals, canonical)
			}
			first, second := newIssue(20, owners[0]), newIssue(40, owners[0])
			first.AffectedRefs = []domain.EntityRef{owners[0].Ref(), outcomes[0].Ref()}
			assignee := domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "duplicate-investigator"}
			first.AssigneeRef = &assignee
			must(first.MarkDuplicate(canonicals[0].ID, now.Add(time.Minute)))
			must(second.MarkDuplicate(canonicals[0].ID, now.Add(time.Minute)))
			open, investigating := newIssue(21, owners[0]), newIssue(22, owners[0])
			must(investigating.StartInvestigationWithAssignee(assignee, now.Add(time.Minute)))
			resolved, wontFix := newIssue(23, owners[0]), newIssue(24, owners[0])
			must(resolved.Resolve("Resolved outside duplicate queue", now.Add(time.Minute)))
			must(wontFix.MarkWontFix("Explicit triage decision", now.Add(time.Minute)))
			other := newIssue(50, otherOwner)
			must(other.MarkDuplicate(canonicals[0].ID, now.Add(time.Minute)))
			foreignOutcome, foreignNamespace := newIssue(60, owners[1]), newIssue(70, owners[2])
			must(foreignOutcome.MarkDuplicate(canonicals[1].ID, now.Add(time.Minute)))
			must(foreignNamespace.MarkDuplicate(canonicals[2].ID, now.Add(time.Minute)))
			// Insert in deliberately unsorted order; lifecycle and owner determine
			// membership independently of the canonical target and affected refs.
			for _, issue := range []domain.Issue{second, open, foreignNamespace, resolved, first, other, investigating, wontFix, foreignOutcome} {
				must(repo.Insert(ctx, issue))
			}
			must(write.Commit())
			read, err := store.Begin(ctx)
			must(err)
			defer read.Rollback()
			repo = read.(ports.IssueBlockerUnitOfWork).Issues()
			page, total, err := repo.ListDuplicateIssuesByObjective(ctx, scopes[0], owners[0].ID, "", 1)
			must(err)
			if len(page) != 1 || total != 2 || page[0].ID != first.ID || page[0].Lifecycle != domain.IssueLifecycleDuplicate || page[0].DuplicateOfIssueID == nil || *page[0].DuplicateOfIssueID != canonicals[0].ID {
				t.Fatalf("first duplicate page lost order/count/canonical target: %#v, total=%d", page, total)
			}
			if !reflect.DeepEqual(page[0].AffectedRefs, first.AffectedRefs) || page[0].AssigneeRef == nil || *page[0].AssigneeRef != assignee || page[0].ReportedBy != actor {
				t.Fatalf("duplicate page lost full Issue hydration: %#v", page[0])
			}
			page[0].AffectedRefs[0] = outcomes[0].Ref()
			page[0].AssigneeRef.ID = "caller edit"
			*page[0].DuplicateOfIssueID = id(999)
			stored, err := repo.Get(ctx, scopes[0], first.ID)
			must(err)
			if !reflect.DeepEqual(stored.AffectedRefs, first.AffectedRefs) || stored.AssigneeRef == nil || *stored.AssigneeRef != assignee || stored.DuplicateOfIssueID == nil || *stored.DuplicateOfIssueID != canonicals[0].ID {
				t.Fatal("page references alias mutable persisted Issue state")
			}
			for _, cursor := range []string{first.ID.String(), id(30).String()} {
				page, total, err = repo.ListDuplicateIssuesByObjective(ctx, scopes[0], owners[0].ID, cursor, 1)
				must(err)
				if total != 2 || len(page) != 1 || page[0].ID != second.ID || len(page[0].AffectedRefs) != 0 {
					t.Fatalf("exclusive cursor/count or zero-reference hydration failed: %#v, total=%d", page, total)
				}
			}
			page, total, err = repo.ListDuplicateIssuesByObjective(ctx, scopes[0], owners[0].ID, second.ID.String(), 101)
			must(err)
			if len(page) != 0 || total != 2 {
				t.Fatal("exhausted page lost full count or repeated cursor")
			}
			for _, owner := range []domain.Objective{otherOwner, owners[1], owners[2]} {
				page, total, err = repo.ListDuplicateIssuesByObjective(ctx, owner.Scope, owner.ID, "", 101)
				must(err)
				if total != 1 || len(page) != 1 || page[0].Scope != owner.Scope || page[0].ObjectiveID == nil || *page[0].ObjectiveID != owner.ID {
					t.Fatalf("Objective/Outcome/Namespace filter leaked Issues: %#v, total=%d", page, total)
				}
			}
			for _, boundary := range scopes {
				page, total, err = repo.ListDuplicateIssuesByObjective(ctx, boundary, id(999), "", 1)
				must(err)
				if len(page) != 0 || total != 0 {
					t.Fatal("unknown owner returned unrelated duplicates")
				}
			}
			page, total, err = repo.ListDuplicateIssuesByObjective(ctx, scopes[1], owners[0].ID, "", 1)
			must(err)
			if len(page) != 0 || total != 0 {
				t.Fatal("owner ID alone bypassed Outcome scope")
			}
			pending, pendingTotal, err := repo.ListPendingIssuesByObjective(ctx, scopes[0], owners[0].ID, "", 101)
			must(err)
			if pendingTotal != 2 || len(pending) != 2 || pending[0].ID != open.ID || pending[1].ID != investigating.ID {
				t.Fatal("pending query mixed duplicates or other closed lifecycles into its unchanged contract")
			}
			for _, limit := range []int{-1, 0, 102} {
				_, _, err := repo.ListDuplicateIssuesByObjective(ctx, scopes[0], owners[0].ID, "", limit)
				if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeInvalidArgument {
					t.Fatalf("invalid limit %d accepted: %v", limit, err)
				}
			}
			if _, _, err := repo.ListDuplicateIssuesByObjective(ctx, scopes[0], owners[0].ID, "bad-cursor", 1); err == nil {
				t.Fatal("invalid duplicate cursor was accepted")
			}
			// Count and page must see this UoW's writes, not a separate database
			// connection. Rolling back restores both lifecycle queue memberships.
			version := second.Version
			must(second.Reopen(now.Add(2 * time.Minute)))
			must(repo.Save(ctx, second, version))
			page, total, err = repo.ListDuplicateIssuesByObjective(ctx, scopes[0], owners[0].ID, "", 101)
			must(err)
			if total != 1 || len(page) != 1 || page[0].ID != first.ID {
				t.Fatal("count/page did not share the caller's transaction snapshot")
			}
			must(read.Rollback())
			if _, _, err := repo.ListDuplicateIssuesByObjective(ctx, scopes[0], owners[0].ID, "", 1); err == nil {
				t.Fatal("closed unit of work accepted duplicate query")
			}
			restored, err := store.Begin(ctx)
			must(err)
			defer restored.Rollback()
			page, total, err = restored.(ports.IssueBlockerUnitOfWork).Issues().ListDuplicateIssuesByObjective(ctx, scopes[0], owners[0].ID, "", 101)
			must(err)
			if len(page) != 2 || total != 2 {
				t.Fatal("rolled-back lifecycle mutation changed duplicate pages/count")
			}
		})
	}
}

func TestListPendingIssuesByObjectiveUsesStableScopedPages(t *testing.T) {
	ctx := context.Background()
	store, err := Open(filepath.Join(t.TempDir(), "pending-issues.db"), Options{BusyTimeout: time.Second, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	now := time.Date(2026, 10, 20, 12, 0, 0, 0, time.UTC)
	service, err := application.NewService(store, sqliteFixedClock{now}, &sqliteSequenceIDs{prefix: "0199f146", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := service.CreateOutcome(ctx, sqliteCommandContext("0199f147-0000-7000-8000-000000000001", ""), application.CreateOutcomeCommand{
		NamespaceID: domain.MustParseID("0199f148-0000-7000-8000-000000000001"), Title: "Issue query scope", DesiredState: "Pending Issues have an Objective", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	scope := outcome.Value.Scope()
	owner, err := service.CreateObjective(ctx, sqliteCommandContext("0199f147-0000-7000-8000-000000000002", ""), application.CreateObjectiveCommand{
		Scope: scope, Title: "Selected owner", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	otherOwner, err := service.CreateObjective(ctx, sqliteCommandContext("0199f147-0000-7000-8000-000000000003", ""), application.CreateObjectiveCommand{
		Scope: scope, Title: "Other owner", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	actor := sqliteCommandContext("0199f147-0000-7000-8000-000000000004", "").Actor
	newIssue := func(id string, objectiveID domain.ID) domain.Issue {
		t.Helper()
		issue, err := domain.NewIssueForObjective(
			domain.MustParseID(id), scope, &objectiveID, "Operational issue", "", domain.IssueSeverityMinor,
			[]domain.EntityRef{owner.Value.Ref()}, actor, now,
		)
		if err != nil {
			t.Fatal(err)
		}
		return issue
	}
	first := newIssue("0199f149-0000-7000-8000-000000000010", owner.Value.ID)
	assignee := domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "test", ID: "issue-worker"}
	first.AssigneeRef = &assignee
	if err := first.Investigate(now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	second := newIssue("0199f149-0000-7000-8000-000000000020", owner.Value.ID)
	closed := newIssue("0199f149-0000-7000-8000-000000000030", owner.Value.ID)
	if err := closed.Resolve("accepted resolution", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	other := newIssue("0199f149-0000-7000-8000-000000000040", otherOwner.Value.ID)
	write, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	issues := write.(ports.IssueBlockerUnitOfWork).Issues()
	for _, issue := range []domain.Issue{first, second, closed, other} {
		if err := issues.Insert(ctx, issue); err != nil {
			_ = write.Rollback()
			t.Fatal(err)
		}
	}
	if err := write.Commit(); err != nil {
		t.Fatal(err)
	}

	read, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer read.Rollback()
	repo := read.(ports.IssueBlockerUnitOfWork).Issues()
	page, total, err := repo.ListPendingIssuesByObjective(ctx, scope, owner.Value.ID, "", 1)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(page) != 1 || page[0].ID != first.ID || page[0].Lifecycle != domain.IssueLifecycleInvestigating {
		t.Fatalf("first pending page = %#v total=%d, want first investigating Issue and total 2", page, total)
	}
	if len(page[0].AffectedRefs) != 1 || page[0].AffectedRefs[0] != owner.Value.Ref() {
		t.Fatalf("affected references were not hydrated with the page: %#v", page[0].AffectedRefs)
	}
	if page[0].AssigneeRef == nil || *page[0].AssigneeRef != assignee {
		t.Fatalf("assignee was not persisted and hydrated with the page: %#v", page[0].AssigneeRef)
	}
	stored, err := repo.Get(ctx, scope, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.AssigneeRef == nil || *stored.AssigneeRef != assignee || stored.ReportedBy != actor {
		t.Fatalf("Issue Get did not preserve assignee and reporter: %#v", stored)
	}
	previousVersion := stored.Version
	stored.AssigneeRef = nil
	stored.Version++
	stored.UpdatedAt = now.Add(2 * time.Minute)
	if err := repo.Save(ctx, stored, previousVersion); err != nil {
		t.Fatal(err)
	}
	stored, err = repo.Get(ctx, scope, first.ID)
	if err != nil {
		t.Fatal(err)
	}
	if stored.AssigneeRef != nil || stored.ReportedBy != actor {
		t.Fatalf("Issue Save did not clear assignee while preserving reporter: %#v", stored)
	}
	page, total, err = repo.ListPendingIssuesByObjective(ctx, scope, owner.Value.ID, first.ID.String(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if total != 2 || len(page) != 1 || page[0].ID != second.ID {
		t.Fatalf("second pending page = %#v total=%d, want second open Issue and total 2", page, total)
	}
	page, total, err = repo.ListPendingIssuesByObjective(ctx, scope, otherOwner.Value.ID, "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(page) != 1 || page[0].ID != other.ID {
		t.Fatalf("other Objective page = %#v total=%d, want only its Issue", page, total)
	}
	page, total, err = repo.ListPendingIssuesByObjective(ctx, scope, domain.MustParseID("0199f149-0000-7000-8000-000000000099"), "", 10)
	if err != nil {
		t.Fatal(err)
	}
	if total != 0 || len(page) != 0 {
		t.Fatalf("unknown Objective page = %#v total=%d, want empty", page, total)
	}
	if _, _, err := repo.ListPendingIssuesByObjective(ctx, scope, owner.Value.ID, "bad-cursor", 1); err == nil {
		t.Fatal("invalid cursor was accepted")
	}
	if _, _, err := repo.ListPendingIssuesByObjective(ctx, scope, owner.Value.ID, "", 0); err == nil {
		t.Fatal("invalid page limit was accepted")
	}
}

func TestIssueBlockerStateSurvivesSQLiteRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "wos.db")
	store, err := Open(path, Options{BusyTimeout: time.Second, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.NewService(
		store,
		sqliteFixedClock{now: time.Date(2026, 10, 2, 17, 0, 0, 0, time.UTC)},
		&sqliteSequenceIDs{prefix: "0199f100", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	namespaceID := domain.MustParseID("0199f101-0000-7000-8000-000000000001")
	created, err := service.CreateOutcome(ctx, sqliteCommandContext("0199f101-0000-7000-8000-000000000101", ""), application.CreateOutcomeCommand{
		NamespaceID: namespaceID, Title: "Issue blocker durability",
		DesiredState: "Blocking state survives restart", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value
	if _, err := service.AddCriterion(ctx, sqliteCommandContext("0199f101-0000-7000-8000-000000000102", ""), application.AddCriterionCommand{
		Owner: outcome.Ref(), ExpectedVersion: outcome.Version, Title: "activation",
		Required: true, VerificationMode: domain.VerificationModeAttestation,
	}); err != nil {
		t.Fatal(err)
	}
	outcome.Version++
	activated, err := service.ActivateOutcome(ctx, sqliteCommandContext("0199f101-0000-7000-8000-000000000103", ""), application.ActivateOutcomeCommand{
		Scope: outcome.Scope(), ExpectedVersion: outcome.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome = activated.Value
	workResult, err := service.CreateWorkItem(ctx, sqliteCommandContext("0199f101-0000-7000-8000-000000000104", ""), application.CreateWorkItemCommand{
		Scope: outcome.Scope(), ObjectiveID: sqliteObjectiveForWorkFixture(t, service, outcome.Scope()), Title: "blocked work", Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleTodo,
	})
	if err != nil {
		t.Fatal(err)
	}
	work := workResult.Value
	issueResult, err := service.CreateIssue(ctx, sqliteCommandContext("0199f101-0000-7000-8000-000000000105", ""), application.CreateIssueCommand{
		Scope: outcome.Scope(), ObjectiveID: *work.ObjectiveID, Title: "root cause", Severity: domain.IssueSeverityMajor,
		AffectedRefs: []domain.EntityRef{work.Ref()},
	})
	if err != nil {
		t.Fatal(err)
	}
	issue := issueResult.Value
	cause := issue.Ref()
	blockerResult, err := service.CreateBlocker(ctx, sqliteCommandContext("0199f101-0000-7000-8000-000000000106", ""), application.CreateBlockerCommand{
		Scope: outcome.Scope(), BlockedRef: work.Ref(), CauseRef: &cause, Description: "execution blocked",
	})
	if err != nil {
		t.Fatal(err)
	}
	blocker := blockerResult.Value
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path, Options{BusyTimeout: time.Second, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	restarted, err := application.NewService(
		reopened,
		sqliteFixedClock{now: time.Date(2026, 10, 2, 17, 5, 0, 0, time.UTC)},
		&sqliteSequenceIDs{prefix: "0199f102", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	readIssue, err := restarted.GetIssue(ctx, outcome.Scope(), issue.ID)
	if err != nil {
		t.Fatal(err)
	}
	if readIssue.Value.Title != issue.Title || len(readIssue.Value.AffectedRefs) != 1 || readIssue.Value.ObjectiveID == nil || *readIssue.Value.ObjectiveID != *issue.ObjectiveID {
		t.Fatalf("issue after restart = %#v", readIssue.Value)
	}
	state, _, err := restarted.GetBlockingState(ctx, work.Ref())
	if err != nil {
		t.Fatal(err)
	}
	if !state.IsBlocked || len(state.ActiveBlockers) != 1 || state.ActiveBlockers[0].Blocker.ID != blocker.ID {
		t.Fatalf("blocking state after restart = %#v", state)
	}
}
