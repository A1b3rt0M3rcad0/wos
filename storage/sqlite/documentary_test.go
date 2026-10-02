package sqlite

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func documentarySQLiteContext(commandID string) domain.CommandContext {
	return domain.CommandContext{
		PrincipalID: "documentary-operator",
		Actor: domain.ActorRef{
			Kind:     domain.ActorKindHuman,
			Provider: "sqlite-test",
			ID:       "documentary-operator",
		},
		CommandID: domain.MustParseID(commandID),
	}
}

func TestSQLiteDocumentaryRecordsSurviveRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "wos.db")
	store, err := Open(path, Options{BusyTimeout: time.Second, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 10, 3, 8, 0, 0, 0, time.UTC)
	service, err := application.NewService(
		store,
		sqliteFixedClock{now: now},
		&sqliteSequenceIDs{prefix: "0199f010", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	cc := documentarySQLiteContext("0199f011-0000-7000-8000-000000000001")
	namespaceID := domain.MustParseID("0199f012-0000-7000-8000-000000000001")

	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Documentary durability",
		DesiredState: "records survive restart",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value

	artifact, err := service.RegisterArtifact(ctx, cc, application.RegisterArtifactCommand{
		Scope:         outcome.Scope(),
		ArtifactType:  "report",
		Name:          "benchmark report",
		URI:           "https://example.test/report",
		MediaType:     "application/json",
		Checksum:      "sha256:artifact",
		SourceVersion: "commit-9f0d2a",
		ProducerRef:   domain.ActorRef{Kind: domain.ActorKindService, Provider: "ci", ID: "benchmark"},
	})
	if err != nil {
		t.Fatal(err)
	}

	evidence, err := service.RegisterEvidence(ctx, cc, application.RegisterEvidenceCommand{
		Scope:        outcome.Scope(),
		EvidenceType: domain.EvidenceTypeMeasurement,
		Description:  "p95 latency was 180ms",
		SourceRef:    domain.ExternalReference{Provider: "benchmark-ci", ID: "run-99"},
		ProducerRef:  domain.ActorRef{Kind: domain.ActorKindService, Provider: "ci", ID: "benchmark"},
		CapturedAt:   now.Add(-time.Minute),
		ArtifactID:   &artifact.Value.ID,
		Measurement: &domain.Measurement{
			Value:      json.RawMessage("180"),
			Unit:       "ms",
			Method:     "p95",
			Conditions: json.RawMessage(`{"requests":10000}`),
		},
		SourceVersion: "scenario-c1",
		Checksum:      "sha256:evidence",
	})
	if err != nil {
		t.Fatal(err)
	}

	proposed, err := service.ProposeDecision(ctx, cc, application.ProposeDecisionCommand{
		Scope:        outcome.Scope(),
		Title:        "Storage engine",
		Proposal:     "Choose the first durable store",
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
		Rationale:         "standalone-first release",
	})
	if err != nil {
		t.Fatal(err)
	}

	retracted, err := service.RetractEvidence(ctx, cc, application.RetractEvidenceCommand{
		Scope:           outcome.Scope(),
		EvidenceID:      evidence.Value.ID,
		ExpectedVersion: evidence.Value.Version,
		Reason:          "invalid benchmark configuration",
	})
	if err != nil {
		t.Fatal(err)
	}
	withdrawn, err := service.WithdrawArtifact(ctx, cc, application.WithdrawArtifactCommand{
		Scope:           outcome.Scope(),
		ArtifactID:      artifact.Value.ID,
		ExpectedVersion: artifact.Value.Version,
		Reason:          "replaced report",
	})
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

	readService, err := application.NewService(
		reopened,
		sqliteFixedClock{now: now.Add(time.Hour)},
		&sqliteSequenceIDs{prefix: "0199f013", next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	gotArtifact, err := readService.GetArtifact(ctx, outcome.Scope(), artifact.Value.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotArtifact.Value.Lifecycle != domain.ArtifactLifecycleWithdrawn ||
		gotArtifact.Value.URI != artifact.Value.URI ||
		gotArtifact.Value.WithdrawalReason != withdrawn.Value.WithdrawalReason {
		t.Fatalf("artifact after restart = %#v", gotArtifact.Value)
	}
	gotEvidence, err := readService.GetEvidence(ctx, outcome.Scope(), evidence.Value.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotEvidence.Value.Lifecycle != domain.EvidenceLifecycleRetracted ||
		gotEvidence.Value.Description != evidence.Value.Description ||
		gotEvidence.Value.RetractionReason != retracted.Value.RetractionReason {
		t.Fatalf("evidence after restart = %#v", gotEvidence.Value)
	}
	gotDecision, err := readService.GetDecision(ctx, outcome.Scope(), accepted.Value.ID)
	if err != nil {
		t.Fatal(err)
	}
	if gotDecision.Value.Lifecycle != domain.DecisionLifecycleAccepted ||
		gotDecision.Value.ChosenAlternative != "SQLite" ||
		gotDecision.Value.Rationale != "standalone-first release" {
		t.Fatalf("decision after restart = %#v", gotDecision.Value)
	}

	events, err := reopened.SnapshotDomainEvents(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	found := map[string]bool{}
	for _, event := range events {
		found[event.EventType] = true
	}
	for _, eventType := range []string{
		"artifact.registered",
		"artifact.withdrawn",
		"evidence.registered",
		"evidence.retracted",
		"decision.proposed",
		"decision.accepted",
	} {
		if !found[eventType] {
			t.Fatalf("missing persisted event %q", eventType)
		}
	}
}
