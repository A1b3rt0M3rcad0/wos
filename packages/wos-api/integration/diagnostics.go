package integration

import (
	"errors"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sync"
	"time"
)

var ErrWorkerPanic = errors.New("delivery worker panicked; inspect isolated server diagnostics")

// Diagnostics exposes process metadata only. Durable per-Namespace delivery
// times/counts belong to the database read port, not this in-memory heartbeat.
type Diagnostics struct {
	mu     sync.Mutex
	status ports.DeliveryWorkerStatus
}

func NewDiagnostics(enabled bool) *Diagnostics {
	concurrency := 0
	if enabled {
		concurrency = 1
	}
	return &Diagnostics{status: ports.DeliveryWorkerStatus{Enabled: enabled, Concurrency: concurrency}}
}
func (s *Diagnostics) DeliveryWorkerStatus() ports.DeliveryWorkerStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := s.status
	if out.StartedAt != nil {
		at := *out.StartedAt
		out.StartedAt = &at
	}
	if out.LastTick != nil {
		at := *out.LastTick
		out.LastTick = &at
	}
	return out
}
func (s *Diagnostics) started(now time.Time, concurrency int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	at := now.UTC()
	s.status.Concurrency = concurrency
	s.status.Running = true
	s.status.Started = true
	s.status.Starts++
	s.status.StartedAt = &at
	s.status.LastFailureClass = ""
}
func (s *Diagnostics) tick(now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	at := now.UTC()
	s.status.LastTick = &at
}
func (s *Diagnostics) stopped(err error, intentional bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.status.Running = false
	if !intentional {
		if err == nil {
			s.status.LastFailureClass = "unexpected_exit"
		} else {
			s.status.LastFailureClass = deliveryFailureClass(err)
		}
	}
}
func deliveryFailureClass(err error) string {
	if errors.Is(err, ErrWorkerPanic) {
		return "worker_panic"
	}
	code, _ := d.ErrorCodeOf(err)
	switch code {
	case d.ErrorCodeTransactionConflict:
		return "transaction_conflict"
	case d.ErrorCodeLease:
		return "stale_delivery_lease"
	case d.ErrorCodeInvalidConfig:
		return "invalid_configuration"
	case d.ErrorCodeInvalidArgument:
		return "invalid_delivery_state"
	default:
		return "storage_error"
	}
}
func fatalWorkerError(err error) bool {
	code, _ := d.ErrorCodeOf(err)
	return code != d.ErrorCodeTransactionConflict && code != d.ErrorCodeLease
}
