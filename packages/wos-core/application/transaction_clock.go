package application

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"time"
)

func (s *Service) transactionTime(ctx context.Context, uow ports.UnitOfWork) (time.Time, error) {
	// The trusted embedded constructor explicitly accepts its caller's Clock.
	// Remote compositions must use the storage authority when one is available.
	if s.requireIdentity {
		if clock, ok := uow.(ports.TransactionClock); ok {
			return clock.EvaluationTime(ctx)
		}
	}
	return s.clock.Now().UTC(), nil
}
