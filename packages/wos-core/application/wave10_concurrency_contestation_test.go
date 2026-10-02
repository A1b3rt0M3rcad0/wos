package application_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

type concurrentWave10IDs struct {
	mu   sync.Mutex
	next uint64
}

func (g *concurrentWave10IDs) NewID() (domain.ID, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	value := domain.MustParseID(fmt.Sprintf("0199efb0-0000-7000-8000-%012x", g.next))
	g.next++
	return value, nil
}

func TestWave10ConcurrentAssessmentsHaveSingleCurrentWinner(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	service, err := application.NewService(
		store,
		fixedClock{now: time.Date(2026, 10, 2, 20, 15, 0, 0, time.UTC)},
		&concurrentWave10IDs{next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	cc := commandContext()
	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  domain.MustParseID("0199efb1-0000-7000-8000-000000000001"),
		Title:        "Concurrent assessments",
		DesiredState: "one current assessment wins",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value
	added, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner:            outcome.Ref(),
		ExpectedVersion:  outcome.Version,
		Title:            "Concurrent criterion",
		Required:         true,
		VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	current, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	expectedVersion := current.Value.Version

	type result struct {
		value application.MutationResult[domain.CriterionAssessment]
		err   error
	}
	start := make(chan struct{})
	results := make(chan result, 2)
	for i, assessmentResult := range []domain.AssessmentResult{
		domain.AssessmentResultMet,
		domain.AssessmentResultNotMet,
	} {
		i := i
		assessmentResult := assessmentResult
		go func() {
			<-start
			localCC := commandContext()
			localCC.CommandID = domain.MustParseID(fmt.Sprintf(
				"0199efb2-0000-7000-8000-%012x",
				i+1,
			))
			value, err := service.RecordCriterionAssessment(
				ctx,
				localCC,
				application.RecordCriterionAssessmentCommand{
					Owner:             outcome.Ref(),
					CriterionID:       added.Value.ID,
					CriterionRevision: added.Value.Revision,
					ExpectedVersion:   expectedVersion,
					Result:            assessmentResult,
					Rationale:         "concurrent assessment",
				},
			)
			results <- result{value: value, err: err}
		}()
	}
	close(start)

	var successes int
	var conflicts int
	for range 2 {
		result := <-results
		if result.err == nil {
			successes++
			continue
		}
		code, ok := domain.ErrorCodeOf(result.err)
		if ok && code == domain.ErrorCodeVersionConflict {
			conflicts++
			continue
		}
		t.Fatalf("unexpected concurrent assessment error: %v", result.err)
	}
	if successes != 1 || conflicts != 1 {
		t.Fatalf("successes=%d conflicts=%d, want 1/1", successes, conflicts)
	}

	persisted, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	currentAssessment, ok := persisted.Value.Criteria.CurrentAssessments[added.Value.ID]
	if !ok {
		t.Fatal("current assessment missing")
	}
	if len(persisted.Value.Criteria.Assessments) != 1 {
		t.Fatalf("assessment history length = %d, want 1", len(persisted.Value.Criteria.Assessments))
	}
	if currentAssessment.ID != persisted.Value.Criteria.Assessments[0].ID {
		t.Fatal("current assessment does not reference the only committed immutable assessment")
	}
}

func TestWave10RetractedEvidenceContestsConclusionWithoutReopening(t *testing.T) {
	ctx := context.Background()
	service, _ := newWave09Service(t)
	cc := commandContext()
	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  domain.MustParseID("0199efb3-0000-7000-8000-000000000001"),
		Title:        "Evidence contestation",
		DesiredState: "later retraction remains visible",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value
	criterionResult, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner:            outcome.Ref(),
		ExpectedVersion:  outcome.Version,
		Title:            "Evidence-backed criterion",
		Required:         true,
		VerificationMode: domain.VerificationModeEvidenceReview,
	})
	if err != nil {
		t.Fatal(err)
	}
	criterion := criterionResult.Value

	evidenceResult, err := service.RegisterEvidence(ctx, cc, application.RegisterEvidenceCommand{
		Scope:        outcome.Scope(),
		EvidenceType: domain.EvidenceTypeSource,
		Description:  "source later invalidated",
		SourceRef:    domain.SourceReference{Provider: "test", ID: "source-wave10"},
		CapturedAt:   time.Date(2026, 10, 2, 20, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatal(err)
	}
	evidence := evidenceResult.Value

	current, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	activated, err := service.ActivateOutcome(ctx, cc, application.ActivateOutcomeCommand{
		Scope: outcome.Scope(), ExpectedVersion: current.Value.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	assessment, err := service.RecordCriterionAssessment(ctx, cc, application.RecordCriterionAssessmentCommand{
		Owner:             outcome.Ref(),
		CriterionID:       criterion.ID,
		CriterionRevision: criterion.Revision,
		ExpectedVersion:   activated.Value.Version,
		Result:            domain.AssessmentResultMet,
		Rationale:         "source supports criterion",
		EvidenceIDs:       []domain.ID{evidence.ID},
	})
	if err != nil {
		t.Fatal(err)
	}
	current, err = service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	achieved, err := service.AchieveOutcome(ctx, cc, application.AchieveOutcomeCommand{
		Scope: outcome.Scope(), ExpectedVersion: current.Value.Version, Reason: "evidence satisfied",
	})
	if err != nil {
		t.Fatal(err)
	}
	conclusionID := achieved.Value.CurrentConclusion.ID

	if _, err := service.RetractEvidence(ctx, cc, application.RetractEvidenceCommand{
		Scope:           outcome.Scope(),
		EvidenceID:      evidence.ID,
		ExpectedVersion: evidence.Version,
		Reason:          "source was invalidated",
	}); err != nil {
		t.Fatal(err)
	}

	state, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if state.Outcome.Lifecycle != domain.OutcomeLifecycleAchieved {
		t.Fatalf("lifecycle = %q, want achieved", state.Outcome.Lifecycle)
	}
	if state.Outcome.CurrentConclusion == nil ||
		state.Outcome.CurrentConclusion.ID != conclusionID {
		t.Fatal("evidence retraction replaced or removed the historical conclusion")
	}
	if !state.ConclusionContested {
		t.Fatal("retracted Evidence did not contest conclusion")
	}
	var found bool
	for _, cause := range state.ConclusionContestations {
		if cause.Kind == domain.ConclusionContestationEvidenceRetracted &&
			cause.AssessmentID == assessment.Value.ID &&
			cause.EvidenceID != nil &&
			*cause.EvidenceID == evidence.ID {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("evidence retraction cause missing: %#v", state.ConclusionContestations)
	}
}
