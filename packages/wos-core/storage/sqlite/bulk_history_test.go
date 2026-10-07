package sqlite

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestBulkCollectionsPreserveSingleAggregateHistory(t *testing.T) {
	ctx := context.Background()
	s, _, outcome := releaseServices(t)
	scope := outcome.Scope()
	cc := releaseCC(90, "")
	check := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err := s.AddCriterion(ctx, cc, application.AddCriterionCommand{Owner: outcome.Ref(), ExpectedVersion: 1, Title: "Outcome", VerificationMode: domain.VerificationModeAttestation, Required: true})
	check(err)
	_, err = s.ActivateOutcome(ctx, cc, application.ActivateOutcomeCommand{Scope: scope, ExpectedVersion: 2})
	check(err)
	obj, err := s.CreateObjective(ctx, cc, application.CreateObjectiveCommand{Scope: scope, Title: "Rich objective", Priority: domain.PriorityNormal})
	check(err)
	work, err := s.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{Scope: scope, ObjectiveID: &obj.Value.ID, Title: "Rich work", Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleTodo})
	check(err)
	_, err = s.CreateObjective(ctx, cc, application.CreateObjectiveCommand{Scope: scope, Title: "Empty objective", Priority: domain.PriorityNormal})
	check(err)
	_, err = s.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{Scope: scope, Title: "Empty work", Priority: domain.PriorityNormal})
	check(err)
	proof, err := s.RegisterEvidence(ctx, cc, application.RegisterEvidenceCommand{Scope: scope, EvidenceType: domain.EvidenceTypeTestResult, Description: "Proof", SourceRef: domain.SourceReference{Provider: "test", URI: "https://example.test/proof"}, CapturedAt: time.Now().UTC()})
	check(err)
	for _, owner := range []domain.EntityRef{obj.Value.Ref(), work.Value.Ref()} {
		criterion, err := s.AddCriterion(ctx, cc, application.AddCriterionCommand{Owner: owner, ExpectedVersion: 1, Title: "Observed", VerificationMode: domain.VerificationModeEvidenceReview, Required: true})
		check(err)
		_, err = s.RecordCriterionAssessment(ctx, cc, application.RecordCriterionAssessmentCommand{Owner: owner, ExpectedVersion: 2, CriterionID: criterion.Value.ID, CriterionRevision: 1, Result: domain.AssessmentResultMet, Rationale: "Reviewed", EvidenceIDs: []domain.ID{proof.Value.ID}})
		check(err)
	}
	actors := []domain.ActorRef{{Kind: domain.ActorKindHuman, Provider: "test", ID: "reviewer"}}
	_, err = s.SetObjectiveOwners(ctx, cc, application.SetObjectiveOwnersCommand{Scope: scope, ObjectiveID: obj.Value.ID, ExpectedVersion: 3, OwnerRefs: actors})
	check(err)
	_, err = s.SetWorkItemAssignees(ctx, cc, application.SetWorkItemAssigneesCommand{Scope: scope, WorkItemID: work.Value.ID, ExpectedVersion: 3, AssigneeRefs: actors})
	check(err)
	_, err = s.StartObjective(ctx, cc, application.StartObjectiveCommand{Scope: scope, ObjectiveID: obj.Value.ID, ExpectedVersion: 4})
	check(err)
	claimed, err := s.ClaimWorkItem(ctx, cc, application.ClaimWorkItemCommand{Scope: scope, WorkItemID: work.Value.ID, ExpectedVersion: 4, TTL: time.Minute})
	check(err)
	_, err = s.CompleteWorkItem(ctx, cc, application.CompleteWorkItemCommand{Scope: scope, WorkItemID: work.Value.ID, ExpectedVersion: 5, ClaimID: claimed.Value.CurrentLease.ClaimID, FencingToken: claimed.Value.CurrentLease.FencingToken, ResultSummary: "Done", Reason: "Proof recorded"})
	check(err)
	_, err = s.AchieveObjective(ctx, cc, application.AchieveObjectiveCommand{Scope: scope, ObjectiveID: obj.Value.ID, ExpectedVersion: 5, Reason: "Explicit achievement"})
	check(err)
	compare := func() {
		t.Helper()
		snapshot, err := s.GetOutcomeState(ctx, scope)
		check(err)
		equal := func(a, b any) {
			t.Helper()
			x, err := json.Marshal(a)
			check(err)
			y, err := json.Marshal(b)
			check(err)
			if string(x) != string(y) {
				t.Fatalf("bulk/single history divergence\nbulk: %s\nsingle: %s", x, y)
			}
		}
		for _, v := range snapshot.Objectives {
			one, err := s.GetObjective(ctx, scope, v.ID)
			check(err)
			equal(v, one.Value)
		}
		for _, v := range snapshot.WorkItems {
			one, err := s.GetWorkItem(ctx, scope, v.ID)
			check(err)
			equal(v, one.Value)
		}
	}
	compare()
	_, err = s.ReopenObjective(ctx, cc, application.ReopenObjectiveCommand{Scope: scope, ObjectiveID: obj.Value.ID, ExpectedVersion: 6, Reason: "Another review"})
	check(err)
	_, err = s.ReopenWorkItem(ctx, cc, application.ReopenWorkItemCommand{Scope: scope, WorkItemID: work.Value.ID, ExpectedVersion: 6, Reason: "Another execution"})
	check(err)
	compare()
}

