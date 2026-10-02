package application_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/storage/memory"
)

func TestDocumentaryCommandsPersistImmutableRecordsAndEvents(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	now := time.Date(2026, 10, 3, 7, 0, 0, 0, time.UTC)
	service, err := application.NewService(store, fixedClock{now: now}, &sequenceIDs{next: 7000})
	if err != nil {
		t.Fatal(err)
	}
	cc := commandContext()
	namespaceID := id("0199ef00-0000-7000-8000-000000000001")

	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Documented result",
		DesiredState: "facts and decisions are resumable",
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
		URI:           "https://example.test/benchmark",
		MediaType:     "application/json",
		Checksum:      "sha256:abc",
		SourceVersion: "commit-42",
		ProducerRef:   domain.ActorRef{Kind: domain.ActorKindService, Provider: "ci", ID: "benchmark"},
	})
	if err != nil {
		t.Fatal(err)
	}

	evidence, err := service.RegisterEvidence(ctx, cc, application.RegisterEvidenceCommand{
		Scope:        outcome.Scope(),
		EvidenceType: domain.EvidenceTypeMeasurement,
		Description:  "p95 latency was 180ms",
		SourceRef:    domain.ExternalReference{Provider: "benchmark-ci", ID: "run-42"},
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
		Checksum:      "sha256:def",
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
	decision := proposed.Value
	accepted, err := service.AcceptDecision(ctx, cc, application.AcceptDecisionCommand{
		Scope:             outcome.Scope(),
		DecisionID:        decision.ID,
		ExpectedVersion:   decision.Version,
		ChosenAlternative: "SQLite",
		Rationale:         "standalone-first release",
	})
	if err != nil {
		t.Fatal(err)
	}
	if accepted.Value.Lifecycle != domain.DecisionLifecycleAccepted {
		t.Fatalf("decision lifecycle = %q", accepted.Value.Lifecycle)
	}

	retracted, err := service.RetractEvidence(ctx, cc, application.RetractEvidenceCommand{
		Scope:           outcome.Scope(),
		EvidenceID:      evidence.Value.ID,
		ExpectedVersion: evidence.Value.Version,
		Reason:          "benchmark configuration invalid",
	})
	if err != nil {
		t.Fatal(err)
	}
	if retracted.Value.Description != evidence.Value.Description ||
		retracted.Value.Lifecycle != domain.EvidenceLifecycleRetracted {
		t.Fatalf("retraction rewrote evidence: %#v", retracted.Value)
	}

	withdrawn, err := service.WithdrawArtifact(ctx, cc, application.WithdrawArtifactCommand{
		Scope:           outcome.Scope(),
		ArtifactID:      artifact.Value.ID,
		ExpectedVersion: artifact.Value.Version,
		Reason:          "report replaced",
	})
	if err != nil {
		t.Fatal(err)
	}
	if withdrawn.Value.URI != artifact.Value.URI ||
		withdrawn.Value.Lifecycle != domain.ArtifactLifecycleWithdrawn {
		t.Fatalf("withdrawal rewrote artifact: %#v", withdrawn.Value)
	}

	events := store.SnapshotDomainEvents(outcome.Scope())
	found := map[string]bool{}
	for _, event := range events {
		found[event.EventType] = true
	}
	for _, eventType := range []string{
		"artifact.registered",
		"evidence.registered",
		"decision.proposed",
		"decision.accepted",
		"evidence.retracted",
		"artifact.withdrawn",
	} {
		if !found[eventType] {
			t.Fatalf("missing domain event %q", eventType)
		}
	}
}

func TestDocumentaryMutationsAllowTerminalOutcomeButRejectArchivedOutcome(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	now := time.Date(2026, 10, 3, 7, 30, 0, 0, time.UTC)
	service, err := application.NewService(store, fixedClock{now: now}, &sequenceIDs{next: 8000})
	if err != nil {
		t.Fatal(err)
	}
	cc := commandContext()
	namespaceID := id("0199ef00-0000-7000-8000-000000000011")

	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Terminal documentation",
		DesiredState: "documentable after conclusion",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value
	if _, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner:            outcome.Ref(),
		ExpectedVersion:  outcome.Version,
		Title:            "verified",
		Required:         true,
		VerificationMode: domain.VerificationModeAttestation,
	}); err != nil {
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

	criterionID := outcome.Criteria.Items[0].ID
	if _, err := service.AttestCriterion(ctx, cc, application.AttestCriterionCommand{
		Owner:             outcome.Ref(),
		CriterionID:       criterionID,
		CriterionRevision: domain.InitialCriterionRevision,
		ExpectedVersion:   outcome.Version,
		Result:            domain.AssessmentResultMet,
		Rationale:         "verified",
	}); err != nil {
		t.Fatal(err)
	}
	outcome.Version++
	achieved, err := service.AchieveOutcome(ctx, cc, application.AchieveOutcomeCommand{
		Scope:           outcome.Scope(),
		ExpectedVersion: outcome.Version,
		Reason:          "done",
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome = achieved.Value

	if _, err := service.RegisterArtifact(ctx, cc, application.RegisterArtifactCommand{
		Scope:        outcome.Scope(),
		ArtifactType: "report",
		Name:         "post-conclusion report",
		URI:          "https://example.test/report",
		ProducerRef:  cc.Actor,
	}); err != nil {
		t.Fatalf("terminal outcome rejected documentary record: %v", err)
	}

	archived, err := service.ArchiveOutcome(ctx, cc, application.ArchiveOutcomeCommand{
		Scope:           outcome.Scope(),
		ExpectedVersion: outcome.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome = archived.Value
	if _, err := service.RegisterArtifact(ctx, cc, application.RegisterArtifactCommand{
		Scope:        outcome.Scope(),
		ArtifactType: "report",
		Name:         "archived report",
		URI:          "https://example.test/archived",
		ProducerRef:  cc.Actor,
	}); err == nil {
		t.Fatal("archived outcome unexpectedly accepted documentary mutation")
	}
}
