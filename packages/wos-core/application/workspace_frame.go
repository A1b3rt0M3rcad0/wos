package application

import (
	"context"
	d "github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

// openWorkspaceFrame pins authorization, counters, entity facts and capability
// predicates to one transaction snapshot. Querying does not take the command
// coordinator's exclusive lock; mutations still use their existing guards.
// Supported adapters open serializable snapshots (memory holds a private clone).
func (s *Service) openWorkspaceFrame(ctx context.Context, scope d.Scope) (*continuityReadFrame, bool, error) {
	if e := scope.Validate(); e != nil {
		return nil, true, e
	}
	if e := s.authorizeScopedRead(ctx, scope); e != nil {
		return nil, true, e
	}
	u, e := s.tx.Begin(ctx)
	if e != nil {
		return nil, true, e
	}
	keep := false
	defer func() {
		if !keep {
			u.Rollback()
		}
	}()
	if e = s.authorizeScopedReadInUnitOfWork(ctx, u, scope); e != nil {
		return nil, true, e
	}
	reads, ok := u.(ports.ContinuityReadUnitOfWork)
	if !ok {
		return nil, false, nil
	}
	repo := reads.ContinuityReads()
	projection, ok := repo.(ports.ContinuityProjectionRepository)
	if !ok {
		return nil, false, nil
	}
	workspace, ok := repo.(ports.WorkspaceReadRepository)
	if !ok {
		return nil, false, nil
	}
	if _, e = repo.Outcome(ctx, scope); e != nil {
		return nil, true, e
	}
	revision, e := workspace.WorkspaceRevision(ctx, scope)
	if e != nil {
		return nil, true, e
	}
	now, e := s.transactionTime(ctx, u)
	if e != nil {
		return nil, true, e
	}
	keep = true
	return &continuityReadFrame{u: u, repo: repo, projection: projection, revision: revision, at: now}, true, nil
}
