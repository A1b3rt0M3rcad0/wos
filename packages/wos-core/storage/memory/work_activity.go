package memory

import (
	"context"
	"sort"

	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

type workActivityRepository struct{ tx *transaction }

func (tx *transaction) WorkActivities() ports.WorkActivityRepository {
	return workActivityRepository{tx}
}

func cloneWorkActivities(src map[d.ID]d.WorkActivity) map[d.ID]d.WorkActivity {
	out := make(map[d.ID]d.WorkActivity, len(src))
	for id, activity := range src {
		out[id] = contractCopy(activity)
	}
	return out
}

func (r workActivityRepository) Insert(ctx context.Context, activity d.WorkActivity) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := activity.Validate(); err != nil {
		return err
	}
	contract, err := r.tx.SignedWorkContracts().Get(ctx, activity.Scope, activity.ContractID)
	if err != nil {
		return err
	}
	if contract.WorkItemID != activity.WorkItemID {
		return d.NewError(d.ErrorCodeInvalidScope, "activity task does not match its signed contract")
	}
	if _, exists := r.tx.workActivities[activity.ID]; exists {
		return d.NewError(d.ErrorCodeAlreadyExists, "work activity is immutable")
	}
	for _, previous := range r.tx.workActivities {
		if previous.Scope == activity.Scope && previous.Sequence == activity.Sequence {
			return d.NewError(d.ErrorCodeAlreadyExists, "work activity sequence already exists")
		}
	}
	r.tx.workActivities[activity.ID] = contractCopy(activity)
	return nil
}

func (r workActivityRepository) List(ctx context.Context, scope d.Scope, filter ports.WorkActivityFilter) ([]d.WorkActivity, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return nil, err
	}
	if err := scope.Validate(); err != nil {
		return nil, err
	}
	if filter.Limit < 1 || filter.Limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "activity limit outside 1..101")
	}
	if filter.ThroughSequence == 0 || filter.AfterSequence > filter.ThroughSequence {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "invalid activity sequence range")
	}
	out := []d.WorkActivity{}
	for _, activity := range r.tx.workActivities {
		inRange := activity.Sequence > filter.AfterSequence
		if filter.Descending {
			inRange = filter.AfterSequence == 0 || activity.Sequence < filter.AfterSequence
		}
		if activity.Scope == scope &&
			(filter.WorkItemID == "" || activity.WorkItemID == filter.WorkItemID) &&
			(filter.ContractID == "" || activity.ContractID == filter.ContractID) &&
			inRange && activity.Sequence <= filter.ThroughSequence {
			out = append(out, contractCopy(activity))
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if filter.Descending {
			return out[i].Sequence > out[j].Sequence
		}
		return out[i].Sequence < out[j].Sequence
	})
	if len(out) > filter.Limit {
		out = out[:filter.Limit]
	}
	return out, nil
}
