package application

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// SetIndependentReviewer is a deployment composition policy, not a remote command.
// A principal that claimed/reclaimed/completed work in this Outcome cannot assess
// criteria or certify its Outcome/Objectives. Actor aliases do not bypass it.
func (s *Service) SetIndependentReviewer(enabled bool) { s.independentReviewer = enabled }
func (s *Service) requireIndependentReview(ctx context.Context, u ports.UnitOfWork, scope domain.Scope, principal string) error {
	if !s.independentReviewer {
		return nil
	}
	timeline, ok := u.Events().(ports.TimelineRepository)
	if !ok {
		return domain.NewError(domain.ErrorCodeInvalidConfig, "independent review requires execution history")
	}
	for _, typ := range []string{"work_item.claimed", "work_item.reclaimed", "work_item.completed", "work_item.admin_completed", "work_contract.acquired", "work_contract.execution_resumed", "work_contract.result_submitted"} {
		facts, err := timeline.ListEvents(ctx, scope, ports.EventFilter{PrincipalID: principal, EventType: typ, Limit: 1})
		if err != nil {
			return err
		}
		if len(facts) > 0 {
			return domain.NewError(domain.ErrorCodeForbidden, "deployment requires a reviewer principal independent from work execution in this outcome")
		}
	}
	return nil
}
