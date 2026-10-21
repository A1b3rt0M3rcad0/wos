package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

type revokingTransactionalPermissionReader struct {
	revokeAfterExternalRead bool
	revoked                 bool
	inUnitCalls             int
}

func (a *revokingTransactionalPermissionReader) Authorize(_ context.Context, request ports.AuthorizationRequest) error {
	if request.Permission != ports.PermissionStateRead {
		return domain.NewError(domain.ErrorCodeForbidden, "not granted")
	}
	if a.revoked {
		return domain.NewError(domain.ErrorCodeForbidden, "read permission revoked")
	}
	if a.revokeAfterExternalRead {
		a.revokeAfterExternalRead = false
		a.revoked = true
	}
	return nil
}

func (a *revokingTransactionalPermissionReader) AuthorizeInUnitOfWork(_ context.Context, _ ports.UnitOfWork, request ports.AuthorizationRequest) error {
	a.inUnitCalls++
	if request.Permission == ports.PermissionStateRead && !a.revoked {
		return nil
	}
	return domain.NewError(domain.ErrorCodeForbidden, "permission revoked or not granted")
}

func capabilityForTarget(t *testing.T, view application.EntityWorkspaceView, action string, lifecycle domain.WorkItemLifecycle) application.WorkspaceCapability {
	t.Helper()
	for _, capability := range view.Capabilities {
		if capability.ActionID == action && capability.TargetLifecycle == lifecycle {
			return capability
		}
	}
	t.Fatalf("capability %s targeting %s missing", action, lifecycle)
	return application.WorkspaceCapability{}
}

func TestOperationalCapabilityReasonsMatchActivationWrite(t *testing.T) {
	fixture := newTaskActivationFixture(t, true)
	objective := createTaskActivationObjective(t, fixture, "Planned owner", nil, false)
	task := createTaskActivationTask(t, fixture, objective, domain.WorkItemLifecycleBacklog, false)

	view, err := fixture.service.GetEntityWorkspaceView(fixture.ctx, fixture.scope, application.WorkspaceViewQuery{
		Subject: application.WorkspaceSubject{Kind: "entity", Ref: task.Ref()},
	})
	if err != nil {
		t.Fatal(err)
	}
	var capability *application.WorkspaceCapability
	for i := range view.Capabilities {
		if view.Capabilities[i].ActionID == "ActivateWorkItem" {
			capability = &view.Capabilities[i]
			break
		}
	}
	if capability == nil || capability.Availability != "blocked" {
		t.Fatalf("activation capability = %#v", capability)
	}

	_, err = fixture.service.ActivateWorkItem(fixture.ctx, fixture.command, application.ActivateWorkItemCommand{
		Scope: fixture.scope, WorkItemID: task.ID, ExpectedVersion: task.Version,
	})
	var typed *domain.Error
	if !errors.As(err, &typed) || typed.Details == nil {
		t.Fatalf("activation did not return structured preconditions: %v", err)
	}
	wantCodes := []string{
		string(domain.ReadinessReasonTaskDefinitionIncomplete),
		string(domain.ReadinessReasonObjectiveNotStarted),
	}
	if typed.Details.ReasonCode != wantCodes[0] || !reflect.DeepEqual(typed.Details.ReasonCodes, wantCodes) {
		t.Fatalf("write reasons = %q / %#v, want %#v", typed.Details.ReasonCode, typed.Details.ReasonCodes, wantCodes)
	}
	if !reflect.DeepEqual(capability.ReasonCodes, typed.Details.ReasonCodes) ||
		!reflect.DeepEqual(capability.RelatedRefs, typed.Details.RelatedRefs) ||
		!reflect.DeepEqual(capability.Remedies, typed.Details.Remedies) {
		t.Fatalf("capability/write precondition drift: capability=%#v write=%#v", capability, typed.Details)
	}
	if len(typed.Details.RelatedRefs) != 1 || typed.Details.RelatedRefs[0] != objective.Ref() {
		t.Fatalf("expected only the authorized owner Objective reference, got %#v", typed.Details.RelatedRefs)
	}
}

