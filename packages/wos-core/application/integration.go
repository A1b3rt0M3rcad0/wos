package application

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type ConfigureTriggerCommand struct {
	Scope             domain.Scope
	Name              string
	EventTypes        []string
	Predicate         domain.EventPredicate
	TargetEndpointIDs []domain.ID
	SignalType        string
}
type UpdateTriggerCommand struct {
	Scope             domain.Scope
	TriggerID         domain.ID
	ExpectedVersion   domain.Version
	Name              string
	EventTypes        []string
	Predicate         domain.EventPredicate
	TargetEndpointIDs []domain.ID
	SignalType        string
}
type SetTriggerEnabledCommand struct {
	Scope           domain.Scope
	TriggerID       domain.ID
	ExpectedVersion domain.Version
	Enabled         bool
}
type RedeliverDeliveryCommand struct {
	Scope           domain.Scope
	ExpectedVersion domain.Version
	DeliveryID      string
	Reason          string
}

func integrationRepositories(u ports.UnitOfWork) (ports.IntegrationUnitOfWork, error) {
	repo, ok := u.(ports.IntegrationUnitOfWork)
	if !ok {
		return nil, domain.NewError(domain.ErrorCodeInvalidConfig, "integration repositories unavailable")
	}
	return repo, nil
}
func validateTriggerEndpoints(ctx context.Context, r ports.IntegrationUnitOfWork, t domain.Trigger) error {
	for _, id := range t.TargetEndpointIDs {
		if _, err := r.Endpoints().Get(ctx, t.Scope.NamespaceID, id); err != nil {
			return err
		}
	}
	return nil
}
func (s *Service) ConfigureTrigger(ctx context.Context, cc domain.CommandContext, cmd ConfigureTriggerCommand) (MutationResult[domain.Trigger], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Trigger]{}, err
	}
	id, err := s.ids.NewID()
	if err != nil {
		return MutationResult[domain.Trigger]{}, err
	}
	now := s.clock.Now().UTC()
	trigger := domain.Trigger{ID: id, Scope: cmd.Scope, Version: domain.InitialVersion, Name: cmd.Name, Enabled: true, EventTypes: cmd.EventTypes, Predicate: cmd.Predicate, TargetEndpointIDs: cmd.TargetEndpointIDs, SignalType: cmd.SignalType, CreatedAt: now, UpdatedAt: now}
	if err = trigger.Validate(); err != nil {
		return MutationResult[domain.Trigger]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (domain.Trigger, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, u, cmd.Scope); err != nil {
			return domain.Trigger{}, 0, err
		}
		r, err := integrationRepositories(u)
		if err != nil {
			return domain.Trigger{}, 0, err
		}
		values, err := r.Triggers().List(ctx, cmd.Scope)
		if err != nil {
			return domain.Trigger{}, 0, err
		}
		if len(values) >= 100 {
			return domain.Trigger{}, 0, domain.NewError(domain.ErrorCodeGraphLimitExceeded, "at most 100 trigger configurations per outcome")
		}
		if err = validateTriggerEndpoints(ctx, r, trigger); err != nil {
			return domain.Trigger{}, 0, err
		}
		if err = r.Triggers().Save(ctx, trigger, 0); err != nil {
			return domain.Trigger{}, 0, err
		}
		rev, err := u.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return trigger, rev, err
	})
}
func (s *Service) UpdateTrigger(ctx context.Context, cc domain.CommandContext, cmd UpdateTriggerCommand) (MutationResult[domain.Trigger], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Trigger]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (domain.Trigger, domain.OutcomeRevision, error) {
		if err := requireOpenOutcome(ctx, u, cmd.Scope); err != nil {
			return domain.Trigger{}, 0, err
		}
		r, err := integrationRepositories(u)
		if err != nil {
			return domain.Trigger{}, 0, err
		}
		t, err := r.Triggers().Get(ctx, cmd.Scope, cmd.TriggerID)
		if err != nil {
			return t, 0, err
		}
		if t.Version != cmd.ExpectedVersion {
			return t, 0, domain.NewError(domain.ErrorCodeVersionConflict, "trigger version changed")
		}
		if err = t.Replace(cmd.Name, cmd.EventTypes, cmd.Predicate, cmd.TargetEndpointIDs, cmd.SignalType, s.clock.Now()); err != nil {
			return t, 0, err
		}
		if err = validateTriggerEndpoints(ctx, r, t); err != nil {
			return t, 0, err
		}
		if err = r.Triggers().Save(ctx, t, cmd.ExpectedVersion); err != nil {
			return t, 0, err
		}
		rev, err := u.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return t, rev, err
	})
}
func (s *Service) SetTriggerEnabled(ctx context.Context, cc domain.CommandContext, cmd SetTriggerEnabledCommand) (MutationResult[domain.Trigger], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Trigger]{}, err
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (domain.Trigger, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, u, cmd.Scope); err != nil {
			return domain.Trigger{}, 0, err
		}
		r, err := integrationRepositories(u)
		if err != nil {
			return domain.Trigger{}, 0, err
		}
		t, err := r.Triggers().Get(ctx, cmd.Scope, cmd.TriggerID)
		if err != nil {
			return t, 0, err
		}
		if t.Version != cmd.ExpectedVersion {
			return t, 0, domain.NewError(domain.ErrorCodeVersionConflict, "trigger version changed")
		}
		if t.Enabled == cmd.Enabled {
			coord, err := u.Coordination().LockOutcome(ctx, cmd.Scope)
			return t, coord.Revision, err
		}
		if err = t.SetEnabled(cmd.Enabled, s.clock.Now()); err != nil {
			return t, 0, err
		}
		if err = r.Triggers().Save(ctx, t, cmd.ExpectedVersion); err != nil {
			return t, 0, err
		}
		rev, err := u.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return t, rev, err
	})
}
func (s *Service) RedeliverDelivery(ctx context.Context, cc domain.CommandContext, cmd RedeliverDeliveryCommand) (MutationResult[domain.Outcome], error) {
	if err := cc.Validate(); err != nil {
		return MutationResult[domain.Outcome]{}, err
	}
	if cmd.Reason == "" {
		return MutationResult[domain.Outcome]{}, domain.NewError(domain.ErrorCodeInvalidArgument, "redelivery reason required")
	}
	return transactCommand(ctx, s, cc, cmd, func(u ports.UnitOfWork) (domain.Outcome, domain.OutcomeRevision, error) {
		if err := lockExistingOutcome(ctx, u, cmd.Scope); err != nil {
			return domain.Outcome{}, 0, err
		}
		r, err := integrationRepositories(u)
		if err != nil {
			return domain.Outcome{}, 0, err
		}
		o, err := u.Outcomes().Get(ctx, cmd.Scope.NamespaceID, cmd.Scope.OutcomeID)
		if err != nil {
			return o, 0, err
		}
		if err = o.RecordExternalReferenceChange(s.clock.Now()); err != nil {
			return o, 0, err
		}
		if err = r.Deliveries().Redeliver(ctx, cmd.Scope, cmd.DeliveryID); err != nil {
			return o, 0, err
		}
		if err = u.Outcomes().Save(ctx, o, cmd.ExpectedVersion); err != nil {
			return o, 0, err
		}
		rev, err := u.Coordination().AdvanceOutcome(ctx, cmd.Scope)
		return o, rev, err
	})
}
func (s *Service) ListTriggers(ctx context.Context, scope domain.Scope) ([]domain.Trigger, domain.OutcomeRevision, error) {
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
	r, err := integrationRepositories(u)
	if err != nil {
		return nil, 0, err
	}
	v, err := r.Triggers().List(ctx, scope)
	if err != nil {
		return nil, 0, err
	}
	coord, err := u.Coordination().LockOutcome(ctx, scope)
	return v, coord.Revision, err
}

