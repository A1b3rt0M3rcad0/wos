package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	a "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	sqlite3 "github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type loadIDs struct{ next atomic.Uint64 }

func (g *loadIDs) NewID() (d.ID, error) {
	return d.ParseID(fmt.Sprintf("0199ff80-0000-7000-8000-%012x", g.next.Add(1)))
}

type loadObserver struct {
	mu    sync.Mutex
	items []ports.CommandObservation
}

func (o *loadObserver) ObserveCommand(v ports.CommandObservation) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.items = append(o.items, v)
}
func loadQuantile(values []float64, q float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sort.Float64s(values)
	return values[int(float64(len(values)-1)*q)]
}
func TestWorkContractLoadMatrix(t *testing.T) {
	if os.Getenv("WOS_CONTRACT_LOAD") != "1" {
		t.Skip("opt-in measured fixture; set WOS_CONTRACT_LOAD=1")
	}
	for _, workCount := range []int{100, 1000, 10000} {
		t.Run(fmt.Sprint(workCount), func(t *testing.T) {
			ctx := context.Background()
			store := openTestStore(t, filepath.Join(t.TempDir(), "load.db"))
			now := time.Now().UTC()
			generator := &loadIDs{}
			service, err := a.NewService(store, sqliteFixedClock{now}, generator)
			if err != nil {
				t.Fatal(err)
			}
			w := contractStorageFixture(t, store, now)
			workIDs := []d.ID{w.ID}
			tx, err := store.Begin(ctx)
			if err != nil {
				t.Fatal(err)
			}
			for i := 1; i < workCount; i++ {
				id, _ := generator.NewID()
				item, e := d.NewWorkItem(id, w.Scope, fmt.Sprintf("Bounded work %d", i), "Focused task", d.PriorityNormal, d.WorkItemLifecycleTodo, now)
				if e != nil {
					t.Fatal(e)
				}
				item.ContractsEnabled = true
				if e = tx.WorkItems().Insert(ctx, item); e != nil {
					t.Fatal(e)
				}
				if i < 50 {
					workIDs = append(workIDs, id)
				}
			}
			if err = tx.Commit(); err != nil {
				t.Fatal(err)
			}
			contracts := []d.WorkContract{}
			for i, id := range workIDs {
				cc := sqliteCommandContext("0199ff10-0000-7000-8000-000000000080", fmt.Sprintf("contract-load-acquire-%d", i))
				got, e := service.AcquireWorkContract(ctx, cc, a.AcquireWorkContractCommand{Scope: w.Scope, WorkItemID: id, ExpectedWorkItemVersion: 1, TTLSeconds: 3600})
				if e != nil {
					t.Fatal(e)
				}
				contracts = append(contracts, got.Value.Contract)
			}
			// SQL_TRACE_START
			var sqlCount atomic.Uint64
			conn, err := store.db.Conn(ctx)
			if err != nil {
				t.Fatal(err)
			}
			err = conn.Raw(func(raw any) error {
				return raw.(driver.Conn).Raw().Trace(sqlite3.TRACE_STMT, func(sqlite3.TraceEvent, any, any) error { sqlCount.Add(1); return nil })
			})
			conn.Close()
			if err != nil {
				t.Fatal(err)
			}
			sqlMeasured := true
			// SQL_TRACE_END
			previous := 0
			for _, history := range []int{0, 10, 100} {
				// Checkpoint density is exercised on one representative active contract,
				// not advertised as 100 checkpoints on each of all 10,000 tasks.
				for j := previous; j < history; j++ {
					c := contracts[0]
					cc := sqliteCommandContext("0199ff10-0000-7000-8000-000000000080", fmt.Sprintf("contract-load-checkpoint-%d", j))
					got, e := service.SyncWorkContract(ctx, cc, a.SyncWorkContractCommand{Scope: w.Scope, ContractID: c.ID, Authority: a.ContractAuthority{ExecutionID: c.ExecutionID, FencingToken: c.FencingToken, SpecDigest: c.SpecDigest}, ExpectedContractVersion: c.Version, Checkpoint: a.ContractCheckpointInput{Summary: fmt.Sprintf("Durable checkpoint %d", j), NextAction: "Execute only the selected task"}})
					if e != nil {
						t.Fatal(e)
					}
					contracts[0] = got.Value.Contract
				}
				previous = history
				for _, consumers := range []int{2, 10, 50} {
					observer := &loadObserver{}
					service.SetObserver(observer)
					before := sqlCount.Load()
					poolBefore := store.db.Stats()
					started := time.Now()
					var conflicts, retries atomic.Uint64
					var mu sync.Mutex
					latencies := []float64{}
					maxBytes := 0
					failures := []string{}
					var wg sync.WaitGroup
					for i := 0; i < consumers; i++ {
						wg.Add(1)
						go func(i int) {
							defer wg.Done()
							c := contracts[i]
							for round := 0; round < 2; round++ {
								cc := sqliteCommandContext("0199ff10-0000-7000-8000-000000000080", fmt.Sprintf("contract-load-renew-%d-%d-%d-%d", history, consumers, i, round))
								renewCommand := a.RenewWorkContractCommand{Scope: w.Scope, ContractID: c.ID, Authority: a.ContractAuthority{ExecutionID: c.ExecutionID, FencingToken: c.FencingToken, SpecDigest: c.SpecDigest}, ExpectedLeaseVersion: c.LeaseVersion, TTLSeconds: 3600}
								got, e := retryLoadMutation(func() (a.MutationResult[a.WorkContractResult], error) {
									return service.RenewWorkContract(ctx, cc, renewCommand)
								}, &conflicts, &retries)
								if e != nil {
									mu.Lock()
									failures = append(failures, e.Error())
									mu.Unlock()
									return
								}
								c = got.Value.Contract
								queryStart := time.Now()
								view, e := retryLoadQuery(func() (a.WorkContext, error) { return service.GetWorkContext(ctx, w.Scope, c.WorkItemID) }, &conflicts, &retries)
								duration := float64(time.Since(queryStart).Microseconds()) / 1000
								raw, _ := json.Marshal(view)
								mu.Lock()
								latencies = append(latencies, duration)
								if e != nil {
									failures = append(failures, e.Error())
								}
								if len(raw) > maxBytes {
									maxBytes = len(raw)
								}
								mu.Unlock()
							}
							contracts[i] = c
						}(i)
					}
					wg.Wait()
					elapsed := time.Since(started)
					if len(failures) > 0 {
						t.Fatalf("load failures: %v", failures)
					}
					if maxBytes > a.MaxSnapshotBytes {
						t.Fatal("unbounded focal response")
					}
					waits := []float64{}
					guards := []float64{}
					mutation := []float64{}
					for _, v := range observer.items {
						waits = append(waits, float64(v.TransactionWait.Microseconds())/1000)
						guards = append(guards, float64(v.GuardDuration.Microseconds())/1000)
						mutation = append(mutation, float64(v.Duration.Microseconds())/1000)
					}
					report := map[string]any{"work_items": workCount, "checkpoint_sample": history, "checkpoint_sample_tasks": 1, "consumers": consumers, "renewals": consumers * 2, "focal_queries": len(latencies), "elapsed_ms": elapsed.Milliseconds(), "response_max_bytes": maxBytes, "focal_p50_ms": loadQuantile(latencies, .5), "focal_p95_ms": loadQuantile(latencies, .95), "focal_p99_ms": loadQuantile(latencies, .99), "renew_p95_ms": loadQuantile(mutation, .95), "transaction_wait_p95_ms": loadQuantile(waits, .95), "guard_wait_p95_ms": loadQuantile(guards, .95), "pool_wait_ms": float64((store.db.Stats().WaitDuration - poolBefore.WaitDuration).Microseconds()) / 1000, "conflicts": conflicts.Load(), "retries": retries.Load(), "sql_count_available": sqlMeasured}
					if sqlMeasured {
						report["sql_statements"] = sqlCount.Load() - before
					}
					encoded, _ := json.Marshal(report)
					t.Log("LOAD " + string(encoded))
				}
			}
		})
	}
}

func retryLoadMutation(fn func() (a.MutationResult[a.WorkContractResult], error), conflicts, retries *atomic.Uint64) (a.MutationResult[a.WorkContractResult], error) {
	for attempt := 0; ; attempt++ {
		v, err := fn()
		code, _ := d.ErrorCodeOf(err)
		if err == nil || code != d.ErrorCodeTransactionConflict || attempt >= 100 {
			return v, err
		}
		conflicts.Add(1)
		retries.Add(1)
		time.Sleep(time.Duration(5+min(attempt*4, 200)+rand.IntN(80)) * time.Millisecond)
	}
}

func retryLoadQuery(fn func() (a.WorkContext, error), conflicts, retries *atomic.Uint64) (a.WorkContext, error) {
	for attempt := 0; ; attempt++ {
		v, err := fn()
		code, _ := d.ErrorCodeOf(err)
		if err == nil || code != d.ErrorCodeTransactionConflict || attempt >= 100 {
			return v, err
		}
		conflicts.Add(1)
		retries.Add(1)
		time.Sleep(time.Duration(5+min(attempt*4, 200)+rand.IntN(80)) * time.Millisecond)
	}
}
