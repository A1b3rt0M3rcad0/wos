package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/memory"
)

func initialProof(title string) application.InitialCriterion {
	return application.InitialCriterion{Title: title, Required: true, VerificationMode: domain.VerificationModeEvidenceReview}
}

func TestPlanningCreationInitialCriteriaPreserveExplicitLifecycleAndAssessments(t *testing.T) {
	ctx := context.Background()
	service, store := newWave09Service(t)
	cc := commandContext()
	cc.IdempotencyKey = "planning-outcome"
	cmd := application.CreateOutcomeCommand{NamespaceID: id("0199e200-0000-7000-8000-000000000001"), Title: "Reviewed installation", DesiredState: "A newcomer can install", Priority: domain.PriorityNormal, InitialCriteria: []application.InitialCriterion{initialProof("Installation instructions verified")}}
	created, err := service.CreateOutcome(ctx, cc, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if created.Value.Lifecycle != domain.OutcomeLifecycleDraft || len(created.Value.Criteria.Items) != 1 || len(created.Value.Criteria.Assessments) != 0 || created.Value.Version != 2 || created.OutcomeRevision != 1 {
		t.Fatalf("initial state=%#v", created)
	}
	replay, err := service.CreateOutcome(ctx, cc, cmd)
	if err != nil || !replay.IdempotentReplay || !sameWorkspaceJSON(created.Value, replay.Value) {
		t.Fatalf("replay=%#v err=%v", replay, err)
	}
	events := store.SnapshotDomainEvents(created.Value.Scope())
	if len(events) != 2 || events[0].EventType != "outcome.created" || events[1].EventType != "outcome.criterion_added" || events[0].OutcomeRevision != events[1].OutcomeRevision {
		t.Fatalf("events=%#v", events)
	}
	var criterionFact struct {
		CriterionID domain.ID `json:"criterion_id"`
	}
	if err := json.Unmarshal(events[1].Payload, &criterionFact); err != nil || criterionFact.CriterionID != created.Value.Criteria.Items[0].ID {
		t.Fatalf("criterion event does not identify its generated definition: %s", events[1].Payload)
	}
	objective, err := service.CreateObjective(ctx, commandContext(), application.CreateObjectiveCommand{Scope: created.Value.Scope(), Title: "Reviewed README", Priority: domain.PriorityNormal, RequiredForOutcome: true, InitialCriteria: []application.InitialCriterion{initialProof("Example tested")}})
	if err != nil {
		t.Fatal(err)
	}
	work, err := service.CreateWorkItem(ctx, commandContext(), application.CreateWorkItemCommand{Scope: created.Value.Scope(), Title: "Review command", Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleTodo, ObjectiveID: &objective.Value.ID, InitialCriteria: []application.InitialCriterion{initialProof("Command works")}})
	if err != nil {
		t.Fatal(err)
	}
	if work.Value.Lifecycle != domain.WorkItemLifecycleTodo || work.Value.CurrentLease != nil || work.Value.CurrentContractID != nil || len(work.Value.Criteria.Assessments) != 0 {
		t.Fatalf("new Task changed execution/approval semantics: %#v", work.Value)
	}
}

func TestCreatePlanItemsResolvesForwardParentsAndReplaysSameIdentities(t *testing.T) {
	ctx := context.Background()
	service, _ := newWave09Service(t)
	outcome := setupActiveOutcome(t, service)
	cc := commandContext()
	cc.IdempotencyKey = "plan-forward-parents"
	cmd := application.CreatePlanItemsCommand{Scope: outcome.Scope(), Items: []application.PlanItemInput{
		{LocalKey: "task", Kind: domain.EntityKindWorkItem, Title: "Write example", ParentLocalKey: "child", InitialCriteria: []application.InitialCriterion{initialProof("Example runs")}},
		{LocalKey: "child", Kind: domain.EntityKindObjective, Title: "Runnable example", ParentLocalKey: "parent", RequiredForOutcome: true},
		{LocalKey: "parent", Kind: domain.EntityKindObjective, Title: "Installation documentation", RequiredForOutcome: true},
		{LocalKey: "direct", Kind: domain.EntityKindWorkItem, Title: "Check terminology", ParentLocalKey: "direct-owner"},
		{LocalKey: "direct-owner", Kind: domain.EntityKindObjective, Title: "Terminology review"},
	}}
	result, err := service.CreatePlanItems(ctx, cc, cmd)
	if err != nil {
		t.Fatal(err)
	}
	if result.Value.Items[0].LocalKey != "task" || len(result.Value.Refs) != 5 {
		t.Fatalf("result=%#v", result.Value)
	}
	work, err := service.GetWorkItem(ctx, outcome.Scope(), result.Value.Refs["task"].ID)
	if err != nil {
		t.Fatal(err)
	}
	if work.Value.ObjectiveID == nil || *work.Value.ObjectiveID != result.Value.Refs["child"].ID || work.Value.Lifecycle != domain.WorkItemLifecycleBacklog || work.Value.CurrentLease != nil {
		t.Fatalf("work=%#v", work.Value)
	}
	child, err := service.GetObjective(ctx, outcome.Scope(), result.Value.Refs["child"].ID)
	if err != nil {
		t.Fatal(err)
	}
	if child.Value.ParentObjectiveID == nil || *child.Value.ParentObjectiveID != result.Value.Refs["parent"].ID || !child.Value.RequiredForOutcome {
		t.Fatalf("child=%#v", child.Value)
	}
	direct, err := service.GetWorkItem(ctx, outcome.Scope(), result.Value.Refs["direct"].ID)
	if err != nil || direct.Value.ObjectiveID == nil || *direct.Value.ObjectiveID != result.Value.Refs["direct-owner"].ID {
		t.Fatalf("direct=%#v err=%v", direct, err)
	}
	replay, err := service.CreatePlanItems(ctx, cc, cmd)
	if err != nil || !replay.IdempotentReplay || !sameWorkspaceJSON(result.Value, replay.Value) {
		t.Fatalf("replay=%#v err=%v", replay, err)
	}
}

func TestCreatePlanItemsBoundariesAndRollback(t *testing.T) {
	ctx := context.Background()
	service, store := newWave09Service(t)
	outcome := setupActiveOutcome(t, service)
	items := make([]application.PlanItemInput, 50)
	items[0] = application.PlanItemInput{LocalKey: "owner", Kind: domain.EntityKindObjective, Title: "Planned work owner", InitialCriteria: []application.InitialCriterion{initialProof("First condition"), initialProof("Second condition")}}
	for i := 1; i < len(items); i++ {
		items[i] = application.PlanItemInput{LocalKey: fmt.Sprintf("task-%d", i), Kind: domain.EntityKindWorkItem, ParentLocalKey: "owner", Title: "Planned work", InitialCriteria: []application.InitialCriterion{initialProof("First condition"), initialProof("Second condition")}}
	}
	result, err := service.CreatePlanItems(ctx, planningIntent(t.Name()), application.CreatePlanItemsCommand{Scope: outcome.Scope(), Items: items})
	if err != nil || len(result.Value.Items) != 50 {
		t.Fatalf("50 items/100 criteria: %v", err)
	}
	beforeEvents := len(store.SnapshotDomainEvents(outcome.Scope()))
	beforeState, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	missing := id("0199ed90-0000-7000-8000-999999999999")
	bad := []application.PlanItemInput{
		{LocalKey: "written-first", Kind: domain.EntityKindWorkItem, Title: "Must roll back", ParentLocalKey: "valid-owner"},
		{LocalKey: "valid-owner", Kind: domain.EntityKindObjective, Title: "Valid owner"},
		{LocalKey: "bad-last", Kind: domain.EntityKindObjective, Title: "Missing parent", ParentObjectiveID: &missing},
	}
	if _, err = service.CreatePlanItems(ctx, planningIntent(t.Name()+"-rollback"), application.CreatePlanItemsCommand{Scope: outcome.Scope(), Items: bad}); err == nil {
		t.Fatal("expected missing parent rejection")
	}
	afterState, _ := service.GetOutcomeState(ctx, outcome.Scope())
	if len(afterState.WorkItems) != len(beforeState.WorkItems) || len(store.SnapshotDomainEvents(outcome.Scope())) != beforeEvents {
		t.Fatal("failed batch persisted partial work/events")
	}
	for name, mutate := range map[string]func([]application.PlanItemInput) []application.PlanItemInput{
		"51 items": func(v []application.PlanItemInput) []application.PlanItemInput {
			return append(v, application.PlanItemInput{LocalKey: "extra", Kind: domain.EntityKindWorkItem, ParentLocalKey: "owner", Title: "Extra"})
		},
		"101 criteria": func(v []application.PlanItemInput) []application.PlanItemInput {
			v[0].InitialCriteria = append(v[0].InitialCriteria, initialProof("Extra"))
			return v
		},
		"bytes": func(v []application.PlanItemInput) []application.PlanItemInput {
			for i := range v {
				v[i].Description = strings.Repeat("a", 6000)
			}
			return v
		},
	} {
		t.Run(name, func(t *testing.T) {
			v := append([]application.PlanItemInput(nil), items...)
			if _, err := service.CreatePlanItems(ctx, planningIntent(t.Name()), application.CreatePlanItemsCommand{Scope: outcome.Scope(), Items: mutate(v)}); err == nil {
				t.Fatal("expected bounded rejection")
			}
		})
	}
}

func TestCreatePlanItemsRejectsInvalidLocalStructure(t *testing.T) {
	ctx := context.Background()
	service, _ := newWave09Service(t)
	outcome := setupActiveOutcome(t, service)
	for name, items := range map[string][]application.PlanItemInput{
		"duplicate":              {{LocalKey: "a", Kind: domain.EntityKindObjective, Title: "A"}, {LocalKey: "a", Kind: domain.EntityKindWorkItem, Title: "B"}},
		"cycle":                  {{LocalKey: "a", Kind: domain.EntityKindObjective, Title: "A", ParentLocalKey: "b"}, {LocalKey: "b", Kind: domain.EntityKindObjective, Title: "B", ParentLocalKey: "a"}},
		"missing":                {{LocalKey: "a", Kind: domain.EntityKindObjective, Title: "A", ParentLocalKey: "absent"}},
		"Task without Objective": {{LocalKey: "a", Kind: domain.EntityKindWorkItem, Title: "A"}},
		"Task parent":            {{LocalKey: "owner", Kind: domain.EntityKindObjective, Title: "Owner"}, {LocalKey: "a", Kind: domain.EntityKindWorkItem, Title: "A", ParentLocalKey: "owner"}, {LocalKey: "b", Kind: domain.EntityKindWorkItem, Title: "B", ParentLocalKey: "a"}},
		"unsupported kind":       {{LocalKey: "a", Kind: domain.EntityKindIssue, Title: "A"}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := service.CreatePlanItems(ctx, planningIntent(t.Name()), application.CreatePlanItemsCommand{Scope: outcome.Scope(), Items: items}); err == nil {
				t.Fatal("invalid local structure accepted")
			}
		})
	}
}

