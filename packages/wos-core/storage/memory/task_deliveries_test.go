package memory

import (
	"context"
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func memoryTaskDeliveryFixture(t *testing.T, store *Store, now time.Time) (d.Scope, d.ID, d.ID, d.ActorRef) {
	t.Helper()
	ctx := context.Background()
	scope := d.Scope{NamespaceID: d.MustParseID("019a4510-0000-7000-8000-000000000001"), OutcomeID: d.MustParseID("019a4510-0000-7000-8000-000000000002")}
	workID := d.MustParseID("019a4510-0000-7000-8000-000000000003")
	objectiveID := d.MustParseID("019a4510-0000-7000-8000-000000000004")
	credentialID := d.MustParseID("019a4510-0000-7000-8000-000000000005")
	claimID := d.MustParseID("019a4510-0000-7000-8000-000000000006")
	sessionID := d.MustParseID("019a4510-0000-7000-8000-000000000007")
	principalID := "memory-delivery-executor"
	actor := d.ActorRef{Kind: d.ActorKindAgent, Provider: "memory-test", ID: principalID}
	outcome, err := d.NewOutcome(scope.OutcomeID, scope.NamespaceID, "Delivery storage", "", "A persisted delivery", d.PriorityNormal, now)
	if err != nil {
		t.Fatal(err)
	}
	objective, err := d.NewObjective(objectiveID, scope, "Review delivered work", "", d.PriorityNormal, false, now)
	if err != nil {
		t.Fatal(err)
	}
	work, err := d.NewWorkItemForObjective(workID, scope, &objective.ID, "Persist exact work", "", d.PriorityNormal, d.WorkItemLifecycleBacklog, now)
	if err != nil {
		t.Fatal(err)
	}
	criterion, err := d.NewSuccessCriterion(d.MustParseID("019a4510-0000-7000-8000-000000000012"), work.Ref(), "Delivery can be verified", "Keep the criterion revision in the delivery snapshot", true, d.VerificationModeEvidenceReview)
	if err != nil {
		t.Fatal(err)
	}
	if err := work.AddCriterion(criterion, now); err != nil {
		t.Fatal(err)
	}
	if err := work.Activate(now.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := work.ClaimForCredential(claimID, principalID, credentialID, &sessionID, actor, d.MinLeaseTTL, now.Add(2*time.Second)); err != nil {
		t.Fatal(err)
	}
	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Coordination().LockOutcome(ctx, scope); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Outcomes().Insert(ctx, outcome); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Objectives().Insert(ctx, objective); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.WorkItems().Insert(ctx, work); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	return scope, workID, credentialID, actor
}

func TestMemoryTaskDeliveryRollbackAndAppendOnlyHistory(t *testing.T) {
	ctx := context.Background()
	store := New()
	now := time.Date(2026, 10, 21, 18, 0, 0, 0, time.UTC)
	scope, workID, credentialID, _ := memoryTaskDeliveryFixture(t, store, now)
	deliveryID := d.MustParseID("019a4510-0000-7000-8000-000000000008")
	deliveryAt := now.Add(3 * time.Second)

	create := func(tx ports.UnitOfWork) (d.WorkItem, d.TaskDelivery) {
		work, err := tx.WorkItems().Get(ctx, scope, workID)
		if err != nil {
			t.Fatal(err)
		}
		delivery, err := d.NewLegacyTaskDelivery(deliveryID, work, d.TaskDeliveryMaterial{Summary: "The exact result is ready"}, "memory-delivery-executor", &credentialID, deliveryAt)
		if err != nil {
			t.Fatal(err)
		}
		expected := work.Version
		if err := work.RecordTaskDelivery(delivery, nil, nil, expected, deliveryAt); err != nil {
			definitionDigest, _ := d.TaskDefinitionDigest(work)
			t.Fatalf("record delivery failed: %v; definition=%q delivery_definition=%q snapshot=%#v delivery_snapshot=%#v", err, definitionDigest, delivery.DefinitionDigest, work.Criteria.Items, delivery.CriterionSnapshot)
		}
		if err := tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries().Insert(ctx, delivery); err != nil {
			t.Fatal(err)
		}
		if err := tx.WorkItems().Save(ctx, work, expected); err != nil {
			t.Fatal(err)
		}
		return work, delivery
	}

	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, delivery := create(tx)
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	repo := tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries()
	if _, err := repo.Get(ctx, scope, delivery.ID); err == nil {
		t.Fatal("rolled-back delivery remained visible")
	}
	work, err := tx.WorkItems().Get(ctx, scope, workID)
	if err != nil || work.PendingDeliveryID != nil || work.DeliveryRound != 0 {
		t.Fatalf("rolled-back work navigation = %#v, err=%v", work, err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	work, delivery = create(tx)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	repo = tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries()
	got, err := repo.Pending(ctx, scope, workID)
	if err != nil || got == nil || got.ID != delivery.ID {
		t.Fatalf("pending memory delivery = %#v, err=%v", got, err)
	}
	page, err := repo.List(ctx, scope, ports.TaskDeliveryFilter{Pending: true, Limit: 10})
	if err != nil || len(page) != 1 {
		t.Fatalf("pending memory delivery page = %#v, err=%v", page, err)
	}
	if err := repo.Insert(ctx, delivery); err == nil {
		t.Fatal("a second open delivery was accepted for the Task")
	}
	decisionAt := delivery.DeliveredAt.Add(time.Second)
	decision, err := d.NewTaskDeliveryDecision(d.MustParseID("019a4510-0000-7000-8000-000000000009"), delivery, d.TaskDeliveryDecisionChangesRequested, "reviewer", d.ActorRef{Kind: d.ActorKindHuman, Provider: "local", ID: "reviewer"}, "Correct the result", []string{"Add the missing check"}, decisionAt)
	if err != nil {
		t.Fatal(err)
	}
	work, err = tx.WorkItems().Get(ctx, scope, workID)
	if err != nil {
		t.Fatal(err)
	}
	expected := work.Version
	if err := work.RequestTaskDeliveryChanges(delivery, decision, expected, decisionAt); err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertDecision(ctx, decision); err != nil {
		t.Fatal(err)
	}
	duplicateDecision, err := d.NewTaskDeliveryDecision(d.MustParseID("019a4510-0000-7000-8000-000000000010"), delivery, d.TaskDeliveryDecisionChangesRequested, "reviewer", d.ActorRef{Kind: d.ActorKindHuman, Provider: "local", ID: "reviewer"}, "Second decision", []string{"This must not replace the first"}, decisionAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertDecision(ctx, duplicateDecision); err == nil {
		t.Fatal("a second decision was accepted for the same delivery")
	}
	if err := tx.WorkItems().Save(ctx, work, expected); err != nil {
		t.Fatal(err)
	}
}

func TestMemoryTaskVerificationRejectsStalePendingDefinition(t *testing.T) {
	ctx := context.Background()
	store := New()
	now := time.Date(2026, 10, 21, 18, 30, 0, 0, time.UTC)
	scope, workID, credentialID, actor := memoryTaskDeliveryFixture(t, store, now)
	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	deliveryID := d.MustParseID("019a4510-0000-7000-8000-000000000021")
	work, delivery := createMemoryTaskDeliveryForVerification(t, tx, scope, workID, credentialID, actor, deliveryID, now.Add(3*time.Second))
	if work.PendingDeliveryID == nil || *work.PendingDeliveryID != delivery.ID {
		t.Fatal("fixture did not create a pending delivery")
	}
	reads := tx.(ports.TaskVerificationReadUnitOfWork).TaskVerificationReads()
	if _, err := reads.TaskVerificationFacts(ctx, scope, []d.ID{workID}); err != nil {
		t.Fatalf("valid pending delivery was rejected: %v", err)
	}
	key := entityKey(scope, workID)
	corrupt := tx.(*transaction).workItems[key]
	corrupt.Description = "changed after delivery"
	tx.(*transaction).workItems[key] = corrupt
	if _, err := reads.TaskVerificationFacts(ctx, scope, []d.ID{workID}); err == nil {
		t.Fatal("Task verification accepted a pending delivery for a stale Task definition")
	}
}

func createMemoryTaskDeliveryForVerification(t *testing.T, tx ports.UnitOfWork, scope d.Scope, workID, credentialID d.ID, actor d.ActorRef, deliveryID d.ID, at time.Time) (d.WorkItem, d.TaskDelivery) {
	t.Helper()
	work, err := tx.WorkItems().Get(context.Background(), scope, workID)
	if err != nil {
		t.Fatal(err)
	}
	delivery, err := d.NewLegacyTaskDelivery(deliveryID, work, d.TaskDeliveryMaterial{Summary: "The exact result is ready"}, "memory-delivery-executor", &credentialID, at)
	if err != nil {
		t.Fatal(err)
	}
	expected := work.Version
	if err := work.RecordTaskDelivery(delivery, nil, nil, expected, at); err != nil {
		t.Fatal(err)
	}
	if err := tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries().Insert(context.Background(), delivery); err != nil {
		t.Fatal(err)
	}
	if err := tx.WorkItems().Save(context.Background(), work, expected); err != nil {
		t.Fatal(err)
	}
	return work, delivery
}

func TestMemoryTaskDeliveryWithdrawalIsScopedAppendOnlyAndRollsBack(t *testing.T) {
	ctx := context.Background()
	store := New()
	now := time.Date(2026, 10, 21, 20, 0, 0, 0, time.UTC)
	scope, workID, credentialID, _ := memoryTaskDeliveryFixture(t, store, now)
	deliveryID := d.MustParseID("019a4510-0000-7000-8000-000000000020")
	deliveryAt := now.Add(3 * time.Second)
	withdrawalID := d.MustParseID("019a4510-0000-7000-8000-000000000021")
	withdrawnAt := deliveryAt.Add(time.Second)

	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	work, delivery := func() (d.WorkItem, d.TaskDelivery) {
		work, err := tx.WorkItems().Get(ctx, scope, workID)
		if err != nil {
			t.Fatal(err)
		}
		delivery, err := d.NewLegacyTaskDelivery(deliveryID, work, d.TaskDeliveryMaterial{Summary: "The exact result is ready"}, "memory-delivery-executor", &credentialID, deliveryAt)
		if err != nil {
			t.Fatal(err)
		}
		expected := work.Version
		if err := work.RecordTaskDelivery(delivery, nil, nil, expected, deliveryAt); err != nil {
			t.Fatal(err)
		}
		if err := tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries().Insert(ctx, delivery); err != nil {
			t.Fatal(err)
		}
		if err := tx.WorkItems().Save(ctx, work, expected); err != nil {
			t.Fatal(err)
		}
		return work, delivery
	}()
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	withdrawal := d.TaskDeliveryWithdrawal{
		ID: withdrawalID, Scope: scope, WorkItemID: workID, DeliveryID: delivery.ID,
		DeliveryDigest: delivery.Digest, Round: delivery.Round,
		PrincipalID: "memory-reviewer", Actor: d.ActorRef{Kind: d.ActorKindHuman, Provider: "local", ID: "reviewer"},
		Reason: "Withdrawn before review", WithdrawnAt: withdrawnAt,
	}
	withdraw := func(tx ports.UnitOfWork) {
		work, err := tx.WorkItems().Get(ctx, scope, workID)
		if err != nil {
			t.Fatal(err)
		}
		expected := work.Version
		if err := work.WithdrawTaskDelivery(delivery, withdrawal, expected, withdrawnAt); err != nil {
			t.Fatal(err)
		}
		repo := tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries()
		if err := repo.InsertWithdrawal(ctx, withdrawal); err != nil {
			t.Fatal(err)
		}
		if err := tx.WorkItems().Save(ctx, work, expected); err != nil {
			t.Fatal(err)
		}
	}

	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	withdraw(tx)
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	repo := tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries()
	if _, err := repo.GetWithdrawal(ctx, scope, withdrawalID); err == nil {
		t.Fatal("rolled-back TaskDeliveryWithdrawal remained visible")
	}
	work, err = tx.WorkItems().Get(ctx, scope, workID)
	if err != nil || work.PendingDeliveryID == nil || *work.PendingDeliveryID != delivery.ID || work.LatestDeliveryWithdrawalID != nil {
		t.Fatalf("rolled-back withdrawal navigation = %#v, err=%v", work, err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	withdraw(tx)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	repo = tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries()
	got, err := repo.GetWithdrawal(ctx, scope, withdrawalID)
	if err != nil || got.DeliveryID != delivery.ID || got.DeliveryDigest != delivery.Digest || got.Reason != withdrawal.Reason {
		t.Fatalf("withdrawal after commit = %#v, err=%v", got, err)
	}
	byDelivery, err := repo.WithdrawalForDelivery(ctx, scope, delivery.ID)
	if err != nil || byDelivery == nil || byDelivery.ID != withdrawal.ID {
		t.Fatalf("withdrawal by exact delivery = %#v, err=%v", byDelivery, err)
	}
	work, err = tx.WorkItems().Get(ctx, scope, workID)
	if err != nil || work.LatestDeliveryWithdrawalID == nil || *work.LatestDeliveryWithdrawalID != withdrawal.ID || work.PendingDeliveryID != nil {
		t.Fatalf("withdrawn WorkItem navigation = %#v, err=%v", work, err)
	}
	if err := repo.InsertWithdrawal(ctx, withdrawal); err == nil {
		t.Fatal("a second withdrawal was accepted for the same delivery")
	}
}
