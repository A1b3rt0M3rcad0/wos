package memory

import (
	"context"
	"encoding/json"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sort"
	"time"
)

type triggerRepository struct{ tx *transaction }
type endpointRepository struct{ tx *transaction }
type deliveryRepository struct{ tx *transaction }

func (t *transaction) Triggers() ports.TriggerRepository    { return triggerRepository{t} }
func (t *transaction) Endpoints() ports.EndpointRepository  { return endpointRepository{t} }
func (t *transaction) Deliveries() ports.DeliveryRepository { return deliveryRepository{t} }
func (r triggerRepository) Get(ctx context.Context, scope domain.Scope, id domain.ID) (domain.Trigger, error) {
	if err := ctx.Err(); err != nil {
		return domain.Trigger{}, err
	}
	v, ok := r.tx.triggers[entityKey(scope, id)]
	if !ok {
		return v, domain.NewError(domain.ErrorCodeNotFound, "trigger not found")
	}
	return cloneTrigger(v), nil
}
func (r triggerRepository) List(ctx context.Context, scope domain.Scope) ([]domain.Trigger, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	values := []domain.Trigger{}
	for _, v := range r.tx.triggers {
		if v.Scope == scope {
			values = append(values, cloneTrigger(v))
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	return values, nil
}
func (r triggerRepository) Save(ctx context.Context, v domain.Trigger, expected domain.Version) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := v.Validate(); err != nil {
		return err
	}
	key := entityKey(v.Scope, v.ID)
	old, exists := r.tx.triggers[key]
	if expected == 0 && exists {
		return domain.NewError(domain.ErrorCodeAlreadyExists, "trigger exists")
	}
	if expected != 0 && (!exists || old.Version != expected) {
		return domain.NewError(domain.ErrorCodeVersionConflict, "trigger version changed")
	}
	r.tx.triggers[key] = cloneTrigger(v)
	return nil
}
func (r endpointRepository) Get(ctx context.Context, ns, id domain.ID) (ports.WebhookEndpoint, error) {
	if err := ctx.Err(); err != nil {
		return ports.WebhookEndpoint{}, err
	}
	v, ok := r.tx.endpoints[id.String()]
	if !ok || v.NamespaceID != ns {
		return ports.WebhookEndpoint{}, domain.NewError(domain.ErrorCodeNotFound, "endpoint not configured in namespace")
	}
	return v, nil
}
func (s *Store) InstallEndpoints(ctx context.Context, values []ports.WebhookEndpoint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := ctx.Err(); err != nil {
		return err
	}
	for _, v := range values {
		if err := v.ID.Validate(); err != nil {
			return err
		}
		if err := v.NamespaceID.Validate(); err != nil {
			return err
		}
		if existing, ok := s.endpoints[v.ID.String()]; ok && existing.NamespaceID != v.NamespaceID {
			return domain.NewError(domain.ErrorCodeForbidden, "endpoint namespace immutable")
		}
		s.endpoints[v.ID.String()] = v
	}
	return nil
}
func (l eventLog) appendIntegration(e domain.DomainEvent) error {
	for _, t := range l.tx.triggers {
		if !t.Matches(e) {
			continue
		}
		f := ports.NewFiring(t, e)
		key := f.ID.String()
		if _, exists := l.tx.firings[key]; exists {
			return domain.NewError(domain.ErrorCodeAlreadyExists, "firing already exists")
		}
		l.tx.firings[key] = f
		raw, err := json.Marshal(f)
		if err != nil {
			return err
		}
		for _, endpointID := range t.TargetEndpointIDs {
			ep, err := l.tx.Endpoints().Get(context.Background(), e.NamespaceID, endpointID)
			if err != nil {
				return err
			}
			id := key + "/" + endpointID.String()
			l.tx.deliveries[id] = ports.Delivery{ID: id, Scope: e.Scope(), IntegrationEventID: f.ID, EndpointID: endpointID, Status: "pending", NextAttempt: e.RecordedAt, URL: ep.URL, SecretRef: ep.SecretRef, KeyID: ep.KeyID, Body: raw}
		}
	}
	return nil
}
func (r deliveryRepository) List(ctx context.Context, scope domain.Scope, limit int, after string) ([]ports.Delivery, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	values := []ports.Delivery{}
	for _, v := range r.tx.deliveries {
		if v.Scope == scope && v.ID > after {
			v.Body = append(json.RawMessage{}, v.Body...)
			values = append(values, v)
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}
func (r deliveryRepository) Redeliver(ctx context.Context, scope domain.Scope, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	v, ok := r.tx.deliveries[id]
	if !ok || v.Scope != scope || (v.Status != "succeeded" && v.Status != "exhausted") {
		return domain.NewError(domain.ErrorCodePreconditionFailed, "only succeeded or exhausted deliveries can be redelivered")
	}
	v.Status = "pending"
	v.Attempts = 0
	v.NextAttempt = time.Now().UTC()
	v.LeaseID = ""
	v.LeaseUntil = time.Time{}
	v.LastResult = "manual_redelivery"
	r.tx.deliveries[id] = v
	return nil
}
func (r deliveryRepository) ListFirings(ctx context.Context, scope domain.Scope, limit int, after string) ([]ports.TriggerFiring, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	values := []ports.TriggerFiring{}
	for _, v := range r.tx.firings {
		if v.Source.NamespaceID == scope.NamespaceID && v.Source.OutcomeID == scope.OutcomeID && v.ID.String() > after {
			values = append(values, v)
		}
	}
	sort.Slice(values, func(i, j int) bool { return values[i].ID < values[j].ID })
	if len(values) > limit {
		values = values[:limit]
	}
	return values, nil
}
func cloneTrigger(v domain.Trigger) domain.Trigger {
	b, _ := json.Marshal(v)
	var result domain.Trigger
	_ = json.Unmarshal(b, &result)
	return result
}
func cloneTriggers(src map[string]domain.Trigger) map[string]domain.Trigger {
	dst := map[string]domain.Trigger{}
	for k, v := range src {
		dst[k] = cloneTrigger(v)
	}
	return dst
}
func cloneEndpoints(src map[string]ports.WebhookEndpoint) map[string]ports.WebhookEndpoint {
	dst := map[string]ports.WebhookEndpoint{}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
func cloneFirings(src map[string]ports.TriggerFiring) map[string]ports.TriggerFiring {
	dst := map[string]ports.TriggerFiring{}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
func cloneDeliveries(src map[string]ports.Delivery) map[string]ports.Delivery {
	dst := map[string]ports.Delivery{}
	for k, v := range src {
		v.Body = append(json.RawMessage{}, v.Body...)
		dst[k] = v
	}
	return dst
}
