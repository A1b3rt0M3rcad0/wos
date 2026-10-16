package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/postgres"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/storage/sqlite"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type deliveryFixtureStore interface {
	ports.TransactionManager
	ports.DeliveryStore
	InstallEndpoints(context.Context, []ports.WebhookEndpoint) error
	Close() error
}
type loadIDs struct{ next atomic.Uint64 }

func (g *loadIDs) NewID() (d.ID, error) {
	return d.ParseID(fmt.Sprintf("0199d170-0000-7000-8000-%012x", g.next.Add(1)))
}

type dropOneACK struct {
	ports.DeliveryStore
	once atomic.Bool
}

func (s *dropOneACK) FinishDelivery(ctx context.Context, v ports.Delivery, now time.Time, result string, success bool, next time.Time) error {
	if success && s.once.CompareAndSwap(false, true) {
		return d.NewError(d.ErrorCodeTransactionConflict, "injected ACK loss after receiver side effect")
	}
	return s.DeliveryStore.FinishDelivery(ctx, v, now, result, success, next)
}
func (s *dropOneACK) ClaimDelivery(ctx context.Context, now time.Time, id d.ID, _ time.Duration) (*ports.Delivery, error) {
	return s.DeliveryStore.ClaimDelivery(ctx, now, id, 500*time.Millisecond)
}
func openDeliveryLoadStores(t *testing.T, backend string) (deliveryFixtureStore, deliveryFixtureStore) {
	t.Helper()
	if backend == "sqlite" {
		path := filepath.Join(t.TempDir(), "queue.db")
		one, e := sqlite.Open(path, sqlite.Options{MigrateOnOpen: true})
		if e != nil {
			t.Fatal(e)
		}
		two, e := sqlite.Open(path, sqlite.Options{})
		if e != nil {
			t.Fatal(e)
		}
		t.Cleanup(func() { two.Close(); one.Close() })
		return one, two
	}
	dsn := os.Getenv("WOS_TEST_POSTGRES_DSN")
	if dsn == "" {
		t.Skip("real PostgreSQL required")
	}
	schema := fmt.Sprintf("delivery_load_%d", time.Now().UnixNano())
	one, e := postgres.Open(dsn, postgres.Options{MigrateOnOpen: true, Schema: schema})
	if e != nil {
		t.Fatal(e)
	}
	two, e := postgres.Open(dsn, postgres.Options{Schema: schema})
	if e != nil {
		one.Close()
		t.Fatal(e)
	}
	t.Cleanup(func() {
		two.Close()
		one.Close()
		db, e := sql.Open("pgx", dsn)
		if e != nil {
			t.Error(e)
			return
		}
		defer db.Close()
		if _, e = db.ExecContext(context.Background(), "DROP SCHEMA "+schema+" CASCADE"); e != nil {
			t.Error(e)
		}
	})
	return one, two
}
func seedDeliveryLoad(t *testing.T, store deliveryFixtureStore, url string, n int, ids *loadIDs) ([]d.Scope, time.Time) {
	t.Helper()
	ctx := context.Background()
	service, e := a.NewService(store, diagnosticClock{}, ids)
	if e != nil {
		t.Fatal(e)
	}
	ns := d.MustParseID("0199d171-0000-7000-8000-000000000001")
	endpoint := ports.WebhookEndpoint{ID: d.MustParseID("0199d171-0000-7000-8000-000000000002"), NamespaceID: ns, URL: url, SecretRef: "load", KeyID: "hmac-load"}
	if e = store.InstallEndpoints(ctx, []ports.WebhookEndpoint{endpoint}); e != nil {
		t.Fatal(e)
	}
	cc := d.CommandContext{PrincipalID: "fixture", Actor: d.ActorRef{Kind: d.ActorKindService, Provider: "test", ID: "fixture"}}
	scopes := []d.Scope{}
	for i := 0; i < 4; i++ {
		cc.CommandID, _ = ids.NewID()
		out, e := service.CreateOutcome(ctx, cc, a.CreateOutcomeCommand{NamespaceID: ns, Title: fmt.Sprint("Outcome ", i), DesiredState: "signals delivered", Priority: d.PriorityNormal})
		if e != nil {
			t.Fatal(e)
		}
		scopes = append(scopes, out.Value.Scope())
		cc.CommandID, _ = ids.NewID()
		_, e = service.ConfigureTrigger(ctx, cc, a.ConfigureTriggerCommand{Scope: out.Value.Scope(), Name: "signal", EventTypes: []string{"work_item.created"}, TargetEndpointIDs: []d.ID{endpoint.ID}, SignalType: "load.work"})
		if e != nil {
			t.Fatal(e)
		}
	}
	for i := 0; i < n; i++ {
		cc.CommandID, _ = ids.NewID()
		_, e = service.CreateWorkItem(ctx, cc, a.CreateWorkItemCommand{Scope: scopes[i%4], Title: fmt.Sprint("Signal ", i), Priority: d.PriorityNormal})
		if e != nil {
			t.Fatal(e)
		}
	}
	return scopes, time.Now()
}
func TestDeliveryBoundedConcurrencyReplicaFairnessAndACKLoss(t *testing.T) {
	n := 100
	if v := os.Getenv("WOS_DELIVERY_LOAD_MESSAGES"); v != "" {
		var e error
		n, e = strconv.Atoi(v)
		if e != nil || n < 100 || n > 1000 {
			t.Fatal("load messages must be 100..1000")
		}
	}
	for _, backend := range []string{"sqlite", "postgres"} {
		for _, topology := range []struct{ replicas, lanes int }{{1, 1}, {1, 4}, {2, 2}} {
			t.Run(fmt.Sprintf("%s/%dx%d/%d", backend, topology.replicas, topology.lanes, n), func(t *testing.T) {
				one, two := openDeliveryLoadStores(t, backend)
				ids := &loadIDs{}
				secret := strings.Repeat("s", 40)
				var mu sync.Mutex
				unique := map[string]int{}
				seenScopes := map[d.ID]bool{}
				latencies := []time.Duration{}
				var inflight, maxInflight atomic.Int64
				var lastReceive time.Time
				var callbackErr string
				started := time.Now()
				receiver := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					current := inflight.Add(1)
					defer inflight.Add(-1)
					for old := maxInflight.Load(); current > old && !maxInflight.CompareAndSwap(old, current); old = maxInflight.Load() {
					}
					body, e := io.ReadAll(io.LimitReader(r.Body, 256<<10))
					if e == nil {
						e = VerifySignature(secret, r.Header.Get("X-WOS-Timestamp"), r.Header.Get("X-WOS-Event-ID"), r.Header.Get("X-WOS-Signature"), body, time.Now())
					}
					var signal ports.TriggerFiring
					if e == nil {
						e = json.Unmarshal(body, &signal)
					}
					if e != nil {
						mu.Lock()
						callbackErr = e.Error()
						mu.Unlock()
						w.WriteHeader(400)
						return
					}
					time.Sleep(2 * time.Millisecond) // isolated receiver backpressure, not external availability
					mu.Lock()
					unique[signal.ID.String()]++
					seenScopes[signal.Source.OutcomeID] = true
					latencies = append(latencies, time.Since(started))
					lastReceive = time.Now()
					mu.Unlock()
					w.WriteHeader(204)
				}))
				defer receiver.Close()
				scopes, seeded := seedDeliveryLoad(t, one, receiver.URL, n, ids)
				mu.Lock()
				started = seeded
				mu.Unlock()
				// First remote side effect succeeds but its authoritative ACK is deliberately
				// lost. The expired persisted lease is retried under the same integration ID.
				lost := &dropOneACK{DeliveryStore: one}
				injector, e := New(lost, diagnosticClock{}, ids, map[string]string{"load": secret}, Policy{AllowedHosts: []string{"127.0.0.1"}, AllowLoopback: true})
				if e != nil {
					t.Fatal(e)
				}
				if worked, e := injector.Tick(context.Background()); !worked || e == nil || !lost.once.Load() {
					t.Fatal("ACK loss was not exercised", worked, e)
				}
				stores := []ports.DeliveryStore{lost, two}
				ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
				defer cancel()
				done := make(chan error, topology.replicas)
				var lanes sync.WaitGroup
				defer func() { cancel(); lanes.Wait() }()
				for i := 0; i < topology.replicas; i++ {
					worker, e := New(stores[i], diagnosticClock{}, ids, map[string]string{"load": secret}, Policy{AllowedHosts: []string{"127.0.0.1"}, AllowLoopback: true})
					if e != nil {
						t.Fatal(e)
					}
					worker.Concurrency = topology.lanes
					worker.Diagnostics = NewDiagnostics(true)
					lanes.Add(1)
					go func() { defer lanes.Done(); done <- worker.Run(ctx) }()
				}
				ticker := time.NewTicker(20 * time.Millisecond)
				defer ticker.Stop()
				maxBacklog := int64(0)
				for {
					u, e := one.Begin(ctx)
					if e != nil {
						t.Fatal(e)
					}
					health, e := u.(ports.DeliveryHealthUnitOfWork).DeliveryHealth().Health(ctx, d.Scope{NamespaceID: scopes[0].NamespaceID}, time.Now())
					u.Rollback()
					if e != nil {
						t.Fatal(e)
					}
					maxBacklog = max(maxBacklog, health.Pending+health.Leased)
					if health.Succeeded == int64(n) {
						if health.AttemptsTotal < int64(n+1) {
							t.Fatal("lost ACK did not remain in durable history")
						}
						break
					}
					select {
					case <-ctx.Done():
						t.Fatal("backlog not drained", health)
					case e := <-done:
						t.Fatalf("worker exited before drain: %v", e)
					case <-ticker.C:
					}
				}
				cancel()
				for i := 0; i < topology.replicas; i++ {
					if e := <-done; e != nil {
						t.Fatal(e)
					}
				}
				explainDeliveryQueue(t, backend, one, scopes[0])
				mu.Lock()
				defer mu.Unlock()
				if callbackErr != "" || len(unique) != n || len(seenScopes) != 4 {
					t.Fatalf("HMAC/dedupe/starvation: unique=%d scopes=%d error=%s", len(unique), len(seenScopes), callbackErr)
				}
				duplicates := 0
				for _, count := range unique {
					duplicates += count - 1
				}
				if duplicates < 1 {
					t.Fatal("external duplicate after lost ACK not exercised")
				}
				if maxInflight.Load() > int64(topology.replicas*topology.lanes) {
					t.Fatal("unbounded request prefetch")
				}
				sort.Slice(latencies, func(i, j int) bool { return latencies[i] < latencies[j] })
				report := map[string]any{"backend": backend, "replicas": topology.replicas, "lanes_per_replica": topology.lanes, "messages": n, "max_backlog": maxBacklog, "unique_receiver_effects_with_dedupe": len(unique), "external_duplicates": duplicates, "max_inflight": maxInflight.Load(), "elapsed_ms": lastReceive.Sub(seeded).Milliseconds(), "throughput_per_second": float64(n) / lastReceive.Sub(seeded).Seconds(), "queue_to_receiver_p95_ms": latencies[(len(latencies)*95+99)/100-1].Milliseconds(), "guarantee": "at_least_once"}
				raw, _ := json.Marshal(report)
				t.Log(string(raw))
			})
		}
	}
}

