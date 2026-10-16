package postgres

import (
	"context"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/stdlib"
	"sync/atomic"
	"testing"
)

type operationalQueryCounter struct{ count atomic.Uint64 }

func (q *operationalQueryCounter) TraceQueryStart(ctx context.Context, _ *pgx.Conn, _ pgx.TraceQueryStartData) context.Context {
	q.count.Add(1)
	return ctx
}
func (*operationalQueryCounter) TraceQueryEnd(context.Context, *pgx.Conn, pgx.TraceQueryEndData) {}

func traceOperationalSQL(b *testing.B, store *Store) *atomic.Uint64 {
	b.Helper()
	conn, err := store.db.Conn(context.Background())
	if err != nil {
		b.Fatal(err)
	}
	var config *pgx.ConnConfig
	err = conn.Raw(func(raw any) error { config = raw.(*stdlib.Conn).Conn().Config(); return nil })
	conn.Close()
	if err != nil {
		b.Fatal(err)
	}
	if config.Tracer != nil {
		b.Fatal("fixture already has a query tracer")
	}
	maximum := store.db.Stats().MaxOpenConnections
	if err = store.db.Close(); err != nil {
		b.Fatal(err)
	}
	counter := &operationalQueryCounter{}
	config.Tracer = counter
	store.db = stdlib.OpenDB(*config)
	store.db.SetMaxOpenConns(maximum)
	store.db.SetMaxIdleConns(maximum)
	return &counter.count
}
