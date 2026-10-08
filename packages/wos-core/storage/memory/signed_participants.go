package memory

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"sort"
)

func (r signedContractRepository) ExecutionParticipants(ctx context.Context, scope d.Scope, work d.ID, limit int) ([]d.ExecutionParticipant, error) {
	if err := r.ready(ctx); err != nil {
		return nil, err
	}
	if scope.Validate() != nil || work.Validate() != nil || limit < 1 || limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "invalid bounded independence query")
	}
	seen := map[d.ExecutionParticipant]bool{}
	add := func(principal, group string) {
		seen[d.ExecutionParticipant{PrincipalID: principal, SeparationGroup: group}] = true
	}
	for _, c := range r.tx.signedContracts {
		if c.Scope == scope && c.WorkItemID == work {
			group := ""
			if c.SignedBinding != nil {
				group = c.SignedBinding.SeparationGroup
			}
			add(c.HolderPrincipalID, group)
		}
	}
	for _, c := range r.tx.contracts {
		if c.Scope == scope && c.WorkItemID == work {
			add(c.HolderPrincipalID, "")
		}
	}
	for _, e := range r.tx.events {
		if e.Scope() == scope && e.AggregateRef.ID == work {
			switch e.EventType {
			case "work_item.claimed", "work_item.reclaimed", "work_item.admin_completed":
				add(e.PrincipalID, "")
			}
		}
	}
	out := []d.ExecutionParticipant{}
	for p := range seen {
		out = append(out, p)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].PrincipalID == out[j].PrincipalID {
			return out[i].SeparationGroup < out[j].SeparationGroup
		}
		return out[i].PrincipalID < out[j].PrincipalID
	})
	if len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}