type IntegrationPage[T any] struct {
	Items       []T    `json:"items"`
	NextCursor  string `json:"next_cursor,omitempty"`
	Consistency string `json:"consistency"`
}

func (s *Service) ListDeliveries(ctx context.Context, scope domain.Scope, limit int, cursor string) (IntegrationPage[ports.Delivery], error) {
	var page IntegrationPage[ports.Delivery]
	if err := s.Authorize(ctx, scope.NamespaceID, ports.PermissionIntegrationWrite); err != nil && s.requireIdentity {
		return page, err
	}
	n, err := queryLimit(limit)
	if err != nil {
		return page, err
	}
	after := ""
	if cursor != "" {
		c, err := decodeCursor(cursor, scope.NamespaceID, scope.OutcomeID, "deliveries", "")
		if err != nil {
			return page, err
		}
		after = c.Key
	}
	u, err := s.tx.Begin(ctx)
	if err != nil {
		return page, err
	}
	defer u.Rollback()
	if err = lockExistingOutcome(ctx, u, scope); err != nil {
		return page, err
	}
	r, err := integrationRepositories(u)
	if err != nil {
		return page, err
	}
	values, err := r.Deliveries().List(ctx, scope, n+1, after)
	if err != nil {
		return page, err
	}
	if len(values) > n {
		values = values[:n]
		page.NextCursor = encodeCursor(queryCursor{Namespace: scope.NamespaceID, Outcome: scope.OutcomeID, Section: "deliveries", Key: values[n-1].ID})
	}
	page.Items = values
	page.Consistency = "live_keyset"
	return page, nil
}
func (s *Service) ListTriggerFirings(ctx context.Context, scope domain.Scope, limit int, cursor string) (IntegrationPage[ports.TriggerFiring], error) {
	var page IntegrationPage[ports.TriggerFiring]
	if err := s.authorizeRead(ctx, scope.NamespaceID); err != nil {
		return page, err
	}
	n, err := queryLimit(limit)
	if err != nil {
		return page, err
	}
	after := ""
	if cursor != "" {
		c, err := decodeCursor(cursor, scope.NamespaceID, scope.OutcomeID, "firings", "")
		if err != nil {
			return page, err
		}
		after = c.Key
	}
	u, err := s.tx.Begin(ctx)
	if err != nil {
		return page, err
	}
	defer u.Rollback()
	if err = lockExistingOutcome(ctx, u, scope); err != nil {
		return page, err
	}
	r, err := integrationRepositories(u)
	if err != nil {
		return page, err
	}
	reader, ok := r.Deliveries().(interface {
		ListFirings(context.Context, domain.Scope, int, string) ([]ports.TriggerFiring, error)
	})
	if !ok {
		return page, domain.NewError(domain.ErrorCodeInvalidConfig, "firing history unavailable")
	}
	values, err := reader.ListFirings(ctx, scope, n+1, after)
	if err != nil {
		return page, err
	}
	if len(values) > n {
		values = values[:n]
		page.NextCursor = encodeCursor(queryCursor{Namespace: scope.NamespaceID, Outcome: scope.OutcomeID, Section: "firings", Key: values[n-1].ID.String()})
	}
	page.Items = values
	page.Consistency = "live_keyset"
	return page, nil
}
