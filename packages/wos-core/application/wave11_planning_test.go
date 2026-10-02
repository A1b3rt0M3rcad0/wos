package application_test

import (
	"context"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestWave11ObjectiveScopedRoadmapValidatesOperationalReferences(t *testing.T) {
	ctx := context.Background()
	service, _ := newWave09Service(t)
	cc := commandContext()

	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  domain.MustParseID("0199f300-0000-7000-8000-000000000001"),
		Title:        "Planning scope",
		DesiredState: "Roadmap references live work without owning it",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value

	rootResult, err := service.CreateObjective(ctx, cc, application.CreateObjectiveCommand{
		Scope: outcome.Scope(), Title: "Root objective", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	root := rootResult.Value

	childResult, err := service.CreateObjective(ctx, cc, application.CreateObjectiveCommand{
		Scope: outcome.Scope(), Title: "Child objective", Priority: domain.PriorityNormal,
		ParentObjectiveID: &root.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	child := childResult.Value

	otherResult, err := service.CreateObjective(ctx, cc, application.CreateObjectiveCommand{
		Scope: outcome.Scope(), Title: "Other branch", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	other := otherResult.Value

	workResult, err := service.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{
		Scope: outcome.Scope(), Title: "Child work", Priority: domain.PriorityNormal,
		Lifecycle: domain.WorkItemLifecycleTodo, ObjectiveID: &child.ID,
	})
	if err != nil {
		t.Fatal(err)
	}
	work := workResult.Value

	roadmapResult, err := service.CreateRoadmap(ctx, cc, application.CreateRoadmapCommand{
		Scope:     outcome.Scope(),
		PlanScope: domain.RoadmapPlanScope{Kind: domain.RoadmapScopeObjective, ID: root.ID},
		Title:     "Root objective plan",
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
	beforeWork, err := service.GetWorkItem(ctx, outcome.Scope(), work.ID)
	if err != nil {
		t.Fatal(err)
	}

	nodes := []domain.RoadmapNode{
		{NodeKey: "phase", NodeType: domain.RoadmapNodePhase, Title: "Phase", Position: 0},
		{NodeKey: "child", NodeType: domain.RoadmapNodeReference, ParentNodeKey: "phase", TargetRef: refPtr(child.Ref()), Title: child.Title, Position: 0},
		{NodeKey: "work", NodeType: domain.RoadmapNodeReference, ParentNodeKey: "phase", TargetRef: refPtr(work.Ref()), Title: work.Title, Position: 1},
	}
	updated, err := service.ReplaceRoadmapDraft(ctx, cc, application.ReplaceRoadmapDraftCommand{
		Scope: outcome.Scope(), RoadmapID: roadmap.ID,
		ExpectedVersion: opened.Value.Version, ExpectedDraftVersion: opened.Value.Draft.DraftVersion,
		Nodes: nodes,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Value.Draft == nil || len(updated.Value.Draft.Nodes) != 3 {
		t.Fatalf("updated draft = %#v", updated.Value.Draft)
	}

	afterWork, err := service.GetWorkItem(ctx, outcome.Scope(), work.ID)
	if err != nil {
		t.Fatal(err)
	}
	if afterWork.Value.Version != beforeWork.Value.Version ||
		afterWork.Value.Lifecycle != beforeWork.Value.Lifecycle {
		t.Fatal("Roadmap editing mutated referenced operational WorkItem")
	}

	invalidNodes := []domain.RoadmapNode{
		{NodeKey: "other", NodeType: domain.RoadmapNodeReference, TargetRef: refPtr(other.Ref()), Title: other.Title, Position: 0},
	}
	if _, err := service.ReplaceRoadmapDraft(ctx, cc, application.ReplaceRoadmapDraftCommand{
		Scope: outcome.Scope(), RoadmapID: roadmap.ID,
		ExpectedVersion: updated.Value.Version, ExpectedDraftVersion: updated.Value.Draft.DraftVersion,
		Nodes: invalidNodes,
	}); err == nil {
		t.Fatal("Objective-scoped Roadmap accepted reference outside its subtree")
	}
}

func refPtr(ref domain.EntityRef) *domain.EntityRef {
	value := ref
	return &value
}

func TestWave11OutcomeRoadmapAcceptsOutcomeCriterionMilestone(t *testing.T) {
	ctx := context.Background()
	service, _ := newWave09Service(t)
	cc := commandContext()

	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  domain.MustParseID("0199f301-0000-7000-8000-000000000001"),
		Title:        "Outcome milestone",
		DesiredState: "milestone can reference Outcome criterion",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value

	criterionResult, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner:            outcome.Ref(),
		ExpectedVersion:  outcome.Version,
		Title:            "Outcome verified",
		Required:         true,
		VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	criterion := criterionResult.Value

	roadmapResult, err := service.CreateRoadmap(ctx, cc, application.CreateRoadmapCommand{
		Scope:     outcome.Scope(),
		PlanScope: domain.RoadmapPlanScope{Kind: domain.RoadmapScopeOutcome, ID: outcome.ID},
		Title:     "Outcome plan",
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
		ExpectedVersion:      opened.Value.Version,
		ExpectedDraftVersion: opened.Value.Draft.DraftVersion,
		Nodes: []domain.RoadmapNode{
			{
				NodeKey:  "milestone",
				NodeType: domain.RoadmapNodeMilestone,
				Title:    "Outcome verified",
				Position: 0,
				CriterionRefs: []domain.RoadmapCriterionRef{
					{OwnerRef: outcome.Ref(), CriterionID: criterion.ID},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
}


func TestWave11PublicationSnapshotsLiveReferencesAndCriteria(t *testing.T) {
	ctx := context.Background()
	service, _ := newWave09Service(t)
	cc := commandContext()

	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  domain.MustParseID("0199f302-0000-7000-8000-000000000001"),
		Title:        "Published plan",
		DesiredState: "historical planning remains intelligible",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value

	objectiveResult, err := service.CreateObjective(ctx, cc, application.CreateObjectiveCommand{
		Scope: outcome.Scope(), Title: "Original objective title", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	objective := objectiveResult.Value
	criterionResult, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner: objective.Ref(), ExpectedVersion: objective.Version,
		Title: "Original criterion title", Required: true,
		VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	criterion := criterionResult.Value

	roadmapResult, err := service.CreateRoadmap(ctx, cc, application.CreateRoadmapCommand{
		Scope: outcome.Scope(),
		PlanScope: domain.RoadmapPlanScope{Kind: domain.RoadmapScopeOutcome, ID: outcome.ID},
		Title: "Main plan",
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
	nodes := []domain.RoadmapNode{
		{
			NodeKey: "objective", NodeType: domain.RoadmapNodeReference,
			TargetRef: refPtr(objective.Ref()), Title: "Objective in plan", Position: 0,
		},
		{
			NodeKey: "milestone", NodeType: domain.RoadmapNodeMilestone,
			Title: "Criterion milestone", Position: 1,
			CriterionRefs: []domain.RoadmapCriterionRef{
				{OwnerRef: objective.Ref(), CriterionID: criterion.ID},
			},
		},
	}
	edited, err := service.ReplaceRoadmapDraft(ctx, cc, application.ReplaceRoadmapDraftCommand{
		Scope: outcome.Scope(), RoadmapID: roadmap.ID,
		ExpectedVersion: opened.Value.Version, ExpectedDraftVersion: opened.Value.Draft.DraftVersion,
		Nodes: nodes,
	})
	if err != nil {
		t.Fatal(err)
	}
	published, err := service.PublishRoadmapDraft(ctx, cc, application.PublishRoadmapDraftCommand{
		Scope: outcome.Scope(), RoadmapID: roadmap.ID,
		ExpectedVersion: edited.Value.Version, ExpectedDraftVersion: edited.Value.Draft.DraftVersion,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(published.Value.Revisions) != 1 || published.Value.Draft != nil {
		t.Fatalf("published roadmap = %#v", published.Value)
	}
	revision := published.Value.Revisions[0]
	if revision.ContentHash == "" {
		t.Fatal("published revision content hash is empty")
	}
	if revision.Nodes[0].ReferenceSnapshot == nil ||
		revision.Nodes[0].ReferenceSnapshot.Title != "Original objective title" {
		t.Fatalf("reference snapshot = %#v", revision.Nodes[0].ReferenceSnapshot)
	}
	if len(revision.Nodes[1].CriterionSnapshots) != 1 ||
		revision.Nodes[1].CriterionSnapshots[0].Title != "Original criterion title" ||
		revision.Nodes[1].CriterionSnapshots[0].CriterionRevision != criterion.Revision {
		t.Fatalf("criterion snapshots = %#v", revision.Nodes[1].CriterionSnapshots)
	}
}
