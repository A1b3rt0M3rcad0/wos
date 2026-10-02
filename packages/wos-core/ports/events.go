package ports

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
)

type DomainEventLog interface {
	Append(ctx context.Context, events []domain.DomainEvent) error
}

type IdempotencyStore interface {
	Reserve(ctx context.Context, identity domain.IdempotencyIdentity, fingerprint string) (domain.IdempotencyReservation, error)
	Complete(ctx context.Context, reservation domain.IdempotencyReservation, result domain.StoredCommandResult) error
}

type AdministrativeAuditLog interface {
	AppendAdministrative(ctx context.Context, record domain.AdministrativeAuditRecord) error
}
