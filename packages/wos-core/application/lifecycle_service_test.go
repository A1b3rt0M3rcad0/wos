package application_test

import (
	"context"
	"testing"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

func TestOutcomeLifecycleCommandsPreserveArchiveAndConclusionSemantics(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	cc := commandContext()
	namespaceID := id("0199e200-0000-7000-8000-000000000001")

	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Lifecycle",
		DesiredState: "State",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value

	owner := domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "woobe", ID: "agent-owner"}
	owned, err := service.SetOutcomeOwners(ctx, cc, application.SetOutcomeOwnersCommand{
		Scope: outcome.Scope(), ExpectedVersion: outcome.Version, OwnerRefs: []domain.ActorRef{owner},
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome = owned.Value
	if len(outcome.OwnerRefs) != 1 || outcome.OwnerRefs[0] != owner {
		t.Fatalf("owners = %#v", outcome.OwnerRefs)
	}

	criterion, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner: outcome.Ref(), ExpectedVersion: outcome.Version,
		Title: "Required", Required: true, VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome.Version++

	activated, err := service.ActivateOutcome(ctx, cc, application.ActivateOutcomeCommand{
		Scope: outcome.Scope(), ExpectedVersion: outcome.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome = activated.Value

	archived, err := service.ArchiveOutcome(ctx, cc, application.ArchiveOutcomeCommand{
		Scope: outcome.Scope(), ExpectedVersion: outcome.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !archived.Value.IsArchived() || archived.Value.Lifecycle != domain.OutcomeLifecycleActive {
		t.Fatalf("archive changed semantic lifecycle: archived=%v lifecycle=%q", archived.Value.IsArchived(), archived.Value.Lifecycle)
	}
	outcome = archived.Value

	unarchived, err := service.UnarchiveOutcome(ctx, cc, application.UnarchiveOutcomeCommand{
		Scope: outcome.Scope(), ExpectedVersion: outcome.Version, Reason: "resume coordination",
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome = unarchived.Value

	failed, err := service.FailOutcome(ctx, cc, application.FailOutcomeCommand{
		Scope: outcome.Scope(), ExpectedVersion: outcome.Version, Reason: "verification failed",
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome = failed.Value

	reopened, err := service.ReopenOutcome(ctx, cc, application.ReopenOutcomeCommand{
		Scope: outcome.Scope(), ExpectedVersion: outcome.Version, Reason: "retry",
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome = reopened.Value
	if outcome.Lifecycle != domain.OutcomeLifecycleActive || len(outcome.ConclusionHistory) != 1 {
		t.Fatalf("reopen state lifecycle=%q history=%d", outcome.Lifecycle, len(outcome.ConclusionHistory))
	}

	abandoned, err := service.AbandonOutcome(ctx, cc, application.AbandonOutcomeCommand{
		Scope: outcome.Scope(), ExpectedVersion: outcome.Version, Reason: "stop pursuing result",
	})
	if err != nil {
		t.Fatal(err)
	}
	if abandoned.Value.Lifecycle != domain.OutcomeLifecycleAbandoned {
		t.Fatalf("lifecycle = %q, want abandoned", abandoned.Value.Lifecycle)
	}

	_ = criterion
}

func TestObjectiveAndWorkItemRemainingLifecycleCommands(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	cc := commandContext()
	namespaceID := id("0199e200-0000-7000-8000-000000000001")

	createdOutcome, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Coordinate",
		DesiredState: "State",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := createdOutcome.Value

	criterion, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner: outcome.Ref(), ExpectedVersion: outcome.Version,
		Title: "Required", Required: true, VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = criterion
	outcome.Version++

	activated, err := service.ActivateOutcome(ctx, cc, application.ActivateOutcomeCommand{
		Scope: outcome.Scope(), ExpectedVersion: outcome.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome = activated.Value

	createdObjective, err := service.CreateObjective(ctx, cc, application.CreateObjectiveCommand{
		Scope: outcome.Scope(), Title: "Objective", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	objective := createdObjective.Value

	owner := domain.ActorRef{Kind: domain.ActorKindHuman, Provider: "local", ID: "owner-2"}
	ownedObjective, err := service.SetObjectiveOwners(ctx, cc, application.SetObjectiveOwnersCommand{
		Scope: outcome.Scope(), ObjectiveID: objective.ID, ExpectedVersion: objective.Version,
		OwnerRefs: []domain.ActorRef{owner},
	})
	if err != nil {
		t.Fatal(err)
	}
	objective = ownedObjective.Value

	cancelledObjective, err := service.CancelObjective(ctx, cc, application.CancelObjectiveCommand{
		Scope: outcome.Scope(), ObjectiveID: objective.ID, ExpectedVersion: objective.Version, Reason: "defer objective",
	})
	if err != nil {
		t.Fatal(err)
	}
	objective = cancelledObjective.Value

	reopenedObjective, err := service.ReopenObjective(ctx, cc, application.ReopenObjectiveCommand{
		Scope: outcome.Scope(), ObjectiveID: objective.ID, ExpectedVersion: objective.Version, Reason: "needed again",
	})
	if err != nil {
		t.Fatal(err)
	}
	if reopenedObjective.Value.Lifecycle != domain.ObjectiveLifecyclePlanned {
		t.Fatalf("objective lifecycle = %q, want planned", reopenedObjective.Value.Lifecycle)
	}

	createdWork, err := service.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{
		Scope: outcome.Scope(), Title: "Work", Priority: domain.PriorityNormal,
		Lifecycle: domain.WorkItemLifecycleTodo,
	})
	if err != nil {
		t.Fatal(err)
	}
	work := createdWork.Value

	assignee := domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "woobe", ID: "agent-assignee"}
	assigned, err := service.SetWorkItemAssignees(ctx, cc, application.SetWorkItemAssigneesCommand{
		Scope: outcome.Scope(), WorkItemID: work.ID, ExpectedVersion: work.Version,
		AssigneeRefs: []domain.ActorRef{assignee},
	})
	if err != nil {
		t.Fatal(err)
	}
	work = assigned.Value

	deferred, err := service.DeferWorkItem(ctx, cc, application.DeferWorkItemCommand{
		Scope: outcome.Scope(), WorkItemID: work.ID, ExpectedVersion: work.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	work = deferred.Value

	activatedWork, err := service.ActivateWorkItem(ctx, cc, application.ActivateWorkItemCommand{
		Scope: outcome.Scope(), WorkItemID: work.ID, ExpectedVersion: work.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	work = activatedWork.Value

	claimed, err := service.ClaimWorkItem(ctx, cc, application.ClaimWorkItemCommand{
		Scope: outcome.Scope(), WorkItemID: work.ID, ExpectedVersion: work.Version,
		TTL: domain.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	work = claimed.Value
	if work.CurrentLease == nil {
		t.Fatal("claim did not create lease")
	}

	released, err := service.ReleaseWorkItem(ctx, cc, application.ReleaseWorkItemCommand{
		Scope: outcome.Scope(), WorkItemID: work.ID, ExpectedVersion: work.Version,
		ClaimID: work.CurrentLease.ClaimID, FencingToken: work.CurrentLease.FencingToken,
	})
	if err != nil {
		t.Fatal(err)
	}
	work = released.Value
	if work.Lifecycle != domain.WorkItemLifecycleTodo || work.CurrentLease != nil {
		t.Fatalf("release state lifecycle=%q lease=%#v", work.Lifecycle, work.CurrentLease)
	}

	cancelledWork, err := service.CancelWorkItem(ctx, cc, application.CancelWorkItemCommand{
		Scope: outcome.Scope(), WorkItemID: work.ID, ExpectedVersion: work.Version, Reason: "cancel work",
	})
	if err != nil {
		t.Fatal(err)
	}
	work = cancelledWork.Value

	reopenedWork, err := service.ReopenWorkItem(ctx, cc, application.ReopenWorkItemCommand{
		Scope: outcome.Scope(), WorkItemID: work.ID, ExpectedVersion: work.Version, Reason: "work required again",
	})
	if err != nil {
		t.Fatal(err)
	}
	if reopenedWork.Value.Lifecycle != domain.WorkItemLifecycleTodo {
		t.Fatalf("work lifecycle = %q, want todo", reopenedWork.Value.Lifecycle)
	}
	if len(reopenedWork.Value.AssigneeRefs) != 1 || reopenedWork.Value.AssigneeRefs[0] != assignee {
		t.Fatalf("reopen unexpectedly changed assignees: %#v", reopenedWork.Value.AssigneeRefs)
	}
}

func TestRetireCriterionUsesAggregateVersioning(t *testing.T) {
	ctx := context.Background()
	service, _ := newService(t)
	cc := commandContext()
	namespaceID := id("0199e200-0000-7000-8000-000000000001")

	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID: namespaceID, Title: "Retire criterion", DesiredState: "State", Priority: domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value

	added, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner: outcome.Ref(), ExpectedVersion: outcome.Version,
		Title: "Temporary criterion", Required: true, VerificationMode: domain.VerificationModeAttestation,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome.Version++

	retired, err := service.RetireCriterion(ctx, cc, application.RetireCriterionCommand{
		Owner: outcome.Ref(), CriterionID: added.Value.ID, ExpectedVersion: outcome.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	if retired.Value.Status != domain.CriterionStatusRetired {
		t.Fatalf("criterion status = %q, want retired", retired.Value.Status)
	}
}
