package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

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
		Scope: outcome.Scope(), Title: "blocked work", Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleTodo,
	})
	if err != nil {
		t.Fatal(err)
	}
	work := workResult.Value
	issueResult, err := service.CreateIssue(ctx, sqliteCommandContext("0199f101-0000-7000-8000-000000000105", ""), application.CreateIssueCommand{
		Scope: outcome.Scope(), Title: "root cause", Severity: domain.IssueSeverityMajor,
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
	if readIssue.Value.Title != issue.Title || len(readIssue.Value.AffectedRefs) != 1 {
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
