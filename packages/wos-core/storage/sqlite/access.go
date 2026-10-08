package sqlite

import (
	"context"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/domain"
	"github.com/A1b3rt0M3rcad0/wos/packages/wos-core/ports"
)

func (u *unitOfWork) AuthorizeAccessSnapshot(ctx context.Context, r ports.AccessSnapshotRequest) error {
	denied := domain.NewError(domain.ErrorCodeForbidden, "credential or namespace permission inactive")
	var lifecycle string
	if err := u.tx.QueryRowContext(ctx, `SELECT lifecycle FROM namespaces WHERE id=?`, r.Authorization.NamespaceID.String()).Scan(&lifecycle); err != nil || lifecycle != "active" {
		return denied
	}
	var raw string
	err := u.tx.QueryRowContext(ctx, `SELECT g.permissions_json FROM namespace_grants g JOIN principals p ON p.id=g.principal_id WHERE g.namespace_id=? AND g.principal_id=? AND p.lifecycle='active'`, r.Authorization.NamespaceID.String(), r.Authorization.PrincipalID).Scan(&raw)
	if err != nil {
		return denied
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
	err = u.tx.QueryRowContext(ctx, `SELECT principal_id,namespace_id,actor_json,COALESCE(parent_digest,''),revoked,expires_at FROM credentials WHERE digest=?`, r.CredentialDigest).Scan(&principal, &ns, &actor, &parent, &revoked, &expires)
	var binding domain.ActorRef
	if err != nil || principal != r.Authorization.PrincipalID || ns != r.Authorization.NamespaceID.String() || revoked != 0 || expires <= encodeTime(r.Now) || unmarshalJSON(actor, &binding) != nil || binding != r.Actor {
		return denied
	}
	if parent != "" {
		err = u.tx.QueryRowContext(ctx, `SELECT principal_id,namespace_id,actor_json,COALESCE(parent_digest,''),revoked,expires_at FROM credentials WHERE digest=?`, parent).Scan(&principal, &ns, &actor, &parent, &revoked, &expires)
		if err != nil || parent != "" || principal != r.Authorization.PrincipalID || ns != r.Authorization.NamespaceID.String() || revoked != 0 || expires <= encodeTime(r.Now) || unmarshalJSON(actor, &binding) != nil || binding != r.Actor {
			return denied
		}
	}

	signing := signingIdentityRepository{u}
	credential, err := signing.CredentialByDigest(ctx, r.CredentialDigest)
	if err != nil {
		return denied
	}
	if credential.ParentDigest != "" {
		credential, err = signing.CredentialByDigest(ctx, credential.ParentDigest)
		if err != nil {
			return denied
		}
	}
	policy, err := signing.CredentialPolicy(ctx, credential.NamespaceID, credential.ID)
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
