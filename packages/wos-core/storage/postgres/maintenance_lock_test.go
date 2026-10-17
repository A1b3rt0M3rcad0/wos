package postgres

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"
)

func TestCacheMaintenanceSkipsLockedRowsAcrossWorkers(t *testing.T) {
	requirePostgres(t)
	ctx := context.Background()
	store := openTestStore(t, filepath.Join(t.TempDir(), "cleanup.db"))
	ns := testID("0199a999-0000-7000-8000-000000000002")
	u, err := store.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	tu := u.(*unitOfWork)
	if err := ensureNamespace(ctx, tu.tx, ns); err != nil {
		t.Fatal(err)
	}
	if err := ensurePrincipal(ctx, tu.tx, "maintenance"); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		_, err := tu.tx.ExecContext(ctx, `INSERT INTO idempotency_records(namespace_id,principal_id,command_name,idempotency_key,request_hash,status,command_id,outcome_revision,response_json,created_at,expires_at) VALUES($1,'maintenance','cleanup',$2,'fingerprint','completed',$3,1,'{}',0,1)`, ns.String(), fmt.Sprintf("cleanup-worker-%d", i), testID("0199a999-0000-7000-8000-000000009999").String())
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := u.Commit(); err != nil {
		t.Fatal(err)
	}
	locked, err := store.db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer locked.Rollback()
	var key string
	if err := locked.QueryRowContext(ctx, `SELECT idempotency_key FROM idempotency_records WHERE idempotency_key='cleanup-worker-0' FOR UPDATE`).Scan(&key); err != nil {
		t.Fatal(err)
	}
	bounded, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	count, err := store.PruneExpiredIdempotency(bounded, 3)
	if err != nil || count != 2 {
		t.Fatalf("cleanup waited for an unrelated locked cache row: count=%d,error=%v", count, err)
	}
	if err := locked.Rollback(); err != nil {
		t.Fatal(err)
	}
	count, err = store.PruneExpiredIdempotency(ctx, 3)
	if err != nil || count != 1 {
		t.Fatalf("unlocked cache row was lost: count=%d,error=%v", count, err)
	}
}
