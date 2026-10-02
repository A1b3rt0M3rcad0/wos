# ADR 0006 — SQLite driver and writer acquisition

- Status: Accepted
- Date: 2026-10-02
- Scope: Wave 04

## Decision

The SQLite adapter uses `github.com/ncruces/go-sqlite3` through its `database/sql` driver.

The dependency is pinned in `go.mod`. The adapter configures every physical connection through the DSN with:

- `foreign_keys(1)`;
- WAL journal mode;
- synchronous NORMAL;
- the configured busy timeout;
- `_txlock=immediate`.

The writer pool is restricted to one connection per Store. Every WOS UnitOfWork starts a serializable SQL transaction. With this driver profile the transaction acquires the SQLite writer lock immediately instead of deferring writer acquisition until the first mutation.

## Rationale

Wave 04 requires explicit writer acquisition and per-connection foreign-key initialization. Those are correctness requirements, not performance hints.

Using the `database/sql` adapter preserves the public storage boundary while keeping SQLite-specific lock semantics inside `storage/sqlite`.

## Backup

The first backup contract uses SQLite `VACUUM ... INTO` to create a consistent standalone backup database while WAL mode is active. Restore consumes that standalone backup only while the destination Store is closed.

## Consequences

- the SQLite adapter remains CGO-free;
- local write concurrency is intentionally serialized;
- PostgreSQL will implement the same functional UnitOfWork contract with a different locking strategy in Wave 14;
- no SQLite connection may be opened outside the adapter with weaker PRAGMA settings and then used as a WOS writer.
