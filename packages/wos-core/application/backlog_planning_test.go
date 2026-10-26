package application_test

import (
	"context"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"testing"
)

func TestActivatedTaskFreezesDefinitionButKeepsProofAndCompletionOperational(t *testing.T) {
	ctx := context.Background()
	s, store := newService(t)
	outcome := setupActiveOutcome(t, s)
	created, err := s.CreateWorkItem(ctx, commandContext(), a.CreateWorkItemCommand{Scope: outcome.Scope(), ObjectiveID: objectiveForWorkFixture(t, s, outcome.Scope()), Title: "Agreed work", Priority: d.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	work := created.Value
	criterion, err := s.AddCriterion(ctx, commandContext(), a.AddCriterionCommand{Owner: work.Ref(), ExpectedVersion: work.Version, Title: "Result inspected", Required: true, VerificationMode: d.VerificationModeAttestation})
	if err != nil {
		t.Fatal(err)
	}
	target := createTodoWork(t, s, work.Scope, "Independent target")
	relation, err := s.AddDependency(ctx, commandContext(), a.AddDependencyCommand{Scope: work.Scope, SourceRef: work.Ref(), TargetRef: target.Ref(), Strength: d.DependencyStrengthAdvisory})
	if err != nil {
		t.Fatal(err)
	}
	work = activatePlanningWork(t, s, work)
	title := "Changed after agreement"
	before := len(store.SnapshotDomainEvents(work.Scope))
	for name, mutate := range map[string]func() error{
		"details": func() error {
			_, e := s.UpdateWorkItem(ctx, commandContext(), a.UpdateWorkItemCommand{Scope: work.Scope, WorkItemID: work.ID, ExpectedVersion: work.Version, Title: &title})
			return e
		},
		"instructions": func() error {
			_, e := s.SetWorkItemExecutionSpec(ctx, commandContext(), a.SetWorkItemExecutionSpecCommand{Scope: work.Scope, WorkItemID: work.ID, ExpectedVersion: work.Version, ExecutionSpec: &d.ExecutionSpec{Instructions: []string{"Changed"}}})
			return e
		},
		"schedule": func() error {
			_, e := s.SetWorkItemNotBefore(ctx, commandContext(), a.SetWorkItemNotBeforeCommand{Scope: work.Scope, WorkItemID: work.ID, ExpectedVersion: work.Version})
			return e
		},
		"placement": func() error {
			_, e := s.SetWorkItemObjective(ctx, commandContext(), a.SetWorkItemObjectiveCommand{Scope: work.Scope, WorkItemID: work.ID, ExpectedVersion: work.Version})
			return e
		},
		"new criterion": func() error {
			_, e := s.AddCriterion(ctx, commandContext(), a.AddCriterionCommand{Owner: work.Ref(), ExpectedVersion: work.Version, Title: "Late obligation", Required: true, VerificationMode: d.VerificationModeAttestation})
			return e
		},
		"revise criterion": func() error {
			_, e := s.ReviseCriterion(ctx, commandContext(), a.ReviseCriterionCommand{Owner: work.Ref(), ExpectedVersion: work.Version, CriterionID: criterion.Value.ID, Title: "Changed obligation", Required: true, VerificationMode: d.VerificationModeAttestation})
			return e
		},
		"retire criterion": func() error {
			_, e := s.RetireCriterion(ctx, commandContext(), a.RetireCriterionCommand{Owner: work.Ref(), ExpectedVersion: work.Version, CriterionID: criterion.Value.ID})
			return e
		},
		"remove dependency": func() error {
			_, e := s.RemoveDependency(ctx, commandContext(), a.RemoveDependencyCommand{Scope: work.Scope, RelationID: relation.Value.ID, ExpectedVersion: relation.Value.Version, Reason: "Late definition edit"})
			return e
		},
		"dependency": func() error {
			_, e := s.AddDependency(ctx, commandContext(), a.AddDependencyCommand{Scope: work.Scope, SourceRef: work.Ref(), TargetRef: target.Ref(), Strength: d.DependencyStrengthHard})
			return e
		},
	} {
		t.Run(name, func(t *testing.T) {
			requireEditingReason(t, mutate(), d.ErrorCodePreconditionFailed, "work_item_planning_frozen")
		})
	}
	loaded, err := s.GetWorkItem(ctx, work.Scope, work.ID)
	if err != nil || loaded.Value.Version != work.Version || loaded.Value.Title != work.Title || len(store.SnapshotDomainEvents(work.Scope)) != before {
		t.Fatal("rejected planning mutated state/events", err)
	}
	assertPlanningCapabilities := func() {
		view, err := s.GetEntityWorkspaceView(ctx, work.Scope, a.WorkspaceViewQuery{Subject: a.WorkspaceSubject{Kind: "entity", Ref: work.Ref()}})
		if err != nil {
			t.Fatal(err)
		}
		found := 0
		for _, cap := range view.Capabilities {
			switch cap.ActionID {
			case "UpdateWorkItem", "SetWorkItemExecutionSpec", "SetWorkItemNotBefore", "SetWorkItemObjective", "AddCriterion":
				found++
				if cap.Availability != "blocked" || len(cap.ReasonCodes) != 1 || cap.ReasonCodes[0] != "work_item_planning_frozen" {
					t.Fatalf("capability disagrees with command gate: %+v", cap)
				}
			}
		}
		if found != 5 {
			t.Fatalf("missing planning capabilities: %d", found)
		}
	}
	assertPlanningCapabilities()
	_, err = s.RecordCriterionAssessment(ctx, commandContext(), a.RecordCriterionAssessmentCommand{Owner: work.Ref(), ExpectedVersion: work.Version, CriterionID: criterion.Value.ID, CriterionRevision: criterion.Value.Revision, Result: d.AssessmentResultMet, Rationale: "Inspected the agreed condition"})
	if err != nil {
		t.Fatal("planning freeze blocked legitimate proof", err)
	}
	loaded, err = s.GetWorkItem(ctx, work.Scope, work.ID)
	if err != nil {
		t.Fatal(err)
	}
	claimed, err := s.ClaimWorkItem(ctx, commandContext(), a.ClaimWorkItemCommand{Scope: work.Scope, WorkItemID: work.ID, ExpectedVersion: loaded.Value.Version})
	if err != nil {
		t.Fatal(err)
	}
	w := claimed.Value
	assertPlanningCapabilities()
	completed, err := s.CompleteWorkItem(ctx, commandContext(), a.CompleteWorkItemCommand{Scope: w.Scope, WorkItemID: w.ID, ExpectedVersion: w.Version, ClaimID: w.CurrentLease.ClaimID, FencingToken: w.CurrentLease.FencingToken, ResultSummary: "Delivered and inspected", Reason: "Accept the agreed result"})
	if err != nil || completed.Value.Lifecycle != d.WorkItemLifecycleDone {
		t.Fatal("legitimate completion failed", err)
	}
}
