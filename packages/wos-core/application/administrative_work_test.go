package application_test

import (
	"context"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

type permissionAuthorizer struct {
	allowed map[ports.Permission]bool
}

func (a permissionAuthorizer) Authorize(_ context.Context, request ports.AuthorizationRequest) error {
	if err := request.Validate(); err != nil {
		return err
	}
	if !a.allowed[request.Permission] {
		return domain.NewError(domain.ErrorCodeForbidden, "permission denied")
	}
	return nil
}

func TestAdministrativeWorkOverridesRequireExplicitAuthorization(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	now := time.Date(2026, 10, 3, 2, 0, 0, 0, time.UTC)
	ids := &sequenceIDs{next: 3000}
	defaultService, err := application.NewService(store, fixedClock{now: now}, ids)
	if err != nil {
		t.Fatal(err)
	}
	cc := commandContext()
	namespaceID := id("0199f240-0000-7000-8000-000000000001")

	created, err := defaultService.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Administrative override",
		DesiredState: "leased work can be recovered only by authorized override",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value
	if _, err := defaultService.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner:            outcome.Ref(),
		ExpectedVersion:  outcome.Version,
		Title:            "verified",
		Required:         true,
		VerificationMode: domain.VerificationModeAttestation,
	}); err != nil {
		t.Fatal(err)
	}
	outcome.Version++
	activated, err := defaultService.ActivateOutcome(ctx, cc, application.ActivateOutcomeCommand{
		Scope:           outcome.Scope(),
		ExpectedVersion: outcome.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome = activated.Value

	createdWork, err := defaultService.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{
		Scope:     outcome.Scope(),
		Title:     "leased work",
		Priority:  domain.PriorityNormal,
		Lifecycle: domain.WorkItemLifecycleTodo,
	})
	if err != nil {
		t.Fatal(err)
	}
	work := createdWork.Value
	claimed, err := defaultService.ClaimWorkItem(ctx, cc, application.ClaimWorkItemCommand{
		Scope:           outcome.Scope(),
		WorkItemID:      work.ID,
		ExpectedVersion: work.Version,
		TTL:             domain.DefaultLeaseTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	work = claimed.Value

	if _, err := defaultService.CancelWorkItem(ctx, cc, application.CancelWorkItemCommand{
		Scope:           outcome.Scope(),
		WorkItemID:      work.ID,
		ExpectedVersion: work.Version,
		Reason:          "ordinary caller tries to cancel leased work",
	}); err == nil {
		t.Fatal("ordinary cancellation unexpectedly cancelled in-progress leased work")
	}

	if _, err := defaultService.AdministrativeCancelWorkItem(ctx, cc, application.AdministrativeCancelWorkItemCommand{
		Scope:           outcome.Scope(),
		WorkItemID:      work.ID,
		ExpectedVersion: work.Version,
		Reason:          "operator intervention",
	}); err == nil {
		t.Fatal("default service unexpectedly authorized administrative cancellation")
	} else if code, ok := domain.ErrorCodeOf(err); !ok || code != domain.ErrorCodeForbidden {
		t.Fatalf("administrative cancellation error = %v, want forbidden", err)
	}

	authorizedService, err := application.NewServiceWithAuthorizer(
		store,
		fixedClock{now: now.Add(time.Minute)},
		&sequenceIDs{next: 4000},
		permissionAuthorizer{allowed: map[ports.Permission]bool{
			ports.PermissionWorkAdminCancel: true,
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	cancelled, err := authorizedService.AdministrativeCancelWorkItem(ctx, cc, application.AdministrativeCancelWorkItemCommand{
		Scope:           outcome.Scope(),
		WorkItemID:      work.ID,
		ExpectedVersion: work.Version,
		Reason:          "operator confirms cancellation",
	})
	if err != nil {
		t.Fatal(err)
	}
	if cancelled.Value.Lifecycle != domain.WorkItemLifecycleCancelled || cancelled.Value.CurrentLease != nil {
		t.Fatalf("administrative cancellation state = %#v", cancelled.Value)
	}

	events := store.SnapshotDomainEvents(outcome.Scope())
	found := false
	for _, event := range events {
		if event.EventType == "work_item.admin_cancelled" {
			found = true
			if event.PrincipalID != cc.PrincipalID {
				t.Fatalf("admin event principal = %q, want %q", event.PrincipalID, cc.PrincipalID)
			}
		}
	}
	if !found {
		t.Fatal("administrative cancellation event was not recorded")
	}
}

func TestAdministrativeCompletionOverridesLeaseButNotCompletionGates(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	now := time.Date(2026, 10, 3, 3, 0, 0, 0, time.UTC)
	cc := commandContext()
	service, err := application.NewServiceWithAuthorizer(
		store,
		fixedClock{now: now},
		&sequenceIDs{next: 5000},
		permissionAuthorizer{allowed: map[ports.Permission]bool{
			ports.PermissionWorkAdminComplete: true,
		}},
	)
	if err != nil {
		t.Fatal(err)
	}
	namespaceID := id("0199f240-0000-7000-8000-000000000011")
	created, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  namespaceID,
		Title:        "Administrative completion",
		DesiredState: "override ownership without bypassing quality gates",
		Priority:     domain.PriorityNormal,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome := created.Value
	if _, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{
		Owner:            outcome.Ref(),
		ExpectedVersion:  outcome.Version,
		Title:            "verified",
		Required:         true,
		VerificationMode: domain.VerificationModeAttestation,
	}); err != nil {
		t.Fatal(err)
	}
	outcome.Version++
	activated, err := service.ActivateOutcome(ctx, cc, application.ActivateOutcomeCommand{
		Scope:           outcome.Scope(),
		ExpectedVersion: outcome.Version,
	})
	if err != nil {
		t.Fatal(err)
	}
	outcome = activated.Value

	createdWork, err := service.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{
		Scope:     outcome.Scope(),
		Title:     "externally finished leased work",
		Priority:  domain.PriorityNormal,
		Lifecycle: domain.WorkItemLifecycleTodo,
	})
	if err != nil {
		t.Fatal(err)
	}
	work := createdWork.Value
	claimed, err := service.ClaimWorkItem(ctx, cc, application.ClaimWorkItemCommand{
		Scope:           outcome.Scope(),
		WorkItemID:      work.ID,
		ExpectedVersion: work.Version,
		TTL:             domain.MinLeaseTTL,
	})
	if err != nil {
		t.Fatal(err)
	}
	work = claimed.Value
	oldClaim := *work.CurrentLease

	completed, err := service.AdministrativeCompleteWorkItem(ctx, cc, application.AdministrativeCompleteWorkItemCommand{
		Scope:           outcome.Scope(),
		WorkItemID:      work.ID,
		ExpectedVersion: work.Version,
		ResultSummary:   "operator verified external completion",
		Reason:          "worker became unavailable after producing the result",
	})
	if err != nil {
		t.Fatal(err)
	}
	if completed.Value.Lifecycle != domain.WorkItemLifecycleDone || completed.Value.CurrentLease != nil {
		t.Fatalf("administrative completion state = %#v", completed.Value)
	}

	if _, err := service.CompleteWorkItem(ctx, cc, application.CompleteWorkItemCommand{
		Scope:           outcome.Scope(),
		WorkItemID:      work.ID,
		ExpectedVersion: completed.Value.Version,
		ClaimID:         oldClaim.ClaimID,
		FencingToken:    oldClaim.FencingToken,
		ResultSummary:   "stale",
		Reason:          "stale claimant",
	}); err == nil {
		t.Fatal("stale claimant unexpectedly completed after administrative override")
	}

	events := store.SnapshotDomainEvents(outcome.Scope())
	found := false
	for _, event := range events {
		if event.EventType == "work_item.admin_completed" {
			found = true
		}
	}
	if !found {
		t.Fatal("administrative completion event was not recorded")
	}
}
