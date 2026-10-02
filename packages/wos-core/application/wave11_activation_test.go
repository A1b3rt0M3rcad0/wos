package application_test

import (
	"context"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestWave11ActiveRoadmapSlotIsUniqueAndArchiveClearsIt(t *testing.T) {
	ctx := context.Background()
	service, _ := newWave09Service(t)
	cc := commandContext()
	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  domain.MustParseID("0199f310-0000-7000-8000-000000000001"),
		Title:        "Active plan",
		DesiredState: "only one active roadmap revision per scope",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value
	planScope := domain.RoadmapPlanScope{Kind: domain.RoadmapScopeOutcome, ID: outcome.ID}

	publish := func(title string) domain.Roadmap {
		createdRoadmap, err := service.CreateRoadmap(ctx, cc, application.CreateRoadmapCommand{
			Scope: outcome.Scope(), PlanScope: planScope, Title: title,
		})
		if err != nil {
			t.Fatal(err)
		}
		roadmap := createdRoadmap.Value
		opened, err := service.OpenRoadmapDraft(ctx, cc, application.OpenRoadmapDraftCommand{
			Scope: outcome.Scope(), RoadmapID: roadmap.ID, ExpectedVersion: roadmap.Version,
		})
		if err != nil {
			t.Fatal(err)
		}
		published, err := service.PublishRoadmapDraft(ctx, cc, application.PublishRoadmapDraftCommand{
			Scope: outcome.Scope(), RoadmapID: roadmap.ID,
			ExpectedVersion: opened.Value.Version,
			ExpectedDraftVersion: opened.Value.Draft.DraftVersion,
		})
		if err != nil {
			t.Fatal(err)
		}
		return published.Value
	}

	first := publish("First")
	second := publish("Second")

	if _, err := service.ActivateRoadmapRevision(ctx, cc, application.ActivateRoadmapRevisionCommand{
		Scope: outcome.Scope(), RoadmapID: first.ID,
		ExpectedVersion: first.Version, RevisionNumber: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ActivateRoadmapRevision(ctx, cc, application.ActivateRoadmapRevisionCommand{
		Scope: outcome.Scope(), RoadmapID: second.ID,
		ExpectedVersion: second.Version, RevisionNumber: 1,
	}); err != nil {
		t.Fatal(err)
	}

	slot, _, err := service.GetActiveRoadmapSlot(ctx, outcome.Scope(), planScope)
	if err != nil {
		t.Fatal(err)
	}
	if slot == nil || slot.RoadmapID != second.ID || slot.RevisionNumber != 1 {
		t.Fatalf("active slot = %#v", slot)
	}
	history, _, err := service.ListRoadmapActivationHistory(ctx, outcome.Scope(), planScope)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 3 ||
		history[0].Action != domain.RoadmapActivationActivated ||
		history[1].Action != domain.RoadmapActivationSuperseded ||
		history[2].Action != domain.RoadmapActivationActivated {
		t.Fatalf("activation history = %#v", history)
	}

	archived, err := service.ArchiveRoadmap(ctx, cc, application.ArchiveRoadmapCommand{
		Scope: outcome.Scope(), RoadmapID: second.ID, ExpectedVersion: second.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	slot, _, err = service.GetActiveRoadmapSlot(ctx, outcome.Scope(), planScope)
	if err != nil {
		t.Fatal(err)
	}
	if slot != nil {
		t.Fatalf("archived roadmap left active slot = %#v", slot)
	}
	history, _, err = service.ListRoadmapActivationHistory(ctx, outcome.Scope(), planScope)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 4 || history[3].Action != domain.RoadmapActivationDeactivated {
		t.Fatalf("archive activation history = %#v", history)
	}

	reopened, err := service.ReopenRoadmap(ctx, cc, application.ReopenRoadmapCommand{
		Scope: outcome.Scope(), RoadmapID: second.ID, ExpectedVersion: archived.Value.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if reopened.Value.Lifecycle != domain.RoadmapLifecycleOpen {
		t.Fatalf("reopened lifecycle = %q", reopened.Value.Lifecycle)
	}
	slot, _, err = service.GetActiveRoadmapSlot(ctx, outcome.Scope(), planScope)
	if err != nil {
		t.Fatal(err)
	}
	if slot != nil {
		t.Fatal("reopening roadmap silently reactivated revision")
	}
}
