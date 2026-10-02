package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/storage/memory"
)

func TestEvidenceLinkValidatesEvidenceTargetAndCriterion(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	now := time.Date(2026, 10, 3, 9, 0, 0, 0, time.UTC)
	service, err := application.NewService(store, fixedClock{now: now}, &sequenceIDs{next: 9000})
	if err != nil {
		t.Fatal(err)
	}
	cc := commandContext()
	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  id("0199f100-0000-7000-8000-000000000001"),
		Title:        "Evidence links",
		DesiredState: "Evidence is explicitly connected to claims",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value

	added, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner:            outcome.Ref(),
		ExpectedVersion:  outcome.Version,
		Title:            "Latency verified",
		Required:         true,
		VerificationMode: domain.VerificationModeEvidenceReview,
	})
	if err != nil {
		t.Fatal(err)
	}
	criterionID := added.Value.ID

	evidence, err := service.RegisterEvidence(ctx, cc, application.RegisterEvidenceCommand{
		Scope:        outcome.Scope(),
		EvidenceType: domain.EvidenceTypeTestResult,
		Description:  "benchmark passed",
		SourceRef:    domain.ExternalReference{Provider: "ci", ID: "run-100"},
		ProducerRef:  domain.ActorRef{Kind: domain.ActorKindService, Provider: "ci", ID: "bench"},
		CapturedAt:   now,
	})
	if err != nil {
		t.Fatal(err)
	}

	link, err := service.CreateEvidenceLink(ctx, cc, application.CreateEvidenceLinkCommand{
		Scope:       outcome.Scope(),
		EvidenceID:  evidence.Value.ID,
		TargetRef:   outcome.Ref(),
		CriterionID: &criterionID,
		Stance:      domain.EvidenceStanceSupports,
		Rationale:   "benchmark result supports the latency criterion",
	})
	if err != nil {
		t.Fatal(err)
	}
	if link.Value.CriterionID == nil || *link.Value.CriterionID != criterionID {
		t.Fatalf("criterion link = %#v", link.Value)
	}

	missingCriterion := id("0199f100-0000-7000-8000-000000000099")
	if _, err := service.CreateEvidenceLink(ctx, cc, application.CreateEvidenceLinkCommand{
		Scope:       outcome.Scope(),
		EvidenceID:  evidence.Value.ID,
		TargetRef:   outcome.Ref(),
		CriterionID: &missingCriterion,
		Stance:      domain.EvidenceStanceContext,
		Rationale:   "invalid criterion",
	}); err == nil {
		t.Fatal("CreateEvidenceLink accepted a criterion that does not belong to target")
	}

	missingEvidence := id("0199f100-0000-7000-8000-000000000098")
	if _, err := service.CreateEvidenceLink(ctx, cc, application.CreateEvidenceLinkCommand{
		Scope:      outcome.Scope(),
		EvidenceID: missingEvidence,
		TargetRef:  outcome.Ref(),
		Stance:     domain.EvidenceStanceContext,
		Rationale:  "missing evidence",
	}); err == nil {
		t.Fatal("CreateEvidenceLink accepted missing evidence")
	}

	retractedEvidence, err := service.RetractEvidence(ctx, cc, application.RetractEvidenceCommand{
		Scope:           outcome.Scope(),
		EvidenceID:      evidence.Value.ID,
		ExpectedVersion: evidence.Value.Version,
		Reason:          "source invalidated",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateEvidenceLink(ctx, cc, application.CreateEvidenceLinkCommand{
		Scope:      outcome.Scope(),
		EvidenceID: retractedEvidence.Value.ID,
		TargetRef:  outcome.Ref(),
		Stance:     domain.EvidenceStanceContext,
		Rationale:  "retracted evidence",
	}); err == nil {
		t.Fatal("CreateEvidenceLink accepted retracted evidence")
	}
}

func TestEvidenceLinkLifecycleIsAudited(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	now := time.Date(2026, 10, 3, 9, 15, 0, 0, time.UTC)
	service, err := application.NewService(store, fixedClock{now: now}, &sequenceIDs{next: 9100})
	if err != nil {
		t.Fatal(err)
	}
	cc := commandContext()
	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  id("0199f101-0000-7000-8000-000000000001"),
		Title:        "Evidence link lifecycle",
		DesiredState: "links remain auditable",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value
	evidence, err := service.RegisterEvidence(ctx, cc, application.RegisterEvidenceCommand{
		Scope:        outcome.Scope(),
		EvidenceType: domain.EvidenceTypeAttestation,
		Description:  "reviewer attested the current state",
		SourceRef:    domain.ExternalReference{Provider: "review", ID: "attestation-1"},
		ProducerRef:  cc.Actor,
		CapturedAt:   now,
	})
	if err != nil {
		t.Fatal(err)
	}
	link, err := service.CreateEvidenceLink(ctx, cc, application.CreateEvidenceLinkCommand{
		Scope:      outcome.Scope(),
		EvidenceID: evidence.Value.ID,
		TargetRef:  outcome.Ref(),
		Stance:     domain.EvidenceStanceContext,
		Rationale:  "documents the outcome context",
	})
	if err != nil {
		t.Fatal(err)
	}
	retracted, err := service.RetractEvidenceLink(ctx, cc, application.RetractEvidenceLinkCommand{
		Scope:           outcome.Scope(),
		EvidenceLinkID:  link.Value.ID,
		ExpectedVersion: link.Value.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if retracted.Value.Lifecycle != domain.EvidenceLinkLifecycleRetracted {
		t.Fatalf("link lifecycle = %q", retracted.Value.Lifecycle)
	}

	read, err := service.GetEvidenceLink(ctx, outcome.Scope(), link.Value.ID)
	if err != nil {
		t.Fatal(err)
	}
	if read.Value.Version != retracted.Value.Version {
		t.Fatalf("persisted link version = %d, want %d", read.Value.Version, retracted.Value.Version)
	}

	found := map[string]bool{}
	for _, event := range store.SnapshotDomainEvents(outcome.Scope()) {
		found[event.EventType] = true
	}
	if !found["evidence.link_created"] || !found["evidence.link_retracted"] {
		t.Fatalf("evidence link events missing: %#v", found)
	}
}
