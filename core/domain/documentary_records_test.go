package domain_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/domain"
)

func documentaryScope() domain.Scope {
	return domain.Scope{
		NamespaceID: domain.MustParseID("0199ee00-0000-7000-8000-000000000001"),
		OutcomeID:   domain.MustParseID("0199ee00-0000-7000-8000-000000000002"),
	}
}

func documentaryActor() domain.ActorRef {
	return domain.ActorRef{Kind: domain.ActorKindService, Provider: "ci", ID: "benchmark"}
}

func TestArtifactIsImmutableExceptWithdrawalLifecycle(t *testing.T) {
	now := time.Date(2026, 10, 3, 6, 0, 0, 0, time.UTC)
	producedAt := now.Add(-time.Minute)
	artifact, err := domain.NewArtifact(
		domain.MustParseID("0199ee00-0000-7000-8000-000000000010"),
		documentaryScope(),
		"report",
		"benchmark report",
		"https://example.test/report",
		"application/pdf",
		"sha256:abc",
		"commit-123",
		documentaryActor(),
		&producedAt,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	originalURI := artifact.URI
	if err := artifact.Withdraw("superseded by corrected report", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if artifact.Lifecycle != domain.ArtifactLifecycleWithdrawn || artifact.URI != originalURI {
		t.Fatalf("withdrawal changed immutable content: %#v", artifact)
	}
	if artifact.Version != 2 {
		t.Fatalf("artifact version = %d, want 2", artifact.Version)
	}
}

func TestEvidenceRequiresProvenanceAndPreservesObservationOnRetraction(t *testing.T) {
	now := time.Date(2026, 10, 3, 6, 10, 0, 0, time.UTC)
	measurement := &domain.Measurement{
		Value:      json.RawMessage("180"),
		Unit:       "ms",
		Method:     "p95",
		Conditions: json.RawMessage(`{"requests":10000}`),
	}
	evidence, err := domain.NewEvidence(
		domain.MustParseID("0199ee00-0000-7000-8000-000000000020"),
		documentaryScope(),
		domain.EvidenceTypeMeasurement,
		"p95 latency was 180ms",
		domain.ExternalReference{Provider: "benchmark-ci", ID: "run-42"},
		documentaryActor(),
		now.Add(-time.Minute),
		nil,
		measurement,
		"scenario-c1",
		"sha256:def",
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	originalDescription := evidence.Description
	originalValue := string(evidence.Measurement.Value)
	if err := evidence.Retract("benchmark configuration was invalid", now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if evidence.Lifecycle != domain.EvidenceLifecycleRetracted ||
		evidence.Description != originalDescription ||
		string(evidence.Measurement.Value) != originalValue {
		t.Fatalf("retraction rewrote evidence observation: %#v", evidence)
	}
}

func TestEvidenceLinkKeepsStanceOnLink(t *testing.T) {
	now := time.Date(2026, 10, 3, 6, 20, 0, 0, time.UTC)
	scope := documentaryScope()
	link, err := domain.NewEvidenceLink(
		domain.MustParseID("0199ee00-0000-7000-8000-000000000030"),
		scope,
		domain.MustParseID("0199ee00-0000-7000-8000-000000000020"),
		domain.EntityRef{Scope: scope, Kind: domain.EntityKindObjective, ID: domain.MustParseID("0199ee00-0000-7000-8000-000000000031")},
		nil,
		domain.EvidenceStanceContradicts,
		"measurement contradicts the current claim",
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if link.Stance != domain.EvidenceStanceContradicts {
		t.Fatalf("stance = %q", link.Stance)
	}
	if err := link.Retract(now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if link.Lifecycle != domain.EvidenceLinkLifecycleRetracted {
		t.Fatalf("link lifecycle = %q", link.Lifecycle)
	}
}

func TestDecisionFreezesAcceptedContentAndSupportsSupersessionState(t *testing.T) {
	now := time.Date(2026, 10, 3, 6, 30, 0, 0, time.UTC)
	decision, err := domain.NewDecision(
		domain.MustParseID("0199ee00-0000-7000-8000-000000000040"),
		documentaryScope(),
		"Storage engine",
		"Choose the durable store",
		"",
		[]string{"SQLite", "PostgreSQL"},
		nil,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := decision.ReviseProposal(
		"Storage engine",
		"Choose the initial durable store",
		"local-first release",
		[]string{"SQLite", "PostgreSQL"},
		now.Add(time.Minute),
	); err != nil {
		t.Fatal(err)
	}
	if err := decision.Accept("SQLite", "best fit for standalone M1", documentaryActor(), now.Add(2*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := decision.ReviseProposal(
		"Changed",
		"Changed",
		"Changed",
		[]string{"Other"},
		now.Add(3*time.Minute),
	); err == nil {
		t.Fatal("accepted decision unexpectedly remained editable")
	}
	if err := decision.MarkSuperseded(now.Add(4 * time.Minute)); err != nil {
		t.Fatal(err)
	}
	if decision.Lifecycle != domain.DecisionLifecycleSuperseded ||
		decision.ChosenAlternative != "SQLite" ||
		decision.Rationale != "best fit for standalone M1" {
		t.Fatalf("supersession did not preserve accepted content: %#v", decision)
	}
}
