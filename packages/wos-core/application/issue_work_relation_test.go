package application_test

import (
	"context"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestIssueAndCorrectiveTaskUseInformationalRelatesToWithoutChangingReadiness(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	outcome := setupActiveOutcome(t, service)
	task := createTodoWork(t, service, outcome.Scope(), "Correct the reported login issue")
	if task.ObjectiveID == nil {
		t.Fatal("fixture Task must be owned by an Objective")
	}
	issueResult, err := service.CreateIssue(ctx, commandContext(), application.CreateIssueCommand{
		Scope: outcome.Scope(), ObjectiveID: *task.ObjectiveID, Title: "Login form does not submit", Severity: domain.IssueSeverityMajor,
	})
	if err != nil {
		t.Fatal(err)
	}
	issue := issueResult.Value
	readyBefore, err := service.ListReadyWork(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if !containsReady(readyBefore.Items, task.ID) {
		t.Fatal("fixture Task must be ready before its informational Issue relation")
	}
	current, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := len(store.SnapshotDomainEvents(outcome.Scope()))
	related, err := service.RelateIssueToWorkItem(ctx, commandContext(), application.RelateIssueWorkItemCommand{
		Scope: outcome.Scope(), IssueID: issue.ID, ExpectedIssueVersion: issue.Version,
		WorkItemID: task.ID, ExpectedWorkItemVersion: task.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if related.Value.RelationType != domain.RelationTypeRelatesTo || related.Value.Strength != "" || related.Value.SourceRef.Scope != outcome.Scope() || related.Value.TargetRef.Scope != outcome.Scope() {
		t.Fatalf("Issue↔Task link must be a same-Outcome informative relation: %+v", related.Value)
	}
	if !issueTaskEndpoints(related.Value, issue.Ref(), task.Ref()) {
		t.Fatalf("relation endpoints do not identify the Issue and corrective Task: %+v", related.Value)
	}
	if related.OutcomeRevision != current.OutcomeRevision+1 {
		t.Fatalf("link revision = %d, want %d", related.OutcomeRevision, current.OutcomeRevision+1)
	}
	events := store.SnapshotDomainEvents(outcome.Scope())
	if len(events) != eventsBefore+1 || events[len(events)-1].EventType != "relation.created" {
		t.Fatalf("Issue↔Task association did not emit one relation event: %+v", events)
	}
	issueView, err := service.GetEntityWorkspaceView(ctx, outcome.Scope(), application.WorkspaceViewQuery{Subject: application.WorkspaceSubject{Kind: "entity", Ref: issue.Ref()}})
	if err != nil {
		t.Fatal(err)
	}
	taskView, err := service.GetEntityWorkspaceView(ctx, outcome.Scope(), application.WorkspaceViewQuery{Subject: application.WorkspaceSubject{Kind: "entity", Ref: task.Ref()}})
	if err != nil {
		t.Fatal(err)
	}
	if !workspaceRelatedTo(issueView, task.Ref()) || !workspaceRelatedTo(taskView, issue.Ref()) {
		t.Fatalf("association must appear in both entity workspace views: issue=%+v task=%+v", issueView.Relationships, taskView.Relationships)
	}
	readyAfter, err := service.ListReadyWork(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if !containsReady(readyAfter.Items, task.ID) || len(readyAfter.Items) != len(readyBefore.Items) {
		t.Fatalf("informational relation changed Task readiness: before=%+v after=%+v", readyBefore.Items, readyAfter.Items)
	}
	if _, err := service.RelateIssueToWorkItem(ctx, commandContext(), application.RelateIssueWorkItemCommand{
		Scope: outcome.Scope(), IssueID: issue.ID, ExpectedIssueVersion: issue.Version,
		WorkItemID: task.ID, ExpectedWorkItemVersion: task.Version,
	}); err == nil {
		t.Fatal("duplicate active Issue↔Task association unexpectedly succeeded")
	}

	removed, err := service.RemoveIssueWorkItemRelation(ctx, commandContext(), application.RemoveIssueWorkItemRelationCommand{
		Scope: outcome.Scope(), RelationID: related.Value.ID, ExpectedVersion: related.Value.Version, Reason: "The corrective work is no longer relevant",
	})
	if err != nil {
		t.Fatal(err)
	}
	if removed.Value.Lifecycle != domain.RelationLifecycleRemoved {
		t.Fatalf("removed relation lifecycle = %s", removed.Value.Lifecycle)
	}
	if removed.OutcomeRevision != related.OutcomeRevision+1 {
		t.Fatalf("removal revision = %d, want %d", removed.OutcomeRevision, related.OutcomeRevision+1)
	}
	events = store.SnapshotDomainEvents(outcome.Scope())
	if len(events) != eventsBefore+2 || events[len(events)-1].EventType != "relation.removed" {
		t.Fatalf("Issue↔Task removal did not emit one relation event: %+v", events)
	}
	issueView, err = service.GetEntityWorkspaceView(ctx, outcome.Scope(), application.WorkspaceViewQuery{Subject: application.WorkspaceSubject{Kind: "entity", Ref: issue.Ref()}})
	if err != nil {
		t.Fatal(err)
	}
	taskView, err = service.GetEntityWorkspaceView(ctx, outcome.Scope(), application.WorkspaceViewQuery{Subject: application.WorkspaceSubject{Kind: "entity", Ref: task.Ref()}})
	if err != nil {
		t.Fatal(err)
	}
	if workspaceRelatedTo(issueView, task.Ref()) || workspaceRelatedTo(taskView, issue.Ref()) {
		t.Fatal("removed Issue↔Task relation remained in an active workspace view")
	}
	readyFinal, err := service.ListReadyWork(ctx, outcome.Scope())
	if err != nil || !containsReady(readyFinal.Items, task.ID) || len(readyFinal.Items) != len(readyBefore.Items) {
		t.Fatalf("removing informational relation changed readiness: result=%+v err=%v", readyFinal, err)
	}
}

func TestIssueTaskRelationRequiresCurrentIssueAndTaskVersions(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	outcome := setupActiveOutcome(t, service)
	task := createTodoWork(t, service, outcome.Scope(), "Corrective Task")
	issueResult, err := service.CreateIssue(ctx, commandContext(), application.CreateIssueCommand{
		Scope: outcome.Scope(), ObjectiveID: *task.ObjectiveID, Title: "Issue", Severity: domain.IssueSeverityMinor,
	})
	if err != nil {
		t.Fatal(err)
	}
	issue := issueResult.Value
	before, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := len(store.SnapshotDomainEvents(outcome.Scope()))
	_, err = service.RelateIssueToWorkItem(ctx, commandContext(), application.RelateIssueWorkItemCommand{
		Scope: outcome.Scope(), IssueID: issue.ID, ExpectedIssueVersion: issue.Version + 1,
		WorkItemID: task.ID, ExpectedWorkItemVersion: task.Version,
	})
	if code, ok := domain.ErrorCodeOf(err); !ok || code != domain.ErrorCodePreconditionFailed {
		t.Fatalf("stale Issue version error = %v", err)
	}
	_, err = service.RelateIssueToWorkItem(ctx, commandContext(), application.RelateIssueWorkItemCommand{
		Scope: outcome.Scope(), IssueID: issue.ID, ExpectedIssueVersion: issue.Version,
		WorkItemID: task.ID, ExpectedWorkItemVersion: task.Version + 1,
	})
	if code, ok := domain.ErrorCodeOf(err); !ok || code != domain.ErrorCodePreconditionFailed {
		t.Fatalf("stale Task version error = %v", err)
	}
	current, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if current.OutcomeRevision != before.OutcomeRevision || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsBefore {
		t.Fatalf("stale versions produced side effects: outcome=%+v events=%+v", current, store.SnapshotDomainEvents(outcome.Scope()))
	}
	issueView, err := service.GetEntityWorkspaceView(ctx, outcome.Scope(), application.WorkspaceViewQuery{Subject: application.WorkspaceSubject{Kind: "entity", Ref: issue.Ref()}})
	if err != nil {
		t.Fatal(err)
	}
	taskView, err := service.GetEntityWorkspaceView(ctx, outcome.Scope(), application.WorkspaceViewQuery{Subject: application.WorkspaceSubject{Kind: "entity", Ref: task.Ref()}})
	if err != nil {
		t.Fatal(err)
	}
	if workspaceRelatedTo(issueView, task.Ref()) || workspaceRelatedTo(taskView, issue.Ref()) {
		t.Fatal("stale version persisted an Issue↔Task relation")
	}
}

func TestRemoveIssueWorkItemRelationCannotRemoveDependencyRelation(t *testing.T) {
	ctx := context.Background()
	service, store := newService(t)
	outcome := setupActiveOutcome(t, service)
	source := deferPlanningWork(t, service, createTodoWork(t, service, outcome.Scope(), "Corrective Task"))
	target := createTodoWork(t, service, outcome.Scope(), "Prerequisite Task")
	dependency, err := service.AddDependency(ctx, commandContext(), application.AddDependencyCommand{
		Scope: outcome.Scope(), SourceRef: source.Ref(), TargetRef: target.Ref(), Strength: domain.DependencyStrengthHard,
	})
	if err != nil {
		t.Fatal(err)
	}
	before, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	eventsBefore := len(store.SnapshotDomainEvents(outcome.Scope()))
	_, err = service.RemoveIssueWorkItemRelation(ctx, commandContext(), application.RemoveIssueWorkItemRelationCommand{
		Scope: outcome.Scope(), RelationID: dependency.Value.ID, ExpectedVersion: dependency.Value.Version, Reason: "Do not remove an operational dependency through the Issue link command",
	})
	if code, ok := domain.ErrorCodeOf(err); !ok || code != domain.ErrorCodeInvalidRelation {
		t.Fatalf("removing dependency through Issue relation command error = %v", err)
	}
	after, err := service.GetOutcome(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if after.OutcomeRevision != before.OutcomeRevision || len(store.SnapshotDomainEvents(outcome.Scope())) != eventsBefore {
		t.Fatalf("invalid removal produced side effects: before=%+v after=%+v", before, after)
	}
	dependencies := dependencyRelations(t, store, outcome.Scope())
	if len(dependencies) != 1 || dependencies[0].ID != dependency.Value.ID || dependencies[0].Lifecycle != domain.RelationLifecycleActive {
		t.Fatalf("invalid removal changed the dependency: %+v", dependencies)
	}
}

func issueTaskEndpoints(relation domain.Relation, issue, task domain.EntityRef) bool {
	return relation.SourceRef == issue && relation.TargetRef == task || relation.SourceRef == task && relation.TargetRef == issue
}

func workspaceRelatedTo(view application.EntityWorkspaceView, ref domain.EntityRef) bool {
	for _, item := range view.Relationships.Items {
		if item.SourceOfTruth == "relation" && item.Kind == string(domain.RelationTypeRelatesTo) && (item.Source == ref || item.Target == ref) {
			return true
		}
	}
	return false
}
