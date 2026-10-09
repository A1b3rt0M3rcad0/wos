package memory

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
	"sort"
)

type serverIdentityRepository struct{ tx *transaction }

func (tx *transaction) ServerIdentity() ports.ServerIdentityRepository {
	return serverIdentityRepository{tx}
}
func (r serverIdentityRepository) Server(ctx context.Context) (d.ServerIdentity, error) {
	var zero d.ServerIdentity
	if err := ctx.Err(); err != nil {
		return zero, err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return zero, err
	}
	if r.tx.serverIdentity == nil {
		return zero, missingSigning()
	}
	s := *r.tx.serverIdentity
	return s, s.Validate()
}
func (r serverIdentityRepository) InsertServer(ctx context.Context, s d.ServerIdentity) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := r.tx.ensureOpen(); err != nil {
		return err
	}
	if err := s.Validate(); err != nil {
		return err
	}
	if r.tx.serverIdentity != nil {
		return d.NewError(d.ErrorCodeAlreadyExists, "server identity already pinned")
	}
	copy := s
	r.tx.serverIdentity = &copy
	return nil
}
func cloneServerIdentity(s *d.ServerIdentity) *d.ServerIdentity {
	if s == nil {
		return nil
	}
	copy := *s
	return &copy
}

func cloneIssuerHistory(values map[d.ID]d.ServerIdentity) map[d.ID]d.ServerIdentity {
	r := map[d.ID]d.ServerIdentity{}
	for k, v := range values {
		r[k] = v
	}
	return r
}
func cloneIssuerRecoveries(values map[d.ID]d.IssuerRecovery) map[d.ID]d.IssuerRecovery {
	r := map[d.ID]d.IssuerRecovery{}
	for k, v := range values {
		r[k] = v
	}
	return r
}
func (r serverIdentityRepository) LockServerIssuer(ctx context.Context) (d.ServerIdentity, error) {
	return r.Server(ctx)
}
func (r serverIdentityRepository) IssuerRecovery(ctx context.Context, id d.ID) (d.IssuerRecovery, error) {
	if e := ctx.Err(); e != nil {
		return d.IssuerRecovery{}, e
	}
	if e := r.tx.ensureOpen(); e != nil {
		return d.IssuerRecovery{}, e
	}
	v, ok := r.tx.issuerRecoveries[id]
	if !ok {
		return d.IssuerRecovery{}, missingSigning()
	}
	return v, v.Validate()
}
func (r serverIdentityRepository) ReplaceServerIssuer(ctx context.Context, record d.IssuerRecovery) error {
	current, e := r.Server(ctx)
	if e != nil {
		return e
	}
	if e = record.Validate(); e != nil {
		return e
	}
	if current != record.Previous {
		return d.NewError(d.ErrorCodeVersionConflict, "persistent issuer changed")
	}
	if _, exists := r.tx.issuerRecoveries[record.Replacement.IssuerKeyID]; exists {
		return d.NewError(d.ErrorCodeIdempotencyConflict, "issuer recovery already exists")
	}
	if old, exists := r.tx.issuerHistory[record.Replacement.IssuerKeyID]; exists && old != record.Replacement {
		return d.NewError(d.ErrorCodeAlreadyExists, "issuer identity already used")
	}
	for _, historical := range r.tx.issuerHistory {
		if historical.Fingerprint == record.Replacement.Fingerprint && historical.IssuerKeyID != record.Replacement.IssuerKeyID {
			return d.NewError(d.ErrorCodeAlreadyExists, "issuer fingerprint already used")
		}
	}
	r.tx.issuerHistory[current.IssuerKeyID] = current
	r.tx.issuerHistory[record.Replacement.IssuerKeyID] = record.Replacement
	r.tx.issuerRecoveries[record.Replacement.IssuerKeyID] = record
	copy := record.Replacement
	r.tx.serverIdentity = &copy
	return nil
}
func (r serverIdentityRepository) IssuerHistory(ctx context.Context, after d.ID, limit int) ([]d.ServerIdentity, error) {
	current, e := r.Server(ctx)
	if e != nil {
		return nil, e
	}
	if limit < 1 || limit > 101 {
		return nil, d.NewError(d.ErrorCodeInvalidArgument, "issuer history limit must be 1..101")
	}
	values := cloneIssuerHistory(r.tx.issuerHistory)
	values[current.IssuerKeyID] = current
	ids := []d.ID{}
	for id := range values {
		if id > after {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) > limit {
		ids = ids[:limit]
	}
	result := []d.ServerIdentity{}
	for _, id := range ids {
		result = append(result, values[id])
	}
	return result, nil
}
