package sqlite

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func TestSQLiteDecisionSupersessionRaceKeepsOneDirectSuccessor(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "wos.db")
	store, err := Open(path, Options{BusyTimeout: 2 * time.Second, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	now := time.Date(2026, 10, 3, 13, 0, 0, 0, time.UTC)
	setup, err := application.NewService(
		store,
		sqliteFixedClock{now: now},
		&sqliteSequenceIDs{prefix: "0199f500", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	created, err := setup.CreateOutcome(
		ctx,
		documentarySQLiteContext("0199f501-0000-7000-8000-000000000001"),
		application.CreateOutcomeCommand{
			NamespaceID:  domain.MustParseID("0199f502-0000-7000-8000-000000000001"),
			Title:        "Supersession race",
			DesiredState: "Only one direct successor survives",
			Priority:     domain.PriorityNormal,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value
	proposed, err := setup.ProposeDecision(
		ctx,
		documentarySQLiteContext("0199f501-0000-7000-8000-000000000002"),
		application.ProposeDecisionCommand{
			Scope:        outcome.Scope(),
			Title:        "Store",
			Proposal:     "Choose durable store",
			Alternatives: []string{"SQLite", "PostgreSQL", "DistributedSQL"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := setup.AcceptDecision(
		ctx,
		documentarySQLiteContext("0199f501-0000-7000-8000-000000000003"),
		application.AcceptDecisionCommand{
			Scope:             outcome.Scope(),
			DecisionID:        proposed.Value.ID,
			ExpectedVersion:   proposed.Value.Version,
			ChosenAlternative: "SQLite",
			Rationale:         "standalone-first",
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	serviceA, err := application.NewService(
		store,
		sqliteFixedClock{now: now.Add(time.Minute)},
		&sqliteSequenceIDs{prefix: "0199f510", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	serviceB, err := application.NewService(
		store,
		sqliteFixedClock{now: now.Add(time.Minute)},
		&sqliteSequenceIDs{prefix: "0199f520", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}

	type result struct {
		value application.MutationResult[application.SupersedeDecisionResult]
		err   error
	}
	results := make(chan result, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	run := func(
		service *application.Service,
		cc domain.CommandContext,
		title, chosen, rationale string,
	) {
		defer wg.Done()
		value, err := service.SupersedeDecision(ctx, cc, application.SupersedeDecisionCommand{
			Scope:             outcome.Scope(),
			DecisionID:        accepted.Value.ID,
			ExpectedVersion:   accepted.Value.Version,
			Title:             title,
			Proposal:          "Choose replacement durable store",
			Alternatives:      []string{"PostgreSQL", "DistributedSQL"},
			ChosenAlternative: chosen,
			Rationale:         rationale,
		})
		results <- result{value: value, err: err}
	}
	go run(
		serviceA,
		documentarySQLiteContext("0199f501-0000-7000-8000-000000000010"),
		"Shared SQL",
		"PostgreSQL",
		"shared database required",
	)
	go run(
		serviceB,
		documentarySQLiteContext("0199f501-0000-7000-8000-000000000011"),
		"Distributed SQL",
		"DistributedSQL",
		"distributed topology required",
	)
	wg.Wait()
	close(results)

	successes := 0
	var winner domain.Decision
	for item := range results {
		if item.err == nil {
			successes++
			winner = item.value.Value.Successor
		}
	}
	if successes != 1 {
		t.Fatalf("successful supersessions = %d, want exactly 1", successes)
	}

	decisions, _, err := setup.ListDecisions(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if len(decisions) != 2 {
		t.Fatalf("decision count = %d, want original + one successor", len(decisions))
	}
	current := 0
	for _, decision := range decisions {
		if decision.Lifecycle == domain.DecisionLifecycleAccepted {
			current++
			if decision.ID != winner.ID {
				t.Fatalf("accepted decision = %s, winner = %s", decision.ID, winner.ID)
			}
		}
	}
	if current != 1 {
		t.Fatalf("current accepted decisions = %d, want 1", current)
	}
}

func TestSQLiteDocumentaryGraphAndSupersessionSurviveRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "wos.db")
	store, err := Open(path, Options{BusyTimeout: time.Second, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 3, 13, 30, 0, 0, time.UTC)
	service, err := application.NewService(
		store,
		sqliteFixedClock{now: now},
		&sqliteSequenceIDs{prefix: "0199f530", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateOutcome(
		ctx,
		documentarySQLiteContext("0199f531-0000-7000-8000-000000000001"),
		application.CreateOutcomeCommand{
			NamespaceID:  domain.MustParseID("0199f532-0000-7000-8000-000000000001"),
			Title:        "Documentary graph restart",
			DesiredState: "History remains resumable",
			Priority:     domain.PriorityNormal,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value

	evidence, err := service.RegisterEvidence(
		ctx,
		documentarySQLiteContext("0199f531-0000-7000-8000-000000000002"),
		application.RegisterEvidenceCommand{
			Scope:        outcome.Scope(),
			EvidenceType: domain.EvidenceTypeAttestation,
			Description:  "operator observed the initial choice",
			SourceRef:    domain.ExternalReference{Provider: "review", ID: "review-1"},
			ProducerRef:  domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "review", ID: "reviewer"},
			CapturedAt:   now,
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	proposed, err := service.ProposeDecision(
		ctx,
		documentarySQLiteContext("0199f531-0000-7000-8000-000000000003"),
		application.ProposeDecisionCommand{
			Scope:        outcome.Scope(),
			Title:        "Store",
			Proposal:     "Choose storage",
			Alternatives: []string{"SQLite", "PostgreSQL"},
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := service.AcceptDecision(
		ctx,
		documentarySQLiteContext("0199f531-0000-7000-8000-000000000004"),
		application.AcceptDecisionCommand{
			Scope:             outcome.Scope(),
			DecisionID:        proposed.Value.ID,
			ExpectedVersion:   proposed.Value.Version,
			ChosenAlternative: "SQLite",
			Rationale:         "first standalone release",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	link, err := service.CreateEvidenceLink(
		ctx,
		documentarySQLiteContext("0199f531-0000-7000-8000-000000000005"),
		application.CreateEvidenceLinkCommand{
			Scope:      outcome.Scope(),
			EvidenceID: evidence.Value.ID,
			TargetRef:  accepted.Value.Ref(),
			Stance:     domain.EvidenceStanceContext,
			Rationale:  "records the evidence considered with the initial decision",
		},
	)
	if err != nil {
		t.Fatal(err)
	}
	superseded, err := service.SupersedeDecision(
		ctx,
		documentarySQLiteContext("0199f531-0000-7000-8000-000000000006"),
		application.SupersedeDecisionCommand{
			Scope:             outcome.Scope(),
			DecisionID:        accepted.Value.ID,
			ExpectedVersion:   accepted.Value.Version,
			Title:             "Store after scale-out",
			Proposal:          "Choose shared storage",
			Alternatives:      []string{"SQLite", "PostgreSQL"},
			ChosenAlternative: "PostgreSQL",
			Rationale:         "multi-node deployment requires shared durability",
		},
	)
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
	restarted, err := application.NewService(
		reopened,
		sqliteFixedClock{now: now.Add(time.Hour)},
		&sqliteSequenceIDs{prefix: "0199f540", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}

	state, err := restarted.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Documentary.EvidenceLinks) != 1 ||
		state.Documentary.EvidenceLinks[0].ID != link.Value.ID {
		t.Fatalf("evidence links after restart = %#v", state.Documentary.EvidenceLinks)
	}
	if len(state.Documentary.Decisions) != 2 ||
		len(state.Documentary.CurrentDecisions) != 1 {
		t.Fatalf("decision state after restart = %#v", state.Documentary)
	}
	if state.Documentary.CurrentDecisions[0].ID != superseded.Value.Successor.ID {
		t.Fatalf("current decision after restart = %#v", state.Documentary.CurrentDecisions)
	}
	old, err := restarted.GetDecision(ctx, outcome.Scope(), accepted.Value.ID)
	if err != nil {
		t.Fatal(err)
	}
	if old.Value.Lifecycle != domain.DecisionLifecycleSuperseded {
		t.Fatalf("old lifecycle after restart = %q", old.Value.Lifecycle)
	}
	if state.Documentary.CurrentDecisions[0].SupersedesDecisionID == nil ||
		*state.Documentary.CurrentDecisions[0].SupersedesDecisionID != old.Value.ID {
		t.Fatalf("supersession chain after restart = %#v", state.Documentary.CurrentDecisions[0])
	}
}
