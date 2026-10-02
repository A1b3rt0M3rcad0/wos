package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestSQLiteWave10ConclusionObligationsSurviveRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "wos.db")
	store, err := Open(path, Options{BusyTimeout: time.Second, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	clock := sqliteFixedClock{now: time.Date(2026, 10, 3, 0, 0, 0, 0, time.UTC)}
	service, err := application.NewService(store, clock, &sqliteSequenceIDs{prefix: "0199ef50", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	cc := sqliteCommandContext("0199ef51-0000-7000-8000-000000000101", "")
	namespaceID := domain.MustParseID("0199ef51-0000-7000-8000-000000000001")
	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Conclusion restart",
		DesiredState: "preserve proof snapshot",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value

	added, err := service.AddCriterion(ctx, sqliteCommandContext(
		"0199ef51-0000-7000-8000-000000000102", "",
	), application.AddCriterionCommand{
		Owner:            outcome.Ref(),
		ExpectedVersion:  outcome.Version,
		Title:            "Verified",
		Required:         true,
		VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	criterion := added.Value
	current, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	assessed, err := service.AttestCriterion(ctx, sqliteCommandContext(
		"0199ef51-0000-7000-8000-000000000103", "",
	), application.AttestCriterionCommand{
		Owner:             outcome.Ref(),
		CriterionID:       criterion.ID,
		CriterionRevision: criterion.Revision,
		ExpectedVersion:   current.Value.Version,
		Result:            domain.AssessmentResultMet,
		Rationale:         "verified",
	})
	if err != nil {
		t.Fatal(err)
	}
	current, err = service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	activated, err := service.ActivateOutcome(ctx, sqliteCommandContext(
		"0199ef51-0000-7000-8000-000000000104", "",
	), application.ActivateOutcomeCommand{
		Scope: outcome.Scope(), ExpectedVersion: current.Value.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	achieved, err := service.AchieveOutcome(ctx, sqliteCommandContext(
		"0199ef51-0000-7000-8000-000000000105", "",
	), application.AchieveOutcomeCommand{
		Scope: outcome.Scope(), ExpectedVersion: activated.Value.Version, Reason: "all proof satisfied",
	})
	if err != nil {
		t.Fatal(err)
	}
	expectedConclusion := achieved.Value.CurrentConclusion
	if expectedConclusion == nil {
		t.Fatal("conclusion missing before restart")
	}
	if len(expectedConclusion.Assessments) != 1 || expectedConclusion.Assessments[0].AssessmentID != assessed.Value.ID {
		t.Fatalf("conclusion assessment snapshot = %#v", expectedConclusion.Assessments)
	}

	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(path, Options{BusyTimeout: time.Second, MigrateOnOpen: true})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	restarted, err := application.NewService(reopened, clock, &sqliteSequenceIDs{prefix: "0199ef52", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	persisted, err := restarted.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	conclusion := persisted.Value.CurrentConclusion
	if conclusion == nil {
		t.Fatal("conclusion missing after restart")
	}
	if conclusion.OwnerRef == nil || *conclusion.OwnerRef != outcome.Ref() {
		t.Fatalf("owner ref = %#v", conclusion.OwnerRef)
	}
	if conclusion.OwnerVersion == nil || *conclusion.OwnerVersion != achieved.Value.Version {
		t.Fatalf("owner version = %#v, want %d", conclusion.OwnerVersion, achieved.Value.Version)
	}
	if conclusion.LifecycleResult != string(domain.OutcomeLifecycleAchieved) {
		t.Fatalf("lifecycle result = %q", conclusion.LifecycleResult)
	}
	if len(conclusion.Obligations.RequiredCriteria) != 1 ||
		conclusion.Obligations.RequiredCriteria[0].CriterionID != criterion.ID {
		t.Fatalf("obligation snapshot = %#v", conclusion.Obligations)
	}
}
