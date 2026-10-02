package sqlite

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

type sqliteFixedClock struct{ now time.Time }

func (c sqliteFixedClock) Now() time.Time { return c.now }

type sqliteSequenceIDs struct {
	prefix string
	next   uint64
}

func (g *sqliteSequenceIDs) NewID() (domain.ID, error) {
	value := domain.MustParseID(fmt.Sprintf("%s-0000-7000-8000-%012x", g.prefix, g.next))
	g.next++
	return value, nil
}

func sqliteCommandContext(commandID, key string) domain.CommandContext {
	return domain.CommandContext{
		PrincipalID: "sqlite-human",
		Actor: domain.ActorRef{
			Kind:     domain.ActorKindHuman,
			Provider: "local",
			ID:       "sqlite-human",
		},
		CommandID:      domain.MustParseID(commandID),
		IdempotencyKey: key,
		CorrelationID:  "sqlite-wave-04",
	}
}

func TestSQLiteWave02ScenarioSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "wos.db")
	store, err := Open(path, Options{BusyTimeout: time.Second, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	clock := sqliteFixedClock{now: time.Date(2026, 10, 2, 5, 0, 0, 0, time.UTC)}
	service, err := application.NewService(
		store,
		clock,
		&sqliteSequenceIDs{prefix: "0199e950", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	namespaceID := domain.MustParseID("0199e951-0000-7000-8000-000000000001")

	createdOutcome, err := service.CreateOutcome(ctx, sqliteCommandContext(
		"0199e951-0000-7000-8000-000000000101", "",
	), application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Durable local capability",
		DesiredState: "Capability is available and verified after restart",
		Priority:     domain.PriorityHigh,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := createdOutcome.Value

	outcomeCriterion, err := service.AddCriterion(ctx, sqliteCommandContext(
		"0199e951-0000-7000-8000-000000000102", "",
	), application.AddCriterionCommand{
		Owner:            outcome.Ref(),
		ExpectedVersion:  outcome.Version,
		Title:            "Outcome verified",
		Required:         true,
		VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome.Version++

	activated, err := service.ActivateOutcome(ctx, sqliteCommandContext(
		"0199e951-0000-7000-8000-000000000103", "",
	), application.ActivateOutcomeCommand{
		Scope:           outcome.Scope(),
		ExpectedVersion: outcome.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome = activated.Value

	createdObjective, err := service.CreateObjective(ctx, sqliteCommandContext(
		"0199e951-0000-7000-8000-000000000104", "",
	), application.CreateObjectiveCommand{
		Scope:              outcome.Scope(),
		Title:              "Prepare durable capability",
		Priority:           domain.PriorityNormal,
		RequiredForOutcome: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	objective := createdObjective.Value

	objectiveCriterion, err := service.AddCriterion(ctx, sqliteCommandContext(
		"0199e951-0000-7000-8000-000000000105", "",
	), application.AddCriterionCommand{
		Owner:            objective.Ref(),
		ExpectedVersion:  objective.Version,
		Title:            "Preparation verified",
		Required:         true,
		VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	objective.Version++

	startedObjective, err := service.StartObjective(ctx, sqliteCommandContext(
		"0199e951-0000-7000-8000-000000000106", "",
	), application.StartObjectiveCommand{
		Scope:           outcome.Scope(),
		ObjectiveID:     objective.ID,
		ExpectedVersion: objective.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	objective = startedObjective.Value

	createdWork, err := service.CreateWorkItem(ctx, sqliteCommandContext(
		"0199e951-0000-7000-8000-000000000107", "",
	), application.CreateWorkItemCommand{
		Scope:       outcome.Scope(),
		Title:       "Perform durable preparation",
		Priority:    domain.PriorityNormal,
		Lifecycle:   domain.WorkItemLifecycleBacklog,
		ObjectiveID: &objective.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	work := createdWork.Value

	activatedWork, err := service.ActivateWorkItem(ctx, sqliteCommandContext(
		"0199e951-0000-7000-8000-000000000108", "",
	), application.ActivateWorkItemCommand{
		Scope:           outcome.Scope(),
		WorkItemID:      work.ID,
		ExpectedVersion: work.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	work = activatedWork.Value

	claimed, err := service.ClaimWorkItem(ctx, sqliteCommandContext(
		"0199e951-0000-7000-8000-000000000109", "",
	), application.ClaimWorkItemCommand{
		Scope:           outcome.Scope(),
		WorkItemID:      work.ID,
		ExpectedVersion: work.Version,
		TTL:             domain.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	work = claimed.Value
	if work.CurrentLease == nil {
		t.Fatal("claim did not create a lease")
	}

	completed, err := service.CompleteWorkItem(ctx, sqliteCommandContext(
		"0199e951-0000-7000-8000-000000000110", "",
	), application.CompleteWorkItemCommand{
		Scope:           outcome.Scope(),
		WorkItemID:      work.ID,
		ExpectedVersion: work.Version,
		ClaimID:         work.CurrentLease.ClaimID,
		FencingToken:    work.CurrentLease.FencingToken,
		ResultSummary:   "Preparation completed",
		Reason:          "work performed",
	})
	if err != nil {
		t.Fatal(err)
	}
	work = completed.Value

	if _, err := service.AttestCriterion(ctx, sqliteCommandContext(
		"0199e951-0000-7000-8000-000000000111", "",
	), application.AttestCriterionCommand{
		Owner:             objective.Ref(),
		CriterionID:       objectiveCriterion.Value.ID,
		CriterionRevision: domain.InitialCriterionRevision,
		ExpectedVersion:   objective.Version,
		Result:            domain.AssessmentResultMet,
		Rationale:         "human verified preparation",
	}); err != nil {
		t.Fatal(err)
	}
	objective.Version++

	achievedObjective, err := service.AchieveObjective(ctx, sqliteCommandContext(
		"0199e951-0000-7000-8000-000000000112", "",
	), application.AchieveObjectiveCommand{
		Scope:           outcome.Scope(),
		ObjectiveID:     objective.ID,
		ExpectedVersion: objective.Version,
		Reason:          "objective verified",
	})
	if err != nil {
		t.Fatal(err)
	}
	objective = achievedObjective.Value

	if _, err := service.AttestCriterion(ctx, sqliteCommandContext(
		"0199e951-0000-7000-8000-000000000113", "",
	), application.AttestCriterionCommand{
		Owner:             outcome.Ref(),
		CriterionID:       outcomeCriterion.Value.ID,
		CriterionRevision: domain.InitialCriterionRevision,
		ExpectedVersion:   outcome.Version,
		Result:            domain.AssessmentResultMet,
		Rationale:         "human verified outcome",
	}); err != nil {
		t.Fatal(err)
	}
	outcome.Version++

	achievedOutcome, err := service.AchieveOutcome(ctx, sqliteCommandContext(
		"0199e951-0000-7000-8000-000000000114", "",
	), application.AchieveOutcomeCommand{
		Scope:           outcome.Scope(),
		ExpectedVersion: outcome.Version,
		Reason:          "required objective and criterion satisfied",
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome = achievedOutcome.Value

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path, Options{BusyTimeout: time.Second, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()

	uow, err := reopened.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	persistedOutcome, err := uow.Outcomes().Get(ctx, namespaceID, outcome.ID)
	if err != nil {
		t.Fatal(err)
	}
	persistedObjective, err := uow.Objectives().Get(ctx, outcome.Scope(), objective.ID)
	if err != nil {
		t.Fatal(err)
	}
	persistedWork, err := uow.WorkItems().Get(ctx, outcome.Scope(), work.ID)
	if err != nil {
		t.Fatal(err)
	}
	coord, err := uow.Coordination().LockOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}

	if persistedOutcome.Lifecycle != domain.OutcomeLifecycleAchieved {
		t.Fatalf("outcome lifecycle = %q, want achieved", persistedOutcome.Lifecycle)
	}
	if persistedOutcome.CurrentConclusion == nil {
		t.Fatal("outcome conclusion was not restored")
	}
	if persistedObjective.Lifecycle != domain.ObjectiveLifecycleAchieved {
		t.Fatalf("objective lifecycle = %q, want achieved", persistedObjective.Lifecycle)
	}
	if persistedObjective.CurrentConclusion == nil {
		t.Fatal("objective conclusion was not restored")
	}
	if persistedWork.Lifecycle != domain.WorkItemLifecycleDone {
		t.Fatalf("work lifecycle = %q, want done", persistedWork.Lifecycle)
	}
	if persistedWork.CurrentLease != nil {
		t.Fatal("completed work item restored an active lease")
	}
	if coord.Revision != 14 {
		t.Fatalf("outcome revision = %d, want 14", coord.Revision)
	}
	if err := uow.Rollback(); err != nil {
		t.Fatal(err)
	}

	events, err := reopened.SnapshotDomainEvents(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 15 {
		t.Fatalf("event count = %d, want 15", len(events))
	}
	if events[len(events)-1].OutcomeRevision != 14 {
		t.Fatalf("last event revision = %d, want 14", events[len(events)-1].OutcomeRevision)
	}
}

func TestSQLiteIdempotentReplaySurvivesRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "wos.db")
	store, err := Open(path, Options{BusyTimeout: time.Second, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	clock := sqliteFixedClock{now: time.Date(2026, 10, 2, 6, 0, 0, 0, time.UTC)}
	service, err := application.NewService(
		store,
		clock,
		&sqliteSequenceIDs{prefix: "0199e960", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	namespaceID := domain.MustParseID("0199e961-0000-7000-8000-000000000001")
	outcomeResult, err := service.CreateOutcome(ctx, sqliteCommandContext(
		"0199e961-0000-7000-8000-000000000101", "",
	), application.CreateOutcomeCommand{
		NamespaceID: namespaceID,
		Title: "Replay",
		DesiredState: "Replay survives process restart",
		Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	cmd := application.CreateWorkItemCommand{
		Scope:     outcomeResult.Value.Scope(),
		Title:     "Persistent replay",
		Priority:  domain.PriorityNormal,
		Lifecycle: domain.WorkItemLifecycleTodo,
	}
	key := "sqlite-restart-replay-0001"
	first, err := service.CreateWorkItem(ctx, sqliteCommandContext(
		"0199e961-0000-7000-8000-000000000102", key,
	), cmd)
	if err != nil {
		t.Fatal(err)
	}
	before, err := store.SnapshotDomainEvents(ctx, outcomeResult.Value.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := Open(path, Options{BusyTimeout: time.Second, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	restartedService, err := application.NewService(
		reopened,
		clock,
		&sqliteSequenceIDs{prefix: "0199e962", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := restartedService.CreateWorkItem(ctx, sqliteCommandContext(
		"0199e961-0000-7000-8000-000000000103", key,
	), cmd)
	if err != nil {
		t.Fatal(err)
	}
	after, err := reopened.SnapshotDomainEvents(ctx, outcomeResult.Value.Scope())
	if err != nil {
		t.Fatal(err)
	}

	if !replay.IdempotentReplay {
		t.Fatal("replay was not marked idempotent")
	}
	if replay.Value.ID != first.Value.ID {
		t.Fatalf("replayed work id = %s, want %s", replay.Value.ID, first.Value.ID)
	}
	if replay.CommandID != first.CommandID {
		t.Fatalf("replayed command id = %s, want original %s", replay.CommandID, first.CommandID)
	}
	if replay.OutcomeRevision != first.OutcomeRevision {
		t.Fatalf("replayed revision = %d, want %d", replay.OutcomeRevision, first.OutcomeRevision)
	}
	if len(after) != len(before) {
		t.Fatalf("replay duplicated events: before=%d after=%d", len(before), len(after))
	}
}