func TestCreateWorkItemCapabilitiesNameBacklogAndTodoTargets(t *testing.T) {
	fixture := newTaskActivationFixture(t, true)
	objective := createTaskActivationObjective(t, fixture, "Planned owner", nil, false)
	view, err := fixture.service.GetEntityWorkspaceView(fixture.ctx, fixture.scope, application.WorkspaceViewQuery{
		Subject: application.WorkspaceSubject{Kind: "entity", Ref: objective.Ref()},
	})
	if err != nil {
		t.Fatal(err)
	}
	backlog := capabilityForTarget(t, view, "CreateWorkItem", domain.WorkItemLifecycleBacklog)
	if backlog.Availability != "available" {
		t.Fatalf("Backlog creation should be available on a Planned Objective: %#v", backlog)
	}
	todo := capabilityForTarget(t, view, "CreateWorkItem", domain.WorkItemLifecycleTodo)
	if todo.Availability != "blocked" || !reflect.DeepEqual(todo.ReasonCodes, []string{string(domain.ReadinessReasonObjectiveNotStarted)}) {
		t.Fatalf("To Do creation should expose the owner gate: %#v", todo)
	}
	if len(todo.RelatedRefs) != 1 || todo.RelatedRefs[0] != objective.Ref() {
		t.Fatalf("To Do target did not include the in-scope Objective: %#v", todo.RelatedRefs)
	}

	_, err = fixture.service.CreateWorkItem(fixture.ctx, fixture.command, application.CreateWorkItemCommand{
		Scope: fixture.scope, ObjectiveID: &objective.ID, Title: "Direct To Do Task",
		Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleTodo,
		InitialCriteria: []application.InitialCriterion{{Title: "Completed work", Required: true, VerificationMode: domain.VerificationModeAttestation}},
	})
	var typed *domain.Error
	if !errors.As(err, &typed) || typed.Details == nil || typed.Details.ReasonCode != string(domain.ReadinessReasonObjectiveNotStarted) {
		t.Fatalf("direct To Do write did not match its known parent gate: %v", err)
	}
	if !reflect.DeepEqual(todo.ReasonCodes, typed.Details.ReasonCodes) || !reflect.DeepEqual(todo.RelatedRefs, typed.Details.RelatedRefs) {
		t.Fatalf("To Do capability/write precondition drift: capability=%#v write=%#v", todo, typed.Details)
	}

	activeOwner := createTaskActivationObjective(t, fixture, "In-progress owner", nil, true)
	activeView, err := fixture.service.GetEntityWorkspaceView(fixture.ctx, fixture.scope, application.WorkspaceViewQuery{
		Subject: application.WorkspaceSubject{Kind: "entity", Ref: activeOwner.Ref()},
	})
	if err != nil {
		t.Fatal(err)
	}
	unknownTodo := capabilityForTarget(t, activeView, "CreateWorkItem", domain.WorkItemLifecycleTodo)
	if activeOwner.Lifecycle != domain.ObjectiveLifecycleInProgress {
		t.Fatalf("fixture owner lifecycle = %s, want in_progress", activeOwner.Lifecycle)
	}
	if unknownTodo.Availability != "unknown" || !reflect.DeepEqual(unknownTodo.ReasonCodes, []string{string(domain.ReadinessReasonTargetDefinitionRequired)}) {
		t.Fatalf("To Do target must remain unknown until a Task definition is supplied: %#v", unknownTodo)
	}
}

