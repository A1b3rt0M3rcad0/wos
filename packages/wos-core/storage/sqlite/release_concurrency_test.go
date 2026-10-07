package sqlite

import (
	"context"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"path/filepath"
	"testing"
	"time"
)

// Separate Stores force separate connections/pools; a single Store would hide
// storage contention behind SQLite's one-connection pool.
func releaseServices(t *testing.T) (*application.Service, *application.Service, domain.Outcome) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "release.db")
	a, err := Open(path, Options{MigrateOnOpen: true, BusyTimeout: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	b, err := Open(path, Options{MigrateOnOpen: true, BusyTimeout: 5 * time.Second})
	if err != nil {
		a.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { b.Close(); a.Close() })
	clock := sqliteFixedClock{now: time.Now().UTC()}
	first, err := application.NewService(a, clock, &sqliteSequenceIDs{prefix: "01a11740", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	second, err := application.NewService(b, clock, &sqliteSequenceIDs{prefix: "01a11741", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	result, err := first.CreateOutcome(context.Background(), releaseCC(1, ""), application.CreateOutcomeCommand{NamespaceID: testID("01a11742-0000-7000-8000-000000000001"), Title: "Release concurrency", DesiredState: "separate connections preserve invariants", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	return first, second, result.Value
}
func releaseCC(n int, key string) domain.CommandContext {
	return sqliteCommandContext(fmt.Sprintf("01a11743-0000-7000-8000-%012x", n), key)
}
func releaseRace(a, b func() error) []error {
	start := make(chan struct{})
	results := make(chan error, 2)
	for _, f := range []func() error{a, b} {
		go func(f func() error) { <-start; results <- f() }(f)
	}
	close(start)
	return []error{<-results, <-results}
}
func expectedContention(t *testing.T, errs []error) int {
	t.Helper()
	success := 0
	for _, err := range errs {
		if err == nil {
			success++
			continue
		}
		code, _ := domain.ErrorCodeOf(err)
		if code != domain.ErrorCodeVersionConflict && code != domain.ErrorCodeTransactionConflict && code != domain.ErrorCodeDependencyCycle {
			t.Fatalf("unexpected contention failure: %v (%s)", err, code)
		}
	}
	if success == 0 {
		t.Fatalf("neither transaction committed: %v", errs)
	}
	return success
}
func TestReleaseConcurrentActiveSlotsOnSeparateConnections(t *testing.T) {
	ctx := context.Background()
	a, b, outcome := releaseServices(t)
	scope := outcome.Scope()
	planScope := domain.RoadmapPlanScope{Kind: domain.RoadmapScopeOutcome, ID: outcome.ID}
	var plans []domain.Roadmap
	for i := 0; i < 2; i++ {
		p, err := a.CreateRoadmap(ctx, releaseCC(10+i*3, ""), application.CreateRoadmapCommand{Scope: scope, PlanScope: planScope, Title: fmt.Sprintf("Plan %d", i)})
		if err != nil {
			t.Fatal(err)
		}
		draft, err := a.OpenRoadmapDraft(ctx, releaseCC(11+i*3, ""), application.OpenRoadmapDraftCommand{Scope: scope, RoadmapID: p.Value.ID, ExpectedVersion: p.Value.Version})
		if err != nil {
			t.Fatal(err)
		}
		published, err := a.PublishRoadmapDraft(ctx, releaseCC(12+i*3, ""), application.PublishRoadmapDraftCommand{Scope: scope, RoadmapID: p.Value.ID, ExpectedVersion: draft.Value.Version, ExpectedDraftVersion: draft.Value.Draft.DraftVersion})
		if err != nil {
			t.Fatal(err)
		}
		plans = append(plans, published.Value)
	}
	run := func(s *application.Service, i int) func() error {
		return func() error {
			_, err := s.ActivateRoadmapRevision(ctx, releaseCC(20+i, ""), application.ActivateRoadmapRevisionCommand{Scope: scope, RoadmapID: plans[i].ID, ExpectedVersion: plans[i].Version, RevisionNumber: 1})
			return err
		}
	}
	count := expectedContention(t, releaseRace(run(a, 0), run(b, 1)))
	slot, _, err := a.GetActiveRoadmapSlot(ctx, scope, planScope)
	if err != nil || slot == nil {
		t.Fatalf("active slot missing: %v", err)
	}
	history, _, err := a.ListRoadmapActivationHistory(ctx, scope, planScope)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 2*count-1 {
		t.Fatalf("history lost serialized activation: %d committed, %d facts", count, len(history))
	}
	if slot.RoadmapID != plans[0].ID && slot.RoadmapID != plans[1].ID {
		t.Fatal("unexpected active plan")
	}
}
func TestReleaseConcurrentIdempotencyOnSeparateConnections(t *testing.T) {
	ctx := context.Background()
	a, b, outcome := releaseServices(t)
	cmd := application.CreateWorkItemCommand{Scope: outcome.Scope(), Title: "One intent", Priority: domain.PriorityNormal}
	key := "release-concurrent-create-0001"
	run := func(s *application.Service, n int) func() error {
		return func() error { _, err := s.CreateWorkItem(ctx, releaseCC(n, key), cmd); return err }
	}
	expectedContention(t, releaseRace(run(a, 30), run(b, 31)))
	// An aborted transaction is explicitly retried with the exact same intent/key.
	replay, err := b.CreateWorkItem(ctx, releaseCC(32, key), cmd)
	if err != nil || !replay.IdempotentReplay {
		t.Fatalf("retry did not reconcile: %v", err)
	}
	snapshot, err := a.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.WorkItems) != 1 || snapshot.WorkItems[0].ID != replay.Value.ID {
		t.Fatal("duplicate intent created work")
	}
	timeline, err := a.GetTimeline(ctx, outcome.Scope(), application.TimelineQuery{EventType: "work_item.created", Limit: 100})
	if err != nil {
		t.Fatal(err)
	}
	if len(timeline.Items) != 1 {
		t.Fatalf("idempotent race emitted %d creation events", len(timeline.Items))
	}
}
func TestReleaseConcurrentAssessmentsOnSeparateConnections(t *testing.T) {
	ctx := context.Background()
	a, b, outcome := releaseServices(t)
	added, err := a.AddCriterion(ctx, releaseCC(40, ""), application.AddCriterionCommand{Owner: outcome.Ref(), ExpectedVersion: outcome.Version, Title: "Reviewed", Required: true, VerificationMode: domain.VerificationModeAttestation})
	if err != nil {
		t.Fatal(err)
	}
	live, err := a.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	run := func(s *application.Service, n int, result domain.AssessmentResult) func() error {
		return func() error {
			_, err := s.RecordCriterionAssessment(ctx, releaseCC(n, ""), application.RecordCriterionAssessmentCommand{Owner: outcome.Ref(), ExpectedVersion: live.Value.Version, CriterionID: added.Value.ID, CriterionRevision: 1, Result: result, Rationale: "explicit review"})
			return err
		}
	}
	count := expectedContention(t, releaseRace(run(a, 41, domain.AssessmentResultMet), run(b, 42, domain.AssessmentResultNotMet)))
	if count != 1 {
		t.Fatal("two stale assessments replaced current binding")
	}
	h, err := a.GetCriterionHistory(ctx, outcome.Ref(), added.Value.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(h.Value.Assessments) != 1 || h.Value.CurrentAssessment == nil {
		t.Fatal("assessment history/current binding differs")
	}
}

func TestReleaseConcurrentClaimsOnSeparateConnections(t *testing.T) {
	ctx := context.Background()
	a, b, outcome := releaseServices(t)
	_, err := a.AddCriterion(ctx, releaseCC(50, ""), application.AddCriterionCommand{Owner: outcome.Ref(), ExpectedVersion: 1, Title: "Required", Required: true, VerificationMode: domain.VerificationModeAttestation})
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.ActivateOutcome(ctx, releaseCC(51, ""), application.ActivateOutcomeCommand{Scope: outcome.Scope(), ExpectedVersion: 2})
	if err != nil {
		t.Fatal(err)
	}
	work, err := a.CreateWorkItem(ctx, releaseCC(52, ""), application.CreateWorkItemCommand{Scope: outcome.Scope(), Title: "One claimant", Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleTodo})
	if err != nil {
		t.Fatal(err)
	}
	run := func(s *application.Service, n int) func() error {
		return func() error {
			_, err := s.ClaimWorkItem(ctx, releaseCC(n, ""), application.ClaimWorkItemCommand{Scope: outcome.Scope(), WorkItemID: work.Value.ID, ExpectedVersion: 1, TTL: time.Minute})
			return err
		}
	}
	errs := releaseRace(run(a, 53), run(b, 54))
	success := 0
	for _, err := range errs {
		if err == nil {
			success++
			continue
		}
		code, _ := domain.ErrorCodeOf(err)
		if code != domain.ErrorCodeVersionConflict && code != domain.ErrorCodeTransactionConflict && code != domain.ErrorCodePreconditionFailed {
			t.Fatalf("unexpected claim rejection: %v", err)
		}
	}
	if success != 1 {
		t.Fatalf("%d claimants committed", success)
	}
	live, err := a.GetWorkItem(ctx, outcome.Scope(), work.Value.ID)
	if err != nil {
		t.Fatal(err)
	}
	if live.Value.CurrentLease == nil || live.Value.CurrentLease.FencingToken != 1 || live.Value.Version != 2 {
		t.Fatal("race corrupted lease/version")
	}
}

func TestReleaseSnapshotIsCoherentDuringSeparateConnectionWrites(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	a, b, outcome := releaseServices(t)
	done := make(chan error, 1)
	firstCommitted := make(chan struct{})
	resume := make(chan struct{})
	go func() {
		for i := 0; i < 30; i++ {
			var err error
			cmd := application.CreateWorkItemCommand{Scope: outcome.Scope(), Title: fmt.Sprintf("Concurrent work %d", i), Priority: domain.PriorityNormal}
			for attempt := 0; attempt < 10; attempt++ {
				_, err = a.CreateWorkItem(ctx, releaseCC(100+i, fmt.Sprintf("release-snapshot-%04d", i)), cmd)
				if err == nil {
					break
				}
				code, _ := domain.ErrorCodeOf(err)
				if code != domain.ErrorCodeTransactionConflict {
					break
				}
			}
			if err != nil {
				if i == 0 {
					close(firstCommitted)
				}
				done <- err
				return
			}
			if i == 0 {
				close(firstCommitted)
				select {
				case <-resume:
				case <-ctx.Done():
					done <- ctx.Err()
					return
				}
			}
		}
		done <- nil
	}()
	<-firstCommitted
	first, err := b.GetOutcomeState(ctx, outcome.Scope())
	close(resume)
	if err != nil || first.OutcomeRevision != 2 || len(first.WorkItems) != 1 {
		t.Fatalf("initial snapshot: revision %d work %d error %v", first.OutcomeRevision, len(first.WorkItems), err)
	}
	samples := 1
	for {
		snapshot, err := b.GetOutcomeState(ctx, outcome.Scope())
		if err != nil {
			code, _ := domain.ErrorCodeOf(err)
			if code != domain.ErrorCodeTransactionConflict {
				t.Fatal(err)
			}
		} else {
			samples++
			if int(snapshot.OutcomeRevision) != 1+len(snapshot.WorkItems) {
				t.Fatalf("mixed snapshot: revision %d, work %d", snapshot.OutcomeRevision, len(snapshot.WorkItems))
			}
		}
		select {
		case err := <-done:
			if err != nil {
				t.Fatal(err)
			}
			last, err := b.GetOutcomeState(ctx, outcome.Scope())
			if err != nil || len(last.WorkItems) != 30 || last.OutcomeRevision != 31 || samples == 0 {
				t.Fatalf("incomplete history: revision %d work %d error %v samples %d", last.OutcomeRevision, len(last.WorkItems), err, samples)
			}
			return
		default:
		}
	}
}
