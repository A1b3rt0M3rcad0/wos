package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/core/application"
	"github.com/A1b3rt0M3rcad0/wos/core/domain"
	"github.com/A1b3rt0M3rcad0/wos/storage/memory"
)

func TestOutcomeStateIncludesDocumentaryHistoryAndCurrentProjection(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	now := time.Date(2026, 10, 3, 11, 0, 0, 0, time.UTC)
	service, err := application.NewService(store, fixedClock{now: now}, &sequenceIDs{next: 11000})
	if err != nil {
		t.Fatal(err)
	}
	cc := commandContext()
	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  id("0199f300-0000-7000-8000-000000000001"),
		Title:        "Resumable documentary state",
		DesiredState: "consumer understands facts and choices from one state read",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value

	artifact, err := service.RegisterArtifact(ctx, cc, application.RegisterArtifactCommand{
		Scope:        outcome.Scope(),
		ArtifactType: "report",
		Name:         "old report",
		URI:          "https://example.test/old-report",
		ProducerRef:  cc.Actor,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.WithdrawArtifact(ctx, cc, application.WithdrawArtifactCommand{
		Scope:           outcome.Scope(),
		ArtifactID:      artifact.Value.ID,
		ExpectedVersion: artifact.Value.Version,
		Reason:          "replaced",
	}); err != nil {
		t.Fatal(err)
	}

	currentEvidence, err := service.RegisterEvidence(ctx, cc, application.RegisterEvidenceCommand{
		Scope:        outcome.Scope(),
		EvidenceType: domain.EvidenceTypeAttestation,
		Description:  "current attestation",
		SourceRef:    domain.ExternalReference{Provider: "review", ID: "attestation-current"},
		ProducerRef:  cc.Actor,
		CapturedAt:   now,
	})
	if err != nil {
		t.Fatal(err)
	}
	retractedEvidence, err := service.RegisterEvidence(ctx, cc, application.RegisterEvidenceCommand{
		Scope:        outcome.Scope(),
		EvidenceType: domain.EvidenceTypeSource,
		Description:  "obsolete source",
		SourceRef:    domain.ExternalReference{Provider: "docs", ID: "source-old"},
		ProducerRef:  cc.Actor,
		CapturedAt:   now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RetractEvidence(ctx, cc, application.RetractEvidenceCommand{
		Scope:           outcome.Scope(),
		EvidenceID:      retractedEvidence.Value.ID,
		ExpectedVersion: retractedEvidence.Value.Version,
		Reason:          "source superseded",
	}); err != nil {
		t.Fatal(err)
	}

	activeLink, err := service.CreateEvidenceLink(ctx, cc, application.CreateEvidenceLinkCommand{
		Scope:      outcome.Scope(),
		EvidenceID: currentEvidence.Value.ID,
		TargetRef:  outcome.Ref(),
		Stance:     domain.EvidenceStanceSupports,
		Rationale:  "supports current outcome context",
	})
	if err != nil {
		t.Fatal(err)
	}
	retractedLink, err := service.CreateEvidenceLink(ctx, cc, application.CreateEvidenceLinkCommand{
		Scope:      outcome.Scope(),
		EvidenceID: currentEvidence.Value.ID,
		TargetRef:  outcome.Ref(),
		Stance:     domain.EvidenceStanceContext,
		Rationale:  "temporary context",
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.RetractEvidenceLink(ctx, cc, application.RetractEvidenceLinkCommand{
		Scope:           outcome.Scope(),
		EvidenceLinkID:  retractedLink.Value.ID,
		ExpectedVersion: retractedLink.Value.Version,
	}); err != nil {
		t.Fatal(err)
	}

	proposed, err := service.ProposeDecision(ctx, cc, application.ProposeDecisionCommand{
		Scope:        outcome.Scope(),
		Title:        "Store",
		Proposal:     "Choose store",
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
		Rationale:         "initial standalone release",
	})
	if err != nil {
		t.Fatal(err)
	}
	supersession, err := service.SupersedeDecision(ctx, cc, application.SupersedeDecisionCommand{
		Scope:             outcome.Scope(),
		DecisionID:        accepted.Value.ID,
		ExpectedVersion:   accepted.Value.Version,
		Title:             "Store after scale-out",
		Proposal:          "Choose shared store",
		Alternatives:      []string{"SQLite", "PostgreSQL"},
		ChosenAlternative: "PostgreSQL",
		Rationale:         "shared durability required",
	})
	if err != nil {
		t.Fatal(err)
	}

	state, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Documentary.Artifacts) != 1 || len(state.Documentary.RegisteredArtifacts) != 0 {
		t.Fatalf("artifact projection = %#v", state.Documentary)
	}
	if len(state.Documentary.Evidence) != 2 || len(state.Documentary.RegisteredEvidence) != 1 {
		t.Fatalf("evidence projection = %#v", state.Documentary)
	}
	if state.Documentary.RegisteredEvidence[0].ID != currentEvidence.Value.ID {
		t.Fatalf("registered evidence = %#v", state.Documentary.RegisteredEvidence)
	}
	if len(state.Documentary.EvidenceLinks) != 2 || len(state.Documentary.ActiveEvidenceLinks) != 1 {
		t.Fatalf("evidence link projection = %#v", state.Documentary)
	}
	if state.Documentary.ActiveEvidenceLinks[0].ID != activeLink.Value.ID {
		t.Fatalf("active evidence links = %#v", state.Documentary.ActiveEvidenceLinks)
	}
	if len(state.Documentary.Decisions) != 2 || len(state.Documentary.CurrentDecisions) != 1 {
		t.Fatalf("decision projection = %#v", state.Documentary)
	}
	if state.Documentary.CurrentDecisions[0].ID != supersession.Value.Successor.ID ||
		state.Documentary.CurrentDecisions[0].ChosenAlternative != "PostgreSQL" {
		t.Fatalf("current decisions = %#v", state.Documentary.CurrentDecisions)
	}
}
