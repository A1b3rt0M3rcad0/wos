package postgres

import (
	"context"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"path/filepath"
	"testing"
	"time"
)

// A slow ACK/recovery on one row must not prevent another lane from reclaiming
// an unrelated expired lease. Lock parent before attempt, as FinishDelivery does.
func TestDeliveryRecoverySkipsLockedExpiredDelivery(t *testing.T) {
	requirePostgres(t)
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "recovery-lock.db"))
	now := time.Now().UTC()
	ids := &sqliteSequenceIDs{prefix: "0199d198", next: 1}
	service, err := a.NewService(store, sqliteFixedClock{now: now}, ids)
	if err != nil {
		t.Fatal(err)
	}
	cc := sqliteCommandContext("0199d198-0000-7000-8000-000000000011", "")
	out, err := service.CreateOutcome(ctx, cc, a.CreateOutcomeCommand{NamespaceID: testID("0199d198-0000-7000-8000-000000000012"), Title: "Independent recovery", DesiredState: "No queue-wide recovery lock", Priority: d.PriorityNormal})
	if err != nil {
		t.Fatal(err)
	}
	endpoint := ports.WebhookEndpoint{ID: testID("0199d198-0000-7000-8000-000000000013"), NamespaceID: out.Value.NamespaceID, URL: "https://example.test/recovery", SecretRef: "test", KeyID: "test"}
	if err = store.InstallEndpoints(ctx, []ports.WebhookEndpoint{endpoint}); err != nil {
		t.Fatal(err)
	}
	if _, err = service.ConfigureTrigger(ctx, cc, a.ConfigureTriggerCommand{Scope: out.Value.Scope(), Name: "recovery", EventTypes: []string{"work_item.created"}, TargetEndpointIDs: []d.ID{endpoint.ID}, SignalType: "recovery.work"}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if _, err = service.CreateWorkItem(ctx, cc, a.CreateWorkItemCommand{Scope: out.Value.Scope(), ObjectiveID: postgresObjectiveForWorkFixture(t, service, out.Value.Scope()), Title: "signal", Priority: d.PriorityNormal}); err != nil {
			t.Fatal(err)
		}
	}
	first, err := store.ClaimDelivery(ctx, now, testID("0199d198-0000-7000-8000-000000000101"), time.Second)
	if err != nil || first == nil {
		t.Fatal(first, err)
	}
	second, err := store.ClaimDelivery(ctx, now, testID("0199d198-0000-7000-8000-000000000102"), time.Second)
	if err != nil || second == nil {
		t.Fatal(second, err)
	}
	locked, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer locked.Rollback()
	if _, err = locked.ExecContext(ctx, "SELECT id FROM integration_deliveries WHERE id=$1 FOR UPDATE", first.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = locked.ExecContext(ctx, "UPDATE delivery_attempts SET result=result WHERE delivery_id=$1", first.ID); err != nil {
		t.Fatal(err)
	}
	bounded, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	reclaimed, err := store.ClaimDelivery(bounded, now.Add(2*time.Second), testID("0199d198-0000-7000-8000-000000000103"), time.Minute)
	if err != nil || reclaimed == nil || reclaimed.ID != second.ID || reclaimed.FencingToken != second.FencingToken+1 {
		t.Fatalf("unrelated locked lease stalled recovery: %+v %v", reclaimed, err)
	}
	if err = store.FinishDelivery(ctx, *second, now.Add(2*time.Second), "stale", true, now); err == nil {
		t.Fatal("old fence accepted")
	}
	if err = store.FinishDelivery(ctx, *reclaimed, now.Add(2*time.Second), "http_204", true, now); err != nil {
		t.Fatal(err)
	}
}