func TestCreatePlanItemsRejectsTaskWithoutObjectiveWithoutWrites(t *testing.T) {
	ctx := context.Background()
	service, store := newWave09Service(t)
	outcome := setupActiveOutcome(t, service)
	before, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	events := len(store.SnapshotDomainEvents(outcome.Scope()))
	_, err = service.CreatePlanItems(ctx, planningIntent(t.Name()), application.CreatePlanItemsCommand{
		Scope: outcome.Scope(), Items: []application.PlanItemInput{{LocalKey: "orphan", Kind: domain.EntityKindWorkItem, Title: "Unplaced Task"}},
	})
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeInvalidArgument {
		t.Fatalf("unplaced Task error=%v", err)
	}
	var typed *domain.Error
	if !errors.As(err, &typed) || typed.Details == nil || typed.Details.ReasonCode != "task_objective_required" {
		t.Fatalf("unplaced Task reason=%+v", typed)
	}
	after, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if after.OutcomeRevision != before.OutcomeRevision || len(after.WorkItems) != 0 || len(store.SnapshotDomainEvents(outcome.Scope())) != events {
		t.Fatal("rejected batch changed revision, work or domain events")
	}
}

type planningPermissionPolicy struct{ calls []ports.Permission }

func (p *planningPermissionPolicy) Authorize(_ context.Context, r ports.AuthorizationRequest) error {
	p.calls = append(p.calls, r.Permission)
	if r.Permission == ports.PermissionTaskCreate {
		return domain.NewError(domain.ErrorCodeForbidden, "work write missing")
	}
	return nil
}
func (p *planningPermissionPolicy) AuthorizeInUnitOfWork(ctx context.Context, _ ports.UnitOfWork, r ports.AuthorizationRequest) error {
	return p.Authorize(ctx, r)
}