// EXPLAIN runs only in this disposable, owned fixture. The DROP is rolled back:
// it compares the actual pre-index plan with the scoped index on the same rows.
func explainDeliveryQueue(t *testing.T, backend string, store deliveryFixtureStore, scope d.Scope) {
	t.Helper()
	var db *sql.DB
	var e error
	query := "SELECT COUNT(*),MIN(next_attempt) FROM integration_deliveries WHERE namespace_id=? AND outcome_id=?"
	if backend == "sqlite" {
		db, e = sql.Open("sqlite3", store.(*sqlite.Store).Path())
	} else {
		db, e = sql.Open("pgx", os.Getenv("WOS_TEST_POSTGRES_DSN"))
		if e == nil {
			db.SetMaxOpenConns(1)
			_, e = db.Exec("SET search_path TO " + store.(*postgres.Store).Path())
		}
		query = strings.Replace(query, "?", "$1", 1)
		query = strings.Replace(query, "?", "$2", 1)
	}
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	// Queue ACK updates leave dead tuples. Record a maintained PostgreSQL plan
	// as well as the earlier unvacuumed load runs; do not force planner switches.
	if backend == "postgres" {
		if _, e = db.Exec("VACUUM (ANALYZE) integration_deliveries"); e != nil {
			t.Fatal(e)
		}
	}
	tx, e := db.Begin()
	if e != nil {
		t.Fatal(e)
	}
	defer tx.Rollback()
	if backend == "postgres" {
		if _, e = tx.Exec("ANALYZE integration_deliveries"); e != nil {
			t.Fatal(e)
		}
	}
	explain := func(label string) {
		prefix := "EXPLAIN QUERY PLAN "
		if backend == "postgres" {
			prefix = "EXPLAIN (ANALYZE,BUFFERS,FORMAT TEXT) "
		}
		rows, e := tx.Query(prefix+query, scope.NamespaceID.String(), scope.OutcomeID.String())
		if e != nil {
			t.Fatal(e)
		}
		defer rows.Close()
		parts := []string{}
		for rows.Next() {
			var line string
			if backend == "sqlite" {
				var a, b, c int
				e = rows.Scan(&a, &b, &c, &line)
			} else {
				e = rows.Scan(&line)
			}
			if e != nil {
				t.Fatal(e)
			}
			parts = append(parts, line)
		}
		if e = rows.Err(); e != nil {
			t.Fatal(e)
		}
		raw, _ := json.Marshal(map[string]any{"queue_index_plan": label, "backend": backend, "lines": parts})
		t.Log(string(raw))
	}
	explain("with_scope_index")
	if _, e = tx.Exec("DROP INDEX delivery_scope_health"); e != nil {
		t.Fatal(e)
	}
	explain("without_scope_index")
	if e = tx.Rollback(); e != nil {
		t.Fatal(e)
	}
}
