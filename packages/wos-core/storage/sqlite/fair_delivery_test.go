package sqlite

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func fairDeliveryFixture(t *testing.T) (*Store, time.Time, []domain.Scope) {
	t.Helper()
	store := openTestStore(t, filepath.Join(t.TempDir(), "fair.db"))
	now := time.Now().UTC()
	service, err := application.NewService(store, sqliteFixedClock{now: now}, &sqliteSequenceIDs{prefix: "0199df01", next: 1})
	if err != nil {
		t.Fatal(err)
	}
	cc := sqliteCommandContext("0199df02-0000-7000-8000-000000000001", "")
	var scopes []domain.Scope
	for i := 1; i <= 2; i++ {
		ns := testID(fmt.Sprintf("0199df03-0000-7000-8000-%012d", i))
		out, e := service.CreateOutcome(context.Background(), cc, application.CreateOutcomeCommand{TenantID: ns, Title: fmt.Sprintf("Tenant %d", i), DesiredState: "Delivery fairness", Priority: domain.PriorityNormal})
		if e != nil {
			t.Fatal(e)
		}
		scope := out.Value.Scope()
		scopes = append(scopes, scope)
		var endpoints []ports.WebhookEndpoint
		count := 1
		if i == 1 {
			count = 2
		}
		var endpointIDs []domain.ID
		for j := 1; j <= count; j++ {
			id := testID(fmt.Sprintf("0199df04-0000-7000-8000-%012d", i*10+j))
			endpointIDs = append(endpointIDs, id)
			endpoints = append(endpoints, ports.WebhookEndpoint{ID: id, TenantID: ns, URL: "https://example.test/hooks", SecretRef: fmt.Sprintf("tenant-%d", i), KeyID: "key"})
		}
		if e = store.InstallEndpoints(context.Background(), endpoints); e != nil {
			t.Fatal(e)
		}
		_, e = service.ConfigureTrigger(context.Background(), cc, application.ConfigureTriggerCommand{Scope: scope, Name: "Delivery", EventTypes: []string{"work_item.created"}, TargetEndpointIDs: endpointIDs, SignalType: "work.created"})
		if e != nil {
			t.Fatal(e)
		}
		for k := 0; k < 4; k++ {
			_, e = service.CreateWorkItem(context.Background(), cc, application.CreateWorkItemCommand{Scope: scope, Title: "Pending work", Priority: domain.PriorityNormal, Lifecycle: domain.WorkItemLifecycleTodo})
			if e != nil {
				t.Fatal(e)
			}
		}
	}
	return store, now, scopes
}

func TestDeliveryTenantTurnsSharedCapsAndSecretEligibility(t *testing.T) {
	store, now, scopes := fairDeliveryFixture(t)
	ctx := context.Background()
	policy := ports.DeliveryClaimPolicy{MaxInflightPerTenant: 1, MaxInflightPerEndpoint: 1, SecretRefs: []string{"tenant-1", "tenant-2"}}
	claim := func(index int, p ports.DeliveryClaimPolicy) *ports.Delivery {
		t.Helper()
		v, e := store.ClaimDeliveryWithPolicy(ctx, now, testID(fmt.Sprintf("0199df05-0000-7000-8000-%012d", index)), time.Minute, p)
		if e != nil {
			t.Fatal(e)
		}
		return v
	}
	if v := claim(1, ports.DeliveryClaimPolicy{SecretRefs: []string{}}); v != nil {
		t.Fatal("worker without eligible secrets consumed a retry")
	}
	first, second := claim(2, policy), claim(3, policy)
	if first == nil || second == nil || first.Scope.TenantID == second.Scope.TenantID {
		t.Fatalf("independent tenants did not get capacity: %+v %+v", first, second)
	}
	if claim(4, policy) != nil {
		t.Fatal("shared Tenant cap exceeded")
	}
	if err := store.FinishDelivery(ctx, *second, now, "http_200", true, now); err != nil {
		t.Fatal(err)
	}
	if next := claim(5, policy); next == nil || next.Scope.TenantID != second.Scope.TenantID {
		t.Fatal("saturated tenant blocked another tenant")
	}
	// Inspect the queue, not a process-local counter: no ineligible attempt ran.
	var attempts int
	if err := store.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM delivery_attempts`).Scan(&attempts); err != nil || attempts != 3 {
		t.Fatalf("attempt history changed unexpectedly: %d %v", attempts, err)
	}
	_ = scopes
}

func TestDeliveryEndpointCapsAndConcurrentClaimers(t *testing.T) {
	store, now, scopes := fairDeliveryFixture(t)
	ctx := context.Background()
	policy := ports.DeliveryClaimPolicy{MaxInflightPerTenant: 2, MaxInflightPerEndpoint: 1, SecretRefs: []string{"tenant-1"}}
	var mu sync.Mutex
	var leased []*ports.Delivery
	var wg sync.WaitGroup
	for i := 1; i <= 8; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			v, e := store.ClaimDeliveryWithPolicy(ctx, now, testID(fmt.Sprintf("0199df06-0000-7000-8000-%012d", i)), time.Minute, policy)
			if e != nil {
				t.Error(e)
				return
			}
			if v != nil {
				mu.Lock()
				leased = append(leased, v)
				mu.Unlock()
			}
		}(i)
	}
	wg.Wait()
	if len(leased) > 2 {
		t.Fatalf("shared limits failed: %+v", leased)
	}
	// SKIP LOCKED may let simultaneous claimers observe the only scheduler row
	// busy and return no work. Once they finish, fill remaining capacity without
	// retrying any failed or uncertain claim.
	for len(leased) < 2 {
		v, err := store.ClaimDeliveryWithPolicy(ctx, now, testID(fmt.Sprintf("0199df06-0000-7000-8000-%012d", 100+len(leased))), time.Minute, policy)
		if err != nil || v == nil {
			t.Fatalf("available endpoint capacity not claimable: %+v %v", v, err)
		}
		leased = append(leased, v)
	}
	if leased[0].EndpointID == leased[1].EndpointID {
		t.Fatal("shared endpoint cap exceeded")
	}
	for _, v := range leased {
		if v.Scope != scopes[0] {
			t.Fatal("worker claimed a secret it does not possess")
		}
	}
	// Expiry releases capacity and the next claim fences a crashed worker.
	newLease, e := store.ClaimDeliveryWithPolicy(ctx, now.Add(2*time.Minute), testID("0199df07-0000-7000-8000-000000000001"), time.Minute, policy)
	if e != nil || newLease == nil {
		t.Fatalf("expired capacity did not recover: %+v %v", newLease, e)
	}
	if e = store.FinishDelivery(ctx, *leased[0], now.Add(2*time.Minute), "http_200", true, now); e == nil {
		t.Fatal("expired lease acknowledged an external effect")
	}
}
