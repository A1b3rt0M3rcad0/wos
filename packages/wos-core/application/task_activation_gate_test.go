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
)

type taskActivationFixture struct {
	service *application.Service
	store   interface {
		IdempotencyRecordCount() int
		SnapshotDomainEvents(domain.Scope) []domain.DomainEvent
		Begin(context.Context) (ports.UnitOfWork, error)
	}
	ctx     context.Context
	command domain.CommandContext
	scope   domain.Scope
}

func newTaskActivationFixture(t *testing.T, operational bool) taskActivationFixture {
	t.Helper()
	ctx := context.Background()
	service, store := newService(t)
	command := commandContext()
	created, err := service.CreateOutcome(ctx, command, application.CreateOutcomeCommand{
		NamespaceID:  id("0199e200-0000-7000-8000-000000000001"),
		Title:        "Task activation gate",
		DesiredState: "Tasks become available only when their operational scope is ready",
		Priority:     domain.PriorityNormal,
		InitialCriteria: []application.InitialCriterion{{
			Title: "Outcome criterion", Required: true,
			VerificationMode: domain.VerificationModeAttestation,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	active, err := service.ActivateOutcome(ctx, command, application.ActivateOutcomeCommand{
		Scope: created.Value.Scope(), ExpectedVersion: created.Value.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if operational {
		uow, err := store.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		outcome, err := uow.Outcomes().Get(ctx, active.Value.NamespaceID, active.Value.ID)
		if err != nil {
			_ = uow.Rollback()
			t.Fatal(err)
		}
		previousVersion := outcome.Version
		outcome.OperationalProfile = domain.OperationalProfileOperationalV2
		outcome.CompletionPolicy = domain.CompletionPolicyStrictWork
		outcome.Version++
		outcome.UpdatedAt = outcome.UpdatedAt.Add(time.Second)
		if err := uow.Outcomes().Save(ctx, outcome, previousVersion); err != nil {
			_ = uow.Rollback()
			t.Fatal(err)
		}
		if err := uow.Commit(); err != nil {
			t.Fatal(err)
		}
	}
	return taskActivationFixture{service: service, store: store, ctx: ctx, command: command, scope: created.Value.Scope()}
}

func createTaskActivationObjective(t *testing.T, fixture taskActivationFixture, title string, parent *domain.ID, start bool) domain.Objective {
	t.Helper()
	created, err := fixture.service.CreateObjective(fixture.ctx, fixture.command, application.CreateObjectiveCommand{
		Scope: fixture.scope, ParentObjectiveID: parent, Title: title, Priority: domain.PriorityNormal,
		InitialCriteria: []application.InitialCriterion{{
			Title: "Objective criterion", Required: true,
			VerificationMode: domain.VerificationModeAttestation,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !start {
		return created.Value
	}
	started, err := fixture.service.StartObjective(fixture.ctx, fixture.command, application.StartObjectiveCommand{
		Scope: fixture.scope, ObjectiveID: created.Value.ID, ExpectedVersion: created.Value.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	return started.Value
}

func createTaskActivationTask(t *testing.T, fixture taskActivationFixture, objective domain.Objective, lifecycle domain.WorkItemLifecycle, withCriterion bool) domain.WorkItem {
	t.Helper()
	criteria := []application.InitialCriterion(nil)
	if withCriterion {
		criteria = []application.InitialCriterion{{
			Title: "Required Task criterion", Required: true,
			VerificationMode: domain.VerificationModeAttestation,
		}}
	}
	created, err := fixture.service.CreateWorkItem(fixture.ctx, fixture.command, application.CreateWorkItemCommand{
		Scope: fixture.scope, ObjectiveID: &objective.ID, Title: "Prepare work", Priority: domain.PriorityNormal,
		Lifecycle: lifecycle, InitialCriteria: criteria,
	})
	if err != nil {
		t.Fatal(err)
	}
	return created.Value
}

func forceTaskActivationObjectiveLifecycle(t *testing.T, fixture taskActivationFixture, objective domain.Objective, lifecycle domain.ObjectiveLifecycle) domain.Objective {
	t.Helper()
	uow, err := fixture.store.Begin(fixture.ctx)
	if err != nil {
		t.Fatal(err)
	}
	current, err := uow.Objectives().Get(fixture.ctx, fixture.scope, objective.ID)
	if err != nil {
		_ = uow.Rollback()
		t.Fatal(err)
	}
	previousVersion := current.Version
	current.Lifecycle = lifecycle
	current.Version++
	current.UpdatedAt = current.UpdatedAt.Add(time.Second)
	if err := uow.Objectives().Save(fixture.ctx, current, previousVersion); err != nil {
		_ = uow.Rollback()
		t.Fatal(err)
	}
	if err := uow.Commit(); err != nil {
		t.Fatal(err)
	}
	return current
}

func taskActivationSnapshot(t *testing.T, fixture taskActivationFixture) (application.OutcomeState, []domain.DomainEvent, int) {
	t.Helper()
	state, err := fixture.service.GetOutcomeState(fixture.ctx, fixture.scope)
	if err != nil {
		t.Fatal(err)
	}
	return state, fixture.store.SnapshotDomainEvents(fixture.scope), fixture.store.IdempotencyRecordCount()
}

func assertTaskActivationNoWrites(t *testing.T, fixture taskActivationFixture, state application.OutcomeState, events []domain.DomainEvent, receipts int) {
	t.Helper()
	after, err := fixture.service.GetOutcomeState(fixture.ctx, fixture.scope)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(state, after) || !reflect.DeepEqual(events, fixture.store.SnapshotDomainEvents(fixture.scope)) || receipts != fixture.store.IdempotencyRecordCount() {
		t.Fatal("rejected Task activation changed state, revision, events, or idempotency receipts")
	}
}

func taskActivationReason(t *testing.T, err error) string {
	t.Helper()
	var typed *domain.Error
	if !errors.As(err, &typed) || typed.Details == nil {
		t.Fatalf("error has no structured reason: %v", err)
	}
	return typed.Details.ReasonCode
}

func TestActivateWorkItemOperationalGatesAreAtomicAndProfileAware(t *testing.T) {
	t.Run("operational profile rejects planned owner without writes and explains capability", func(t *testing.T) {
		fixture := newTaskActivationFixture(t, true)
		objective := createTaskActivationObjective(t, fixture, "Planned Objective", nil, false)
		task := createTaskActivationTask(t, fixture, objective, domain.WorkItemLifecycleBacklog, true)

		before, events, receipts := taskActivationSnapshot(t, fixture)
		view, err := fixture.service.GetEntityWorkspaceView(fixture.ctx, fixture.scope, application.WorkspaceViewQuery{
			Subject: application.WorkspaceSubject{Kind: "entity", Ref: task.Ref()},
		})
		if err != nil {
			t.Fatal(err)
		}
		var activation *application.WorkspaceCapability
		for i := range view.Capabilities {
			if view.Capabilities[i].ActionID == "ActivateWorkItem" {
				activation = &view.Capabilities[i]
				break
			}
		}
		if activation == nil || activation.Availability != "blocked" || len(activation.ReasonCodes) != 1 || activation.ReasonCodes[0] != string(domain.ReadinessReasonObjectiveNotStarted) || len(activation.Remedies) == 0 {
			t.Fatalf("activation capability does not explain the actual gate: %#v", activation)
		}
		assertTaskActivationNoWrites(t, fixture, before, events, receipts)

		_, err = fixture.service.ActivateWorkItem(fixture.ctx, fixture.command, application.ActivateWorkItemCommand{
			Scope: fixture.scope, WorkItemID: task.ID, ExpectedVersion: task.Version,
		})
		if reason := taskActivationReason(t, err); reason != string(domain.ReadinessReasonObjectiveNotStarted) {
			t.Fatalf("activation reason = %q, want Objective-not-started", reason)
		}
		assertTaskActivationNoWrites(t, fixture, before, events, receipts)
	})

	t.Run("operational profile rejects incomplete definition without writes", func(t *testing.T) {
		fixture := newTaskActivationFixture(t, true)
		objective := createTaskActivationObjective(t, fixture, "In-progress Objective", nil, true)
		task := createTaskActivationTask(t, fixture, objective, domain.WorkItemLifecycleBacklog, false)
		before, events, receipts := taskActivationSnapshot(t, fixture)

		_, err := fixture.service.ActivateWorkItem(fixture.ctx, fixture.command, application.ActivateWorkItemCommand{
			Scope: fixture.scope, WorkItemID: task.ID, ExpectedVersion: task.Version,
		})
		if reason := taskActivationReason(t, err); reason != string(domain.ReadinessReasonTaskDefinitionIncomplete) {
			t.Fatalf("activation reason = %q, want incomplete definition", reason)
		}
		assertTaskActivationNoWrites(t, fixture, before, events, receipts)
	})

	t.Run("operational profile rejects a Planned ancestor without writes", func(t *testing.T) {
		fixture := newTaskActivationFixture(t, true)
		root := createTaskActivationObjective(t, fixture, "Planned ancestor", nil, false)
		child := createTaskActivationObjective(t, fixture, "In-progress child", &root.ID, false)
		child = forceTaskActivationObjectiveLifecycle(t, fixture, child, domain.ObjectiveLifecycleInProgress)
		task := createTaskActivationTask(t, fixture, child, domain.WorkItemLifecycleBacklog, true)
		before, events, receipts := taskActivationSnapshot(t, fixture)

		_, err := fixture.service.ActivateWorkItem(fixture.ctx, fixture.command, application.ActivateWorkItemCommand{
			Scope: fixture.scope, WorkItemID: task.ID, ExpectedVersion: task.Version,
		})
		if reason := taskActivationReason(t, err); reason != string(domain.ReadinessReasonAncestorObjectiveNotStarted) {
			t.Fatalf("activation reason = %q, want Planned-ancestor", reason)
		}
		assertTaskActivationNoWrites(t, fixture, before, events, receipts)
	})

	t.Run("valid operational activation succeeds", func(t *testing.T) {
		fixture := newTaskActivationFixture(t, true)
		objective := createTaskActivationObjective(t, fixture, "In-progress Objective", nil, true)
		task := createTaskActivationTask(t, fixture, objective, domain.WorkItemLifecycleBacklog, true)
		before, events, _ := taskActivationSnapshot(t, fixture)

		activated, err := fixture.service.ActivateWorkItem(fixture.ctx, fixture.command, application.ActivateWorkItemCommand{
			Scope: fixture.scope, WorkItemID: task.ID, ExpectedVersion: task.Version,
		})
		if err != nil {
			t.Fatal(err)
		}
		if activated.Value.Lifecycle != domain.WorkItemLifecycleTodo || activated.Value.Version != task.Version+1 || activated.OutcomeRevision != before.OutcomeRevision+1 {
			t.Fatalf("activation result = %#v", activated)
		}
		if got := len(fixture.store.SnapshotDomainEvents(fixture.scope)); got != len(events)+1 {
			t.Fatalf("event count = %d, want %d", got, len(events)+1)
		}
	})

	t.Run("compatibility profile retains historical activation", func(t *testing.T) {
		fixture := newTaskActivationFixture(t, false)
		objective := createTaskActivationObjective(t, fixture, "Planned Objective", nil, false)
		task := createTaskActivationTask(t, fixture, objective, domain.WorkItemLifecycleBacklog, false)
		activated, err := fixture.service.ActivateWorkItem(fixture.ctx, fixture.command, application.ActivateWorkItemCommand{
			Scope: fixture.scope, WorkItemID: task.ID, ExpectedVersion: task.Version,
		})
		if err != nil || activated.Value.Lifecycle != domain.WorkItemLifecycleTodo {
			t.Fatalf("compatibility activation = %#v, err=%v", activated, err)
		}
	})
}

func TestCreateWorkItemDirectToDoGatesAreAtomicAndProfileAware(t *testing.T) {
	t.Run("backlog remains available under planned Objective", func(t *testing.T) {
		fixture := newTaskActivationFixture(t, true)
		objective := createTaskActivationObjective(t, fixture, "Planned Objective", nil, false)
		task := createTaskActivationTask(t, fixture, objective, domain.WorkItemLifecycleBacklog, false)
		if task.Lifecycle != domain.WorkItemLifecycleBacklog || len(task.Criteria.Items) != 0 {
			t.Fatalf("Backlog creation changed: %#v", task)
		}
	})

	t.Run("direct To Do under Planned owner rolls back initial criteria", func(t *testing.T) {
		fixture := newTaskActivationFixture(t, true)
		objective := createTaskActivationObjective(t, fixture, "Planned Objective", nil, false)
		before, events, receipts := taskActivationSnapshot(t, fixture)
		command := fixture.command
		command.IdempotencyKey = "e09-rejected-direct-todo"
		_, err := fixture.service.CreateWorkItem(fixture.ctx, command, application.CreateWorkItemCommand{
			Scope: fixture.scope, ObjectiveID: &objective.ID, Title: "Must remain in Backlog", Priority: domain.PriorityNormal,
			Lifecycle: domain.WorkItemLifecycleTodo,
			InitialCriteria: []application.InitialCriterion{{
				Title: "Created atomically with Task", Required: true,
				VerificationMode: domain.VerificationModeAttestation,
			}},
		})
		if reason := taskActivationReason(t, err); reason != string(domain.ReadinessReasonObjectiveNotStarted) {
			t.Fatalf("direct To Do reason = %q, want Objective-not-started", reason)
		}
		assertTaskActivationNoWrites(t, fixture, before, events, receipts)
	})

	t.Run("direct To Do requires an active required criterion", func(t *testing.T) {
		fixture := newTaskActivationFixture(t, true)
		objective := createTaskActivationObjective(t, fixture, "In-progress Objective", nil, true)
		before, events, receipts := taskActivationSnapshot(t, fixture)
		command := fixture.command
		command.IdempotencyKey = "e09-rejected-incomplete-todo"
		_, err := fixture.service.CreateWorkItem(fixture.ctx, command, application.CreateWorkItemCommand{
			Scope: fixture.scope, ObjectiveID: &objective.ID, Title: "Missing required criterion", Priority: domain.PriorityNormal,
			Lifecycle: domain.WorkItemLifecycleTodo,
		})
		if reason := taskActivationReason(t, err); reason != string(domain.ReadinessReasonTaskDefinitionIncomplete) {
			t.Fatalf("direct To Do reason = %q, want incomplete definition", reason)
		}
		assertTaskActivationNoWrites(t, fixture, before, events, receipts)
	})

	t.Run("valid direct To Do validates assembled initial criterion", func(t *testing.T) {
		fixture := newTaskActivationFixture(t, true)
		objective := createTaskActivationObjective(t, fixture, "In-progress Objective", nil, true)
		created, err := fixture.service.CreateWorkItem(fixture.ctx, fixture.command, application.CreateWorkItemCommand{
			Scope: fixture.scope, ObjectiveID: &objective.ID, Title: "Prepared Task", Priority: domain.PriorityNormal,
			Lifecycle: domain.WorkItemLifecycleTodo,
			InitialCriteria: []application.InitialCriterion{{
				Title: "Created atomically with Task", Required: true,
				VerificationMode: domain.VerificationModeAttestation,
			}},
		})
		if err != nil {
			t.Fatal(err)
		}
		if created.Value.Lifecycle != domain.WorkItemLifecycleTodo || len(created.Value.Criteria.Items) != 1 || created.Value.CurrentLease != nil || created.Value.CurrentContractID != nil {
			t.Fatalf("direct To Do created authority or lost definition: %#v", created.Value)
		}
	})

	t.Run("compatibility profile retains direct To Do behavior", func(t *testing.T) {
		fixture := newTaskActivationFixture(t, false)
		objective := createTaskActivationObjective(t, fixture, "Planned Objective", nil, false)
		created, err := fixture.service.CreateWorkItem(fixture.ctx, fixture.command, application.CreateWorkItemCommand{
			Scope: fixture.scope, ObjectiveID: &objective.ID, Title: "Historical direct To Do", Priority: domain.PriorityNormal,
			Lifecycle: domain.WorkItemLifecycleTodo,
		})
		if err != nil || created.Value.Lifecycle != domain.WorkItemLifecycleTodo {
			t.Fatalf("compatibility direct To Do = %#v, err=%v", created, err)
		}
	})
}

func TestTaskActivationAndDirectToDoRejectArchivedOrTerminalOutcome(t *testing.T) {
	for _, test := range []struct {
		name       string
		archive    bool
		wantReason string
	}{
		{name: "archived", archive: true, wantReason: string(domain.ReadinessReasonOutcomeArchived)},
		{name: "abandoned", wantReason: string(domain.ReadinessReasonOutcomeNotActive)},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := newTaskActivationFixture(t, true)
			objective := createTaskActivationObjective(t, fixture, "In-progress Objective", nil, true)
			task := createTaskActivationTask(t, fixture, objective, domain.WorkItemLifecycleBacklog, true)
			if test.archive {
				state, err := fixture.service.GetOutcomeState(fixture.ctx, fixture.scope)
				if err != nil {
					t.Fatal(err)
				}
				_, err = fixture.service.ArchiveOutcome(fixture.ctx, fixture.command, application.ArchiveOutcomeCommand{
					Scope: fixture.scope, ExpectedVersion: state.Outcome.Version,
				})
				if err != nil {
					t.Fatal(err)
				}
			} else {
				state, err := fixture.service.GetOutcomeState(fixture.ctx, fixture.scope)
				if err != nil {
					t.Fatal(err)
				}
				_, err = fixture.service.AbandonOutcome(fixture.ctx, fixture.command, application.AbandonOutcomeCommand{
					Scope: fixture.scope, ExpectedVersion: state.Outcome.Version, Reason: "Close scope for activation gate test",
				})
				if err != nil {
					t.Fatal(err)
				}
			}

			before, events, receipts := taskActivationSnapshot(t, fixture)
			_, err := fixture.service.ActivateWorkItem(fixture.ctx, fixture.command, application.ActivateWorkItemCommand{
				Scope: fixture.scope, WorkItemID: task.ID, ExpectedVersion: task.Version,
			})
			if reason := taskActivationReason(t, err); reason != test.wantReason {
				t.Fatalf("activation reason = %q, want %q", reason, test.wantReason)
			}
			assertTaskActivationNoWrites(t, fixture, before, events, receipts)

			command := fixture.command
			command.IdempotencyKey = "e09-closed-outcome-direct-todo"
			_, err = fixture.service.CreateWorkItem(fixture.ctx, command, application.CreateWorkItemCommand{
				Scope: fixture.scope, ObjectiveID: &objective.ID, Title: "Must remain unavailable", Priority: domain.PriorityNormal,
				Lifecycle: domain.WorkItemLifecycleTodo,
				InitialCriteria: []application.InitialCriterion{{
					Title: "Created atomically with Task", Required: true,
					VerificationMode: domain.VerificationModeAttestation,
				}},
			})
			if err == nil {
				t.Fatal("direct To Do creation succeeded under archived/terminal Outcome")
			}
			assertTaskActivationNoWrites(t, fixture, before, events, receipts)
		})
	}
}
