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
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"
)

func BenchmarkContinuity(b *testing.B) {
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
		v, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{NamespaceID: ns, Title: fmt.Sprintf("Portfolio %03d", i), DesiredState: "benchmark", Priority: domain.PriorityNormal})
		if err != nil {
			b.Fatal(err)
		}
		if i == 0 {
			first = v.Value
		}
	}
	for i := 0; i < 100; i++ {
		if _, err = service.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{Scope: first.Scope(), Title: fmt.Sprintf("Work %03d", i), Priority: domain.PriorityNormal}); err != nil {
			b.Fatal(err)
		}
	}

	rich, err := service.CreateOutcome(ctx, cc, application.CreateOutcomeCommand{NamespaceID: ns, Title: "Portfolio rich", DesiredState: "batch immutable history", Priority: domain.PriorityNormal})
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < 100; i++ {
		work, err := service.CreateWorkItem(ctx, cc, application.CreateWorkItemCommand{Scope: rich.Value.Scope(), Title: fmt.Sprintf("Reviewed work %03d", i), Priority: domain.PriorityNormal})
		if err != nil {
			b.Fatal(err)
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
	for _, name := range []string{"discovery_100_outcomes", "snapshot_100_work_items", "snapshot_100_work_items_with_proof"} {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			before := sqlCount.Load()
			poolBefore := store.db.Stats()
			for b.Loop() {
				var value any
				var err error
				if name == "discovery_100_outcomes" {
					value, err = service.SearchOutcomes(ctx, ns, ports.OutcomeFilter{}, 25, "")
				} else {
					scope := first.Scope()
					if name == "snapshot_100_work_items_with_proof" {
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
				b.ReportMetric(float64(len(raw)), "response_bytes")
			}
			b.StopTimer()
			b.ReportMetric(float64(sqlCount.Load()-before)/float64(b.N), "sql_statements/op")
			b.ReportMetric(float64(store.db.Stats().WaitDuration-poolBefore.WaitDuration)/float64(b.N), "pool_wait_ns/op")
		})
	}
	b.Run("parallel_snapshot_100_work_items_with_proof", func(b *testing.B) {
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
