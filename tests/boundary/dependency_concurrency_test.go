package boundary_test

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
	sqlitestore "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
)

func TestConcurrentOppositeDependencyEdgesCannotCreateCycle(t *testing.T) {
	now := time.Date(2026, 10, 2, 14, 0, 0, 0, time.UTC)

	t.Run("memory", func(t *testing.T) {
		store := memory.New()
		assertConcurrentOppositeEdges(t, store, store, now)
	})

	t.Run("sqlite", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "wos.db")
		first, err := sqlitestore.Open(path, sqlitestore.Options{
			BusyTimeout:   2 * time.Second,
			MigrateOnOpen: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		defer first.Close()
		second, err := sqlitestore.Open(path, sqlitestore.Options{
			BusyTimeout:   2 * time.Second,
			MigrateOnOpen: true,
		})
		if err != nil {
			t.Fatal(err)
		}
		defer second.Close()
		assertConcurrentOppositeEdges(t, first, second, now)
	})
	t.Run("postgres", func(t *testing.T) {
		dsn := os.Getenv("WOS_TEST_POSTGRES_DSN")
		if dsn == "" {
			t.Skip("real PostgreSQL required")
		}
		schema := fmt.Sprintf("wos_graph_%x", sha256.Sum256([]byte(t.TempDir())))[:42]
		opts := postgres.Options{Schema: schema, MigrateOnOpen: true, BusyTimeout: 2 * time.Second}
		first, err := postgres.Open(dsn, opts)
		if err != nil {
			t.Fatal(err)
		}
		defer first.Close()
		second, err := postgres.Open(dsn, opts)
		if err != nil {
			t.Fatal(err)
		}
		defer second.Close()
		assertConcurrentOppositeEdges(t, first, second, now)
	})
}

func assertConcurrentOppositeEdges(
	t *testing.T,
	first ports.TransactionManager,
	second ports.TransactionManager,
	now time.Time,
) {
	t.Helper()
	ctx := context.Background()
	clock := &readinessClock{now: now}
	ids := &readinessIDs{}
	setup, err := application.NewService(first, clock, ids)
	if err != nil {
		t.Fatal(err)
	}
	commands := &readinessCommands{}
	namespaceID := domain.MustParseID("0199ed10-0000-7000-8000-000000000001")

	created, err := setup.CreateOutcome(ctx, commands.Context(), application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Concurrent graph",
		DesiredState: "Opposite edges never commit together",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value
	if _, err := setup.AddCriterion(ctx, commands.Context(), application.AddCriterionCommand{
		Owner:            outcome.Ref(),
		ExpectedVersion:  outcome.Version,
		Title:            "Activation criterion",
		Required:         true,
		VerificationMode: domain.VerificationModeAttestation,
	}); err != nil {
		t.Fatal(err)
	}
	outcome.Version++
	activated, err := setup.ActivateOutcome(ctx, commands.Context(), application.ActivateOutcomeCommand{
		Scope:           outcome.Scope(),
		ExpectedVersion: outcome.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome = activated.Value

	aResult, err := setup.CreateWorkItem(ctx, commands.Context(), application.CreateWorkItemCommand{
		Scope: outcome.Scope(), Title: "A", Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleTodo,
	})
	if err != nil {
		t.Fatal(err)
	}
	bResult, err := setup.CreateWorkItem(ctx, commands.Context(), application.CreateWorkItemCommand{
		Scope: outcome.Scope(), Title: "B", Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleTodo,
	})
	if err != nil {
		t.Fatal(err)
	}
	a := aResult.Value
	b := bResult.Value

	serviceA, err := application.NewService(first, clock, ids)
	if err != nil {
		t.Fatal(err)
	}
	serviceB, err := application.NewService(second, clock, ids)
	if err != nil {
		t.Fatal(err)
	}

	type result struct {
		direction string
		err       error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		_, err := serviceA.AddDependency(ctx, concurrentCommandContext(
			"0199ed11-0000-7000-8000-000000000001",
		), application.AddDependencyCommand{
			Scope: outcome.Scope(), SourceRef: a.Ref(), TargetRef: b.Ref(),
			Strength: domain.DependencyStrengthHard,
		})
		results <- result{direction: "a-b", err: err}
	}()
	go func() {
		defer wg.Done()
		<-start
		_, err := serviceB.AddDependency(ctx, concurrentCommandContext(
			"0199ed11-0000-7000-8000-000000000002",
		), application.AddDependencyCommand{
			Scope: outcome.Scope(), SourceRef: b.Ref(), TargetRef: a.Ref(),
			Strength: domain.DependencyStrengthHard,
		})
		results <- result{direction: "b-a", err: err}
	}()
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	cycleErrors := 0
	for value := range results {
		if value.err == nil {
			successes++
			continue
		}
		code, ok := domain.ErrorCodeOf(value.err)
		if ok && (code == domain.ErrorCodeDependencyCycle || code == domain.ErrorCodeTransactionConflict) {
			cycleErrors++
			continue
		}
		t.Fatalf("%s returned unexpected error: %v", value.direction, value.err)
	}
	if successes != 1 || cycleErrors != 1 {
		t.Fatalf("concurrent opposite edges successes=%d rejected_edges=%d, want 1/1", successes, cycleErrors)
	}

	relations, _, err := serviceA.ListRelations(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	active := 0
	for _, relation := range relations {
		if relation.Lifecycle == domain.RelationLifecycleActive &&
			relation.RelationType == domain.RelationTypeDependsOn {
			active++
		}
	}
	if active != 1 {
		t.Fatalf("active dependency edges = %d, want 1", active)
	}
}

func concurrentCommandContext(commandID string) domain.CommandContext {
	return domain.CommandContext{
		PrincipalID: "concurrency-human",
		Actor: domain.ActorRef{
			Kind:     domain.ActorKindHuman,
			Provider: "local",
			ID:       "concurrency-human",
		},
		CommandID: domain.MustParseID(commandID),
	}
}
