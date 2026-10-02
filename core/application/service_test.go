package application_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/storage/memory"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

type sequenceIDs struct {
	values []domain.ID
	next   int
}

func (g *sequenceIDs) NewID() (domain.ID, error) {
	if g.next >= len(g.values) {
		value := domain.MustParseID(fmt.Sprintf("0199e2ff-0000-7000-8000-%012x", g.next+1))
		g.next++
		return value, nil
	}
	id := g.values[g.next]
	g.next++
	return id, nil
}

func id(value string) domain.ID { return domain.MustParseID(value) }

func commandContext() domain.CommandContext {
	return domain.CommandContext{
		PrincipalID: "human-1",
		Actor: domain.ActorRef{
			Kind:     domain.ActorKindHuman,
			Provider: "local",
			ID:       "human-1",
		},
		CommandID: id("0199e200-0000-7000-8000-000000000099"),
	}
}

func newService(t *testing.T) (*application.Service, *memory.Store) {
	t.Helper()
	store := memory.New()
	ids := &sequenceIDs{values: []domain.ID{
		id("0199e200-0000-7000-8000-000000000010"),
		id("0199e200-0000-7000-8000-000000000011"),
		id("0199e200-0000-7000-8000-000000000012"),
		id("0199e200-0000-7000-8000-000000000013"),
		id("0199e200-0000-7000-8000-000000000014"),
		id("0199e200-0000-7000-8000-000000000015"),
		id("0199e200-0000-7000-8000-000000000016"),
		id("0199e200-0000-7000-8000-000000000017"),
		id("0199e200-0000-7000-8000-000000000018"),
	}}
	service, err := application.NewService(
		store,
		fixedClock{now: time.Date(2026, 10, 1, 21, 30, 0, 0, time.UTC)},
		ids,
	)
	if err != nil {
		t.Fatal(err)
	}
	return service, store
}

