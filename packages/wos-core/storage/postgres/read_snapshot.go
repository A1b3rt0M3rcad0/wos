package postgres

import (
	"context"
	"crypto/sha256"
	"encoding/binary"

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
	err := u.tx.QueryRowContext(ctx, `SELECT state_revision FROM outcome_coordination WHERE tenant_id=$1 AND outcome_id=$2`, scope.TenantID.String(), scope.OutcomeID.String()).Scan(&revision)
	return ports.OutcomeCoordination{Scope: scope, Revision: revision}, mapSQLError("read outcome revision", err)
}

func (r signingIdentityRepository) TenantVersion(ctx context.Context, ns domain.ID) (domain.Version, error) {
	if err := r.u.ensureOpen(); err != nil {
		return 0, err
	}
	var version domain.Version
	err := r.u.tx.QueryRowContext(ctx, `SELECT version FROM tenants WHERE id=$1 AND lifecycle='active' FOR SHARE`, ns.String()).Scan(&version)
	return version, mapSQLError("read signing Tenant version", err)
}

func (u *unitOfWork) LockSignedQuota(ctx context.Context, ns domain.ID, principal, kind string, tenant bool) error {
	if err := u.ensureOpen(); err != nil {
		return err
	}
	if u.readSnapshot || ns.Validate() != nil || principal == "" || kind != "work" && kind != "review" {
		return domain.NewError(domain.ErrorCodeInvalidArgument, "invalid signed quota guard")
	}
	// Fixed order: shared protocol/security barrier, optional Tenant quota,
	// Principal quota, then Outcome. Hash collisions only over-serialize safely.
	keys := []string{}
	if tenant {
		keys = append(keys, "tenant")
	}
	keys = append(keys, "principal:"+principal)
	for _, key := range keys {
		digest := sha256.Sum256([]byte("wos:signed-quota:" + ns.String() + ":" + kind + ":" + key))
		lock := int64(binary.BigEndian.Uint64(digest[:8]))
		if _, err := u.tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock($1)`, lock); err != nil {
			return mapSQLError("lock signed acquisition quota", err)
		}
	}
	return nil
}

var _ ports.ReadSnapshotManager = (*Store)(nil)
var _ ports.SignedQuotaUnitOfWork = (*unitOfWork)(nil)
