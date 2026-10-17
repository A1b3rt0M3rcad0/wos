package ports

import (
	"context"
	"time"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

// DeliveryClaimPolicy bounds shared delivery capacity. All workers in an
// installation must use the same limits. Zero leaves that dimension unlimited.
// A nil SecretRefs accepts every configured reference; an empty non-nil slice
// accepts none, so a worker cannot consume retries for secrets it does not hold.
type DeliveryClaimPolicy struct {
	// DatabaseTime is used by fleet workers for shared lease/expiry decisions.
	// Explicit-clock callers retain the original deterministic storage contract.
	DatabaseTime            bool
	MaxInflightPerNamespace int
	MaxInflightPerEndpoint  int
	SecretRefs              []string
}

func (p DeliveryClaimPolicy) Validate() error {
	if p.MaxInflightPerNamespace < 0 || p.MaxInflightPerNamespace > 4096 || p.MaxInflightPerEndpoint < 0 || p.MaxInflightPerEndpoint > 4096 || len(p.SecretRefs) > 256 {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "delivery limits must be 0..4096 and secret references at most 256")
	}
	for _, ref := range p.SecretRefs {
		if ref == "" || len(ref) > 256 {
			return domain.NewError(domain.ErrorCodeInvalidConfig, "invalid delivery secret reference")
		}
	}
	return nil
}

// FairDeliveryStore applies namespace turns and capacity checks in the same
// transaction as the durable delivery lease, across worker processes.
type FairDeliveryStore interface {
	ClaimDeliveryWithPolicy(context.Context, time.Time, domain.ID, time.Duration, DeliveryClaimPolicy) (*Delivery, error)
}

// DurableDeliveryStore evaluates both lease acquisition and ACK deadlines on
// the authoritative database clock. Retry delay is relative to that same clock.
type DurableDeliveryStore interface {
	FairDeliveryStore
	FinishDeliveryUsingDatabaseTime(context.Context, Delivery, string, bool, time.Duration) error
}
