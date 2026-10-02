package application_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

type wave09IDs struct {
	next uint64
}

func (g *wave09IDs) NewID() (domain.ID, error) {
	value := domain.MustParseID(fmt.Sprintf("0199ed90-0000-7000-8000-%012x", g.next))
	g.next++
	return value, nil
}

func newWave09Service(t *testing.T) (*application.Service, *memory.Store) {
	t.Helper()
	store := memory.New()
	service, err := application.NewService(
		store,
		fixedClock{now: time.Date(2026, 10, 2, 18, 0, 0, 0, time.UTC)},
		&wave09IDs{next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	return service, store
}

func TestWave09DocumentaryRecordsAndDecisionSupersession(t *testing.T) {
	ctx := context.Background()
	service, store := newWave09Service(t)
	outcome := setupActiveOutcome(t, service)
	scope := outcome.Scope()
	cc := commandContext()

	artifactResult, err := service.RegisterArtifact(ctx, cc, application.RegisterArtifactCommand{
		Scope: scope, ArtifactType: "document", Name: "benchmark", URI: "file:///benchmark.json", MediaType: "application/json", Checksum: "sha256:abc", SourceVersion: "v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	artifact := artifactResult.Value

	evidenceResult, err := service.RegisterEvidence(ctx, cc, application.RegisterEvidenceCommand{
		Scope: scope, EvidenceType: domain.EvidenceTypeSource, Description: "benchmark result",
		SourceRef:  domain.SourceReference{Provider: "test", URI: "file:///benchmark.json"},
		CapturedAt: time.Date(2026, 10, 2, 18, 0, 0, 0, time.UTC), ArtifactID: &artifact.ID, SourceVersion: "v1", Checksum: "sha256:abc",
	})
	if err != nil {
		t.Fatal(err)
	}
	evidence := evidenceResult.Value

	objectiveResult, err := service.CreateObjective(ctx, cc, application.CreateObjectiveCommand{Scope: scope, Title: "Meet SLO", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	objective := objectiveResult.Value
	link1, err := service.CreateEvidenceLink(ctx, cc, application.CreateEvidenceLinkCommand{
		Scope: scope, EvidenceID: evidence.ID, TargetRef: objective.Ref(), Stance: domain.EvidenceStanceSupports, Rationale: "measured below threshold",
	})
	if err != nil {
		t.Fatal(err)
	}

	decisionResult, err := service.ProposeDecision(ctx, cc, application.ProposeDecisionCommand{
		Scope: scope, Title: "Storage", Proposal: "Use SQLite", Alternatives: []string{"SQLite", "PostgreSQL"}, ChosenAlternative: "SQLite", Rationale: "local-first",
	})
	if err != nil {
		t.Fatal(err)
	}
	decision := decisionResult.Value
	acceptedResult, err := service.AcceptDecision(ctx, cc, application.AcceptDecisionCommand{Scope: scope, DecisionID: decision.ID, ExpectedVersion: decision.Version})
	if err != nil {
		t.Fatal(err)
	}
	accepted := acceptedResult.Value

	link2, err := service.CreateEvidenceLink(ctx, cc, application.CreateEvidenceLinkCommand{
		Scope: scope, EvidenceID: evidence.ID, TargetRef: accepted.Ref(), Stance: domain.EvidenceStanceContradicts, Rationale: "remote deployment needs different storage",
	})
	if err != nil {
		t.Fatal(err)
	}
	if link1.Value.Stance == link2.Value.Stance {
		t.Fatal("same evidence should support different link-local stances")
	}

	title := "mutated"
	if _, err := service.UpdateDecision(ctx, cc, application.UpdateDecisionCommand{Scope: scope, DecisionID: accepted.ID, ExpectedVersion: accepted.Version, Title: &title}); err == nil {
		t.Fatal("accepted decision must be immutable")
	}

	superseded, err := service.SupersedeDecision(ctx, cc, application.SupersedeDecisionCommand{
		Scope: scope, DecisionID: accepted.ID, ExpectedVersion: accepted.Version,
		Title: "Storage v2", Proposal: "Use PostgreSQL remotely", Alternatives: []string{"SQLite", "PostgreSQL"}, ChosenAlternative: "PostgreSQL", Rationale: "multi-user deployment",
	})
	if err != nil {
		t.Fatal(err)
	}
	if superseded.Value.Predecessor.Lifecycle != domain.DecisionLifecycleSuperseded || superseded.Value.Successor.Lifecycle != domain.DecisionLifecycleAccepted {
		t.Fatalf("supersession=%#v", superseded.Value)
	}
	if superseded.Value.Successor.SupersedesDecisionID == nil || *superseded.Value.Successor.SupersedesDecisionID != accepted.ID {
		t.Fatal("successor does not preserve predecessor reference")
	}
	if _, err := service.SupersedeDecision(ctx, cc, application.SupersedeDecisionCommand{
		Scope: scope, DecisionID: accepted.ID, ExpectedVersion: accepted.Version,
		Title: "conflict", Proposal: "conflict", Rationale: "conflict",
	}); err == nil {
		t.Fatal("stale/concurrent supersession must fail")
	}

	events := store.SnapshotDomainEvents(scope)
	foundAccepted, foundSuperseded := false, false
	for _, event := range events {
		if event.CommandID == superseded.CommandID && event.EventType == "decision.accepted" {
			foundAccepted = true
		}
		if event.CommandID == superseded.CommandID && event.EventType == "decision.superseded" {
			foundSuperseded = true
		}
	}
	if !foundAccepted || !foundSuperseded {
		t.Fatal("supersession must emit both decision events atomically")
	}
}

func TestWave09RejectsCrossOutcomeEvidenceLink(t *testing.T) {
	ctx := context.Background()
	service, _ := newWave09Service(t)
	first := setupActiveOutcome(t, service)
	secondResult, err := service.CreateOutcome(ctx, commandContext(), application.CreateOutcomeCommand{NamespaceID: first.NamespaceID, Title: "Other", DesiredState: "Other state", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	second := secondResult.Value
	evidenceResult, err := service.RegisterEvidence(ctx, commandContext(), application.RegisterEvidenceCommand{
		Scope: first.Scope(), EvidenceType: domain.EvidenceTypeSource, Description: "source",
		SourceRef: domain.SourceReference{Provider: "test", ID: "e1"}, CapturedAt: time.Date(2026, 10, 2, 18, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.CreateEvidenceLink(ctx, commandContext(), application.CreateEvidenceLinkCommand{
		Scope: first.Scope(), EvidenceID: evidenceResult.Value.ID, TargetRef: second.Ref(), Stance: domain.EvidenceStanceContext, Rationale: "cross scope",
	})
	if err == nil {
		t.Fatal("cross-outcome documentary link must fail")
	}
}


func TestWave09OutcomeStateProjectsDocumentaryContext(t *testing.T) {
	ctx := context.Background()
	service, _ := newWave09Service(t)
	outcome := setupActiveOutcome(t, service)
	scope := outcome.Scope()
	cc := commandContext()

	artifactResult, err := service.RegisterArtifact(ctx, cc, application.RegisterArtifactCommand{
		Scope: scope, ArtifactType: "report", Name: "state report", URI: "file:///state-report.json",
	})
	if err != nil {
		t.Fatal(err)
	}
	artifact := artifactResult.Value

	evidenceResult, err := service.RegisterEvidence(ctx, cc, application.RegisterEvidenceCommand{
		Scope: scope,
		EvidenceType: domain.EvidenceTypeSource,
		Description: "state evidence",
		SourceRef: domain.SourceReference{Provider: "test", URI: "file:///state-report.json"},
		CapturedAt: time.Date(2026, 10, 2, 18, 0, 0, 0, time.UTC),
		ArtifactID: &artifact.ID,
	})
	if err != nil {
		t.Fatal(err)
	}

	decisionResult, err := service.ProposeDecision(ctx, cc, application.ProposeDecisionCommand{
		Scope: scope, Title: "State decision", Proposal: "Expose documentary state", Rationale: "completion gate",
	})
	if err != nil {
		t.Fatal(err)
	}
	decision := decisionResult.Value

	linkResult, err := service.CreateEvidenceLink(ctx, cc, application.CreateEvidenceLinkCommand{
		Scope: scope,
		EvidenceID: evidenceResult.Value.ID,
		TargetRef: decision.Ref(),
		Stance: domain.EvidenceStanceSupports,
		Rationale: "evidence supports the proposed state projection",
	})
	if err != nil {
		t.Fatal(err)
	}

	state, err := service.GetOutcomeState(ctx, scope)
	if err != nil {
		t.Fatal(err)
	}
	if len(state.Artifacts) != 1 || state.Artifacts[0].ID != artifact.ID {
		t.Fatalf("artifacts projection = %#v", state.Artifacts)
	}
	if len(state.Evidence) != 1 || state.Evidence[0].ID != evidenceResult.Value.ID {
		t.Fatalf("evidence projection = %#v", state.Evidence)
	}
	if len(state.EvidenceLinks) != 1 || state.EvidenceLinks[0].ID != linkResult.Value.ID {
		t.Fatalf("evidence links projection = %#v", state.EvidenceLinks)
	}
	if len(state.Decisions) != 1 || state.Decisions[0].ID != decision.ID {
		t.Fatalf("decisions projection = %#v", state.Decisions)
	}
	if state.OutcomeRevision != linkResult.OutcomeRevision {
		t.Fatalf("outcome revision = %d, want %d", state.OutcomeRevision, linkResult.OutcomeRevision)
	}
}
