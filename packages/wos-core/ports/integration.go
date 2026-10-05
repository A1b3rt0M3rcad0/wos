package ports

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"time"
)

type WebhookEndpoint struct {
	ID          domain.ID `json:"id"`
	NamespaceID domain.ID `json:"namespace_id"`
	URL         string    `json:"url"`
	SecretRef   string    `json:"secret_ref"`
	KeyID       string    `json:"key_id"`
}
type TriggerRepository interface {
	Get(context.Context, domain.Scope, domain.ID) (domain.Trigger, error)
	List(context.Context, domain.Scope) ([]domain.Trigger, error)
	Save(context.Context, domain.Trigger, domain.Version) error
}
type EndpointRepository interface {
	Get(context.Context, domain.ID, domain.ID) (WebhookEndpoint, error)
}
type IntegrationUnitOfWork interface {
	Triggers() TriggerRepository
	Endpoints() EndpointRepository
	Deliveries() DeliveryRepository
}
type IntegrationFact struct {
	SchemaVersion   int                    `json:"schema_version"`
	ID              domain.ID              `json:"id"`
	EventType       string                 `json:"event_type"`
	NamespaceID     domain.ID              `json:"namespace_id"`
	OutcomeID       domain.ID              `json:"outcome_id"`
	OutcomeRevision domain.OutcomeRevision `json:"outcome_revision"`
	EventIndex      uint32                 `json:"event_index"`
	Entity          domain.EntityRef       `json:"entity"`
	PrincipalID     string                 `json:"principal_id"`
	Actor           domain.ActorRef        `json:"actor_ref"`
	CommandID       domain.ID              `json:"command_id"`
	RecordedAt      time.Time              `json:"recorded_at"`
	CorrelationID   string                 `json:"correlation_id,omitempty"`
}

func PublicFact(e domain.DomainEvent) IntegrationFact {
	return IntegrationFact{1, e.EventID, e.EventType, e.NamespaceID, e.OutcomeID, e.OutcomeRevision, e.EventIndex, e.AggregateRef, e.PrincipalID, e.Actor, e.CommandID, e.RecordedAt, e.CorrelationID}
}

type TriggerFiring struct {
	ID             domain.ID       `json:"integration_event_id"`
	SchemaVersion  int             `json:"schema_version"`
	TriggerID      domain.ID       `json:"trigger_id"`
	TriggerVersion domain.Version  `json:"trigger_version"`
	SignalType     string          `json:"signal_type"`
	Source         IntegrationFact `json:"source"`
}

func NewFiring(t domain.Trigger, e domain.DomainEvent) TriggerFiring {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d:%s", t.ID, t.Version, e.EventID)))
	h := hex.EncodeToString(sum[:])
	id := domain.MustParseID(e.EventID.String()[:18] + "-8" + h[:3] + "-" + h[3:15])
	return TriggerFiring{ID: id, SchemaVersion: 1, TriggerID: t.ID, TriggerVersion: t.Version, SignalType: t.SignalType, Source: PublicFact(e)}
}

type Delivery struct {
	ID                 string          `json:"id"`
	Scope              domain.Scope    `json:"scope"`
	IntegrationEventID domain.ID       `json:"integration_event_id"`
	EndpointID         domain.ID       `json:"endpoint_id"`
	Status             string          `json:"status"`
	Attempts           int             `json:"attempts"`
	NextAttempt        time.Time       `json:"next_attempt"`
	LeaseID            domain.ID       `json:"lease_id,omitempty"`
	LeaseUntil         time.Time       `json:"lease_until,omitempty"`
	FencingToken       uint64          `json:"fencing_token"`
	LastResult         string          `json:"last_result,omitempty"`
	URL                string          `json:"-"`
	SecretRef          string          `json:"-"`
	KeyID              string          `json:"-"`
	Body               json.RawMessage `json:"-"`
}
type DeliveryRepository interface {
	List(context.Context, domain.Scope, int, string) ([]Delivery, error)
	Redeliver(context.Context, domain.Scope, string, time.Time) error
}
type DeliveryStore interface {
	ClaimDelivery(context.Context, time.Time, domain.ID, time.Duration) (*Delivery, error)
	FinishDelivery(context.Context, Delivery, time.Time, string, bool, time.Time) error
}
