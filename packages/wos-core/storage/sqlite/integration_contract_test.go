package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"path/filepath"
	"testing"
	"time"
)

func TestDurableSignalsAtomicRollbackRestartAndDeliveryFencing(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "outbox.db")
	store := openTestStore(t, path)
	now := time.Now().UTC()
	s, _ := application.NewService(store, sqliteFixedClock{now: now}, &sqliteSequenceIDs{prefix: "0199d081", next: 1})
	cc := sqliteCommandContext("0199d082-0000-7000-8000-000000000001", "")
	ns := testID("0199d083-0000-7000-8000-000000000001")
	security := application.SecurityService{Store: store, Clock: sqliteFixedClock{now: now}, IDs: &sqliteSequenceIDs{prefix: "0199d096", next: 1}}
	bootstrap := "backup-operator-credential-32-characters"
	if err := security.Bootstrap(ctx, ports.Namespace{ID: ns, Name: "Restore scope"}, cc.PrincipalID, bootstrap); err != nil {
		t.Fatal(err)
	}
	identity, err := security.Authenticate(ctx, bootstrap)
	if err != nil {
		t.Fatal(err)
	}
	adminCtx := application.WithIdentity(ctx, identity)
	_, _, err = security.Administer(adminCtx, "backup-grant-receipt-0001", application.AdministrativeIntent{NamespaceID: ns, ExpectedVersion: 1, Operation: "set_grant", PrincipalID: "backup-agent", Permissions: []ports.Permission{ports.PermissionStateRead}})
	if err != nil {
		t.Fatal(err)
	}
	expires := now.Add(time.Hour)
	issueIntent := application.AdministrativeIntent{NamespaceID: ns, ExpectedVersion: 2, Operation: "issue_credential", PrincipalID: "backup-agent", Actor: &domain.ActorRef{Kind: domain.ActorKindAgent, Provider: "backup", ID: "agent"}, ExpiresAt: &expires}
	issued, credential, err := security.Administer(adminCtx, "backup-issue-receipt-0001", issueIntent)
	if err != nil {
		t.Fatal(err)
	}
	_, session, err := security.CreateSession(ctx, credential)
	if err != nil {
		t.Fatal(err)
	}
	outcome, err := s.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{NamespaceID: ns, Title: "Durable integration", DesiredState: "Signals survive consumers and restart", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	endpoint := ports.WebhookEndpoint{ID: testID("0199d084-0000-7000-8000-000000000001"), NamespaceID: ns, URL: "https://example.test/wos", SecretRef: "product-key", KeyID: "key-1"}
	if err = store.InstallEndpoints(ctx, []ports.WebhookEndpoint{endpoint}); err != nil {
		t.Fatal(err)
	}
	trigger, err := s.ConfigureTrigger(ctx, cc, application.ConfigureTriggerCommand{Scope: outcome.Value.Scope(), Name: "Inform consumer", EventTypes: []string{"work_item.created"}, Predicate: domain.EventPredicate{Field: "aggregate.kind", Op: "eq", Value: "work_item"}, TargetEndpointIDs: []domain.ID{endpoint.ID}, SignalType: "product.work_available"})
	if err != nil {
		t.Fatal(err)
	}
	cc.IdempotencyKey = "outbox-work-create-0001"
	command := application.CreateWorkItemCommand{Scope: outcome.Value.Scope(), Title: "Client chooses execution", Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleTodo}
	created, err := s.CreateWorkItem(ctx, cc, command)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.CreateWorkItem(ctx, cc, command)
	if err != nil || !replay.IdempotentReplay {
		t.Fatal("idempotent source retry failed", err)
	}
	deliveries, err := s.ListDeliveries(ctx, outcome.Value.Scope(), 10, "")
	if err != nil || len(deliveries.Items) != 1 {
		t.Fatalf("outbox duplicate or missing: %v %+v", err, deliveries)
	}
	var firing ports.TriggerFiring
	if err = json.Unmarshal(deliveries.Items[0].Body, &firing); err != nil {
		t.Fatal(err)
	}
	if firing.TriggerID != trigger.Value.ID || firing.Source.Entity.ID != created.Value.ID || firing.SchemaVersion != 1 {
		t.Fatal("incorrect stable integration envelope")
	}
	// The SQLite helper restores a consistent backup into an empty installation.
	// PostgreSQL uses restart; both preserve credentials, receipts and pending deliveries.
	store = reopenIntegrationFixture(t, store, path)
	security.Store = store
	if _, err = security.Authenticate(ctx, session); err != nil {
		t.Fatal("restored session/credential parent lost", err)
	}
	audit, err := security.AdministrativeSnapshot(adminCtx, ns)
	if err != nil || len(audit.Audit) != 2 || audit.NamespaceVersion != 3 {
		t.Fatal("restored grants/audit lost", err)
	}
	adminReplay, hidden, err := security.Administer(adminCtx, "backup-issue-receipt-0001", issueIntent)
	if err != nil || hidden != "" || !adminReplay.IdempotentReplay || adminReplay.Credential.ID != issued.Credential.ID {
		t.Fatal("restored administrative receipt lost", err)
	}

	s, _ = application.NewService(store, sqliteFixedClock{now: now}, &sqliteSequenceIDs{prefix: "0199d085", next: 1})
	restoredReplay, err := s.CreateWorkItem(ctx, cc, command)
	if err != nil || !restoredReplay.IdempotentReplay || restoredReplay.Value.ID != created.Value.ID {
		t.Fatal("restored source receipt lost", err)
	}
	restoredEvents, err := store.SnapshotDomainEvents(ctx, outcome.Value.Scope())
	if err != nil || len(restoredEvents) < 3 {
		t.Fatal("restored source history lost", err)
	}
	deliveries, err = s.ListDeliveries(ctx, outcome.Value.Scope(), 10, "")
	if err != nil || len(deliveries.Items) != 1 || deliveries.Items[0].Status != "pending" {
		t.Fatal("restart lost pending delivery", err)
	}
	leaseA, err := store.ClaimDelivery(ctx, now, testID("0199d086-0000-7000-8000-000000000001"), time.Second)
	if err != nil || leaseA == nil {
		t.Fatal("claim", err)
	}
	leaseB, err := store.ClaimDelivery(ctx, now.Add(2*time.Second), testID("0199d086-0000-7000-8000-000000000002"), time.Minute)
	if err != nil || leaseB == nil || leaseB.FencingToken <= leaseA.FencingToken {
		t.Fatal("reclaim did not fence old worker", err)
	}
	if err = store.FinishDelivery(ctx, *leaseA, now.Add(2*time.Second), "http_200", true, now); err == nil {
		t.Fatal("stale worker acknowledged delivery")
	}
	if err = store.FinishDelivery(ctx, *leaseB, now.Add(2*time.Second), "http_200", true, now); err != nil {
		t.Fatal(err)
	}
	endpoint2 := endpoint
	endpoint2.ID = testID("0199d084-0000-7000-8000-000000000002")
	if err = store.InstallEndpoints(ctx, []ports.WebhookEndpoint{endpoint2}); err != nil {
		t.Fatal(err)
	}
	cc.IdempotencyKey = ""
	if _, err = s.ConfigureTrigger(ctx, cc, application.ConfigureTriggerCommand{Scope: outcome.Value.Scope(), Name: "Failure injection target", EventTypes: []string{"work_item.created"}, TargetEndpointIDs: []domain.ID{endpoint2.ID}, SignalType: "product.work_available"}); err != nil {
		t.Fatal(err)
	}
	// Removing a configured target after validation injects a failure between the
	// source mutation/event and delivery persistence in the same transaction.
	if _, err = store.db.ExecContext(ctx, `DELETE FROM webhook_endpoints WHERE id=?`, endpoint2.ID.String()); err != nil {
		t.Fatal(err)
	}
	before, err := store.SnapshotDomainEvents(ctx, outcome.Value.Scope())
	if err != nil {
		t.Fatal(err)
	}
	cc.IdempotencyKey = "outbox-failed-create-0001"
	command.Title = "Must roll back"
	if _, err = s.CreateWorkItem(ctx, cc, command); err == nil {
		t.Fatal("expected outbox persistence failure")
	}
	after, _ := store.SnapshotDomainEvents(ctx, outcome.Value.Scope())
	if len(before) != len(after) {
		t.Fatal("failed outbox left source event")
	}
	work, err := s.GetOutcomeState(ctx, outcome.Value.Scope())
	if err != nil || len(work.WorkItems) != 1 {
		t.Fatal("failed outbox left source work", err)
	}
	deliveries, err = s.ListDeliveries(ctx, outcome.Value.Scope(), 10, "")
	if err != nil || len(deliveries.Items) != 1 {
		t.Fatal("failed outbox left partial delivery", err)
	}
	if err = store.InstallEndpoints(ctx, []ports.WebhookEndpoint{endpoint2}); err != nil {
		t.Fatal(err)
	}
	retried, err := s.CreateWorkItem(ctx, cc, command)
	if err != nil || retried.IdempotentReplay {
		t.Fatal("rolled-back idempotency reservation survived", err)
	}
	deliveries, err = s.ListDeliveries(ctx, outcome.Value.Scope(), 10, "")
	if err != nil || len(deliveries.Items) != 3 {
		t.Fatal("retry did not commit both target signals", err)
	}
}

