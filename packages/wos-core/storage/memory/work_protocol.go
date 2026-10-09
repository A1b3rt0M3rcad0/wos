package memory

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sort"
	"time"
)

type workProtocolRepository struct{ tx *transaction }

func (tx *transaction) WorkProtocol() ports.WorkProtocolRepository { return workProtocolRepository{tx} }
func cloneProtocols(src map[d.ID]d.NamespaceWorkProtocol) map[d.ID]d.NamespaceWorkProtocol {
	out := map[d.ID]d.NamespaceWorkProtocol{}
	for k, v := range src {
		out[k] = v
	}
	return out
}
func (r workProtocolRepository) Lock(ctx context.Context, ns d.ID, _ bool) (d.NamespaceWorkProtocol, error) {
	if err := ctx.Err(); err != nil {
		return d.NamespaceWorkProtocol{}, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return d.NamespaceWorkProtocol{}, err
	}
	if p, ok := r.tx.protocols[ns]; ok {
		return p, p.Validate()
	}
	return d.DefaultWorkProtocol(ns), nil
}
func (r workProtocolRepository) Scopes(ctx context.Context, ns d.ID) ([]d.Scope, error) {
	out := []d.Scope{}
	for _, o := range r.tx.outcomes {
		if o.NamespaceID == ns {
			out = append(out, o.Scope())
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].OutcomeID < out[j].OutcomeID })
	return out, ctx.Err()
}
func (r workProtocolRepository) ValidLegacyLeases(ctx context.Context, ns d.ID, now time.Time) (int, error) {
	count := 0
	for _, w := range r.tx.workItems {
		if w.Scope.NamespaceID == ns && w.CurrentLease != nil && now.Before(w.CurrentLease.ExpiresAt) {
			count++
		}
	}
	return count, ctx.Err()
}
func (r workProtocolRepository) ValidUnsignedContracts(ctx context.Context, ns d.ID, now time.Time) (int, error) {
	count := 0
	for _, c := range r.tx.contracts {
		if c.Scope.NamespaceID == ns && c.Status == d.ContractActive && now.Before(c.ExpiresAt) {
			count++
		}
	}
	return count, ctx.Err()
}
func (r workProtocolRepository) EnableContracts(ctx context.Context, ns d.ID, now time.Time) error {
	if n, err := r.ValidLegacyLeases(ctx, ns, now); err != nil {
		return err
	} else if n > 0 {
		return d.NewError(d.ErrorCodePreconditionFailed, "valid legacy leases remain")
	}
	for key, w := range r.tx.workItems {
		if w.Scope.NamespaceID == ns {
			w.ContractsEnabled = true
			w.CurrentLease = nil
			r.tx.workItems[key] = w
		}
	}
	return nil
}
func (r workProtocolRepository) Save(ctx context.Context, p d.NamespaceWorkProtocol, expected d.Version) error {
	if err := p.Validate(); err != nil {
		return err
	}
	old, err := r.Lock(ctx, p.NamespaceID, true)
	if err != nil {
		return err
	}
	if old.Version != expected || p.Version != expected+1 {
		return d.NewError(d.ErrorCodeVersionConflict, "protocol version conflict")
	}
	r.tx.protocols[p.NamespaceID] = p
	return nil
}
