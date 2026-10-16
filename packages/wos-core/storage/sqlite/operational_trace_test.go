package sqlite

import (
	"context"
	sqlite3 "github.com/ncruces/go-sqlite3"
	"github.com/ncruces/go-sqlite3/driver"
	"sync/atomic"
	"testing"
)

func traceOperationalSQL(b *testing.B, store *Store) *atomic.Uint64 {
	b.Helper()
	count := &atomic.Uint64{}
	conn, err := store.db.Conn(context.Background())
	if err != nil {
		b.Fatal(err)
	}
	err = conn.Raw(func(raw any) error {
		return raw.(driver.Conn).Raw().Trace(sqlite3.TRACE_STMT, func(sqlite3.TraceEvent, any, any) error { count.Add(1); return nil })
	})
	conn.Close()
	if err != nil {
		b.Fatal(err)
	}
	return count
}
