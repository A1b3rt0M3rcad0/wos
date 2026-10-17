package application

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"time"
)

type DeliveryHealthQuery struct {
	TenantID  d.ID  `json:"tenant_id"`
	OutcomeID *d.ID `json:"outcome_id,omitempty"`
}
type DeliveryHealthResult struct {
	State                  string                     `json:"state"`
	QueueState             string                     `json:"queue_state"`
	Queue                  ports.DeliveryQueueHealth  `json:"queue"`
	Worker                 ports.DeliveryWorkerStatus `json:"worker"`
	BacklogAgeLimitSeconds float64                    `json:"backlog_age_limit_seconds"`
	DeliveryGuarantee      string                     `json:"delivery_guarantee"`
}

// ConfigureDeliveryDiagnostics is host composition, before concurrent requests.
// It observes an integration signal worker; it never starts or schedules agents.
func (s *Service) ConfigureDeliveryDiagnostics(observer ports.DeliveryDiagnostics, maxAge time.Duration) error {
	if maxAge <= 0 || maxAge > 7*24*time.Hour {
		return d.NewError(d.ErrorCodeInvalidConfig, "delivery backlog age limit must be positive and at most seven days")
	}
	s.deliveryDiagnostics = observer
	s.deliveryBacklogAge = maxAge
	return nil
}
func (s *Service) GetDeliveryHealth(ctx context.Context, q DeliveryHealthQuery) (DeliveryHealthResult, error) {
	scope := d.Scope{TenantID: q.TenantID}
	if q.OutcomeID != nil {
		scope.OutcomeID = *q.OutcomeID
	}
	var zero DeliveryHealthResult
	if scope.TenantID.Validate() != nil || q.OutcomeID != nil && q.OutcomeID.Validate() != nil {
		return zero, d.NewError(d.ErrorCodeInvalidArgument, "valid Tenant and optional Outcome required")
	}
	identity, ok := IdentityFromContext(ctx)
	if !ok || identity.TenantID != scope.TenantID || identity.CredentialDigest == "" {
		return zero, d.NewError(d.ErrorCodeForbidden, "authenticated scoped administrator required")
	}
	request := ports.AuthorizationRequest{TenantID: scope.TenantID, OutcomeID: scope.OutcomeID, PrincipalID: identity.PrincipalID, Permission: ports.PermissionTenantAdmin}
	if s.requireIdentity {
		if e := s.authorizer.Authorize(ctx, request); e != nil {
			return zero, e
		}
	}
	u, e := s.beginReadSnapshot(ctx)
	if e != nil {
		return zero, e
	}
	defer u.Rollback()
	now, e := securityTransactionTime(ctx, u, s.clock)
	if e != nil {
		return zero, e
	}
	access, ok := u.(ports.AccessSnapshotUnitOfWork)
	if !ok {
		return zero, d.NewError(d.ErrorCodeInvalidConfig, "delivery diagnostic access snapshot unavailable")
	}
	if e = access.AuthorizeAccessSnapshot(ctx, ports.AccessSnapshotRequest{Authorization: request, CredentialDigest: identity.CredentialDigest, Actor: identity.Actor, Now: now}); e != nil {
		return zero, e
	}
	// Tenant metadata policy permits a zero Outcome. A queue aggregate,
	// however, reveals all Outcomes: resolve the current credential policy in
	// this same snapshot and reject restricted credentials before reading it.
	if scope.OutcomeID.IsZero() {
		signing, e := signingRepository(u)
		if e != nil {
			return zero, e
		}
		credential, e := signing.CredentialByDigest(ctx, identity.CredentialDigest)
		if e != nil {
			return zero, e
		}
		if credential.ParentDigest != "" {
			credential, e = signing.CredentialByDigest(ctx, credential.ParentDigest)
			if e != nil {
				return zero, e
			}
		}
		policy, e := signing.CredentialPolicy(ctx, scope.TenantID, credential.ID)
		if e != nil {
			if code, _ := d.ErrorCodeOf(e); code != d.ErrorCodeNotFound {
				return zero, e
			}
		} else if len(policy.AllowedOutcomeIDs) > 0 {
			return zero, d.NewError(d.ErrorCodeForbidden, "Outcome-restricted credential requires an explicit permitted Outcome")
		}
	}
	repo, ok := u.(ports.DeliveryHealthUnitOfWork)
	if !ok {
		return zero, d.NewError(d.ErrorCodeInvalidConfig, "authoritative delivery diagnostic repository unavailable")
	}
	queue, e := repo.DeliveryHealth().Health(ctx, scope, now)
	if e != nil {
		return zero, e
	}
	maxAge := s.deliveryBacklogAge
	if maxAge == 0 {
		maxAge = 5 * time.Minute
	}
	result := DeliveryHealthResult{State: "unknown", QueueState: "healthy", Queue: queue, BacklogAgeLimitSeconds: maxAge.Seconds(), DeliveryGuarantee: "at_least_once; consumer deduplicates integration_event_id; no receiver availability guarantee"}
	if queue.Exhausted > 0 || queue.OldestDueAgeSeconds != nil && *queue.OldestDueAgeSeconds > maxAge.Seconds() {
		result.QueueState = "degraded"
	}
	if s.deliveryDiagnostics != nil {
		result.Worker = s.deliveryDiagnostics.DeliveryWorkerStatus()
		switch {
		case !result.Worker.Enabled:
			result.State = "disabled"
		case !result.Worker.Running:
			result.State = "degraded"
		case result.Worker.LastTick == nil:
			result.State = "unknown"
		case now.Sub(*result.Worker.LastTick) > time.Minute:
			result.State = "degraded"
		default:
			result.State = "healthy"
		}
	}
	if result.QueueState == "degraded" {
		result.State = "degraded"
	}
	return result, nil
}