type planningAuthorizationRequestRecorder struct{ requests []ports.AuthorizationRequest }

func (p *planningAuthorizationRequestRecorder) Authorize(_ context.Context, request ports.AuthorizationRequest) error {
	p.requests = append(p.requests, request)
	return nil
}

func (p *planningAuthorizationRequestRecorder) AuthorizeInUnitOfWork(ctx context.Context, _ ports.UnitOfWork, request ports.AuthorizationRequest) error {
	return p.Authorize(ctx, request)
}

func TestCreateOutcomeInitialCriteriaUsesResolvedNamespaceForConditionalAuthorization(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	policy := &planningAuthorizationRequestRecorder{}
	service, err := application.NewAuthorizedService(store, fixedClock{now: time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)}, &wave09IDs{next: 1}, policy)
	if err != nil {
		t.Fatal(err)
	}
	cc := planningIntent(t.Name())
	namespace := id("0199e200-0000-7000-8000-000000000001")
	ctx = application.WithIdentity(ctx, application.Identity{NamespaceID: namespace, PrincipalID: cc.PrincipalID, Actor: cc.Actor})
	_, err = service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{
		NamespaceID:  namespace,
		Title:        "Outcome with an initial criterion",
		DesiredState: "The criterion is authorized in the owning namespace",
		Priority:     domain.PriorityNormal,
		InitialCriteria: []application.InitialCriterion{
			initialProof("Initial criterion is persisted"),
		},
	})
	if err != nil {
		t.Fatalf("CreateOutcome with initial criteria: %v", err)
	}
	for _, request := range policy.requests {
		if request.Permission == ports.PermissionCriterionDefine {
			if request.NamespaceID != namespace {
				t.Fatalf("criterion:define namespace = %q, want %q", request.NamespaceID, namespace)
			}
			return
		}
	}
	t.Fatal("CreateOutcome did not request criterion:define for initial criteria")
}