func TestHumanOnlyScenarioCreatesAndCompletesOutcome(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	cc := commandContext()
	namespaceID := id("0199e200-0000-7000-8000-000000000001")

	createdOutcome, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Deliver local capability",
		DesiredState: "Capability is available and verified",
		Priority:     domain.PriorityHigh,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := createdOutcome.Value
	if createdOutcome.OutcomeRevision != 1 {
		t.Fatalf("create outcome revision = %d, want 1", createdOutcome.OutcomeRevision)
	}

	outcomeCriterion, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
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

	activated, err := service.ActivateOutcome(ctx, cc, application.ActivateOutcomeCommand{
		Scope:           outcome.Scope(),
		ExpectedVersion: outcome.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome = activated.Value

	createdObjective, err := service.CreateObjective(ctx, cc, application.CreateObjectiveCommand{
		Scope:              outcome.Scope(),
		Title:              "Prepare capability",
		Priority:           domain.PriorityNormal,
		RequiredForOutcome: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	objective := createdObjective.Value

	objectiveCriterion, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
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

	startedObjective, err := service.StartObjective(ctx, cc, application.StartObjectiveCommand{
		Scope:           outcome.Scope(),
		ObjectiveID:     objective.ID,
		ExpectedVersion: objective.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	objective = startedObjective.Value

	createdWork, err := service.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{
		Scope:       outcome.Scope(),
		Title:       "Perform preparation",
		Priority:    domain.PriorityNormal,
		Lifecycle:   domain.WorkItemLifecycleBacklog,
		ObjectiveID: &objective.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	work := createdWork.Value

	activatedWork, err := service.ActivateWorkItem(ctx, cc, application.ActivateWorkItemCommand{
		Scope:           outcome.Scope(),
		WorkItemID:      work.ID,
		ExpectedVersion: work.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	work = activatedWork.Value

	claimed, err := service.ClaimWorkItem(ctx, cc, application.ClaimWorkItemCommand{
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

	completed, err := service.CompleteWorkItem(ctx, cc, application.CompleteWorkItemCommand{
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
	if completed.Value.Lifecycle != domain.WorkItemLifecycleDone {
		t.Fatalf("work lifecycle = %q, want done", completed.Value.Lifecycle)
	}

	attestedObjective, err := service.AttestCriterion(ctx, cc, application.AttestCriterionCommand{
		Owner:             objective.Ref(),
		CriterionID:       objectiveCriterion.Value.ID,
		CriterionRevision: domain.InitialCriterionRevision,
		ExpectedVersion:   objective.Version,
		Result:            domain.AssessmentResultMet,
		Rationale:         "human verified preparation",
	})
	if err != nil {
		t.Fatal(err)
	}
	objective.Version++
	if attestedObjective.Value.Result != domain.AssessmentResultMet {
		t.Fatal("objective assessment should be met")
	}

	achievedObjective, err := service.AchieveObjective(ctx, cc, application.AchieveObjectiveCommand{
		Scope:           outcome.Scope(),
		ObjectiveID:     objective.ID,
		ExpectedVersion: objective.Version,
		Reason:          "objective verified",
	})
	if err != nil {
		t.Fatal(err)
	}
	objective = achievedObjective.Value
	if objective.Lifecycle != domain.ObjectiveLifecycleAchieved {
		t.Fatalf("objective lifecycle = %q, want achieved", objective.Lifecycle)
	}

	_, err = service.AttestCriterion(ctx, cc, application.AttestCriterionCommand{
		Owner:             outcome.Ref(),
		CriterionID:       outcomeCriterion.Value.ID,
		CriterionRevision: domain.InitialCriterionRevision,
		ExpectedVersion:   outcome.Version,
		Result:            domain.AssessmentResultMet,
		Rationale:         "human verified outcome",
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome.Version++

	achievedOutcome, err := service.AchieveOutcome(ctx, cc, application.AchieveOutcomeCommand{
		Scope:           outcome.Scope(),
		ExpectedVersion: outcome.Version,
		Reason:          "required objective and criterion satisfied",
	})
	if err != nil {
		t.Fatal(err)
	}
	if achievedOutcome.Value.Lifecycle != domain.OutcomeLifecycleAchieved {
		t.Fatalf("outcome lifecycle = %q, want achieved", achievedOutcome.Value.Lifecycle)
	}

	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	coord, err := tx.Coordination().LockOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if coord.Revision != 14 {
		t.Fatalf("outcome revision = %d, want 14", coord.Revision)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	events := store.SnapshotDomainEvents(outcome.Scope())
	var released, completedEvent *domain.DomainEvent
	for i := range events {
		switch events[i].EventType {
		case "work_item.released":
			if events[i].AggregateRef.ID == work.ID {
				released = &events[i]
			}
		case "work_item.completed":
			if events[i].AggregateRef.ID == work.ID {
				completedEvent = &events[i]
			}
		}
	}
	if released == nil || completedEvent == nil {
		t.Fatalf("completion did not emit both ordered events: released=%#v completed=%#v", released, completedEvent)
	}
	if released.OutcomeRevision != completedEvent.OutcomeRevision || released.EventIndex != 0 || completedEvent.EventIndex != 1 {
		t.Fatalf("completion event ordering is unstable: released=%#v completed=%#v", released, completedEvent)
	}
}

func TestFailedCommandRollsBackStateAndRevision(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	cc := commandContext()
	namespaceID := id("0199e200-0000-7000-8000-000000000001")

	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID: namespaceID, Title: "Rollback", DesiredState: "State", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value

	criterion, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner: outcome.Ref(), ExpectedVersion: outcome.Version, Title: "Required",
		Required: true, VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = criterion
	outcome.Version++

	activated, err := service.ActivateOutcome(ctx, cc, application.ActivateOutcomeCommand{
		Scope: outcome.Scope(), ExpectedVersion: outcome.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome = activated.Value

	if _, err := service.AchieveOutcome(ctx, cc, application.AchieveOutcomeCommand{
		Scope: outcome.Scope(), ExpectedVersion: outcome.Version, Reason: "premature",
	}); err == nil {
		t.Fatal("AchieveOutcome() unexpectedly succeeded without assessment")
	}

	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	persisted, err := tx.Outcomes().Get(ctx, namespaceID, outcome.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Lifecycle != domain.OutcomeLifecycleActive || persisted.Version != outcome.Version {
		t.Fatalf("failed command mutated persisted outcome: lifecycle=%q version=%d", persisted.Lifecycle, persisted.Version)
	}
	coord, err := tx.Coordination().LockOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if coord.Revision != 3 {
		t.Fatalf("revision = %d, want 3 after failed command", coord.Revision)
	}
}
