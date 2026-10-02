package boundary_test

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
	sqlitestore "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
)

type readinessClock struct {
	mu  sync.Mutex
	now time.Time
}

func (c *readinessClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *readinessClock) Set(value time.Time) {
	c.mu.Lock()
	c.now = value
	c.mu.Unlock()
}

type readinessIDs struct {
	mu   sync.Mutex
	next uint64
}

func (g *readinessIDs) NewID() (domain.ID, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.next++
	return domain.MustParseID(fmt.Sprintf("0199ed00-0000-7000-8000-%012x", g.next)), nil
}

type readinessCommands struct {
	next uint64
}

func (g *readinessCommands) Context() domain.CommandContext {
	g.next++
	return domain.CommandContext{
		PrincipalID: "readiness-human",
		Actor: domain.ActorRef{
			Kind:     domain.ActorKindHuman,
			Provider: "local",
			ID:       "readiness-human",
		},
		CommandID: domain.MustParseID(fmt.Sprintf("0199ed01-0000-7000-8000-%012x", g.next)),
	}
}

type readinessSnapshot struct {
	BeforeTargetCompletion []string
	AfterTargetCompletion  []string
	AtNotBeforeBoundary    []string
}

func TestReadyWorkParityBetweenMemoryAndSQLite(t *testing.T) {
	now := time.Date(2026, 10, 2, 13, 0, 0, 0, time.UTC)

	memoryClock := &readinessClock{now: now}
	memoryResult := runReadinessScenario(t, memory.New(), memoryClock)

	sqliteStore, err := sqlitestore.Open(filepath.Join(t.TempDir(), "wos.db"), sqlitestore.Options{
		BusyTimeout:   time.Second,
		MigrateOnOpen: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer sqliteStore.Close()
	sqliteClock := &readinessClock{now: now}
	sqliteResult := runReadinessScenario(t, sqliteStore, sqliteClock)

	if !reflect.DeepEqual(memoryResult, sqliteResult) {
		t.Fatalf("readiness parity mismatch\nmemory=%#v\nsqlite=%#v", memoryResult, sqliteResult)
	}
}

func runReadinessScenario(t *testing.T, tx ports.TransactionManager, clock *readinessClock) readinessSnapshot {
	t.Helper()
	ctx := context.Background()
	ids := &readinessIDs{}
	commands := &readinessCommands{}
	service, err := application.NewService(tx, clock, ids)
	if err != nil {
		t.Fatal(err)
	}
	namespaceID := domain.MustParseID("0199ed02-0000-7000-8000-000000000001")

	createdOutcome, err := service.CreateOutcome(ctx, commands.Context(), application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Readiness parity",
		DesiredState: "Memory and SQLite agree",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := createdOutcome.Value
	if _, err := service.AddCriterion(ctx, commands.Context(), application.AddCriterionCommand{
		Owner:            outcome.Ref(),
		ExpectedVersion:  outcome.Version,
		Title:            "Activation criterion",
		Required:         true,
		VerificationMode: domain.VerificationModeAttestation,
	}); err != nil {
		t.Fatal(err)
	}
	outcome.Version++
	activated, err := service.ActivateOutcome(ctx, commands.Context(), application.ActivateOutcomeCommand{
		Scope:           outcome.Scope(),
		ExpectedVersion: outcome.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome = activated.Value

	createWork := func(title string, notBefore *time.Time) domain.WorkItem {
		result, err := service.CreateWorkItem(ctx, commands.Context(), application.CreateWorkItemCommand{
			Scope:     outcome.Scope(),
			Title:     title,
			Priority:  domain.PriorityNormal,
			Lifecycle: domain.WorkItemLifecycleTodo,
			NotBefore: notBefore,
		})
		if err != nil {
			t.Fatal(err)
		}
		return result.Value
	}

	target := createWork("target", nil)
	hardSource := createWork("hard-source", nil)
	advisorySource := createWork("advisory-source", nil)
	futureTime := clock.Now().Add(time.Hour)
	future := createWork("future", &futureTime)
	boundary := clock.Now()
	atBoundary := createWork("at-boundary", &boundary)

	if _, err := service.AddDependency(ctx, commands.Context(), application.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: hardSource.Ref(), TargetRef: target.Ref(),
		Strength: domain.DependencyStrengthHard,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AddDependency(ctx, commands.Context(), application.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: advisorySource.Ref(), TargetRef: target.Ref(),
		Strength: domain.DependencyStrengthAdvisory,
	}); err != nil {
		t.Fatal(err)
	}

	before, err := service.ListReadyWork(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}

	claimed, err := service.ClaimWorkItem(ctx, commands.Context(), application.ClaimWorkItemCommand{
		Scope:           outcome.Scope(),
		WorkItemID:      target.ID,
		ExpectedVersion: target.Version,
		TTL:             domain.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	target = claimed.Value
	if _, err := service.CompleteWorkItem(ctx, commands.Context(), application.CompleteWorkItemCommand{
		Scope:           outcome.Scope(),
		WorkItemID:      target.ID,
		ExpectedVersion: target.Version,
		ClaimID:         target.CurrentLease.ClaimID,
		FencingToken:    target.CurrentLease.FencingToken,
		ResultSummary:   "target completed",
		Reason:          "dependency satisfied",
	}); err != nil {
		t.Fatal(err)
	}

	after, err := service.ListReadyWork(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	clock.Set(futureTime)
	atFuture, err := service.ListReadyWork(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}

	_ = future
	_ = atBoundary
	return readinessSnapshot{
		BeforeTargetCompletion: readyTitles(before.Items),
		AfterTargetCompletion:  readyTitles(after.Items),
		AtNotBeforeBoundary:    readyTitles(atFuture.Items),
	}
}

func readyTitles(items []application.ReadyWorkItem) []string {
	result := make([]string, len(items))
	for i, item := range items {
		result[i] = item.WorkItem.Title
	}
	return result
}
