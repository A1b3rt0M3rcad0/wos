package memory_test

import (
	"context"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

func TestRollbackDiscardsChanges(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	ns := domain.MustParseID("0199e100-0000-7000-8000-000000000001")
	outID := domain.MustParseID("0199e100-0000-7000-8000-000000000010")
	outcome, err := domain.NewOutcome(outID, ns, "Rollback", "", "No state after rollback", domain.PriorityNormal, time.Now().UTC())
	if err != nil {
		t.Fatal(err)
	}

	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Outcomes().Insert(ctx, outcome); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Coordination().AdvanceOutcome(ctx, outcome.Scope()); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	readTx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer readTx.Rollback()
	if _, err := readTx.Outcomes().Get(ctx, ns, outID); err == nil {
		t.Fatal("rolled back outcome unexpectedly persisted")
	}
	coord, err := readTx.Coordination().LockOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if coord.Revision != 0 {
		t.Fatalf("revision = %d, want 0 after rollback", coord.Revision)
	}
}

func TestCommitPersistsAndVersionConflictIsDetected(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	ns := domain.MustParseID("0199e100-0000-7000-8000-000000000001")
	outID := domain.MustParseID("0199e100-0000-7000-8000-000000000010")
	now := time.Now().UTC()
	outcome, err := domain.NewOutcome(outID, ns, "Persist", "", "State", domain.PriorityNormal, now)
	if err != nil {
		t.Fatal(err)
	}

	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Outcomes().Insert(ctx, outcome); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Coordination().AdvanceOutcome(ctx, outcome.Scope()); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	tx2, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	current, err := tx2.Outcomes().Get(ctx, ns, outID)
	if err != nil {
		t.Fatal(err)
	}
	criterionID := domain.MustParseID("0199e100-0000-7000-8000-000000000040")
	criterion, err := domain.NewSuccessCriterion(criterionID, current.Ref(), "Verified", "", true, domain.VerificationModeAttestation)
	if err != nil {
		t.Fatal(err)
	}
	expected := current.Version
	if err := current.AddCriterion(criterion, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if err := tx2.Outcomes().Save(ctx, current, expected-1); err == nil {
		t.Fatal("Save() unexpectedly accepted stale expected_version")
	}
	if err := tx2.Rollback(); err != nil {
		t.Fatal(err)
	}

	tx3, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx3.Rollback()
	coord, err := tx3.Coordination().LockOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if coord.Revision != 1 {
		t.Fatalf("revision = %d, want 1", coord.Revision)
	}
}

func TestDocumentaryRepositoriesEnforceHistoricalImmutability(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	namespaceID := domain.MustParseID("0199ee00-0000-7000-8000-000000000001")
	outcomeID := domain.MustParseID("0199ee00-0000-7000-8000-000000000010")
	scope := domain.Scope{NamespaceID: namespaceID, OutcomeID: outcomeID}
	now := time.Date(2026, 10, 2, 22, 0, 0, 0, time.UTC)
	actor := domain.ActorRef{Kind: domain.ActorKindService, Provider: "memory-test", ID: "producer"}

	outcome, err := domain.NewOutcome(outcomeID, namespaceID, "Documentary parity", "", "history is immutable", domain.PriorityNormal, now)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := domain.NewArtifact(
		domain.MustParseID("0199ee00-0000-7000-8000-000000000020"),
		scope,
		"document",
		"report",
		"file:///report.pdf",
		"application/pdf",
		"sha256:abc",
		"v1",
		actor,
		nil,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	decision, err := domain.NewDecision(
		domain.MustParseID("0199ee00-0000-7000-8000-000000000030"),
		scope,
		"Storage",
		"Use SQLite",
		[]string{"SQLite", "PostgreSQL"},
		"SQLite",
		"local-first",
		actor,
		now,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := decision.Accept(actor, now.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	documentary, ok := tx.(ports.DocumentaryUnitOfWork)
	if !ok {
		t.Fatal("memory transaction does not expose documentary repositories")
	}
	if err := tx.Outcomes().Insert(ctx, outcome); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Coordination().AdvanceOutcome(ctx, scope); err != nil {
		t.Fatal(err)
	}
	if err := documentary.Artifacts().Insert(ctx, artifact); err != nil {
		t.Fatal(err)
	}
	if err := documentary.Decisions().Insert(ctx, decision); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	mutation, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer mutation.Rollback()
	documentary, ok = mutation.(ports.DocumentaryUnitOfWork)
	if !ok {
		t.Fatal("memory transaction does not expose documentary repositories")
	}

	storedArtifact, err := documentary.Artifacts().Get(ctx, scope, artifact.ID)
	if err != nil {
		t.Fatal(err)
	}
	storedArtifact.Name = "rewritten report"
	storedArtifact.Version++
	storedArtifact.UpdatedAt = now.Add(2 * time.Minute)
	if err := documentary.Artifacts().Save(ctx, storedArtifact, artifact.Version); err == nil {
		t.Fatal("memory artifact repository accepted documentary content rewrite")
	}

	storedDecision, err := documentary.Decisions().Get(ctx, scope, decision.ID)
	if err != nil {
		t.Fatal(err)
	}
	storedDecision.Title = "rewritten accepted decision"
	storedDecision.Version++
	storedDecision.UpdatedAt = now.Add(2 * time.Minute)
	if err := documentary.Decisions().Save(ctx, storedDecision, decision.Version); err == nil {
		t.Fatal("memory decision repository accepted terminal content rewrite")
	}
}

func TestAssessmentProvenanceIsDeepCopied(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	now := time.Date(2026, 10, 2, 23, 0, 0, 0, time.UTC)
	namespaceID := domain.MustParseID("0199ef30-0000-7000-8000-000000000001")
	outcomeID := domain.MustParseID("0199ef30-0000-7000-8000-000000000010")
	scope := domain.Scope{NamespaceID: namespaceID, OutcomeID: outcomeID}

	outcome, err := domain.NewOutcome(outcomeID, namespaceID, "Clone assessment", "", "preserve provenance", domain.PriorityNormal, now)
	if err != nil {
		t.Fatal(err)
	}
	criterion, err := domain.NewSuccessCriterion(
		domain.MustParseID("0199ef30-0000-7000-8000-000000000020"),
		outcome.Ref(),
		"External result",
		"",
		true,
		domain.VerificationModeExternalEvaluation,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := outcome.AddCriterion(criterion, now); err != nil {
		t.Fatal(err)
	}
	assessment := domain.CriterionAssessment{
		ID:                domain.MustParseID("0199ef30-0000-7000-8000-000000000030"),
		CriterionID:       criterion.ID,
		CriterionRevision: criterion.Revision,
		Result:            domain.AssessmentResultMet,
		Rationale:         "external validation",
		EvidenceIDs:       []domain.ID{domain.MustParseID("0199ef30-0000-7000-8000-000000000040")},
		EvaluatorRef:      &domain.EvaluatorRef{Provider: "validator", ID: "engine", Version: "1"},
		PrincipalID:       "tester",
		Actor:             domain.ActorRef{Kind: domain.ActorKindService, Provider: "test", ID: "tester"},
		AssessedAt:        now,
	}
	if err := outcome.RecordCriterionAssessment(assessment, false, now); err != nil {
		t.Fatal(err)
	}

	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Outcomes().Insert(ctx, outcome); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Coordination().AdvanceOutcome(ctx, scope); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	readTx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	stored, err := readTx.Outcomes().Get(ctx, namespaceID, outcomeID)
	if err != nil {
		t.Fatal(err)
	}
	current := stored.Criteria.CurrentAssessments[criterion.ID]
	current.EvidenceIDs[0] = domain.MustParseID("0199ef30-0000-7000-8000-000000000099")
	current.EvaluatorRef.Version = "mutated"
	_ = readTx.Rollback()

	verifyTx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer verifyTx.Rollback()
	again, err := verifyTx.Outcomes().Get(ctx, namespaceID, outcomeID)
	if err != nil {
		t.Fatal(err)
	}
	persisted := again.Criteria.CurrentAssessments[criterion.ID]
	if persisted.EvidenceIDs[0] == current.EvidenceIDs[0] || persisted.EvaluatorRef.Version == "mutated" {
		t.Fatalf("assessment provenance leaked through clone: %#v", persisted)
	}
}
