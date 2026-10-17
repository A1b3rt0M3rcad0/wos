package server

import (
	"context"
	"testing"
	"time"
)

type recordingMaintenanceStore struct {
	runtimeStore
	passes chan int
}

func (s recordingMaintenanceStore) PruneExpiredIdempotency(ctx context.Context, limit int) (int64, error) {
	if deadline, ok := ctx.Deadline(); !ok || time.Until(deadline) > 5*time.Second {
		panic("maintenance pass has no bounded deadline")
	}
	select {
	case s.passes <- limit:
	case <-ctx.Done():
	}
	return 0, nil
}

func TestMaintenancePassIsBoundedAndStopsOnCancellation(t *testing.T) {
	store := recordingMaintenanceStore{passes: make(chan int, 1)}
	runtime := &Runtime{store: store}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { defer close(done); runtime.runMaintenance(ctx, time.Millisecond) }()
	select {
	case limit := <-store.passes:
		if limit != 256 {
			t.Fatal("maintenance batch is unbounded", limit)
		}
	case <-time.After(time.Second):
		t.Fatal("maintenance never ran")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("maintenance did not stop")
	}
}

type fullBatchMaintenanceStore struct{ calls int }

func (s *fullBatchMaintenanceStore) PruneExpiredIdempotency(context.Context, int) (int64, error) {
	s.calls++
	return 256, nil
}

func TestMaintenanceDrainsMultipleBatchesWithAnUpperBound(t *testing.T) {
	store := &fullBatchMaintenanceStore{}
	runMaintenancePass(context.Background(), store)
	if store.calls != 4096 {
		t.Fatal("busy maintenance did not respect batch upper bound", store.calls)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	runMaintenancePass(ctx, store)
	if store.calls != 4096 {
		t.Fatal("cancelled maintenance acquired another batch", store.calls)
	}
}
