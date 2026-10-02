package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestSQLiteWave10AssessmentEvidenceSurvivesRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "wos.db")
	store, err := Open(path, Options{BusyTimeout: time.Second, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	clock := sqliteFixedClock{now: time.Date(2026, 10, 2, 21, 0, 0, 0, time.UTC)}
	service, err := application.NewService(store, clock, &sqliteSequenceIDs{prefix: "0199ef20", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	namespaceID := domain.MustParseID("0199ef21-0000-7000-8000-000000000001")
	created, err := service.CreateOutcome(ctx, sqliteCommandContext(
		"0199ef21-0000-7000-8000-000000000101", "",
	), application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Assessment persistence",
		DesiredState: "assessment Evidence survives restart",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value

	added, err := service.AddCriterion(ctx, sqliteCommandContext(
		"0199ef21-0000-7000-8000-000000000102", "",
	), application.AddCriterionCommand{
		Owner:            outcome.Ref(),
		ExpectedVersion:  outcome.Version,
		Title:            "Evidence reviewed",
		Required:         true,
		VerificationMode: domain.VerificationModeEvidenceReview,
	})
	if err != nil {
		t.Fatal(err)
	}
	criterion := added.Value

	evidenceResult, err := service.RegisterEvidence(ctx, sqliteCommandContext(
		"0199ef21-0000-7000-8000-000000000103", "",
	), application.RegisterEvidenceCommand{
		Scope:        outcome.Scope(),
		EvidenceType: domain.EvidenceTypeSource,
		Description:  "restart source",
		SourceRef:    domain.SourceReference{Provider: "sqlite-test", ID: "source"},
		CapturedAt:   clock.now,
	})
	if err != nil {
		t.Fatal(err)
	}
	current, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	recorded, err := service.RecordCriterionAssessment(ctx, sqliteCommandContext(
		"0199ef21-0000-7000-8000-000000000104", "",
	), application.RecordCriterionAssessmentCommand{
		Owner:             outcome.Ref(),
		CriterionID:       criterion.ID,
		CriterionRevision: criterion.Revision,
		ExpectedVersion:   current.Value.Version,
		Result:            domain.AssessmentResultMet,
		Rationale:         "reviewed persisted Evidence",
		EvidenceIDs:       []domain.ID{evidenceResult.Value.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	assessmentID := recorded.Value.ID

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path, Options{BusyTimeout: time.Second, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	restarted, err := application.NewService(reopened, clock, &sqliteSequenceIDs{prefix: "0199ef22", next: 1})
	if err != nil {
		t.Fatal(err)
	}

	persisted, err := restarted.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	currentAssessment, ok := persisted.Value.Criteria.CurrentAssessments[criterion.ID]
	if !ok {
		t.Fatal("current assessment missing after restart")
	}
	if currentAssessment.ID != assessmentID {
		t.Fatalf("assessment id = %s, want %s", currentAssessment.ID, assessmentID)
	}
	if len(currentAssessment.EvidenceIDs) != 1 || currentAssessment.EvidenceIDs[0] != evidenceResult.Value.ID {
		t.Fatalf("persisted Evidence IDs = %#v", currentAssessment.EvidenceIDs)
	}
}
