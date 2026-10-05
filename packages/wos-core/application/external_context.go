package application

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type LinkExternalReferenceCommand struct {
	Scope           domain.Scope
	ExpectedVersion domain.Version
	Reference       domain.ExternalReference
}
type RemoveExternalReferenceCommand struct {
	Scope           domain.Scope
	ExpectedVersion domain.Version
	Reference       domain.ExternalReference
}

func (s *Service) LinkExternalReference(ctx context.Context, cc domain.CommandContext, cmd LinkExternalReferenceCommand) (MutationResult[domain.Outcome], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Outcome]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (domain.Outcome, domain.OutcomeRevision, error) {
		return s.changeExternalReference(ctx, u, cmd.Scope, cmd.ExpectedVersion, cmd.Reference, false)
	})
}
func (s *Service) RemoveExternalReference(ctx context.Context, cc domain.CommandContext, cmd RemoveExternalReferenceCommand) (MutationResult[domain.Outcome], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Outcome]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (domain.Outcome, domain.OutcomeRevision, error) {
		return s.changeExternalReference(ctx, u, cmd.Scope, cmd.ExpectedVersion, cmd.Reference, true)
	})
}
func (s *Service) changeExternalReference(ctx context.Context, u ports.UnitOfWork, scope domain.Scope, expected domain.Version, ref domain.ExternalReference, remove bool) (domain.Outcome, domain.OutcomeRevision, error) {
	if err := ref.Validate(); err != nil {
		return domain.Outcome{}, 0, err
	}
	if err := lockExistingOutcome(ctx, u, scope); err != nil {
		return domain.Outcome{}, 0, err
	}
	repo, ok := u.(ports.ExternalContextUnitOfWork)
	if !ok {
		return domain.Outcome{}, 0, domain.NewError(domain.ErrorCodeInvalidConfig, "external context repository unavailable")
	}
	outcome, err := u.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
	if err != nil {
		return outcome, 0, err
	}
	if outcome.Version != expected {
		return outcome, 0, domain.NewError(domain.ErrorCodeVersionConflict, "outcome version changed")
	}
	values, err := repo.ExternalContexts().List(ctx, scope)
	if err != nil {
		return outcome, 0, err
	}
	exists := false
	for _, v := range values {
		if v.Provider == ref.Provider && v.Kind == ref.Kind && v.ExternalID == ref.ExternalID {
			exists = true
			if !remove && v.URL != ref.URL {
				return outcome, 0, domain.NewError(domain.ErrorCodeAlreadyExists, "reference already exists with another URL")
			}
		}
	}
	if remove && !exists {
		return outcome, 0, domain.NewError(domain.ErrorCodeNotFound, "external reference not found")
	}
	if !remove && exists {
		coord, err := u.Coordination().LockOutcome(ctx, scope)
		return outcome, coord.Revision, err
	}
	if !remove && len(values) >= 100 {
		return outcome, 0, domain.NewError(domain.ErrorCodeGraphLimitExceeded, "at most 100 external references per outcome")
	}
	if err = outcome.RecordExternalReferenceChange(s.clock.Now()); err != nil {
		return outcome, 0, err
	}
	if remove {
		err = repo.ExternalContexts().Remove(ctx, scope, ref)
	} else {
		err = repo.ExternalContexts().Add(ctx, scope, ref)
	}
	if err != nil {
		return outcome, 0, err
	}
	if err = u.Outcomes().Save(ctx, outcome, expected); err != nil {
		return outcome, 0, err
	}
	revision, err := u.Coordination().AdvanceOutcome(ctx, scope)
	return outcome, revision, err
}
func (s *Service) ListExternalReferences(ctx context.Context, scope domain.Scope) ([]domain.ExternalReference, domain.OutcomeRevision, error) {
	if err := s.authorizeRead(ctx, scope.NamespaceID); err != nil {
		return nil, 0, err
	}
	u, err := s.tx.Begin(ctx)
	if err != nil {
		return nil, 0, err
	}
	defer u.Rollback()
	if err = lockExistingOutcome(ctx, u, scope); err != nil {
		return nil, 0, err
	}
	repo, ok := u.(ports.ExternalContextUnitOfWork)
	if !ok {
		return nil, 0, domain.NewError(domain.ErrorCodeInvalidConfig, "external context unavailable")
	}
	values, err := repo.ExternalContexts().List(ctx, scope)
	if err != nil {
		return nil, 0, err
	}
	coord, err := u.Coordination().LockOutcome(ctx, scope)
	return values, coord.Revision, err
}
