package sqlite

import (
	"context"
	"errors"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func taskDeliveryStorageFixture(t *testing.T, store *Store, now time.Time) (d.Scope, d.ID, d.ID, d.ID, d.ActorRef) {
	t.Helper()
	ctx := context.Background()
	scope := d.Scope{NamespaceID: testID("019a4500-0000-7000-8000-000000000001"), OutcomeID: testID("019a4500-0000-7000-8000-000000000002")}
	workID := testID("019a4500-0000-7000-8000-000000000003")
	objectiveID := testID("019a4500-0000-7000-8000-000000000004")
	principalID := "delivery-executor"
	credentialID := testID("019a4500-0000-7000-8000-000000000005")
	claimID := testID("019a4500-0000-7000-8000-000000000006")
	sessionID := testID("019a4500-0000-7000-8000-000000000007")
	actor := d.ActorRef{Kind: d.ActorKindAgent, Provider: "sqlite-test", ID: principalID}
	if _, err := store.db.ExecContext(ctx, `INSERT INTO principals(id,display_name,kind,lifecycle,created_at) VALUES(?, 'Delivery executor', 'service', 'active', ?)`, principalID, encodeTime(now)); err != nil {
		t.Fatal(err)
	}
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
	criterion, err := d.NewSuccessCriterion(testID("019a4500-0000-7000-8000-000000000012"), work.Ref(), "Delivery can be verified", "Keep the criterion revision in the delivery snapshot", true, d.VerificationModeEvidenceReview)
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
	return scope, workID, objectiveID, credentialID, actor
}

func makeStorageTaskDelivery(t *testing.T, tx ports.UnitOfWork, scope d.Scope, workID, credentialID d.ID, actor d.ActorRef, deliveryID d.ID, at time.Time) (d.WorkItem, d.TaskDelivery) {
	t.Helper()
	work, err := tx.WorkItems().Get(context.Background(), scope, workID)
	if err != nil {
		t.Fatal(err)
	}
	delivery, err := d.NewLegacyTaskDelivery(deliveryID, work, d.TaskDeliveryMaterial{Summary: "The exact result is ready"}, "delivery-executor", &credentialID, at)
	if err != nil {
		t.Fatal(err)
	}
	return work, delivery
}

func persistTaskDelivery(t *testing.T, tx ports.UnitOfWork, work d.WorkItem, delivery d.TaskDelivery) d.WorkItem {
	t.Helper()
	expected := work.Version
	if err := work.RecordTaskDelivery(delivery, nil, nil, expected, delivery.DeliveredAt); err != nil {
		t.Fatal(err)
	}
	repo := tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries()
	if err := repo.Insert(context.Background(), delivery); err != nil {
		t.Fatal(err)
	}
	if err := tx.WorkItems().Save(context.Background(), work, expected); err != nil {
		t.Fatal(err)
	}
	return work
}

func TestTaskDeliverySQLiteRoundTripDecisionAndRollback(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "task-deliveries.db")
	store := openTestStore(t, path)
	now := time.Date(2026, 10, 21, 16, 0, 0, 0, time.UTC)
	scope, workID, _, credentialID, actor := taskDeliveryStorageFixture(t, store, now)
	deliveryID := testID("019a4500-0000-7000-8000-000000000008")
	deliveryAt := now.Add(3 * time.Second)

	// A failed multi-record operation must leave neither its immutable delivery
	// nor the WorkItem navigation behind.
	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	work, delivery := makeStorageTaskDelivery(t, tx, scope, workID, credentialID, actor, deliveryID, deliveryAt)
	persistTaskDelivery(t, tx, work, delivery)
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries().Get(ctx, scope, deliveryID); err == nil {
		t.Fatal("rolled-back delivery remained visible")
	}
	work, err = tx.WorkItems().Get(ctx, scope, workID)
	if err != nil || work.PendingDeliveryID != nil || work.DeliveryRound != 0 {
		t.Fatalf("rolled-back TaskDelivery navigation = %#v, err=%v", work, err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	work, delivery = makeStorageTaskDelivery(t, tx, scope, workID, credentialID, actor, deliveryID, deliveryAt)
	persistTaskDelivery(t, tx, work, delivery)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store = openTestStore(t, path)
	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	repo := tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries()
	got, err := repo.Get(ctx, scope, deliveryID)
	if err != nil || got.Digest != delivery.Digest || got.Material == nil || got.Material.Summary != delivery.Summary {
		t.Fatalf("delivery after restart = %#v, err=%v", got, err)
	}
	pending, err := repo.Pending(ctx, scope, workID)
	if err != nil || pending == nil || pending.ID != delivery.ID {
		t.Fatalf("pending delivery after restart = %#v, err=%v", pending, err)
	}
	page, err := repo.List(ctx, scope, ports.TaskDeliveryFilter{WorkItemID: workID, Pending: true, Limit: 10})
	if err != nil || len(page) != 1 || page[0].ID != delivery.ID {
		t.Fatalf("pending delivery page = %#v, err=%v", page, err)
	}
	if err := repo.Insert(ctx, delivery); err == nil {
		t.Fatal("a second open delivery was accepted for the Task")
	}
	uow := tx.(*unitOfWork)
	if _, err := uow.tx.ExecContext(ctx, `SAVEPOINT duplicate_pending_delivery`); err != nil {
		t.Fatal(err)
	}
	secondDeliveryID := testID("019a4500-0000-7000-8000-000000000011")
	payload := `{"supersedes_delivery_id":"` + delivery.ID.String() + `"}`
	_, insertErr := uow.tx.ExecContext(ctx, `INSERT INTO task_deliveries(id,namespace_id,outcome_id,work_item_id,delivery_round,authority_kind,delivery_digest,definition_digest,payload_json)
	VALUES(?,?,?,?,'2','legacy_lease',?,?,?)`, secondDeliveryID.String(), scope.NamespaceID.String(), scope.OutcomeID.String(), workID.String(), "sha256:"+strings.Repeat("1", 64), "sha256:"+strings.Repeat("2", 64), payload)
	if insertErr == nil {
		t.Fatal("database allowed a second delivery while the first was still pending")
	}
	if _, err := uow.tx.ExecContext(ctx, `ROLLBACK TO duplicate_pending_delivery`); err != nil {
		t.Fatal(err)
	}
	if _, err := uow.tx.ExecContext(ctx, `RELEASE duplicate_pending_delivery`); err != nil {
		t.Fatal(err)
	}

	decisionID := testID("019a4500-0000-7000-8000-000000000009")
	decisionAt := deliveryAt.Add(time.Second)
	decision, err := d.NewTaskDeliveryDecision(decisionID, delivery, d.TaskDeliveryDecisionChangesRequested, "reviewer", d.ActorRef{Kind: d.ActorKindHuman, Provider: "local", ID: "reviewer"}, "Please correct the result", []string{"Include the missing verification"}, decisionAt)
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
	duplicateDecision, err := d.NewTaskDeliveryDecision(testID("019a4500-0000-7000-8000-000000000010"), delivery, d.TaskDeliveryDecisionChangesRequested, "reviewer", d.ActorRef{Kind: d.ActorKindHuman, Provider: "local", ID: "reviewer"}, "Duplicate decision", []string{"This decision must not be recorded"}, decisionAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertDecision(ctx, duplicateDecision); err == nil {
		t.Fatal("a second decision was accepted for the same delivery")
	}
	if err := tx.WorkItems().Save(ctx, work, expected); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	repo = tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries()
	gotDecision, err := repo.GetDecision(ctx, scope, decisionID)
	if err != nil || gotDecision.DeliveryID != delivery.ID || gotDecision.Result != d.TaskDeliveryDecisionChangesRequested {
		t.Fatalf("decision after restart = %#v, err=%v", gotDecision, err)
	}
	listed, err := repo.ListDecisions(ctx, scope, delivery.ID, "", 10)
	if err != nil || len(listed) != 1 || listed[0].ID != decision.ID {
		t.Fatalf("decision history = %#v, err=%v", listed, err)
	}
	pending, err = repo.Pending(ctx, scope, workID)
	if err != nil || pending != nil {
		t.Fatalf("resolved delivery is still pending: %#v, err=%v", pending, err)
	}
	if _, err := uow.tx.ExecContext(ctx, `UPDATE task_deliveries SET delivery_digest='tampered' WHERE namespace_id=? AND outcome_id=? AND id=?`, scope.NamespaceID.String(), scope.OutcomeID.String(), delivery.ID.String()); err == nil {
		t.Fatal("database allowed an immutable TaskDelivery to be changed")
	}
	if _, err := uow.tx.ExecContext(ctx, `DELETE FROM task_deliveries WHERE namespace_id=? AND outcome_id=? AND id=?`, scope.NamespaceID.String(), scope.OutcomeID.String(), delivery.ID.String()); err == nil {
		t.Fatal("database allowed an immutable TaskDelivery to be deleted")
	}
	if _, err := uow.tx.ExecContext(ctx, `UPDATE task_delivery_decisions SET result='accepted' WHERE namespace_id=? AND outcome_id=? AND id=?`, scope.NamespaceID.String(), scope.OutcomeID.String(), decision.ID.String()); err == nil {
		t.Fatal("database allowed a TaskDeliveryDecision to be changed")
	}
	if _, err := uow.tx.ExecContext(ctx, `DELETE FROM task_delivery_decisions WHERE namespace_id=? AND outcome_id=? AND id=?`, scope.NamespaceID.String(), scope.OutcomeID.String(), decision.ID.String()); err == nil {
		t.Fatal("database allowed a TaskDeliveryDecision to be deleted")
	}
}

func TestTaskVerificationRejectsStalePendingDefinition(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "task-verification-stale-definition.db"))
	now := time.Date(2026, 10, 21, 16, 30, 0, 0, time.UTC)
	scope, workID, _, credentialID, actor := taskDeliveryStorageFixture(t, store, now)
	uow, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer uow.Rollback()
	work, delivery := makeStorageTaskDelivery(t, uow, scope, workID, credentialID, actor, testID("019a4500-0000-7000-8000-000000000021"), now.Add(3*time.Second))
	persistTaskDelivery(t, uow, work, delivery)
	reads := uow.(ports.TaskVerificationReadUnitOfWork).TaskVerificationReads()
	if _, err := reads.TaskVerificationFacts(ctx, scope, []d.ID{workID}); err != nil {
		t.Fatalf("valid pending delivery was rejected: %v", err)
	}
	if _, err := uow.(*unitOfWork).tx.ExecContext(ctx, `UPDATE work_items SET description=? WHERE namespace_id=? AND outcome_id=? AND id=?`, "changed after delivery", scope.NamespaceID.String(), scope.OutcomeID.String(), workID.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := reads.TaskVerificationFacts(ctx, scope, []d.ID{workID}); err == nil {
		t.Fatal("Task verification accepted a pending delivery for a stale Task definition")
	}
}

func TestTaskDeliveryWithdrawalExactBindingRollbackAndMutualExclusion(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "task-delivery-withdrawal.db")
	store := openTestStore(t, path)
	now := time.Date(2026, 10, 21, 21, 0, 0, 0, time.UTC)
	scope, workID, _, credentialID, _ := taskDeliveryStorageFixture(t, store, now)
	deliveryID := testID("019a4500-0000-7000-8000-000000000030")
	deliveryAt := now.Add(3 * time.Second)
	withdrawal := d.TaskDeliveryWithdrawal{
		ID: testID("019a4500-0000-7000-8000-000000000031"), Scope: scope, WorkItemID: workID, DeliveryID: deliveryID,
		DeliveryDigest: "", Round: 0, PrincipalID: "sqlite-reviewer",
		Actor:  d.ActorRef{Kind: d.ActorKindHuman, Provider: "local", ID: "reviewer"},
		Reason: "Withdrawn before review", WithdrawnAt: deliveryAt.Add(time.Second),
	}

	// The composite foreign key binds a withdrawal to the exact delivery tuple,
	// including round and digest, even if a caller bypasses the repository.
	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	work, delivery := makeStorageTaskDelivery(t, tx, scope, workID, credentialID, d.ActorRef{Kind: d.ActorKindAgent, Provider: "sqlite-test", ID: "delivery-executor"}, deliveryID, deliveryAt)
	persistTaskDelivery(t, tx, work, delivery)
	uow := tx.(*unitOfWork)
	if _, err := uow.tx.ExecContext(ctx, `SAVEPOINT mismatched_withdrawal`); err != nil {
		t.Fatal(err)
	}
	if _, err := uow.tx.ExecContext(ctx, `INSERT INTO task_delivery_withdrawals(id,namespace_id,outcome_id,work_item_id,delivery_id,delivery_round,delivery_digest,payload_json)
VALUES(?,?,?,?,?,'1',?,'{}')`, testID("019a4500-0000-7000-8000-000000000032").String(), scope.NamespaceID.String(), scope.OutcomeID.String(), workID.String(), delivery.ID.String(), "sha256:"+strings.Repeat("0", 64)); err == nil {
		t.Fatal("database accepted a withdrawal whose digest did not match the exact delivery")
	}
	if _, err := uow.tx.ExecContext(ctx, `ROLLBACK TO mismatched_withdrawal`); err != nil {
		t.Fatal(err)
	}
	if _, err := uow.tx.ExecContext(ctx, `RELEASE mismatched_withdrawal`); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	withdrawal.DeliveryDigest = delivery.Digest
	withdrawal.Round = delivery.Round
	work, err = tx.WorkItems().Get(ctx, scope, workID)
	if err != nil {
		t.Fatal(err)
	}
	expected := work.Version
	if err := work.WithdrawTaskDelivery(delivery, withdrawal, expected, withdrawal.WithdrawnAt); err != nil {
		t.Fatal(err)
	}
	repo := tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries()
	if err := repo.InsertWithdrawal(ctx, withdrawal); err != nil {
		t.Fatal(err)
	}
	if err := tx.WorkItems().Save(ctx, work, expected); err != nil {
		t.Fatal(err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	// Rollback restores both the immutable history row and navigation pointer.
	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	repo = tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries()
	if _, err := repo.GetWithdrawal(ctx, scope, withdrawal.ID); err == nil {
		t.Fatal("rolled-back withdrawal remained visible")
	}
	work, err = tx.WorkItems().Get(ctx, scope, workID)
	if err != nil || work.PendingDeliveryID == nil || *work.PendingDeliveryID != delivery.ID || work.LatestDeliveryWithdrawalID != nil {
		t.Fatalf("rolled-back withdrawal navigation = %#v, err=%v", work, err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	// Persist one withdrawal and validate restart, immutable storage, scoped
	// navigation, and the database guard against a competing review decision.
	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	work, err = tx.WorkItems().Get(ctx, scope, workID)
	if err != nil {
		t.Fatal(err)
	}
	expected = work.Version
	if err := work.WithdrawTaskDelivery(delivery, withdrawal, expected, withdrawal.WithdrawnAt); err != nil {
		t.Fatal(err)
	}
	repo = tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries()
	if err := repo.InsertWithdrawal(ctx, withdrawal); err != nil {
		t.Fatal(err)
	}
	if err := tx.WorkItems().Save(ctx, work, expected); err != nil {
		t.Fatal(err)
	}
	if err := repo.InsertWithdrawal(ctx, withdrawal); err == nil {
		t.Fatal("repository accepted a second withdrawal for the same delivery")
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store = openTestStore(t, path)
	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	uow = tx.(*unitOfWork)
	repo = tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries()
	got, err := repo.GetWithdrawal(ctx, scope, withdrawal.ID)
	if err != nil || got.DeliveryDigest != delivery.Digest || got.Reason != withdrawal.Reason {
		t.Fatalf("withdrawal after restart = %#v, err=%v", got, err)
	}
	byDelivery, err := repo.WithdrawalForDelivery(ctx, scope, delivery.ID)
	if err != nil || byDelivery == nil || byDelivery.ID != withdrawal.ID {
		t.Fatalf("withdrawal by delivery = %#v, err=%v", byDelivery, err)
	}
	work, err = tx.WorkItems().Get(ctx, scope, workID)
	if err != nil || work.LatestDeliveryWithdrawalID == nil || *work.LatestDeliveryWithdrawalID != withdrawal.ID || work.PendingDeliveryID != nil {
		t.Fatalf("persisted withdrawal navigation = %#v, err=%v", work, err)
	}
	decision, err := d.NewTaskDeliveryDecision(testID("019a4500-0000-7000-8000-000000000033"), delivery, d.TaskDeliveryDecisionAccepted, "reviewer", d.ActorRef{Kind: d.ActorKindHuman, Provider: "local", ID: "reviewer"}, "Accept", nil, withdrawal.WithdrawnAt.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	decisionPayload, err := marshalJSON(decision)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uow.tx.ExecContext(ctx, `INSERT INTO task_delivery_decisions(id,namespace_id,outcome_id,work_item_id,delivery_id,delivery_round,delivery_digest,result,payload_json)
VALUES(?,?,?,?,?,?,?,?,?)`, decision.ID.String(), scope.NamespaceID.String(), scope.OutcomeID.String(), workID.String(), delivery.ID.String(), strconv.FormatUint(delivery.Round, 10), delivery.Digest, string(decision.Result), decisionPayload); err == nil {
		t.Fatal("database allowed both a withdrawal and a decision for one delivery")
	}
	if _, err := uow.tx.ExecContext(ctx, `UPDATE task_delivery_withdrawals SET delivery_digest='tampered' WHERE namespace_id=? AND outcome_id=? AND id=?`, scope.NamespaceID.String(), scope.OutcomeID.String(), withdrawal.ID.String()); err == nil {
		t.Fatal("database allowed an immutable TaskDeliveryWithdrawal to be changed")
	}
	if _, err := uow.tx.ExecContext(ctx, `DELETE FROM task_delivery_withdrawals WHERE namespace_id=? AND outcome_id=? AND id=?`, scope.NamespaceID.String(), scope.OutcomeID.String(), withdrawal.ID.String()); err == nil {
		t.Fatal("database allowed an immutable TaskDeliveryWithdrawal to be deleted")
	}
}

func TestTaskDeliverySQLiteScopedForeignKeyRejectsCrossScopeWorkItem(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "task-delivery-fk.db"))
	now := time.Date(2026, 10, 21, 17, 0, 0, 0, time.UTC)
	scope, workID, _, _, _ := taskDeliveryStorageFixture(t, store, now)
	otherScope := d.Scope{NamespaceID: testID("019a4501-0000-7000-8000-000000000001"), OutcomeID: testID("019a4501-0000-7000-8000-000000000002")}
	_, err := store.db.ExecContext(ctx, `INSERT INTO task_deliveries(id,namespace_id,outcome_id,work_item_id,delivery_round,authority_kind,delivery_digest,definition_digest,payload_json)
VALUES(?,?,?,?,'1','legacy_lease','digest','definition','{}')`, testID("019a4501-0000-7000-8000-000000000003").String(), otherScope.NamespaceID.String(), otherScope.OutcomeID.String(), workID.String())
	if err == nil {
		t.Fatal("cross-scope TaskDelivery WorkItem foreign key was accepted")
	}
	if scope.NamespaceID == otherScope.NamespaceID {
		t.Fatal("fixture scopes unexpectedly match")
	}
}

func TestTaskDeliveryReplanAuthorizationSurvivesSQLiteRestart(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "task-delivery-replan.db")
	store := openTestStore(t, path)
	now := time.Date(2026, 10, 21, 22, 0, 0, 0, time.UTC)
	scope, workID, _, credentialID, actor := taskDeliveryStorageFixture(t, store, now)
	firstID := testID("019a4500-0000-7000-8000-000000000040")
	secondID := testID("019a4500-0000-7000-8000-000000000041")
	withdrawalID := testID("019a4500-0000-7000-8000-000000000042")

	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	work, first := makeStorageTaskDelivery(t, tx, scope, workID, credentialID, actor, firstID, now.Add(3*time.Second))
	work = persistTaskDelivery(t, tx, work, first)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	work, err = tx.WorkItems().Get(ctx, scope, workID)
	if err != nil {
		t.Fatal(err)
	}
	withdrawnAt := now.Add(4 * time.Second)
	withdrawal := d.TaskDeliveryWithdrawal{
		ID: withdrawalID, Scope: scope, WorkItemID: workID, DeliveryID: first.ID,
		DeliveryDigest: first.Digest, Round: first.Round, PrincipalID: "sqlite-reviewer",
		Actor:  d.ActorRef{Kind: d.ActorKindHuman, Provider: "local", ID: "reviewer"},
		Reason: "Return this delivery to planning", WithdrawnAt: withdrawnAt,
	}
	expected := work.Version
	if err := work.WithdrawTaskDelivery(first, withdrawal, expected, withdrawnAt); err != nil {
		t.Fatal(err)
	}
	if err := tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries().InsertWithdrawal(ctx, withdrawal); err != nil {
		t.Fatal(err)
	}
	if err := tx.WorkItems().Save(ctx, work, expected); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	// The replan marker is created only by the explicit To do -> Backlog
	// transition and survives a process restart before activation.
	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	work, err = tx.WorkItems().Get(ctx, scope, workID)
	if err != nil {
		t.Fatal(err)
	}
	expected = work.Version
	if err := work.Defer(withdrawnAt.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	if !work.TaskDeliveryReplanAuthorized {
		t.Fatal("explicit Defer did not set the replan marker")
	}
	if err := tx.WorkItems().Save(ctx, work, expected); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store = openTestStore(t, path)

	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	work, err = tx.WorkItems().Get(ctx, scope, workID)
	if err != nil || work.Lifecycle != d.WorkItemLifecycleBacklog || !work.TaskDeliveryReplanAuthorized || work.LatestDeliveryWithdrawalID == nil || *work.LatestDeliveryWithdrawalID != withdrawal.ID {
		t.Fatalf("Backlog replan state after restart = %+v, err=%v", work, err)
	}
	expected = work.Version
	if err := work.Activate(withdrawnAt.Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := tx.WorkItems().Save(ctx, work, expected); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	work, err = tx.WorkItems().Get(ctx, scope, workID)
	if err != nil || work.Lifecycle != d.WorkItemLifecycleTodo || !work.TaskDeliveryReplanAuthorized {
		t.Fatalf("activated replan state = %+v, err=%v", work, err)
	}
	expected = work.Version
	if err := work.ClaimForCredential(testID("019a4500-0000-7000-8000-000000000043"), "delivery-executor", credentialID, nil, actor, d.MinLeaseTTL, now.Add(7*time.Second)); err != nil {
		t.Fatalf("claim after explicit replan = %v", err)
	}
	if err := tx.WorkItems().Save(ctx, work, expected); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store = openTestStore(t, path)

	// A second delivery consumes the marker while preserving the immutable
	// withdrawal and delivery predecessor chain.
	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	work, err = tx.WorkItems().Get(ctx, scope, workID)
	if err != nil || work.Lifecycle != d.WorkItemLifecycleInProgress || !work.TaskDeliveryReplanAuthorized || work.CurrentLease == nil {
		t.Fatalf("claimed replan state after restart = %+v, err=%v", work, err)
	}
	second, err := d.NewLegacyTaskDelivery(secondID, work, d.TaskDeliveryMaterial{Summary: "Corrected after replan"}, "delivery-executor", &credentialID, now.Add(8*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	expected = work.Version
	if err := work.RecordTaskDelivery(second, nil, nil, expected, now.Add(8*time.Second)); err != nil {
		t.Fatal(err)
	}
	repo := tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries()
	if err := repo.Insert(ctx, second); err != nil {
		t.Fatal(err)
	}
	if err := tx.WorkItems().Save(ctx, work, expected); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if err := store.Close(); err != nil {
		t.Fatal(err)
	}
	store = openTestStore(t, path)
	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	work, err = tx.WorkItems().Get(ctx, scope, workID)
	if err != nil || work.TaskDeliveryReplanAuthorized || work.LatestDeliveryWithdrawalID != nil || work.LatestDeliveryID == nil || *work.LatestDeliveryID != second.ID || work.DeliveryRound != first.Round+1 {
		t.Fatalf("second-round Task navigation after restart = %+v, err=%v", work, err)
	}
	storedFirst, err := tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries().Get(ctx, scope, first.ID)
	if err != nil || storedFirst.ID != first.ID || storedFirst.Digest != first.Digest {
		t.Fatalf("first delivery after replanning = %+v, err=%v", storedFirst, err)
	}
	storedSecond, err := tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries().Get(ctx, scope, second.ID)
	if err != nil || storedSecond.Round != first.Round+1 || storedSecond.SupersedesDeliveryID == nil || *storedSecond.SupersedesDeliveryID != first.ID {
		t.Fatalf("second delivery chain after replanning = %+v, err=%v", storedSecond, err)
	}
}

func TestTaskDeliverySQLitePredecessorBindingHasExactScopedForeignKeys(t *testing.T) {
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "task-delivery-predecessor-fk.db"))
	now := time.Date(2026, 10, 21, 17, 30, 0, 0, time.UTC)
	scope, workID, _, credentialID, actor := taskDeliveryStorageFixture(t, store, now)
	deliveryID := testID("019a4511-0000-7000-8000-000000000001")
	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	work, delivery := makeStorageTaskDelivery(t, tx, scope, workID, credentialID, actor, deliveryID, now.Add(3*time.Second))
	persistTaskDelivery(t, tx, work, delivery)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	_, err = store.db.ExecContext(ctx, `INSERT INTO task_delivery_predecessors(namespace_id,outcome_id,work_item_id,delivery_id,delivery_round,delivery_digest,predecessor_delivery_id,predecessor_round,predecessor_digest)
VALUES(?,?,?,?,?,?,?,?,?)`, scope.NamespaceID.String(), scope.OutcomeID.String(), workID.String(), delivery.ID.String(), strconv.FormatUint(delivery.Round, 10), delivery.Digest,
		testID("019a4511-0000-7000-8000-000000000002").String(), "1", "sha256:"+strings.Repeat("0", 64))
	if err == nil {
		t.Fatal("database accepted a TaskDelivery predecessor without an exact scoped parent row")
	}
}

func TestTaskDeliverySQLiteMaterialReferencesRequireScopedRows(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 21, 19, 0, 0, 0, time.UTC)
	for index, material := range []d.TaskDeliveryMaterial{
		{Summary: "Missing Artifact", Artifacts: []d.SubmissionArtifact{{ArtifactID: testID("019a4520-0000-7000-8000-000000000001"), Checksum: "sha256:artifact"}}},
		{Summary: "Missing Evidence", EvidenceIDs: []d.ID{testID("019a4520-0000-7000-8000-000000000002")}},
	} {
		t.Run(material.Summary, func(t *testing.T) {
			store := openTestStore(t, filepath.Join(t.TempDir(), "material-reference.db"))
			scope, workID, _, credentialID, _ := taskDeliveryStorageFixture(t, store, now.Add(time.Duration(index)*time.Minute))
			tx, err := store.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			work, err := tx.WorkItems().Get(ctx, scope, workID)
			if err != nil {
				t.Fatal(err)
			}
			delivery, err := d.NewLegacyTaskDelivery(testID("019a4520-0000-7000-8000-000000000003"), work, material, "delivery-executor", &credentialID, now.Add(time.Duration(index)*time.Minute+3*time.Second))
			if err != nil {
				t.Fatal(err)
			}
			err = tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries().Insert(ctx, delivery)
			if err == nil {
				t.Fatalf("TaskDelivery with missing %s row was accepted", material.Summary)
			}
			if err := tx.Rollback(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestTaskDeliverySQLiteMaterialReferencesRoundTrip(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 21, 20, 0, 0, 0, time.UTC)
	store := openTestStore(t, filepath.Join(t.TempDir(), "material-roundtrip.db"))
	scope, workID, _, credentialID, actor := taskDeliveryStorageFixture(t, store, now)
	artifactID := testID("019a4530-0000-7000-8000-000000000001")
	evidenceID := testID("019a4530-0000-7000-8000-000000000002")
	deliveryID := testID("019a4530-0000-7000-8000-000000000003")
	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	artifact, err := d.NewArtifact(artifactID, scope, "build", "Verified build", "repo://build/42", "", "sha256:artifact", "commit-42", actor, nil, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.(ports.DocumentaryUnitOfWork).Artifacts().Insert(ctx, artifact); err != nil {
		t.Fatal(err)
	}
	evidence, err := d.NewEvidence(evidenceID, scope, d.EvidenceTypeTestResult, "The test suite passed", d.SourceReference{Provider: "ci", ID: "run-42"}, actor, now.Add(time.Second), &artifactID, nil, "commit-42", "sha256:result", now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.(ports.DocumentaryUnitOfWork).Evidence().Insert(ctx, evidence); err != nil {
		t.Fatal(err)
	}
	work, err := tx.WorkItems().Get(ctx, scope, workID)
	if err != nil {
		t.Fatal(err)
	}
	material := d.TaskDeliveryMaterial{Summary: "Verified build and test run", Artifacts: []d.SubmissionArtifact{{ArtifactID: artifactID, SourceVersion: "commit-42", Checksum: "sha256:artifact"}}, EvidenceIDs: []d.ID{evidenceID}, CriterionEvidence: []d.SubmissionCriterionEvidence{{CriterionID: work.Criteria.Items[0].ID, CriterionRevision: 1, EvidenceIDs: []d.ID{evidenceID}}}}
	delivery, err := d.NewLegacyTaskDelivery(deliveryID, work, material, "delivery-executor", &credentialID, now.Add(3*time.Second))
	if err != nil {
		t.Fatal(err)
	}
	persistTaskDelivery(t, tx, work, delivery)
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}
	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	got, err := tx.(ports.TaskDeliveryUnitOfWork).TaskDeliveries().Get(ctx, scope, deliveryID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Material == nil || !equalTaskDeliveryArtifacts(got.Material.Artifacts, material.Artifacts) || !equalTaskDeliveryIDs(got.Material.EvidenceIDs, material.EvidenceIDs) {
		t.Fatalf("TaskDelivery references failed to round-trip: %#v", got.Material)
	}
}

func TestTaskDeliverySQLiteAssessmentBindingRoundTripAndImmutableOwnerRevision(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 21, 21, 0, 0, 0, time.UTC)
	store := openTestStore(t, filepath.Join(t.TempDir(), "assessment-binding.db"))
	scope, workID, objectiveID, credentialID, actor := taskDeliveryStorageFixture(t, store, now)
	criterionID := testID("019a4500-0000-7000-8000-000000000012")
	otherWorkID := testID("019a4500-0000-7000-8000-000000000013")
	otherCriterionID := testID("019a4500-0000-7000-8000-000000000014")
	deliveryID := testID("019a4500-0000-7000-8000-000000000015")
	assessmentID := testID("019a4500-0000-7000-8000-000000000016")
	evidenceID := testID("019a4500-0000-7000-8000-000000000017")
	deliveryAt := now.Add(3 * time.Second)

	// A second Task supplies a valid, same-scope criterion that must still be
	// rejected when an assessment bound to the first Task's delivery is retargeted.
	tx, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	otherWork, err := d.NewWorkItemForObjective(otherWorkID, scope, &objectiveID, "Other task", "", d.PriorityNormal, d.WorkItemLifecycleBacklog, now.Add(time.Second))
	if err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	otherCriterion, err := d.NewSuccessCriterion(otherCriterionID, otherWork.Ref(), "Other task criterion", "", true, d.VerificationModeAttestation)
	if err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := otherWork.AddCriterion(otherCriterion, now.Add(time.Second)); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.WorkItems().Insert(ctx, otherWork); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	work, delivery := makeStorageTaskDelivery(t, tx, scope, workID, credentialID, actor, deliveryID, deliveryAt)
	work = persistTaskDelivery(t, tx, work, delivery)
	evidence, err := d.NewEvidence(evidenceID, scope, d.EvidenceTypeTestResult, "The delivered result passed its focused checks.", d.SourceReference{Provider: "ci", ID: "run-binding-1"}, actor, deliveryAt, nil, nil, "commit-binding-1", "sha256:evidence", deliveryAt)
	if err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.(ports.DocumentaryUnitOfWork).Evidence().Insert(ctx, evidence); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	assessment := d.CriterionAssessment{
		ID:                assessmentID,
		CriterionID:       criterionID,
		CriterionRevision: 1,
		TaskDeliveryRef:   &d.TaskDeliveryAssessmentRef{ID: delivery.ID, Digest: delivery.Digest, Round: delivery.Round},
		Result:            d.AssessmentResultMet,
		Rationale:         "The exact delivered work satisfies the criterion.",
		EvidenceIDs:       []d.ID{evidenceID},
		PrincipalID:       "delivery-executor",
		Actor:             actor,
		AssessedAt:        deliveryAt.Add(time.Second),
	}
	expectedVersion := work.Version
	if err := work.RecordCriterionAssessment(assessment, false, assessment.AssessedAt); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.WorkItems().Save(ctx, work, expectedVersion); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := tx.WorkItems().Get(ctx, scope, workID)
	if err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	got, ok := loaded.Criteria.CurrentAssessments[criterionID]
	if !ok || got.TaskDeliveryRef == nil || *got.TaskDeliveryRef != *assessment.TaskDeliveryRef {
		tx.Rollback()
		t.Fatalf("assessment TaskDelivery binding failed to round-trip: %#v", got)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatal(err)
	}

	// Make a valid later revision available in the FK table. It is not part of
	// this delivery's immutable criterion snapshot and therefore cannot replace
	// the revision the assessment says it reviewed.
	if _, err := store.db.ExecContext(ctx, `INSERT INTO criterion_revisions(namespace_id,outcome_id,criterion_id,criterion_revision,definition_json,required,verification_mode,created_at,created_by_json)
SELECT namespace_id,outcome_id,criterion_id,2,definition_json,required,verification_mode,created_at,created_by_json
FROM criterion_revisions WHERE namespace_id=? AND outcome_id=? AND criterion_id=? AND criterion_revision=1`, scope.NamespaceID.String(), scope.OutcomeID.String(), criterionID.String()); err != nil {
		t.Fatal(err)
	}

	assertRejectedUpdate := func(name, query string, args ...any) {
		t.Helper()
		dbtx, err := store.db.BeginTx(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := dbtx.ExecContext(ctx, `DELETE FROM criterion_current_assessments WHERE namespace_id=? AND outcome_id=? AND criterion_id=?`, scope.NamespaceID.String(), scope.OutcomeID.String(), criterionID.String()); err != nil {
			dbtx.Rollback()
			t.Fatal(err)
		}
		if _, err := dbtx.ExecContext(ctx, query, args...); err == nil {
			dbtx.Rollback()
			t.Fatalf("database accepted %s after TaskDelivery assessment binding", name)
		}
		if err := dbtx.Rollback(); err != nil {
			t.Fatal(err)
		}
	}
	assertRejectedUpdate("criterion owner change", `UPDATE criterion_assessments SET criterion_id=? WHERE namespace_id=? AND outcome_id=? AND id=?`, otherCriterionID.String(), scope.NamespaceID.String(), scope.OutcomeID.String(), assessmentID.String())
	assertRejectedUpdate("criterion revision change", `UPDATE criterion_assessments SET criterion_revision=2 WHERE namespace_id=? AND outcome_id=? AND id=?`, scope.NamespaceID.String(), scope.OutcomeID.String(), assessmentID.String())
	assertRejectedUpdate("TaskDelivery reference change", `UPDATE criterion_assessments SET task_delivery_round='2' WHERE namespace_id=? AND outcome_id=? AND id=?`, scope.NamespaceID.String(), scope.OutcomeID.String(), assessmentID.String())
	if _, err := store.db.ExecContext(ctx, `DELETE FROM criterion_revisions WHERE namespace_id=? AND outcome_id=? AND criterion_id=? AND criterion_revision=2`, scope.NamespaceID.String(), scope.OutcomeID.String(), criterionID.String()); err != nil {
		t.Fatal(err)
	}

	tx, err = store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	loaded, err = tx.WorkItems().Get(ctx, scope, workID)
	if err != nil {
		var coreError *d.Error
		if errors.As(err, &coreError) && coreError.Cause != nil {
			t.Fatalf("load Task after rejected assessment edits: %v (cause: %v)", err, coreError.Cause)
		}
		t.Fatal(err)
	}
	got = loaded.Criteria.CurrentAssessments[criterionID]
	if got.CriterionID != criterionID || got.CriterionRevision != 1 || got.TaskDeliveryRef == nil || *got.TaskDeliveryRef != *assessment.TaskDeliveryRef {
		t.Fatalf("failed mutations changed persisted assessment: %#v", got)
	}
}
