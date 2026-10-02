package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func TestDependencyAndNotBeforeSurviveSQLiteRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "wos.db")
	now := time.Date(2026, 10, 2, 15, 0, 0, 0, time.UTC)
	future := now.Add(time.Hour)

	store, err := Open(path, Options{BusyTimeout: time.Second, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	service, err := application.NewService(
		store,
		sqliteFixedClock{now: now},
		&sqliteSequenceIDs{prefix: "0199ef00", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	namespaceID := testID("0199ef01-0000-7000-8000-000000000001")
	created, err := service.CreateOutcome(ctx, sqliteCommandContext(
		"0199ef01-0000-7000-8000-000000000101", "",
	), application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Durable dependencies",
		DesiredState: "Relations and schedule survive restart",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value
	if _, err := service.AddCriterion(ctx, sqliteCommandContext(
		"0199ef01-0000-7000-8000-000000000102", "",
	), application.AddCriterionCommand{
		Owner:            outcome.Ref(),
		ExpectedVersion:  outcome.Version,
		Title:            "Activation criterion",
		Required:         true,
		VerificationMode: domain.VerificationModeAttestation,
	}); err != nil {
		t.Fatal(err)
	}
	outcome.Version++
	activated, err := service.ActivateOutcome(ctx, sqliteCommandContext(
		"0199ef01-0000-7000-8000-000000000103", "",
	), application.ActivateOutcomeCommand{
		Scope:           outcome.Scope(),
		ExpectedVersion: outcome.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome = activated.Value

	targetResult, err := service.CreateWorkItem(ctx, sqliteCommandContext(
		"0199ef01-0000-7000-8000-000000000104", "",
	), application.CreateWorkItemCommand{
		Scope: outcome.Scope(), Title: "target", Priority: domain.PriorityNormal,
		Lifecycle: domain.WorkItemLifecycleTodo,
	})
	if err != nil {
		t.Fatal(err)
	}
	target := targetResult.Value
	sourceResult, err := service.CreateWorkItem(ctx, sqliteCommandContext(
		"0199ef01-0000-7000-8000-000000000105", "",
	), application.CreateWorkItemCommand{
		Scope: outcome.Scope(), Title: "source", Priority: domain.PriorityNormal,
		Lifecycle: domain.WorkItemLifecycleTodo, NotBefore: &future,
	})
	if err != nil {
		t.Fatal(err)
	}
	source := sourceResult.Value
	relationResult, err := service.AddDependency(ctx, sqliteCommandContext(
		"0199ef01-0000-7000-8000-000000000106", "",
	), application.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: source.Ref(), TargetRef: target.Ref(),
		Strength: domain.DependencyStrengthHard,
		Reason:   "durable prerequisite",
	})
	if err != nil {
		t.Fatal(err)
	}
	relation := relationResult.Value

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
		sqliteFixedClock{now: now},
		&sqliteSequenceIDs{prefix: "0199ef02", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}

	readRelation, err := restarted.GetRelation(ctx, outcome.Scope(), relation.ID)
	if err != nil {
		t.Fatal(err)
	}
	if readRelation.Value.SourceRef != source.Ref() || readRelation.Value.TargetRef != target.Ref() {
		t.Fatalf("relation after restart = %#v", readRelation.Value)
	}
	readSource, err := restarted.GetWorkItem(ctx, outcome.Scope(), source.ID)
	if err != nil {
		t.Fatal(err)
	}
	if readSource.Value.NotBefore == nil || !readSource.Value.NotBefore.Equal(future) {
		t.Fatalf("not_before after restart = %v, want %s", readSource.Value.NotBefore, future)
	}

	ready, err := restarted.ListReadyWork(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if readyWorkContains(ready.Items, source.ID) {
		t.Fatal("source unexpectedly ready before not_before and target completion")
	}
	if !readyWorkContains(ready.Items, target.ID) {
		t.Fatal("target should be ready after restart")
	}

	readTarget, err := restarted.GetWorkItem(ctx, outcome.Scope(), target.ID)
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := restarted.ClaimWorkItem(ctx, sqliteCommandContext(
		"0199ef01-0000-7000-8000-000000000107", "",
	), application.ClaimWorkItemCommand{
		Scope:           outcome.Scope(),
		WorkItemID:      target.ID,
		ExpectedVersion: readTarget.Value.Version,
		TTL:             domain.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := restarted.CompleteWorkItem(ctx, sqliteCommandContext(
		"0199ef01-0000-7000-8000-000000000108", "",
	), application.CompleteWorkItemCommand{
		Scope:           outcome.Scope(),
		WorkItemID:      target.ID,
		ExpectedVersion: claimed.Value.Version,
		ClaimID:         claimed.Value.CurrentLease.ClaimID,
		FencingToken:    claimed.Value.CurrentLease.FencingToken,
		ResultSummary:   "target complete",
		Reason:          "dependency satisfied",
	}); err != nil {
		t.Fatal(err)
	}

	later, err := application.NewService(
		reopened,
		sqliteFixedClock{now: future},
		&sqliteSequenceIDs{prefix: "0199ef03", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	ready, err = later.ListReadyWork(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if !readyWorkContains(ready.Items, source.ID) {
		t.Fatal("source should be ready at not_before after target completion")
	}
}

func readyWorkContains(items []application.ReadyWorkItem, id domain.ID) bool {
	for _, item := range items {
		if item.WorkItem.ID == id {
			return true
		}
	}
	return false
}