func TestBacklogOperationalReadUsesActivationEligibilityWithoutMutation(t *testing.T) {
	fixture := newTaskActivationFixture(t, true)
	owner := createTaskActivationObjective(t, fixture, "Active owner", nil, true)
	backlog := createTaskActivationTask(t, fixture, owner, domain.WorkItemLifecycleBacklog, true)
	events := fixture.store.SnapshotDomainEvents(fixture.scope)

	read, err := fixture.service.GetWorkItemOperationalState(fixture.ctx, fixture.scope, backlog.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(read.State.ReadinessReasons) != 0 || read.State.DisplayState != domain.WorkItemDisplayStateBacklog {
		t.Fatalf("eligible Backlog projection = %#v", read.State)
	}

	owner = forceTaskActivationObjectiveLifecycle(t, fixture, owner, domain.ObjectiveLifecyclePlanned)
	read, err = fixture.service.GetWorkItemOperationalState(fixture.ctx, fixture.scope, backlog.ID)
	if err != nil {
		t.Fatal(err)
	}
	want := []domain.ReadinessReason{domain.ReadinessReasonObjectiveNotStarted}
	if !reflect.DeepEqual(read.State.ReadinessReasons, want) || read.State.DisplayState != domain.WorkItemDisplayStateBacklog {
		t.Fatalf("blocked Backlog projection = %#v, want reasons %#v while retaining backlog lifecycle", read.State, want)
	}
	if !reflect.DeepEqual(events, fixture.store.SnapshotDomainEvents(fixture.scope)) {
		t.Fatal("Backlog operational reads mutated the domain")
	}
}

func TestOperationalReadAndAcquisitionShareReadinessSnapshot(t *testing.T) {
	fixture := newTaskActivationFixture(t, true)
	owner := createTaskActivationObjective(t, fixture, "Initially active owner", nil, true)
	task := createTaskActivationTask(t, fixture, owner, domain.WorkItemLifecycleTodo, true)
	owner = forceTaskActivationObjectiveLifecycle(t, fixture, owner, domain.ObjectiveLifecyclePlanned)
	events := fixture.store.SnapshotDomainEvents(fixture.scope)

	read, err := fixture.service.GetWorkItemOperationalState(fixture.ctx, fixture.scope, task.ID)
	if err != nil {
		t.Fatal(err)
	}
	wantReasons := []domain.ReadinessReason{domain.ReadinessReasonObjectiveNotStarted}
	if !reflect.DeepEqual(read.State.ReadinessReasons, wantReasons) || read.State.DisplayState != domain.WorkItemDisplayStateWaitingScope {
		t.Fatalf("operational read = %#v, want owner eligibility reasons %#v", read.State, wantReasons)
	}
	view, err := fixture.service.GetEntityWorkspaceView(fixture.ctx, fixture.scope, application.WorkspaceViewQuery{
		Subject: application.WorkspaceSubject{Kind: "entity", Ref: task.Ref()},
	})
	if err != nil {
		t.Fatal(err)
	}
	var claim *application.WorkspaceCapability
	for i := range view.Capabilities {
		if view.Capabilities[i].ActionID == "ClaimWorkItem" {
			claim = &view.Capabilities[i]
			break
		}
	}
	if claim == nil || claim.Availability != "blocked" || !reflect.DeepEqual(claim.ReasonCodes, []string{string(domain.ReadinessReasonObjectiveNotStarted)}) ||
		len(claim.RelatedRefs) != 1 || claim.RelatedRefs[0] != owner.Ref() {
		t.Fatalf("acquisition capability diverged from operational read: %#v", claim)
	}
	_, err = fixture.service.ClaimWorkItem(fixture.ctx, fixture.command, application.ClaimWorkItemCommand{
		Scope: fixture.scope, WorkItemID: task.ID, ExpectedVersion: task.Version, TTL: domain.DefaultLeaseTTL,
	})
	var typed *domain.Error
	if !errors.As(err, &typed) || typed.Details == nil || !reflect.DeepEqual(typed.Details.ReasonCodes, claim.ReasonCodes) || !reflect.DeepEqual(typed.Details.RelatedRefs, claim.RelatedRefs) {
		t.Fatalf("acquisition write diverged from capability: %v / %#v", err, claim)
	}
	if !reflect.DeepEqual(events, fixture.store.SnapshotDomainEvents(fixture.scope)) {
		t.Fatal("operational read/capability or rejected acquisition mutated the domain")
	}
}

func TestEffectivePermissionsRechecksDynamicReadGrantInsideSnapshot(t *testing.T) {
	authorizer := &revokingTransactionalPermissionReader{revokeAfterExternalRead: true}
	service, err := application.NewAuthorizedService(memory.New(), fixedClock{time.Now()}, &sequenceIDs{}, authorizer)
	if err != nil {
		t.Fatal(err)
	}
	scope := domain.Scope{NamespaceID: id("0199ec00-0000-7000-8000-000000000001"), OutcomeID: id("0199ec00-0000-7000-8000-000000000002")}
	ctx := application.WithIdentity(context.Background(), application.Identity{NamespaceID: scope.NamespaceID, PrincipalID: "operator"})
	if _, err := service.EffectivePermissions(ctx, scope); err == nil {
		t.Fatal("revoked read permission remained visible in the in-transaction permission snapshot")
	}
	if authorizer.inUnitCalls != 1 {
		t.Fatalf("dynamic authorization was not checked transactionally before disclosing permissions: %d checks", authorizer.inUnitCalls)
	}
}
