package sqlite

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/application"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	sqlite3 "github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkContinuity(b *testing.B) {
	historySize := 100
	if requested := os.Getenv("WOS_CONTINUITY_BENCHMARK_HISTORY"); requested != "" {
		var err error
		historySize, err = strconv.Atoi(requested)
		if err != nil || (historySize != 100 && historySize != 1000 && historySize != 10000) {
			b.Fatal("history must be 100, 1000 or 10000")
		}
	}

	store, err := Open(filepath.Join(b.TempDir(), "bench.db"), Options{MigrateOnOpen: true})
	if err != nil {
		b.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	service, _ := application.NewService(store, sqliteFixedClock{now: time.Now().UTC()}, &sqliteSequenceIDs{prefix: "0199d092", next: 1})
	ns := domain.MustParseID("0199d092-0000-7000-8000-000000000010")
	cc := sqliteCommandContext("0199d092-0000-7000-8000-000000000011", "")
	var first domain.Outcome
	for i := 0; i < 100; i++ {
		v, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{TenantID: ns, Title: fmt.Sprintf("Portfolio %03d", i), DesiredState: "benchmark", Priority: domain.PriorityNormal})
		if err != nil {
			b.Fatal(err)
		}
		if i == 0 {
			first = v.Value
		}
	}
	for i := 0; i < historySize; i++ {
		if _, err = service.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{Scope: first.Scope(), Title: fmt.Sprintf("Work %03d", i), Priority: domain.PriorityNormal}); err != nil {
			b.Fatal(err)
		}
	}

	rich, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{TenantID: ns, Title: "Portfolio rich", DesiredState: "batch immutable history", Priority: domain.PriorityNormal})
	if err != nil {
		b.Fatal(err)
	}
	var focalWorkID domain.ID
	for i := 0; i < historySize; i++ {
		work, err := service.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{Scope: rich.Value.Scope(), Title: fmt.Sprintf("Reviewed work %03d", i), Priority: domain.PriorityNormal})
		if err != nil {
			b.Fatal(err)
		}
		if i == 0 {
			focalWorkID = work.Value.ID
		}
		criterion, err := service.AddCriterion(ctx, cc, application.AddCriterionCommand{Owner: work.Value.Ref(), ExpectedVersion: 1, Title: "Reviewed", Required: true, VerificationMode: domain.VerificationModeAttestation})
		if err != nil {
			b.Fatal(err)
		}
		_, err = service.RecordCriterionAssessment(ctx, cc, application.RecordCriterionAssessmentCommand{Owner: work.Value.Ref(), ExpectedVersion: 2, CriterionID: criterion.Value.ID, CriterionRevision: 1, Result: domain.AssessmentResultMet, Rationale: "Benchmark observed review"})
		if err != nil {
			b.Fatal(err)
		}
	}
	var sqlCount atomic.Uint64
	conn, err := store.db.Conn(ctx)
	if err != nil {
		b.Fatal(err)
	}
	err = conn.Raw(func(raw any) error {
		return raw.(driver.Conn).Raw().Trace(sqlite3.TRACE_STMT, func(sqlite3.TraceEvent, any, any) error { sqlCount.Add(1); return nil })
	})
	conn.Close()
	if err != nil {
		b.Fatal(err)
	}
	for _, name := range []string{"discovery_101_outcomes", fmt.Sprintf("snapshot_%d_work_items", historySize), fmt.Sprintf("snapshot_%d_work_items_with_proof", historySize), fmt.Sprintf("work_context_%d_work_items_with_proof", historySize), fmt.Sprintf("section_%d_work_items_with_proof", historySize)} {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			before := sqlCount.Load()
			poolBefore := store.db.Stats()
			var samples []time.Duration
			for b.Loop() {
				started := time.Now()
				var value any
				var err error
				if name == "discovery_101_outcomes" {
					value, err = service.SearchOutcomes(ctx, ns, ports.OutcomeFilter{}, 25, "")
				} else if strings.HasPrefix(name, "section_") {
					value, err = service.GetContinuitySection(ctx, rich.Value.Scope(), "work_items", 25, "")
				} else if strings.HasPrefix(name, "work_context_") {
					value, err = service.GetWorkContext(ctx, rich.Value.Scope(), focalWorkID)
				} else {
					scope := first.Scope()
					if strings.HasSuffix(name, "_with_proof") {
						scope = rich.Value.Scope()
					}
					value, err = service.GetContinuity(ctx, scope, 25)
				}
				if err != nil {
					b.Fatal(err)
				}
				raw, err := json.Marshal(value)
				if err != nil {
					b.Fatal(err)
				}
				if len(raw) > application.MaxSnapshotBytes {
					b.Fatal("response cap exceeded")
				}
				samples = append(samples, time.Since(started))
				b.ReportMetric(float64(len(raw)), "response_bytes")
			}
			b.StopTimer()
			sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
			for _, quantile := range []int{50, 95, 99} {
				index := (len(samples)*quantile+99)/100 - 1
				b.ReportMetric(float64(samples[index].Nanoseconds()), fmt.Sprintf("p%d_ns", quantile))
			}
			b.ReportMetric(float64(sqlCount.Load()-before)/float64(b.N), "sql_statements/op")
			b.ReportMetric(float64(store.db.Stats().WaitDuration-poolBefore.WaitDuration)/float64(b.N), "pool_wait_ns/op")
		})
	}
	b.Run(fmt.Sprintf("parallel_snapshot_%d_work_items_with_proof", historySize), func(b *testing.B) {
		b.ReportAllocs()
		var durations atomic.Uint64
		poolBefore := store.db.Stats()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				started := time.Now()
				value, err := service.GetContinuity(ctx, rich.Value.Scope(), 25)
				if err != nil {
					b.Error(err)
					return
				}
				raw, err := json.Marshal(value)
				if err != nil || len(raw) > application.MaxSnapshotBytes {
					b.Error("invalid snapshot", err)
					return
				}
				durations.Add(uint64(time.Since(started)))
			}
		})
		b.StopTimer()
		b.ReportMetric(float64(b.N)/b.Elapsed().Seconds(), "snapshots/s")
		b.ReportMetric(float64(durations.Load())/float64(b.N), "mean_latency_ns/op")
		b.ReportMetric(float64(store.db.Stats().WaitDuration-poolBefore.WaitDuration)/float64(b.N), "pool_wait_ns/op")
	})

}
