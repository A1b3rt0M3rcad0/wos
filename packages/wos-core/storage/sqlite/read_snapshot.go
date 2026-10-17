package sqlite

import (
	"context"

	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func readOutcomeCoordination(ctx context.Context, u *unitOfWork, scope domain.Scope) (ports.OutcomeCoordination, error) {
	if err := u.ensureOpen(); err != nil {
		return ports.OutcomeCoordination{}, err
	}
	if err := scope.Validate(); err != nil {
		return ports.OutcomeCoordination{}, err
	}
	var revision domain.OutcomeRevision
	err := u.tx.QueryRowContext(ctx, `SELECT state_revision FROM outcome_coordination WHERE tenant_id=? AND outcome_id=?`, scope.TenantID.String(), scope.OutcomeID.String()).Scan(&revision)
	return ports.OutcomeCoordination{Scope: scope, Revision: revision}, mapSQLError("read outcome revision", err)
}

func (r signingIdentityRepository) TenantVersion(ctx context.Context, ns domain.ID) (domain.Version, error) {
	if err := r.u.ensureOpen(); err != nil {
		return 0, err
	}
	var version domain.Version
	err := r.u.tx.QueryRowContext(ctx, `SELECT version FROM tenants WHERE id=? AND lifecycle='active'`, ns.String()).Scan(&version)
	return version, mapSQLError("read signing Tenant version", err)
}

// BEGIN IMMEDIATE already excludes competing SQLite writers. Read snapshots
// use the driver's ReadOnly transaction, which is deferred and query-only.
func (u *unitOfWork) LockSignedQuota(ctx context.Context, ns domain.ID, principal, kind string, tenant bool) error {
	if err := u.ensureOpen(); err != nil {
		return err
	}
	if u.readSnapshot || ns.Validate() != nil || principal == "" || kind != "work" && kind != "review" {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "invalid signed quota guard")
	}
	return ctx.Err()
}

var _ ports.ReadSnapshotManager = (*Store)(nil)
var _ ports.SignedQuotaUnitOfWork = (*unitOfWork)(nil)
