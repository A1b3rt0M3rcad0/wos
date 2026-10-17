package sqlite

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func (u *unitOfWork) AuthorizeAccessSnapshot(ctx context.Context, r ports.AccessSnapshotRequest) error {
	denied := domain.NewError(domain.ErrorCodeForbidden, "credential or tenant permission inactive")
	var lifecycle string
	if err := u.tx.QueryRowContext(ctx, `SELECT lifecycle FROM tenants WHERE id=?`, r.Authorization.TenantID.String()).Scan(&lifecycle); err != nil {
		return accessSnapshotReadError("read Tenant access barrier", err)
	}
	if lifecycle != "active" {
		return denied
	}
	var raw string
	err := u.tx.QueryRowContext(ctx, `SELECT g.permissions_json FROM tenant_grants g JOIN principals p ON p.id=g.principal_id WHERE g.tenant_id=? AND g.principal_id=? AND p.lifecycle='active'`, r.Authorization.TenantID.String(), r.Authorization.PrincipalID).Scan(&raw)
	if err != nil {
		return accessSnapshotReadError("read Tenant permission", err)
	}
	var permissions []ports.Permission
	if unmarshalJSON(raw, &permissions) != nil {
		return denied
	}
	allowed := false
	for _, p := range permissions {
		allowed = allowed || p == r.Authorization.Permission
	}
	if !allowed {
		return denied
	}
	if r.CredentialDigest == "" {
		return denied
	}
	var principal, ns, actor, parent string
	var revoked int
	var expires int64
	err = u.tx.QueryRowContext(ctx, `SELECT principal_id,tenant_id,actor_json,COALESCE(parent_digest,''),revoked,expires_at FROM credentials WHERE digest=?`, r.CredentialDigest).Scan(&principal, &ns, &actor, &parent, &revoked, &expires)
	if err != nil {
		return accessSnapshotReadError("read credential access barrier", err)
	}
	var binding domain.ActorRef
	if principal != r.Authorization.PrincipalID || ns != r.Authorization.TenantID.String() || revoked != 0 || expires <= encodeTime(r.Now) || unmarshalJSON(actor, &binding) != nil || binding != r.Actor {
		return denied
	}
	if parent != "" {
		err = u.tx.QueryRowContext(ctx, `SELECT principal_id,tenant_id,actor_json,COALESCE(parent_digest,''),revoked,expires_at FROM credentials WHERE digest=?`, parent).Scan(&principal, &ns, &actor, &parent, &revoked, &expires)
		if err != nil {
			return accessSnapshotReadError("read parent credential access barrier", err)
		}
		if parent != "" || principal != r.Authorization.PrincipalID || ns != r.Authorization.TenantID.String() || revoked != 0 || expires <= encodeTime(r.Now) || unmarshalJSON(actor, &binding) != nil || binding != r.Actor {
			return denied
		}
	}

	signing := signingIdentityRepository{u}
	credential, err := signing.CredentialByDigest(ctx, r.CredentialDigest)
	if err != nil {
		return accessSnapshotReadError("read signed credential access", err)
	}
	if credential.ParentDigest != "" {
		credential, err = signing.CredentialByDigest(ctx, credential.ParentDigest)
		if err != nil {
			return accessSnapshotReadError("read signed parent credential access", err)
		}
	}
	policy, err := signing.CredentialPolicy(ctx, credential.TenantID, credential.ID)
	if err == nil && r.Authorization.Permission.InstallationScoped() && len(policy.AllowedOutcomeIDs) > 0 {
		return denied
	}
	if err == nil && !policy.Permits(string(r.Authorization.Permission), r.Authorization.OutcomeID) {
		return denied
	}
	if err != nil {
		code, _ := domain.ErrorCodeOf(err)
		if code != domain.ErrorCodeNotFound {
			return err
		}
	}
	return nil
}

var _ ports.AccessSnapshotUnitOfWork = (*unitOfWork)(nil)

// A failed SQL read grants no authority. Missing/inactive security state is a
// denial; a transaction conflict must remain retryable as the same exact intent
// instead of being disguised as a permanent credential revocation.
func accessSnapshotReadError(operation string, err error) error {
	if _, typed := domain.ErrorCodeOf(err); !typed {
		err = mapSQLError(operation, err)
	}
	if code, _ := domain.ErrorCodeOf(err); code == domain.ErrorCodeNotFound {
		return domain.NewError(domain.ErrorCodeForbidden, "credential or tenant permission inactive")
	}
	return err
}
