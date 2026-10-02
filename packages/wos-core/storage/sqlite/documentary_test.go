package sqlite

import (
	"context"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestSQLiteWave09DocumentaryHistorySurvivesRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "wos.db")
	store, err := Open(path, Options{BusyTimeout: time.Second, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	clock := sqliteFixedClock{now: time.Date(2026, 10, 2, 19, 0, 0, 0, time.UTC)}
	service, err := application.NewService(store, clock, &sqliteSequenceIDs{prefix: "0199ed91", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	namespaceID := domain.MustParseID("0199ed92-0000-7000-8000-000000000001")
	created, err := service.CreateOutcome(ctx, sqliteCommandContext(
		"0199ed92-0000-7000-8000-000000000101", "",
	), application.CreateOutcomeCommand{
		NamespaceID: namespaceID, Title: "Documentary durability",
		DesiredState: "documentary state survives restart", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	scope := created.Value.Scope()

	artifactResult, err := service.RegisterArtifact(ctx, sqliteCommandContext(
		"0199ed92-0000-7000-8000-000000000102", "",
	), application.RegisterArtifactCommand{
		Scope: scope, ArtifactType: "document", Name: "evidence bundle",
		URI: "file:///evidence.json", MediaType: "application/json", Checksum: "sha256:abc",
	})
	if err != nil {
		t.Fatal(err)
	}
	artifact := artifactResult.Value

	evidenceResult, err := service.RegisterEvidence(ctx, sqliteCommandContext(
		"0199ed92-0000-7000-8000-000000000103", "",
	), application.RegisterEvidenceCommand{
		Scope: scope, EvidenceType: domain.EvidenceTypeSource, Description: "durable observation",
		SourceRef:  domain.SourceReference{Provider: "sqlite-test", URI: "file:///evidence.json"},
		CapturedAt: clock.now, ArtifactID: &artifact.ID, Checksum: "sha256:abc",
	})
	if err != nil {
		t.Fatal(err)
	}
	evidence := evidenceResult.Value

	proposedResult, err := service.ProposeDecision(ctx, sqliteCommandContext(
		"0199ed92-0000-7000-8000-000000000104", "",
	), application.ProposeDecisionCommand{
		Scope: scope, Title: "Storage", Proposal: "Use SQLite",
		Alternatives: []string{"SQLite", "PostgreSQL"}, ChosenAlternative: "SQLite",
		Rationale: "local-first",
	})
	if err != nil {
		t.Fatal(err)
	}
	acceptedResult, err := service.AcceptDecision(ctx, sqliteCommandContext(
		"0199ed92-0000-7000-8000-000000000105", "",
	), application.AcceptDecisionCommand{
		Scope: scope, DecisionID: proposedResult.Value.ID, ExpectedVersion: proposedResult.Value.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	accepted := acceptedResult.Value

	linkResult, err := service.CreateEvidenceLink(ctx, sqliteCommandContext(
		"0199ed92-0000-7000-8000-000000000106", "",
	), application.CreateEvidenceLinkCommand{
		Scope: scope, EvidenceID: evidence.ID, TargetRef: accepted.Ref(),
		Stance: domain.EvidenceStanceContext, Rationale: "observation considered by decision",
	})
	if err != nil {
		t.Fatal(err)
	}
	link := linkResult.Value

	supersededResult, err := service.SupersedeDecision(ctx, sqliteCommandContext(
		"0199ed92-0000-7000-8000-000000000107", "",
	), application.SupersedeDecisionCommand{
		Scope: scope, DecisionID: accepted.ID, ExpectedVersion: accepted.Version,
		Title: "Storage v2", Proposal: "Use PostgreSQL", Alternatives: []string{"SQLite", "PostgreSQL"},
		ChosenAlternative: "PostgreSQL", Rationale: "remote multi-user operation",
	})
	if err != nil {
		t.Fatal(err)
	}
	successor := supersededResult.Value.Successor

	if _, err := service.RetractEvidence(ctx, sqliteCommandContext(
		"0199ed92-0000-7000-8000-000000000108", "",
	), application.RetractEvidenceCommand{
		Scope: scope, EvidenceID: evidence.ID, ExpectedVersion: evidence.Version, Reason: "source replaced",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.WithdrawArtifact(ctx, sqliteCommandContext(
		"0199ed92-0000-7000-8000-000000000109", "",
	), application.WithdrawArtifactCommand{
		Scope: scope, ArtifactID: artifact.ID, ExpectedVersion: artifact.Version, Reason: "obsolete bundle",
	}); err != nil {
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
	restarted, err := application.NewService(reopened, clock, &sqliteSequenceIDs{prefix: "0199ed93", next: 1})
	if err != nil {
		t.Fatal(err)
	}

	persistedArtifact, err := restarted.GetArtifact(ctx, scope, artifact.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persistedArtifact.Value.Lifecycle != domain.ArtifactLifecycleWithdrawn ||
		persistedArtifact.Value.URI != artifact.URI {
		t.Fatalf("persisted artifact = %#v", persistedArtifact.Value)
	}
	persistedEvidence, err := restarted.GetEvidence(ctx, scope, evidence.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persistedEvidence.Value.Lifecycle != domain.EvidenceLifecycleRetracted ||
		persistedEvidence.Value.Description != evidence.Description {
		t.Fatalf("persisted evidence = %#v", persistedEvidence.Value)
	}
	persistedLink, err := restarted.GetEvidenceLink(ctx, scope, link.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persistedLink.Value.Stance != domain.EvidenceStanceContext {
		t.Fatalf("persisted evidence link = %#v", persistedLink.Value)
	}
	persistedSuccessor, err := restarted.GetDecision(ctx, scope, successor.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persistedSuccessor.Value.Lifecycle != domain.DecisionLifecycleAccepted ||
		persistedSuccessor.Value.SupersedesDecisionID == nil ||
		*persistedSuccessor.Value.SupersedesDecisionID != accepted.ID {
		t.Fatalf("persisted successor = %#v", persistedSuccessor.Value)
	}
	persistedPredecessor, err := restarted.GetDecision(ctx, scope, accepted.ID)
	if err != nil {
		t.Fatal(err)
	}
	if persistedPredecessor.Value.Lifecycle != domain.DecisionLifecycleSuperseded {
		t.Fatalf("persisted predecessor = %#v", persistedPredecessor.Value)
	}
}

func TestSQLiteConcurrentDecisionSupersessionHasSingleWinner(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "wos.db")
	store, err := Open(path, Options{BusyTimeout: 2 * time.Second, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	clock := sqliteFixedClock{now: time.Date(2026, 10, 2, 20, 0, 0, 0, time.UTC)}
	setup, err := application.NewService(store, clock, &sqliteSequenceIDs{prefix: "0199ed94", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	namespaceID := domain.MustParseID("0199ed95-0000-7000-8000-000000000001")
	outcome, err := setup.CreateOutcome(ctx, sqliteCommandContext(
		"0199ed95-0000-7000-8000-000000000101", "",
	), application.CreateOutcomeCommand{
		NamespaceID: namespaceID, Title: "Supersession race",
		DesiredState: "one accepted successor", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	proposed, err := setup.ProposeDecision(ctx, sqliteCommandContext(
		"0199ed95-0000-7000-8000-000000000102", "",
	), application.ProposeDecisionCommand{
		Scope: outcome.Value.Scope(), Title: "Original", Proposal: "Option A", Rationale: "initial",
	})
	if err != nil {
		t.Fatal(err)
	}
	accepted, err := setup.AcceptDecision(ctx, sqliteCommandContext(
		"0199ed95-0000-7000-8000-000000000103", "",
	), application.AcceptDecisionCommand{
		Scope: outcome.Value.Scope(), DecisionID: proposed.Value.ID, ExpectedVersion: proposed.Value.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	predecessor := accepted.Value

	serviceA, err := application.NewService(store, clock, &sqliteSequenceIDs{prefix: "0199ed96", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	serviceB, err := application.NewService(store, clock, &sqliteSequenceIDs{prefix: "0199ed97", next: 1})
	if err != nil {
		t.Fatal(err)
	}

	type result struct {
		err error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	run := func(service *application.Service, commandID, title string) {
		defer wg.Done()
		<-start
		_, err := service.SupersedeDecision(ctx, sqliteCommandContext(commandID, ""), application.SupersedeDecisionCommand{
			Scope: outcome.Value.Scope(), DecisionID: predecessor.ID, ExpectedVersion: predecessor.Version,
			Title: title, Proposal: title, Rationale: "replacement",
		})
		results <- result{err: err}
	}
	go run(serviceA, "0199ed95-0000-7000-8000-000000000104", "Successor A")
	go run(serviceB, "0199ed95-0000-7000-8000-000000000105", "Successor B")
	close(start)
	wg.Wait()
	close(results)

	successes := 0
	for item := range results {
		if item.err == nil {
			successes++
		}
	}
	if successes != 1 {
		t.Fatalf("successful supersessions = %d, want exactly 1", successes)
	}
	decisions, _, err := setup.ListDecisions(ctx, outcome.Value.Scope())
	if err != nil {
		t.Fatal(err)
	}
	acceptedSuccessors := 0
	for _, decision := range decisions {
		if decision.SupersedesDecisionID != nil && *decision.SupersedesDecisionID == predecessor.ID &&
			decision.Lifecycle == domain.DecisionLifecycleAccepted {
			acceptedSuccessors++
		}
	}
	if acceptedSuccessors != 1 {
		t.Fatalf("accepted direct successors = %d, want exactly 1", acceptedSuccessors)
	}
}
