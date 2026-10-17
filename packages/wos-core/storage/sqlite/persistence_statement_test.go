package sqlite

import (
	"context"
	"sync/atomic"
	"testing"

	sqlite3 "github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
)

// This checks the database work, not wall-clock performance: growing immutable
// history must not add SELECT/INSERT/DELETE statements to a title-only update.
func TestHistoryGrowthDoesNotMultiplyCurrentUpdateStatements(t *testing.T) {
	var counts []uint64
	for _, history := range []int{8, 512} {
		store, outcome := persistenceHistoryFixture(t, history)
		ctx := context.Background()
		var count atomic.Uint64
		conn, err := store.db.Conn(ctx)
		if err != nil {
			t.Fatal(err)
		}
		err = conn.Raw(func(raw any) error {
			return raw.(driver.Conn).Raw().Trace(sqlite3.TRACE_STMT, func(sqlite3.TraceEvent, any, any) error { count.Add(1); return nil })
		})
		conn.Close()
		if err != nil {
			t.Fatal(err)
		}
		u, err := store.Begin(ctx)
		if err != nil {
			t.Fatal(err)
		}
		loaded, err := u.Outcomes().Get(ctx, outcome.NamespaceID, outcome.ID)
		if err != nil {
			t.Fatal(err)
		}
		expected := loaded.Version
		loaded.Title = "Current title"
		loaded.Version++
		if err := u.Outcomes().Save(ctx, loaded, expected); err != nil {
			t.Fatal(err)
		}
		if err := u.Commit(); err != nil {
			t.Fatal(err)
		}
		counts = append(counts, count.Load())
	}
	if counts[0] != counts[1] || counts[1] > 32 {
		t.Fatalf("history multiplied SQL statements: 8=%d,512=%d", counts[0], counts[1])
	}
	t.Logf("title update including full aggregate read: %d statements for both 8 and 512 historical assessments", counts[0])
}
