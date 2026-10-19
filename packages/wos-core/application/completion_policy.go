package application

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"strings"
)

// Read all applicable work within the caller's coherent read/guarded write
// transaction. A client page or a bounded Board sample never defines the gate.
func strictCompletionWork(ctx context.Context, u ports.UnitOfWork, owner d.EntityRef) ([]d.EntityRef, []d.EntityRef, error) {
	var tasks []d.WorkItem
	var objectives []d.Objective
	var err error
	if reads, ok := u.(ports.ContinuityReadUnitOfWork); ok {
		if repo, ok := reads.ContinuityReads().(ports.CompletionWorkRepository); ok {
			tasks, objectives, err = repo.CompletionWork(ctx, owner.Scope)
		} else {
			tasks, err = u.WorkItems().ListByOutcome(ctx, owner.Scope)
			if err == nil {
				objectives, err = u.Objectives().ListByOutcome(ctx, owner.Scope)
			}
		}
	} else {
		tasks, err = u.WorkItems().ListByOutcome(ctx, owner.Scope)
		if err == nil {
			objectives, err = u.Objectives().ListByOutcome(ctx, owner.Scope)
		}
	}
	if err != nil {
		return nil, nil, err
	}
	applicable := map[d.ID]bool{}
	if owner.Kind == d.EntityKindObjective {
		applicable[owner.ID] = true
		for changed := true; changed; {
			changed = false
			for _, v := range objectives {
				if v.ParentObjectiveID != nil && applicable[*v.ParentObjectiveID] && !applicable[v.ID] {
					applicable[v.ID] = true
					changed = true
				}
			}
		}
	}
	open, cancelled := []d.EntityRef{}, []d.EntityRef{}
	for _, v := range tasks {
		include := owner.Kind == d.EntityKindOutcome && v.ObjectiveID == nil || v.ObjectiveID != nil && applicable[*v.ObjectiveID]
		if !include {
			continue
		}
		switch v.Lifecycle {
		case d.WorkItemLifecycleDone:
		case d.WorkItemLifecycleCancelled:
			if v.CurrentConclusion == nil || strings.TrimSpace(v.CurrentConclusion.Reason) == "" {
				open = append(open, v.Ref())
			} else {
				cancelled = append(cancelled, v.Ref())
			}
		default:
			open = append(open, v.Ref())
		}
	}
	if owner.Kind == d.EntityKindObjective {
		for _, v := range objectives {
			if v.ID == owner.ID || !applicable[v.ID] {
				continue
			}
			switch v.Lifecycle {
			case d.ObjectiveLifecycleAchieved:
			case d.ObjectiveLifecycleCancelled:
				if v.CurrentConclusion == nil || strings.TrimSpace(v.CurrentConclusion.Reason) == "" {
					open = append(open, v.Ref())
				} else {
					cancelled = append(cancelled, v.Ref())
				}
			default:
				open = append(open, v.Ref())
			}
		}
	}
	return open, cancelled, nil
}
func checkStrictCompletion(ctx context.Context, u ports.UnitOfWork, owner d.EntityRef, policy d.CompletionPolicy, ack []d.EntityRef) error {
	if policy.Effective() != d.CompletionPolicyStrictWork {
		if len(ack) > 0 {
			return d.NewError(d.ErrorCodeInvalidArgument, "compatible completion does not acknowledge strict-work exclusions")
		}
		return nil
	}
	open, cancelled, err := strictCompletionWork(ctx, u, owner)
	if err != nil {
		return err
	}
	if len(open) > 0 {
		return d.NewDetailedError(d.ErrorCodePreconditionFailed, "Required work is still open. Complete or explicitly cancel the listed work before accepting this result.", d.ErrorDetails{ReasonCode: "required_child_work_open", SubjectRef: &owner, RelatedRefs: open})
	}
	expected := map[d.EntityRef]bool{}
	for _, ref := range cancelled {
		expected[ref] = true
	}
	seen := map[d.EntityRef]bool{}
	for _, ref := range ack {
		if !expected[ref] || seen[ref] {
			return d.NewError(d.ErrorCodeInvalidArgument, "cancellation acknowledgement must match the current excluded work exactly")
		}
		seen[ref] = true
	}
	if len(seen) != len(expected) {
		return d.NewDetailedError(d.ErrorCodePreconditionFailed, "Review and explicitly acknowledge each cancelled item excluded from this result. Cancelled work is not accepted work.", d.ErrorDetails{ReasonCode: "cancelled_work_requires_ack", SubjectRef: &owner, RelatedRefs: cancelled})
	}
	return nil
}
func completionAcknowledgements(values [][]d.EntityRef) []d.EntityRef {
	if len(values) > 0 {
		return values[0]
	}
	return nil
}

func completionOutcome(ctx context.Context, u ports.UnitOfWork, scope d.Scope) (d.Outcome, error) {
	if reads, ok := u.(ports.ContinuityReadUnitOfWork); ok {
		if repo, ok := reads.ContinuityReads().(ports.FocalReadRepository); ok {
			return repo.FocalOutcome(ctx, scope)
		}
	}
	return u.Outcomes().Get(ctx, scope.NamespaceID, scope.OutcomeID)
}
