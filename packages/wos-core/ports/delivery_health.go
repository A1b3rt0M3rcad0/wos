package ports

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"time"
)

// Queue facts come from one database snapshot, without reading webhook bodies,
// URLs or secret references. A zero OutcomeID selects the authorized Namespace.
type DeliveryQueueHealth struct {
	NamespaceID            d.ID       `json:"namespace_id"`
	OutcomeID              *d.ID      `json:"outcome_id,omitempty"`
	EvaluatedAt            time.Time  `json:"evaluated_at"`
	Pending                int64      `json:"pending"`
	Leased                 int64      `json:"leased"`
	ExpiredLeased          int64      `json:"expired_leased"`
	Succeeded              int64      `json:"succeeded"`
	Exhausted              int64      `json:"exhausted"`
	AttemptsTotal          int64      `json:"attempts_total"`
	PersistedDestinations  int64      `json:"persisted_destinations"`
	OldestDueAgeSeconds    *float64   `json:"oldest_due_age_seconds"`
	OldestLeaseAgeSeconds  *float64   `json:"oldest_lease_age_seconds"`
	NextDue                *time.Time `json:"next_due"`
	LastClaim              *time.Time `json:"last_claim"`
	LastSuccessfulDelivery *time.Time `json:"last_successful_delivery"`
}
type DeliveryHealthRepository interface {
	Health(context.Context, d.Scope, time.Time) (DeliveryQueueHealth, error)
}
type DeliveryHealthUnitOfWork interface {
	DeliveryHealth() DeliveryHealthRepository
}

// Process diagnostics never claim that a durable queue is empty. Last claim and
// delivery times are exposed only by the scoped database projection above.
type DeliveryWorkerStatus struct {
	Enabled          bool       `json:"enabled"`
	Running          bool       `json:"worker_running"`
	Started          bool       `json:"worker_started"`
	Starts           uint64     `json:"starts"`
	StartedAt        *time.Time `json:"started_at"`
	LastTick         *time.Time `json:"last_tick"`
	LastFailureClass string     `json:"last_failure_class,omitempty"`
	Concurrency      int        `json:"concurrency"`
}
type DeliveryDiagnostics interface{ DeliveryWorkerStatus() DeliveryWorkerStatus }
