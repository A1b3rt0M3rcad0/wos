package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/storage/memory"
)

func TestSupersedeDecisionIsAtomicAndAudited(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	now := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	service, err := application.NewService(store, fixedClock{now: now}, &sequenceIDs{next: 10000})
	if err != nil {
		t.Fatal(err)
	}
	cc := commandContext()
	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  id("0199f200-0000-7000-8000-000000000001"),
		Title:        "Decision history",
		DesiredState: "Current choice is explicit",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value

	proposed, err := service.ProposeDecision(ctx, cc, application.ProposeDecisionCommand{
		Scope:        outcome.Scope(),
		Title:        "Persistence",
		Proposal:     "Choose initial durable store",
		Alternatives: []string{"SQLite", "PostgreSQL"},
	})
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := service.AcceptDecision(ctx, cc, application.AcceptDecisionCommand{
		Scope:             outcome.Scope(),
		DecisionID:        proposed.Value.ID,
		ExpectedVersion:   proposed.Value.Version,
		ChosenAlternative: "SQLite",
		Rationale:         "standalone-first",
	})
	if err != nil {
		t.Fatal(err)
	}

	superseded, err := service.SupersedeDecision(ctx, cc, application.SupersedeDecisionCommand{
		Scope:             outcome.Scope(),
		DecisionID:        accepted.Value.ID,
		ExpectedVersion:   accepted.Value.Version,
		Title:             "Persistence after scale-out",
		Proposal:          "Choose durable store after multi-node requirements",
		Alternatives:      []string{"SQLite", "PostgreSQL"},
		ChosenAlternative: "PostgreSQL",
		Rationale:         "multi-node deployment now requires shared durability",
	})
	if err != nil {
		t.Fatal(err)
	}

	if superseded.Value.Superseded.Lifecycle != domain.DecisionLifecycleSuperseded {
		t.Fatalf("old lifecycle = %q", superseded.Value.Superseded.Lifecycle)
	}
	if superseded.Value.Successor.Lifecycle != domain.DecisionLifecycleAccepted {
		t.Fatalf("successor lifecycle = %q", superseded.Value.Successor.Lifecycle)
	}
	if superseded.Value.Successor.SupersedesDecisionID == nil ||
		*superseded.Value.Successor.SupersedesDecisionID != accepted.Value.ID {
		t.Fatalf("successor does not reference superseded decision: %#v", superseded.Value.Successor)
	}

	oldRead, err := service.GetDecision(ctx, outcome.Scope(), accepted.Value.ID)
	if err != nil {
		t.Fatal(err)
	}
	newRead, err := service.GetDecision(ctx, outcome.Scope(), superseded.Value.Successor.ID)
	if err != nil {
		t.Fatal(err)
	}
	if oldRead.Value.Lifecycle != domain.DecisionLifecycleSuperseded ||
		newRead.Value.Lifecycle != domain.DecisionLifecycleAccepted {
		t.Fatalf("supersession did not commit atomically: old=%q new=%q", oldRead.Value.Lifecycle, newRead.Value.Lifecycle)
	}

	var supersessionEvents []domain.DomainEvent
	for _, event := range store.SnapshotDomainEvents(outcome.Scope()) {
		if event.CommandID == cc.CommandID && (event.EventType == "decision.proposed" ||
			event.EventType == "decision.accepted" ||
			event.EventType == "decision.superseded") {
			supersessionEvents = append(supersessionEvents, event)
		}
	}
	if len(supersessionEvents) < 3 {
		t.Fatalf("supersession events = %d, want at least 3", len(supersessionEvents))
	}
	last := supersessionEvents[len(supersessionEvents)-3:]
	if last[0].EventType != "decision.proposed" ||
		last[1].EventType != "decision.accepted" ||
		last[2].EventType != "decision.superseded" {
		t.Fatalf("unexpected supersession event order: %#v", last)
	}
	if last[0].OutcomeRevision != last[1].OutcomeRevision ||
		last[1].OutcomeRevision != last[2].OutcomeRevision {
		t.Fatal("supersession events do not share one Outcome revision")
	}
}

func TestPersistedDecisionCanCauseBlockerWithoutImplicitRelease(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	now := time.Date(2026, 10, 3, 10, 30, 0, 0, time.UTC)
	service, err := application.NewService(store, fixedClock{now: now}, &sequenceIDs{next: 10100})
	if err != nil {
		t.Fatal(err)
	}
	cc := commandContext()
	outcome := setupActiveOutcome(t, service)
	work := createTodoWork(t, service, outcome.Scope(), "wait for decision")

	proposed, err := service.ProposeDecision(ctx, cc, application.ProposeDecisionCommand{
		Scope:        outcome.Scope(),
		Title:        "Deployment target",
		Proposal:     "Choose deployment target",
		Alternatives: []string{"single-node", "multi-node"},
	})
	if err != nil {
		t.Fatal(err)
	}
	decisionRef := proposed.Value.Ref()
	blocker, err := service.CreateBlocker(ctx, cc, application.CreateBlockerCommand{
		Scope:       outcome.Scope(),
		BlockedRef:  work.Ref(),
		CauseRef:    &decisionRef,
		Description: "waiting for deployment decision",
	})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := service.AcceptDecision(ctx, cc, application.AcceptDecisionCommand{
		Scope:             outcome.Scope(),
		DecisionID:        proposed.Value.ID,
		ExpectedVersion:   proposed.Value.Version,
		ChosenAlternative: "single-node",
		Rationale:         "initial release",
	}); err != nil {
		t.Fatal(err)
	}

	state, _, err := service.GetBlockingState(ctx, work.Ref())
	if err != nil {
		t.Fatal(err)
	}
	if !state.IsBlocked {
		t.Fatal("accepting Decision implicitly released blocker")
	}
	if len(state.ActiveBlockers) != 1 ||
		state.ActiveBlockers[0].Blocker.ID != blocker.Value.ID {
		t.Fatalf("active blockers = %#v", state.ActiveBlockers)
	}
}