func TestActivePlanProjectsLiveStateWithoutRewritingPublication(t *testing.T) {
	ctx := context.Background()
	s, _, outcome := releaseServices(t)
	scope := outcome.Scope()
	cc := releaseCC(91, "")
	check := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err := s.AddCriterion(ctx, cc, application.AddCriterionCommand{Owner: outcome.Ref(), ExpectedVersion: 1, Title: "Outcome", VerificationMode: domain.VerificationModeAttestation, Required: true})
	check(err)
	_, err = s.ActivateOutcome(ctx, cc, application.ActivateOutcomeCommand{Scope: scope, ExpectedVersion: 2})
	check(err)
	work, err := s.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{Scope: scope, Title: "Published work title", Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleTodo})
	check(err)
	plan, err := s.CreateRoadmap(ctx, cc, application.CreateRoadmapCommand{Scope: scope, PlanScope: domain.RoadmapPlanScope{Kind: domain.RoadmapScopeOutcome, ID: outcome.ID}, Title: "Current plan"})
	check(err)
	draft, err := s.OpenRoadmapDraft(ctx, cc, application.OpenRoadmapDraftCommand{Scope: scope, RoadmapID: plan.Value.ID, ExpectedVersion: 1})
	check(err)
	ref := work.Value.Ref()
	edited, err := s.ReplaceRoadmapDraft(ctx, cc, application.ReplaceRoadmapDraftCommand{Scope: scope, RoadmapID: plan.Value.ID, ExpectedVersion: draft.Value.Version, ExpectedDraftVersion: draft.Value.Draft.DraftVersion, Nodes: []domain.RoadmapNode{{NodeKey: "delivery", NodeType: domain.RoadmapNodeReference, TargetRef: &ref, Title: "Scheduled delivery"}}})
	check(err)
	published, err := s.PublishRoadmapDraft(ctx, cc, application.PublishRoadmapDraftCommand{Scope: scope, RoadmapID: plan.Value.ID, ExpectedVersion: edited.Value.Version, ExpectedDraftVersion: edited.Value.Draft.DraftVersion})
	check(err)
	_, err = s.ActivateRoadmapRevision(ctx, cc, application.ActivateRoadmapRevisionCommand{Scope: scope, RoadmapID: plan.Value.ID, ExpectedVersion: published.Value.Version, RevisionNumber: 1})
	check(err)
	before, err := s.GetRoadmapRevision(ctx, scope, plan.Value.ID, 1)
	check(err)
	original, _ := json.Marshal(before.Value)
	title := "Renamed live work"
	_, err = s.UpdateWorkItem(ctx, cc, application.UpdateWorkItemCommand{Scope: scope, WorkItemID: work.Value.ID, ExpectedVersion: 1, Title: &title})
	check(err)
	blocker, err := s.CreateBlocker(ctx, cc, application.CreateBlockerCommand{Scope: scope, BlockedRef: ref, Propagation: domain.BlockerPropagationDirect, Description: "Release explicitly"})
	check(err)
	snapshot, err := s.GetContinuity(ctx, scope, 25)
	check(err)
	raw, _ := json.Marshal(snapshot.Sections["active_plan_references"])
	var refs []application.PlanLiveReference
	check(json.Unmarshal(raw, &refs))
	if len(refs) != 1 || refs[0].Current == nil || refs[0].Current.Title != title || refs[0].PublishedReferenceTitle != "Published work title" || refs[0].PlanLabel != "Scheduled delivery" || refs[0].Operational == nil || !refs[0].Operational.IsBlocked {
		t.Fatalf("incorrect live plan projection: %+v", refs)
	}
	_, err = s.ResolveBlocker(ctx, cc, application.ResolveBlockerCommand{Scope: scope, BlockerID: blocker.Value.ID, ExpectedVersion: 1, ResolutionSummary: "Explicit release"})
	check(err)
	snapshot, err = s.GetContinuity(ctx, scope, 25)
	check(err)
	raw, _ = json.Marshal(snapshot.Sections["active_plan_references"])
	check(json.Unmarshal(raw, &refs))
	if refs[0].Operational.IsBlocked || refs[0].Operational.DisplayState != domain.WorkItemDisplayStateReady {
		t.Fatal("plan failed to reflect live release")
	}
	after, err := s.GetRoadmapRevision(ctx, scope, plan.Value.ID, 1)
	check(err)
	unchanged, _ := json.Marshal(after.Value)
	if string(original) != string(unchanged) {
		t.Fatal("live state rewrote immutable publication")
	}
	_, err = s.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{Scope: scope, Title: "Second graph edge", Priority: domain.PriorityNormal})
	check(err)
	graph, err := s.GetOutcomeGraph(ctx, scope, application.GraphQuery{Depth: 8, Limit: 1})
	check(err)
	if !graph.Truncated {
		t.Fatal("bounded graph failed to declare omissions")
	}
	nodes := map[domain.EntityRef]bool{}
	for _, v := range graph.Nodes {
		nodes[v.Ref] = true
	}
	for _, e := range graph.Edges {
		if !nodes[e.Source] || !nodes[e.Target] {
			t.Fatal("truncated graph contains false references")
		}
	}
	_, err = s.GetWorkContext(ctx, scope, work.Value.ID)
	check(err)
}