func TestCreatePlanItemsChecksPermissionsForEachKindInsideTransaction(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	policy := &planningPermissionPolicy{}
	service, err := application.NewAuthorizedService(store, fixedClock{now: time.Date(2026, 10, 2, 18, 0, 0, 0, time.UTC)}, &wave09IDs{next: 1}, policy)
	if err != nil {
		t.Fatal(err)
	}
	cc := commandContext()
	ns := id("0199e200-0000-7000-8000-000000000001")
	ctx = application.WithIdentity(ctx, application.Identity{NamespaceID: ns, PrincipalID: cc.PrincipalID, Actor: cc.Actor})
	outcome, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{NamespaceID: ns, Title: "Scoped creation", DesiredState: "Authorization preserved", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	cc.IdempotencyKey = t.Name()
	_, err = service.CreatePlanItems(ctx, cc, application.CreatePlanItemsCommand{Scope: outcome.Value.Scope(), Items: []application.PlanItemInput{{LocalKey: "o", Kind: domain.EntityKindObjective, Title: "Allowed"}, {LocalKey: "t", Kind: domain.EntityKindWorkItem, Title: "Forbidden", ParentLocalKey: "o"}}})
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeForbidden {
		t.Fatalf("err=%v", err)
	}
	if len(store.SnapshotDomainEvents(outcome.Value.Scope())) != 1 {
		t.Fatal("unauthorized mixed batch wrote partial state")
	}
}

func planningIntent(key string) domain.CommandContext {
	cc := commandContext()
	cc.IdempotencyKey = key
	return cc
}

type revocablePlanningPolicy struct{ denied ports.Permission }

func (p *revocablePlanningPolicy) Authorize(_ context.Context, r ports.AuthorizationRequest) error {
	if p.denied != "" && p.denied == r.Permission {
		return domain.NewError(domain.ErrorCodeForbidden, "permission revoked")
	}
	return nil
}
func (p *revocablePlanningPolicy) AuthorizeInUnitOfWork(ctx context.Context, _ ports.UnitOfWork, r ports.AuthorizationRequest) error {
	return p.Authorize(ctx, r)
}

func TestPlanningReplayRechecksConstituentPermissions(t *testing.T) {
	ctx := context.Background()
	store := memory.New()
	policy := &revocablePlanningPolicy{}
	service, err := application.NewAuthorizedService(store, fixedClock{now: time.Date(2026, 10, 19, 12, 0, 0, 0, time.UTC)}, &wave09IDs{next: 1}, policy)
	if err != nil {
		t.Fatal(err)
	}
	cc := commandContext()
	ns := id("0199e200-0000-7000-8000-000000000001")
	ctx = application.WithIdentity(ctx, application.Identity{NamespaceID: ns, PrincipalID: cc.PrincipalID, Actor: cc.Actor})
	owner, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{NamespaceID: ns, Title: "Scoped plan", DesiredState: "Authorization is preserved", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	cmd := application.CreatePlanItemsCommand{Scope: owner.Value.Scope(), Items: []application.PlanItemInput{{LocalKey: "objective", Kind: domain.EntityKindObjective, Title: "Bounded work owner"}, {LocalKey: "task", Kind: domain.EntityKindWorkItem, Title: "Bounded work", ParentLocalKey: "objective"}}}
	cc.IdempotencyKey = t.Name()
	if _, err := service.CreatePlanItems(ctx, cc, cmd); err != nil {
		t.Fatal(err)
	}
	objectiveID := objectiveForWorkFixture(t, service, owner.Value.Scope())
	policy.denied = ports.PermissionTaskCreate
	if _, err := service.CreatePlanItems(ctx, cc, cmd); err == nil {
		t.Fatal("replay disclosed work after constituent permission was revoked")
	}
	policy.denied = ports.PermissionCriterionDefine
	if _, err := service.CreateWorkItem(ctx, commandContext(), application.CreateWorkItemCommand{Scope: owner.Value.Scope(), Title: "Old creation contract", Priority: domain.PriorityNormal}); err == nil {
		t.Fatal("CreateWorkItem accepted a missing Objective")
	}
	_, err = service.CreateWorkItem(ctx, commandContext(), application.CreateWorkItemCommand{Scope: owner.Value.Scope(), ObjectiveID: objectiveID, Title: "Creation with criterion", Priority: domain.PriorityNormal, InitialCriteria: []application.InitialCriterion{initialProof("Verified")}})
	if err == nil {
		t.Fatal("initial criteria bypassed criterion:define")
	}
}

func TestSingleAndBatchCreationRejectTerminalObjectiveParents(t *testing.T) {
	ctx := context.Background()
	service, store := newWave09Service(t)
	outcome := setupActiveOutcome(t, service)
	parent, err := service.CreateObjective(ctx, commandContext(), application.CreateObjectiveCommand{Scope: outcome.Scope(), Title: "Retired planning branch", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.CancelObjective(ctx, commandContext(), application.CancelObjectiveCommand{Scope: outcome.Scope(), ObjectiveID: parent.Value.ID, ExpectedVersion: parent.Value.Version, Reason: "This branch is no longer pursued"})
	if err != nil {
		t.Fatal(err)
	}
	before, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	eventCount := len(store.SnapshotDomainEvents(outcome.Scope()))
	_, err = service.CreateObjective(ctx, commandContext(), application.CreateObjectiveCommand{Scope: outcome.Scope(), Title: "Invalid child", Priority: domain.PriorityNormal, ParentObjectiveID: &parent.Value.ID, InitialCriteria: []application.InitialCriterion{initialProof("Must roll back")}})
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeInvalidTransition {
		t.Fatalf("single parent error=%v", err)
	}
	_, err = service.CreatePlanItems(ctx, planningIntent(t.Name()), application.CreatePlanItemsCommand{Scope: outcome.Scope(), Items: []application.PlanItemInput{{LocalKey: "first", Kind: domain.EntityKindWorkItem, Title: "Must roll back", ParentObjectiveID: &parent.Value.ID}, {LocalKey: "invalid", Kind: domain.EntityKindObjective, Title: "Invalid child", ParentObjectiveID: &parent.Value.ID}}})
	if code, _ := domain.ErrorCodeOf(err); code != domain.ErrorCodeInvalidTransition {
		t.Fatalf("batch parent error=%v", err)
	}
	after, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if len(before.Objectives) != len(after.Objectives) || len(before.WorkItems) != len(after.WorkItems) || eventCount != len(store.SnapshotDomainEvents(outcome.Scope())) || before.OutcomeRevision != after.OutcomeRevision {
		t.Fatal("terminal parent creation persisted partial state")
	}
}

func TestCreateWorkItemWithoutObjectiveDoesNotMutateOutcome(t *testing.T) {
	ctx := context.Background()
	service, store := newWave09Service(t)
	outcome := setupActiveOutcome(t, service)
	before, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	events := len(store.SnapshotDomainEvents(outcome.Scope()))
	if _, err := service.CreateWorkItem(ctx, commandContext(), application.CreateWorkItemCommand{
		Scope: outcome.Scope(), Title: "Orphan Task", Priority: domain.PriorityNormal,
	}); err == nil {
		t.Fatal("CreateWorkItem accepted a missing Objective")
	}
	after, err := service.GetOutcomeState(ctx, outcome.Scope())
	if err != nil {
		t.Fatal(err)
	}
	if after.OutcomeRevision != before.OutcomeRevision || len(after.WorkItems) != len(before.WorkItems) || len(store.SnapshotDomainEvents(outcome.Scope())) != events {
		t.Fatal("rejected Task creation changed Outcome revision, work or events")
	}
}

func sameWorkspaceJSON(a, b any) bool {
	first, err := json.Marshal(a)
	if err != nil {
		return false
	}
	second, err := json.Marshal(b)
	return err == nil && string(first) == string(second)
}
