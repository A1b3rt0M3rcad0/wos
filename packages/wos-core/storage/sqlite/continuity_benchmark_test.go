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
	for _, name := range []string{"discovery_100_outcomes", "snapshot_100_work_items"} {
		b.Run(name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				var value any
				var err error
				if name == "discovery_100_outcomes" {
					value, err = service.SearchOutcomes(ctx, ns, ports.OutcomeFilter{}, 25, "")
				} else {
					value, err = service.GetContinuity(ctx, first.Scope(), 25)
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
		})
	}
}
