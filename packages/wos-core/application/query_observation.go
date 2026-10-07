package application

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"time"
)

func (s *Service) observeQuery(ctx context.Context, name string, scope domain.Scope, revision *domain.OutcomeRevision, err *error, started time.Time) {
	observer, ok := s.observer.(ports.QueryObserver)
	if !ok {
		return
	}
	code, _ := domain.ErrorCodeOf(*err)
	if *err != nil && code == "" {
		code = domain.ErrorCode("internal_error")
	}
	identity, _ := IdentityFromContext(ctx)
	observer.ObserveQuery(ports.QueryObservation{Name: name, Scope: scope, PrincipalID: identity.PrincipalID, Actor: identity.Actor, Revision: *revision, Duration: time.Since(started), ErrorCode: code})
}
