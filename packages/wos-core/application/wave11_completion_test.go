package application_test

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

type wave11ConcurrentIDs struct {
	mu   sync.Mutex
	next uint64
}

func (g *wave11ConcurrentIDs) NewID() (domain.ID, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	value := domain.MustParseID(fmt.Sprintf("0199f330-0000-7000-8000-%012x", g.next))
	g.next++
	return value, nil
}

func newWave11ConcurrentService(t *testing.T) (*application.Service, *memory.Store) {
	t.Helper()
	store := memory.New()
	service, err := application.NewService(
		store,
		fixedClock{now: time.Date(2026, 10, 2, 23, 30, 0, 0, time.UTC)},
		&wave11ConcurrentIDs{next: 1},
	)
	if err != nil {
		t.Fatal(err)
	}
	return service, store
}

func wave11PublishedEmptyRoadmap(
	t *testing.T,
	ctx context.Context,
	service *application.Service,
	outcome domain.Outcome,
	title string,
) domain.Roadmap {
	t.Helper()
	cc := commandContext()
	created, err := service.CreateRoadmap(ctx, cc, application.CreateRoadmapCommand{
		Scope: outcome.Scope(),
		PlanScope: domain.RoadmapPlanScope{
			Kind: domain.RoadmapScopeOutcome,
			ID:   outcome.ID,
		},
		Title: title,
	})
	if err != nil {
		t.Fatal(err)
	}
	opened, err := service.OpenRoadmapDraft(ctx, cc, application.OpenRoadmapDraftCommand{
		Scope:           outcome.Scope(),
		RoadmapID:       created.Value.ID,
		ExpectedVersion: created.Value.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	published, err := service.PublishRoadmapDraft(ctx, cc, application.PublishRoadmapDraftCommand{
		Scope:                outcome.Scope(),
		RoadmapID:            created.Value.ID,
		ExpectedVersion:      opened.Value.Version,
		ExpectedDraftVersion: opened.Value.Draft.DraftVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	return published.Value
}

func TestWave11ConcurrentPublicationHasSingleWinner(t *testing.T) {
	ctx := context.Background()
	service, _ := newWave11ConcurrentService(t)
	created, err := service.CreateOutcome(ctx, commandContext(), application.CreateOutcomeCommand{
		NamespaceID:  domain.MustParseID("0199f331-0000-7000-8000-000000000001"),
		Title:        "Concurrent publication",
		DesiredState: "one immutable revision is published",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value

	roadmapResult, err := service.CreateRoadmap(ctx, commandContext(), application.CreateRoadmapCommand{
		Scope: outcome.Scope(),
		PlanScope: domain.RoadmapPlanScope{
			Kind: domain.RoadmapScopeOutcome,
			ID:   outcome.ID,
		},
		Title: "Concurrent plan",
	})
	if err != nil {
		t.Fatal(err)
	}
	roadmap := roadmapResult.Value
	opened, err := service.OpenRoadmapDraft(ctx, commandContext(), application.OpenRoadmapDraftCommand{
		Scope: outcome.Scope(), RoadmapID: roadmap.ID, ExpectedVersion: roadmap.Version,
	})
	if err != nil {
		t.Fatal(err)
	}

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			<-start
			cc := commandContext()
			cc.CommandID = domain.MustParseID(fmt.Sprintf(
				"0199f331-0000-7000-8000-%012x",
				100+index,
			))
			_, err := service.PublishRoadmapDraft(ctx, cc, application.PublishRoadmapDraftCommand{
				Scope:                outcome.Scope(),
				RoadmapID:            roadmap.ID,
				ExpectedVersion:      opened.Value.Version,
				ExpectedDraftVersion: opened.Value.Draft.DraftVersion,
			})
			errs <- err
		}(i)
	}
	close(start)
	wg.Wait()
	close(errs)

	successes := 0
	failures := 0
	for err := range errs {
		if err == nil {
			successes++
		} else {
			failures++
		}
	}
	if successes != 1 || failures != 1 {
		t.Fatalf("publication results: successes=%d failures=%d", successes, failures)
	}
	read, err := service.GetRoadmap(ctx, outcome.Scope(), roadmap.ID)
	if err != nil {
		t.Fatal(err)
	}
	if read.Value.Draft != nil || len(read.Value.Revisions) != 1 {
		t.Fatalf("concurrent publication state = %#v", read.Value)
	}
}

func TestWave11ConcurrentActivationPreservesUniqueSlot(t *testing.T) {
	ctx := context.Background()
	service, _ := newWave11ConcurrentService(t)
	created, err := service.CreateOutcome(ctx, commandContext(), application.CreateOutcomeCommand{
		NamespaceID:  domain.MustParseID("0199f332-0000-7000-8000-000000000001"),
		Title:        "Concurrent activation",
		DesiredState: "one active revision per planning scope",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value
	first := wave11PublishedEmptyRoadmap(t, ctx, service, outcome, "First")
	second := wave11PublishedEmptyRoadmap(t, ctx, service, outcome, "Second")

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i, roadmap := range []domain.Roadmap{first, second} {
		wg.Add(1)
		go func(index int, value domain.Roadmap) {
			defer wg.Done()
			<-start
			cc := commandContext()
			cc.CommandID = domain.MustParseID(fmt.Sprintf(
				"0199f332-0000-7000-8000-%012x",
				100+index,
			))
			_, err := service.ActivateRoadmapRevision(ctx, cc, application.ActivateRoadmapRevisionCommand{
				Scope:           outcome.Scope(),
				RoadmapID:       value.ID,
				ExpectedVersion: value.Version,
				RevisionNumber:  1,
			})
			errs <- err
		}(i, roadmap)
	}
	close(start)
	wg.Wait()
	close(errs)

	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent activation failed: %v", err)
		}
	}
	planScope := domain.RoadmapPlanScope{Kind: domain.RoadmapScopeOutcome, ID: outcome.ID}
	slot, _, err := service.GetActiveRoadmapSlot(ctx, outcome.Scope(), planScope)
	if err != nil {
		t.Fatal(err)
	}
	if slot == nil {
		t.Fatal("concurrent activation left no active slot")
	}
	if slot.RoadmapID != first.ID && slot.RoadmapID != second.ID {
		t.Fatalf("active slot points to unexpected roadmap: %#v", slot)
	}
	history, _, err := service.ListRoadmapActivationHistory(ctx, outcome.Scope(), planScope)
	if err != nil {
		t.Fatal(err)
	}
	activated := 0
	superseded := 0
	for _, record := range history {
		switch record.Action {
		case domain.RoadmapActivationActivated:
			activated++
		case domain.RoadmapActivationSuperseded:
			superseded++
		}
	}
	if len(history) != 3 || activated != 2 || superseded != 1 {
		t.Fatalf("concurrent activation history = %#v", history)
	}
}

func TestWave11ReplanningAndLiveCancellationDoNotRewriteHistory(t *testing.T) {
	ctx := context.Background()
	service, _ := newWave09Service(t)
	cc := commandContext()
	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  domain.MustParseID("0199f333-0000-7000-8000-000000000001"),
		Title:        "Historical planning",
		DesiredState: "published revisions stay immutable",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value
	workResult, err := service.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{
		Scope: outcome.Scope(), Title: "Live work", Priority: domain.PriorityNormal,
		Lifecycle: domain.WorkItemLifecycleTodo,
	})
	if err != nil {
		t.Fatal(err)
	}
	work := workResult.Value
	roadmapResult, err := service.CreateRoadmap(ctx, cc, application.CreateRoadmapCommand{
		Scope: outcome.Scope(),
		PlanScope: domain.RoadmapPlanScope{
			Kind: domain.RoadmapScopeOutcome,
			ID:   outcome.ID,
		},
		Title: "Historical plan",
	})
	if err != nil {
		t.Fatal(err)
	}
	roadmap := roadmapResult.Value
	opened, err := service.OpenRoadmapDraft(ctx, cc, application.OpenRoadmapDraftCommand{
		Scope: outcome.Scope(), RoadmapID: roadmap.ID, ExpectedVersion: roadmap.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	edited, err := service.ReplaceRoadmapDraft(ctx, cc, application.ReplaceRoadmapDraftCommand{
		Scope:                outcome.Scope(),
		RoadmapID:            roadmap.ID,
		ExpectedVersion:      opened.Value.Version,
		ExpectedDraftVersion: opened.Value.Draft.DraftVersion,
		Nodes: []domain.RoadmapNode{
			{
				NodeKey: "work", NodeType: domain.RoadmapNodeReference,
				TargetRef: refPtr(work.Ref()), Title: "Live work in plan", Position: 0,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	published, err := service.PublishRoadmapDraft(ctx, cc, application.PublishRoadmapDraftCommand{
		Scope:                outcome.Scope(),
		RoadmapID:            roadmap.ID,
		ExpectedVersion:      edited.Value.Version,
		ExpectedDraftVersion: edited.Value.Draft.DraftVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	revisionOne := published.Value.Revisions[0]

	base := uint64(1)
	nextDraft, err := service.OpenRoadmapDraft(ctx, cc, application.OpenRoadmapDraftCommand{
		Scope:              outcome.Scope(),
		RoadmapID:          roadmap.ID,
		ExpectedVersion:    published.Value.Version,
		BaseRevisionNumber: &base,
	})
	if err != nil {
		t.Fatal(err)
	}
	removed, err := service.ReplaceRoadmapDraft(ctx, cc, application.ReplaceRoadmapDraftCommand{
		Scope:                outcome.Scope(),
		RoadmapID:            roadmap.ID,
		ExpectedVersion:      nextDraft.Value.Version,
		ExpectedDraftVersion: nextDraft.Value.Draft.DraftVersion,
		Nodes:                nil,
	})
	if err != nil {
		t.Fatal(err)
	}
	replanned, err := service.PublishRoadmapDraft(ctx, cc, application.PublishRoadmapDraftCommand{
		Scope:                outcome.Scope(),
		RoadmapID:            roadmap.ID,
		ExpectedVersion:      removed.Value.Version,
		ExpectedDraftVersion: removed.Value.Draft.DraftVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(replanned.Value.Revisions) != 2 {
		t.Fatalf("revision count = %d, want 2", len(replanned.Value.Revisions))
	}
	liveBeforeCancel, err := service.GetWorkItem(ctx, outcome.Scope(), work.ID)
	if err != nil {
		t.Fatal(err)
	}
	if liveBeforeCancel.Value.Lifecycle != domain.WorkItemLifecycleTodo {
		t.Fatalf("removing node changed live lifecycle to %q", liveBeforeCancel.Value.Lifecycle)
	}

	if _, err := service.CancelWorkItem(ctx, cc, application.CancelWorkItemCommand{
		Scope:           outcome.Scope(),
		WorkItemID:      work.ID,
		ExpectedVersion: liveBeforeCancel.Value.Version,
		Reason:          "operational cancellation after replanning",
	}); err != nil {
		t.Fatal(err)
	}
	readRevision, err := service.GetRoadmapRevision(ctx, outcome.Scope(), roadmap.ID, 1)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(readRevision.Value, revisionOne) {
		t.Fatalf("live cancellation rewrote historical revision\nwant=%#v\ngot=%#v", revisionOne, readRevision.Value)
	}
}

func TestWave11StaleRoadmapAndDraftVersionsFailExplicitly(t *testing.T) {
	ctx := context.Background()
	service, _ := newWave09Service(t)
	cc := commandContext()
	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  domain.MustParseID("0199f334-0000-7000-8000-000000000001"),
		Title:        "Stale planning writes",
		DesiredState: "stale versions fail explicitly",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value
	roadmapResult, err := service.CreateRoadmap(ctx, cc, application.CreateRoadmapCommand{
		Scope: outcome.Scope(),
		PlanScope: domain.RoadmapPlanScope{
			Kind: domain.RoadmapScopeOutcome,
			ID:   outcome.ID,
		},
		Title: "Versioned plan",
	})
	if err != nil {
		t.Fatal(err)
	}
	roadmap := roadmapResult.Value
	opened, err := service.OpenRoadmapDraft(ctx, cc, application.OpenRoadmapDraftCommand{
		Scope: outcome.Scope(), RoadmapID: roadmap.ID, ExpectedVersion: roadmap.Version,
	})
	if err != nil {
		t.Fatal(err)
	}

	_, err = service.ReplaceRoadmapDraft(ctx, cc, application.ReplaceRoadmapDraftCommand{
		Scope:                outcome.Scope(),
		RoadmapID:            roadmap.ID,
		ExpectedVersion:      roadmap.Version,
		ExpectedDraftVersion: opened.Value.Draft.DraftVersion,
	})
	if code, ok := domain.ErrorCodeOf(err); !ok || code != domain.ErrorCodeVersionConflict {
		t.Fatalf("stale roadmap version error = %v, code=%q", err, code)
	}

	_, err = service.ReplaceRoadmapDraft(ctx, cc, application.ReplaceRoadmapDraftCommand{
		Scope:                outcome.Scope(),
		RoadmapID:            roadmap.ID,
		ExpectedVersion:      opened.Value.Version,
		ExpectedDraftVersion: opened.Value.Draft.DraftVersion + 1,
	})
	if code, ok := domain.ErrorCodeOf(err); !ok || code != domain.ErrorCodeVersionConflict {
		t.Fatalf("stale draft version error = %v, code=%q", err, code)
	}
}