func TestDeliveryCrashBudgetRequiresExplicitRedelivery(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "crash-budget.db")
	store := openTestStore(t, path)
	now := time.Now().UTC()
	s, _ := application.NewService(store, sqliteFixedClock{now: now}, &sqliteSequenceIDs{prefix: "0199d097", next: 1})
	cc := sqliteCommandContext("0199d097-0000-7000-8000-000000000011", "")
	created, err := s.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{NamespaceID: testID("0199d097-0000-7000-8000-000000000012"), Title: "Bounded crashed workers", DesiredState: "Delivery eventually exhausts", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	scope := created.Value.Scope()
	endpoint := ports.WebhookEndpoint{ID: testID("0199d097-0000-7000-8000-000000000013"), NamespaceID: scope.NamespaceID, URL: "https://example.test/webhook", SecretRef: "key", KeyID: "key-1"}
	if err = store.InstallEndpoints(ctx, []ports.WebhookEndpoint{endpoint}); err != nil {
		t.Fatal(err)
	}
	_, err = s.ConfigureTrigger(ctx, cc, application.ConfigureTriggerCommand{Scope: scope, Name: "Crash budget", EventTypes: []string{"work_item.created"}, TargetEndpointIDs: []domain.ID{endpoint.ID}, SignalType: "product.work"})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{Scope: scope, Title: "Signal", Priority: domain.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	var last *ports.Delivery
	for i := 1; i <= 6; i++ {
		last, err = store.ClaimDelivery(ctx, now.Add(time.Duration(i)*time.Second), testID(fmt.Sprintf("0199d097-0000-7000-8000-%012d", 100+i)), time.Millisecond)
		if err != nil || last == nil || last.Attempts != i {
			t.Fatalf("crash attempt %d: %+v %v", i, last, err)
		}
	}
	next, err := store.ClaimDelivery(ctx, now.Add(8*time.Second), testID("0199d097-0000-7000-8000-000000000200"), time.Minute)
	if err != nil || next != nil {
		t.Fatal("crash retries exceeded budget", err)
	}
	store.Close()
	store = openTestStore(t, path)
	s, _ = application.NewService(store, sqliteFixedClock{now: now}, &sqliteSequenceIDs{prefix: "0199d095", next: 1})
	page, err := s.ListDeliveries(ctx, scope, 10, "")
	if err != nil || len(page.Items) != 1 || page.Items[0].Status != "exhausted" || page.Items[0].Attempts != 6 {
		t.Fatal("exhaustion recovery rolled back or restart lost it", err)
	}
	if err = store.FinishDelivery(ctx, *last, now.Add(8*time.Second), "late_200", true, now); err == nil {
		t.Fatal("expired worker resurrected exhausted delivery")
	}
	cc.IdempotencyKey = "explicit-redelivery-receipt-001"
	redelivery := application.RedeliverDeliveryCommand{Scope: scope, ExpectedVersion: created.Value.Version, DeliveryID: last.ID, Reason: "Operator retries after repairing consumer"}
	result, err := s.RedeliverDelivery(ctx, cc, redelivery)
	if err != nil {
		t.Fatal(err)
	}
	replay, err := s.RedeliverDelivery(ctx, cc, redelivery)
	if err != nil || !replay.IdempotentReplay || result.Value.Version != replay.Value.Version {
		t.Fatal("redelivery replay duplicated intent", err)
	}
	next, err = store.ClaimDelivery(ctx, now, testID("0199d097-0000-7000-8000-000000000201"), time.Minute)
	if err != nil || next == nil || next.Attempts != 1 || next.IntegrationEventID != last.IntegrationEventID || next.FencingToken <= last.FencingToken {
		t.Fatal("manual redelivery lost stable identity, reset or fencing", err)
	}
}
