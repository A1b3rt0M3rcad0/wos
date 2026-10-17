package integration

import (
	"bytes"
	"context"
	"encoding/json"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestDeliveryWorkerCrashChild(t *testing.T) {
	if os.Getenv("WOS_DELIVERY_CRASH_CHILD") != "1" {
		t.Skip("owned child process only")
	}
	var store deliveryFixtureStore
	var e error
	if os.Getenv("WOS_DELIVERY_CRASH_BACKEND") == "sqlite" {
		store, e = sqlite.Open(os.Getenv("WOS_DELIVERY_CRASH_PATH"), sqlite.Options{})
	} else {
		store, e = postgres.Open(os.Getenv("WOS_TEST_POSTGRES_DSN"), postgres.Options{Schema: os.Getenv("WOS_DELIVERY_CRASH_PATH")})
	}
	if e != nil {
		t.Fatal(e)
	}
	defer store.Close()
	lease := &dropOneACK{DeliveryStore: store}
	lease.once.Store(true)
	worker, e := New(lease, diagnosticClock{}, &loadIDs{}, map[string]string{"load": strings.Repeat("s", 40)}, Policy{AllowedHosts: []string{"127.0.0.1"}, AllowLoopback: true})
	if e != nil {
		t.Fatal(e)
	}
	_, e = worker.Tick(context.Background())
	t.Fatalf("child unexpectedly survived the held receiver: %v", e)
}
func TestDeliveryWorkerProcessDeathReclaimsPersistedLease(t *testing.T) {
	for _, backend := range []string{"sqlite", "postgres"} {
		t.Run(backend, func(t *testing.T) {
			one, two := openDeliveryLoadStores(t, backend)
			ids := &loadIDs{}
			secret := strings.Repeat("s", 40)
			received := make(chan ports.TriggerFiring, 1)
			var mu sync.Mutex
			seen := map[string]int{}
			first := true
			receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, e := io.ReadAll(r.Body)
				if e == nil {
					e = VerifySignature(secret, r.Header.Get("X-WOS-Timestamp"), r.Header.Get("X-WOS-Event-ID"), r.Header.Get("X-WOS-Signature"), body, time.Now())
				}
				if e != nil {
					w.WriteHeader(400)
					return
				}
				var signal ports.TriggerFiring
				if e = json.Unmarshal(body, &signal); e != nil {
					w.WriteHeader(400)
					return
				}
				mu.Lock()
				seen[signal.ID.String()]++
				held := first
				first = false
				mu.Unlock()
				if held {
					received <- signal
					<-r.Context().Done()
					return
				}
				w.WriteHeader(204)
			}))
			defer receiver.Close()
			seedDeliveryLoad(t, one, receiver.URL, 4, ids)
			path := ""
			switch s := one.(type) {
			case *sqlite.Store:
				path = s.Path()
			case *postgres.Store:
				path = s.Path()
			}
			child := exec.Command(os.Args[0], "-test.run=^TestDeliveryWorkerCrashChild$", "-test.v", "-test.timeout=20s")
			child.Env = append(os.Environ(), "WOS_DELIVERY_CRASH_CHILD=1", "WOS_DELIVERY_CRASH_BACKEND="+backend, "WOS_DELIVERY_CRASH_PATH="+path)
			var output bytes.Buffer
			child.Stdout = &output
			child.Stderr = &output
			if e := child.Start(); e != nil {
				t.Fatal(e)
			}
			waited := false
			defer func() {
				if !waited {
					child.Process.Kill()
					child.Wait()
				}
			}()
			var signal ports.TriggerFiring
			select {
			case signal = <-received:
			case <-time.After(10 * time.Second):
				t.Fatal("child did not persist lease and reach receiver", output.String())
			}
			scope := d.Scope{TenantID: signal.Source.TenantID, OutcomeID: signal.Source.OutcomeID}
			u, e := one.Begin(context.Background())
			if e != nil {
				t.Fatal(e)
			}
			page, e := u.(ports.IntegrationUnitOfWork).Deliveries().List(context.Background(), scope, 10, "")
			u.Rollback()
			if e != nil || len(page) != 1 || page[0].Status != "leased" {
				t.Fatal("child lease absent", page, e)
			}
			original := page[0]
			if e = child.Process.Kill(); e != nil {
				t.Fatal(e)
			}
			e = child.Wait()
			waited = true
			if e == nil {
				t.Fatal("child was not killed")
			}
			// Wait for the real persisted deadline; no clock/row/signature is altered.
			deadline := time.NewTimer(time.Until(original.LeaseUntil.Add(30 * time.Millisecond)))
			<-deadline.C
			replacement := &dropOneACK{DeliveryStore: two}
			replacement.once.Store(true)
			worker, e := New(replacement, diagnosticClock{}, ids, map[string]string{"load": secret}, Policy{AllowedHosts: []string{"127.0.0.1"}, AllowLoopback: true})
			if e != nil {
				t.Fatal(e)
			}
			// Other pending Outcomes can precede this reclaimed row; drain the bounded fixture.
			for i := 0; i < 5; i++ {
				worked, e := worker.Tick(context.Background())
				if e != nil {
					t.Fatal(e)
				}
				if !worked {
					break
				}
			}
			if e = one.FinishDelivery(context.Background(), original, time.Now(), "late_204", true, time.Now()); e == nil {
				t.Fatal("killed worker ACK was not fenced")
			}
			u, e = one.Begin(context.Background())
			if e != nil {
				t.Fatal(e)
			}
			page, e = u.(ports.IntegrationUnitOfWork).Deliveries().List(context.Background(), scope, 10, "")
			health, he := u.(ports.DeliveryHealthUnitOfWork).DeliveryHealth().Health(context.Background(), scope, time.Now())
			u.Rollback()
			if e != nil || he != nil || len(page) != 1 || page[0].Status != "succeeded" || page[0].FencingToken <= original.FencingToken || page[0].IntegrationEventID != original.IntegrationEventID || health.AttemptsTotal != 2 {
				t.Fatal("crash recovery lost queue authority", page, health, e, he)
			}
			mu.Lock()
			defer mu.Unlock()
			if seen[signal.ID.String()] != 2 {
				t.Fatal("duplicate external side effect not demonstrated", seen)
			}
			t.Logf("backend=%s real_process_killed=true attempts=2 authoritative_successes=1 receiver_requests=2 deduplicated_effects=1 fence=%d->%d", backend, original.FencingToken, page[0].FencingToken)
		})
	}
}
