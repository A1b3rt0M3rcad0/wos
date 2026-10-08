package memory

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
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
